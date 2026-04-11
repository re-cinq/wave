package onboarding

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/recinq/wave/internal/defaults"
	"github.com/recinq/wave/internal/manifest"
	"gopkg.in/yaml.v3"
)

// getDefaultTierModels returns the default tier model mappings for a given adapter.
func getDefaultTierModels(adapter string) map[string]string {
	switch adapter {
	case "claude":
		return map[string]string{
			"cheapest":  "haiku",
			"balanced":  "",
			"strongest": "opus",
		}
	case "opencode":
		return map[string]string{
			"cheapest":  "",
			"balanced":  "",
			"strongest": "",
		}
	case "gemini":
		return map[string]string{
			"cheapest":  "gemini-2.5-flash-lite",
			"balanced":  "gemini-2.5-flash-lite",
			"strongest": "gemini-2.0-pro",
		}
	case "codex":
		return map[string]string{
			"cheapest":  "gpt-4o-mini",
			"balanced":  "gpt-4o",
			"strongest": "o3",
		}
	default:
		return map[string]string{
			"cheapest":  "",
			"balanced":  "",
			"strongest": "",
		}
	}
}

// WizardConfig holds configuration for the onboarding wizard.
type WizardConfig struct {
	WaveDir        string                      // Path to .wave directory
	Interactive    bool                        // false when --yes or no TTY
	Reconfigure    bool                        // true when --reconfigure flag is set
	Existing       *manifest.Manifest          // non-nil when reconfiguring
	All            bool                        // true when --all flag includes all pipelines
	Adapter        string                      // default adapter name
	Workspace      string                      // workspace directory path
	OutputPath     string                      // path for wave.yaml output
	PersonaConfigs map[string]manifest.Persona // persona configs for manifest generation
}

// WizardResult holds the collected results from all wizard steps.
type WizardResult struct {
	Adapter              string
	Model                string
	Flavour              string
	TestCommand          string
	LintCommand          string
	BuildCommand         string
	FormatCommand        string
	Language             string
	SourceGlob           string
	Skill                string   // language skill name for pipeline templates
	Pipelines            []string // selected pipeline names
	Skills               []string // installed skill names from onboarding
	WaveCommandGenerated bool     // true if .claude/commands/wave.md was created
	Dependencies         []DependencyStatus
	OntologyTelos        string                            // project purpose statement
	OntologyContexts     []string                          // bounded context names
	Services             map[string]manifest.ServiceConfig // detected monorepo services
}

// DependencyStatus reports the status of a required dependency.
type DependencyStatus struct {
	Name       string
	Found      bool
	InstallURL string
}

// StepResult holds the output of a single wizard step.
type StepResult struct {
	Skipped bool
	Data    map[string]interface{}
}

// WizardStep defines the interface for individual wizard steps.
type WizardStep interface {
	Name() string
	Run(cfg *WizardConfig) (*StepResult, error)
}

// RunWizard executes the onboarding wizard with all steps.
func RunWizard(cfg WizardConfig) (*WizardResult, error) {
	result := &WizardResult{
		Adapter: cfg.Adapter,
	}

	// Step 1: Dependency verification
	depStep := &DependencyStep{}
	depResult, err := depStep.Run(&cfg)
	if err != nil {
		return nil, fmt.Errorf("dependency verification failed: %w", err)
	}
	if depResult != nil && depResult.Data != nil {
		if deps, ok := depResult.Data["dependencies"].([]DependencyStatus); ok {
			result.Dependencies = deps
		}
	}

	// Step 2: Test command configuration
	testStep := &TestConfigStep{}
	testResult, err := testStep.Run(&cfg)
	if err != nil {
		return nil, fmt.Errorf("test configuration failed: %w", err)
	}
	if testResult != nil && testResult.Data != nil {
		if v, ok := testResult.Data["test_command"].(string); ok {
			result.TestCommand = v
		}
		if v, ok := testResult.Data["lint_command"].(string); ok {
			result.LintCommand = v
		}
		if v, ok := testResult.Data["build_command"].(string); ok {
			result.BuildCommand = v
		}
		if v, ok := testResult.Data["format_command"].(string); ok {
			result.FormatCommand = v
		}
		if v, ok := testResult.Data["language"].(string); ok {
			result.Language = v
		}
		if v, ok := testResult.Data["source_glob"].(string); ok {
			result.SourceGlob = v
		}
		if v, ok := testResult.Data["flavour"].(string); ok {
			result.Flavour = v
		}
		if v, ok := testResult.Data["skill"].(string); ok {
			result.Skill = v
		}
	}

	// Detect monorepo services from metadata + sub-projects
	cwd, _ := os.Getwd()
	meta := ExtractProjectMetadata(cwd)
	if len(meta.SubProjects) > 0 {
		services := make(map[string]manifest.ServiceConfig, len(meta.SubProjects))
		for _, sub := range meta.SubProjects {
			svc := manifest.ServiceConfig{
				Path:     sub.Path,
				Language: sub.Language,
			}
			// Try to detect per-service flavour for commands
			subFlavour := DetectFlavour(filepath.Join(cwd, sub.Path))
			if subFlavour != nil {
				svc.TestCommand = subFlavour.TestCommand
				svc.BuildCommand = subFlavour.BuildCommand
				svc.SourceGlob = subFlavour.SourceGlob
			}
			services[sub.Name] = svc
		}
		result.Services = services
	}

	// Auto-suggest and install skills based on detected project signals
	suggested := suggestSkills(result.Language, meta, cwd)
	if len(suggested) > 0 {
		installed := autoInstallSkills(cfg.WaveDir, suggested)
		result.Skills = mergeSkills(result.Skills, installed)
	}

	// Step 3: Pipeline selection
	pipelineStep := &PipelineSelectionStep{}
	pipelineResult, err := pipelineStep.Run(&cfg)
	if err != nil {
		return nil, fmt.Errorf("pipeline selection failed: %w", err)
	}
	if pipelineResult != nil && pipelineResult.Data != nil {
		if pipelines, ok := pipelineResult.Data["pipelines"].([]string); ok {
			result.Pipelines = pipelines
		}
	}

	// Step 4: Adapter configuration
	adapterStep := &AdapterConfigStep{}
	adapterResult, err := adapterStep.Run(&cfg)
	if err != nil {
		return nil, fmt.Errorf("adapter configuration failed: %w", err)
	}
	if adapterResult != nil && adapterResult.Data != nil {
		if v, ok := adapterResult.Data["adapter"].(string); ok {
			result.Adapter = v
			cfg.Adapter = v // Pass adapter to model step
		}
	}

	// Step 5: Model selection
	modelStep := &ModelSelectionStep{}
	modelResult, err := modelStep.Run(&cfg)
	if err != nil {
		return nil, fmt.Errorf("model selection failed: %w", err)
	}
	if modelResult != nil && modelResult.Data != nil {
		if v, ok := modelResult.Data["model"].(string); ok {
			result.Model = v
		}
	}

	// Step 6: Skill selection
	skillStep := &SkillSelectionStep{}
	skillResult, err := skillStep.Run(&cfg)
	if err != nil {
		return nil, fmt.Errorf("skill selection failed: %w", err)
	}
	if skillResult != nil && skillResult.Data != nil {
		if skills, ok := skillResult.Data["skills"].([]string); ok {
			result.Skills = skills
		}
	}

	// Step 7: Project ontology
	ontologyStep := &OntologyStep{}
	ontologyResult, err := ontologyStep.Run(&cfg)
	if err != nil {
		return nil, fmt.Errorf("ontology configuration failed: %w", err)
	}
	if ontologyResult != nil && !ontologyResult.Skipped && ontologyResult.Data != nil {
		if v, ok := ontologyResult.Data["telos"].(string); ok {
			result.OntologyTelos = v
		}
		if v, ok := ontologyResult.Data["contexts"].([]string); ok {
			result.OntologyContexts = v
		}
	}

	// Step 8: Wave command registration
	waveCommandStep := &WaveCommandStep{}
	waveCommandResult, err := waveCommandStep.Run(&cfg)
	if err != nil {
		return nil, fmt.Errorf("wave command registration failed: %w", err)
	}
	if waveCommandResult != nil && waveCommandResult.Data != nil {
		if v, ok := waveCommandResult.Data["wave_command_generated"].(bool); ok {
			result.WaveCommandGenerated = v
		}
	}

	// Write manifest
	if err := writeManifest(cfg, result); err != nil {
		return nil, fmt.Errorf("failed to write manifest: %w", err)
	}

	// Mark onboarding complete
	if err := MarkOnboarded(cfg.WaveDir); err != nil {
		return nil, fmt.Errorf("failed to mark onboarding complete: %w", err)
	}

	return result, nil
}

// writeManifest creates or updates the wave.yaml manifest from wizard results.
func writeManifest(cfg WizardConfig, result *WizardResult) error {
	m := buildManifest(cfg, result)

	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	return os.WriteFile(cfg.OutputPath, data, 0644)
}

// inferTokenScopes returns recommended token_scopes for a persona based on its permission profile.
// Personas with Bash in their allowed tools are considered forge-interacting with write access.
// Personas with only read-type tools get read-only scopes.
// Returns nil if the persona has no forge-relevant tools.
func inferTokenScopes(pcfg manifest.Persona) []string {
	hasBash := false
	hasReadTool := false
	for _, tool := range pcfg.Permissions.AllowedTools {
		switch tool {
		case "Bash":
			hasBash = true
		case "Read", "Glob", "Grep":
			hasReadTool = true
		}
	}
	if hasBash {
		return []string{"issues:read", "pulls:write", "repos:write"}
	}
	if hasReadTool {
		return []string{"issues:read", "pulls:read"}
	}
	return nil
}

// buildManifest constructs a manifest map from wizard results.
func buildManifest(cfg WizardConfig, result *WizardResult) map[string]interface{} {
	adapter := result.Adapter
	if adapter == "" {
		adapter = "claude"
	}

	// Build tier_models based on adapter type
	tierModels := getDefaultTierModels(adapter)

	m := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "WaveManifest",
		"metadata": map[string]interface{}{
			"name":        "wave-project",
			"description": "A Wave multi-agent project",
		},
		"adapters": map[string]interface{}{
			adapter: map[string]interface{}{
				"binary":        adapter,
				"mode":          "headless",
				"output_format": "json",
				"project_files": []string{"AGENTS.md", ".claude/settings.json"},
				"tier_models":   tierModels,
				"default_permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Write", "Edit", "Bash"},
					"deny":          []string{},
				},
			},
		},
		"runtime": map[string]interface{}{
			"workspace_root":          cfg.Workspace,
			"max_concurrent_workers":  5,
			"default_timeout_minutes": 30,
			"timeouts": map[string]interface{}{
				"step_default_minutes":       5,
				"relay_compaction_minutes":   5,
				"meta_default_minutes":       30,
				"skill_install_seconds":      120,
				"skill_cli_seconds":          120,
				"skill_http_seconds":         120,
				"skill_http_header_seconds":  30,
				"skill_publish_seconds":      30,
				"process_grace_seconds":      3,
				"stdout_drain_seconds":       1,
				"gate_approval_hours":        24,
				"gate_poll_interval_seconds": 30,
				"gate_poll_timeout_minutes":  30,
				"git_command_seconds":        30,
				"forge_api_seconds":          15,
				"retry_max_delay_seconds":    60,
			},
			"relay": map[string]interface{}{
				"token_threshold_percent": 80,
				"strategy":                "summarize_to_checkpoint",
			},
			"audit": map[string]interface{}{
				"log_dir":                 ".wave/traces/",
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

	// Extract project metadata for manifest name/description
	cwd, _ := os.Getwd()
	meta := ExtractProjectMetadata(cwd)
	if metaMap, ok := m["metadata"].(map[string]interface{}); ok {
		if meta.Name != "" {
			metaMap["name"] = meta.Name
		}
		if meta.Description != "" {
			metaMap["description"] = meta.Description
		}
	}

	// Add project configuration if detected
	project := map[string]interface{}{}
	if result.Flavour != "" {
		project["flavour"] = result.Flavour
	}
	if result.Language != "" {
		project["language"] = result.Language
	}
	if result.TestCommand != "" {
		project["test_command"] = result.TestCommand
	}
	if result.LintCommand != "" {
		project["lint_command"] = result.LintCommand
	}
	if result.BuildCommand != "" {
		project["build_command"] = result.BuildCommand
	}
	if result.FormatCommand != "" {
		project["format_command"] = result.FormatCommand
	}
	if result.SourceGlob != "" {
		project["source_glob"] = result.SourceGlob
	}
	if result.Skill != "" {
		project["skill"] = result.Skill
	}
	// Add per-service configurations for monorepo projects
	if len(result.Services) > 0 {
		services := make(map[string]interface{})
		for name, svc := range result.Services {
			entry := map[string]interface{}{}
			if svc.Path != "" {
				entry["path"] = svc.Path
			}
			if svc.Language != "" {
				entry["language"] = svc.Language
			}
			if svc.BuildCommand != "" {
				entry["build_command"] = svc.BuildCommand
			}
			if svc.TestCommand != "" {
				entry["test_command"] = svc.TestCommand
			}
			if svc.ContractTestCommand != "" {
				entry["contract_test_command"] = svc.ContractTestCommand
			}
			if svc.SourceGlob != "" {
				entry["source_glob"] = svc.SourceGlob
			}
			services[name] = entry
		}
		project["services"] = services
	}
	if len(project) > 0 {
		m["project"] = project
	}

	// Build personas section from PersonaConfigs
	if len(cfg.PersonaConfigs) > 0 {
		personas := make(map[string]interface{})
		for name, pcfg := range cfg.PersonaConfigs {
			entry := map[string]interface{}{
				"adapter":            adapter,
				"description":        pcfg.Description,
				"system_prompt_file": fmt.Sprintf(".wave/personas/%s.md", name),
				"temperature":        pcfg.Temperature,
				"permissions": map[string]interface{}{
					"allowed_tools": pcfg.Permissions.AllowedTools,
					"deny":          pcfg.Permissions.Deny,
				},
			}
			if result.Model != "" {
				entry["model"] = result.Model
			} else if pcfg.Model != "" {
				entry["model"] = pcfg.Model
			}
			if scopes := inferTokenScopes(pcfg); len(scopes) > 0 {
				entry["token_scopes"] = scopes
			}
			personas[name] = entry
		}
		m["personas"] = personas
	}

	if len(result.Skills) > 0 {
		m["skills"] = result.Skills
	}

	// Add ontology section — always include base quality context
	ontology := map[string]interface{}{}
	if result.OntologyTelos != "" {
		ontology["telos"] = result.OntologyTelos
	}

	// Build contexts: user-provided + base quality context
	var contexts []map[string]interface{}
	for _, name := range result.OntologyContexts {
		contexts = append(contexts, map[string]interface{}{
			"name": name,
		})
	}

	// Inject base quality context (validation-as-expected-loop)
	hasQuality := false
	for _, name := range result.OntologyContexts {
		if name == "quality" {
			hasQuality = true
			break
		}
	}
	if !hasQuality {
		contexts = append(contexts, map[string]interface{}{
			"name":        "quality",
			"description": "Validation and quality gates — first-pass failure is expected, rework is the norm",
			"invariants": []string{
				"First-pass success is the exception, not the rule — validation exists to catch and correct",
				"Every pipeline output must pass through a validation gate before being considered done",
				"Rework after review is not a failure — it is the expected path to quality",
				"Contract validation, PR review, and test suites are gates, not formalities",
			},
		})
	}

	if len(contexts) > 0 {
		ontology["contexts"] = contexts
	}
	if len(ontology) > 0 {
		m["ontology"] = ontology
	}

	return m
}

// suggestSkills returns skill template names based on detected project signals.
func suggestSkills(language string, meta ProjectMetadata, cwd string) []string {
	var suggested []string

	// Language-based suggestions
	switch language {
	case "go":
		suggested = append(suggested, "testing", "docs", "security")
	case "typescript", "javascript":
		suggested = append(suggested, "react", "tailwind", "testing")
	case "python":
		suggested = append(suggested, "testing", "docs", "security")
	case "rust":
		suggested = append(suggested, "testing", "docs", "security")
	}

	// Check for Docker signals (services detected from compose files)
	if len(meta.Services) > 0 {
		suggested = append(suggested, "docker")
	}

	// Check for GitHub directory
	if _, err := os.Stat(filepath.Join(cwd, ".github")); err == nil {
		suggested = append(suggested, "gh-cli")
	}

	// Filter to only templates that actually exist in bundled defaults
	templates := defaults.GetSkillTemplates()
	var valid []string
	seen := make(map[string]bool)
	for _, name := range suggested {
		if _, ok := templates[name]; ok && !seen[name] {
			seen[name] = true
			valid = append(valid, name)
		}
	}

	return valid
}

// autoInstallSkills copies bundled skill templates to .wave/skills/ for each name.
// Returns the list of successfully installed skill names.
func autoInstallSkills(waveDir string, names []string) []string {
	templates := defaults.GetSkillTemplates()
	skillsDir := filepath.Join(waveDir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return nil
	}

	var installed []string
	for _, name := range names {
		data, ok := templates[name]
		if !ok {
			continue
		}

		destDir := filepath.Join(skillsDir, name)
		destFile := filepath.Join(destDir, "SKILL.md")

		// Skip if already installed
		if _, err := os.Stat(destFile); err == nil {
			installed = append(installed, name)
			continue
		}

		if err := os.MkdirAll(destDir, 0755); err != nil {
			continue
		}

		if err := os.WriteFile(destFile, data, 0644); err != nil {
			continue
		}

		installed = append(installed, name)
	}

	return installed
}
