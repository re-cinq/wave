package contract

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestValidationErrorFormatter_SchemaErrorAnalysis tests the detailed error analysis
func TestValidationErrorFormatter_SchemaErrorAnalysis(t *testing.T) {
	tests := []struct {
		name                string
		errorMessage        string
		expectedErrorType   string
		expectedMainMessage string
		expectedSuggestions int // minimum number of suggestions expected
		expectedPitfalls    int // minimum number of pitfalls expected
		hasExample          bool
	}{
		{
			name:                "missing required fields",
			errorMessage:        "missing property 'name' at root",
			expectedErrorType:   "missing_required_fields",
			expectedMainMessage: "Required fields are missing from the JSON output",
			expectedSuggestions: 3,
			expectedPitfalls:    2,
			hasExample:          true,
		},
		{
			name:                "type mismatch - got string want number",
			errorMessage:        "invalid type: got string, want integer",
			expectedErrorType:   "type_mismatch",
			expectedMainMessage: "Field types don't match the schema requirements",
			expectedSuggestions: 4,
			expectedPitfalls:    3,
			hasExample:          true,
		},
		{
			name:                "enum violation",
			errorMessage:        "value is not one of: [\"bug\", \"feature\", \"task\"]",
			expectedErrorType:   "enum_violation",
			expectedMainMessage: "Field value is not in the allowed list of options",
			expectedSuggestions: 3,
			expectedPitfalls:    3,
			hasExample:          false,
		},
		{
			name:                "additional properties not allowed",
			errorMessage:        "additional property 'extraField' not allowed",
			expectedErrorType:   "additional_properties",
			expectedMainMessage: "Extra fields found that are not defined in the schema",
			expectedSuggestions: 3,
			expectedPitfalls:    2,
			hasExample:          false,
		},
		{
			name:                "array validation failure",
			errorMessage:        "invalid array: minimum length is 1",
			expectedErrorType:   "array_issues",
			expectedMainMessage: "Array field validation failed",
			expectedSuggestions: 3,
			expectedPitfalls:    2,
			hasExample:          true,
		},
		{
			name:                "string format violation",
			errorMessage:        "format validation failed: not a valid email",
			expectedErrorType:   "format_violation",
			expectedMainMessage: "String format validation failed",
			expectedSuggestions: 4,
			expectedPitfalls:    3,
			hasExample:          false,
		},
		{
			name:                "unknown error pattern - generic guidance",
			errorMessage:        "some unknown validation error occurred",
			expectedErrorType:   "",
			expectedMainMessage: "JSON schema validation failed",
			expectedSuggestions: 4, // generic suggestions
			expectedPitfalls:    3, // generic pitfalls
			hasExample:          false,
		},
	}

	formatter := &ValidationErrorFormatter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := formatter.analyzeSchemaError(tt.errorMessage, nil)

			// Check error type classification
			if analysis.ErrorType != tt.expectedErrorType {
				t.Errorf("Expected error type %q, got %q", tt.expectedErrorType, analysis.ErrorType)
			}

			// Check main message
			if !strings.Contains(analysis.MainMessage, tt.expectedMainMessage) {
				t.Errorf("Expected main message to contain %q, got %q", tt.expectedMainMessage, analysis.MainMessage)
			}

			// Check suggestions count
			if len(analysis.Suggestions) < tt.expectedSuggestions {
				t.Errorf("Expected at least %d suggestions, got %d: %v", tt.expectedSuggestions, len(analysis.Suggestions), analysis.Suggestions)
			}

			// Check pitfalls count
			if len(analysis.CommonPitfalls) < tt.expectedPitfalls {
				t.Errorf("Expected at least %d pitfalls, got %d: %v", tt.expectedPitfalls, len(analysis.CommonPitfalls), analysis.CommonPitfalls)
			}

			// Check example presence
			hasExample := analysis.Example != ""
			if hasExample != tt.hasExample {
				t.Errorf("Expected example present: %v, got: %v", tt.hasExample, hasExample)
			}

			// Check that suggestions are actionable (not empty)
			for i, suggestion := range analysis.Suggestions {
				if strings.TrimSpace(suggestion) == "" {
					t.Errorf("Suggestion %d is empty", i)
				}
			}

			// Check that all suggestions are unique
			suggestionSet := make(map[string]bool)
			for _, suggestion := range analysis.Suggestions {
				if suggestionSet[suggestion] {
					t.Errorf("Duplicate suggestion found: %q", suggestion)
				}
				suggestionSet[suggestion] = true
			}

			t.Logf("Analysis for %q:\n  Type: %s\n  Main: %s\n  Suggestions: %d\n  Pitfalls: %d\n  Example: %t",
				tt.errorMessage, analysis.ErrorType, analysis.MainMessage,
				len(analysis.Suggestions), len(analysis.CommonPitfalls), hasExample)
		})
	}
}

// TestValidationErrorFormatter_WithRecoveryContext tests error formatting with recovery information
func TestValidationErrorFormatter_WithRecoveryContext(t *testing.T) {
	tests := []struct {
		name           string
		errorMessage   string
		recoveryResult *RecoveryResult
		expectedInDetails []string
		description    string
	}{
		{
			name:         "with successful recovery",
			errorMessage: "missing property 'name'",
			recoveryResult: &RecoveryResult{
				AppliedFixes: []string{"removed_trailing_commas", "quoted_unquoted_keys"},
				Warnings:     []string{"aggressive_reconstruction_applied"},
				IsValid:      false, // JSON was fixed but schema validation still failed
			},
			expectedInDetails: []string{
				"JSON Recovery Applied: removed_trailing_commas, quoted_unquoted_keys",
				"Recovery Warnings:",
				"aggressive_reconstruction_applied",
			},
			description: "Should include recovery information in error details",
		},
		{
			name:         "with no recovery needed",
			errorMessage: "invalid type: got string, want integer",
			recoveryResult: &RecoveryResult{
				AppliedFixes: []string{},
				Warnings:     []string{},
				IsValid:      true,
			},
			expectedInDetails: []string{
				"Schema Validation Errors:",
			},
			description: "Should not mention recovery when none was applied",
		},
		{
			name:           "with nil recovery result",
			errorMessage:   "missing property 'count'",
			recoveryResult: nil,
			expectedInDetails: []string{
				"Schema Validation Errors:",
				"Suggested Fixes:",
			},
			description: "Should handle nil recovery result gracefully",
		},
	}

	formatter := &ValidationErrorFormatter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalErr := fmt.Errorf("%s", tt.errorMessage)
			artifactPath := "/test/path/artifact.json"

			formatted := formatter.FormatJSONSchemaError(originalErr, tt.recoveryResult, artifactPath)

			// Check that formatted error is properly structured
			if formatted.ContractType != "json_schema" {
				t.Errorf("Expected contract type 'json_schema', got %q", formatted.ContractType)
			}

			if len(formatted.Details) == 0 {
				t.Error("Expected error details, got none")
			}

			// Check for expected content in details
			detailsText := strings.Join(formatted.Details, "\n")
			for _, expected := range tt.expectedInDetails {
				if !strings.Contains(detailsText, expected) {
					t.Errorf("Expected to find %q in details, but didn't. Details:\n%s", expected, detailsText)
				}
			}

			// Validate error message quality
			if formatted.Message == "" {
				t.Error("Expected non-empty error message")
			}

			// Check that the full error string is well-formatted
			fullError := formatted.Error()
			if !strings.Contains(fullError, "[json_schema]") {
				t.Error("Expected contract type in error string")
			}

			t.Logf("Formatted error:\n%s", fullError)
		})
	}
}

// TestValidationErrorFormatter_ProgressiveValidationWarnings tests progressive validation warning format
func TestValidationErrorFormatter_ProgressiveValidationWarnings(t *testing.T) {
	tests := []struct {
		name           string
		errorMessage   string
		recoveryResult *RecoveryResult
		expectedWarnings int
		description    string
	}{
		{
			name:         "schema error with recovery",
			errorMessage: "missing property 'name'",
			recoveryResult: &RecoveryResult{
				AppliedFixes: []string{"removed_trailing_commas"},
				Warnings:     []string{},
			},
			expectedWarnings: 3, // recovery + schema issue + suggestions
			description:      "Should generate warnings for both recovery and schema issues",
		},
		{
			name:         "schema error without recovery",
			errorMessage: "invalid type: got string, want integer",
			recoveryResult: &RecoveryResult{
				AppliedFixes: []string{},
				Warnings:     []string{},
			},
			expectedWarnings: 4, // schema issue + top 3 suggestions
			description:      "Should generate warnings for schema issues without recovery info",
		},
	}

	formatter := &ValidationErrorFormatter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalErr := fmt.Errorf("%s", tt.errorMessage)

			warnings := formatter.FormatProgressiveValidationWarning(originalErr, tt.recoveryResult)

			if len(warnings) < tt.expectedWarnings {
				t.Errorf("Expected at least %d warnings, got %d: %v", tt.expectedWarnings, len(warnings), warnings)
			}

			// Check warning content quality
			for i, warning := range warnings {
				if strings.TrimSpace(warning) == "" {
					t.Errorf("Warning %d is empty", i)
				}
				if !strings.HasPrefix(warning, "JSON automatically corrected:") &&
				   !strings.HasPrefix(warning, "Schema validation issue:") &&
				   !strings.HasPrefix(warning, "Suggestion:") {
					t.Errorf("Warning %d has unexpected format: %q", i, warning)
				}
			}

			t.Logf("Progressive validation warnings for %q:\n%s", tt.errorMessage, strings.Join(warnings, "\n"))
		})
	}
}

// TestValidationErrorFormatter_ExtractFieldPath tests field path extraction from error messages
func TestValidationErrorFormatter_ExtractFieldPath(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		expectedPath string
		description  string
	}{
		{
			name:         "path with 'at' keyword",
			errorMessage: "validation failed at '/user/name'",
			expectedPath: "/user/name",
			description:  "Should extract path after 'at' keyword",
		},
		{
			name:         "path with 'field' keyword",
			errorMessage: "invalid field 'email' found",
			expectedPath: "email",
			description:  "Should extract field name from 'field' context",
		},
		{
			name:         "quoted property name",
			errorMessage: "property 'firstName' is required",
			expectedPath: "firstName",
			description:  "Should extract first quoted property name",
		},
		{
			name:         "no recognizable path",
			errorMessage: "general validation error occurred",
			expectedPath: "",
			description:  "Should return empty string when no path is found",
		},
		{
			name:         "nested path",
			errorMessage: "validation error at '/data/user/profile/email'",
			expectedPath: "/data/user/profile/email",
			description:  "Should handle nested JSON paths",
		},
	}

	formatter := &ValidationErrorFormatter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractedPath := formatter.ExtractFieldPath(tt.errorMessage)

			if extractedPath != tt.expectedPath {
				t.Errorf("Expected path %q, got %q", tt.expectedPath, extractedPath)
			}

			t.Logf("Error message: %q → Path: %q", tt.errorMessage, extractedPath)
		})
	}
}

// TestValidationErrorFormatter_RealWorldScenarios tests with realistic AI-generated content
func TestValidationErrorFormatter_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name               string
		schema             string
		malformedJSON      string
		expectedRecovery   bool
		expectedValidation bool
		description        string
	}{
		{
			name: "GitHub issue analysis with comments and trailing commas",
			schema: `{
				"type": "object",
				"properties": {
					"repository": {
						"type": "object",
						"properties": {
							"owner": {"type": "string"},
							"name": {"type": "string"}
						},
						"required": ["owner", "name"]
					},
					"total_issues": {"type": "integer"},
					"poor_quality_issues": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"number": {"type": "integer"},
								"title": {"type": "string"},
								"quality_score": {"type": "integer"}
							},
							"required": ["number", "title", "quality_score"]
						}
					}
				},
				"required": ["repository", "total_issues", "poor_quality_issues"]
			}`,
			malformedJSON: `{
				"repository": {
					"owner": "re-cinq",
					"name": "wave"
				},
				// Total issues analyzed
				"total_issues": 10,
				"poor_quality_issues": [
					{
						"number": 20,
						"title": "Poorly written issue",
						"quality_score": 45,
					}, // trailing comma issue
				],
				// end of data
			}`,
			expectedRecovery:   true,
			expectedValidation: true,
			description:        "Should recover from comments and trailing commas in realistic JSON",
		},
		{
			name: "Code implementation results with unquoted keys",
			schema: `{
				"type": "object",
				"properties": {
					"files_changed": {"type": "array", "items": {"type": "string"}},
					"tests_passed": {"type": "boolean"},
					"implementation_notes": {"type": "string"}
				},
				"required": ["files_changed", "tests_passed", "implementation_notes"]
			}`,
			malformedJSON: `{
				files_changed: ["src/main.go", "internal/service.go"],
				tests_passed: true,
				implementation_notes: "Successfully implemented the new feature",
			}`,
			expectedRecovery:   true,
			expectedValidation: true,
			description:        "Should recover from unquoted keys and trailing commas",
		},
		{
			name: "Markdown-wrapped JSON output",
			schema: `{
				"type": "object",
				"properties": {
					"analysis": {"type": "string"},
					"score": {"type": "integer"}
				},
				"required": ["analysis", "score"]
			}`,
			malformedJSON: `Based on my analysis:

` + "```json" + `
{
	"analysis": "The code quality is good overall",
	"score": 85
}
` + "```" + `

This completes the evaluation.`,
			expectedRecovery:   true,
			expectedValidation: true,
			description:        "Should extract JSON from markdown code blocks",
		},
		{
			name: "Unfixable malformed JSON",
			schema: `{
				"type": "object",
				"properties": {
					"result": {"type": "string"}
				},
				"required": ["result"]
			}`,
			malformedJSON: `{this is completely broken JSON that cannot be recovered}`,
			expectedRecovery:   false,
			expectedValidation: false,
			description:        "Should fail gracefully with completely invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON recovery first
			parser := NewJSONRecoveryParser(ProgressiveRecovery)
			recoveryResult, recoveryErr := parser.ParseWithRecovery(tt.malformedJSON)

			if tt.expectedRecovery {
				if recoveryErr != nil || !recoveryResult.IsValid {
					t.Errorf("Expected successful recovery, got error: %v, valid: %v", recoveryErr, recoveryResult.IsValid)
				}
				if len(recoveryResult.AppliedFixes) == 0 {
					t.Error("Expected recovery fixes to be applied")
				}
			} else {
				if recoveryErr == nil && recoveryResult.IsValid {
					t.Error("Expected recovery to fail, but it succeeded")
				}
			}

			t.Logf("Recovery result:\n%s", recoveryResult.FormatRecoveryReport())

			// If recovery succeeded, test full validation
			if tt.expectedRecovery && recoveryResult.IsValid {
				// Create a temporary workspace for validation testing
				workspacePath := t.TempDir()
				artifactPath := workspacePath + "/artifact.json"

				// Write the recovered JSON
				if err := writeFile(artifactPath, []byte(recoveryResult.RecoveredJSON)); err != nil {
					t.Fatalf("Failed to write recovered JSON: %v", err)
				}

				// Test schema validation
				validator := &jsonSchemaValidator{}
				config := ContractConfig{
					Type:                  "json_schema",
					Schema:                tt.schema,
					AllowRecovery:         true,
					RecoveryLevel:         "progressive",
					ProgressiveValidation: false,
					MustPass:              true,
				}

				validationErr := validator.Validate(config, workspacePath)

				if tt.expectedValidation {
					if validationErr != nil {
						t.Errorf("Expected validation to succeed, got error: %v", validationErr)
					}
				} else {
					if validationErr == nil {
						t.Error("Expected validation to fail, but it succeeded")
					}
				}

				if validationErr != nil {
					t.Logf("Validation error: %v", validationErr)
				}
			}
		})
	}
}

// Helper function to write file (mimics os.WriteFile for testing)
func writeFile(filename string, data []byte) error {
	// This would typically use os.WriteFile, but for testing we'll implement a simple version
	// In the actual implementation, this should just be os.WriteFile
	file, err := os.Create(filename)
	_ = file
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}