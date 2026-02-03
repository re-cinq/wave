package contract

import (
	"os"
	"strings"
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

// T072: Test for JSON schema validation failure with detailed error messages
func TestJSONSchemaValidator_ValidationFailure_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		schema         string
		artifact       string
		expectError    bool
		errorContains  string
	}{
		{
			name:          "valid object",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "integer"}}, "required": ["name"]}`,
			artifact:      `{"name": "Alice", "age": 30}`,
			expectError:   false,
		},
		{
			name:          "wrong type for string field",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			artifact:      `{"name": 123}`,
			expectError:   true,
			errorContains: "does not match schema",
		},
		{
			name:          "wrong type for integer field",
			schema:        `{"type": "object", "properties": {"age": {"type": "integer"}}, "required": ["age"]}`,
			artifact:      `{"age": "thirty"}`,
			expectError:   true,
			errorContains: "does not match schema",
		},
		{
			name:          "missing required field",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			artifact:      `{"other": "value"}`,
			expectError:   true,
			errorContains: "does not match schema",
		},
		{
			name:          "invalid JSON artifact",
			schema:        `{"type": "object"}`,
			artifact:      `{not valid json}`,
			expectError:   true,
			errorContains: "failed to clean malformed JSON",
		},
		{
			name:          "array instead of object",
			schema:        `{"type": "object"}`,
			artifact:      `[1, 2, 3]`,
			expectError:   true,
			errorContains: "does not match schema",
		},
		{
			name:          "nested object validation",
			schema:        `{"type": "object", "properties": {"user": {"type": "object", "properties": {"email": {"type": "string", "format": "email"}}}}}`,
			artifact:      `{"user": {"email": "valid@example.com"}}`,
			expectError:   false,
		},
		{
			name:          "additional properties not allowed",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}, "additionalProperties": false}`,
			artifact:      `{"name": "test", "extra": "field"}`,
			expectError:   true,
			errorContains: "does not match schema",
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
			artifactPath := workspacePath + "/artifact.json"
			os.WriteFile(artifactPath, []byte(tt.artifact), 0644)

			err := v.Validate(cfg, workspacePath)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error should contain %q, got: %v", tt.errorContains, err)
				}
				// Verify it's a ValidationError with details
				if validErr, ok := err.(*ValidationError); ok {
					if validErr.ContractType != "json_schema" {
						t.Errorf("expected contract type json_schema, got %s", validErr.ContractType)
					}
					if len(validErr.Details) == 0 {
						t.Error("expected validation details, got none")
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

// T072: Test ValidationError formatting
func TestValidationError_Format(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		contains []string
	}{
		{
			name: "basic error",
			err: &ValidationError{
				ContractType: "json_schema",
				Message:      "validation failed",
			},
			contains: []string{"json_schema", "validation failed"},
		},
		{
			name: "error with details",
			err: &ValidationError{
				ContractType: "json_schema",
				Message:      "validation failed",
				Details:      []string{"field 'name' is required", "type mismatch at /age"},
			},
			contains: []string{"json_schema", "validation failed", "Details:", "field 'name' is required", "type mismatch at /age"},
		},
		{
			name: "error with retry info",
			err: &ValidationError{
				ContractType: "test_suite",
				Message:      "test failed",
				Attempt:      2,
				MaxRetries:   3,
			},
			contains: []string{"test_suite", "attempt 2/3", "test failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, c := range tt.contains {
				if !strings.Contains(errStr, c) {
					t.Errorf("error string should contain %q, got: %s", c, errStr)
				}
			}
		})
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
	// Reset cache before test
	ResetTypeScriptAvailabilityCache()
	defer ResetTypeScriptAvailabilityCache()

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
	// Reset cache before test
	ResetTypeScriptAvailabilityCache()
	defer ResetTypeScriptAvailabilityCache()

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
		// Verify the error message mentions installation instructions
		if validErr, ok := err.(*ValidationError); ok {
			foundInstallHint := false
			for _, detail := range validErr.Details {
				if strings.Contains(detail, "npm install") {
					foundInstallHint = true
					break
				}
			}
			if !foundInstallHint {
				t.Error("expected error to contain installation instructions")
			}
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

// T075: Test for max_retries exhaustion
func TestValidateWithRetries_MaxRetriesExhausted(t *testing.T) {
	tests := []struct {
		name       string
		maxRetries int
		wantAttempts int
	}{
		{
			name:       "single retry",
			maxRetries: 1,
			wantAttempts: 1,
		},
		{
			name:       "three retries",
			maxRetries: 3,
			wantAttempts: 3,
		},
		{
			name:       "five retries",
			maxRetries: 5,
			wantAttempts: 5,
		},
		{
			name:       "zero defaults to one",
			maxRetries: 0,
			wantAttempts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create an invalid artifact that will always fail validation
			workspacePath := t.TempDir()
			artifactPath := workspacePath + "/artifact.json"
			os.WriteFile(artifactPath, []byte(`{"name": 123}`), 0644) // Invalid: name should be string

			cfg := ContractConfig{
				Type:       "json_schema",
				Schema:     `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
				MaxRetries: tt.maxRetries,
			}

			err := ValidateWithRetries(cfg, workspacePath)
			if err == nil {
				t.Fatal("expected error after retries exhausted")
			}

			validErr, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("expected ValidationError, got %T", err)
			}

			if validErr.Attempt != tt.wantAttempts {
				t.Errorf("expected %d attempts, got %d", tt.wantAttempts, validErr.Attempt)
			}

			if validErr.MaxRetries != tt.wantAttempts {
				t.Errorf("expected max retries %d, got %d", tt.wantAttempts, validErr.MaxRetries)
			}

			if validErr.Retryable {
				t.Error("expected Retryable to be false after exhausting retries")
			}

			if !strings.Contains(validErr.Error(), "attempt") {
				t.Error("expected error message to contain attempt information")
			}
		})
	}
}

// T075: Test ValidateWithRetries succeeds on first try
func TestValidateWithRetries_SuccessFirstTry(t *testing.T) {
	workspacePath := t.TempDir()
	artifactPath := workspacePath + "/artifact.json"
	os.WriteFile(artifactPath, []byte(`{"name": "valid"}`), 0644)

	cfg := ContractConfig{
		Type:       "json_schema",
		Schema:     `{"type": "object", "properties": {"name": {"type": "string"}}}`,
		MaxRetries: 3,
	}

	err := ValidateWithRetries(cfg, workspacePath)
	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
}

// T075: Test ValidateWithRetries with unknown validator type
func TestValidateWithRetries_UnknownType(t *testing.T) {
	cfg := ContractConfig{
		Type:       "unknown",
		MaxRetries: 3,
	}

	err := ValidateWithRetries(cfg, t.TempDir())
	if err != nil {
		t.Errorf("expected nil for unknown validator type, got: %v", err)
	}
}

// Test WrapValidationError helper
func TestWrapValidationError(t *testing.T) {
	originalErr := os.ErrNotExist
	wrapped := WrapValidationError("json_schema", originalErr, "file not found", "check path")

	if wrapped.ContractType != "json_schema" {
		t.Errorf("expected contract type json_schema, got %s", wrapped.ContractType)
	}

	if !wrapped.Retryable {
		t.Error("expected Retryable to be true")
	}

	if len(wrapped.Details) != 2 {
		t.Errorf("expected 2 details, got %d", len(wrapped.Details))
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
		name         string
		config       ContractConfig
		expectNil    bool
		expectedType string
	}{
		{"json_schema", ContractConfig{Type: "json_schema"}, false, "*contract.jsonSchemaValidator"},
		{"typescript_interface", ContractConfig{Type: "typescript_interface"}, false, "*contract.typeScriptValidator"},
		{"test_suite", ContractConfig{Type: "test_suite"}, false, "*contract.testSuiteValidator"},
		{"unknown", ContractConfig{Type: "unknown"}, true, ""},
		{"empty type", ContractConfig{Type: ""}, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator(tt.config)
			if tt.expectNil {
				if validator != nil {
					t.Errorf("expected nil validator for type %q", tt.config.Type)
				}
			} else {
				if validator == nil {
					t.Errorf("expected non-nil validator for type %q", tt.config.Type)
				}
			}
		})
	}
}

// Test Validate function with all contract types
func TestValidate_AllTypes(t *testing.T) {
	// Reset TypeScript cache
	ResetTypeScriptAvailabilityCache()
	defer ResetTypeScriptAvailabilityCache()

	tests := []struct {
		name          string
		config        ContractConfig
		setupArtifact func(workspacePath string)
		expectError   bool
	}{
		{
			name: "json_schema valid",
			config: ContractConfig{
				Type:   "json_schema",
				Schema: `{"type": "object", "properties": {"value": {"type": "number"}}}`,
			},
			setupArtifact: func(workspacePath string) {
				os.WriteFile(workspacePath+"/artifact.json", []byte(`{"value": 42}`), 0644)
			},
			expectError: false,
		},
		{
			name: "json_schema invalid",
			config: ContractConfig{
				Type:   "json_schema",
				Schema: `{"type": "object", "properties": {"value": {"type": "number"}}}`,
			},
			setupArtifact: func(workspacePath string) {
				os.WriteFile(workspacePath+"/artifact.json", []byte(`{"value": "not a number"}`), 0644)
			},
			expectError: true,
		},
		{
			name: "test_suite success",
			config: ContractConfig{
				Type:    "test_suite",
				Command: "true",
			},
			setupArtifact: func(workspacePath string) {},
			expectError:   false,
		},
		{
			name: "test_suite failure",
			config: ContractConfig{
				Type:    "test_suite",
				Command: "false",
			},
			setupArtifact: func(workspacePath string) {},
			expectError:   true,
		},
		{
			name: "unknown type",
			config: ContractConfig{
				Type: "custom_validator",
			},
			setupArtifact: func(workspacePath string) {},
			expectError:   false, // Unknown types return nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspacePath := t.TempDir()
			tt.setupArtifact(workspacePath)

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
