package pipeline

import (
	"testing"

	"github.com/recinq/wave/internal/forge"
)

func TestPipelineContext_ResolvePlaceholders(t *testing.T) {
	ctx := &PipelineContext{
		BranchName:   "018-enhanced-progress",
		FeatureNum:   "018-enhanced-progress",
		PipelineID:   "test-pipeline",
		PipelineName: "feature-worktree",
		StepID:       "test-step",
		SpeckitMode:  true,
		CustomVariables: map[string]string{
			"custom_var": "custom_value",
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "branch_name_resolution",
			template: "specs/{{pipeline_context.branch_name}}/spec.md",
			expected: "specs/018-enhanced-progress/spec.md",
		},
		{
			name:     "multiple_variables",
			template: "{{pipeline_context.pipeline_id}}/{{pipeline_context.step_id}}/output.json",
			expected: "test-pipeline/test-step/output.json",
		},
		{
			name:     "custom_variables",
			template: "path/{{custom_var}}/file.txt",
			expected: "path/custom_value/file.txt",
		},
		{
			name:     "legacy_variables",
			template: "{{pipeline_id}}/{{step_id}}",
			expected: "test-pipeline/test-step",
		},
		{
			name:     "no_variables",
			template: "static/path/file.txt",
			expected: "static/path/file.txt",
		},
		{
			name:     "empty_template",
			template: "",
			expected: "",
		},
		{
			name:     "spaced_pipeline_name",
			template: "feat/{{ pipeline_name }}",
			expected: "feat/feature-worktree",
		},
		{
			name:     "unspaced_pipeline_name",
			template: "feat/{{pipeline_name}}",
			expected: "feat/feature-worktree",
		},
		{
			name:     "spaced_pipeline_context_variable",
			template: "{{ pipeline_context.pipeline_name }}/{{ pipeline_context.step_id }}",
			expected: "feature-worktree/test-step",
		},
		{
			name:     "bare_pipeline_id",
			template: "{{ pipeline_id }}",
			expected: "test-pipeline",
		},
		{
			name:     "bare_pipeline_id_unspaced",
			template: "{{pipeline_id}}",
			expected: "test-pipeline",
		},
		{
			name:     "bare_step_id",
			template: "{{ step_id }}",
			expected: "test-step",
		},
		{
			name:     "spaced_custom_variable",
			template: "path/{{ custom_var }}/file.txt",
			expected: "path/custom_value/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.ResolvePlaceholders(tt.template)
			if result != tt.expected {
				t.Errorf("Expected %s but got %s", tt.expected, result)
			}
		})
	}
}

func TestPipelineContext_GetSpeckitPath(t *testing.T) {
	tests := []struct {
		name        string
		context     *PipelineContext
		filename    string
		expected    string
		description string
	}{
		{
			name: "speckit_mode_with_feature_num",
			context: &PipelineContext{
				BranchName:  "018-enhanced-progress",
				FeatureNum:  "018-enhanced-progress",
				SpeckitMode: true,
			},
			filename:    "spec.md",
			expected:    "specs/018-enhanced-progress/spec.md",
			description: "Should use feature number for Speckit path",
		},
		{
			name: "non_speckit_mode",
			context: &PipelineContext{
				BranchName:  "feature-branch",
				SpeckitMode: false,
			},
			filename:    "spec.md",
			expected:    "specs/999-feature-branch/spec.md",
			description: "Should generate path for branch with dash even in non-Speckit mode",
		},
		{
			name: "branch_with_dash_but_no_feature_num",
			context: &PipelineContext{
				BranchName:  "feature-branch",
				FeatureNum:  "",
				SpeckitMode: false,
			},
			filename:    "plan.md",
			expected:    "specs/999-feature-branch/plan.md",
			description: "Should generate feature directory for dash-containing branch",
		},
		{
			name: "fallback_to_default",
			context: &PipelineContext{
				BranchName:  "",
				FeatureNum:  "",
				SpeckitMode: false,
			},
			filename:    "spec.md",
			expected:    "spec.md",
			description: "Should return filename as-is when no Speckit indicators",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.context.GetSpeckitPath(tt.filename)
			if result != tt.expected {
				t.Errorf("Expected %s but got %s (%s)", tt.expected, result, tt.description)
			}
		})
	}
}

func TestPipelineContext_ResolveArtifactPath(t *testing.T) {
	ctx := &PipelineContext{
		BranchName:  "018-enhanced-progress",
		FeatureNum:  "018-enhanced-progress",
		SpeckitMode: true,
	}

	tests := []struct {
		name     string
		artifact ArtifactDef
		expected string
	}{
		{
			name: "template_path_with_context",
			artifact: ArtifactDef{
				Name: "spec",
				Path: "specs/{{pipeline_context.branch_name}}/spec.md",
				Type: "markdown",
			},
			expected: "specs/018-enhanced-progress/spec.md",
		},
		{
			name: "simple_markdown_filename_speckit_mode",
			artifact: ArtifactDef{
				Name: "plan",
				Path: "plan.md",
				Type: "markdown",
			},
			expected: "specs/018-enhanced-progress/plan.md",
		},
		{
			name: "json_file_no_speckit_transformation",
			artifact: ArtifactDef{
				Name: "analysis",
				Path: ".wave/output/analysis.json",
				Type: "json",
			},
			expected: ".wave/output/analysis.json",
		},
		{
			name: "absolute_path_unchanged",
			artifact: ArtifactDef{
				Name: "external",
				Path: "/absolute/path/file.md",
				Type: "markdown",
			},
			expected: "/absolute/path/file.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.ResolveArtifactPath(tt.artifact)
			if result != tt.expected {
				t.Errorf("Expected %s but got %s", tt.expected, result)
			}
		})
	}
}

func TestExtractFeatureNumber(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		expected   string
	}{
		{
			name:       "standard_speckit_format",
			branchName: "018-enhanced-progress",
			expected:   "018-enhanced-progress",
		},
		{
			name:       "different_number",
			branchName: "001-user-authentication",
			expected:   "001-user-authentication",
		},
		{
			name:       "feature_prefix_with_number",
			branchName: "feature/123-new-feature",
			expected:   "feature/123-new-feature", // Will be transformed in padNumber
		},
		{
			name:       "no_number_pattern",
			branchName: "feature-branch",
			expected:   "",
		},
		{
			name:       "just_number",
			branchName: "123",
			expected:   "",
		},
		{
			name:       "empty_branch",
			branchName: "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFeatureNumber(tt.branchName)
			if result != tt.expected {
				t.Errorf("Expected %s but got %s", tt.expected, result)
			}
		})
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		expected   string
	}{
		{
			name:       "normal_branch",
			branchName: "feature-branch",
			expected:   "feature-branch",
		},
		{
			name:       "with_invalid_chars",
			branchName: "feature/branch:name",
			expected:   "feature-branch-name",
		},
		{
			name:       "consecutive_dashes",
			branchName: "feature--branch---name",
			expected:   "feature-branch-name",
		},
		{
			name:       "leading_trailing_dashes",
			branchName: "-feature-branch-",
			expected:   "feature-branch",
		},
		{
			name:       "very_long_name",
			branchName: "this-is-a-very-long-branch-name-that-should-be-truncated-to-fifty-characters-maximum",
			expected:   "this-is-a-very-long-branch-name-that-should-be-tru",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeBranchName(tt.branchName)
			if result != tt.expected {
				t.Errorf("Expected %s but got %s", tt.expected, result)
			}
			if len(result) > 50 {
				t.Errorf("Result should be max 50 characters but got %d", len(result))
			}
		})
	}
}

func TestPipelineContext_IsSpeckitCompatible(t *testing.T) {
	tests := []struct {
		name     string
		context  *PipelineContext
		expected bool
	}{
		{
			name: "explicit_speckit_mode",
			context: &PipelineContext{
				SpeckitMode: true,
			},
			expected: true,
		},
		{
			name: "branch_with_dash",
			context: &PipelineContext{
				BranchName:  "018-enhanced-progress",
				SpeckitMode: false,
			},
			expected: true,
		},
		{
			name: "feature_number_present",
			context: &PipelineContext{
				FeatureNum:  "001-feature",
				SpeckitMode: false,
			},
			expected: true,
		},
		{
			name: "no_speckit_indicators",
			context: &PipelineContext{
				BranchName:  "main",
				SpeckitMode: false,
			},
			expected: false,
		},
		{
			name: "empty_context",
			context: &PipelineContext{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.context.IsSpeckitCompatible()
			if result != tt.expected {
				t.Errorf("Expected %v but got %v", tt.expected, result)
			}
		})
	}
}

func TestPipelineContext_ToTemplateVars(t *testing.T) {
	ctx := &PipelineContext{
		BranchName:  "018-enhanced-progress",
		FeatureNum:  "018-enhanced-progress",
		PipelineID:  "test-pipeline",
		StepID:      "test-step",
		SpeckitMode: true,
		CustomVariables: map[string]string{
			"custom_var": "custom_value",
		},
	}

	vars := ctx.ToTemplateVars()

	expectedKeys := []string{
		"pipeline_id",
		"step_id",
		"branch_name",
		"feature_num",
		"pipeline_context.branch_name",
		"pipeline_context.feature_num",
		"pipeline_context.pipeline_id",
		"pipeline_context.step_id",
		"custom_var",
	}

	for _, key := range expectedKeys {
		if _, exists := vars[key]; !exists {
			t.Errorf("Expected key %s to exist in template vars", key)
		}
	}

	// Verify values
	if vars["branch_name"] != "018-enhanced-progress" {
		t.Errorf("Expected branch_name to be '018-enhanced-progress' but got %s", vars["branch_name"])
	}

	if vars["custom_var"] != "custom_value" {
		t.Errorf("Expected custom_var to be 'custom_value' but got %s", vars["custom_var"])
	}

	// Verify pipeline_context values match regular values
	if vars["pipeline_context.branch_name"] != vars["branch_name"] {
		t.Errorf("Pipeline context values should match regular values")
	}
}

func TestPipelineContext_ArtifactPathResolution(t *testing.T) {
	ctx := &PipelineContext{
		PipelineID:   "test-pipeline",
		PipelineName: "test",
		StepID:       "step1",
	}

	// Set artifact paths
	ctx.SetArtifactPath("spec", "/workspace/.wave/artifacts/spec")
	ctx.SetArtifactPath("plan", "/workspace/.wave/artifacts/plan")

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "single artifact reference",
			template: "Read the spec at {{ artifacts.spec }}",
			expected: "Read the spec at /workspace/.wave/artifacts/spec",
		},
		{
			name:     "multiple artifact references",
			template: "{{ artifacts.spec }} and {{ artifacts.plan }}",
			expected: "/workspace/.wave/artifacts/spec and /workspace/.wave/artifacts/plan",
		},
		{
			name:     "artifact reference with unspaced syntax",
			template: "Path: {{artifacts.spec}}",
			expected: "Path: /workspace/.wave/artifacts/spec",
		},
		{
			name:     "mixed variables and artifacts",
			template: "{{ pipeline_id }}: {{ artifacts.spec }}",
			expected: "test-pipeline: /workspace/.wave/artifacts/spec",
		},
		{
			name:     "unregistered artifact stays as-is",
			template: "{{ artifacts.unknown }}",
			expected: "{{ artifacts.unknown }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.ResolvePlaceholders(tt.template)
			if result != tt.expected {
				t.Errorf("Expected %q but got %q", tt.expected, result)
			}
		})
	}
}

func TestPipelineContext_GetArtifactPath(t *testing.T) {
	ctx := &PipelineContext{
		PipelineID: "test",
	}

	// Test empty initially
	if path := ctx.GetArtifactPath("nonexistent"); path != "" {
		t.Errorf("Expected empty string for nonexistent artifact, got %q", path)
	}

	// Set and get
	ctx.SetArtifactPath("report", "/path/to/report.json")
	if path := ctx.GetArtifactPath("report"); path != "/path/to/report.json" {
		t.Errorf("Expected '/path/to/report.json', got %q", path)
	}
}

func TestPipelineContext_ToTemplateVars_IncludesArtifacts(t *testing.T) {
	ctx := &PipelineContext{
		PipelineID:   "test",
		PipelineName: "test-pipeline",
		StepID:       "step1",
	}
	ctx.SetArtifactPath("data", "/artifacts/data.json")

	vars := ctx.ToTemplateVars()

	// Verify artifact path is included
	if vars["artifacts.data"] != "/artifacts/data.json" {
		t.Errorf("Expected artifact path to be included in template vars, got %q", vars["artifacts.data"])
	}
}

func TestInjectForgeVariables_GitHub(t *testing.T) {
	ctx := &PipelineContext{
		PipelineID:      "test-pipeline",
		PipelineName:    "implement",
		StepID:          "fetch-assess",
		CustomVariables: make(map[string]string),
	}

	info := forge.ForgeInfo{
		Type:           forge.ForgeGitHub,
		Host:           "github.com",
		Owner:          "recinq",
		Repo:           "wave",
		CLITool:        "gh",
		PipelinePrefix: "gh",
		PRTerm:         "Pull Request",
		PRCommand:      "pr",
	}

	InjectForgeVariables(ctx, info)

	// Verify all 8 forge variables are injected correctly
	expectedVars := map[string]string{
		"forge.type":       "github",
		"forge.host":       "github.com",
		"forge.owner":      "recinq",
		"forge.repo":       "wave",
		"forge.cli_tool":   "gh",
		"forge.prefix":     "gh",
		"forge.pr_term":    "Pull Request",
		"forge.pr_command": "pr",
	}

	for key, want := range expectedVars {
		got := ctx.CustomVariables[key]
		if got != want {
			t.Errorf("CustomVariables[%q] = %q, want %q", key, got, want)
		}
	}
}

func TestInjectForgeVariables_ResolvePlaceholders_GitHub(t *testing.T) {
	ctx := &PipelineContext{
		PipelineID:      "test-pipeline",
		PipelineName:    "implement",
		StepID:          "fetch-assess",
		CustomVariables: make(map[string]string),
	}

	info := forge.ForgeInfo{
		Type:           forge.ForgeGitHub,
		Host:           "github.com",
		Owner:          "recinq",
		Repo:           "wave",
		CLITool:        "gh",
		PipelinePrefix: "gh",
		PRTerm:         "Pull Request",
		PRCommand:      "pr",
	}

	InjectForgeVariables(ctx, info)

	// Verify all forge variables round-trip through ResolvePlaceholders
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "forge.type spaced",
			template: "{{ forge.type }}",
			expected: "github",
		},
		{
			name:     "forge.type unspaced",
			template: "{{forge.type}}",
			expected: "github",
		},
		{
			name:     "forge.host",
			template: "{{ forge.host }}",
			expected: "github.com",
		},
		{
			name:     "forge.owner",
			template: "{{ forge.owner }}",
			expected: "recinq",
		},
		{
			name:     "forge.repo",
			template: "{{ forge.repo }}",
			expected: "wave",
		},
		{
			name:     "forge.cli_tool",
			template: "{{ forge.cli_tool }}",
			expected: "gh",
		},
		{
			name:     "forge.prefix",
			template: "{{ forge.prefix }}",
			expected: "gh",
		},
		{
			name:     "forge.pr_term",
			template: "{{ forge.pr_term }}",
			expected: "Pull Request",
		},
		{
			name:     "forge.pr_command",
			template: "{{ forge.pr_command }}",
			expected: "pr",
		},
		{
			name:     "forge variable in persona resolution",
			template: "{{ forge.prefix }}-commenter",
			expected: "gh-commenter",
		},
		{
			name:     "forge variable in prompt text",
			template: "Use {{ forge.cli_tool }} {{ forge.pr_command }} create to create a {{ forge.pr_term }}",
			expected: "Use gh pr create to create a Pull Request",
		},
		{
			name:     "forge variable mixed with pipeline vars",
			template: "{{ pipeline_name }}: {{ forge.type }} @ {{ forge.owner }}/{{ forge.repo }}",
			expected: "implement: github @ recinq/wave",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.ResolvePlaceholders(tt.template)
			if result != tt.expected {
				t.Errorf("ResolvePlaceholders(%q) = %q, want %q", tt.template, result, tt.expected)
			}
		})
	}
}

func TestInjectForgeVariables_GitLab(t *testing.T) {
	ctx := &PipelineContext{
		PipelineID:      "test-pipeline",
		PipelineName:    "implement",
		StepID:          "create-pr",
		CustomVariables: make(map[string]string),
	}

	info := forge.ForgeInfo{
		Type:           forge.ForgeGitLab,
		Host:           "gitlab.com",
		Owner:          "myorg",
		Repo:           "myrepo",
		CLITool:        "glab",
		PipelinePrefix: "gl",
		PRTerm:         "Merge Request",
		PRCommand:      "mr",
	}

	InjectForgeVariables(ctx, info)

	// Verify GitLab-specific values differ from GitHub
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "forge.type",
			template: "{{ forge.type }}",
			expected: "gitlab",
		},
		{
			name:     "forge.host",
			template: "{{ forge.host }}",
			expected: "gitlab.com",
		},
		{
			name:     "forge.owner",
			template: "{{ forge.owner }}",
			expected: "myorg",
		},
		{
			name:     "forge.repo",
			template: "{{ forge.repo }}",
			expected: "myrepo",
		},
		{
			name:     "forge.cli_tool",
			template: "{{ forge.cli_tool }}",
			expected: "glab",
		},
		{
			name:     "forge.prefix",
			template: "{{ forge.prefix }}",
			expected: "gl",
		},
		{
			name:     "forge.pr_term is Merge Request",
			template: "{{ forge.pr_term }}",
			expected: "Merge Request",
		},
		{
			name:     "forge.pr_command is mr",
			template: "{{ forge.pr_command }}",
			expected: "mr",
		},
		{
			name:     "GitLab persona resolution",
			template: "{{ forge.prefix }}-commenter",
			expected: "gl-commenter",
		},
		{
			name:     "GitLab MR creation command",
			template: "{{ forge.cli_tool }} {{ forge.pr_command }} create",
			expected: "glab mr create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.ResolvePlaceholders(tt.template)
			if result != tt.expected {
				t.Errorf("ResolvePlaceholders(%q) = %q, want %q", tt.template, result, tt.expected)
			}
		})
	}
}