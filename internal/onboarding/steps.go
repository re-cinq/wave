package onboarding

import (
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/charmbracelet/huh"
	"github.com/recinq/wave/internal/tui"
)

// knownAdapters lists the adapters available for selection.
var knownAdapters = []struct {
	Name       string
	Binary     string
	InstallURL string
}{
	{Name: "claude", Binary: "claude", InstallURL: "https://docs.anthropic.com/en/docs/claude-code"},
	{Name: "opencode", Binary: "opencode", InstallURL: "https://opencode.ai"},
	{Name: "ollama", Binary: "ollama", InstallURL: "https://ollama.com"},
}

// adapterModels maps adapter names to their available models.
var adapterModels = map[string][]string{
	"claude":   {"opus", "sonnet", "haiku"},
	"opencode": {"gpt-4o", "gpt-4o-mini", "o3-mini"},
	"ollama":   {"llama3.1", "codellama", "deepseek-coder"},
}

// knownDependencies lists the tools checked during dependency verification.
var knownDependencies = []struct {
	Name       string
	Binary     string
	InstallURL string
}{
	{Name: "GitHub CLI", Binary: "gh", InstallURL: "https://cli.github.com"},
}

// --- DependencyStep ---

// DependencyStep checks for required dependencies on PATH.
type DependencyStep struct{}

func (s *DependencyStep) Name() string { return "Dependency Verification" }

func (s *DependencyStep) Run(cfg *WizardConfig) (*StepResult, error) {
	var deps []DependencyStatus

	// Check known dependencies
	for _, dep := range knownDependencies {
		_, err := exec.LookPath(dep.Binary)
		deps = append(deps, DependencyStatus{
			Name:       dep.Name,
			Found:      err == nil,
			InstallURL: dep.InstallURL,
		})
	}

	// Check adapter binaries
	for _, adapter := range knownAdapters {
		_, err := exec.LookPath(adapter.Binary)
		deps = append(deps, DependencyStatus{
			Name:       adapter.Name + " adapter",
			Found:      err == nil,
			InstallURL: adapter.InstallURL,
		})
	}

	// Report dependency status
	if cfg.Interactive {
		fmt.Fprintf(os.Stderr, "\n  Step 1 of 5 — Dependency Verification\n\n")
		for _, dep := range deps {
			if dep.Found {
				fmt.Fprintf(os.Stderr, "  ✓ %s\n", dep.Name)
			} else {
				fmt.Fprintf(os.Stderr, "  ✗ %s — install: %s\n", dep.Name, dep.InstallURL)
			}
		}
		fmt.Fprintln(os.Stderr)
	}

	return &StepResult{
		Data: map[string]interface{}{
			"dependencies": deps,
		},
	}, nil
}

// --- TestConfigStep ---

// TestConfigStep detects project type and configures test/lint/build commands.
type TestConfigStep struct{}

func (s *TestConfigStep) Name() string { return "Test Command Configuration" }

func (s *TestConfigStep) Run(cfg *WizardConfig) (*StepResult, error) {
	// Detect project type
	detected := detectProjectType()

	testCmd := getStringDefault(detected, "test_command", "")
	lintCmd := getStringDefault(detected, "lint_command", "")
	buildCmd := getStringDefault(detected, "build_command", "")
	language := getStringDefault(detected, "language", "")
	sourceGlob := getStringDefault(detected, "source_glob", "")

	// Pre-fill from existing manifest if reconfiguring
	if cfg.Reconfigure && cfg.Existing != nil && cfg.Existing.Project != nil {
		if cfg.Existing.Project.TestCommand != "" {
			testCmd = cfg.Existing.Project.TestCommand
		}
		if cfg.Existing.Project.LintCommand != "" {
			lintCmd = cfg.Existing.Project.LintCommand
		}
		if cfg.Existing.Project.BuildCommand != "" {
			buildCmd = cfg.Existing.Project.BuildCommand
		}
		if cfg.Existing.Project.Language != "" {
			language = cfg.Existing.Project.Language
		}
		if cfg.Existing.Project.SourceGlob != "" {
			sourceGlob = cfg.Existing.Project.SourceGlob
		}
	}

	if cfg.Interactive {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Test command").
					Value(&testCmd).
					Placeholder("e.g. go test ./..."),
				huh.NewInput().
					Title("Lint command").
					Value(&lintCmd).
					Placeholder("e.g. go vet ./..."),
				huh.NewInput().
					Title("Build command").
					Value(&buildCmd).
					Placeholder("e.g. go build ./..."),
			).Title("Step 2 of 5 — Test Commands").
				Description("Confirm or override the detected project commands."),
		).WithTheme(tui.WaveTheme())

		if err := form.Run(); err != nil {
			if err == huh.ErrUserAborted {
				return nil, fmt.Errorf("wizard cancelled by user")
			}
			return nil, err
		}
	}

	return &StepResult{
		Data: map[string]interface{}{
			"test_command":  testCmd,
			"lint_command":  lintCmd,
			"build_command": buildCmd,
			"language":      language,
			"source_glob":   sourceGlob,
		},
	}, nil
}

// detectProjectType probes the current directory for project type markers.
func detectProjectType() map[string]string {
	fileExists := func(name string) bool {
		_, err := os.Stat(name)
		return err == nil
	}

	switch {
	case fileExists("go.mod"):
		return map[string]string{
			"language":      "go",
			"test_command":  "go test ./...",
			"lint_command":  "go vet ./...",
			"build_command": "go build ./...",
			"source_glob":   "*.go",
		}
	case fileExists("deno.json") || fileExists("deno.jsonc"):
		return map[string]string{
			"language":      "typescript",
			"test_command":  "deno test",
			"lint_command":  "deno lint",
			"build_command": "deno compile",
			"source_glob":   "*.{ts,tsx}",
		}
	case fileExists("package.json"):
		return map[string]string{
			"language":     "javascript",
			"test_command": "npm test",
			"lint_command": "npm run lint",
			"source_glob":  "*.{js,jsx}",
		}
	case fileExists("Cargo.toml"):
		return map[string]string{
			"language":      "rust",
			"test_command":  "cargo test",
			"lint_command":  "cargo clippy",
			"build_command": "cargo build",
			"source_glob":   "*.rs",
		}
	case fileExists("pyproject.toml") || fileExists("setup.py"):
		return map[string]string{
			"language":     "python",
			"test_command": "pytest",
			"lint_command": "ruff check .",
			"source_glob":  "*.py",
		}
	}

	return map[string]string{}
}

func getStringDefault(m map[string]string, key, fallback string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return fallback
}

// --- PipelineSelectionStep ---

// PipelineSelectionStep discovers and presents pipelines for user selection.
type PipelineSelectionStep struct{}

func (s *PipelineSelectionStep) Name() string { return "Pipeline Selection" }

func (s *PipelineSelectionStep) Run(cfg *WizardConfig) (*StepResult, error) {
	pipelinesDir := ".wave/pipelines"
	pipelines, err := tui.DiscoverPipelines(pipelinesDir)
	if err != nil {
		// No pipelines directory yet — not an error during init
		return &StepResult{
			Data: map[string]interface{}{
				"pipelines": []string{},
			},
		}, nil
	}

	if len(pipelines) == 0 {
		return &StepResult{
			Data: map[string]interface{}{
				"pipelines": []string{},
			},
		}, nil
	}

	// Build options grouped by category
	groups := make(map[string][]tui.PipelineInfo)
	for _, p := range pipelines {
		cat := p.Category
		if cat == "" {
			if p.Release {
				cat = "stable"
			} else {
				cat = "experimental"
			}
		}
		groups[cat] = append(groups[cat], p)
	}

	// Sort categories
	categoryOrder := []string{"stable", "experimental", "contrib"}
	var sortedCategories []string
	for _, cat := range categoryOrder {
		if _, ok := groups[cat]; ok {
			sortedCategories = append(sortedCategories, cat)
		}
	}
	// Add any remaining categories not in the predefined order
	for cat := range groups {
		found := false
		for _, sc := range sortedCategories {
			if cat == sc {
				found = true
				break
			}
		}
		if !found {
			sortedCategories = append(sortedCategories, cat)
		}
	}

	// Default selection: all release pipelines (or stable category)
	var selectedPipelines []string
	if !cfg.Interactive {
		// Non-interactive: select all release/stable pipelines
		if cfg.All {
			for _, p := range pipelines {
				selectedPipelines = append(selectedPipelines, p.Name)
			}
		} else {
			for _, p := range pipelines {
				if p.Release || p.Category == "stable" {
					selectedPipelines = append(selectedPipelines, p.Name)
				}
			}
		}
		return &StepResult{
			Data: map[string]interface{}{
				"pipelines": selectedPipelines,
			},
		}, nil
	}

	// Build multi-select options
	var options []huh.Option[string]
	for _, cat := range sortedCategories {
		pipelineList := groups[cat]
		sort.Slice(pipelineList, func(i, j int) bool {
			return pipelineList[i].Name < pipelineList[j].Name
		})
		for _, p := range pipelineList {
			label := fmt.Sprintf("[%s] %s", cat, p.Name)
			if p.Description != "" {
				label = fmt.Sprintf("[%s] %-20s %s", cat, p.Name, p.Description)
			}
			opt := huh.NewOption(label, p.Name)
			// Pre-select release/stable pipelines
			if p.Release || p.Category == "stable" {
				opt = opt.Selected(true)
			}
			options = append(options, opt)
		}
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select pipelines to enable").
				Options(options...).
				Value(&selectedPipelines).
				Height(12),
		).Title("Step 3 of 5 — Pipeline Selection").
			Description("Choose which pipelines to include in your project."),
	).WithTheme(tui.WaveTheme())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return nil, fmt.Errorf("wizard cancelled by user")
		}
		return nil, err
	}

	return &StepResult{
		Data: map[string]interface{}{
			"pipelines": selectedPipelines,
		},
	}, nil
}

// --- AdapterConfigStep ---

// AdapterConfigStep configures the LLM adapter.
type AdapterConfigStep struct{}

func (s *AdapterConfigStep) Name() string { return "Adapter Configuration" }

func (s *AdapterConfigStep) Run(cfg *WizardConfig) (*StepResult, error) {
	selectedAdapter := cfg.Adapter
	if selectedAdapter == "" {
		selectedAdapter = "claude"
	}

	// Pre-fill from existing manifest if reconfiguring
	if cfg.Reconfigure && cfg.Existing != nil {
		for name := range cfg.Existing.Adapters {
			selectedAdapter = name
			break
		}
	}

	if cfg.Interactive {
		var options []huh.Option[string]
		for _, a := range knownAdapters {
			status := "not found"
			if _, err := exec.LookPath(a.Binary); err == nil {
				status = "installed"
			}
			label := fmt.Sprintf("%-12s (%s)", a.Name, status)
			options = append(options, huh.NewOption(label, a.Name))
		}
		options = append(options, huh.NewOption("Other (type manually)", "other"))

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select LLM adapter").
					Options(options...).
					Value(&selectedAdapter),
			).Title("Step 4 of 5 — Adapter Configuration").
				Description("Choose the LLM adapter for pipeline execution."),
		).WithTheme(tui.WaveTheme())

		if err := form.Run(); err != nil {
			if err == huh.ErrUserAborted {
				return nil, fmt.Errorf("wizard cancelled by user")
			}
			return nil, err
		}

		// Handle custom adapter input
		if selectedAdapter == "other" {
			var customAdapter string
			inputForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Enter adapter binary name").
						Value(&customAdapter).
						Placeholder("e.g. my-adapter"),
				),
			).WithTheme(tui.WaveTheme())

			if err := inputForm.Run(); err != nil {
				if err == huh.ErrUserAborted {
					return nil, fmt.Errorf("wizard cancelled by user")
				}
				return nil, err
			}
			if customAdapter != "" {
				selectedAdapter = customAdapter
			}
		}

		// Verify adapter binary exists
		if _, err := exec.LookPath(selectedAdapter); err != nil {
			fmt.Fprintf(os.Stderr, "\n  ⚠ Warning: %s binary not found on PATH. Pipelines will fail until installed.\n\n", selectedAdapter)
		}
	}

	return &StepResult{
		Data: map[string]interface{}{
			"adapter": selectedAdapter,
		},
	}, nil
}

// --- ModelSelectionStep ---

// ModelSelectionStep configures the default model for the selected adapter.
type ModelSelectionStep struct{}

func (s *ModelSelectionStep) Name() string { return "Model Selection" }

func (s *ModelSelectionStep) Run(cfg *WizardConfig) (*StepResult, error) {
	// Determine adapter from prior step result (stored in cfg.Adapter during orchestration)
	adapter := cfg.Adapter
	if adapter == "" {
		adapter = "claude"
	}

	models, knownAdapter := adapterModels[adapter]

	var selectedModel string

	// Pre-fill from existing manifest if reconfiguring
	if cfg.Reconfigure && cfg.Existing != nil {
		for _, persona := range cfg.Existing.Personas {
			if persona.Model != "" {
				selectedModel = persona.Model
				break
			}
		}
	}

	if knownAdapter && len(models) > 0 {
		// Known adapter with model list
		if selectedModel == "" {
			selectedModel = models[0]
		}

		if cfg.Interactive {
			var options []huh.Option[string]
			for _, model := range models {
				options = append(options, huh.NewOption(model, model))
			}
			options = append(options, huh.NewOption("Other (type manually)", "other"))

			form := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Select default model").
						Options(options...).
						Value(&selectedModel),
				).Title("Step 5 of 5 — Model Selection").
					Description("Choose the default model for pipeline execution."),
			).WithTheme(tui.WaveTheme())

			if err := form.Run(); err != nil {
				if err == huh.ErrUserAborted {
					return nil, fmt.Errorf("wizard cancelled by user")
				}
				return nil, err
			}

			if selectedModel == "other" {
				selectedModel = ""
				inputForm := huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Enter model name").
							Value(&selectedModel).
							Placeholder("e.g. claude-3-opus-20240229"),
					),
				).WithTheme(tui.WaveTheme())

				if err := inputForm.Run(); err != nil {
					if err == huh.ErrUserAborted {
						return nil, fmt.Errorf("wizard cancelled by user")
					}
					return nil, err
				}
			}
		}
	} else {
		// Unknown adapter — prompt for free-text model name
		if cfg.Interactive {
			inputForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Enter model name (leave blank for adapter default)").
						Value(&selectedModel).
						Placeholder("e.g. gpt-4o"),
				).Title("Step 5 of 5 — Model Selection").
					Description("Enter the model name for your adapter."),
			).WithTheme(tui.WaveTheme())

			if err := inputForm.Run(); err != nil {
				if err == huh.ErrUserAborted {
					return nil, fmt.Errorf("wizard cancelled by user")
				}
				return nil, err
			}
		}
		// Non-interactive unknown adapter: leave model empty (no override)
	}

	return &StepResult{
		Data: map[string]interface{}{
			"model": selectedModel,
		},
	}, nil
}
