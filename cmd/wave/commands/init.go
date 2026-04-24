package commands

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/defaults"
	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/onboarding"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

type InitOptions struct {
	Force       bool
	Merge       bool
	All         bool
	Adapter     string
	Workspace   string
	OutputPath  string
	Yes         bool // Skip confirmation prompts
	Reconfigure bool // Re-run wizard with existing values as defaults
}

func NewInitCmd() *cobra.Command {
	var opts InitOptions

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Wave project",
		Long: `Create a new Wave project structure with default configuration.
Creates a wave.yaml manifest and .agents/personas/ directory with example prompts.

By default, only release-ready pipelines are included. Use --all to include
all embedded pipelines (useful for Wave contributors and developers).

Use --merge to add default configuration to an existing wave.yaml while
preserving your custom settings.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.Force, "force", false, "Overwrite existing files without prompting")
	cmd.Flags().BoolVar(&opts.Merge, "merge", false, "Merge defaults into existing configuration")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Include all pipelines regardless of release status")
	cmd.Flags().StringVar(&opts.Adapter, "adapter", "claude", "Default adapter to use")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", ".agents/workspaces", "Workspace directory path")
	cmd.Flags().StringVar(&opts.OutputPath, "manifest-path", "wave.yaml", "Output path for wave.yaml")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Answer yes to all confirmation prompts")
	cmd.Flags().BoolVar(&opts.Reconfigure, "reconfigure", false, "Re-run onboarding wizard with current settings as defaults")

	return cmd
}

// initAssets holds the resolved asset maps for init/merge operations.
type initAssets struct {
	personas       map[string]string
	personaConfigs map[string]manifest.Persona
	pipelines      map[string]string
	contracts      map[string]string
	prompts        map[string]string
}

// FileStatus represents the status of a file in the merge change summary.
type FileStatus string

const (
	FileStatusNew       FileStatus = "new"        // File does not exist, will be created
	FileStatusPreserved FileStatus = "preserved"  // File exists, differs from default
	FileStatusUpToDate  FileStatus = "up_to_date" // File exists, matches default byte-for-byte
)

// FileChangeEntry represents a single file's status in the change summary.
type FileChangeEntry struct {
	RelPath  string
	Category string
	Status   FileStatus
}

// ManifestAction represents the type of change to a manifest key.
type ManifestAction string

const (
	ManifestActionAdded     ManifestAction = "added"
	ManifestActionPreserved ManifestAction = "preserved"
)

// ManifestChangeEntry represents a change to a manifest key.
type ManifestChangeEntry struct {
	KeyPath string
	Action  ManifestAction
}

// ChangeSummary holds the complete pre-mutation change report.
type ChangeSummary struct {
	Files           []FileChangeEntry
	ManifestChanges []ManifestChangeEntry
	MergedManifest  map[string]interface{}
	Assets          *initAssets
	AlreadyUpToDate bool
}

// getFilteredAssets returns the asset maps for init, applying release filtering
// unless opts.All is true.
func getFilteredAssets(cmd *cobra.Command, opts InitOptions) (*initAssets, error) {
	personas, err := defaults.GetPersonas()
	if err != nil {
		return nil, fmt.Errorf("failed to get default personas: %w", err)
	}

	allPersonaConfigs, err := defaults.GetPersonaConfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to get persona configs: %w", err)
	}

	if opts.All {
		pipelines, err := defaults.GetPipelines()
		if err != nil {
			return nil, fmt.Errorf("failed to get default pipelines: %w", err)
		}
		contracts, err := defaults.GetContracts()
		if err != nil {
			return nil, fmt.Errorf("failed to get default contracts: %w", err)
		}
		prompts, err := defaults.GetPrompts()
		if err != nil {
			return nil, fmt.Errorf("failed to get default prompts: %w", err)
		}
		return &initAssets{
			personas:       personas,
			personaConfigs: allPersonaConfigs,
			pipelines:      pipelines,
			contracts:      contracts,
			prompts:        prompts,
		}, nil
	}

	// Release-filtered mode
	pipelines, err := defaults.GetReleasePipelines()
	if err != nil {
		return nil, fmt.Errorf("failed to get release pipelines: %w", err)
	}

	if len(pipelines) == 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: no pipelines are marked with release: true\n")
	}

	allContracts, err := defaults.GetContracts()
	if err != nil {
		return nil, fmt.Errorf("failed to get default contracts: %w", err)
	}
	allPrompts, err := defaults.GetPrompts()
	if err != nil {
		return nil, fmt.Errorf("failed to get default prompts: %w", err)
	}

	contracts, prompts, personaConfigs := filterTransitiveDeps(cmd, pipelines, allContracts, allPrompts, allPersonaConfigs)

	return &initAssets{
		personas:       personas,
		personaConfigs: personaConfigs,
		pipelines:      pipelines,
		contracts:      contracts,
		prompts:        prompts,
	}, nil
}

// systemPersonas are always included in the manifest regardless of pipeline references.
// These are used by relay compaction, meta-pipelines, and adhoc operations.
var systemPersonas = map[string]bool{
	"summarizer":  true,
	"navigator":   true,
	"philosopher": true,
}

// filterTransitiveDeps filters contracts, prompts, and persona configs to only
// those referenced by the given pipeline set. System personas are always included.
func filterTransitiveDeps(cmd *cobra.Command, pipelines, allContracts, allPrompts map[string]string, allPersonaConfigs map[string]manifest.Persona) (contracts, prompts map[string]string, personaConfigs map[string]manifest.Persona) {
	contractRefs := make(map[string]bool)
	promptRefs := make(map[string]bool)
	personaRefs := make(map[string]bool)

	for name, content := range pipelines {
		var p pipeline.Pipeline
		if err := yaml.Unmarshal([]byte(content), &p); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to parse pipeline %s for dependency resolution: %v\n", name, err)
			continue
		}

		for _, step := range p.Steps {
			// Extract persona references
			if step.Persona != "" {
				personaRefs[step.Persona] = true
			}

			// Extract compaction persona references
			if step.Handover.Compaction.Persona != "" {
				personaRefs[step.Handover.Compaction.Persona] = true
			}

			// Extract contract references from schema_path
			if sp := step.Handover.Contract.SchemaPath; sp != "" {
				normalized := strings.TrimPrefix(sp, ".agents/contracts/")
				contractRefs[normalized] = true
			}

			// Extract prompt references from source_path
			if sp := step.Exec.SourcePath; sp != "" {
				if strings.HasPrefix(sp, ".agents/prompts/") {
					normalized := strings.TrimPrefix(sp, ".agents/prompts/")
					promptRefs[normalized] = true
				}
			}
		}
	}

	// Always include system personas
	for name := range systemPersonas {
		personaRefs[name] = true
	}

	// Expand forge-templated persona refs (e.g. "{{ forge.type }}-analyst") into
	// all 4 forge variants so they survive filtering. filterPersonasByForge (called
	// later) trims to only the detected forge.
	expandedRefs := make(map[string]bool)
	for ref := range personaRefs {
		if strings.Contains(ref, "{{ forge.type }}") || strings.Contains(ref, "{{forge.type}}") {
			for _, forgeType := range []string{"github", "gitlab", "gitea", "bitbucket"} {
				expanded := strings.ReplaceAll(ref, "{{ forge.type }}", forgeType)
				expanded = strings.ReplaceAll(expanded, "{{forge.type}}", forgeType)
				expandedRefs[expanded] = true
			}
		} else {
			expandedRefs[ref] = true
		}
	}
	personaRefs = expandedRefs

	// Filter persona configs to only referenced ones
	personaConfigs = make(map[string]manifest.Persona)
	for name, cfg := range allPersonaConfigs {
		if personaRefs[name] {
			personaConfigs[name] = cfg
		}
	}

	// Filter contracts to only referenced ones
	contracts = make(map[string]string)
	for key, content := range allContracts {
		if contractRefs[key] {
			contracts[key] = content
		}
	}

	// Warn about referenced but missing contracts
	for ref := range contractRefs {
		if _, ok := allContracts[ref]; !ok {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: pipeline references contract %s which is not in embedded defaults\n", ref)
		}
	}

	// Filter prompts to only referenced ones
	prompts = make(map[string]string)
	for key, content := range allPrompts {
		if promptRefs[key] {
			prompts[key] = content
		}
	}

	return contracts, prompts, personaConfigs
}

func runInit(cmd *cobra.Command, opts InitOptions) error {
	// Handle --reconfigure: clear onboarding state and re-run wizard
	if opts.Reconfigure {
		return runReconfigure(cmd, opts)
	}

	// Determine if we should run the interactive wizard
	interactive := !opts.Yes && isInitInteractive()
	if interactive && !opts.Force && !opts.Merge {
		return runWizardInit(cmd, opts)
	}

	// Cold-start: ensure git repo exists
	if err := ensureGitRepo(cmd.ErrOrStderr()); err != nil {
		return err
	}

	// Get absolute path for clearer error messages
	absOutputPath, err := filepath.Abs(opts.OutputPath)
	if err != nil {
		absOutputPath = opts.OutputPath
	}

	existingFile, err := os.Stat(opts.OutputPath)
	fileExists := err == nil

	if fileExists {
		if opts.Force && !opts.Merge {
			// --force (without --merge): warn and require confirmation before destructive overwrite
			if !opts.Yes {
				confirmed, err := confirmForceOverwrite(cmd, absOutputPath)
				if err != nil {
					return fmt.Errorf("failed to read confirmation: %w", err)
				}
				if !confirmed {
					return fmt.Errorf("aborted: force overwrite cancelled (use --merge to preserve custom settings)")
				}
			}
			// Check file permissions before overwriting
			if existingFile.Mode().Perm()&0200 == 0 {
				return fmt.Errorf("cannot overwrite %s: file is read-only", absOutputPath)
			}
		} else {
			// Default behavior when wave.yaml exists: merge
			// Explicit --merge flag, --merge --force combo, or implicit default all route here.
			// When --force is combined with --merge, it acts as a prompt-skip modifier
			// (handled by confirmMerge which checks opts.Force).
			return runMerge(cmd, opts, absOutputPath)
		}
	}

	// Create .wave directory structure
	waveDirs := []string{
		".agents/personas",
		".agents/pipelines",
		".agents/contracts",
		".agents/prompts",
		".agents/traces",
		".agents/workspaces",
	}
	for _, dir := range waveDirs {
		absDir, _ := filepath.Abs(dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", absDir, err)
		}
	}

	// Detect project flavour
	cwd, _ := os.Getwd()
	flavour := onboarding.DetectFlavour(cwd)
	project := flavourToProjectMap(flavour)

	// Get filtered assets based on --all flag (needed for persona configs in manifest)
	assets, err := getFilteredAssets(cmd, opts)
	if err != nil {
		return err
	}

	// Filter personas by detected forge
	forgeInfo, _ := forge.DetectFromGitRemotes()
	assets.personaConfigs = filterPersonasByForge(assets.personaConfigs, forgeInfo.Type)

	// Extract project metadata for manifest name/description
	meta := onboarding.ExtractProjectMetadata(cwd)

	manifest := createDefaultManifest(opts.Adapter, opts.Workspace, project, assets.personaConfigs)

	// Override default metadata with extracted values
	if metaMap, ok := manifest["metadata"].(map[string]interface{}); ok {
		if meta.Name != "" {
			metaMap["name"] = meta.Name
		}
		if meta.Description != "" {
			metaMap["description"] = meta.Description
		}
	}

	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Ensure parent directory exists for custom output path
	outputDir := filepath.Dir(opts.OutputPath)
	if outputDir != "." && outputDir != "" {
		absOutputDir, _ := filepath.Abs(outputDir)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", absOutputDir, err)
		}
	}

	if err := os.WriteFile(opts.OutputPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest to %s: %w", absOutputPath, err)
	}

	if err := createExamplePersonas(assets.personas); err != nil {
		return fmt.Errorf("failed to create example personas in .agents/personas/: %w", err)
	}

	if err := createExamplePipelines(assets.pipelines); err != nil {
		return fmt.Errorf("failed to create example pipelines in .agents/pipelines/: %w", err)
	}

	if err := createExampleContracts(assets.contracts); err != nil {
		return fmt.Errorf("failed to create example contracts in .agents/contracts/: %w", err)
	}

	if err := createExamplePrompts(assets.prompts); err != nil {
		return fmt.Errorf("failed to create example prompts in .agents/prompts/: %w", err)
	}

	if err := createProjectInstructionFiles(); err != nil {
		return fmt.Errorf("failed to create project instruction files: %w", err)
	}

	// Cold-start: create initial commit if no commits exist
	if err := createInitialCommit(cmd.ErrOrStderr(), opts.OutputPath); err != nil {
		return err
	}

	printInitSuccess(cmd, opts.OutputPath, assets)
	suggestFirstRun(cmd.OutOrStdout(), flavour)
	return nil
}

func runMerge(cmd *cobra.Command, opts InitOptions, absOutputPath string) error {
	// Read existing manifest
	existingData, err := os.ReadFile(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to read existing manifest %s: %w", absOutputPath, err)
	}

	// Parse existing manifest — abort on parse failure (FR-013)
	var existingManifest map[string]interface{}
	if err := yaml.Unmarshal(existingData, &existingManifest); err != nil {
		return fmt.Errorf("failed to parse existing manifest %s: %w", absOutputPath, err)
	}

	// Get filtered assets
	cwd, _ := os.Getwd()
	flavour := onboarding.DetectFlavour(cwd)
	project := flavourToProjectMap(flavour)
	assets, err := getFilteredAssets(cmd, opts)
	if err != nil {
		return err
	}

	defaultManifest := createDefaultManifest(opts.Adapter, opts.Workspace, project, assets.personaConfigs)

	// Pre-mutation: compute all changes
	summary := computeChangeSummary(assets, existingManifest, defaultManifest)

	// Check if already up to date (US1-AS4)
	if summary.AlreadyUpToDate {
		fmt.Fprintf(cmd.ErrOrStderr(), "\n  Already up to date — no changes needed.\n\n")
		return nil
	}

	// Display change summary to stderr (FR-001, FR-006)
	displayChangeSummary(cmd.ErrOrStderr(), summary)

	// Confirm merge (FR-002, FR-007, FR-008, FR-014)
	confirmed, err := confirmMerge(cmd, opts)
	if err != nil {
		return err
	}
	if !confirmed {
		return fmt.Errorf("aborted: merge cancelled by user")
	}

	// Apply changes (FR-003, FR-004, FR-005)
	if err := applyChanges(summary, opts.OutputPath); err != nil {
		return err
	}

	printMergeSuccess(cmd, opts.OutputPath)
	return nil
}

func confirmOverwrite(cmd *cobra.Command, path string) (bool, error) {
	// If not running interactively, don't prompt
	if cmd.InOrStdin() == nil {
		return false, nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "File %s already exists. Overwrite? [y/N]: ", path)

	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

// confirmForceOverwrite prints a warning about data loss and asks for confirmation.
// This is used when --force is specified to ensure the user understands that custom
// personas, adapter configurations, and ontology settings will be lost.
func confirmForceOverwrite(cmd *cobra.Command, path string) (bool, error) {
	if cmd.InOrStdin() == nil {
		return false, nil
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "\n  WARNING: --force will overwrite %s\n", path)
	fmt.Fprintf(cmd.ErrOrStderr(), "  This will REPLACE all custom settings including:\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "    - Custom personas and adapter configurations\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "    - Ontology section (telos, contexts, conventions)\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "    - Project metadata (name, description)\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "\n  Consider using 'wave init --merge' to preserve custom settings.\n\n")
	fmt.Fprintf(cmd.OutOrStdout(), "Proceed with force overwrite? [y/N]: ")

	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

func mergeManifests(defaults, existing map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy all default values first
	for k, v := range defaults {
		result[k] = v
	}

	// Override with existing values, merging nested maps
	for k, v := range existing {
		if existingMap, isMap := v.(map[string]interface{}); isMap {
			if defaultMap, isDefaultMap := result[k].(map[string]interface{}); isDefaultMap {
				// Deep merge for maps
				result[k] = mergeMaps(defaultMap, existingMap)
			} else {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result
}

func mergeMaps(defaults, existing map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy all default values
	for k, v := range defaults {
		result[k] = v
	}

	// Override/add existing values
	for k, v := range existing {
		if existingMap, isMap := v.(map[string]interface{}); isMap {
			if defaultMap, isDefaultMap := result[k].(map[string]interface{}); isDefaultMap {
				result[k] = mergeMaps(defaultMap, existingMap)
			} else {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result
}

// mergeTypedManifests merges a generated manifest into an existing one,
// preserving custom settings from the existing manifest while updating
// infrastructure defaults from the generated manifest.
//
// Preservation rules:
//   - Custom personas (not in generated) are preserved
//   - Custom adapter configurations are preserved
//   - Ontology section is preserved entirely from existing
//   - Metadata.Name is preserved if already set
//   - Metadata.Description is preserved if already set
//   - apiVersion, kind, and runtime settings are updated from generated
func mergeTypedManifests(existing, generated *manifest.Manifest) *manifest.Manifest {
	result := &manifest.Manifest{}

	// Update infrastructure defaults from generated
	result.APIVersion = generated.APIVersion
	result.Kind = generated.Kind

	// Preserve metadata from existing if set, otherwise use generated
	result.Metadata = generated.Metadata
	if existing.Metadata.Name != "" {
		result.Metadata.Name = existing.Metadata.Name
	}
	if existing.Metadata.Description != "" {
		result.Metadata.Description = existing.Metadata.Description
	}
	if existing.Metadata.Repo != "" {
		result.Metadata.Repo = existing.Metadata.Repo
	}
	if existing.Metadata.Forge != "" {
		result.Metadata.Forge = existing.Metadata.Forge
	}

	// Merge adapters: generated as base, existing overrides/adds
	result.Adapters = make(map[string]manifest.Adapter)
	for name, adapter := range generated.Adapters {
		result.Adapters[name] = adapter
	}
	for name, adapter := range existing.Adapters {
		result.Adapters[name] = adapter
	}

	// Merge personas: generated as base, existing overrides/adds
	// Custom personas (not in generated) are preserved
	result.Personas = make(map[string]manifest.Persona)
	for name, persona := range generated.Personas {
		result.Personas[name] = persona
	}
	for name, persona := range existing.Personas {
		result.Personas[name] = persona
	}

	// Preserve ontology section entirely from existing if present
	if existing.Ontology != nil {
		result.Ontology = existing.Ontology
	} else {
		result.Ontology = generated.Ontology
	}

	// Preserve project section from existing if present
	if existing.Project != nil {
		result.Project = existing.Project
	} else {
		result.Project = generated.Project
	}

	// Update runtime settings from generated
	result.Runtime = generated.Runtime

	// Preserve server config from existing if set
	if existing.Server != nil {
		result.Server = existing.Server
	} else {
		result.Server = generated.Server
	}

	// Merge skills: combine both sets, deduplicate
	skillSet := make(map[string]bool)
	for _, s := range generated.Skills {
		skillSet[s] = true
	}
	for _, s := range existing.Skills {
		skillSet[s] = true
	}
	if len(skillSet) > 0 {
		result.Skills = make([]string, 0, len(skillSet))
		for s := range skillSet {
			result.Skills = append(result.Skills, s)
		}
		sort.Strings(result.Skills)
	}

	// Preserve hooks from existing if set
	if len(existing.Hooks) > 0 {
		result.Hooks = existing.Hooks
	} else {
		result.Hooks = generated.Hooks
	}

	return result
}

// computeChangeSummary builds a pre-mutation change report by comparing on-disk
// files with embedded defaults and computing the manifest diff.
func computeChangeSummary(assets *initAssets, existingManifest, defaultManifest map[string]interface{}) *ChangeSummary {
	var files []FileChangeEntry

	// Helper to classify a file
	classifyFile := func(path, category, defaultContent string) FileChangeEntry {
		entry := FileChangeEntry{
			RelPath:  path,
			Category: category,
		}
		existing, err := os.ReadFile(path)
		switch {
		case err != nil:
			entry.Status = FileStatusNew
		case bytes.Equal(existing, []byte(defaultContent)):
			entry.Status = FileStatusUpToDate
		default:
			entry.Status = FileStatusPreserved
		}
		return entry
	}

	// Check personas
	for filename, content := range assets.personas {
		path := filepath.Join(".agents", "personas", filename)
		files = append(files, classifyFile(path, "persona", content))
	}

	// Check pipelines
	for filename, content := range assets.pipelines {
		path := filepath.Join(".agents", "pipelines", filename)
		files = append(files, classifyFile(path, "pipeline", content))
	}

	// Check contracts
	for filename, content := range assets.contracts {
		path := filepath.Join(".agents", "contracts", filename)
		files = append(files, classifyFile(path, "contract", content))
	}

	// Check prompts
	for relPath, content := range assets.prompts {
		path := filepath.Join(".agents", "prompts", relPath)
		files = append(files, classifyFile(path, "prompt", content))
	}

	// Sort for deterministic output
	sort.Slice(files, func(i, j int) bool {
		return files[i].RelPath < files[j].RelPath
	})

	// Compute manifest diff
	merged, manifestChanges := computeManifestDiff(defaultManifest, existingManifest)

	// Determine if already up to date: no new files AND no added manifest keys
	alreadyUpToDate := true
	for _, f := range files {
		if f.Status == FileStatusNew {
			alreadyUpToDate = false
			break
		}
	}
	if alreadyUpToDate {
		for _, mc := range manifestChanges {
			if mc.Action == ManifestActionAdded {
				alreadyUpToDate = false
				break
			}
		}
	}

	return &ChangeSummary{
		Files:           files,
		ManifestChanges: manifestChanges,
		MergedManifest:  merged,
		Assets:          assets,
		AlreadyUpToDate: alreadyUpToDate,
	}
}

// computeManifestDiff performs the manifest merge and tracks what changed.
func computeManifestDiff(defaults, existing map[string]interface{}) (map[string]interface{}, []ManifestChangeEntry) {
	merged := mergeManifests(defaults, existing)
	var changes []ManifestChangeEntry
	collectManifestDiff("", defaults, existing, &changes)

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].KeyPath < changes[j].KeyPath
	})
	return merged, changes
}

// collectManifestDiff recursively walks default and existing manifests, recording
// keys that were added (from defaults) or preserved (user value kept).
func collectManifestDiff(prefix string, defaults, existing map[string]interface{}, entries *[]ManifestChangeEntry) {
	for key, defaultVal := range defaults {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		existingVal, exists := existing[key]
		if !exists {
			// Key in defaults but not in existing → added
			*entries = append(*entries, ManifestChangeEntry{
				KeyPath: path,
				Action:  ManifestActionAdded,
			})
			continue
		}

		// Key exists in both — recurse if both are maps
		defaultMap, defaultIsMap := defaultVal.(map[string]interface{})
		existingMap, existingIsMap := existingVal.(map[string]interface{})

		if defaultIsMap && existingIsMap {
			collectManifestDiff(path, defaultMap, existingMap, entries)
		} else if fmt.Sprintf("%v", defaultVal) != fmt.Sprintf("%v", existingVal) {
			// Values differ → user value is preserved
			*entries = append(*entries, ManifestChangeEntry{
				KeyPath: path,
				Action:  ManifestActionPreserved,
			})
		}
	}

	// User-only keys (not in defaults) are preserved
	for key := range existing {
		if _, inDefaults := defaults[key]; !inDefaults {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			*entries = append(*entries, ManifestChangeEntry{
				KeyPath: path,
				Action:  ManifestActionPreserved,
			})
		}
	}
}

// displayChangeSummary renders the ChangeSummary as a categorized table to the
// given writer (typically stderr).
func displayChangeSummary(w io.Writer, summary *ChangeSummary) {
	fmt.Fprintf(w, "\n  Change Summary:\n\n")

	categories := []struct {
		name  string
		label string
	}{
		{"persona", "Personas"},
		{"pipeline", "Pipelines"},
		{"contract", "Contracts"},
		{"prompt", "Prompts"},
	}

	for _, cat := range categories {
		var catFiles []FileChangeEntry
		for _, f := range summary.Files {
			if f.Category == cat.name {
				catFiles = append(catFiles, f)
			}
		}
		if len(catFiles) == 0 {
			continue
		}

		fmt.Fprintf(w, "  %s:\n", cat.label)
		for _, f := range catFiles {
			var status string
			switch f.Status {
			case FileStatusNew:
				status = "+ new"
			case FileStatusPreserved:
				status = "~ preserved"
			case FileStatusUpToDate:
				status = "= up to date"
			}
			fmt.Fprintf(w, "    %-14s %s\n", status, f.RelPath)
		}
		fmt.Fprintf(w, "\n")
	}

	// Show manifest changes
	added := 0
	preserved := 0
	for _, mc := range summary.ManifestChanges {
		if mc.Action == ManifestActionAdded {
			added++
		} else {
			preserved++
		}
	}

	if len(summary.ManifestChanges) > 0 {
		fmt.Fprintf(w, "  Manifest (wave.yaml):\n")
		for _, mc := range summary.ManifestChanges {
			var action string
			switch mc.Action {
			case ManifestActionAdded:
				action = "+ added"
			case ManifestActionPreserved:
				action = "~ preserved"
			}
			fmt.Fprintf(w, "    %-14s %s\n", action, mc.KeyPath)
		}
		fmt.Fprintf(w, "\n")
	}
}

// confirmMerge prompts the user for confirmation before applying merge changes.
// Skips the prompt when --yes or --force is specified. Requires --yes or --force
// in non-interactive terminals (FR-002, FR-014).
func confirmMerge(cmd *cobra.Command, opts InitOptions) (bool, error) {
	if opts.Yes || opts.Force {
		return true, nil
	}

	if !isInitInteractive() {
		return false, fmt.Errorf("non-interactive terminal detected: use --yes or --force to proceed without confirmation")
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "  Apply these changes? [y/N]: ")

	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

// applyChanges writes only "new" files from the ChangeSummary and writes the
// merged manifest. Files with status "preserved" or "up_to_date" are not touched.
func applyChanges(summary *ChangeSummary, outputPath string) error {
	// Ensure directories exist
	waveDirs := []string{
		".agents/personas",
		".agents/pipelines",
		".agents/contracts",
		".agents/prompts",
		".agents/traces",
		".agents/workspaces",
	}
	for _, dir := range waveDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			absDir, _ := filepath.Abs(dir)
			return fmt.Errorf("failed to create directory %s: %w", absDir, err)
		}
	}

	// Write only "new" files
	for _, f := range summary.Files {
		if f.Status != FileStatusNew {
			continue
		}

		var content string
		switch f.Category {
		case "persona":
			content = summary.Assets.personas[filepath.Base(f.RelPath)]
		case "pipeline":
			content = summary.Assets.pipelines[filepath.Base(f.RelPath)]
		case "contract":
			content = summary.Assets.contracts[filepath.Base(f.RelPath)]
		case "prompt":
			promptPrefix := filepath.Join(".agents", "prompts") + string(filepath.Separator)
			relPath := strings.TrimPrefix(f.RelPath, promptPrefix)
			content = summary.Assets.prompts[relPath]
		}

		// Ensure parent directory exists (for prompts with subdirs)
		if err := os.MkdirAll(filepath.Dir(f.RelPath), 0755); err != nil {
			absPath, _ := filepath.Abs(f.RelPath)
			return fmt.Errorf("failed to create directory for %s: %w", absPath, err)
		}

		if err := os.WriteFile(f.RelPath, []byte(content), 0644); err != nil {
			absPath, _ := filepath.Abs(f.RelPath)
			return fmt.Errorf("failed to write %s: %w", absPath, err)
		}
	}

	// Write merged manifest
	mergedData, err := yaml.Marshal(summary.MergedManifest)
	if err != nil {
		return fmt.Errorf("failed to marshal merged manifest: %w", err)
	}

	if err := os.WriteFile(outputPath, mergedData, 0644); err != nil {
		absPath, _ := filepath.Abs(outputPath)
		return fmt.Errorf("failed to write manifest to %s: %w", absPath, err)
	}

	return nil
}

func printInitSuccess(cmd *cobra.Command, outputPath string, assets *initAssets) {
	out := cmd.OutOrStdout()

	// Get sorted pipeline names for display
	pipelineNames := make([]string, 0, len(assets.pipelines))
	for name := range assets.pipelines {
		pipelineNames = append(pipelineNames, strings.TrimSuffix(name, ".yaml"))
	}
	sort.Strings(pipelineNames)

	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  ╦ ╦╔═╗╦  ╦╔═╗\n")
	fmt.Fprintf(out, "  ║║║╠═╣╚╗╔╝║╣ \n")
	fmt.Fprintf(out, "  ╚╩╝╩ ╩ ╚╝ ╚═╝\n")
	fmt.Fprintf(out, "  Multi-Agent Pipeline Orchestrator\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Project initialized successfully!\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Created:\n")
	fmt.Fprintf(out, "    %-24s Main manifest\n", outputPath)
	fmt.Fprintf(out, "    .agents/personas/          %d persona archetypes\n", len(assets.personas))
	fmt.Fprintf(out, "    .agents/pipelines/         %d pipelines\n", len(assets.pipelines))
	fmt.Fprintf(out, "    .agents/contracts/         %d JSON schema validators\n", len(assets.contracts))
	fmt.Fprintf(out, "    .agents/prompts/           %d prompt templates\n", len(assets.prompts))
	fmt.Fprintf(out, "    .agents/workspaces/        Ephemeral workspace root\n")
	fmt.Fprintf(out, "    .agents/traces/            Audit log directory\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Pipelines: %s\n", strings.Join(pipelineNames, ", "))
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Next steps:\n")
	fmt.Fprintf(out, "    1. Run 'wave validate' to check configuration\n")
	fmt.Fprintf(out, "    2. Run 'wave run ops-hello-world \"test\"' to verify setup\n")
	fmt.Fprintf(out, "    3. Run 'wave run plan-task \"your feature\"' to plan a task\n")
	fmt.Fprintf(out, "\n")
}

func printMergeSuccess(cmd *cobra.Command, outputPath string) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  ╦ ╦╔═╗╦  ╦╔═╗\n")
	fmt.Fprintf(out, "  ║║║╠═╣╚╗╔╝║╣ \n")
	fmt.Fprintf(out, "  ╚╩╝╩ ╩ ╚╝ ╚═╝\n")
	fmt.Fprintf(out, "  Multi-Agent Pipeline Orchestrator\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Configuration merged successfully!\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Updated:\n")
	fmt.Fprintf(out, "    %s       Preserved your settings\n", outputPath)
	fmt.Fprintf(out, "    Added missing default adapters and personas\n")
	fmt.Fprintf(out, "    Created missing .agents/ directories and files\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Next steps:\n")
	fmt.Fprintf(out, "    1. Run 'wave migrate up' to apply pending migrations\n")
	fmt.Fprintf(out, "    2. Run 'wave validate' to check configuration\n")
	fmt.Fprintf(out, "\n")
}

func buildPersonaManifest(configs map[string]manifest.Persona, adapter string) map[string]interface{} {
	result := make(map[string]interface{})
	for name, cfg := range configs {
		entry := map[string]interface{}{
			"adapter":            adapter,
			"description":        cfg.Description,
			"system_prompt_file": fmt.Sprintf(".agents/personas/%s.md", name),
			"temperature":        cfg.Temperature,
			"permissions": map[string]interface{}{
				"allowed_tools": cfg.Permissions.AllowedTools,
				"deny":          cfg.Permissions.Deny,
			},
		}
		if cfg.Model != "" {
			entry["model"] = cfg.Model
		}
		result[name] = entry
	}
	return result
}

func createDefaultManifest(adapter string, workspace string, project map[string]interface{}, personaConfigs map[string]manifest.Persona) map[string]interface{} {
	adapterProjectFiles := map[string][]string{
		"claude":   {"AGENTS.md"},
		"opencode": {"AGENTS.md"},
		"gemini":   {"AGENTS.md"},
		"codex":    {"AGENTS.md"},
	}

	adapterDefaults := map[string]string{
		"claude":   "sonnet",
		"opencode": "zai-coding-plan/glm-5-turbo",
		"gemini":   "gemini-2.5-flash-lite",
		"codex":    "o3",
	}

	adapters := map[string]interface{}{}
	for name, projectFiles := range adapterProjectFiles {
		entry := map[string]interface{}{
			"binary":        name,
			"default_model": adapterDefaults[name],
			"mode":          "headless",
			"output_format": "json",
			"project_files": projectFiles,
			"default_permissions": map[string]interface{}{
				"allowed_tools": []string{"Read", "Write", "Edit", "Bash"},
				"deny":          []string{},
			},
		}
		adapters[name] = entry
	}

	manifest := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "WaveManifest",
		"metadata": map[string]interface{}{
			"name":        "wave-project",
			"description": "A Wave multi-agent project",
		},
		"adapters": adapters,
		"personas": buildPersonaManifest(personaConfigs, adapter),
		"runtime": map[string]interface{}{
			"workspace_root":          workspace,
			"max_concurrent_workers":  5,
			"default_timeout_minutes": 30,
			"relay": map[string]interface{}{
				"token_threshold_percent": 80,
				"strategy":                "summarize_to_checkpoint",
			},
			"audit": map[string]interface{}{
				"log_dir":                 ".agents/traces/",
				"log_all_tool_calls":      true,
				"log_all_file_operations": false,
			},
			"meta_pipeline": map[string]interface{}{
				"max_depth":        2,
				"max_total_steps":  20,
				"max_total_tokens": 500000,
				"timeout_minutes":  60,
			},
		},
	}

	if project != nil {
		manifest["project"] = project
	}

	// Always include base quality context in ontology
	manifest["ontology"] = map[string]interface{}{
		"contexts": []map[string]interface{}{
			{
				"name":        "quality",
				"description": "Validation and quality gates — first-pass failure is expected, rework is the norm",
				"invariants": []string{
					"First-pass success is the exception, not the rule — validation exists to catch and correct",
					"Every pipeline output must pass through a validation gate before being considered done",
					"Rework after review is not a failure — it is the expected path to quality",
					"Contract validation, PR review, and test suites are gates, not formalities",
				},
			},
		},
	}

	return manifest
}

func createExamplePersonas(personas map[string]string) error {
	for filename, content := range personas {
		path := filepath.Join(".agents", "personas", filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			absPath, _ := filepath.Abs(path)
			return fmt.Errorf("failed to write %s: %w", absPath, err)
		}
	}

	return nil
}

func createExamplePipelines(pipelines map[string]string) error {
	for filename, content := range pipelines {
		path := filepath.Join(".agents", "pipelines", filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			absPath, _ := filepath.Abs(path)
			return fmt.Errorf("failed to write %s: %w", absPath, err)
		}
	}

	return nil
}

func createExampleContracts(contracts map[string]string) error {
	for filename, content := range contracts {
		path := filepath.Join(".agents", "contracts", filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			absPath, _ := filepath.Abs(path)
			return fmt.Errorf("failed to write %s: %w", absPath, err)
		}
	}

	return nil
}

func createExamplePrompts(prompts map[string]string) error {
	for relPath, content := range prompts {
		path := filepath.Join(".agents", "prompts", relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			absPath, _ := filepath.Abs(path)
			return fmt.Errorf("failed to write %s: %w", absPath, err)
		}
	}

	return nil
}

func createProjectInstructionFiles() error {
	files := map[string]string{
		"AGENTS.md": "See CLAUDE.md for project guidelines.",
		"CLAUDE.md": "See AGENTS.md for project guidelines.",
		"GEMINI.md": "See AGENTS.md for project guidelines.",
		"CODEX.md":  "See AGENTS.md for project guidelines.",
	}
	for filename, content := range files {
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", filename, err)
			}
		}
	}
	return nil
}

// isInitInteractive returns true when stdin is a TTY and interactive prompts are possible.
func isInitInteractive() bool {
	if v := os.Getenv("WAVE_FORCE_TTY"); v != "" {
		return v == "1" || v == "true"
	}
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// ensureGitRepo checks if the current directory is inside a git repository and
// initializes one if not. Uses git rev-parse to correctly detect parent repos.
func ensureGitRepo(w io.Writer) error {
	check := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	check.Stdout = io.Discard
	check.Stderr = io.Discard
	if check.Run() == nil {
		return nil // already inside a git repo
	}

	fmt.Fprintf(w, "  Initializing git repository...\n")
	cmd := exec.Command("git", "init")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}
	return nil
}

// createInitialCommit creates an initial commit with wave files if no commits
// exist yet. This ensures worktree operations have at least one commit to work with.
func createInitialCommit(w io.Writer, outputPath string) error {
	// Check if any commits exist
	check := exec.Command("git", "rev-parse", "HEAD")
	check.Stdout = io.Discard
	check.Stderr = io.Discard
	if check.Run() == nil {
		return nil // commits already exist
	}

	fmt.Fprintf(w, "  Creating initial commit...\n")

	// Ensure git user is configured (may not exist in fresh repos / CI).
	for _, kv := range [][2]string{
		{"user.name", "wave"},
		{"user.email", "wave@localhost"},
	} {
		check := exec.Command("git", "config", kv[0])
		check.Stdout = io.Discard
		check.Stderr = io.Discard
		if check.Run() != nil {
			cfg := exec.Command("git", "config", kv[0], kv[1])
			cfg.Stdout = io.Discard
			cfg.Stderr = io.Discard
			_ = cfg.Run()
		}
	}

	// Stage wave files specifically (not git add -A)
	add := exec.Command("git", "add", outputPath, ".agents/")
	add.Stdout = io.Discard
	add.Stderr = io.Discard
	if err := add.Run(); err != nil {
		return fmt.Errorf("failed to stage wave files: %w", err)
	}

	// Explicitly disable commit signing for this invocation. If the user's
	// global gpg config is broken (keyboxd issues) or misses a secret key,
	// signing would fail and block cold-start onboarding. The initial
	// Wave-managed commit should always land.
	commit := exec.Command("git", "-c", "commit.gpgsign=false", "commit", "-m", "chore: initialize wave project")
	commit.Stdout = io.Discard
	commit.Stderr = io.Discard
	if err := commit.Run(); err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}
	return nil
}

// flavourToProjectMap converts a FlavourInfo into the map[string]interface{}
// format expected by createDefaultManifest.
func flavourToProjectMap(fi *onboarding.FlavourInfo) map[string]interface{} {
	if fi == nil {
		return nil
	}
	m := map[string]interface{}{}
	if fi.Flavour != "" {
		m["flavour"] = fi.Flavour
	}
	if fi.Language != "" {
		m["language"] = fi.Language
	}
	if fi.TestCommand != "" {
		m["test_command"] = fi.TestCommand
	}
	if fi.LintCommand != "" {
		m["lint_command"] = fi.LintCommand
	}
	if fi.BuildCommand != "" {
		m["build_command"] = fi.BuildCommand
	}
	if fi.FormatCommand != "" {
		m["format_command"] = fi.FormatCommand
	}
	if fi.SourceGlob != "" {
		m["source_glob"] = fi.SourceGlob
	}
	if fi.Skill != "" {
		m["skill"] = fi.Skill
	}
	return m
}

// knownForgePrefixes lists the prefixes used by forge-specific personas.
var knownForgePrefixes = []string{"github-", "gitlab-", "bitbucket-", "gitea-"}

// forgeTypeToPrefix maps forge types to their persona naming convention prefix.
var forgeTypeToPrefix = map[forge.ForgeType]string{
	forge.ForgeGitHub:    "github",
	forge.ForgeGitLab:    "gitlab",
	forge.ForgeBitbucket: "bitbucket",
	forge.ForgeGitea:     "gitea",
	forge.ForgeCodeberg:  "gitea", // Codeberg is Forgejo — shares Gitea personas
}

// filterPersonasByForge filters persona configs to only include personas
// matching the detected forge type. Personas without a forge prefix are always included.
func filterPersonasByForge(configs map[string]manifest.Persona, ft forge.ForgeType) map[string]manifest.Persona {
	if ft == forge.ForgeUnknown {
		return configs // no filtering when forge is unknown
	}

	prefix, ok := forgeTypeToPrefix[ft]
	if !ok {
		return configs
	}

	result := make(map[string]manifest.Persona)
	for name, cfg := range configs {
		hasKnownPrefix := false
		for _, fp := range knownForgePrefixes {
			if strings.HasPrefix(name, fp) {
				hasKnownPrefix = true
				break
			}
		}
		// Include personas that match the forge prefix or have no forge prefix
		if strings.HasPrefix(name, prefix+"-") || !hasKnownPrefix {
			result[name] = cfg
		}
	}
	return result
}

// suggestFirstRun prints a suggestion for what to run after init.
func suggestFirstRun(w io.Writer, flavour *onboarding.FlavourInfo) {
	if flavour == nil || flavour.SourceGlob == "" {
		fmt.Fprintf(w, "  Suggestion: Run 'wave run ops-bootstrap' to scaffold your project\n")
		return
	}

	// Flavour was detected so source files exist — suggest analysis over scaffolding.
	fmt.Fprintf(w, "  Suggestion: Run 'wave run audit-dx' to analyze your codebase\n")
}

// runWizardInit runs the interactive onboarding wizard for first-time setup.
func runWizardInit(cmd *cobra.Command, opts InitOptions) error {
	// Cold-start: ensure git repo exists
	if err := ensureGitRepo(cmd.ErrOrStderr()); err != nil {
		return err
	}

	// Print Wave logo
	fmt.Fprintln(cmd.OutOrStdout(), tui.WaveLogo())

	// Check if wave.yaml already exists
	var existing *manifest.Manifest
	if data, err := os.ReadFile(opts.OutputPath); err == nil {
		var m manifest.Manifest
		if err := yaml.Unmarshal(data, &m); err == nil {
			existing = &m
		}
	}

	// If file exists and not forcing, prompt for confirmation
	if existing != nil && !opts.Force {
		confirmed, err := confirmOverwrite(cmd, opts.OutputPath)
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		if !confirmed {
			return fmt.Errorf("aborted: %s already exists (use --force to overwrite or --merge to merge)", opts.OutputPath)
		}
	}

	// Create .wave directory structure
	waveDirs := []string{
		".agents/personas",
		".agents/pipelines",
		".agents/contracts",
		".agents/prompts",
		".agents/traces",
		".agents/workspaces",
		".claude/commands",
	}
	for _, dir := range waveDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			absDir, _ := filepath.Abs(dir)
			return fmt.Errorf("failed to create directory %s: %w", absDir, err)
		}
	}

	// Copy default assets before running wizard (so pipeline selection can discover them)
	assets, err := getFilteredAssets(cmd, opts)
	if err != nil {
		return err
	}

	if err := createExamplePersonas(assets.personas); err != nil {
		return fmt.Errorf("failed to create example personas: %w", err)
	}
	if err := createExamplePipelines(assets.pipelines); err != nil {
		return fmt.Errorf("failed to create example pipelines: %w", err)
	}
	if err := createExampleContracts(assets.contracts); err != nil {
		return fmt.Errorf("failed to create example contracts: %w", err)
	}
	if err := createExamplePrompts(assets.prompts); err != nil {
		return fmt.Errorf("failed to create example prompts: %w", err)
	}

	cfg := onboarding.WizardConfig{
		WaveDir:        ".agents",
		Interactive:    true,
		Reconfigure:    false,
		Existing:       existing,
		All:            opts.All,
		Adapter:        opts.Adapter,
		Workspace:      opts.Workspace,
		OutputPath:     opts.OutputPath,
		PersonaConfigs: assets.personaConfigs,
	}

	result, err := onboarding.RunWizard(cfg)
	if err != nil {
		return fmt.Errorf("onboarding wizard failed: %w", err)
	}

	// Remove deselected pipelines
	if len(result.Pipelines) > 0 {
		if err := removeDeselectedPipelines(".agents/pipelines", result.Pipelines); err != nil {
			return fmt.Errorf("failed to remove deselected pipelines: %w", err)
		}
	}

	// Cold-start: create initial commit if no commits exist
	if err := createInitialCommit(cmd.ErrOrStderr(), opts.OutputPath); err != nil {
		return err
	}

	printWizardSuccess(cmd, opts.OutputPath, result)
	return nil
}

// runReconfigure re-runs the wizard with existing values as defaults.
func runReconfigure(cmd *cobra.Command, opts InitOptions) error {
	// Read existing manifest
	data, err := os.ReadFile(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("cannot reconfigure: %s not found\nRun 'wave init' first", opts.OutputPath)
	}

	var existing manifest.Manifest
	if err := yaml.Unmarshal(data, &existing); err != nil {
		return fmt.Errorf("failed to parse existing manifest: %w", err)
	}

	// Clear onboarding state so the wizard runs fresh
	_ = onboarding.ClearOnboarding(".agents")

	interactive := !opts.Yes && isInitInteractive()

	// Print Wave logo
	fmt.Fprintln(cmd.OutOrStdout(), tui.WaveLogo())

	// Extract persona configs from existing manifest for buildManifest
	personaConfigs := make(map[string]manifest.Persona)
	for name, p := range existing.Personas {
		personaConfigs[name] = p
	}

	cfg := onboarding.WizardConfig{
		WaveDir:        ".agents",
		Interactive:    interactive,
		Reconfigure:    true,
		Existing:       &existing,
		All:            opts.All,
		Adapter:        opts.Adapter,
		Workspace:      opts.Workspace,
		OutputPath:     opts.OutputPath,
		PersonaConfigs: personaConfigs,
	}

	result, err := onboarding.RunWizard(cfg)
	if err != nil {
		return fmt.Errorf("reconfiguration failed: %w", err)
	}
	// Remove deselected pipelines
	if len(result.Pipelines) > 0 {
		if err := removeDeselectedPipelines(".agents/pipelines", result.Pipelines); err != nil {
			return fmt.Errorf("failed to remove deselected pipelines: %w", err)
		}
	}

	printWizardSuccess(cmd, opts.OutputPath, result)
	return nil
}

// removeDeselectedPipelines deletes pipeline YAML files that are not in the selected list.
func removeDeselectedPipelines(pipelinesDir string, selected []string) error { //nolint:unparam // error return kept for future use
	keep := make(map[string]bool)
	for _, name := range selected {
		keep[name] = true
	}
	entries, err := os.ReadDir(pipelinesDir)
	if err != nil {
		return nil // no dir = nothing to remove
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		if name == e.Name() {
			continue // not a .yaml file
		}
		if !keep[name] {
			_ = os.Remove(filepath.Join(pipelinesDir, e.Name()))
		}
	}
	return nil
}

// printWizardSuccess shows a success message after wizard completion.
func printWizardSuccess(cmd *cobra.Command, outputPath string, result *onboarding.WizardResult) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Onboarding complete!\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Configuration:\n")
	fmt.Fprintf(out, "    %-20s %s\n", "Manifest:", outputPath)
	fmt.Fprintf(out, "    %-20s %s\n", "Adapter:", result.Adapter)
	if result.Model != "" {
		fmt.Fprintf(out, "    %-20s %s\n", "Model:", result.Model)
	}
	if result.Language != "" {
		fmt.Fprintf(out, "    %-20s %s\n", "Language:", result.Language)
	}
	if len(result.Pipelines) > 0 {
		fmt.Fprintf(out, "    %-20s %d selected\n", "Pipelines:", len(result.Pipelines))
	}
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Next steps:\n")
	fmt.Fprintf(out, "    1. Run 'wave validate' to check configuration\n")
	fmt.Fprintf(out, "    2. Run 'wave run' to select and execute a pipeline\n")
	fmt.Fprintf(out, "\n")
}
