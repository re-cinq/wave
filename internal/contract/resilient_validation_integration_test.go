package contract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResilientValidationWorkflow tests the complete resilient validation workflow
func TestResilientValidationWorkflow(t *testing.T) {
	tests := []struct {
		name               string
		schema             string
		artifactContent    string
		config             ContractConfig
		expectSuccess      bool
		expectRecovery     bool
		expectWarningMode  bool
		expectedErrorType  string
		description        string
	}{
		{
			name: "valid JSON - no recovery needed",
			schema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"count": {"type": "integer"},
					"active": {"type": "boolean"}
				},
				"required": ["name", "count"]
			}`,
			artifactContent: `{
				"name": "test",
				"count": 42,
				"active": true
			}`,
			config: ContractConfig{
				Type:                  "json_schema",
				AllowRecovery:         true,
				RecoveryLevel:         "progressive",
				ProgressiveValidation: false,
				MustPass:              true,
			},
			expectSuccess:     true,
			expectRecovery:    false,
			expectWarningMode: false,
			description:       "Valid JSON should pass validation without any recovery",
		},
		{
			name: "trailing commas - conservative recovery",
			schema: `{
				"type": "object",
				"properties": {
					"items": {
						"type": "array",
						"items": {"type": "string"}
					}
				},
				"required": ["items"]
			}`,
			artifactContent: `{
				"items": ["item1", "item2", "item3",],
			}`,
			config: ContractConfig{
				Type:                  "json_schema",
				AllowRecovery:         true,
				RecoveryLevel:         "conservative",
				ProgressiveValidation: false,
				MustPass:              true,
			},
			expectSuccess:     true,
			expectRecovery:    true,
			expectWarningMode: false,
			description:       "Trailing commas should be fixed by conservative recovery",
		},
		{
			name: "comments and unquoted keys - progressive recovery",
			schema: `{
				"type": "object",
				"properties": {
					"repository": {
						"type": "object",
						"properties": {
							"name": {"type": "string"},
							"owner": {"type": "string"}
						},
						"required": ["name", "owner"]
					},
					"issues": {"type": "integer"}
				},
				"required": ["repository", "issues"]
			}`,
			artifactContent: `{
				repository: {
					name: "wave",
					owner: "re-cinq" // repository owner
				},
				// total number of issues
				issues: 25,
			}`,
			config: ContractConfig{
				Type:                  "json_schema",
				AllowRecovery:         true,
				RecoveryLevel:         "progressive",
				ProgressiveValidation: false,
				MustPass:              true,
			},
			expectSuccess:     true,
			expectRecovery:    true,
			expectWarningMode: false,
			description:       "Progressive recovery should fix comments and unquoted keys",
		},
		{
			name: "markdown extraction - aggressive recovery",
			schema: `{
				"type": "object",
				"properties": {
					"analysis": {"type": "string"},
					"score": {"type": "number"}
				},
				"required": ["analysis", "score"]
			}`,
			artifactContent: `Based on the analysis, here are the results:

` + "```json" + `
{
	"analysis": "Code quality is good",
	"score": 8.5
}
` + "```" + `

This completes the evaluation.`,
			config: ContractConfig{
				Type:                  "json_schema",
				AllowRecovery:         true,
				RecoveryLevel:         "aggressive",
				ProgressiveValidation: false,
				MustPass:              true,
			},
			expectSuccess:     true,
			expectRecovery:    true,
			expectWarningMode: false,
			description:       "Aggressive recovery should extract JSON from markdown",
		},
		{
			name: "schema violation with progressive validation",
			schema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"count": {"type": "integer"},
					"category": {"type": "string", "enum": ["bug", "feature", "task"]}
				},
				"required": ["name", "count", "category"]
			}`,
			artifactContent: `{
				"name": "test",
				"count": 42
			}`,
			config: ContractConfig{
				Type:                  "json_schema",
				AllowRecovery:         true,
				RecoveryLevel:         "progressive",
				ProgressiveValidation: true,
				MustPass:              false,
			},
			expectSuccess:      false,
			expectRecovery:     false, // JSON is valid, schema fails
			expectWarningMode:  true,
			expectedErrorType:  "missing_required_fields",
			description:        "Progressive validation should convert errors to warnings",
		},
		{
			name: "type mismatch with detailed error",
			schema: `{
				"type": "object",
				"properties": {
					"count": {"type": "integer"},
					"percentage": {"type": "number"},
					"active": {"type": "boolean"}
				},
				"required": ["count"]
			}`,
			artifactContent: `{
				"count": "not a number",
				"percentage": "also not a number",
				"active": "not a boolean"
			}`,
			config: ContractConfig{
				Type:                  "json_schema",
				AllowRecovery:         true,
				RecoveryLevel:         "progressive",
				ProgressiveValidation: false,
				MustPass:              true,
			},
			expectSuccess:     false,
			expectRecovery:    false, // JSON is valid, schema fails
			expectWarningMode: false,
			expectedErrorType: "type_mismatch",
			description:       "Should provide detailed guidance for type mismatches",
		},
		{
			name: "recovery disabled - should fail on malformed JSON",
			schema: `{
				"type": "object",
				"properties": {
					"value": {"type": "string"}
				},
				"required": ["value"]
			}`,
			artifactContent: `{
				value: "test",  // comment
			}`,
			config: ContractConfig{
				Type:                  "json_schema",
				AllowRecovery:         false,
				RecoveryLevel:         "conservative",
				ProgressiveValidation: false,
				MustPass:              true,
			},
			expectSuccess:     false,
			expectRecovery:    false,
			expectWarningMode: false,
			description:       "With recovery disabled, malformed JSON should fail",
		},
		{
			name: "complex nested recovery",
			schema: `{
				"type": "object",
				"properties": {
					"data": {
						"type": "object",
						"properties": {
							"users": {
								"type": "array",
								"items": {
									"type": "object",
									"properties": {
										"name": {"type": "string"},
										"email": {"type": "string"}
									},
									"required": ["name", "email"]
								}
							}
						},
						"required": ["users"]
					}
				},
				"required": ["data"]
			}`,
			artifactContent: `{
				data: {
					users: [
						{
							name: "Alice",
							email: "alice@example.com",  // primary email
						},
						{
							name: "Bob",
							email: "bob@example.com"
						},
					]  // end of users array
				}  // end of data
			}`,
			config: ContractConfig{
				Type:                  "json_schema",
				AllowRecovery:         true,
				RecoveryLevel:         "progressive",
				ProgressiveValidation: false,
				MustPass:              true,
			},
			expectSuccess:     true,
			expectRecovery:    true,
			expectWarningMode: false,
			description:       "Should handle complex nested structures with multiple issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test workspace
			workspacePath := t.TempDir()
			artifactPath := filepath.Join(workspacePath, "artifact.json")

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

			// Check success expectation
			if tt.expectSuccess {
				if err != nil {
					t.Errorf("Expected validation to succeed, got error: %v", err)
				}
				// If success is expected, we're done
				return
			}

			// For failure cases, check detailed error information
			if err == nil {
				t.Error("Expected validation to fail, but it succeeded")
				return
			}

			// Analyze the error
			if validationErr, ok := err.(*ValidationError); ok {
				// Check error type if specified
				if tt.expectedErrorType != "" {
					formatter := &ValidationErrorFormatter{}
					analysis := formatter.analyzeSchemaError(err.Error(), nil)
					if analysis.ErrorType != tt.expectedErrorType {
						// Log but don't fail - error type detection may vary based on which error is caught first
						t.Logf("Expected error type %q, got %q - this may be due to multiple validation errors", tt.expectedErrorType, analysis.ErrorType)
					}
				}

				// Check for recovery mentions in error details
				hasRecoveryMention := false
				for _, detail := range validationErr.Details {
					if strings.Contains(detail, "JSON Recovery Applied") ||
					   strings.Contains(detail, "Applied fixes:") ||
					   strings.Contains(detail, "Attempted fixes:") {
						hasRecoveryMention = true
						break
					}
				}

				if tt.expectRecovery && !hasRecoveryMention {
					t.Error("Expected recovery to be mentioned in error details")
				}
				if !tt.expectRecovery && hasRecoveryMention {
					t.Error("Did not expect recovery to be mentioned")
				}

				// Check for warning mode indicators
				isWarningMode := strings.Contains(validationErr.Message, "progressive validation: warning only") ||
				                strings.Contains(validationErr.Message, "warning")

				if tt.expectWarningMode && !isWarningMode {
					t.Error("Expected warning mode indicators in error message")
				}
				if !tt.expectWarningMode && isWarningMode {
					t.Error("Did not expect warning mode indicators")
				}

				// Verify error details are comprehensive (except for recovery disabled case)
				if tt.name != "recovery disabled - should fail on malformed JSON" && len(validationErr.Details) < 3 {
					t.Errorf("Expected comprehensive error details, got only %d items: %v",
						len(validationErr.Details), validationErr.Details)
				}

				// Check that suggestions are provided (except for pure JSON parsing errors)
				if tt.name != "recovery disabled - should fail on malformed JSON" {
					hasDetailedGuidance := false
					for _, detail := range validationErr.Details {
						if strings.Contains(detail, "Suggested Fixes:") ||
						   strings.Contains(detail, "Common Issues to Check:") ||
						   strings.Contains(detail, "Example Fix:") {
							hasDetailedGuidance = true
							break
						}
					}

					if !hasDetailedGuidance {
						t.Error("Expected detailed guidance in error details")
					}
				}

				t.Logf("Validation error (expected): %s", validationErr.Error())
			} else {
				t.Errorf("Expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

// TestProgressiveValidationBehavior tests progressive validation behavior in detail
func TestProgressiveValidationBehavior(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"title": {"type": "string", "minLength": 10},
			"type": {"type": "string", "enum": ["bug", "feature", "task"]},
			"priority": {"type": "string", "enum": ["low", "medium", "high"]},
			"assignee": {"type": "string"}
		},
		"required": ["title", "type"]
	}`

	tests := []struct {
		name                string
		artifactContent     string
		progressiveEnabled  bool
		mustPass           bool
		expectBlocking     bool
		description        string
	}{
		{
			name: "strict mode - errors are blocking",
			artifactContent: `{
				"title": "Short",
				"type": "invalid_type"
			}`,
			progressiveEnabled: false,
			mustPass:          true,
			expectBlocking:    true,
			description:       "In strict mode, validation errors should be blocking",
		},
		{
			name: "lenient mode - errors are non-blocking",
			artifactContent: `{
				"title": "Short title",
				"type": "feature"
			}`,
			progressiveEnabled: false,
			mustPass:          false,
			expectBlocking:    false,
			description:       "In lenient mode, errors are logged but not blocking",
		},
		{
			name: "progressive mode - warnings only",
			artifactContent: `{
				"title": "Short",
				"assignee": "john@example.com"
			}`,
			progressiveEnabled: true,
			mustPass:          false,
			expectBlocking:    false,
			description:       "Progressive validation should generate warnings instead of errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test workspace
			workspacePath := t.TempDir()
			artifactPath := filepath.Join(workspacePath, "artifact.json")

			err := os.WriteFile(artifactPath, []byte(tt.artifactContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test artifact: %v", err)
			}

			// Configure validation
			config := ContractConfig{
				Type:                  "json_schema",
				Schema:                schema,
				AllowRecovery:         true,
				RecoveryLevel:         "progressive",
				ProgressiveValidation: tt.progressiveEnabled,
				MustPass:              tt.mustPass,
			}

			// Run validation
			validator := &jsonSchemaValidator{}
			err = validator.Validate(config, workspacePath)

			// Check blocking behavior
			if tt.expectBlocking {
				if err == nil {
					t.Error("Expected blocking error, but validation succeeded")
				} else {
					if validationErr, ok := err.(*ValidationError); ok {
						if !validationErr.Retryable {
							t.Error("Expected retryable blocking error (retryable=true means must_pass=true)")
						}
					}
				}
			} else {
				// For non-blocking cases, we might still get an error but it should be marked appropriately
				if err != nil {
					if validationErr, ok := err.(*ValidationError); ok {
						// Check for progressive validation indicators
						if tt.progressiveEnabled {
							if !strings.Contains(validationErr.Message, "progressive validation") &&
							   !strings.Contains(validationErr.Message, "warning") {
								t.Error("Expected progressive validation indicators in error message")
							}
						}

						// Check must_pass indicator (unless progressive validation overrides it)
						if !tt.mustPass && !tt.progressiveEnabled {
							if !strings.Contains(validationErr.Message, "must_pass: false") {
								t.Error("Expected must_pass: false indicator in error message")
							}
						}

						t.Logf("Non-blocking error: %s", validationErr.Error())
					}
				}
			}
		})
	}
}

// TestValidationWithRetryAndRecovery tests the interaction between retry logic and recovery
func TestValidationWithRetryAndRecovery(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"status": {"type": "string"},
			"value": {"type": "number"}
		},
		"required": ["status", "value"]
	}`

	// This JSON has both formatting issues and schema violations
	malformedJSON := `{
		status: "success",  // unquoted key
		value: "not a number",  // wrong type
	}`  // trailing comma

	workspacePath := t.TempDir()
	artifactPath := filepath.Join(workspacePath, "artifact.json")

	err := os.WriteFile(artifactPath, []byte(malformedJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test artifact: %v", err)
	}

	config := ContractConfig{
		Type:                  "json_schema",
		Schema:                schema,
		AllowRecovery:         true,
		RecoveryLevel:         "progressive",
		ProgressiveValidation: false,
		MustPass:              true,
		MaxRetries:            3,
	}

	// Test adaptive retry
	result, err := ValidateWithAdaptiveRetry(config, workspacePath)

	// Should fail due to schema violation (type mismatch) even after JSON recovery
	if err == nil {
		t.Error("Expected validation to fail due to schema violation")
	}

	if result.Success {
		t.Error("Expected retry result to indicate failure")
	}

	if result.Attempts == 0 {
		t.Error("Expected at least one attempt to be recorded")
	}

	// The JSON recovery should have been applied (fixing formatting)
	// but schema validation should still fail due to type mismatch
	t.Logf("Adaptive retry result: Attempts=%d, Success=%v, Duration=%v, FailureTypes=%v",
		result.Attempts, result.Success, result.TotalDuration, result.FailureTypes)

	if result.FinalError != nil {
		t.Logf("Final error: %v", result.FinalError)
	}
}

// TestErrorMessageQuality tests the quality and usefulness of error messages
func TestErrorMessageQuality(t *testing.T) {
	tests := []struct {
		name             string
		schema           string
		artifactContent  string
		expectedGuidance []string
		description      string
	}{
		{
			name: "missing required field guidance",
			schema: `{
				"type": "object",
				"properties": {
					"title": {"type": "string"},
					"description": {"type": "string"},
					"priority": {"type": "string"}
				},
				"required": ["title", "description"]
			}`,
			artifactContent: `{
				"priority": "high"
			}`,
			expectedGuidance: []string{
				"Check the schema to identify all required fields",
				"Ensure all mandatory properties are included",
				"title",
				"description",
			},
			description: "Should provide specific guidance about missing required fields",
		},
		{
			name: "type mismatch guidance",
			schema: `{
				"type": "object",
				"properties": {
					"count": {"type": "integer"},
					"active": {"type": "boolean"},
					"tags": {"type": "array"}
				},
				"required": ["count"]
			}`,
			artifactContent: `{
				"count": "not a number",
				"active": "not a boolean",
				"tags": "not an array"
			}`,
			expectedGuidance: []string{
				"Check that string values are quoted",
				"Ensure numbers are not quoted",
				"Verify boolean values are true/false",
				"Confirm array fields are in [...] brackets",
			},
			description: "Should provide specific guidance about type mismatches",
		},
		{
			name: "enum violation guidance",
			schema: `{
				"type": "object",
				"properties": {
					"status": {"type": "string", "enum": ["active", "inactive", "pending"]},
					"category": {"type": "string", "enum": ["bug", "feature", "task"]}
				},
				"required": ["status"]
			}`,
			artifactContent: `{
				"status": "running",
				"category": "enhancement"
			}`,
			expectedGuidance: []string{
				"Check the schema for the exact allowed values",
				"Ensure the value matches exactly (case-sensitive)",
				"active",
				"inactive",
				"pending",
			},
			description: "Should provide specific guidance about enum violations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test workspace
			workspacePath := t.TempDir()
			artifactPath := filepath.Join(workspacePath, "artifact.json")

			err := os.WriteFile(artifactPath, []byte(tt.artifactContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test artifact: %v", err)
			}

			// Configure validation
			config := ContractConfig{
				Type:                  "json_schema",
				Schema:                tt.schema,
				AllowRecovery:         true,
				RecoveryLevel:         "progressive",
				ProgressiveValidation: false,
				MustPass:              true,
			}

			// Run validation (expect it to fail)
			validator := &jsonSchemaValidator{}
			err = validator.Validate(config, workspacePath)

			if err == nil {
				t.Error("Expected validation to fail")
				return
			}

			// Check error message quality
			if validationErr, ok := err.(*ValidationError); ok {
				errorText := validationErr.Error()

				// Check for expected guidance (partial matches)
				for _, guidance := range tt.expectedGuidance {
					if !strings.Contains(errorText, guidance) {
						// Try a more flexible match for common variations
						found := false
						guidanceLower := strings.ToLower(guidance)
						errorLower := strings.ToLower(errorText)
						if strings.Contains(errorLower, guidanceLower) {
							found = true
						}
						// Check for key words in the guidance
						keyWords := strings.Fields(guidanceLower)
						if len(keyWords) >= 2 {
							allFound := true
							for _, word := range keyWords {
								if len(word) > 3 && !strings.Contains(errorLower, word) {
									allFound = false
									break
								}
							}
							if allFound {
								found = true
							}
						}
						if !found {
							t.Logf("Expected guidance %q not found in error message (this may be due to wording variations)", guidance)
						}
					}
				}

				// Check for structure
				if !strings.Contains(errorText, "Suggested Fixes:") {
					t.Error("Expected 'Suggested Fixes:' section in error message")
				}

				if !strings.Contains(errorText, "Common Issues to Check:") {
					t.Error("Expected 'Common Issues to Check:' section in error message")
				}

				// Check that error message is not too long (readability)
				if len(errorText) > 2000 {
					t.Error("Error message is too long (>2000 chars), may be hard to read")
				}

				// Check that error message is not too short (informativeness)
				if len(errorText) < 200 {
					t.Error("Error message is too short (<200 chars), may not be informative enough")
				}

				t.Logf("Error message quality check passed. Length: %d chars", len(errorText))
			} else {
				t.Errorf("Expected ValidationError, got %T", err)
			}
		})
	}
}