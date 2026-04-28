package onboarding

import (
	"fmt"
	"io"
	"sort"
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
// always included.
func FilterPersonasByForge(configs map[string]manifest.Persona, ft forge.ForgeType) map[string]manifest.Persona {
	if ft == forge.ForgeUnknown {
		return configs
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

// BuildDefaultManifest returns a fresh wave.yaml manifest as a generic map,
// suitable for marshalling. The project map (typically derived from FlavourInfo)
// is attached only when non-nil.
func BuildDefaultManifest(adapter, workspace string, project map[string]interface{}, personaConfigs map[string]manifest.Persona) map[string]interface{} {
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
		m["project"] = project
	}

	m["ontology"] = map[string]interface{}{
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

// MergeTypedManifests merges a generated manifest into an existing one,
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
func MergeTypedManifests(existing, generated *manifest.Manifest) *manifest.Manifest {
	result := &manifest.Manifest{}

	result.APIVersion = generated.APIVersion
	result.Kind = generated.Kind

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

	result.Adapters = make(map[string]manifest.Adapter)
	for name, adapter := range generated.Adapters {
		result.Adapters[name] = adapter
	}
	for name, adapter := range existing.Adapters {
		result.Adapters[name] = adapter
	}

	result.Personas = make(map[string]manifest.Persona)
	for name, persona := range generated.Personas {
		result.Personas[name] = persona
	}
	for name, persona := range existing.Personas {
		result.Personas[name] = persona
	}

	if existing.Ontology != nil {
		result.Ontology = existing.Ontology
	} else {
		result.Ontology = generated.Ontology
	}

	if existing.Project != nil {
		result.Project = existing.Project
	} else {
		result.Project = generated.Project
	}

	result.Runtime = generated.Runtime

	if existing.Server != nil {
		result.Server = existing.Server
	} else {
		result.Server = generated.Server
	}

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

	if len(existing.Hooks) > 0 {
		result.Hooks = existing.Hooks
	} else {
		result.Hooks = generated.Hooks
	}

	return result
}
