package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	fmt.Fprintf(out, "Initialized Wave project\n")
	fmt.Fprintf(out, "  - Created %s\n", outputPath)
	fmt.Fprintf(out, "  - Created .wave/personas/ (5 persona archetypes)\n")
	fmt.Fprintf(out, "  - Created .wave/pipelines/ (speckit-flow, hotfix)\n")
	fmt.Fprintf(out, "  - Created .wave/contracts/ (navigation, specification schemas)\n")
	fmt.Fprintf(out, "  - Created .wave/workspaces/ (ephemeral workspace root)\n")
	fmt.Fprintf(out, "  - Created .wave/traces/ (audit log directory)\n")
	fmt.Fprintf(out, "\nNext steps:\n")
	fmt.Fprintf(out, "  - Edit %s to configure adapters and personas\n", outputPath)
	fmt.Fprintf(out, "  - Run 'wave validate' to check configuration\n")
	fmt.Fprintf(out, "  - Run 'wave run --pipeline speckit-flow --input \"your task\"' to execute\n")
}

func printMergeSuccess(cmd *cobra.Command, outputPath string) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Merged defaults into Wave project\n")
	fmt.Fprintf(out, "  - Updated %s (preserved your settings)\n", outputPath)
	fmt.Fprintf(out, "  - Added missing default adapters and personas\n")
	fmt.Fprintf(out, "  - Created missing .wave/ directories and files\n")
	fmt.Fprintf(out, "\nNext steps:\n")
	fmt.Fprintf(out, "  - Run 'wave validate' to check configuration\n")
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
	personas := getPersonaContents()

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
	personas := getPersonaContents()

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

func getPersonaContents() map[string]string {
	return map[string]string{
		"navigator.md": `# Navigator

You are a codebase exploration specialist. Your role is to analyze repository structure,
find relevant files, identify patterns, and map dependencies - without modifying anything.

## Responsibilities
- Search and read source files to understand architecture
- Identify relevant code paths for the given task
- Map dependencies between modules and packages
- Report existing patterns (naming conventions, error handling, testing)
- Assess potential impact areas for proposed changes

## Output Format
Always output structured JSON with keys: files, patterns, dependencies, impact_areas

## Constraints
- NEVER write, edit, or delete any files
- NEVER run destructive commands
- Focus on accuracy over speed - missing a relevant file is worse than taking longer
- Report uncertainty explicitly ("unsure if X relates to Y")`,

		"philosopher.md": `# Philosopher

You are a software architect and specification writer. Your role is to transform
analysis reports into detailed, actionable specifications and implementation plans.

## Responsibilities
- Create feature specifications with user stories and acceptance criteria
- Design data models, API schemas, and system interfaces
- Identify edge cases, error scenarios, and security considerations
- Break complex features into ordered implementation steps
- Produce clear, unambiguous technical documentation

## Output Format
Write specifications in markdown with clear sections: Overview, User Stories,
Data Model, API Design, Edge Cases, Testing Strategy

## Constraints
- NEVER write implementation code - only specifications and plans
- NEVER execute shell commands
- Ground all designs in the navigation analysis - don't invent architecture
- Flag assumptions explicitly when the analysis is ambiguous`,

		"craftsman.md": `# Craftsman

You are a senior software developer focused on clean, maintainable implementation.
Your role is to write production-quality code following the specification and plan.

## Responsibilities
- Implement features according to the provided specification
- Write comprehensive tests (unit, integration) for all new code
- Follow existing project patterns and conventions
- Handle errors gracefully with meaningful messages
- Run tests to verify implementation correctness

## Guidelines
- Read the spec and plan artifacts before writing any code
- Follow existing patterns in the codebase - consistency matters
- Write tests BEFORE or alongside implementation, not after
- Keep changes minimal and focused - don't refactor unrelated code
- Run the full test suite before declaring completion

## Constraints
- Stay within the scope of the specification - no feature creep
- Never delete or overwrite test fixtures without explicit instruction
- If the spec is ambiguous, implement the simpler interpretation`,

		"auditor.md": `# Auditor

You are a security and quality reviewer. Your role is to review implementations
for vulnerabilities, bugs, and quality issues without modifying code.

## Responsibilities
- Review for OWASP Top 10 vulnerabilities (injection, XSS, CSRF, etc.)
- Check authentication and authorization correctness
- Verify input validation and error handling completeness
- Assess test coverage and test quality
- Identify performance regressions and resource leaks
- Check code style consistency with project conventions

## Output Format
Produce a structured review report with severity ratings:
- CRITICAL: Security vulnerabilities, data loss risks
- HIGH: Logic errors, missing auth checks, resource leaks
- MEDIUM: Missing edge case handling, incomplete validation
- LOW: Style issues, minor improvements, documentation gaps

## Constraints
- NEVER modify any source files
- NEVER run destructive commands
- Be specific - cite file paths and line numbers
- Distinguish between confirmed issues and potential concerns`,

		"summarizer.md": `# Summarizer

You are a context compaction specialist. Your role is to distill long conversation
histories into concise checkpoint summaries that preserve essential context.

## Responsibilities
- Summarize key decisions and their rationale
- Preserve file paths, function names, and technical specifics
- Maintain the thread of what was attempted and what worked
- Flag any unresolved issues or pending decisions
- Keep summaries under 2000 tokens while retaining critical context

## Output Format
Write checkpoint summaries in markdown with sections:
- Objective: What is being accomplished
- Progress: What has been done so far
- Key Decisions: Important choices and their rationale
- Current State: Where things stand now
- Next Steps: What remains to be done

## Constraints
- NEVER modify any files
- NEVER run any commands
- Accuracy over brevity - never lose a key technical detail
- Include exact file paths and identifiers, not paraphrases`,
	}
}

func createExamplePipelines() error {
	pipelines := getPipelineContents()

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
	pipelines := getPipelineContents()

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

func getPipelineContents() map[string]string {
	return map[string]string{
		"speckit-flow.yaml": `kind: WavePipeline
metadata:
  name: speckit-flow
  description: "Specification-driven feature development"

input:
  source: cli

steps:
  - id: navigate
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Analyze the codebase for: {{ input }}

        Find and report:
        1. Relevant source files and their purposes
        2. Existing patterns (naming, architecture, testing)
        3. Dependencies and integration points
        4. Potential impact areas

        Output as structured JSON with keys:
        files, patterns, dependencies, impact_areas
    output_artifacts:
      - name: analysis
        path: output/analysis.json
        type: json
    handover:
      contract:
        type: json_schema
        schema: .wave/contracts/navigation.schema.json
        source: output/analysis.json
        on_failure: retry
        max_retries: 2

  - id: specify
    persona: philosopher
    dependencies: [navigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: navigate
          artifact: analysis
          as: navigation_report
    exec:
      type: prompt
      source: |
        Based on the navigation report, create a feature specification for: {{ input }}

        Include:
        1. User stories with acceptance criteria
        2. Data model changes
        3. API design (endpoints, request/response schemas)
        4. Edge cases and error handling
        5. Testing strategy
    output_artifacts:
      - name: spec
        path: output/spec.md
        type: markdown
    handover:
      contract:
        type: json_schema
        schema: .wave/contracts/specification.schema.json
        source: output/spec.json
        on_failure: retry
        max_retries: 2

  - id: plan
    persona: philosopher
    dependencies: [specify]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: navigate
          artifact: analysis
          as: navigation_report
        - step: specify
          artifact: spec
          as: feature_spec
    exec:
      type: prompt
      source: |
        Create an implementation plan for the feature specification.

        Include:
        1. Ordered list of implementation steps
        2. File-by-file change descriptions
        3. Testing plan (unit, integration)
        4. Risk assessment
    output_artifacts:
      - name: plan
        path: output/plan.md
        type: markdown

  - id: implement
    persona: craftsman
    dependencies: [plan]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: specify
          artifact: spec
          as: feature_spec
        - step: plan
          artifact: plan
          as: implementation_plan
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readwrite
    exec:
      type: prompt
      source: |
        Implement the feature according to the plan.

        Follow the implementation plan step by step:
        1. Make code changes as specified
        2. Write tests for all new functionality
        3. Run existing tests to prevent regressions
        4. Document public APIs
    handover:
      contract:
        type: test_suite
        command: "go test ./..."
        must_pass: true
        on_failure: retry
        max_retries: 3
      compaction:
        trigger: "token_limit_80%"
        persona: summarizer

  - id: review
    persona: auditor
    dependencies: [implement]
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: |
        Review the implementation for:

        Security:
        - SQL injection, XSS, CSRF vulnerabilities
        - Authentication/authorization gaps
        - Input validation completeness

        Quality:
        - Error handling coverage
        - Test coverage and quality
        - Code style consistency
        - Performance implications

        Output a structured review report with severity ratings.
    output_artifacts:
      - name: review
        path: output/review.md
        type: markdown
`,
		"hotfix.yaml": `kind: WavePipeline
metadata:
  name: hotfix
  description: "Quick investigation and fix for production issues"

input:
  source: cli

steps:
  - id: investigate
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Investigate this production issue: {{ input }}

        1. Search for related code paths
        2. Check recent commits that may have introduced the bug
        3. Identify the root cause
        4. Assess blast radius (what else could be affected)

        Output structured findings as JSON:
        {
          "root_cause": "description",
          "affected_files": ["path1", "path2"],
          "recent_commits": ["hash1", "hash2"],
          "blast_radius": "assessment",
          "fix_approach": "recommended approach"
        }
    output_artifacts:
      - name: findings
        path: output/findings.json
        type: json
    handover:
      contract:
        type: json_schema
        source: output/findings.json
        on_failure: retry
        max_retries: 2

  - id: fix
    persona: craftsman
    dependencies: [investigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: investigate
          artifact: findings
          as: investigation
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readwrite
    exec:
      type: prompt
      source: |
        Fix the production issue based on the investigation findings.

        Requirements:
        1. Apply the minimal fix - don't refactor surrounding code
        2. Add a regression test that would have caught this bug
        3. Ensure all existing tests still pass
        4. Document the fix in a commit-ready message
    handover:
      contract:
        type: test_suite
        command: "go test ./... -count=1"
        must_pass: true
        on_failure: retry
        max_retries: 3

  - id: verify
    persona: auditor
    dependencies: [fix]
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: |
        Verify the hotfix:

        1. Is the fix minimal and focused? (no unrelated changes)
        2. Does the regression test actually test the reported issue?
        3. Are there other code paths with the same vulnerability?
        4. Is the fix safe for production deployment?

        Output a go/no-go recommendation with reasoning.
    output_artifacts:
      - name: verdict
        path: output/verdict.md
        type: markdown
`,
	}
}

func createExampleContracts() error {
	contracts := getContractContents()

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
	contracts := getContractContents()

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

func getContractContents() map[string]string {
	return map[string]string{
		"navigation.schema.json": `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["files", "patterns", "dependencies", "impact_areas"],
  "properties": {
    "files": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["path", "purpose"],
        "properties": {
          "path": { "type": "string" },
          "purpose": { "type": "string" }
        }
      },
      "minItems": 1
    },
    "patterns": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name", "description"],
        "properties": {
          "name": { "type": "string" },
          "description": { "type": "string" }
        }
      }
    },
    "dependencies": { "type": "object" },
    "impact_areas": {
      "type": "array",
      "items": { "type": "string" }
    }
  }
}`,
		"specification.schema.json": `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["title", "user_stories", "data_model"],
  "properties": {
    "title": { "type": "string", "minLength": 5 },
    "user_stories": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["as_a", "i_want", "so_that", "acceptance_criteria"],
        "properties": {
          "as_a": { "type": "string" },
          "i_want": { "type": "string" },
          "so_that": { "type": "string" },
          "acceptance_criteria": {
            "type": "array",
            "items": { "type": "string" },
            "minItems": 1
          }
        }
      },
      "minItems": 1
    },
    "data_model": { "type": "object" },
    "api_design": { "type": "object" },
    "edge_cases": {
      "type": "array",
      "items": { "type": "string" }
    },
    "testing_strategy": { "type": "object" }
  }
}`,
	}
}
