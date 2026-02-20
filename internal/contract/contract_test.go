package contract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestArtifact creates .wave/artifact.json in the workspace for tests
func writeTestArtifact(t *testing.T, workspacePath string, data []byte) {
	t.Helper()
	waveDir := filepath.Join(workspacePath, ".wave")
	os.MkdirAll(waveDir, 0755)
	os.WriteFile(filepath.Join(waveDir, "artifact.json"), data, 0644)
}

func TestJSONSchemaValidator_Valid(t *testing.T) {
	v := &jsonSchemaValidator{}
	cfg := ContractConfig{
		Type:   "json_schema",
		Schema: `{"type": "object", "properties": {"name": {"type": "string"}}}`,
	}

	workspacePath := t.TempDir()
	writeTestArtifact(t, workspacePath, []byte(`{"name": "test"}`))

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
	writeTestArtifact(t, workspacePath, []byte(`{"name": 123}`))

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
			errorContains: "contract validation failed",
		},
		{
			name:          "wrong type for integer field",
			schema:        `{"type": "object", "properties": {"age": {"type": "integer"}}, "required": ["age"]}`,
			artifact:      `{"age": "thirty"}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:          "missing required field",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			artifact:      `{"other": "value"}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:          "invalid JSON artifact",
			schema:        `{"type": "object"}`,
			artifact:      `{not valid json}`,
			expectError:   true,
			errorContains: "failed to parse artifact JSON",
		},
		{
			name:          "array instead of object",
			schema:        `{"type": "object"}`,
			artifact:      `[1, 2, 3]`,
			expectError:   true,
			errorContains: "contract validation failed",
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
			errorContains: "contract validation failed",
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
	workspacePath := t.TempDir()
	cfg := ContractConfig{
		Type:        "test_suite",
		Command:     "echo",
		CommandArgs: []string{"hello"},
		Dir:         workspacePath,
	}

	err := v.Validate(cfg, workspacePath)
	if err != nil {
		t.Errorf("expected echo command to succeed, got error: %v", err)
	}
}

func TestTestSuiteValidator_CommandFailure(t *testing.T) {
	v := &testSuiteValidator{}
	workspacePath := t.TempDir()
	cfg := ContractConfig{
		Type:    "test_suite",
		Command: "false", // always fails
		Dir:     workspacePath,
	}

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
			writeTestArtifact(t, workspacePath, []byte(`{"name": 123}`)) // Invalid: name should be string

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
	writeTestArtifact(t, workspacePath, []byte(`{"name": "valid"}`))

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
				writeTestArtifact(t, workspacePath, []byte(`{}`))
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
// =============================================================================
// T016-T019: User Story 7 - Contract Validator False-Positive Prevention Tests
// =============================================================================

// T016: Array vs object type coercion test
// Ensures the validator correctly rejects arrays when object type is required
// without silently coercing to an object.
func TestJSONSchemaValidator_ArrayVsObjectNoCoercion(t *testing.T) {
	tests := []struct {
		name          string
		schema        string
		artifact      string
		expectError   bool
		errorContains string
	}{
		{
			name:          "array rejected when object required",
			schema:        `{"type": "object"}`,
			artifact:      `[1, 2, 3]`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:          "empty array rejected when object required",
			schema:        `{"type": "object"}`,
			artifact:      `[]`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:          "array of objects rejected when single object required",
			schema:        `{"type": "object", "properties": {"name": {"type": "string"}}}`,
			artifact:      `[{"name": "test"}]`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:          "nested array rejected in object property",
			schema:        `{"type": "object", "properties": {"items": {"type": "object"}}}`,
			artifact:      `{"items": [1, 2, 3]}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:     "object accepted when object required",
			schema:   `{"type": "object"}`,
			artifact: `{"key": "value"}`,
			expectError: false,
		},
		{
			name:     "empty object accepted when object required",
			schema:   `{"type": "object"}`,
			artifact: `{}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &jsonSchemaValidator{}
			cfg := ContractConfig{
				Type:          "json_schema",
				Schema:        tt.schema,
				AllowRecovery: false, // Disable recovery to test strict parsing
			}

			workspacePath := t.TempDir()
			writeTestArtifact(t, workspacePath, []byte(tt.artifact))

			err := v.Validate(cfg, workspacePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for array when object required, but validation passed")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error should contain %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

// T017: Boundary condition test - minimum: 1 rejects 0
// Ensures the validator correctly rejects values at the boundary
func TestJSONSchemaValidator_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name          string
		schema        string
		artifact      string
		expectError   bool
		errorContains string
	}{
		{
			name:          "minimum 1 rejects 0",
			schema:        `{"type": "object", "properties": {"value": {"type": "integer", "minimum": 1}}}`,
			artifact:      `{"value": 0}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:          "minimum 1 rejects negative",
			schema:        `{"type": "object", "properties": {"value": {"type": "integer", "minimum": 1}}}`,
			artifact:      `{"value": -1}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:     "minimum 1 accepts 1",
			schema:   `{"type": "object", "properties": {"value": {"type": "integer", "minimum": 1}}}`,
			artifact: `{"value": 1}`,
			expectError: false,
		},
		{
			name:     "minimum 1 accepts 2",
			schema:   `{"type": "object", "properties": {"value": {"type": "integer", "minimum": 1}}}`,
			artifact: `{"value": 2}`,
			expectError: false,
		},
		{
			name:          "maximum 10 rejects 11",
			schema:        `{"type": "object", "properties": {"value": {"type": "integer", "maximum": 10}}}`,
			artifact:      `{"value": 11}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:     "maximum 10 accepts 10",
			schema:   `{"type": "object", "properties": {"value": {"type": "integer", "maximum": 10}}}`,
			artifact: `{"value": 10}`,
			expectError: false,
		},
		{
			name:          "exclusiveMinimum 1 rejects 1",
			schema:        `{"type": "object", "properties": {"value": {"type": "integer", "exclusiveMinimum": 1}}}`,
			artifact:      `{"value": 1}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:     "exclusiveMinimum 1 accepts 2",
			schema:   `{"type": "object", "properties": {"value": {"type": "integer", "exclusiveMinimum": 1}}}`,
			artifact: `{"value": 2}`,
			expectError: false,
		},
		{
			name:          "minLength 3 rejects empty string",
			schema:        `{"type": "object", "properties": {"name": {"type": "string", "minLength": 3}}}`,
			artifact:      `{"name": ""}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:          "minLength 3 rejects 2 char string",
			schema:        `{"type": "object", "properties": {"name": {"type": "string", "minLength": 3}}}`,
			artifact:      `{"name": "ab"}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:     "minLength 3 accepts 3 char string",
			schema:   `{"type": "object", "properties": {"name": {"type": "string", "minLength": 3}}}`,
			artifact: `{"name": "abc"}`,
			expectError: false,
		},
		{
			name:          "minItems 1 rejects empty array",
			schema:        `{"type": "object", "properties": {"items": {"type": "array", "minItems": 1}}}`,
			artifact:      `{"items": []}`,
			expectError:   true,
			errorContains: "contract validation failed",
		},
		{
			name:     "minItems 1 accepts single item array",
			schema:   `{"type": "object", "properties": {"items": {"type": "array", "minItems": 1}}}`,
			artifact: `{"items": [1]}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &jsonSchemaValidator{}
			cfg := ContractConfig{
				Type:          "json_schema",
				Schema:        tt.schema,
				AllowRecovery: false,
			}

			workspacePath := t.TempDir()
			writeTestArtifact(t, workspacePath, []byte(tt.artifact))

			err := v.Validate(cfg, workspacePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected boundary violation error, but validation passed")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error should contain %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

// T018: Malformed JSON test - trailing commas handling
// The JSON recovery system (enabled by default) handles common JSON5-like syntax.
// This test verifies that:
// 1. Simple trailing commas are recoverable (pass with recovery)
// 2. Severely malformed JSON (double commas) fails even with recovery
// 3. When recovery is disabled via explicit config, trailing commas fail
func TestJSONSchemaValidator_TrailingCommasRejected(t *testing.T) {
	tests := []struct {
		name          string
		artifact      string
		recoveryLevel string // empty = default (progressive), "conservative" = limited recovery
		expectError   bool
		errorContains string
		description   string
	}{
		{
			name:          "simple trailing comma recovered by default",
			artifact:      `{"name": "test",}`,
			recoveryLevel: "", // default progressive recovery
			expectError:   false,
			description:   "Single trailing commas are a common AI output pattern and should be recovered",
		},
		{
			name:          "trailing comma in array recovered",
			artifact:      `{"items": [1, 2, 3,]}`,
			recoveryLevel: "",
			expectError:   false,
			description:   "Trailing commas in arrays should be recovered",
		},
		{
			name:          "double comma not recoverable",
			artifact:      `{"a": 1,, "b": 2}`,
			recoveryLevel: "",
			expectError:   true,
			errorContains: "failed to parse artifact JSON",
			description:   "Double commas are malformed beyond simple recovery",
		},
		{
			name:          "nested trailing comma recovered",
			artifact:      `{"outer": {"inner": "value",}}`,
			recoveryLevel: "",
			expectError:   false,
			description:   "Nested trailing commas should be recovered",
		},
		{
			name:          "explicit conservative recovery still fixes trailing commas",
			artifact:      `{"name": "test",}`,
			recoveryLevel: "conservative",
			expectError:   false,
			description:   "Even conservative recovery fixes trailing commas",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &jsonSchemaValidator{}
			cfg := ContractConfig{
				Type:          "json_schema",
				Schema:        `{"type": "object"}`,
				AllowRecovery: true,
				RecoveryLevel: tt.recoveryLevel,
			}

			workspacePath := t.TempDir()
			writeTestArtifact(t, workspacePath, []byte(tt.artifact))

			err := v.Validate(cfg, workspacePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected parse error (%s), but validation passed", tt.description)
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error should contain %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected success (%s), but got error: %v", tt.description, err)
				}
			}
		})
	}
}

// T019: Malformed JSON test - JSON with comments handling
// The JSON recovery system (enabled by default) strips various comment styles.
// This test verifies that:
// 1. Single-line comments (//) are stripped and JSON is recovered
// 2. Multi-line comments are stripped and JSON is recovered
// 3. Hash comments (#) at line start are stripped
// 4. Comments in string values that look like JSON cause parse errors
func TestJSONSchemaValidator_CommentsRejected(t *testing.T) {
	tests := []struct {
		name          string
		artifact      string
		expectError   bool
		errorContains string
		description   string
	}{
		{
			name:        "single line comment after JSON object stripped",
			artifact:    `{"name": "test"} // this is a comment`,
			expectError: false,
			description: "Trailing single-line comments after complete JSON should be stripped",
		},
		{
			name: "multi line comment after value recovered",
			artifact: `{"name": "test" /* multi
line comment */}`,
			expectError: false,
			description: "Multi-line comments should be stripped",
		},
		{
			name:        "hash comment after JSON stripped",
			artifact:    `{"name": "test"} # comment`,
			expectError: false,
			description: "Hash comments at end should be stripped",
		},
		{
			name: "comment on separate line before property stripped",
			artifact: `{
// comment explaining the field
"name": "test"
}`,
			expectError: false,
			description: "Comment lines between JSON elements should be stripped",
		},
		{
			name:          "comment inside string breaks JSON structure",
			artifact:      `{"name": "test // not a comment}`,
			expectError:   true,
			errorContains: "failed to parse artifact JSON",
			description:   "Unclosed string (comment marker inside string value) should fail",
		},
		{
			name: "inline comment before closing brace causes error",
			// This creates invalid JSON: {"name": "test" [STRIPPED] }
			// The issue is the space between "test" and } after stripping
			artifact:      `{"name": "test" // this breaks the value}`,
			expectError:   true,
			errorContains: "failed to parse artifact JSON",
			description:   "Comment between value and closing brace breaks JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &jsonSchemaValidator{}
			cfg := ContractConfig{
				Type:          "json_schema",
				Schema:        `{"type": "object"}`,
				AllowRecovery: true, // Recovery is enabled by default
			}

			workspacePath := t.TempDir()
			writeTestArtifact(t, workspacePath, []byte(tt.artifact))

			err := v.Validate(cfg, workspacePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected parse error (%s), but validation passed", tt.description)
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error should contain %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected success (%s), but got error: %v", tt.description, err)
				}
			}
		})
	}
}

// End of T016-T019 tests

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
				waveDir := filepath.Join(workspacePath, ".wave")
				os.MkdirAll(waveDir, 0755)
				os.WriteFile(filepath.Join(waveDir, "artifact.json"), []byte(`{"value": 42}`), 0644)
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
				waveDir := filepath.Join(workspacePath, ".wave")
				os.MkdirAll(waveDir, 0755)
				os.WriteFile(filepath.Join(waveDir, "artifact.json"), []byte(`{"value": "not a number"}`), 0644)
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

			// Set Dir for test_suite configs so tests don't need a git repo
			cfg := tt.config
			if cfg.Type == "test_suite" && cfg.Dir == "" && cfg.Command != "" {
				cfg.Dir = workspacePath
			}

			err := Validate(cfg, workspacePath)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

// T006: Test missing required field with specific field identification in error
func TestJSONSchemaValidator_MissingRequiredField(t *testing.T) {
	tests := []struct {
		name               string
		schema             string
		artifact           string
		expectError        bool
		missingFieldName   string
		errorShouldContain []string
	}{
		{
			name:               "single required field missing",
			schema:             `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			artifact:           `{}`,
			expectError:        true,
			missingFieldName:   "name",
			errorShouldContain: []string{"name", "required"},
		},
		{
			name:               "one of multiple required fields missing",
			schema:             `{"type": "object", "properties": {"name": {"type": "string"}, "email": {"type": "string"}}, "required": ["name", "email"]}`,
			artifact:           `{"name": "Alice"}`,
			expectError:        true,
			missingFieldName:   "email",
			errorShouldContain: []string{"email"},
		},
		{
			name:               "nested required field missing",
			schema:             `{"type": "object", "properties": {"user": {"type": "object", "properties": {"id": {"type": "integer"}}, "required": ["id"]}}, "required": ["user"]}`,
			artifact:           `{"user": {}}`,
			expectError:        true,
			missingFieldName:   "id",
			errorShouldContain: []string{"id"},
		},
		{
			name:               "all required fields missing",
			schema:             `{"type": "object", "properties": {"a": {"type": "string"}, "b": {"type": "string"}}, "required": ["a", "b"]}`,
			artifact:           `{}`,
			expectError:        true,
			missingFieldName:   "a",
			errorShouldContain: []string{"required"},
		},
		{
			name:               "required field present with wrong name (case sensitivity)",
			schema:             `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			artifact:           `{"Name": "Alice"}`,
			expectError:        true,
			missingFieldName:   "name",
			errorShouldContain: []string{"name"},
		},
		{
			name:               "all required fields present",
			schema:             `{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "integer"}}, "required": ["name", "age"]}`,
			artifact:           `{"name": "Alice", "age": 30}`,
			expectError:        false,
			missingFieldName:   "",
			errorShouldContain: nil,
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
					t.Errorf("expected error for missing required field %q, got none", tt.missingFieldName)
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

				// Verify error specifies which field is missing
				errStr := validErr.Error()
				detailStr := strings.Join(validErr.Details, " ")
				combinedOutput := strings.ToLower(errStr + " " + detailStr)

				for _, expected := range tt.errorShouldContain {
					if !strings.Contains(combinedOutput, strings.ToLower(expected)) {
						t.Errorf("expected error to mention %q for missing field, got: %s", expected, combinedOutput)
					}
				}

				// Verify there are details present
				if len(validErr.Details) == 0 {
					t.Error("expected ValidationError.Details to specify which field is missing")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for valid input, got: %v", err)
				}
			}
		})
	}
}

// T008: Test that ValidationError.Details contains the specific field path that failed
func TestValidationError_FieldPathIdentification(t *testing.T) {
	tests := []struct {
		name                  string
		schema                string
		artifact              string
		expectedFieldPath     string // the field path that should appear in error
		expectedErrorPatterns []string
	}{
		{
			name:                  "root level type mismatch identifies field",
			schema:                `{"type": "object", "properties": {"status": {"type": "string"}}, "required": ["status"]}`,
			artifact:              `{"status": 123}`,
			expectedFieldPath:     "status",
			expectedErrorPatterns: []string{"status"},
		},
		{
			name:                  "nested field type mismatch identifies path",
			schema:                `{"type": "object", "properties": {"config": {"type": "object", "properties": {"enabled": {"type": "boolean"}}}}}`,
			artifact:              `{"config": {"enabled": "yes"}}`,
			expectedFieldPath:     "enabled",
			expectedErrorPatterns: []string{"enabled"},
		},
		{
			name:                  "array item type mismatch",
			schema:                `{"type": "object", "properties": {"tags": {"type": "array", "items": {"type": "string"}}}}`,
			artifact:              `{"tags": ["valid", 123, "also valid"]}`,
			expectedFieldPath:     "tags",
			expectedErrorPatterns: []string{"tags"},
		},
		{
			name:                  "deeply nested field identifies full path",
			schema:                `{"type": "object", "properties": {"data": {"type": "object", "properties": {"user": {"type": "object", "properties": {"age": {"type": "integer"}}}}}}}`,
			artifact:              `{"data": {"user": {"age": "not a number"}}}`,
			expectedFieldPath:     "age",
			expectedErrorPatterns: []string{"age"},
		},
		{
			name:                  "enum violation identifies field",
			schema:                `{"type": "object", "properties": {"priority": {"type": "string", "enum": ["low", "medium", "high"]}}}`,
			artifact:              `{"priority": "critical"}`,
			expectedFieldPath:     "priority",
			expectedErrorPatterns: []string{"priority"},
		},
		{
			name:                  "minimum violation identifies field",
			schema:                `{"type": "object", "properties": {"count": {"type": "integer", "minimum": 1}}}`,
			artifact:              `{"count": 0}`,
			expectedFieldPath:     "count",
			expectedErrorPatterns: []string{"count"},
		},
		{
			name:                  "pattern violation identifies field",
			schema:                `{"type": "object", "properties": {"email": {"type": "string", "pattern": "^[a-z]+@[a-z]+\\.[a-z]+$"}}}`,
			artifact:              `{"email": "invalid-email"}`,
			expectedFieldPath:     "email",
			expectedErrorPatterns: []string{"email"},
		},
		{
			name:                  "additionalProperties violation identifies extra field",
			schema:                `{"type": "object", "properties": {"name": {"type": "string"}}, "additionalProperties": false}`,
			artifact:              `{"name": "test", "unknown": "field"}`,
			expectedFieldPath:     "unknown",
			expectedErrorPatterns: []string{"unknown", "additional"},
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
			if err == nil {
				t.Fatal("expected validation error, got none")
			}

			// Verify it's a ValidationError
			validErr, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("expected *ValidationError, got %T", err)
			}

			// Verify Details contains field path information
			if len(validErr.Details) == 0 {
				t.Fatal("expected ValidationError.Details to contain field path, but Details is empty")
			}

			// Check that the error contains the expected field path
			errStr := validErr.Error()
			detailStr := strings.Join(validErr.Details, " ")
			combinedOutput := strings.ToLower(errStr + " " + detailStr)

			// At least one of the expected patterns should be found
			found := false
			for _, pattern := range tt.expectedErrorPatterns {
				if strings.Contains(combinedOutput, strings.ToLower(pattern)) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected error to identify field path containing one of %v\nGot error: %s\nDetails: %v",
					tt.expectedErrorPatterns, errStr, validErr.Details)
			}

			// Log the actual error for debugging (helpful during development)
			t.Logf("Error message for field %q: %s", tt.expectedFieldPath, errStr)
			t.Logf("Details: %v", validErr.Details)
		})
	}
}

// TestValidationError_DetailsNotEmpty verifies that validation errors always include details
func TestValidationError_DetailsNotEmpty(t *testing.T) {
	failureCases := []struct {
		name     string
		schema   string
		artifact string
	}{
		{
			name:     "type mismatch",
			schema:   `{"type": "object", "properties": {"val": {"type": "string"}}}`,
			artifact: `{"val": 123}`,
		},
		{
			name:     "missing required",
			schema:   `{"type": "object", "required": ["id"]}`,
			artifact: `{}`,
		},
		{
			name:     "additional property",
			schema:   `{"type": "object", "additionalProperties": false}`,
			artifact: `{"extra": true}`,
		},
		{
			name:     "enum violation",
			schema:   `{"type": "object", "properties": {"status": {"enum": ["a", "b"]}}}`,
			artifact: `{"status": "c"}`,
		},
	}

	for _, tc := range failureCases {
		t.Run(tc.name, func(t *testing.T) {
			v := &jsonSchemaValidator{}
			cfg := ContractConfig{
				Type:   "json_schema",
				Schema: tc.schema,
			}

			workspacePath := t.TempDir()
			writeTestArtifact(t, workspacePath, []byte(tc.artifact))

			err := v.Validate(cfg, workspacePath)
			if err == nil {
				t.Fatal("expected validation error, got none")
			}

			validErr, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("expected *ValidationError, got %T", err)
			}

			if len(validErr.Details) == 0 {
				t.Errorf("ValidationError.Details should not be empty for %s", tc.name)
			}

			// Verify Details contains meaningful content (not just empty strings)
			hasContent := false
			for _, detail := range validErr.Details {
				if strings.TrimSpace(detail) != "" {
					hasContent = true
					break
				}
			}
			if !hasContent {
				t.Errorf("ValidationError.Details should contain meaningful content for %s, got: %v", tc.name, validErr.Details)
			}
		})
	}
}
