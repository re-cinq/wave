package contract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// T004: Test that when a schema expects a string, providing an integer fails
func TestJSONSchemaValidator_TypeMismatch_StringReceivesInteger(t *testing.T) {
	tests := []struct {
		name          string
		schema        string
		artifact      string
		expectError   bool
		errorContains []string
	}{
		{
			name:          "string field receives integer - simple",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			artifact:      `{"name": 123}`,
			expectError:   true,
			errorContains: []string{"name"},
		},
		{
			name:          "string field receives integer - nested object",
			schema:        `{"type": "object", "properties": {"user": {"type": "object", "properties": {"email": {"type": "string"}}, "required": ["email"]}}}`,
			artifact:      `{"user": {"email": 456}}`,
			expectError:   true,
			errorContains: []string{"email"},
		},
		{
			name:          "string field receives boolean",
			schema:        `{"type": "object", "properties": {"enabled": {"type": "string"}}, "required": ["enabled"]}`,
			artifact:      `{"enabled": true}`,
			expectError:   true,
			errorContains: []string{"enabled"},
		},
		{
			name:          "string field receives null",
			schema:        `{"type": "object", "properties": {"value": {"type": "string"}}, "required": ["value"]}`,
			artifact:      `{"value": null}`,
			expectError:   true,
			errorContains: []string{"value"},
		},
		{
			name:          "string field receives array",
			schema:        `{"type": "object", "properties": {"title": {"type": "string"}}, "required": ["title"]}`,
			artifact:      `{"title": ["a", "b"]}`,
			expectError:   true,
			errorContains: []string{"title"},
		},
		{
			name:          "valid string value passes",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			artifact:      `{"name": "valid string"}`,
			expectError:   false,
			errorContains: nil,
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
			writeTestArtifactForJSONSchema(t, workspacePath, []byte(tt.artifact))

			err := v.Validate(cfg, workspacePath)

			if tt.expectError {
				if err == nil {
					t.Error("expected validation error for type mismatch, got none")
					return
				}

				// Verify it's a ValidationError
				validErr, ok := err.(*ValidationError)
				if !ok {
					t.Errorf("expected *ValidationError, got %T", err)
					return
				}

				// Verify contract type
				if validErr.ContractType != "json_schema" {
					t.Errorf("expected contract type json_schema, got %s", validErr.ContractType)
				}

				// Verify error details mention the specific field
				errStr := validErr.Error()
				detailStr := strings.Join(validErr.Details, " ")
				combinedOutput := errStr + " " + detailStr

				for _, expected := range tt.errorContains {
					if !strings.Contains(strings.ToLower(combinedOutput), strings.ToLower(expected)) {
						t.Errorf("expected error to mention %q, got: %s", expected, combinedOutput)
					}
				}

				// Verify there are details present
				if len(validErr.Details) == 0 {
					t.Error("expected ValidationError.Details to contain information about the type mismatch")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for valid input, got: %v", err)
				}
			}
		})
	}
}

// T005: Test that when a schema expects an integer, providing a string fails
func TestJSONSchemaValidator_TypeMismatch_IntegerReceivesString(t *testing.T) {
	tests := []struct {
		name          string
		schema        string
		artifact      string
		expectError   bool
		errorContains []string
	}{
		{
			name:          "integer field receives string",
			schema:        `{"type": "object", "properties": {"age": {"type": "integer"}}, "required": ["age"]}`,
			artifact:      `{"age": "thirty"}`,
			expectError:   true,
			errorContains: []string{"age"},
		},
		{
			name:          "integer field receives numeric string",
			schema:        `{"type": "object", "properties": {"count": {"type": "integer"}}, "required": ["count"]}`,
			artifact:      `{"count": "42"}`,
			expectError:   true,
			errorContains: []string{"count"},
		},
		{
			name:          "number field receives string",
			schema:        `{"type": "object", "properties": {"price": {"type": "number"}}, "required": ["price"]}`,
			artifact:      `{"price": "19.99"}`,
			expectError:   true,
			errorContains: []string{"price"},
		},
		{
			name:          "integer field receives object",
			schema:        `{"type": "object", "properties": {"id": {"type": "integer"}}, "required": ["id"]}`,
			artifact:      `{"id": {"value": 1}}`,
			expectError:   true,
			errorContains: []string{"id"},
		},
		{
			name:          "nested integer field receives string",
			schema:        `{"type": "object", "properties": {"config": {"type": "object", "properties": {"timeout": {"type": "integer"}}, "required": ["timeout"]}}}`,
			artifact:      `{"config": {"timeout": "5000"}}`,
			expectError:   true,
			errorContains: []string{"timeout"},
		},
		{
			name:          "valid integer value passes",
			schema:        `{"type": "object", "properties": {"age": {"type": "integer"}}, "required": ["age"]}`,
			artifact:      `{"age": 30}`,
			expectError:   false,
			errorContains: nil,
		},
		{
			name:          "valid number value passes",
			schema:        `{"type": "object", "properties": {"price": {"type": "number"}}, "required": ["price"]}`,
			artifact:      `{"price": 19.99}`,
			expectError:   false,
			errorContains: nil,
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
			writeTestArtifactForJSONSchema(t, workspacePath, []byte(tt.artifact))

			err := v.Validate(cfg, workspacePath)

			if tt.expectError {
				if err == nil {
					t.Error("expected validation error for type mismatch, got none")
					return
				}

				// Verify it's a ValidationError
				validErr, ok := err.(*ValidationError)
				if !ok {
					t.Errorf("expected *ValidationError, got %T", err)
					return
				}

				// Verify contract type
				if validErr.ContractType != "json_schema" {
					t.Errorf("expected contract type json_schema, got %s", validErr.ContractType)
				}

				// Verify error details mention the specific field
				errStr := validErr.Error()
				detailStr := strings.Join(validErr.Details, " ")
				combinedOutput := errStr + " " + detailStr

				for _, expected := range tt.errorContains {
					if !strings.Contains(strings.ToLower(combinedOutput), strings.ToLower(expected)) {
						t.Errorf("expected error to mention %q, got: %s", expected, combinedOutput)
					}
				}

				// Verify there are details present
				if len(validErr.Details) == 0 {
					t.Error("expected ValidationError.Details to contain information about the type mismatch")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for valid input, got: %v", err)
				}
			}
		})
	}
}

// T007: Test that extra fields are rejected when additionalProperties: false
func TestJSONSchemaValidator_AdditionalPropertiesFalse(t *testing.T) {
	tests := []struct {
		name          string
		schema        string
		artifact      string
		expectError   bool
		errorContains []string
	}{
		{
			name:          "extra field rejected",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}, "additionalProperties": false}`,
			artifact:      `{"name": "test", "extra": "field"}`,
			expectError:   true,
			errorContains: []string{"extra", "additional"},
		},
		{
			name:          "multiple extra fields rejected",
			schema:        `{"type": "object", "properties": {"id": {"type": "integer"}}, "additionalProperties": false}`,
			artifact:      `{"id": 1, "foo": "bar", "baz": 123}`,
			expectError:   true,
			errorContains: []string{"additional"},
		},
		{
			name:          "extra field in nested object rejected",
			schema:        `{"type": "object", "properties": {"config": {"type": "object", "properties": {"value": {"type": "string"}}, "additionalProperties": false}}}`,
			artifact:      `{"config": {"value": "test", "unwanted": true}}`,
			expectError:   true,
			errorContains: []string{"unwanted", "additional"},
		},
		{
			name:          "empty extra object rejected",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}, "additionalProperties": false}`,
			artifact:      `{"name": "test", "metadata": {}}`,
			expectError:   true,
			errorContains: []string{"metadata", "additional"},
		},
		{
			name:          "valid object without extra fields passes",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "integer"}}, "additionalProperties": false}`,
			artifact:      `{"name": "Alice", "age": 30}`,
			expectError:   false,
			errorContains: nil,
		},
		{
			name:          "partial properties valid",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "integer"}}, "additionalProperties": false}`,
			artifact:      `{"name": "Bob"}`,
			expectError:   false,
			errorContains: nil,
		},
		{
			name:          "additionalProperties true allows extra fields",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}, "additionalProperties": true}`,
			artifact:      `{"name": "test", "extra": "allowed"}`,
			expectError:   false,
			errorContains: nil,
		},
		{
			name:          "additionalProperties with schema validates extra fields",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}, "additionalProperties": {"type": "string"}}`,
			artifact:      `{"name": "test", "extra": 123}`,
			expectError:   true,
			errorContains: []string{"extra"},
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
			writeTestArtifactForJSONSchema(t, workspacePath, []byte(tt.artifact))

			err := v.Validate(cfg, workspacePath)

			if tt.expectError {
				if err == nil {
					t.Error("expected validation error for additionalProperties violation, got none")
					return
				}

				// Verify it's a ValidationError
				validErr, ok := err.(*ValidationError)
				if !ok {
					t.Errorf("expected *ValidationError, got %T", err)
					return
				}

				// Verify contract type
				if validErr.ContractType != "json_schema" {
					t.Errorf("expected contract type json_schema, got %s", validErr.ContractType)
				}

				// Verify error mentions additional properties or the specific field
				errStr := validErr.Error()
				detailStr := strings.Join(validErr.Details, " ")
				combinedOutput := strings.ToLower(errStr + " " + detailStr)

				found := false
				for _, expected := range tt.errorContains {
					if strings.Contains(combinedOutput, strings.ToLower(expected)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error to mention one of %v, got: %s", tt.errorContains, combinedOutput)
				}

				// Verify there are details present
				if len(validErr.Details) == 0 {
					t.Error("expected ValidationError.Details to contain information about the additionalProperties violation")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for valid input, got: %v", err)
				}
			}
		})
	}
}

// writeTestArtifactForJSONSchema creates .wave/artifact.json in the workspace for tests
func writeTestArtifactForJSONSchema(t *testing.T, workspacePath string, data []byte) {
	t.Helper()
	waveDir := filepath.Join(workspacePath, ".wave")
	if err := os.MkdirAll(waveDir, 0755); err != nil {
		t.Fatalf("failed to create .wave directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(waveDir, "artifact.json"), data, 0644); err != nil {
		t.Fatalf("failed to write artifact.json: %v", err)
	}
}
