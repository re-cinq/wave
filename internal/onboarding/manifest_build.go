package onboarding

import (
	"fmt"
	"io"
	"strings"

	"github.com/recinq/wave/internal/defaults"
	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"gopkg.in/yaml.v3"
)

// AssetSet holds the resolved asset maps for init/merge operations.
type AssetSet struct {
	Personas       map[string]string
	PersonaConfigs map[string]manifest.Persona
	Pipelines      map[string]string
	Contracts      map[string]string
	Prompts        map[string]string
}

// AssetOptions selects which embedded defaults to load.
type AssetOptions struct {
	// All disables the release-only filter when true.
	All bool
}

// SystemPersonas are always included in the manifest regardless of pipeline
// references. They are used by relay compaction, meta-pipelines, and adhoc ops.
var SystemPersonas = map[string]bool{
	"summarizer":  true,
	"navigator":   true,
	"philosopher": true,
}

// KnownForgePrefixes lists the prefixes used by forge-specific personas.
var KnownForgePrefixes = []string{"github-", "gitlab-", "bitbucket-", "gitea-"}

// ForgeTypeToPrefix maps forge types to their persona naming convention prefix.
var ForgeTypeToPrefix = map[forge.ForgeType]string{
	forge.ForgeGitHub:    "github",
	forge.ForgeGitLab:    "gitlab",
	forge.ForgeBitbucket: "bitbucket",
	forge.ForgeGitea:     "gitea",
	forge.ForgeCodeberg:  "gitea", // Codeberg is Forgejo — shares Gitea personas
}

// LoadAssets returns the asset maps for init, applying release filtering unless
// opts.All is true. Warnings (unparseable pipelines, missing release pipelines,
// dangling contract refs) are written to warnW.
func LoadAssets(warnW io.Writer, opts AssetOptions) (*AssetSet, error) {
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
		return &AssetSet{
			Personas:       personas,
			PersonaConfigs: allPersonaConfigs,
			Pipelines:      pipelines,
			Contracts:      contracts,
			Prompts:        prompts,
		}, nil
	}

	pipelines, err := defaults.GetReleasePipelines()
	if err != nil {
		return nil, fmt.Errorf("failed to get release pipelines: %w", err)
	}

	if len(pipelines) == 0 && warnW != nil {
		fmt.Fprintf(warnW, "warning: no pipelines are marked with release: true\n")
	}

	allContracts, err := defaults.GetContracts()
	if err != nil {
		return nil, fmt.Errorf("failed to get default contracts: %w", err)
	}
	allPrompts, err := defaults.GetPrompts()
	if err != nil {
		return nil, fmt.Errorf("failed to get default prompts: %w", err)
	}

	contracts, prompts, personaConfigs := FilterTransitiveDeps(warnW, pipelines, allContracts, allPrompts, allPersonaConfigs)

	return &AssetSet{
		Personas:       personas,
		PersonaConfigs: personaConfigs,
		Pipelines:      pipelines,
		Contracts:      contracts,
		Prompts:        prompts,
	}, nil
}

// FilterTransitiveDeps filters contracts, prompts, and persona configs to only
// those referenced by the given pipeline set. System personas are always included.
func FilterTransitiveDeps(warnW io.Writer, pipelines, allContracts, allPrompts map[string]string, allPersonaConfigs map[string]manifest.Persona) (contracts, prompts map[string]string, personaConfigs map[string]manifest.Persona) {
	contractRefs := make(map[string]bool)
	promptRefs := make(map[string]bool)
	personaRefs := make(map[string]bool)

	for name, content := range pipelines {
		var p pipeline.Pipeline
		if err := yaml.Unmarshal([]byte(content), &p); err != nil {
			if warnW != nil {
				fmt.Fprintf(warnW, "warning: failed to parse pipeline %s for dependency resolution: %v\n", name, err)
			}
			continue
		}

		for _, step := range p.Steps {
			if step.Persona != "" {
				personaRefs[step.Persona] = true
			}
			if step.Handover.Compaction.Persona != "" {
				personaRefs[step.Handover.Compaction.Persona] = true
			}
			if sp := step.Handover.Contract.SchemaPath; sp != "" {
				normalized := strings.TrimPrefix(sp, ".agents/contracts/")
				contractRefs[normalized] = true
			}
			if sp := step.Exec.SourcePath; sp != "" {
				if strings.HasPrefix(sp, ".agents/prompts/") {
					normalized := strings.TrimPrefix(sp, ".agents/prompts/")
					promptRefs[normalized] = true
				}
			}
		}
	}

	for name := range SystemPersonas {
		personaRefs[name] = true
	}

	// Expand forge-templated persona refs into all 4 forge variants so they
	// survive filtering. FilterPersonasByForge trims to the detected forge later.
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

	personaConfigs = make(map[string]manifest.Persona)
	for name, cfg := range allPersonaConfigs {
		if personaRefs[name] {
			personaConfigs[name] = cfg
		}
	}

	contracts = make(map[string]string)
	for key, content := range allContracts {
		if contractRefs[key] {
			contracts[key] = content
		}
	}

	for ref := range contractRefs {
		if _, ok := allContracts[ref]; !ok && warnW != nil {
			fmt.Fprintf(warnW, "warning: pipeline references contract %s which is not in embedded defaults\n", ref)
		}
	}

	prompts = make(map[string]string)
	for key, content := range allPrompts {
		if promptRefs[key] {
			prompts[key] = content
		}
	}

	return contracts, prompts, personaConfigs
}

// FilterPersonasByForge filters persona configs to only include personas
// matching the detected forge type. Personas without a known forge prefix are
// always included. A repository without a remote (ForgeLocal) gets no forge
// personas at all: they only work against a forge. An unrecognized remote
// host (ForgeUnknown) keeps every variant, since metadata.forge in wave.yaml
// can still name the forge.
func FilterPersonasByForge(configs map[string]manifest.Persona, ft forge.ForgeType) map[string]manifest.Persona {
	if ft == forge.ForgeUnknown {
		return configs
	}
	if ft == forge.ForgeLocal || ft == "" {
		result := make(map[string]manifest.Persona, len(configs))
		for name, cfg := range configs {
			if !IsForgePersona(name) {
				result[name] = cfg
			}
		}
		return result
	}

	prefix, ok := ForgeTypeToPrefix[ft]
	if !ok {
		return configs
	}

	result := make(map[string]manifest.Persona)
	for name, cfg := range configs {
		hasKnownPrefix := false
		for _, fp := range KnownForgePrefixes {
			if strings.HasPrefix(name, fp) {
				hasKnownPrefix = true
				break
			}
		}
		if strings.HasPrefix(name, prefix+"-") || !hasKnownPrefix {
			result[name] = cfg
		}
	}
	return result
}

// IsForgePersona reports whether a persona name carries a forge prefix
// (github-, gitlab-, bitbucket-, gitea-).
func IsForgePersona(name string) bool {
	for _, fp := range KnownForgePrefixes {
		if strings.HasPrefix(name, fp) {
			return true
		}
	}
	return false
}

// ScopeAssetsToForge drops forge personas (configs and their system prompt
// files) that do not match the detected forge, unless all is set (wave init
// --all installs every asset).
func ScopeAssetsToForge(assets *AssetSet, ft forge.ForgeType, all bool) {
	if assets == nil || all {
		return
	}
	assets.PersonaConfigs = FilterPersonasByForge(assets.PersonaConfigs, ft)
	assets.Personas = PersonaFilesFor(assets.Personas, assets.PersonaConfigs)
}

// FlavourToProjectMap converts a FlavourInfo into the map[string]interface{}
// shape expected by BuildDefaultManifest.
func FlavourToProjectMap(fi *FlavourInfo) map[string]interface{} {
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

// BuildPersonaManifest constructs the personas section of the manifest from
// embedded persona configs.
func BuildPersonaManifest(configs map[string]manifest.Persona, adapter string) map[string]interface{} {
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

// ManifestOptions feeds BuildManifest.
type ManifestOptions struct {
	// Adapter is the single adapter written into adapters: and used by every
	// persona. Empty defaults to claude.
	Adapter string
	// Model pins adapters.<Adapter>.default_model and every tier in
	// tier_models. Empty uses DefaultModelFor(Adapter).
	Model string
	// Workspace is runtime.workspace_root. Empty leaves it unset so the
	// per-project default outside the checkout is used.
	Workspace string
	// Project is attached as the project: block when non-empty.
	Project map[string]interface{}
	// PersonaConfigs are the personas written to the manifest.
	PersonaConfigs map[string]manifest.Persona
}

// BuildDefaultManifest returns a fresh wave.yaml manifest as a generic map,
// suitable for marshalling. It is BuildManifest with the adapter's default
// model.
func BuildDefaultManifest(adapter, workspace string, project map[string]interface{}, personaConfigs map[string]manifest.Persona) map[string]interface{} {
	return BuildManifest(ManifestOptions{
		Adapter:        adapter,
		Workspace:      workspace,
		Project:        project,
		PersonaConfigs: personaConfigs,
	})
}

// BuildManifest returns a fresh wave.yaml manifest as a generic map. Only the
// selected adapter is written, so `wave doctor` does not fail on adapters the
// user never asked for, and its model is pinned for every complexity tier so
// shipped pipelines (which use tier names) never resolve to an outdated
// built-in default.
func BuildManifest(o ManifestOptions) map[string]interface{} {
	adapter := o.Adapter
	if adapter == "" {
		adapter = "claude"
	}
	model := o.Model
	if model == "" {
		model = DefaultModelFor(adapter)
	}

	entry := map[string]interface{}{
		"binary":        adapter,
		"mode":          "headless",
		"output_format": "json",
		"project_files": []string{"AGENTS.md"},
		"default_permissions": map[string]interface{}{
			"allowed_tools": []string{"Read", "Write", "Edit", "Bash"},
			"deny":          []string{},
		},
	}
	if model != "" {
		entry["default_model"] = model
		entry["tier_models"] = TierModelsFor(model)
	}
	adapters := map[string]interface{}{adapter: entry}
	workspace := o.Workspace
	project := o.Project
	personaConfigs := o.PersonaConfigs

	m := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "WaveManifest",
		"metadata": map[string]interface{}{
			"name":        "wave-project",
			"description": "A Wave multi-agent project",
		},
		"adapters": adapters,
		"personas": BuildPersonaManifest(personaConfigs, adapter),
		"runtime": map[string]interface{}{
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

	if workspace != "" {
		m["runtime"].(map[string]interface{})["workspace_root"] = workspace
	}

	if len(project) > 0 {
		m["project"] = project
	}

	return m
}

// MergeManifests deep-merges existing into defaults, preserving existing values
// where keys overlap.
func MergeManifests(defaults, existing map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range defaults {
		result[k] = v
	}

	for k, v := range existing {
		if existingMap, isMap := v.(map[string]interface{}); isMap {
			if defaultMap, isDefaultMap := result[k].(map[string]interface{}); isDefaultMap {
				result[k] = mergeMapsRecursive(defaultMap, existingMap)
			} else {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result
}

func mergeMapsRecursive(defaults, existing map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range defaults {
		result[k] = v
	}

	for k, v := range existing {
		if existingMap, isMap := v.(map[string]interface{}); isMap {
			if defaultMap, isDefaultMap := result[k].(map[string]interface{}); isDefaultMap {
				result[k] = mergeMapsRecursive(defaultMap, existingMap)
			} else {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result
}
