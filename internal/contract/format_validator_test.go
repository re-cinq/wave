package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFormatValidator_ValidateGitHubIssueFormat(t *testing.T) {
	tests := []struct {
		name      string
		output    map[string]interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid issue format",
			output: map[string]interface{}{
				"title": "feat: Add user authentication",
				"body": `## Description
This feature adds OAuth2 authentication to the application.

## Acceptance Criteria
- Users can log in with OAuth2
- Tokens are securely stored
- Session management works correctly`,
				"labels": []interface{}{"enhancement", "priority-high"},
			},
			wantError: false,
		},
		{
			name: "title too short",
			output: map[string]interface{}{
				"title": "Add auth",
				"body":  "Some body content that meets the minimum length requirement for validation",
				"labels": []interface{}{"enhancement"},
			},
			wantError: true,
			errorMsg:  "title too short",
		},
		{
			name: "title with placeholder",
			output: map[string]interface{}{
				"title": "[TODO] Implement feature",
				"body":  "Some body content that meets the minimum length requirement for validation",
				"labels": []interface{}{"enhancement"},
			},
			wantError: true,
			errorMsg:  "placeholder text",
		},
		{
			name: "body too short",
			output: map[string]interface{}{
				"title":  "feat: Add user authentication feature",
				"body":   "Short body",
				"labels": []interface{}{"enhancement"},
			},
			wantError: true,
			errorMsg:  "body too short",
		},
		{
			name: "missing required sections",
			output: map[string]interface{}{
				"title": "feat: Add user authentication feature",
				"body": "This is a body without the required sections but long enough to pass length check.",
				"labels": []interface{}{"enhancement"},
			},
			wantError: true,
			errorMsg:  "missing",
		},
		{
			name: "no labels",
			output: map[string]interface{}{
				"title": "feat: Add user authentication feature",
				"body": `## Description
Complete description with all required sections.

## Acceptance Criteria
- Criterion 1
- Criterion 2`,
				"labels": []interface{}{},
			},
			wantError: true,
			errorMsg:  "no labels",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &FormatValidator{}
			err := v.validateGitHubIssueFormat(tt.output)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
				} else if validationErr, ok := err.(*ValidationError); ok {
					found := false
					errorStr := validationErr.Error()
					for _, detail := range validationErr.Details {
						if contains(detail, tt.errorMsg) || contains(errorStr, tt.errorMsg) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("error %q does not contain expected message %q", errorStr, tt.errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestFormatValidator_ValidateGitHubPRFormat(t *testing.T) {
	tests := []struct {
		name      string
		output    map[string]interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid PR format",
			output: map[string]interface{}{
				"title": "feat: Add OAuth2 authentication support",
				"body": `## Summary
This PR adds OAuth2 authentication to the application.

## Changes
- Added OAuth2 client configuration
- Implemented token refresh logic
- Added session management

## Testing
- Unit tests for OAuth2 flow
- Integration tests for token refresh
- Manual testing with Google OAuth

## Related Issues
Closes #123

## Checklist
- [x] Tests added
- [x] Tests passing
- [x] Documentation updated`,
				"head": "feature/oauth2",
				"base": "main",
			},
			wantError: false,
		},
		{
			name: "title too long",
			output: map[string]interface{}{
				"title": "feat: This is a very long pull request title that exceeds the seventy-two character limit recommended for optimal display",
				"body":  "Body content that meets minimum requirements for validation",
				"head":  "feature/test",
				"base":  "main",
			},
			wantError: true,
			errorMsg:  "too long",
		},
		{
			name: "PR from main branch",
			output: map[string]interface{}{
				"title": "feat: Add feature",
				"body":  "Body content that meets minimum requirements for validation with proper sections",
				"head":  "main",
				"base":  "develop",
			},
			wantError: true,
			errorMsg:  "main/master",
		},
		{
			name: "body too short",
			output: map[string]interface{}{
				"title": "feat: Add feature",
				"body":  "Short",
				"head":  "feature/test",
				"base":  "main",
			},
			wantError: true,
			errorMsg:  "too short",
		},
		{
			name: "missing issue reference",
			output: map[string]interface{}{
				"title": "feat: Add feature",
				"body": `## Summary
This PR adds a feature.

## Changes
- Change 1

## Testing
- Test 1`,
				"head": "feature/test",
				"base": "main",
			},
			wantError: true,
			errorMsg:  "reference related issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &FormatValidator{}
			err := v.validateGitHubPRFormat(tt.output)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
				} else {
					errorStr := err.Error()
					if !contains(errorStr, tt.errorMsg) {
						t.Errorf("error %q does not contain expected message %q", errorStr, tt.errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestFormatValidator_ValidateImplementationResults(t *testing.T) {
	tests := []struct {
		name      string
		output    map[string]interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid implementation",
			output: map[string]interface{}{
				"files_changed":        []interface{}{"file1.go", "file2.go"},
				"tests_passed":         true,
				"builds_successfully":  true,
				"implementation_notes": "Implemented OAuth2 authentication with token refresh",
			},
			wantError: false,
		},
		{
			name: "no files changed",
			output: map[string]interface{}{
				"files_changed":        []interface{}{},
				"tests_passed":         true,
				"builds_successfully":  true,
				"implementation_notes": "Implementation notes here",
			},
			wantError: true,
			errorMsg:  "no files changed",
		},
		{
			name: "tests failing",
			output: map[string]interface{}{
				"files_changed":        []interface{}{"file1.go"},
				"tests_passed":         false,
				"builds_successfully":  true,
				"implementation_notes": "Implementation notes here",
			},
			wantError: true,
			errorMsg:  "tests are failing",
		},
		{
			name: "build failing",
			output: map[string]interface{}{
				"files_changed":        []interface{}{"file1.go"},
				"tests_passed":         true,
				"builds_successfully":  false,
				"implementation_notes": "Implementation notes here",
			},
			wantError: true,
			errorMsg:  "does not build",
		},
		{
			name: "notes too brief",
			output: map[string]interface{}{
				"files_changed":        []interface{}{"file1.go"},
				"tests_passed":         true,
				"builds_successfully":  true,
				"implementation_notes": "Done",
			},
			wantError: true,
			errorMsg:  "too brief",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &FormatValidator{}
			err := v.validateImplementationResults(tt.output)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
				} else {
					errorStr := err.Error()
					if !contains(errorStr, tt.errorMsg) {
						t.Errorf("error %q does not contain expected message %q", errorStr, tt.errorMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestFormatValidator_Validate(t *testing.T) {
	// Create temporary workspace
	tmpDir, err := os.MkdirTemp("", "format-validator-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test with GitHub issue format
	issueData := map[string]interface{}{
		"title": "feat: Add comprehensive user authentication system",
		"body": `## Description
This feature adds OAuth2 authentication to the application with support for multiple providers.

## Acceptance Criteria
- Users can log in with OAuth2
- Tokens are securely stored
- Session management works correctly`,
		"labels": []interface{}{"enhancement", "priority-high"},
	}

	issueJSON, _ := json.Marshal(issueData)
	outputPath := filepath.Join(tmpDir, "output.json")
	if err := os.WriteFile(outputPath, issueJSON, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	v := &FormatValidator{}
	cfg := ContractConfig{
		Source:     "output.json",
		SchemaPath: ".wave/contracts/github-issue-analysis.schema.json",
	}

	err = v.Validate(cfg, tmpDir)
	if err != nil {
		t.Errorf("validation failed for valid issue: %v", err)
	}
}

func TestInferFormatType(t *testing.T) {
	tests := []struct {
		schemaPath string
		expected   string
	}{
		{".wave/contracts/github-issue-analysis.schema.json", "github_issue"},
		{".wave/contracts/github-pr-draft.schema.json", "github_pr"},
		{".wave/contracts/github-pr-info.schema.json", "github_pr"},
		{".wave/contracts/implementation-results.schema.json", "implementation_results"},
		{".wave/contracts/analysis.schema.json", "analysis"},
		{".wave/contracts/unknown.schema.json", "generic"},
		{"", "generic"},
	}

	for _, tt := range tests {
		t.Run(tt.schemaPath, func(t *testing.T) {
			cfg := ContractConfig{SchemaPath: tt.schemaPath}
			result := inferFormatType(cfg)
			if result != tt.expected {
				t.Errorf("inferFormatType(%q) = %q, want %q", tt.schemaPath, result, tt.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
