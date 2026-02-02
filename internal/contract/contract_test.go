package contract

import (
	"os"
	"testing"
)

func TestJSONSchemaValidator_Valid(t *testing.T) {
	v := &jsonSchemaValidator{}
	cfg := ContractConfig{
		Type:   "json_schema",
		Schema: `{"type": "object", "properties": {"name": {"type": "string"}}}`,
	}

	workspacePath := t.TempDir()
	artifactPath := workspacePath + "/artifact.json"
	os.WriteFile(artifactPath, []byte(`{"name": "test"}`), 0644)

	err := v.Validate(cfg, workspacePath)
	if err != nil {
		t.Errorf("expected valid artifact to pass, got error: %v", err)
	}
}

func TestJSONSchemaValidator_Invalid(t *testing.T) {
	v := &jsonSchemaValidator{}
	cfg := ContractConfig{
		Type:   "json_schema",
		Schema: `{"type": "object", "properties": {"name": {"type": "string"}}}`,
	}

	workspacePath := t.TempDir()
	artifactPath := workspacePath + "/artifact.json"
	os.WriteFile(artifactPath, []byte(`{"name": 123}`), 0644)

	err := v.Validate(cfg, workspacePath)
	if err == nil {
		t.Error("expected invalid artifact to fail, but got no error")
	}
}

func TestJSONSchemaValidator_MissingSchema(t *testing.T) {
	v := &jsonSchemaValidator{}
	cfg := ContractConfig{
		Type: "json_schema",
	}

	workspacePath := t.TempDir()
	err := v.Validate(cfg, workspacePath)
	if err == nil {
		t.Error("expected error for missing schema, but got none")
	}
}

func TestTypeScriptValidator_MissingFile(t *testing.T) {
	v := &typeScriptValidator{}
	cfg := ContractConfig{
		Type:       "typescript_interface",
		SchemaPath: "/nonexistent/file.ts",
		StrictMode: true, // Enable strict mode to get error for missing file even if tsc unavailable
	}
	workspacePath := t.TempDir()

	err := v.Validate(cfg, workspacePath)
	if err == nil {
		t.Error("expected error for missing file, but got none")
	}
}

func TestTestSuiteValidator_MissingCommand(t *testing.T) {
	v := &testSuiteValidator{}
	cfg := ContractConfig{
		Type: "test_suite",
	}
	workspacePath := t.TempDir()

	err := v.Validate(cfg, workspacePath)
	if err == nil {
		t.Error("expected error for missing command, but got none")
	}
}

func TestTypeScriptValidator_GracefulDegradation(t *testing.T) {
	v := &typeScriptValidator{}
	cfg := ContractConfig{
		Type:       "typescript_interface",
		SchemaPath: "/nonexistent/file.ts",
		StrictMode: false,
	}
	workspacePath := t.TempDir()

	// This test would only pass if tsc is not available
	// In a CI environment with tsc, this would still fail due to file not found
	if !IsTypeScriptAvailable() {
		err := v.Validate(cfg, workspacePath)
		if err != nil {
			t.Errorf("expected graceful degradation when tsc not available, got error: %v", err)
		}
	}
}

func TestTypeScriptValidator_StrictMode(t *testing.T) {
	v := &typeScriptValidator{}
	cfg := ContractConfig{
		Type:       "typescript_interface",
		SchemaPath: "/nonexistent/file.ts",
		StrictMode: true,
	}
	workspacePath := t.TempDir()

	if !IsTypeScriptAvailable() {
		err := v.Validate(cfg, workspacePath)
		if err == nil {
			t.Error("expected error when tsc not available and strict mode is enabled")
		}
	}
}

func TestTestSuiteValidator_CommandExecution(t *testing.T) {
	v := &testSuiteValidator{}
	cfg := ContractConfig{
		Type:        "test_suite",
		Command:     "echo",
		CommandArgs: []string{"hello"},
	}
	workspacePath := t.TempDir()

	err := v.Validate(cfg, workspacePath)
	if err != nil {
		t.Errorf("expected echo command to succeed, got error: %v", err)
	}
}

func TestTestSuiteValidator_CommandFailure(t *testing.T) {
	v := &testSuiteValidator{}
	cfg := ContractConfig{
		Type:    "test_suite",
		Command: "false", // always fails
	}
	workspacePath := t.TempDir()

	err := v.Validate(cfg, workspacePath)
	if err == nil {
		t.Error("expected false command to fail, but got no error")
	}
}

func TestValidate_Function(t *testing.T) {
	tests := []struct {
		name        string
		config      ContractConfig
		expectError bool
	}{
		{"valid json_schema", ContractConfig{Type: "json_schema", Schema: `{"type": "object"}`}, false},
		{"missing schema", ContractConfig{Type: "json_schema"}, true},
		{"unknown type", ContractConfig{Type: "unknown"}, false}, // Should return nil (no error)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspacePath := t.TempDir()
			if tt.config.Type == "json_schema" && tt.config.Schema != "" {
				artifactPath := workspacePath + "/artifact.json"
				os.WriteFile(artifactPath, []byte(`{}`), 0644)
			}

			err := Validate(tt.config, workspacePath)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestNewValidator(t *testing.T) {
	tests := []struct {
		name   string
		config ContractConfig
	}{
		{"json_schema", ContractConfig{Type: "json_schema"}},
		{"typescript_interface", ContractConfig{Type: "typescript_interface"}},
		{"test_suite", ContractConfig{Type: "test_suite"}},
		{"unknown", ContractConfig{Type: "unknown"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator(tt.config)
			if tt.config.Type == "unknown" && validator != nil {
				t.Error("expected nil validator for unknown type")
			}
		})
	}
}
