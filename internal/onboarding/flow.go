package onboarding

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	"gopkg.in/yaml.v3"
)

// ProjectCommands lets the caller supply or override the detected
// project.test_command / lint_command / build_command. Empty fields keep the
// detected value.
type ProjectCommands struct {
	Test  string
	Lint  string
	Build string
}

// GreenfieldOpts captures the inputs needed for cold-start scaffolding.
type GreenfieldOpts struct {
	Adapter    string
	Model      string
	Workspace  string
	OutputPath string
	All        bool
	Commands   ProjectCommands
	Stderr     io.Writer
}

// GreenfieldResult describes what Greenfield wrote, for the success banner.
type GreenfieldResult struct {
	Assets  *AssetSet
	Flavour *FlavourInfo
	Adapter string
	Model   string
	// Project is the project: block written to wave.yaml (may be empty).
	Project map[string]interface{}
	// InstructionFiles lists the root instruction files created (not the
	// ones that already existed).
	InstructionFiles []string
	// GitignoreUpdated is true when runtime paths were added to .gitignore.
	GitignoreUpdated bool
	// Committed is true when an initial commit was created.
	Committed bool
}

// Greenfield runs the cold-start init flow: ensures the .agents directory tree
// exists, detects the project flavour and forge, builds and writes wave.yaml,
// copies embedded persona/pipeline/contract/prompt assets, seeds AGENTS.md
// (and CLAUDE.md for the claude adapter), ignores runtime state in
// .gitignore and creates an initial git commit if the repository has none.
func Greenfield(o GreenfieldOpts) (*GreenfieldResult, error) {
	if o.Stderr == nil {
		o.Stderr = io.Discard
	}
	adapter := o.Adapter
	if adapter == "" {
		adapter = "claude"
	}
	model := o.Model
	if model == "" {
		model = DefaultModelFor(adapter)
	}
	workspace := o.Workspace
	if workspace == "" {
		workspace = ".agents/workspaces"
	}

	if err := EnsureWaveDirs(waveDirsFor(workspace)); err != nil {
		return nil, err
	}

	cwd, _ := os.Getwd()
	flavour := DetectFlavour(cwd)
	project := FlavourToProjectMap(flavour)
	project = ApplyProjectCommands(project, o.Commands)

	assets, err := LoadAssets(o.Stderr, AssetOptions{All: o.All})
	if err != nil {
		return nil, err
	}

	forgeInfo, _ := forge.DetectFromGitRemotes()
	ScopeAssetsToForge(assets, forgeInfo.Type, o.All)

	meta := ExtractProjectMetadata(cwd)

	manifestMap := BuildManifest(ManifestOptions{
		Adapter:        adapter,
		Model:          model,
		Workspace:      workspace,
		Project:        project,
		PersonaConfigs: assets.PersonaConfigs,
	})
	if metaMap, ok := manifestMap["metadata"].(map[string]interface{}); ok {
		if meta.Name != "" {
			metaMap["name"] = meta.Name
		}
		if meta.Description != "" {
			metaMap["description"] = meta.Description
		}
		if slug := forgeInfo.Slug(); slug != "" {
			metaMap["repo"] = slug
		}
		if ft := forgeMetadataValue(forgeInfo.Type); ft != "" {
			metaMap["forge"] = ft
		}
	}

	manifestData, err := yaml.Marshal(manifestMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	outputDir := filepath.Dir(o.OutputPath)
	if outputDir != "." && outputDir != "" {
		absOutputDir, _ := filepath.Abs(outputDir)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory %s: %w", absOutputDir, err)
		}
	}

	absOutputPath, _ := filepath.Abs(o.OutputPath)
	if err := os.WriteFile(o.OutputPath, manifestData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write manifest to %s: %w", absOutputPath, err)
	}

	if err := CreateExamplePersonas(assets.Personas); err != nil {
		return nil, fmt.Errorf("failed to create example personas in .agents/personas/: %w", err)
	}
	if err := CreateExamplePipelines(assets.Pipelines); err != nil {
		return nil, fmt.Errorf("failed to create example pipelines in .agents/pipelines/: %w", err)
	}
	if err := CreateExampleContracts(assets.Contracts); err != nil {
		return nil, fmt.Errorf("failed to create example contracts in .agents/contracts/: %w", err)
	}
	if err := CreateExamplePrompts(assets.Prompts); err != nil {
		return nil, fmt.Errorf("failed to create example prompts in .agents/prompts/: %w", err)
	}
	created, err := CreateProjectInstructionFiles(".", adapter, meta, project)
	if err != nil {
		return nil, fmt.Errorf("failed to create project instruction files: %w", err)
	}
	gitignoreUpdated, err := EnsureGitignore(".", workspace)
	if err != nil {
		return nil, err
	}
	if err := MarkDoneAt("."); err != nil {
		return nil, fmt.Errorf("mark sentinel: %w", err)
	}

	stage := append([]string{o.OutputPath, ".agents"}, created...)
	if gitignoreUpdated {
		stage = append(stage, ".gitignore")
	}
	committed, err := CreateInitialCommit(o.Stderr, stage)
	if err != nil {
		return nil, err
	}

	return &GreenfieldResult{
		Assets:           assets,
		Flavour:          flavour,
		Adapter:          adapter,
		Model:            model,
		Project:          project,
		InstructionFiles: created,
		GitignoreUpdated: gitignoreUpdated,
		Committed:        committed,
	}, nil
}

// waveDirsFor returns DefaultWaveDirs with the workspace directory replaced
// by the configured workspace root, so --workspace creates the directory the
// manifest points at instead of .agents/workspaces.
func waveDirsFor(workspace string) []string {
	dirs := make([]string, 0, len(DefaultWaveDirs))
	for _, d := range DefaultWaveDirs {
		if d == ".agents/workspaces" {
			d = workspace
		}
		dirs = append(dirs, d)
	}
	return dirs
}

// ApplyProjectCommands overlays non-empty commands onto project, creating the
// map when needed.
func ApplyProjectCommands(project map[string]interface{}, c ProjectCommands) map[string]interface{} {
	if c.Test == "" && c.Lint == "" && c.Build == "" {
		return project
	}
	if project == nil {
		project = map[string]interface{}{}
	}
	for key, val := range map[string]string{
		"test_command":  c.Test,
		"lint_command":  c.Lint,
		"build_command": c.Build,
	} {
		if v := strings.TrimSpace(val); v != "" {
			project[key] = v
		}
	}
	return project
}

// MissingProjectCommands returns the project command keys (test, lint, build)
// that are not set in project.
func MissingProjectCommands(project map[string]interface{}) []string {
	var missing []string
	for _, k := range []string{"test", "lint", "build"} {
		if v, _ := project[k+"_command"].(string); strings.TrimSpace(v) == "" {
			missing = append(missing, k)
		}
	}
	return missing
}

// PersonaFilesFor keeps only the persona prompt files that belong to a
// persona in configs, plus the shared base protocol.
func PersonaFilesFor(files map[string]string, configs map[string]manifest.Persona) map[string]string {
	out := make(map[string]string, len(configs)+1)
	for name, content := range files {
		base := strings.TrimSuffix(name, ".md")
		if _, ok := configs[base]; ok || name == "base-protocol.md" {
			out[name] = content
		}
	}
	return out
}

// forgeMetadataValue maps a detected forge type to the metadata.forge value,
// or "" when nothing useful was detected.
func forgeMetadataValue(ft forge.ForgeType) string {
	switch ft {
	case forge.ForgeUnknown, forge.ForgeLocal, "":
		return ""
	default:
		return string(ft)
	}
}
