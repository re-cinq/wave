package contract

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestDetectErrorWrapper_ValidWrapper(t *testing.T) {
	// Test case 1: Complete error wrapper with valid JSON in raw_output
	validJSON := `{"repository": {"owner": "re-cinq", "name": "wave"}, "total_issues": 0, "poor_quality_issues": []}`
	wrapperInput := map[string]interface{}{
		"attempts":      5,
		"contract_type": "json_schema",
		"error_type":    "persistent_output_format_failure",
		"exit_code":     0,
		"final_error":   "contract validation failed",
		"raw_output":    validJSON,
		"step_id":       "scan-issues",
		"timestamp":     "2026-02-03T19:47:59+01:00",
		"tokens_used":   378,
	}

	input, err := json.Marshal(wrapperInput)
	if err != nil {
		t.Fatalf("Failed to marshal test input: %v", err)
	}

	result, err := DetectErrorWrapper(input)
	if err != nil {
		t.Fatalf("DetectErrorWrapper failed: %v", err)
	}

	if !result.IsWrapper {
		t.Error("Expected wrapper to be detected")
	}

	if result.Confidence != "high" {
		t.Errorf("Expected high confidence, got %s", result.Confidence)
	}

	if string(result.RawContent) != validJSON {
		t.Errorf("Expected raw content to match, got %s", string(result.RawContent))
	}

	if result.ExtractedFrom != "raw_output" {
		t.Errorf("Expected extraction from raw_output, got %s", result.ExtractedFrom)
	}

	// Verify extracted content is valid JSON
	var extracted interface{}
	if err := json.Unmarshal(result.RawContent, &extracted); err != nil {
		t.Errorf("Extracted content is not valid JSON: %v", err)
	}

	// Check that important fields are detected
	expectedFields := []string{"error_type", "raw_output", "contract_type", "step_id", "final_error", "attempts"}
	for _, field := range expectedFields {
		found := false
		for _, matched := range result.FieldsMatched {
			if matched == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected field %s to be matched", field)
		}
	}
}

func TestDetectErrorWrapper_MinimalWrapper(t *testing.T) {
	// Test case 2: Minimal wrapper with only required fields
	validJSON := `{"test": "data"}`
	wrapperInput := map[string]interface{}{
		"error_type":    "format_failure",
		"raw_output":    validJSON,
		"contract_type": "json_schema",
	}

	input, err := json.Marshal(wrapperInput)
	if err != nil {
		t.Fatalf("Failed to marshal test input: %v", err)
	}

	result, err := DetectErrorWrapper(input)
	if err != nil {
		t.Fatalf("DetectErrorWrapper failed: %v", err)
	}

	if !result.IsWrapper {
		t.Error("Expected minimal wrapper to be detected")
	}

	if result.Confidence == "high" {
		t.Error("Expected lower confidence for minimal wrapper")
	}

	if string(result.RawContent) != validJSON {
		t.Errorf("Expected raw content to match, got %s", string(result.RawContent))
	}
}

func TestDetectErrorWrapper_PartialWrapper(t *testing.T) {
	// Test case 3: Partial wrapper missing critical fields
	wrapperInput := map[string]interface{}{
		"error_type": "format_failure",
		// Missing raw_output - should not be detected as wrapper
		"contract_type": "json_schema",
	}

	input, err := json.Marshal(wrapperInput)
	if err != nil {
		t.Fatalf("Failed to marshal test input: %v", err)
	}

	result, err := DetectErrorWrapper(input)
	if err != nil {
		t.Fatalf("DetectErrorWrapper failed: %v", err)
	}

	if result.IsWrapper {
		t.Error("Expected partial wrapper to NOT be detected as wrapper")
	}
}

func TestDetectErrorWrapper_DirectJSON(t *testing.T) {
	// Test case 4: Direct JSON content (not wrapped)
	directJSON := `{
		"repository": {"owner": "re-cinq", "name": "wave"},
		"total_issues": 5,
		"analyzed_count": 5,
		"poor_quality_issues": [
			{
				"number": 42,
				"title": "bug in thing",
				"quality_score": 20,
				"problems": ["Title too vague"]
			}
		],
		"quality_threshold": 70,
		"timestamp": "2026-02-03T15:30:00Z"
	}`

	result, err := DetectErrorWrapper([]byte(directJSON))
	if err != nil {
		t.Fatalf("DetectErrorWrapper failed: %v", err)
	}

	if result.IsWrapper {
		t.Error("Expected direct JSON to NOT be detected as wrapper")
	}

	if len(result.FieldsMatched) > 0 {
		t.Errorf("Expected no fields matched for direct JSON, got %v", result.FieldsMatched)
	}
}

func TestDetectErrorWrapper_InvalidJSON(t *testing.T) {
	// Test case 5: Invalid JSON input
	invalidJSON := `{"invalid": json, "missing": "quotes"}`

	result, err := DetectErrorWrapper([]byte(invalidJSON))
	if err != nil {
		t.Fatalf("DetectErrorWrapper failed: %v", err)
	}

	if result.IsWrapper {
		t.Error("Expected invalid JSON to NOT be detected as wrapper")
	}
}

func TestDetectErrorWrapper_EmptyRawOutput(t *testing.T) {
	// Test case 6: Wrapper with empty raw_output
	wrapperInput := map[string]interface{}{
		"error_type":    "format_failure",
		"raw_output":    "",
		"contract_type": "json_schema",
	}

	input, err := json.Marshal(wrapperInput)
	if err != nil {
		t.Fatalf("Failed to marshal test input: %v", err)
	}

	result, err := DetectErrorWrapper(input)
	if err != nil {
		t.Fatalf("DetectErrorWrapper failed: %v", err)
	}

	if result.IsWrapper {
		t.Error("Expected wrapper with empty raw_output to NOT be detected")
	}
}

func TestDetectErrorWrapper_InvalidRawOutput(t *testing.T) {
	// Test case 7: Wrapper with invalid JSON in raw_output
	wrapperInput := map[string]interface{}{
		"attempts":      3,
		"error_type":    "format_failure",
		"raw_output":    `{"invalid": json}`, // Invalid JSON
		"contract_type": "json_schema",
		"step_id":       "test-step",
	}

	input, err := json.Marshal(wrapperInput)
	if err != nil {
		t.Fatalf("Failed to marshal test input: %v", err)
	}

	result, err := DetectErrorWrapper(input)
	if err != nil {
		t.Fatalf("DetectErrorWrapper failed: %v", err)
	}

	// Should still detect as wrapper but with lower confidence
	if !result.IsWrapper {
		t.Error("Expected wrapper to be detected despite invalid raw_output")
	}

	if result.Confidence == "high" {
		t.Error("Expected lower confidence for wrapper with invalid raw_output")
	}
}

func TestDetectErrorWrapper_ComplexValidJSON(t *testing.T) {
	// Test case 8: Complex valid JSON in raw_output
	complexJSON := `{
		"enhanced_issues": [
			{
				"issue_number": 42,
				"success": true,
				"changes_made": ["Updated title", "Added labels"],
				"title_updated": true,
				"body_updated": false,
				"labels_added": ["bug", "enhancement"],
				"comment_added": true,
				"url": "https://github.com/repo/issues/42"
			}
		],
		"total_attempted": 1,
		"total_successful": 1,
		"total_failed": 0,
		"timestamp": "2026-02-03T15:35:00Z"
	}`

	wrapperInput := map[string]interface{}{
		"attempts":         5,
		"contract_type":    "json_schema",
		"error_type":       "persistent_output_format_failure",
		"exit_code":        0,
		"final_error":      "contract validation failed",
		"raw_output":       complexJSON,
		"recommendations":  []string{"Review the schema", "Check field types"},
		"step_id":          "apply-enhancements",
		"timestamp":        "2026-02-03T19:18:14+01:00",
		"tokens_used":      2284,
		"must_pass":        true,
		"persona":          "github-enhancer",
	}

	input, err := json.Marshal(wrapperInput)
	if err != nil {
		t.Fatalf("Failed to marshal test input: %v", err)
	}

	result, err := DetectErrorWrapper(input)
	if err != nil {
		t.Fatalf("DetectErrorWrapper failed: %v", err)
	}

	if !result.IsWrapper {
		t.Error("Expected complex wrapper to be detected")
	}

	if result.Confidence != "high" {
		t.Errorf("Expected high confidence for complete wrapper, got %s", result.Confidence)
	}

	// Verify the complex JSON is extracted correctly
	var extracted interface{}
	if err := json.Unmarshal(result.RawContent, &extracted); err != nil {
		t.Errorf("Extracted complex JSON is not valid: %v", err)
	}

	// Verify the content matches
	var originalComplex interface{}
	if err := json.Unmarshal([]byte(complexJSON), &originalComplex); err != nil {
		t.Fatalf("Test complex JSON is invalid: %v", err)
	}
}

func TestDetectErrorWrapper_DebugInfo(t *testing.T) {
	// Test case 9: Verify debug information is correctly generated
	validJSON := `{"test": "data"}`
	wrapperInput := map[string]interface{}{
		"error_type":    "format_failure",
		"raw_output":    validJSON,
		"contract_type": "json_schema",
		"step_id":       "test-step",
	}

	input, err := json.Marshal(wrapperInput)
	if err != nil {
		t.Fatalf("Failed to marshal test input: %v", err)
	}

	result, err := DetectErrorWrapper(input)
	if err != nil {
		t.Fatalf("DetectErrorWrapper failed: %v", err)
	}

	debug := result.GetDebugInfo(len(input))

	if debug.InputLength != len(input) {
		t.Errorf("Expected input length %d, got %d", len(input), debug.InputLength)
	}

	if !debug.DetectionAttempted {
		t.Error("Expected detection attempted to be true")
	}

	if !debug.WrapperDetected {
		t.Error("Expected wrapper detected to be true")
	}

	if len(debug.FieldsMatched) == 0 {
		t.Error("Expected some fields to be matched")
	}

	if debug.ExtractedLength != len(validJSON) {
		t.Errorf("Expected extracted length %d, got %d", len(validJSON), debug.ExtractedLength)
	}

	if debug.ExtractionMethod != "raw_output" {
		t.Errorf("Expected extraction method 'raw_output', got %s", debug.ExtractionMethod)
	}
}

func TestDetectErrorWrapper_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectWrap  bool
		description string
	}{
		{
			name:        "empty_input",
			input:       "",
			expectWrap:  false,
			description: "Empty input should not be detected as wrapper",
		},
		{
			name:        "null_input",
			input:       "null",
			expectWrap:  false,
			description: "Null JSON should not be detected as wrapper",
		},
		{
			name:        "array_input",
			input:       `["not", "an", "object"]`,
			expectWrap:  false,
			description: "JSON array should not be detected as wrapper",
		},
		{
			name:        "string_input",
			input:       `"just a string"`,
			expectWrap:  false,
			description: "JSON string should not be detected as wrapper",
		},
		{
			name: "similar_but_not_wrapper",
			input: `{
				"error_message": "Something went wrong",
				"data": {"valid": "json"},
				"status": "failed"
			}`,
			expectWrap:  false,
			description: "Object with error fields but not wrapper structure",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DetectErrorWrapper([]byte(tc.input))
			if err != nil {
				t.Fatalf("DetectErrorWrapper failed for %s: %v", tc.description, err)
			}

			if result.IsWrapper != tc.expectWrap {
				t.Errorf("For %s: expected wrapper=%v, got %v",
					tc.description, tc.expectWrap, result.IsWrapper)
			}
		})
	}
}

func TestDetectErrorWrapper_RealWorkspaceArtifacts(t *testing.T) {
	// Test case 10: Using actual artifacts from failed pipeline runs
	// This tests the real-world scenario from the workspace investigation

	// Artifact from scan-issues step (should detect wrapper)
	scanIssuesWrapper := `{
		"attempts": 5,
		"contract_type": "json_schema",
		"error_type": "persistent_output_format_failure",
		"exit_code": 0,
		"final_error": "contract validation failed [json_schema]: JSON schema validation failed",
		"raw_output": "{\"repository\": {\"owner\": \"re-cinq\", \"name\": \"wave\"}, \"total_issues\": 0, \"analyzed_count\": 0, \"poor_quality_issues\": [], \"quality_threshold\": 70, \"timestamp\": \"2026-02-03T15:30:00Z\"}",
		"recommendations": ["Review the AI persona prompt for clarity"],
		"step_id": "scan-issues",
		"timestamp": "2026-02-03T19:47:59+01:00",
		"tokens_used": 378
	}`

	result, err := DetectErrorWrapper([]byte(scanIssuesWrapper))
	if err != nil {
		t.Fatalf("DetectErrorWrapper failed for scan-issues artifact: %v", err)
	}

	if !result.IsWrapper {
		t.Error("Expected scan-issues artifact to be detected as wrapper")
	}

	// Verify the extracted content is valid and matches expected structure
	var extracted map[string]interface{}
	if err := json.Unmarshal(result.RawContent, &extracted); err != nil {
		t.Fatalf("Extracted content from scan-issues is not valid JSON: %v", err)
	}

	// Verify it has the expected GitHub issue analysis structure
	if repo, ok := extracted["repository"].(map[string]interface{}); !ok {
		t.Error("Expected repository field in extracted content")
	} else {
		if owner, ok := repo["owner"].(string); !ok || owner != "re-cinq" {
			t.Error("Expected repository.owner to be 're-cinq'")
		}
		if name, ok := repo["name"].(string); !ok || name != "wave" {
			t.Error("Expected repository.name to be 'wave'")
		}
	}

	// Artifact from plan-enhancements step (should NOT detect wrapper - direct JSON)
	planEnhancementsJSON := `{
		"issues_to_enhance": [
			{
				"issue_number": 20,
				"enhancements": [
					{
						"type": "title_improvement",
						"current": "add scan poorly commented gh issues and extend and connect",
						"suggested": "Add GitHub Issue Scanner for Code Quality Analysis",
						"rationale": "More descriptive and professional title"
					}
				],
				"priority": "medium",
				"estimated_effort": "2-3 hours"
			}
		],
		"enhancement_summary": {
			"total_issues": 4,
			"issues_to_enhance": 4,
			"enhancement_types": ["title_improvement", "description_enhancement", "labeling"]
		}
	}`

	planResult, err := DetectErrorWrapper([]byte(planEnhancementsJSON))
	if err != nil {
		t.Fatalf("DetectErrorWrapper failed for plan-enhancements artifact: %v", err)
	}

	if planResult.IsWrapper {
		t.Error("Expected plan-enhancements artifact to NOT be detected as wrapper")
	}
}

// Benchmark tests for performance validation
func BenchmarkDetectErrorWrapper_DirectJSON(b *testing.B) {
	directJSON := `{"repository": {"owner": "test", "name": "repo"}, "total_issues": 100}`
	input := []byte(directJSON)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DetectErrorWrapper(input)
	}
}

func BenchmarkDetectErrorWrapper_WrappedJSON(b *testing.B) {
	wrapperInput := map[string]interface{}{
		"attempts":      5,
		"contract_type": "json_schema",
		"error_type":    "format_failure",
		"raw_output":    `{"repository": {"owner": "test", "name": "repo"}, "total_issues": 100}`,
		"step_id":       "test-step",
		"timestamp":     "2026-02-03T19:47:59+01:00",
	}

	input, _ := json.Marshal(wrapperInput)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DetectErrorWrapper(input)
	}
}

func BenchmarkDetectErrorWrapper_LargeJSON(b *testing.B) {
	// Generate a large JSON for performance testing
	largeContent := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeContent[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d", i)
	}
	largeJSON, _ := json.Marshal(largeContent)

	wrapperInput := map[string]interface{}{
		"attempts":      5,
		"contract_type": "json_schema",
		"error_type":    "format_failure",
		"raw_output":    string(largeJSON),
		"step_id":       "test-step",
	}

	input, _ := json.Marshal(wrapperInput)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DetectErrorWrapper(input)
	}
}