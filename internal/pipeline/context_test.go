package pipeline

import (
	"testing"
)

func TestPipelineContext_ResolvePlaceholders(t *testing.T) {
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
				Path: "output/analysis.json",
				Type: "json",
			},
			expected: "output/analysis.json",
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