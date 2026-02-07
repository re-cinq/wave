package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/defaults"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type InitOptions struct {
	Force      bool
	Merge      bool
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

Use --merge to add default configuration to an existing wave.yaml while
preserving your custom settings.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.Force, "force", false, "Overwrite existing files without prompting")
	cmd.Flags().BoolVar(&opts.Merge, "merge", false, "Merge defaults into existing configuration")
	cmd.Flags().StringVar(&opts.Adapter, "adapter", "claude", "Default adapter to use")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", ".wave/workspaces", "Workspace directory path")
	cmd.Flags().StringVar(&opts.OutputPath, "output", "wave.yaml", "Output path for wave.yaml")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Answer yes to all confirmation prompts")

	return cmd
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
		".wave/traces",
		".wave/workspaces",
	}
	for _, dir := range waveDirs {
		absDir, _ := filepath.Abs(dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", absDir, err)
		}
	}

	manifest := createDefaultManifest(opts.Adapter, opts.Workspace)
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

	if err := createExamplePersonas(); err != nil {
		return fmt.Errorf("failed to create example personas in .wave/personas/: %w", err)
	}

	if err := createExamplePipelines(); err != nil {
		return fmt.Errorf("failed to create example pipelines in .wave/pipelines/: %w", err)
	}

	if err := createExampleContracts(); err != nil {
		return fmt.Errorf("failed to create example contracts in .wave/contracts/: %w", err)
	}

	printInitSuccess(cmd, opts.OutputPath)
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
	defaultManifest := createDefaultManifest(opts.Adapter, opts.Workspace)

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
		".wave/traces",
		".wave/workspaces",
	}
	for _, dir := range waveDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			absDir, _ := filepath.Abs(dir)
			return fmt.Errorf("failed to create directory %s: %w", absDir, err)
		}
	}

	// Create persona files only if they don't exist
	if err := createExamplePersonasIfMissing(); err != nil {
		return fmt.Errorf("failed to create example personas: %w", err)
	}

	// Create pipeline files only if they don't exist
	if err := createExamplePipelinesIfMissing(); err != nil {
		return fmt.Errorf("failed to create example pipelines: %w", err)
	}

	// Create contract files only if they don't exist
	if err := createExampleContractsIfMissing(); err != nil {
		return fmt.Errorf("failed to create example contracts: %w", err)
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

func printInitSuccess(cmd *cobra.Command, outputPath string) {
	out := cmd.OutOrStdout()

	// Get counts from embedded defaults
	personas, _ := defaults.GetPersonas()
	pipelines, _ := defaults.GetPipelines()
	contracts, _ := defaults.GetContracts()

	// Get sorted pipeline names for display
	pipelineNames := make([]string, 0, len(pipelines))
	for name := range pipelines {
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
	fmt.Fprintf(out, "    .wave/personas/          %d persona archetypes\n", len(personas))
	fmt.Fprintf(out, "    .wave/pipelines/         %d pipelines\n", len(pipelines))
	fmt.Fprintf(out, "    .wave/contracts/         %d JSON schema validators\n", len(contracts))
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

func createDefaultManifest(adapter string, workspace string) map[string]interface{} {
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

	return map[string]interface{}{
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
					"allowed_tools": []string{"Read", "Write", "Edit", "Bash"},
					"deny":          []string{"Bash(rm -rf /*)"},
				},
			},
			"auditor": map[string]interface{}{
				"adapter":            adapter,
				"description":        "Security review and quality assurance",
				"system_prompt_file": ".wave/personas/auditor.md",
				"temperature":        0.1,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Grep", "Bash(go vet*)", "Bash(npm audit*)"},
					"deny":          []string{"Write(*)", "Edit(*)"},
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
			"github-pr-creator": map[string]interface{}{
				"adapter":            adapter,
				"description":        "GitHub pull request creation",
				"system_prompt_file": ".wave/personas/github-pr-creator.md",
				"temperature":        0.3,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Write", "Bash(gh *)"},
					"deny":          []string{},
				},
			},
			"implementer": map[string]interface{}{
				"adapter":            adapter,
				"description":        "Execution specialist for code changes and structured output",
				"system_prompt_file": ".wave/personas/implementer.md",
				"temperature":        0.3,
				"permissions": map[string]interface{}{
					"allowed_tools": []string{"Read", "Write", "Edit", "Bash"},
					"deny":          []string{"Bash(rm -rf /*)"},
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
}

func createExamplePersonas() error {
	personas, err := defaults.GetPersonas()
	if err != nil {
		return fmt.Errorf("failed to get default personas: %w", err)
	}

	for filename, content := range personas {
		path := filepath.Join(".wave", "personas", filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			absPath, _ := filepath.Abs(path)
			return fmt.Errorf("failed to write %s: %w", absPath, err)
		}
	}

	return nil
}

func createExamplePersonasIfMissing() error {
	personas, err := defaults.GetPersonas()
	if err != nil {
		return fmt.Errorf("failed to get default personas: %w", err)
	}

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


func createExamplePipelines() error {
	pipelines, err := defaults.GetPipelines()
	if err != nil {
		return fmt.Errorf("failed to get default pipelines: %w", err)
	}

	for filename, content := range pipelines {
		path := filepath.Join(".wave", "pipelines", filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			absPath, _ := filepath.Abs(path)
			return fmt.Errorf("failed to write %s: %w", absPath, err)
		}
	}

	return nil
}

func createExamplePipelinesIfMissing() error {
	pipelines, err := defaults.GetPipelines()
	if err != nil {
		return fmt.Errorf("failed to get default pipelines: %w", err)
	}

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

func createExampleContracts() error {
	contracts, err := defaults.GetContracts()
	if err != nil {
		return fmt.Errorf("failed to get default contracts: %w", err)
	}

	for filename, content := range contracts {
		path := filepath.Join(".wave", "contracts", filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			absPath, _ := filepath.Abs(path)
			return fmt.Errorf("failed to write %s: %w", absPath, err)
		}
	}

	return nil
}

func createExampleContractsIfMissing() error {
	contracts, err := defaults.GetContracts()
	if err != nil {
		return fmt.Errorf("failed to get default contracts: %w", err)
	}

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
