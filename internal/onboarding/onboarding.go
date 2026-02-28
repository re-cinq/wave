package onboarding

import (
	"fmt"
	"os"

	"github.com/recinq/wave/internal/manifest"
	"gopkg.in/yaml.v3"
)

// WizardConfig holds configuration for the onboarding wizard.
type WizardConfig struct {
	WaveDir     string             // Path to .wave directory
	Interactive bool               // false when --yes or no TTY
	Reconfigure bool               // true when --reconfigure flag is set
	Existing    *manifest.Manifest // non-nil when reconfiguring
	All         bool               // true when --all flag includes all pipelines
	Adapter     string             // default adapter name
	Workspace   string             // workspace directory path
	OutputPath  string             // path for wave.yaml output
}

// WizardResult holds the collected results from all wizard steps.
type WizardResult struct {
	Adapter      string
	Model        string
	TestCommand  string
	LintCommand  string
	BuildCommand string
	Language     string
	SourceGlob   string
	Pipelines    []string // selected pipeline names
	Dependencies []DependencyStatus
}

// DependencyStatus reports the status of a required dependency.
type DependencyStatus struct {
	Name      string
	Found     bool
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
		if v, ok := testResult.Data["language"].(string); ok {
			result.Language = v
		}
		if v, ok := testResult.Data["source_glob"].(string); ok {
			result.SourceGlob = v
		}
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

// buildManifest constructs a manifest map from wizard results.
func buildManifest(cfg WizardConfig, result *WizardResult) map[string]interface{} {
	adapter := result.Adapter
	if adapter == "" {
		adapter = "claude"
	}

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
				"project_files": []string{"CLAUDE.md", ".claude/settings.json"},
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

	// Add project configuration if detected
	project := map[string]interface{}{}
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
	if result.SourceGlob != "" {
		project["source_glob"] = result.SourceGlob
	}
	if len(project) > 0 {
		m["project"] = project
	}

	// Add model if specified
	if result.Model != "" {
		// Model is set at the adapter level in manifest
		adapterCfg := m["adapters"].(map[string]interface{})[adapter].(map[string]interface{})
		adapterCfg["model"] = result.Model
	}

	return m
}
