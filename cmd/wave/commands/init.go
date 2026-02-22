package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/defaults"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type InitOptions struct {
	Force      bool
	Merge      bool
	All        bool
	Adapter    string
	Workspace  string
	OutputPath string
	Yes        bool // Skip confirmation prompts
}

func NewInitCmd() *cobra.Command {
	var opts InitOptions

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Wave project",
		Long: `Create a new Wave project structure with default configuration.
Creates a wave.yaml manifest and .wave/personas/ directory with example prompts.

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
	cmd.Flags().StringVar(&opts.Workspace, "workspace", ".wave/workspaces", "Workspace directory path")
	cmd.Flags().StringVar(&opts.OutputPath, "output", "wave.yaml", "Output path for wave.yaml")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Answer yes to all confirmation prompts")

	return cmd
}

// initAssets holds the resolved asset maps for init/merge operations.
type initAssets struct {
	personas  map[string]string
	pipelines map[string]string
	contracts map[string]string
	prompts   map[string]string
}

// getFilteredAssets returns the asset maps for init, applying release filtering
// unless opts.All is true.
func getFilteredAssets(cmd *cobra.Command, opts InitOptions) (*initAssets, error) {
	personas, err := defaults.GetPersonas()
	if err != nil {
		return nil, fmt.Errorf("failed to get default personas: %w", err)
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
			personas:  personas,
			pipelines: pipelines,
			contracts: contracts,
			prompts:   prompts,
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

	contracts, prompts := filterTransitiveDeps(cmd, pipelines, allContracts, allPrompts)

	return &initAssets{
		personas:  personas,
		pipelines: pipelines,
		contracts: contracts,
		prompts:   prompts,
	}, nil
}

// filterTransitiveDeps filters contracts and prompts to only those referenced
// by the given pipeline set. Personas are never filtered.
func filterTransitiveDeps(cmd *cobra.Command, pipelines, allContracts, allPrompts map[string]string) (contracts, prompts map[string]string) {
	contractRefs := make(map[string]bool)
	promptRefs := make(map[string]bool)

	for name, content := range pipelines {
		var p pipeline.Pipeline
		if err := yaml.Unmarshal([]byte(content), &p); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to parse pipeline %s for dependency resolution: %v\n", name, err)
			continue
		}

		for _, step := range p.Steps {
			// Extract contract references from schema_path
			if sp := step.Handover.Contract.SchemaPath; sp != "" {
				normalized := strings.TrimPrefix(sp, ".wave/contracts/")
				contractRefs[normalized] = true
			}

			// Extract prompt references from source_path
			if sp := step.Exec.SourcePath; sp != "" {
				if strings.HasPrefix(sp, ".wave/prompts/") {
					normalized := strings.TrimPrefix(sp, ".wave/prompts/")
					promptRefs[normalized] = true
				}
			}
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

	return contracts, prompts
}

func runInit(cmd *cobra.Command, opts InitOptions) error {
	// Get absolute path for clearer error messages
	absOutputPath, err := filepath.Abs(opts.OutputPath)
	if err != nil {
		absOutputPath = opts.OutputPath
	}

	existingFile, err := os.Stat(opts.OutputPath)
	fileExists := err == nil

	if fileExists {
		if opts.Merge {
			return runMerge(cmd, opts, absOutputPath)
		}

		if !opts.Force && !opts.Yes {
			// Prompt for confirmation
			confirmed, err := confirmOverwrite(cmd, absOutputPath)
			if err != nil {
				return fmt.Errorf("failed to read confirmation: %w", err)
			}
			if !confirmed {
				return fmt.Errorf("aborted: %s already exists (use --force to overwrite or --merge to merge)", absOutputPath)
			}
		} else if !opts.Force {
			return fmt.Errorf("%s already exists (use --force to overwrite or --merge to merge)", absOutputPath)
		}

		// Check file permissions before overwriting
		if existingFile.Mode().Perm()&0200 == 0 {
			return fmt.Errorf("cannot overwrite %s: file is read-only", absOutputPath)
		}
	}

	// Create .wave directory structure
	waveDirs := []string{
		".wave/personas",
		".wave/pipelines",
		".wave/contracts",
		".wave/prompts",
		".wave/traces",
		".wave/workspaces",
	}
	for _, dir := range waveDirs {
		absDir, _ := filepath.Abs(dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", absDir, err)
		}
	}

	project := detectProject()
	manifest := createDefaultManifest(opts.Adapter, opts.Workspace, project)
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

	// Get filtered assets based on --all flag
	assets, err := getFilteredAssets(cmd, opts)
	if err != nil {
		return err
	}

	if err := createExamplePersonas(assets.personas); err != nil {
		return fmt.Errorf("failed to create example personas in .wave/personas/: %w", err)
	}

	if err := createExamplePipelines(assets.pipelines); err != nil {
		return fmt.Errorf("failed to create example pipelines in .wave/pipelines/: %w", err)
	}

	if err := createExampleContracts(assets.contracts); err != nil {
		return fmt.Errorf("failed to create example contracts in .wave/contracts/: %w", err)
	}

	if err := createExamplePrompts(assets.prompts); err != nil {
		return fmt.Errorf("failed to create example prompts in .wave/prompts/: %w", err)
	}

	printInitSuccess(cmd, opts.OutputPath, assets)
	return nil
}

func runMerge(cmd *cobra.Command, opts InitOptions, absOutputPath string) error {
	// Read existing manifest
	existingData, err := os.ReadFile(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to read existing manifest %s: %w", absOutputPath, err)
	}

	var existingManifest map[string]interface{}
	if err := yaml.Unmarshal(existingData, &existingManifest); err != nil {
		return fmt.Errorf("failed to parse existing manifest %s: %w", absOutputPath, err)
	}

	// Create default manifest
	project := detectProject()
	defaultManifest := createDefaultManifest(opts.Adapter, opts.Workspace, project)

	// Merge manifests (existing values take precedence)
	merged := mergeManifests(defaultManifest, existingManifest)

	// Write merged manifest
	mergedData, err := yaml.Marshal(merged)
	if err != nil {
		return fmt.Errorf("failed to marshal merged manifest: %w", err)
	}

	if err := os.WriteFile(opts.OutputPath, mergedData, 0644); err != nil {
		return fmt.Errorf("failed to write merged manifest to %s: %w", absOutputPath, err)
	}

	// Create directories and files if they don't exist
	waveDirs := []string{
		".wave/personas",
		".wave/pipelines",
		".wave/contracts",
		".wave/prompts",
		".wave/traces",
		".wave/workspaces",
	}
	for _, dir := range waveDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			absDir, _ := filepath.Abs(dir)
			return fmt.Errorf("failed to create directory %s: %w", absDir, err)
		}
	}

	// Get filtered assets based on --all flag
	assets, err := getFilteredAssets(cmd, opts)
	if err != nil {
		return err
	}

	// Create persona files only if they don't exist
	if err := createExamplePersonasIfMissing(assets.personas); err != nil {
		return fmt.Errorf("failed to create example personas: %w", err)
	}

	// Create pipeline files only if they don't exist
	if err := createExamplePipelinesIfMissing(assets.pipelines); err != nil {
		return fmt.Errorf("failed to create example pipelines: %w", err)
	}

	// Create contract files only if they don't exist
	if err := createExampleContractsIfMissing(assets.contracts); err != nil {
		return fmt.Errorf("failed to create example contracts: %w", err)
	}

	if err := createExamplePromptsIfMissing(assets.prompts); err != nil {
		return fmt.Errorf("failed to create example prompts: %w", err)
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
	fmt.Fprintf(out, "    .wave/personas/          %d persona archetypes\n", len(assets.personas))
	fmt.Fprintf(out, "    .wave/pipelines/         %d pipelines\n", len(assets.pipelines))
	fmt.Fprintf(out, "    .wave/contracts/         %d JSON schema validators\n", len(assets.contracts))
	fmt.Fprintf(out, "    .wave/prompts/           %d prompt templates\n", len(assets.prompts))
	fmt.Fprintf(out, "    .wave/workspaces/        Ephemeral workspace root\n")
	fmt.Fprintf(out, "    .wave/traces/            Audit log directory\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Pipelines: %s\n", strings.Join(pipelineNames, ", "))
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Next steps:\n")
	fmt.Fprintf(out, "    1. Run 'wave validate' to check configuration\n")
	fmt.Fprintf(out, "    2. Run 'wave run hello-world \"test\"' to verify setup\n")
	fmt.Fprintf(out, "    3. Run 'wave run plan \"your feature\"' to plan a task\n")
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
	fmt.Fprintf(out, "    Created missing .wave/ directories and files\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "  Next steps:\n")
	fmt.Fprintf(out, "    Run 'wave validate' to check configuration\n")
	fmt.Fprintf(out, "\n")
}

// detectProject probes the current directory for project type markers and returns
// a map suitable for inclusion as the "project" key in the manifest YAML.
// Returns nil if no known project type is detected.
func detectProject() map[string]interface{} {
	fileExists := func(name string) bool {
		_, err := os.Stat(name)
		return err == nil
	}

	// Check markers in priority order — first match wins
	switch {
	case fileExists("go.mod"):
		return map[string]interface{}{
			"language":      "go",
			"test_command":  "go test ./...",
			"lint_command":  "go vet ./...",
			"build_command": "go build ./...",
			"source_glob":   "*.go",
		}

	case fileExists("deno.json") || fileExists("deno.jsonc"):
		return map[string]interface{}{
			"language":      "typescript",
			"test_command":  "deno test",
			"lint_command":  "deno lint",
			"build_command": "deno compile",
			"source_glob":   "*.{ts,tsx}",
		}

	case fileExists("package.json"):
		return detectNodeProject()

	case fileExists("Cargo.toml"):
		return map[string]interface{}{
			"language":      "rust",
			"test_command":  "cargo test",
			"lint_command":  "cargo clippy",
			"build_command": "cargo build",
			"source_glob":   "*.rs",
		}

	case fileExists("pyproject.toml") || fileExists("setup.py"):
		return map[string]interface{}{
			"language":     "python",
			"test_command": "pytest",
			"lint_command": "ruff check .",
			"source_glob":  "*.py",
		}
	}

	return nil
}

// detectNodeProject reads package.json to determine the package manager and
// extract actual script commands for test, lint, and build.
func detectNodeProject() map[string]interface{} {
	fileExists := func(name string) bool {
		_, err := os.Stat(name)
		return err == nil
	}

	// Determine package manager from lockfiles
	runner := "npm"
	switch {
	case fileExists("bun.lockb") || fileExists("bun.lock"):
		runner = "bun"
	case fileExists("pnpm-lock.yaml"):
		runner = "pnpm"
	case fileExists("yarn.lock"):
		runner = "yarn"
	}

	// Determine language from tsconfig presence
	language := "javascript"
	sourceGlob := "*.{js,jsx}"
	if fileExists("tsconfig.json") {
		language = "typescript"
		sourceGlob = "*.{ts,tsx}"
	}

	result := map[string]interface{}{
		"language":    language,
		"source_glob": sourceGlob,
	}

	// Read package.json scripts to derive actual commands
	data, err := os.ReadFile("package.json")
	if err != nil {
		return result
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return result
	}

	runCmd := func(script string) string {
		if runner == "npm" {
			return "npm run " + script
		}
		return runner + " run " + script
	}

	// Map well-known script names to project commands
	if _, ok := pkg.Scripts["test"]; ok {
		if runner == "npm" {
			result["test_command"] = "npm test"
		} else {
			result["test_command"] = runner + " test"
		}
	}
	if _, ok := pkg.Scripts["lint"]; ok {
		result["lint_command"] = runCmd("lint")
	}
	if _, ok := pkg.Scripts["build"]; ok {
		result["build_command"] = runCmd("build")
	}

	return result
}

func createDefaultManifest(adapter string, workspace string, project map[string]interface{}) map[string]interface{} {
	adapters := map[string]interface{}{
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
	}

	manifest := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "WaveManifest",
		"metadata": map[string]interface{}{
			"name":        "wave-project",
			"description": "A Wave multi-agent project",
		},
		"adapters": adapters,
		"personas": map[string]interface{}{
			"navigator": map[string]interface{}{
				"adapter":            adapter,
				"description":        "Read-only codebase exploration and analysis",
				"system_prompt_file": ".wave/personas/navigator.md",
				"temperature":        0.1,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Glob", "Grep", "Bash(git log*)", "Bash(git status*)"},
					"deny":          []string{"Write(*)", "Edit(*)", "Bash(git commit*)", "Bash(git push*)"},
				},
			},
			"philosopher": map[string]interface{}{
				"adapter":            adapter,
				"description":        "Architecture design and specification",
				"system_prompt_file": ".wave/personas/philosopher.md",
				"temperature":        0.3,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Write(.wave/specs/*)"},
					"deny":          []string{"Bash(*)"},
				},
			},
			"craftsman": map[string]interface{}{
				"adapter":            adapter,
				"description":        "Code implementation and testing",
				"system_prompt_file": ".wave/personas/craftsman.md",
				"temperature":        0.7,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
					"deny":          []string{"Bash(rm -rf /*)", "Bash(sudo *)"},
				},
			},
			"summarizer": map[string]interface{}{
				"adapter":            adapter,
				"description":        "Context compaction for relay handoffs",
				"system_prompt_file": ".wave/personas/summarizer.md",
				"temperature":        0.0,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read"},
					"deny":          []string{"Write(*)", "Bash(*)"},
				},
			},
			"planner": map[string]interface{}{
				"adapter":            adapter,
				"description":        "Task breakdown and planning",
				"system_prompt_file": ".wave/personas/planner.md",
				"temperature":        0.2,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Write(.wave/plans/*)"},
					"deny":          []string{"Bash(*)"},
				},
			},
			"debugger": map[string]interface{}{
				"adapter":            adapter,
				"description":        "Systematic debugging and root cause analysis",
				"system_prompt_file": ".wave/personas/debugger.md",
				"temperature":        0.1,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Glob", "Grep", "Bash(git log*)", "Bash(git bisect*)"},
					"deny":          []string{"Write(*)", "Edit(*)"},
				},
			},
			"github-analyst": map[string]interface{}{
				"adapter":            adapter,
				"description":        "GitHub issue analysis and scanning",
				"system_prompt_file": ".wave/personas/github-analyst.md",
				"temperature":        0.1,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Write", "Bash(gh *)"},
					"deny":          []string{},
				},
			},
			"github-enhancer": map[string]interface{}{
				"adapter":            adapter,
				"description":        "GitHub issue enhancement and improvement",
				"system_prompt_file": ".wave/personas/github-enhancer.md",
				"temperature":        0.2,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Write", "Bash(gh *)"},
					"deny":          []string{},
				},
			},
			"researcher": map[string]interface{}{
				"adapter":            adapter,
				"description":        "Deep codebase research and analysis",
				"system_prompt_file": ".wave/personas/researcher.md",
				"temperature":        0.1,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Glob", "Grep", "Bash(gh *)", "Bash(git log*)"},
					"deny":          []string{"Write(*)", "Edit(*)"},
				},
			},
			"reviewer": map[string]interface{}{
				"adapter":            adapter,
				"description":        "Quality and security review, validation, and assessment",
				"system_prompt_file": ".wave/personas/reviewer.md",
				"temperature":        0.1,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Glob", "Grep", "Bash(git diff*)", "Bash(git log*)"},
					"deny":          []string{"Write(*)", "Edit(*)"},
				},
			},
			"github-commenter": map[string]interface{}{
				"adapter":            adapter,
				"description":        "Posts comments on GitHub issues",
				"system_prompt_file": ".wave/personas/github-commenter.md",
				"temperature":        0.2,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Bash(gh *)"},
					"deny":          []string{},
				},
			},
			"provocateur": map[string]interface{}{
				"adapter":            adapter,
				"description":        "Creative challenger for divergent thinking and complexity hunting",
				"model":              "opus",
				"system_prompt_file": ".wave/personas/provocateur.md",
				"temperature":        0.8,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Glob", "Grep", "Bash(wc *)", "Bash(git log*)", "Bash(git diff*)", "Bash(find*)", "Bash(ls*)"},
					"deny":          []string{"Write(*)", "Edit(*)", "Bash(git commit*)", "Bash(git push*)", "Bash(rm*)"},
				},
			},
			"validator": map[string]interface{}{
				"adapter":            adapter,
				"description":        "Skeptical analysis and verification of findings against source code",
				"model":              "sonnet",
				"system_prompt_file": ".wave/personas/validator.md",
				"temperature":        0.1,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Glob", "Grep", "Bash(wc *)", "Bash(git log*)", "Bash(git diff*)"},
					"deny":          []string{"Write(*)", "Edit(*)", "Bash(git commit*)", "Bash(git push*)", "Bash(rm*)"},
				},
			},
			"synthesizer": map[string]interface{}{
				"adapter":            adapter,
				"description":        "Structured synthesis of analysis findings into actionable JSON proposals",
				"model":              "sonnet",
				"system_prompt_file": ".wave/personas/synthesizer.md",
				"temperature":        0.2,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Glob", "Grep"},
					"deny":          []string{"Edit(*)", "Bash(*)"},
				},
			},
		},
		"runtime": map[string]interface{}{
			"workspace_root":          workspace,
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
		"skill_mounts": []map[string]interface{}{
			{"path": ".wave/skills/"},
		},
	}

	if project != nil {
		manifest["project"] = project
	}

	return manifest
}

func createExamplePersonas(personas map[string]string) error {
	for filename, content := range personas {
		path := filepath.Join(".wave", "personas", filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			absPath, _ := filepath.Abs(path)
			return fmt.Errorf("failed to write %s: %w", absPath, err)
		}
	}

	return nil
}

func createExamplePersonasIfMissing(personas map[string]string) error {
	for filename, content := range personas {
		path := filepath.Join(".wave", "personas", filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				absPath, _ := filepath.Abs(path)
				return fmt.Errorf("failed to write %s: %w", absPath, err)
			}
		}
	}

	return nil
}

func createExamplePipelines(pipelines map[string]string) error {
	for filename, content := range pipelines {
		path := filepath.Join(".wave", "pipelines", filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			absPath, _ := filepath.Abs(path)
			return fmt.Errorf("failed to write %s: %w", absPath, err)
		}
	}

	return nil
}

func createExamplePipelinesIfMissing(pipelines map[string]string) error {
	for filename, content := range pipelines {
		path := filepath.Join(".wave", "pipelines", filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				absPath, _ := filepath.Abs(path)
				return fmt.Errorf("failed to write %s: %w", absPath, err)
			}
		}
	}

	return nil
}

func createExampleContracts(contracts map[string]string) error {
	for filename, content := range contracts {
		path := filepath.Join(".wave", "contracts", filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			absPath, _ := filepath.Abs(path)
			return fmt.Errorf("failed to write %s: %w", absPath, err)
		}
	}

	return nil
}

func createExampleContractsIfMissing(contracts map[string]string) error {
	for filename, content := range contracts {
		path := filepath.Join(".wave", "contracts", filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				absPath, _ := filepath.Abs(path)
				return fmt.Errorf("failed to write %s: %w", absPath, err)
			}
		}
	}

	return nil
}

func createExamplePrompts(prompts map[string]string) error {
	for relPath, content := range prompts {
		path := filepath.Join(".wave", "prompts", relPath)
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

func createExamplePromptsIfMissing(prompts map[string]string) error {
	for relPath, content := range prompts {
		path := filepath.Join(".wave", "prompts", relPath)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", path, err)
			}
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				absPath, _ := filepath.Abs(path)
				return fmt.Errorf("failed to write %s: %w", absPath, err)
			}
		}
	}

	return nil
}
