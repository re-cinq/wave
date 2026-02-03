package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSpecPhaseContractValidation(t *testing.T) {
	// Test spec phase contract schema validation
	tests := []struct {
		name     string
		output   map[string]interface{}
		expected bool // true if should validate successfully
	}{
		{
			name: "valid spec phase output",
			output: map[string]interface{}{
				"phase": "spec",
				"artifacts": map[string]interface{}{
					"spec": map[string]interface{}{
						"path":         "spec.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"completeness_score": 85,
					"clarity_score":      90,
					"testability_score":  80,
					"specification_quality": "good",
				},
				"metadata": map[string]interface{}{
					"timestamp":       "2026-02-03T10:30:00Z",
					"duration_seconds": 120.5,
					"input_description": "Build a task management CLI tool",
				},
			},
			expected: true,
		},
		{
			name: "missing required phase field",
			output: map[string]interface{}{
				"artifacts": map[string]interface{}{
					"spec": map[string]interface{}{
						"path":         "spec.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
			},
			expected: false,
		},
		{
			name: "invalid phase value",
			output: map[string]interface{}{
				"phase": "invalid",
				"artifacts": map[string]interface{}{
					"spec": map[string]interface{}{
						"path":         "spec.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"specification_quality": "good",
				},
			},
			expected: false,
		},
		{
			name: "missing spec artifact",
			output: map[string]interface{}{
				"phase": "spec",
				"artifacts": map[string]interface{}{
					"other": map[string]interface{}{
						"path":         "other.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"specification_quality": "good",
				},
			},
			expected: false,
		},
		{
			name: "spec artifact file missing",
			output: map[string]interface{}{
				"phase": "spec",
				"artifacts": map[string]interface{}{
					"spec": map[string]interface{}{
						"path":         "spec.md",
						"exists":       false,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"specification_quality": "good",
				},
			},
			expected: false,
		},
		{
			name: "invalid specification quality",
			output: map[string]interface{}{
				"phase": "spec",
				"artifacts": map[string]interface{}{
					"spec": map[string]interface{}{
						"path":         "spec.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"specification_quality": "invalid_quality",
				},
			},
			expected: false,
		},
		{
			name: "missing metadata",
			output: map[string]interface{}{
				"phase": "spec",
				"artifacts": map[string]interface{}{
					"spec": map[string]interface{}{
						"path":         "spec.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"specification_quality": "good",
				},
				// metadata missing
			},
			expected: false,
		},
	}

	// Load spec phase contract schema
	schemaPath := ".wave/contracts/spec-phase.schema.json"
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to load spec phase schema: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary output file
			tempDir := t.TempDir()
			outputPath := filepath.Join(tempDir, "output.json")

			outputData, err := json.MarshalIndent(tt.output, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal output data: %v", err)
			}

			err = os.WriteFile(outputPath, outputData, 0644)
			if err != nil {
				t.Fatalf("Failed to write output file: %v", err)
			}

			// Test contract validation
			config := ContractConfig{
				Type:       "json_schema",
				Schema:     string(schemaData),
				SchemaPath: schemaPath,
				StrictMode: true,
				MustPass:   true,
			}

			err = Validate(config, tempDir)

			if tt.expected && err != nil {
				t.Errorf("Expected validation to pass, but got error: %v", err)
			}

			if !tt.expected && err == nil {
				t.Error("Expected validation to fail, but it passed")
			}
		})
	}
}

func TestSpecPhaseSchemaExists(t *testing.T) {
	// Test that the spec phase schema file exists and is valid JSON
	schemaPath := ".wave/contracts/spec-phase.schema.json"

	// Check file exists
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		t.Fatalf("Spec phase schema file does not exist: %s", schemaPath)
	}

	// Check file is valid JSON
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read spec phase schema: %v", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		t.Fatalf("Spec phase schema is not valid JSON: %v", err)
	}

	// Check required schema fields
	requiredFields := []string{"$schema", "title", "type", "properties", "required"}
	for _, field := range requiredFields {
		if _, exists := schema[field]; !exists {
			t.Errorf("Spec phase schema missing required field: %s", field)
		}
	}

	// Check that it defines validation for required artifacts
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema properties field is not an object")
	}

	artifacts, ok := properties["artifacts"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema does not define artifacts property")
	}

	artifactProps, ok := artifacts["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Artifacts property does not have properties defined")
	}

	if _, exists := artifactProps["spec"]; !exists {
		t.Error("Schema does not validate spec artifact")
	}
}

func TestSpecPhaseContractIntegration(t *testing.T) {
	// Integration test: create a minimal spec phase output and validate it
	tempDir := t.TempDir()

	// Create a minimal spec.md file
	specContent := `# Feature Specification: Test Feature

## Overview
This is a test feature specification.

## User Stories
- As a user, I want to test the spec phase
- As a developer, I want to validate the pipeline works

## Functional Requirements
1. The spec phase must generate this file
2. The file must be validated by the contract

## Success Criteria
- Specification file is created
- Contract validation passes
`

	specPath := filepath.Join(tempDir, "spec.md")
	err := os.WriteFile(specPath, []byte(specContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create spec file: %v", err)
	}

	// Create spec phase output JSON
	output := map[string]interface{}{
		"phase": "spec",
		"artifacts": map[string]interface{}{
			"spec": map[string]interface{}{
				"path":         "spec.md",
				"exists":       true,
				"content_type": "markdown",
			},
		},
		"validation": map[string]interface{}{
			"completeness_score":    95,
			"clarity_score":        90,
			"testability_score":    85,
			"specification_quality": "excellent",
		},
		"metadata": map[string]interface{}{
			"timestamp":         "2026-02-03T10:30:00Z",
			"duration_seconds":  45.2,
			"input_description": "Test feature for prototype pipeline",
		},
	}

	outputData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal output: %v", err)
	}

	outputPath := filepath.Join(tempDir, "output.json")
	err = os.WriteFile(outputPath, outputData, 0644)
	if err != nil {
		t.Fatalf("Failed to write output file: %v", err)
	}

	// Validate using contract
	config := ContractConfig{
		Type:       "json_schema",
		SchemaPath: ".wave/contracts/spec-phase.schema.json",
		StrictMode: true,
		MustPass:   true,
	}

	err = Validate(config, tempDir)
	if err != nil {
		t.Errorf("Contract validation failed for valid spec phase output: %v", err)
	}
}