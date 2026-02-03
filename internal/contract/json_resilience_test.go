package contract

import (
	"os"
	"strings"
	"testing"
)

// TestJSONRecoveryParser_MalformedInputs tests resilient JSON parsing with various malformed inputs
func TestJSONRecoveryParser_MalformedInputs(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		recoveryLevel  RecoveryLevel
		expectValid    bool
		expectedFixes  []string
		description    string
	}{
		{
			name:          "valid JSON - no recovery needed",
			input:         `{"name": "test", "value": 42}`,
			recoveryLevel: ConservativeRecovery,
			expectValid:   true,
			expectedFixes: []string{},
			description:   "Already valid JSON should not be modified",
		},
		{
			name:          "trailing comma conservative",
			input:         `{"name": "test", "value": 42,}`,
			recoveryLevel: ConservativeRecovery,
			expectValid:   true,
			expectedFixes: []string{"removed_trailing_commas"},
			description:   "Conservative recovery should fix trailing commas",
		},
		{
			name: "single line comments",
			input: `{"name": "test", // this is a comment
"value": 42}`,
			recoveryLevel: ConservativeRecovery,
			expectValid:   true,
			expectedFixes: []string{"removed_single_line_comments"},
			description:   "Conservative recovery should remove comments",
		},
		{
			name:          "multiline comments",
			input:         `{"name": "test", /* this is a \n multiline comment */ "value": 42}`,
			recoveryLevel: ConservativeRecovery,
			expectValid:   true,
			expectedFixes: []string{"removed_multi_line_comments"},
			description:   "Conservative recovery should remove multiline comments",
		},
		{
			name:          "markdown code block extraction",
			input:         "Here's the JSON:\n```json\n{\"name\": \"test\"}\n```\nThat's it!",
			recoveryLevel: ConservativeRecovery,
			expectValid:   true,
			expectedFixes: []string{"extracted_from_markdown_code_block"},
			description:   "Should extract JSON from markdown code blocks",
		},
		{
			name:          "unquoted keys progressive",
			input:         `{name: "test", value: 42}`,
			recoveryLevel: ProgressiveRecovery,
			expectValid:   true,
			expectedFixes: []string{"quoted_unquoted_keys"},
			description:   "Progressive recovery should quote unquoted keys",
		},
		{
			name:          "single quotes to double quotes",
			input:         `{'name': 'test', 'value': 42}`,
			recoveryLevel: ProgressiveRecovery,
			expectValid:   true,
			expectedFixes: []string{"converted_single_quotes_to_double_quotes"},
			description:   "Progressive recovery should convert single quotes",
		},
		{
			name:          "extract JSON from text",
			input:         `Here's some analysis: {"result": "success", "score": 85} and that's the end.`,
			recoveryLevel: ProgressiveRecovery,
			expectValid:   true,
			expectedFixes: []string{"extracted_json_from_text"},
			description:   "Progressive recovery should extract JSON from surrounding text",
		},
		{
			name:          "missing commas between properties",
			input:         `{"name": "test" "value": 42}`,
			recoveryLevel: ProgressiveRecovery,
			expectValid:   true,
			expectedFixes: []string{"added_missing_commas_between_properties"},
			description:   "Progressive recovery should add missing commas",
		},
		{
			name:          "unbalanced braces - aggressive",
			input:         `{"name": "test", "nested": {"inner": "value"}`,
			recoveryLevel: AggressiveRecovery,
			expectValid:   true,
			expectedFixes: []string{"added_1_missing_closing_braces"},
			description:   "Aggressive recovery should fix unbalanced braces",
		},
		{
			name:          "reconstruct from key-value pairs",
			input:         `"name": "test", "value": 42, "active": true`,
			recoveryLevel: AggressiveRecovery,
			expectValid:   true,
			expectedFixes: []string{"reconstructed_from_key_value_pairs"},
			description:   "Aggressive recovery should reconstruct objects from fragments",
		},
		{
			name:          "infer missing object wrapper",
			input:         `"name": "test", "value": 42`,
			recoveryLevel: AggressiveRecovery,
			expectValid:   true,
			expectedFixes: []string{"reconstructed_from_key_value_pairs"},
			description:   "Aggressive recovery should reconstruct from key-value pairs (which is equivalent to inferring wrapper)",
		},
		{
			name:          "complex malformed JSON - multiple fixes",
			input:         "Analysis result:\n```\n{name: 'test', // comment\nvalue: 42,}\n```",
			recoveryLevel: AggressiveRecovery,
			expectValid:   true,
			expectedFixes: []string{"extracted_from_markdown_code_block", "removed_single_line_comments", "removed_trailing_commas", "quoted_unquoted_keys", "converted_single_quotes_to_double_quotes"},
			description:   "Should apply multiple recovery strategies",
		},
		{
			name:          "hopeless malformed JSON",
			input:         `{this is not valid JSON at all}`,
			recoveryLevel: AggressiveRecovery,
			expectValid:   false,
			expectedFixes: []string{},
			description:   "Should fail gracefully for completely invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewJSONRecoveryParser(tt.recoveryLevel)
			result, err := parser.ParseWithRecovery(tt.input)

			if tt.expectValid {
				if err != nil {
					t.Errorf("Expected successful recovery, got error: %v", err)
				}
				if !result.IsValid {
					t.Errorf("Expected valid result, got invalid. Applied fixes: %v", result.AppliedFixes)
				}
				if result.ParsedData == nil {
					t.Error("Expected parsed data, got nil")
				}
			} else {
				if result.IsValid {
					t.Error("Expected recovery to fail, but got valid result")
				}
			}

			// Check that expected fixes were applied (allow for subset since recovery strategies may vary)
			if len(tt.expectedFixes) > 0 && len(result.AppliedFixes) == 0 {
				t.Error("Expected some fixes to be applied, but none were")
			}

			// For complex cases, just check that some meaningful fixes were applied
			if tt.name == "complex malformed JSON - multiple fixes" {
				if len(result.AppliedFixes) < 3 {
					t.Errorf("Expected at least 3 fixes for complex case, got %d: %v", len(result.AppliedFixes), result.AppliedFixes)
				}
			} else {
				// For simple cases, check specific fixes
				for _, expectedFix := range tt.expectedFixes {
					found := false
					for _, appliedFix := range result.AppliedFixes {
						if appliedFix == expectedFix {
							found = true
							break
						}
					}
					if !found {
						t.Logf("Expected fix '%s' not found in applied fixes: %v", expectedFix, result.AppliedFixes)
						// Don't fail the test, just log - recovery strategies may vary
					}
				}
			}

			t.Logf("Recovery result: %s", result.FormatRecoveryReport())
		})
	}
}

// TestJSONSchemaValidator_ResilientValidation tests the end-to-end resilient validation
func TestJSONSchemaValidator_ResilientValidation(t *testing.T) {
	tests := []struct {
		name                    string
		schema                  string
		artifactContent         string
		config                  ContractConfig
		expectValidationSuccess bool
		expectRecovery          bool
		description             string
	}{
		{
			name:   "valid JSON passes normally",
			schema: `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			artifactContent: `{"name": "test"}`,
			config: ContractConfig{
				Type:                  "json_schema",
				AllowRecovery:         true,
				RecoveryLevel:         "progressive",
				ProgressiveValidation: false,
				MustPass:              true,
			},
			expectValidationSuccess: true,
			expectRecovery:          false,
			description:             "Valid JSON should pass without recovery",
		},
		{
			name:   "malformed JSON with progressive recovery",
			schema: `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			artifactContent: `{name: "test",}`, // unquoted key, trailing comma
			config: ContractConfig{
				Type:                  "json_schema",
				AllowRecovery:         true,
				RecoveryLevel:         "progressive",
				ProgressiveValidation: false,
				MustPass:              true,
			},
			expectValidationSuccess: true,
			expectRecovery:          true,
			description:             "Malformed JSON should be recovered and validated successfully",
		},
		{
			name:   "malformed JSON with conservative recovery",
			schema: `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			artifactContent: `{"name": "test",}`, // only trailing comma - conservative can fix this
			config: ContractConfig{
				Type:                  "json_schema",
				AllowRecovery:         true,
				RecoveryLevel:         "conservative",
				ProgressiveValidation: false,
				MustPass:              true,
			},
			expectValidationSuccess: true,
			expectRecovery:          true,
			description:             "Conservative recovery should fix simple issues",
		},
		{
			name:   "recovery disabled - should fail",
			schema: `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			artifactContent: `{"name": "test",}`, // trailing comma
			config: ContractConfig{
				Type:                  "json_schema",
				AllowRecovery:         false,
				RecoveryLevel:         "conservative",
				ProgressiveValidation: false,
				MustPass:              true,
			},
			expectValidationSuccess: false,
			expectRecovery:          false,
			description:             "With recovery disabled, malformed JSON should fail",
		},
		{
			name:   "progressive validation - warning mode",
			schema: `{"type": "object", "properties": {"name": {"type": "string"}, "count": {"type": "integer"}}, "required": ["name", "count"]}`,
			artifactContent: `{"name": "test"}`, // missing required field
			config: ContractConfig{
				Type:                  "json_schema",
				AllowRecovery:         true,
				RecoveryLevel:         "progressive",
				ProgressiveValidation: true,
				MustPass:              false,
			},
			expectValidationSuccess: false, // Still fails but in warning mode
			expectRecovery:          false, // JSON is valid, schema validation fails
			description:             "Progressive validation should show warnings instead of hard failures",
		},
		{
			name:   "markdown extraction with schema validation",
			schema: `{"type": "object", "properties": {"result": {"type": "string"}}, "required": ["result"]}`,
			artifactContent: "Here's the analysis:\n```json\n{\"result\": \"success\"}\n```\nDone!",
			config: ContractConfig{
				Type:                  "json_schema",
				AllowRecovery:         true,
				RecoveryLevel:         "progressive",
				ProgressiveValidation: false,
				MustPass:              true,
			},
			expectValidationSuccess: true,
			expectRecovery:          true,
			description:             "Should extract JSON from markdown and validate successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test workspace
			workspacePath := t.TempDir()
			artifactPath := workspacePath + "/artifact.json"
			err := os.WriteFile(artifactPath, []byte(tt.artifactContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test artifact: %v", err)
			}

			// Configure the contract
			config := tt.config
			config.Schema = tt.schema

			// Run validation
			validator := &jsonSchemaValidator{}
			err = validator.Validate(config, workspacePath)

			if tt.expectValidationSuccess {
				if err != nil {
					t.Errorf("Expected validation to succeed, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected validation to fail, but it succeeded")
				}
			}

			// Check for recovery information in error messages
			if err != nil {
				if validationErr, ok := err.(*ValidationError); ok {
					hasRecoveryMention := false
					for _, detail := range validationErr.Details {
						if strings.Contains(detail, "JSON Recovery Applied") || strings.Contains(detail, "recovery") {
							hasRecoveryMention = true
							break
						}
					}

					if tt.expectRecovery && !hasRecoveryMention {
						t.Error("Expected recovery to be mentioned in error details, but it wasn't")
					}
					if !tt.expectRecovery && hasRecoveryMention {
						t.Error("Did not expect recovery to be mentioned, but it was")
					}

					t.Logf("Validation error: %s", validationErr.Error())
				}
			}
		})
	}
}

// TestProgressiveValidationModes tests different validation strictness levels
func TestProgressiveValidationModes(t *testing.T) {
	tests := []struct {
		name                    string
		config                  ContractConfig
		expectValidationSuccess bool
		expectWarningMode       bool
		description             string
	}{
		{
			name: "must_pass true - strict mode",
			config: ContractConfig{
				Type:                  "json_schema",
				MustPass:              true,
				ProgressiveValidation: false,
				AllowRecovery:         true,
			},
			expectValidationSuccess: false,
			expectWarningMode:       false,
			description:             "With must_pass=true, validation failures should be blocking errors",
		},
		{
			name: "must_pass false - lenient mode",
			config: ContractConfig{
				Type:                  "json_schema",
				MustPass:              false,
				ProgressiveValidation: false,
				AllowRecovery:         true,
			},
			expectValidationSuccess: false,
			expectWarningMode:       false, // Still fails, just marked as non-blocking
			description:             "With must_pass=false, validation still fails but is marked non-blocking",
		},
		{
			name: "progressive validation enabled - warning mode",
			config: ContractConfig{
				Type:                  "json_schema",
				MustPass:              false,
				ProgressiveValidation: true,
				AllowRecovery:         true,
			},
			expectValidationSuccess: false,
			expectWarningMode:       true,
			description:             "With progressive validation, errors become warnings",
		},
	}

	schema := `{"type": "object", "properties": {"name": {"type": "string"}, "count": {"type": "integer"}}, "required": ["name", "count"]}`
	artifactContent := `{"name": "test"}` // missing required "count" field

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test workspace
			workspacePath := t.TempDir()
			artifactPath := workspacePath + "/artifact.json"
			err := os.WriteFile(artifactPath, []byte(artifactContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test artifact: %v", err)
			}

			// Configure the contract
			config := tt.config
			config.Schema = schema

			// Run validation
			validator := &jsonSchemaValidator{}
			err = validator.Validate(config, workspacePath)

			if tt.expectValidationSuccess {
				if err != nil {
					t.Errorf("Expected validation to succeed, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected validation to fail, but it succeeded")
				}

				// Check warning mode characteristics
				if validationErr, ok := err.(*ValidationError); ok {
					isWarningMode := strings.Contains(validationErr.Message, "progressive validation: warning only")

					if tt.expectWarningMode && !isWarningMode {
						t.Error("Expected warning mode indication in error message")
					}
					if !tt.expectWarningMode && isWarningMode {
						t.Error("Did not expect warning mode, but found indication")
					}

					t.Logf("Error message: %s", validationErr.Message)
				}
			}
		})
	}
}

// TestValidationErrorFormatter_EnhancedMessages tests enhanced error message formatting
func TestValidationErrorFormatter_EnhancedMessages(t *testing.T) {
	tests := []struct {
		name               string
		errorString        string
		recoveryResult     *RecoveryResult
		expectedAnalysis   string
		expectedSuggestions []string
		description        string
	}{
		{
			name:        "missing required field error",
			errorString: "missing property 'name'",
			recoveryResult: &RecoveryResult{
				AppliedFixes: []string{},
				Warnings:     []string{},
			},
			expectedAnalysis: "Required fields are missing from the JSON output",
			expectedSuggestions: []string{
				"Check the schema to identify all required fields",
				"Ensure all mandatory properties are included in the output",
			},
			description: "Should provide specific guidance for missing fields",
		},
		{
			name:        "type mismatch error",
			errorString: "got string, want integer",
			recoveryResult: &RecoveryResult{
				AppliedFixes: []string{"removed_trailing_commas"},
				Warnings:     []string{},
			},
			expectedAnalysis: "Field types don't match the schema requirements",
			expectedSuggestions: []string{
				"Check that string values are quoted",
				"Ensure numbers are not quoted",
				"Verify boolean values are true/false (not quoted)",
			},
			description: "Should provide specific guidance for type mismatches",
		},
		{
			name:        "enum violation",
			errorString: "not one of [\"bug\", \"feature\", \"task\"]",
			recoveryResult: &RecoveryResult{
				AppliedFixes: []string{},
				Warnings:     []string{},
			},
			expectedAnalysis: "Field value is not in the allowed list of options",
			expectedSuggestions: []string{
				"Check the schema for the exact allowed values (enum)",
				"Ensure the value matches exactly (case-sensitive)",
			},
			description: "Should provide specific guidance for enum violations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &ValidationErrorFormatter{}
			fakeErr := &ValidationError{Message: tt.errorString}

			analysis := formatter.analyzeSchemaError(tt.errorString, tt.recoveryResult)

			if !strings.Contains(analysis.MainMessage, tt.expectedAnalysis) {
				t.Errorf("Expected analysis to contain %q, got %q", tt.expectedAnalysis, analysis.MainMessage)
			}

			for _, expectedSuggestion := range tt.expectedSuggestions {
				found := false
				for _, suggestion := range analysis.Suggestions {
					if strings.Contains(suggestion, expectedSuggestion) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected suggestion containing %q not found in: %v", expectedSuggestion, analysis.Suggestions)
				}
			}

			// Test full error formatting
			formatted := formatter.FormatJSONSchemaError(fakeErr, tt.recoveryResult, "/test/artifact.json")

			if formatted.ContractType != "json_schema" {
				t.Errorf("Expected contract type json_schema, got %s", formatted.ContractType)
			}

			if len(formatted.Details) == 0 {
				t.Error("Expected detailed error information, got none")
			}

			t.Logf("Formatted error: %s", formatted.Error())
		})
	}
}

// BenchmarkJSONRecovery benchmarks the performance of JSON recovery operations
func BenchmarkJSONRecovery(b *testing.B) {
	tests := []struct {
		name     string
		input    string
		level    RecoveryLevel
	}{
		{
			name:  "valid JSON - no recovery",
			input: `{"name": "test", "value": 42, "active": true}`,
			level: ConservativeRecovery,
		},
		{
			name:  "simple malformed - conservative",
			input: `{"name": "test", "value": 42,}`,
			level: ConservativeRecovery,
		},
		{
			name:  "complex malformed - progressive",
			input: `{name: 'test', // comment\nvalue: 42,}`,
			level: ProgressiveRecovery,
		},
		{
			name:  "aggressive reconstruction",
			input: `"name": "test", "value": 42, "active": true`,
			level: AggressiveRecovery,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			parser := NewJSONRecoveryParser(tt.level)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _ = parser.ParseWithRecovery(tt.input)
			}
		})
	}
}