package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDocsPhaseContractValidation(t *testing.T) {
	// Test docs phase contract schema validation
	tests := []struct {
		name     string
		output   map[string]interface{}
		expected bool // true if should validate successfully
	}{
		{
			name: "valid docs phase output",
			output: map[string]interface{}{
				"phase": "docs",
				"artifacts": map[string]interface{}{
					"feature_docs": map[string]interface{}{
						"path":         "feature-docs.md",
						"exists":       true,
						"content_type": "markdown",
					},
					"stakeholder_summary": map[string]interface{}{
						"path":         "stakeholder-summary.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"coverage_percentage":   95,
					"readability_score":     90,
					"documentation_quality": "excellent",
				},
				"metadata": map[string]interface{}{
					"timestamp":         "2026-02-03T11:00:00Z",
					"duration_seconds":  180.0,
					"source_spec_path":  "artifacts/input-spec.md",
				},
			},
			expected: true,
		},
		{
			name: "missing required phase field",
			output: map[string]interface{}{
				"artifacts": map[string]interface{}{
					"feature_docs": map[string]interface{}{
						"path":         "feature-docs.md",
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
					"feature_docs": map[string]interface{}{
						"path":         "feature-docs.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"documentation_quality": "good",
				},
			},
			expected: false,
		},
		{
			name: "missing required feature_docs artifact",
			output: map[string]interface{}{
				"phase": "docs",
				"artifacts": map[string]interface{}{
					"stakeholder_summary": map[string]interface{}{
						"path":         "stakeholder-summary.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"documentation_quality": "good",
				},
			},
			expected: false,
		},
		{
			name: "missing stakeholder_summary artifact",
			output: map[string]interface{}{
				"phase": "docs",
				"artifacts": map[string]interface{}{
					"feature_docs": map[string]interface{}{
						"path":         "feature-docs.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"documentation_quality": "good",
				},
			},
			expected: false,
		},
		{
			name: "feature_docs file missing",
			output: map[string]interface{}{
				"phase": "docs",
				"artifacts": map[string]interface{}{
					"feature_docs": map[string]interface{}{
						"path":         "feature-docs.md",
						"exists":       false,
						"content_type": "markdown",
					},
					"stakeholder_summary": map[string]interface{}{
						"path":         "stakeholder-summary.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"documentation_quality": "good",
				},
			},
			expected: false,
		},
		{
			name: "invalid documentation quality",
			output: map[string]interface{}{
				"phase": "docs",
				"artifacts": map[string]interface{}{
					"feature_docs": map[string]interface{}{
						"path":         "feature-docs.md",
						"exists":       true,
						"content_type": "markdown",
					},
					"stakeholder_summary": map[string]interface{}{
						"path":         "stakeholder-summary.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"documentation_quality": "invalid_quality",
				},
			},
			expected: false,
		},
		{
			name: "missing metadata",
			output: map[string]interface{}{
				"phase": "docs",
				"artifacts": map[string]interface{}{
					"feature_docs": map[string]interface{}{
						"path":         "feature-docs.md",
						"exists":       true,
						"content_type": "markdown",
					},
					"stakeholder_summary": map[string]interface{}{
						"path":         "stakeholder-summary.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"documentation_quality": "good",
				},
				// metadata missing
			},
			expected: false,
		},
		{
			name: "coverage_percentage out of range",
			output: map[string]interface{}{
				"phase": "docs",
				"artifacts": map[string]interface{}{
					"feature_docs": map[string]interface{}{
						"path":         "feature-docs.md",
						"exists":       true,
						"content_type": "markdown",
					},
					"stakeholder_summary": map[string]interface{}{
						"path":         "stakeholder-summary.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"coverage_percentage":   150, // Invalid: > 100
					"documentation_quality": "good",
				},
				"metadata": map[string]interface{}{
					"timestamp":        "2026-02-03T11:00:00Z",
					"source_spec_path": "artifacts/input-spec.md",
				},
			},
			expected: false,
		},
	}

	// Load docs phase contract schema
	schemaPath := ".wave/contracts/docs-phase.schema.json"
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to load docs phase schema: %v", err)
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

func TestDocsPhaseSchemaExists(t *testing.T) {
	// Test that the docs phase schema file exists and is valid JSON
	schemaPath := ".wave/contracts/docs-phase.schema.json"

	// Check file exists
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		t.Fatalf("Docs phase schema file does not exist: %s", schemaPath)
	}

	// Check file is valid JSON
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read docs phase schema: %v", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		t.Fatalf("Docs phase schema is not valid JSON: %v", err)
	}

	// Check required schema fields
	requiredFields := []string{"$schema", "title", "type", "properties", "required"}
	for _, field := range requiredFields {
		if _, exists := schema[field]; !exists {
			t.Errorf("Docs phase schema missing required field: %s", field)
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

	if _, exists := artifactProps["feature_docs"]; !exists {
		t.Error("Schema does not validate feature_docs artifact")
	}

	if _, exists := artifactProps["stakeholder_summary"]; !exists {
		t.Error("Schema does not validate stakeholder_summary artifact")
	}
}

func TestDocsPhaseContractIntegration(t *testing.T) {
	// Integration test: create a minimal docs phase output and validate it
	tempDir := t.TempDir()

	// Create feature documentation file
	featureDocsContent := `# Feature Documentation: Test Feature

## Overview
This is comprehensive documentation for the test feature that was specified in the previous phase.

## User Guide
Users can interact with this feature through the following interfaces:

### Getting Started
1. Initialize the application
2. Configure your settings
3. Start using the feature

### Usage Examples
\`\`\`
example command --option value
\`\`\`

## Developer Integration Guide
Developers can integrate this feature by following these steps:

1. Import the required modules
2. Configure the feature settings
3. Implement the interface handlers

## API Reference
The feature provides the following API endpoints:

- GET /api/feature - Retrieve feature status
- POST /api/feature - Create new feature instance
- PUT /api/feature/{id} - Update existing feature
`

	featureDocsPath := filepath.Join(tempDir, "feature-docs.md")
	err := os.WriteFile(featureDocsPath, []byte(featureDocsContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create feature docs file: %v", err)
	}

	// Create stakeholder summary file
	stakeholderSummaryContent := `# Stakeholder Summary: Test Feature

## Executive Summary
The test feature provides significant business value by enabling users to perform key tasks efficiently.

## Business Impact
- Improves user productivity by 30%
- Reduces operational overhead
- Enhances user satisfaction

## Timeline
- Development: 2 weeks
- Testing: 1 week
- Deployment: 3 days

## Success Metrics
- User adoption rate: > 80%
- Performance improvement: > 25%
- User satisfaction score: > 4.5/5

## Next Steps
1. Begin implementation phase
2. Coordinate with QA team
3. Prepare deployment strategy
`

	stakeholderSummaryPath := filepath.Join(tempDir, "stakeholder-summary.md")
	err = os.WriteFile(stakeholderSummaryPath, []byte(stakeholderSummaryContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create stakeholder summary file: %v", err)
	}

	// Create docs phase output JSON
	output := map[string]interface{}{
		"phase": "docs",
		"artifacts": map[string]interface{}{
			"feature_docs": map[string]interface{}{
				"path":         "feature-docs.md",
				"exists":       true,
				"content_type": "markdown",
			},
			"stakeholder_summary": map[string]interface{}{
				"path":         "stakeholder-summary.md",
				"exists":       true,
				"content_type": "markdown",
			},
		},
		"validation": map[string]interface{}{
			"coverage_percentage":   100,
			"readability_score":     95,
			"documentation_quality": "excellent",
		},
		"metadata": map[string]interface{}{
			"timestamp":        "2026-02-03T11:00:00Z",
			"duration_seconds": 220.5,
			"source_spec_path": "artifacts/input-spec.md",
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
		SchemaPath: ".wave/contracts/docs-phase.schema.json",
		StrictMode: true,
		MustPass:   true,
	}

	err = Validate(config, tempDir)
	if err != nil {
		t.Errorf("Contract validation failed for valid docs phase output: %v", err)
	}
}