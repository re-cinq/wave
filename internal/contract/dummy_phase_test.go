package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDummyPhaseContractValidation(t *testing.T) {
	// Test dummy phase contract schema validation
	tests := []struct {
		name     string
		output   map[string]interface{}
		expected bool // true if should validate successfully
	}{
		{
			name: "valid dummy phase output",
			output: map[string]interface{}{
				"phase": "dummy",
				"artifacts": map[string]interface{}{
					"prototype": map[string]interface{}{
						"path":         "prototype/",
						"exists":       true,
						"content_type": "code",
					},
					"interface_definitions": map[string]interface{}{
						"path":         "interfaces.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"runnable":              true,
					"interface_completeness": 100,
					"prototype_quality":     "excellent",
				},
				"metadata": map[string]interface{}{
					"timestamp":         "2026-02-03T12:00:00Z",
					"duration_seconds":  300.0,
					"source_docs_path":  "artifacts/feature-docs.md",
				},
			},
			expected: true,
		},
		{
			name: "missing required phase field",
			output: map[string]interface{}{
				"artifacts": map[string]interface{}{
					"prototype": map[string]interface{}{
						"path":         "prototype/",
						"exists":       true,
						"content_type": "code",
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
					"prototype": map[string]interface{}{
						"path":         "prototype/",
						"exists":       true,
						"content_type": "code",
					},
				},
				"validation": map[string]interface{}{
					"runnable":          true,
					"prototype_quality": "good",
				},
			},
			expected: false,
		},
		{
			name: "missing required prototype artifact",
			output: map[string]interface{}{
				"phase": "dummy",
				"artifacts": map[string]interface{}{
					"interface_definitions": map[string]interface{}{
						"path":         "interfaces.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"runnable":          true,
					"prototype_quality": "good",
				},
			},
			expected: false,
		},
		{
			name: "missing interface_definitions artifact",
			output: map[string]interface{}{
				"phase": "dummy",
				"artifacts": map[string]interface{}{
					"prototype": map[string]interface{}{
						"path":         "prototype/",
						"exists":       true,
						"content_type": "code",
					},
				},
				"validation": map[string]interface{}{
					"runnable":          true,
					"prototype_quality": "good",
				},
			},
			expected: false,
		},
		{
			name: "prototype directory missing",
			output: map[string]interface{}{
				"phase": "dummy",
				"artifacts": map[string]interface{}{
					"prototype": map[string]interface{}{
						"path":         "prototype/",
						"exists":       false,
						"content_type": "code",
					},
					"interface_definitions": map[string]interface{}{
						"path":         "interfaces.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"runnable":          true,
					"prototype_quality": "good",
				},
			},
			expected: false,
		},
		{
			name: "invalid prototype quality",
			output: map[string]interface{}{
				"phase": "dummy",
				"artifacts": map[string]interface{}{
					"prototype": map[string]interface{}{
						"path":         "prototype/",
						"exists":       true,
						"content_type": "code",
					},
					"interface_definitions": map[string]interface{}{
						"path":         "interfaces.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"runnable":          true,
					"prototype_quality": "invalid_quality",
				},
			},
			expected: false,
		},
		{
			name: "missing validation",
			output: map[string]interface{}{
				"phase": "dummy",
				"artifacts": map[string]interface{}{
					"prototype": map[string]interface{}{
						"path":         "prototype/",
						"exists":       true,
						"content_type": "code",
					},
					"interface_definitions": map[string]interface{}{
						"path":         "interfaces.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				// validation missing
				"metadata": map[string]interface{}{
					"timestamp":        "2026-02-03T12:00:00Z",
					"source_docs_path": "artifacts/feature-docs.md",
				},
			},
			expected: false,
		},
		{
			name: "interface_completeness out of range",
			output: map[string]interface{}{
				"phase": "dummy",
				"artifacts": map[string]interface{}{
					"prototype": map[string]interface{}{
						"path":         "prototype/",
						"exists":       true,
						"content_type": "code",
					},
					"interface_definitions": map[string]interface{}{
						"path":         "interfaces.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"runnable":              true,
					"interface_completeness": 150, // Invalid: > 100
					"prototype_quality":     "good",
				},
				"metadata": map[string]interface{}{
					"timestamp":        "2026-02-03T12:00:00Z",
					"source_docs_path": "artifacts/feature-docs.md",
				},
			},
			expected: false,
		},
		{
			name: "non-runnable prototype is valid",
			output: map[string]interface{}{
				"phase": "dummy",
				"artifacts": map[string]interface{}{
					"prototype": map[string]interface{}{
						"path":         "prototype/",
						"exists":       true,
						"content_type": "code",
					},
					"interface_definitions": map[string]interface{}{
						"path":         "interfaces.md",
						"exists":       true,
						"content_type": "markdown",
					},
				},
				"validation": map[string]interface{}{
					"runnable":              false, // Non-runnable but still valid
					"interface_completeness": 75,
					"prototype_quality":     "fair",
				},
				"metadata": map[string]interface{}{
					"timestamp":        "2026-02-03T12:00:00Z",
					"source_docs_path": "artifacts/feature-docs.md",
				},
			},
			expected: true,
		},
	}

	// Load dummy phase contract schema
	schemaPath := ".wave/contracts/dummy-phase.schema.json"
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to load dummy phase schema: %v", err)
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

func TestDummyPhaseSchemaExists(t *testing.T) {
	// Test that the dummy phase schema file exists and is valid JSON
	schemaPath := ".wave/contracts/dummy-phase.schema.json"

	// Check file exists
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		t.Fatalf("Dummy phase schema file does not exist: %s", schemaPath)
	}

	// Check file is valid JSON
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read dummy phase schema: %v", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		t.Fatalf("Dummy phase schema is not valid JSON: %v", err)
	}

	// Check required schema fields
	requiredFields := []string{"$schema", "title", "type", "properties", "required"}
	for _, field := range requiredFields {
		if _, exists := schema[field]; !exists {
			t.Errorf("Dummy phase schema missing required field: %s", field)
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

	if _, exists := artifactProps["prototype"]; !exists {
		t.Error("Schema does not validate prototype artifact")
	}

	if _, exists := artifactProps["interface_definitions"]; !exists {
		t.Error("Schema does not validate interface_definitions artifact")
	}
}

func TestDummyPhaseContractIntegration(t *testing.T) {
	// Integration test: create a minimal dummy phase output and validate it
	tempDir := t.TempDir()

	// Create prototype directory with sample code
	prototypeDir := filepath.Join(tempDir, "prototype")
	err := os.MkdirAll(prototypeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create prototype directory: %v", err)
	}

	// Create a simple prototype file
	prototypeCode := `#!/usr/bin/env python3
"""
Test Feature Prototype
======================

This is a working prototype that demonstrates the key interfaces and user flows
for the test feature with stub implementations.
"""

import sys
import json
import argparse

class TestFeature:
    """Main feature class with stub implementations."""

    def __init__(self):
        self.data = {}

    def create_item(self, name, description=""):
        """Create a new item (stub implementation)."""
        item_id = len(self.data) + 1
        self.data[item_id] = {
            "id": item_id,
            "name": name,
            "description": description,
            "status": "active"
        }
        return item_id

    def list_items(self):
        """List all items (stub implementation)."""
        return list(self.data.values())

    def get_item(self, item_id):
        """Get item by ID (stub implementation)."""
        return self.data.get(item_id)

    def delete_item(self, item_id):
        """Delete item by ID (stub implementation)."""
        if item_id in self.data:
            del self.data[item_id]
            return True
        return False

def main():
    """Main entry point demonstrating the interface."""
    parser = argparse.ArgumentParser(description="Test Feature Prototype")
    parser.add_argument("--action", required=True,
                       choices=["create", "list", "get", "delete"],
                       help="Action to perform")
    parser.add_argument("--name", help="Item name for create action")
    parser.add_argument("--id", type=int, help="Item ID for get/delete actions")
    parser.add_argument("--description", help="Item description for create action")

    args = parser.parse_args()

    feature = TestFeature()

    if args.action == "create":
        if not args.name:
            print("Error: --name required for create action")
            sys.exit(1)
        item_id = feature.create_item(args.name, args.description or "")
        print(f"Created item {item_id}: {args.name}")

    elif args.action == "list":
        items = feature.list_items()
        print(json.dumps(items, indent=2))

    elif args.action == "get":
        if args.id is None:
            print("Error: --id required for get action")
            sys.exit(1)
        item = feature.get_item(args.id)
        if item:
            print(json.dumps(item, indent=2))
        else:
            print(f"Item {args.id} not found")

    elif args.action == "delete":
        if args.id is None:
            print("Error: --id required for delete action")
            sys.exit(1)
        if feature.delete_item(args.id):
            print(f"Deleted item {args.id}")
        else:
            print(f"Item {args.id} not found")

if __name__ == "__main__":
    main()
`

	prototypeMainPath := filepath.Join(prototypeDir, "main.py")
	err = os.WriteFile(prototypeMainPath, []byte(prototypeCode), 0755)
	if err != nil {
		t.Fatalf("Failed to create prototype main file: %v", err)
	}

	// Create interface definitions file
	interfaceDefsContent := `# Interface Definitions: Test Feature Prototype

## Command Line Interface

The prototype provides a command-line interface with the following operations:

### Create Item
\`\`\`bash
python main.py --action create --name "Item Name" [--description "Description"]
\`\`\`

### List Items
\`\`\`bash
python main.py --action list
\`\`\`

### Get Item
\`\`\`bash
python main.py --action get --id <item_id>
\`\`\`

### Delete Item
\`\`\`bash
python main.py --action delete --id <item_id>
\`\`\`

## Data Structures

### Item Object
\`\`\`json
{
  "id": 1,
  "name": "Item Name",
  "description": "Item Description",
  "status": "active"
}
\`\`\`

## Stub Implementation Notes

- Data is stored in memory (not persisted)
- No authentication or authorization
- No input validation beyond argument parsing
- No error handling for edge cases
- Uses simple sequential ID generation

## Future Implementation Areas

1. **Persistence**: Replace in-memory storage with database
2. **Validation**: Add proper input validation and sanitization
3. **Authentication**: Implement user authentication system
4. **API**: Add REST API endpoints
5. **Error Handling**: Comprehensive error handling and logging
6. **Configuration**: External configuration file support
`

	interfaceDefsPath := filepath.Join(tempDir, "interfaces.md")
	err = os.WriteFile(interfaceDefsPath, []byte(interfaceDefsContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create interface definitions file: %v", err)
	}

	// Create dummy phase output JSON
	output := map[string]interface{}{
		"phase": "dummy",
		"artifacts": map[string]interface{}{
			"prototype": map[string]interface{}{
				"path":         "prototype/",
				"exists":       true,
				"content_type": "code",
			},
			"interface_definitions": map[string]interface{}{
				"path":         "interfaces.md",
				"exists":       true,
				"content_type": "markdown",
			},
		},
		"validation": map[string]interface{}{
			"runnable":              true,
			"interface_completeness": 95,
			"prototype_quality":     "excellent",
		},
		"metadata": map[string]interface{}{
			"timestamp":        "2026-02-03T12:00:00Z",
			"duration_seconds": 325.0,
			"source_docs_path": "artifacts/feature-docs.md",
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
		SchemaPath: ".wave/contracts/dummy-phase.schema.json",
		StrictMode: true,
		MustPass:   true,
	}

	err = Validate(config, tempDir)
	if err != nil {
		t.Errorf("Contract validation failed for valid dummy phase output: %v", err)
	}
}