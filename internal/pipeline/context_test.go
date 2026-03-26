package pipeline

import (
	"strings"
	"testing"

	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
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

func TestNewContextWithProject_AllFields(t *testing.T) {
	m := &manifest.Manifest{
		Project: &manifest.Project{
			Language:      "rust",
			Flavour:       "rust",
			TestCommand:   "cargo test",
			LintCommand:   "cargo clippy -- -D warnings",
			BuildCommand:  "cargo build",
			FormatCommand: "cargo fmt -- --check",
			SourceGlob:    "*.rs",
			Skill:         "rust",
		},
	}

	ctx := newContextWithProject("pipe-123", "implement", "build-step", m)

	expectedVars := map[string]string{
		"project.language":       "rust",
		"project.flavour":        "rust",
		"project.test_command":   "cargo test",
		"project.lint_command":   "cargo clippy -- -D warnings",
		"project.build_command":  "cargo build",
		"project.format_command": "cargo fmt -- --check",
		"project.source_glob":    "*.rs",
		"project.skill":          "rust",
	}

	for key, want := range expectedVars {
		got := ctx.CustomVariables[key]
		if got != want {
			t.Errorf("CustomVariables[%q] = %q, want %q", key, got, want)
		}
	}
}

func TestNewContextWithProject_NilProject(t *testing.T) {
	m := &manifest.Manifest{Project: nil}
	ctx := newContextWithProject("pipe-123", "implement", "step", m)

	// Should not have any project.* keys
	for k := range ctx.CustomVariables {
		if len(k) > 8 && k[:8] == "project." {
			t.Errorf("unexpected project variable %q with nil project", k)
		}
	}
}

func TestProjectSkillTemplateResolution(t *testing.T) {
	m := &manifest.Manifest{
		Project: &manifest.Project{
			Skill:       "golang",
			TestCommand: "go test -race ./...",
		},
	}

	ctx := newContextWithProject("pipe-123", "implement", "step", m)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "project.skill spaced",
			template: "{{ project.skill }}",
			expected: "golang",
		},
		{
			name:     "project.skill unspaced",
			template: "{{project.skill}}",
			expected: "golang",
		},
		{
			name:     "project.test_command in prompt",
			template: "Run: {{ project.test_command }}",
			expected: "Run: go test -race ./...",
		},
		{
			name:     "mixed project and pipeline vars",
			template: "{{ pipeline_name }}: {{ project.skill }}",
			expected: "implement: golang",
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

func TestInjectForgeVariables_Bitbucket(t *testing.T) {
	ctx := &PipelineContext{
		PipelineID:      "test-pipeline",
		PipelineName:    "implement",
		StepID:          "create-pr",
		CustomVariables: make(map[string]string),
	}

	info := forge.ForgeInfo{
		Type:           forge.ForgeBitbucket,
		Host:           "bitbucket.org",
		Owner:          "myteam",
		Repo:           "myproject",
		CLITool:        "bb",
		PipelinePrefix: "bb",
		PRTerm:         "Pull Request",
		PRCommand:      "pr",
	}

	InjectForgeVariables(ctx, info)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{"forge.type", "{{ forge.type }}", "bitbucket"},
		{"forge.host", "{{ forge.host }}", "bitbucket.org"},
		{"forge.owner", "{{ forge.owner }}", "myteam"},
		{"forge.repo", "{{ forge.repo }}", "myproject"},
		{"forge.cli_tool", "{{ forge.cli_tool }}", "bb"},
		{"forge.prefix", "{{ forge.prefix }}", "bb"},
		{"forge.pr_term", "{{ forge.pr_term }}", "Pull Request"},
		{"forge.pr_command", "{{ forge.pr_command }}", "pr"},
		{"persona resolution", "{{ forge.prefix }}-commenter", "bb-commenter"},
		{"PR creation command", "{{ forge.cli_tool }} {{ forge.pr_command }} create", "bb pr create"},
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

func TestInjectForgeVariables_Gitea(t *testing.T) {
	ctx := &PipelineContext{
		PipelineID:      "test-pipeline",
		PipelineName:    "implement",
		StepID:          "create-pr",
		CustomVariables: make(map[string]string),
	}

	info := forge.ForgeInfo{
		Type:           forge.ForgeGitea,
		Host:           "gitea.example.com",
		Owner:          "devs",
		Repo:           "app",
		CLITool:        "tea",
		PipelinePrefix: "gt",
		PRTerm:         "Pull Request",
		PRCommand:      "pr",
	}

	InjectForgeVariables(ctx, info)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{"forge.type", "{{ forge.type }}", "gitea"},
		{"forge.host", "{{ forge.host }}", "gitea.example.com"},
		{"forge.owner", "{{ forge.owner }}", "devs"},
		{"forge.repo", "{{ forge.repo }}", "app"},
		{"forge.cli_tool", "{{ forge.cli_tool }}", "tea"},
		{"forge.prefix", "{{ forge.prefix }}", "gt"},
		{"forge.pr_term", "{{ forge.pr_term }}", "Pull Request"},
		{"forge.pr_command", "{{ forge.pr_command }}", "pr"},
		{"persona resolution", "{{ forge.prefix }}-commenter", "gt-commenter"},
		{"PR creation command", "{{ forge.cli_tool }} {{ forge.pr_command }} create", "tea pr create"},
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

func TestInjectForgeVariables_Unknown(t *testing.T) {
	ctx := &PipelineContext{
		PipelineID:      "test-pipeline",
		PipelineName:    "implement",
		StepID:          "step1",
		CustomVariables: make(map[string]string),
	}

	info := forge.ForgeInfo{Type: forge.ForgeUnknown}
	InjectForgeVariables(ctx, info)

	// forge.type should be "unknown", all others empty
	for _, key := range []string{
		"forge.type", "forge.host", "forge.owner", "forge.repo",
		"forge.cli_tool", "forge.prefix", "forge.pr_term", "forge.pr_command",
	} {
		got := ctx.CustomVariables[key]
		if key == "forge.type" {
			if got != "unknown" {
				t.Errorf("CustomVariables[%q] = %q, want %q", key, got, "unknown")
			}
		} else if got != "" {
			t.Errorf("CustomVariables[%q] = %q, want empty for unknown forge", key, got)
		}
	}

	// Persona resolution with unknown forge produces "-commenter"
	result := ctx.ResolvePlaceholders("{{ forge.prefix }}-commenter")
	if result != "-commenter" {
		t.Errorf("persona resolution with unknown forge = %q, want %q", result, "-commenter")
	}
}

func TestInjectForgeVariables_EmptyForgeInfo(t *testing.T) {
	ctx := &PipelineContext{
		PipelineID:      "test-pipeline",
		PipelineName:    "implement",
		StepID:          "step1",
		CustomVariables: make(map[string]string),
	}

	InjectForgeVariables(ctx, forge.ForgeInfo{})

	// Zero-value ForgeInfo has empty Type (""), which resolves to empty string
	got := ctx.CustomVariables["forge.type"]
	if got != "" {
		t.Errorf("CustomVariables[forge.type] = %q, want empty for zero-value ForgeInfo", got)
	}
}

func TestForgeVariables_AllForgeTypes_ConsistentStructure(t *testing.T) {
	forges := []struct {
		name string
		info forge.ForgeInfo
	}{
		{"GitHub", forge.Detect("https://github.com/owner/repo.git")},
		{"GitLab", forge.Detect("https://gitlab.com/owner/repo.git")},
		{"Bitbucket", forge.Detect("https://bitbucket.org/owner/repo.git")},
		{"Gitea", forge.Detect("https://gitea.example.com/owner/repo.git")},
	}

	template := "{{ forge.cli_tool }} {{ forge.pr_command }} create --title 'feat: {{ forge.pr_term }}'"

	for _, tt := range forges {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &PipelineContext{
				PipelineID:      "test",
				PipelineName:    "test",
				StepID:          "step",
				CustomVariables: make(map[string]string),
			}
			InjectForgeVariables(ctx, tt.info)
			result := ctx.ResolvePlaceholders(template)

			if strings.Contains(result, "{{") {
				t.Errorf("unresolved placeholders in result: %q", result)
			}
			if result == "" {
				t.Error("result should not be empty for known forge type")
			}
		})
	}
}

func TestForgeVariables_MixedProjectAndForgeVars(t *testing.T) {
	m := &manifest.Manifest{
		Project: &manifest.Project{
			Language:    "go",
			TestCommand: "go test ./...",
		},
	}
	ctx := newContextWithProject("pipe-123", "implement", "step", m)
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

	// Both namespaces should resolve in the same template
	template := "{{ project.test_command }} && {{ forge.cli_tool }} {{ forge.pr_command }} create"
	result := ctx.ResolvePlaceholders(template)

	if strings.Contains(result, "{{ forge.") {
		t.Errorf("unresolved forge variables: %q", result)
	}
	if !strings.Contains(result, "gh pr create") {
		t.Errorf("expected 'gh pr create' in result: %q", result)
	}
}

func TestForgeVariables_ConcurrentAccess(t *testing.T) {
	ctx := &PipelineContext{
		PipelineID:      "concurrent-test",
		PipelineName:    "test",
		StepID:          "step",
		CustomVariables: make(map[string]string),
	}

	forges := []forge.ForgeInfo{
		forge.Detect("https://github.com/a/b.git"),
		forge.Detect("https://gitlab.com/c/d.git"),
		forge.Detect("https://bitbucket.org/e/f.git"),
		forge.Detect("https://gitea.example.com/g/h.git"),
	}

	done := make(chan struct{})

	// Writers
	for i := 0; i < 10; i++ {
		go func(i int) {
			InjectForgeVariables(ctx, forges[i%len(forges)])
			done <- struct{}{}
		}(i)
	}

	// Readers
	for i := 0; i < 10; i++ {
		go func() {
			_ = ctx.ResolvePlaceholders("{{ forge.type }}: {{ forge.cli_tool }}")
			done <- struct{}{}
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}