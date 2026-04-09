package contract

import (
	"strings"
	"testing"
)

func TestFalsePositive_JSONSchemaEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		schema        string
		artifact      string
		expectError   bool
		errorContains string
	}{
		{
			name:          "truncated JSON",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			artifact:      `{"name": "te`,
			expectError:   true,
			errorContains: "failed to parse artifact JSON",
		},
		{
			name:          "string masquerading as integer",
			schema:        `{"type": "object", "properties": {"age": {"type": "integer"}}, "required": ["age"]}`,
			artifact:      `{"age": "123"}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:          "string masquerading as boolean",
			schema:        `{"type": "object", "properties": {"active": {"type": "boolean"}}, "required": ["active"]}`,
			artifact:      `{"active": "true"}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:          "object where array expected",
			schema:        `{"type": "object", "properties": {"items": {"type": "array"}}, "required": ["items"]}`,
			artifact:      `{"items": {"key": "value"}}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:          "empty object with required fields",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "integer"}}, "required": ["name", "age"]}`,
			artifact:      `{}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:          "null value for required string",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			artifact:      `{"name": null}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:          "extra fields with additionalProperties false",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}, "additionalProperties": false}`,
			artifact:      `{"name": "test", "unexpected": "field"}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:          "deeply nested type mismatch",
			schema:        `{"type": "object", "properties": {"user": {"type": "object", "properties": {"profile": {"type": "object", "properties": {"age": {"type": "integer"}}}}}}}`,
			artifact:      `{"user": {"profile": {"age": "not-a-number"}}}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:          "empty string for required string field passes",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			artifact:      `{"name": ""}`,
			expectError:   false,
			errorContains: "",
		},
		{
			name:          "valid object positive control",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "integer"}}, "required": ["name", "age"]}`,
			artifact:      `{"name": "Alice", "age": 30}`,
			expectError:   false,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &jsonSchemaValidator{}
			cfg := ContractConfig{
				Type:   "json_schema",
				Schema: tt.schema,
			}

			workspacePath := t.TempDir()
			writeTestArtifact(t, workspacePath, []byte(tt.artifact))

			err := v.Validate(cfg, workspacePath)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error should contain %q, got: %v", tt.errorContains, err)
				}
			} else if err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}
