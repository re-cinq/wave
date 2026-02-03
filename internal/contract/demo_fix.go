package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DemoValidationFix demonstrates how the wrapper detection fix resolves the validation pipeline issue
func DemoValidationFix() error {
	fmt.Println("🔧 Wave Validation Pipeline Fix Demo")
	fmt.Println("====================================")

	// Create temporary workspace
	tempDir, err := os.MkdirTemp("", "wave_fix_demo")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Define schema (GitHub Issue Analysis)
	schema := map[string]interface{}{
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
			"total_issues":        map[string]interface{}{"type": "integer"},
			"poor_quality_issues": map[string]interface{}{"type": "array"},
		},
		"required": []string{"repository", "total_issues", "poor_quality_issues"},
	}
	schemaBytes, _ := json.Marshal(schema)

	// Valid JSON that AI produced
	validAIOutput := map[string]interface{}{
		"repository": map[string]interface{}{
			"owner": "re-cinq",
			"name":  "wave",
		},
		"total_issues":        0,
		"poor_quality_issues": []interface{}{},
	}
	validJSON, _ := json.Marshal(validAIOutput)

	// Error wrapper that Wave incorrectly creates
	errorWrapper := map[string]interface{}{
		"attempts":      5,
		"contract_type": "json_schema",
		"error_type":    "persistent_output_format_failure",
		"exit_code":     0,
		"final_error":   "contract validation failed",
		"raw_output":    string(validJSON),
		"step_id":       "scan-issues",
		"timestamp":     "2026-02-03T19:47:59+01:00",
		"tokens_used":   378,
	}

	fmt.Printf("📝 AI Output (CORRECT):\n%s\n\n", string(validJSON))

	// Demo 1: What happens WITHOUT the fix
	fmt.Println("❌ BEFORE FIX - Validating error wrapper structure:")
	fmt.Println("   Wave validates the wrapper instead of the AI output")

	wrapperBytes, _ := json.Marshal(errorWrapper)
	fmt.Printf("   Wrapper structure: %s...\n", string(wrapperBytes)[:100])

	// Write wrapper to artifact file
	artifactPath := filepath.Join(tempDir, "artifact.json")
	os.WriteFile(artifactPath, wrapperBytes, 0644)

	// Validate with wrapper detection DISABLED (simulating old behavior)
	cfg := ContractConfig{
		Type:                    "json_schema",
		Schema:                  string(schemaBytes),
		DisableWrapperDetection: true, // OLD BEHAVIOR
	}

	validator := &jsonSchemaValidator{}
	err = validator.Validate(cfg, tempDir)
	if err != nil {
		fmt.Printf("   ❌ VALIDATION FAILED: %s\n\n", err.Error()[:100]+"...")
	} else {
		fmt.Println("   ✅ Unexpectedly passed (this shouldn't happen)")
	}

	// Demo 2: What happens WITH the fix
	fmt.Println("✅ AFTER FIX - Wrapper detection enabled:")
	fmt.Println("   Wave detects wrapper, extracts AI output, validates the correct content")

	// Same artifact, but with wrapper detection ENABLED
	cfg.DisableWrapperDetection = false // NEW BEHAVIOR
	cfg.DebugMode = true

	err = validator.Validate(cfg, tempDir)
	if err != nil {
		fmt.Printf("   ❌ VALIDATION FAILED: %s\n", err.Error())
	} else {
		fmt.Println("   ✅ VALIDATION PASSED: AI output correctly extracted and validated!")
	}

	fmt.Println("\n🎯 Result: The fix resolves the false failures by validating AI output instead of error metadata!")
	return nil
}

// DemoWrapper shows how wrapper detection works
func DemoWrapper() error {
	fmt.Println("\n🔍 Wrapper Detection Demo")
	fmt.Println("=========================")

	// Test different input types
	testCases := []struct {
		name  string
		input interface{}
		desc  string
	}{
		{
			name: "Direct JSON",
			input: map[string]interface{}{
				"test": "value",
				"data": []string{"item1", "item2"},
			},
			desc: "Normal JSON output should NOT be detected as wrapper",
		},
		{
			name: "Error Wrapper",
			input: map[string]interface{}{
				"error_type":    "format_failure",
				"raw_output":    `{"test": "value", "data": ["item1", "item2"]}`,
				"contract_type": "json_schema",
				"step_id":       "demo-step",
				"attempts":      3,
			},
			desc: "Error wrapper should be detected and raw content extracted",
		},
		{
			name: "Incomplete Wrapper",
			input: map[string]interface{}{
				"error_type": "format_failure",
				// Missing raw_output - key field missing
				"contract_type": "json_schema",
			},
			desc: "Incomplete wrapper should NOT be detected (missing raw_output)",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("Test: %s\n", tc.name)
		fmt.Printf("Description: %s\n", tc.desc)

		inputBytes, _ := json.Marshal(tc.input)
		result, err := DetectErrorWrapper(inputBytes)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("Detected as wrapper: %v\n", result.IsWrapper)
		fmt.Printf("Confidence: %s\n", result.Confidence)
		if result.IsWrapper {
			fmt.Printf("Extracted content: %s\n", string(result.RawContent))
			fmt.Printf("Fields matched: %v\n", result.FieldsMatched)
		}
		fmt.Println()
	}

	return nil
}