package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWrapperDetectionIntegration(t *testing.T) {
	// Test the full integration of wrapper detection with validation system

	// Create a temporary workspace directory
	tempDir, err := os.MkdirTemp("", "wrapper_integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test schema for GitHub issue analysis (matching the real use case)
	schema := map[string]interface{}{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]interface{}{
			"repository": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner": map[string]interface{}{"type": "string"},
					"name":  map[string]interface{}{"type": "string"},
				},
				"required": []string{"owner", "name"},
			},
			"total_issues":         map[string]interface{}{"type": "integer"},
			"poor_quality_issues":  map[string]interface{}{"type": "array"},
			"quality_threshold":    map[string]interface{}{"type": "integer"},
			"timestamp":           map[string]interface{}{"type": "string"},
		},
		"required": []string{"repository", "total_issues", "poor_quality_issues"},
	}

	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	testCases := []struct {
		name           string
		artifact       interface{}
		expectSuccess  bool
		useWrapper     bool
		description    string
	}{
		{
			name: "direct_valid_json",
			artifact: map[string]interface{}{
				"repository": map[string]interface{}{
					"owner": "re-cinq",
					"name":  "wave",
				},
				"total_issues":         0,
				"poor_quality_issues":  []interface{}{},
				"quality_threshold":    70,
				"timestamp":           "2026-02-03T15:30:00Z",
			},
			expectSuccess: true,
			useWrapper:    false,
			description:   "Direct valid JSON should pass validation normally",
		},
		{
			name: "wrapped_valid_json",
			artifact: map[string]interface{}{
				"repository": map[string]interface{}{
					"owner": "re-cinq",
					"name":  "wave",
				},
				"total_issues":         0,
				"poor_quality_issues":  []interface{}{},
				"quality_threshold":    70,
				"timestamp":           "2026-02-03T15:30:00Z",
			},
			expectSuccess: true,
			useWrapper:    true,
			description:   "Valid JSON wrapped in error metadata should be extracted and validated successfully",
		},
		{
			name: "wrapped_invalid_json",
			artifact: map[string]interface{}{
				"repository": map[string]interface{}{
					"owner": "re-cinq",
					"name":  "wave",
				},
				"total_issues": 0,
				// Missing required field: poor_quality_issues
			},
			expectSuccess: false,
			useWrapper:    true,
			description:   "Invalid JSON wrapped in error metadata should fail validation on the extracted content",
		},
		{
			name: "direct_invalid_json",
			artifact: map[string]interface{}{
				"repository": map[string]interface{}{
					"owner": "re-cinq",
					"name":  "wave",
				},
				// Missing required fields: total_issues, poor_quality_issues
			},
			expectSuccess: false,
			useWrapper:    false,
			description:   "Direct invalid JSON should fail validation normally",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create workspace for this test
			workspaceDir := filepath.Join(tempDir, tc.name)
			if err := os.MkdirAll(workspaceDir, 0755); err != nil {
				t.Fatalf("Failed to create workspace dir: %v", err)
			}

			// Prepare artifact content
			var artifactContent []byte
			if tc.useWrapper {
				// Wrap the artifact in error metadata
				validJSON, _ := json.Marshal(tc.artifact)
				wrapper := map[string]interface{}{
					"attempts":      5,
					"contract_type": "json_schema",
					"error_type":    "persistent_output_format_failure",
					"exit_code":     0,
					"final_error":   "contract validation failed",
					"raw_output":    string(validJSON),
					"step_id":       tc.name,
					"timestamp":     "2026-02-03T19:47:59+01:00",
					"tokens_used":   378,
				}
				artifactContent, _ = json.Marshal(wrapper)
			} else {
				// Use artifact directly
				artifactContent, _ = json.Marshal(tc.artifact)
			}

			// Write artifact file
			artifactPath := filepath.Join(workspaceDir, "artifact.json")
			if err := os.WriteFile(artifactPath, artifactContent, 0644); err != nil {
				t.Fatalf("Failed to write artifact file: %v", err)
			}

			// Configure validator
			cfg := ContractConfig{
				Type:                    "json_schema",
				Schema:                  string(schemaBytes),
				Source:                  "artifact.json",
				MustPass:                true,
				AllowRecovery:          true,
				DisableWrapperDetection: false, // Ensure wrapper detection is enabled
				DebugMode:               true,  // Enable debug output for troubleshooting
			}

			// Run validation
			validator := &jsonSchemaValidator{}
			err := validator.Validate(cfg, workspaceDir)

			// Check result
			if tc.expectSuccess {
				if err != nil {
					t.Errorf("Expected validation to succeed for %s, but got error: %v", tc.description, err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected validation to fail for %s, but it succeeded", tc.description)
				}
			}

			// For wrapper cases, also verify that wrapper detection was triggered
			if tc.useWrapper && tc.expectSuccess {
				// Read the original artifact to verify it contains wrapper structure
				var originalData map[string]interface{}
				if err := json.Unmarshal(artifactContent, &originalData); err != nil {
					t.Fatalf("Failed to unmarshal original artifact: %v", err)
				}

				if _, hasErrorType := originalData["error_type"]; !hasErrorType {
					t.Error("Expected original artifact to contain error_type (wrapper structure)")
				}

				if rawOutput, hasRawOutput := originalData["raw_output"]; !hasRawOutput {
					t.Error("Expected original artifact to contain raw_output")
				} else {
					// Verify the raw_output contains the valid JSON
					var extractedData map[string]interface{}
					if err := json.Unmarshal([]byte(rawOutput.(string)), &extractedData); err != nil {
						t.Errorf("Failed to parse raw_output as JSON: %v", err)
					}
				}
			}
		})
	}
}

func TestWrapperDetectionDisabled(t *testing.T) {
	// Test that wrapper detection can be disabled via configuration

	tempDir, err := os.MkdirTemp("", "wrapper_disabled_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a schema
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"test": map[string]interface{}{"type": "string"},
		},
		"required": []string{"test"},
	}
	schemaBytes, _ := json.Marshal(schema)

	// Create a wrapped artifact (should fail when detection is disabled)
	validJSON := `{"test": "value"}`
	wrapper := map[string]interface{}{
		"error_type":    "format_failure",
		"raw_output":    validJSON,
		"contract_type": "json_schema",
	}
	wrapperBytes, _ := json.Marshal(wrapper)

	// Write artifact
	artifactPath := filepath.Join(tempDir, "artifact.json")
	if err := os.WriteFile(artifactPath, wrapperBytes, 0644); err != nil {
		t.Fatalf("Failed to write artifact: %v", err)
	}

	// Test with wrapper detection disabled
	cfg := ContractConfig{
		Type:                    "json_schema",
		Schema:                  string(schemaBytes),
		Source:                  "artifact.json",
		MustPass:                true,
		DisableWrapperDetection: true, // Disable wrapper detection
	}

	validator := &jsonSchemaValidator{}
	err = validator.Validate(cfg, tempDir)

	// Should fail because we're validating the wrapper structure against the schema
	if err == nil {
		t.Error("Expected validation to fail when wrapper detection is disabled and wrapper is present")
	}

	// Test with wrapper detection enabled (default behavior)
	cfg.DisableWrapperDetection = false
	err = validator.Validate(cfg, tempDir)

	// Should succeed because wrapper is detected and raw content is extracted
	if err != nil {
		t.Errorf("Expected validation to succeed when wrapper detection is enabled, got: %v", err)
	}
}

func TestWrapperDetectionWithRealWorldArtifacts(t *testing.T) {
	// Test with artifacts that match the real-world examples from the workspace investigation

	tempDir, err := os.MkdirTemp("", "real_artifacts_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// GitHub Issue Analysis schema (from scan-issues step)
	issueAnalysisSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"repository": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner": map[string]interface{}{"type": "string"},
					"name":  map[string]interface{}{"type": "string"},
				},
				"required": []string{"owner", "name"},
			},
			"total_issues":         map[string]interface{}{"type": "integer"},
			"analyzed_count":       map[string]interface{}{"type": "integer"},
			"poor_quality_issues":  map[string]interface{}{"type": "array"},
			"quality_threshold":    map[string]interface{}{"type": "integer"},
			"timestamp":           map[string]interface{}{"type": "string"},
		},
		"required": []string{"repository", "total_issues", "poor_quality_issues"},
	}

	// Enhancement Results schema (from apply-enhancements step)
	enhancementResultsSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"enhanced_issues": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"issue_number": map[string]interface{}{"type": "integer"},
						"success":      map[string]interface{}{"type": "boolean"},
						"changes_made": map[string]interface{}{"type": "array"},
					},
					"required": []string{"issue_number", "success", "changes_made"},
				},
			},
			"total_attempted":  map[string]interface{}{"type": "integer"},
			"total_successful": map[string]interface{}{"type": "integer"},
			"total_failed":     map[string]interface{}{"type": "integer"},
		},
		"required": []string{"enhanced_issues", "total_attempted", "total_successful"},
	}

	testCases := []struct {
		name       string
		schema     map[string]interface{}
		artifact   string
		expectPass bool
	}{
		{
			name:   "scan_issues_wrapper",
			schema: issueAnalysisSchema,
			artifact: `{
				"attempts": 5,
				"contract_type": "json_schema",
				"error_type": "persistent_output_format_failure",
				"exit_code": 0,
				"final_error": "contract validation failed",
				"raw_output": "{\"repository\": {\"owner\": \"re-cinq\", \"name\": \"wave\"}, \"total_issues\": 0, \"analyzed_count\": 0, \"poor_quality_issues\": [], \"quality_threshold\": 70, \"timestamp\": \"2026-02-03T15:30:00Z\"}",
				"step_id": "scan-issues",
				"timestamp": "2026-02-03T19:47:59+01:00",
				"tokens_used": 378
			}`,
			expectPass: true,
		},
		{
			name:   "apply_enhancements_wrapper",
			schema: enhancementResultsSchema,
			artifact: `{
				"attempts": 5,
				"contract_type": "json_schema",
				"error_type": "persistent_output_format_failure",
				"exit_code": 0,
				"final_error": "contract validation failed",
				"raw_output": "{\"enhanced_issues\": [{\"issue_number\": 20, \"success\": true, \"changes_made\": [\"Updated title\", \"Added labels\"]}], \"total_attempted\": 1, \"total_successful\": 1, \"total_failed\": 0}",
				"step_id": "apply-enhancements",
				"timestamp": "2026-02-03T19:18:14+01:00",
				"tokens_used": 2284
			}`,
			expectPass: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workspaceDir := filepath.Join(tempDir, tc.name)
			if err := os.MkdirAll(workspaceDir, 0755); err != nil {
				t.Fatalf("Failed to create workspace dir: %v", err)
			}

			// Write artifact
			artifactPath := filepath.Join(workspaceDir, "artifact.json")
			if err := os.WriteFile(artifactPath, []byte(tc.artifact), 0644); err != nil {
				t.Fatalf("Failed to write artifact: %v", err)
			}

			// Configure validator
			schemaBytes, _ := json.Marshal(tc.schema)
			cfg := ContractConfig{
				Type:      "json_schema",
				Schema:    string(schemaBytes),
				Source:    "artifact.json",
				MustPass:  true,
				DebugMode: true,
			}

			// Run validation
			validator := &jsonSchemaValidator{}
			err := validator.Validate(cfg, workspaceDir)

			if tc.expectPass {
				if err != nil {
					t.Errorf("Expected %s to pass validation after wrapper extraction, but got: %v", tc.name, err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected %s to fail validation, but it passed", tc.name)
				}
			}
		})
	}
}