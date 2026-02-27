package contract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// T073: Test for TypeScript validation without tsc available

// TestTypeScriptValidator_WithoutTsc tests graceful degradation when tsc is not available.
func TestTypeScriptValidator_WithoutTsc(t *testing.T) {
	// Reset cache before tests
	ResetTypeScriptAvailabilityCache()
	defer ResetTypeScriptAvailabilityCache()

	tests := []struct {
		name        string
		cfg         ContractConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "graceful degradation when tsc unavailable and must_pass false",
			cfg: ContractConfig{
				Type:       "typescript_interface",
				SchemaPath: "/some/file.ts",
				MustPass: false,
			},
			expectError: false,
		},
		{
			name: "error when tsc unavailable and must_pass enabled",
			cfg: ContractConfig{
				Type:       "typescript_interface",
				SchemaPath: "/some/file.ts",
				MustPass: true,
			},
			expectError: true,
			errorMsg:    "TypeScript compiler (tsc) not available",
		},
	}

	// Only run these tests if tsc is NOT available
	if available, _ := CheckTypeScriptAvailability(); available {
		t.Skip("skipping tsc-unavailable tests because tsc is available")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &typeScriptValidator{}
			workspacePath := t.TempDir()

			err := v.Validate(tt.cfg, workspacePath)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("error should contain %q, got: %v", tt.errorMsg, err)
				}
				// Verify it's a ValidationError
				validErr, ok := err.(*ValidationError)
				if !ok {
					t.Errorf("expected ValidationError, got %T", err)
					return
				}
				if validErr.ContractType != "typescript_interface" {
					t.Errorf("expected contract type typescript_interface, got %s", validErr.ContractType)
				}
				// Should include installation instructions
				hasInstallHint := false
				for _, detail := range validErr.Details {
					if strings.Contains(detail, "npm install") {
						hasInstallHint = true
						break
					}
				}
				if !hasInstallHint {
					t.Error("expected error to include npm install instructions")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestTypeScriptValidator_TableDriven tests various TypeScript validation scenarios.
func TestTypeScriptValidator_TableDriven(t *testing.T) {
	// Reset cache before tests
	ResetTypeScriptAvailabilityCache()
	defer ResetTypeScriptAvailabilityCache()

	tscAvailable, _ := CheckTypeScriptAvailability()

	tests := []struct {
		name                   string
		cfg                    ContractConfig
		createFile             bool
		fileContent            string
		expectError            bool
		errorContainsWithTsc   string
		errorContainsNoTsc     string
		requiresTsc            bool // Skip if tsc not available
	}{
		{
			name: "missing schema path with tsc",
			cfg: ContractConfig{
				Type:       "typescript_interface",
				SchemaPath: "",
				MustPass: true,
			},
			createFile:           false,
			expectError:          true,
			errorContainsWithTsc: "no contract file path provided",
			errorContainsNoTsc:   "not available",
			requiresTsc:          false,
		},
		{
			name: "nonexistent file with tsc",
			cfg: ContractConfig{
				Type:       "typescript_interface",
				SchemaPath: "/nonexistent/path/interface.ts",
				MustPass: true,
			},
			createFile:           false,
			expectError:          true,
			errorContainsWithTsc: "does not exist",
			errorContainsNoTsc:   "not available",
			requiresTsc:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.requiresTsc && !tscAvailable {
				t.Skip("skipping test that requires tsc")
			}

			v := &typeScriptValidator{}
			workspacePath := t.TempDir()

			cfg := tt.cfg
			if tt.createFile {
				filePath := filepath.Join(workspacePath, "test.ts")
				if err := os.WriteFile(filePath, []byte(tt.fileContent), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				cfg.SchemaPath = filePath
			}

			err := v.Validate(cfg, workspacePath)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				// Check appropriate error message based on tsc availability
				expectedContains := tt.errorContainsWithTsc
				if !tscAvailable {
					expectedContains = tt.errorContainsNoTsc
				}
				if expectedContains != "" && !strings.Contains(err.Error(), expectedContains) {
					t.Errorf("error should contain %q (tsc available: %v), got: %v", expectedContains, tscAvailable, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestTypeScriptValidator_WithTsc tests TypeScript validation when tsc IS available.
func TestTypeScriptValidator_WithTsc(t *testing.T) {
	// Reset cache before tests
	ResetTypeScriptAvailabilityCache()
	defer ResetTypeScriptAvailabilityCache()

	if available, _ := CheckTypeScriptAvailability(); !available {
		t.Skip("skipping tsc-available tests because tsc is not installed")
	}

	tests := []struct {
		name          string
		fileContent   string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid TypeScript interface",
			fileContent: `
interface User {
    name: string;
    age: number;
}
const user: User = { name: "Alice", age: 30 };
`,
			expectError: false,
		},
		{
			name: "invalid TypeScript syntax",
			fileContent: `
interface User {
    name: string
    age: number  // missing semicolon in strict mode
}
const user: User = { name: "Alice", age: "thirty" };  // type error
`,
			expectError:   true,
			errorContains: "TypeScript validation failed",
		},
		{
			name: "type mismatch",
			fileContent: `
interface Config {
    port: number;
    host: string;
}
const config: Config = { port: "8080", host: 123 };  // wrong types
`,
			expectError:   true,
			errorContains: "TypeScript validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &typeScriptValidator{}
			workspacePath := t.TempDir()

			filePath := filepath.Join(workspacePath, "test.ts")
			if err := os.WriteFile(filePath, []byte(tt.fileContent), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			cfg := ContractConfig{
				Type:       "typescript_interface",
				SchemaPath: filePath,
				MustPass: true,
			}

			err := v.Validate(cfg, workspacePath)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error should contain %q, got: %v", tt.errorContains, err)
				}
				// Verify ValidationError structure
				if validErr, ok := err.(*ValidationError); ok {
					if validErr.ContractType != "typescript_interface" {
						t.Errorf("expected contract type typescript_interface, got %s", validErr.ContractType)
					}
					if len(validErr.Details) == 0 {
						t.Error("expected validation details with tsc errors")
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

// TestCheckTypeScriptAvailability tests the availability check function.
func TestCheckTypeScriptAvailability(t *testing.T) {
	// Reset cache before test
	ResetTypeScriptAvailabilityCache()
	defer ResetTypeScriptAvailabilityCache()

	available, version := CheckTypeScriptAvailability()

	if available {
		if version == "" {
			t.Error("tsc is available but version is empty")
		}
		if !strings.Contains(version, "Version") {
			t.Logf("tsc version: %s", version)
		}
	} else {
		if version != "" {
			t.Errorf("tsc is not available but version is non-empty: %s", version)
		}
	}
}

// TestResetTypeScriptAvailabilityCache tests the cache reset function.
func TestResetTypeScriptAvailabilityCache(t *testing.T) {
	// First check
	available1, _ := CheckTypeScriptAvailability()

	// Reset cache
	ResetTypeScriptAvailabilityCache()

	// Second check should work the same
	available2, _ := CheckTypeScriptAvailability()

	if available1 != available2 {
		t.Error("availability check should be consistent after reset")
	}
}

// TestExtractTypeScriptErrors tests the error extraction helper.
func TestExtractTypeScriptErrors(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected int // minimum number of extracted details
	}{
		{
			name:     "empty output",
			output:   "",
			expected: 0,
		},
		{
			name:     "single line",
			output:   "error TS2322: Type 'string' is not assignable to type 'number'.",
			expected: 1,
		},
		{
			name:     "multiple lines",
			output:   "file.ts(1,5): error TS2322: Type 'string' is not assignable.\nfile.ts(2,3): error TS2345: Argument of type 'number' is not assignable.",
			expected: 2,
		},
		{
			name:     "lines with whitespace",
			output:   "\n  error line 1  \n\n  error line 2\n  ",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details := extractTypeScriptErrors(tt.output)
			if len(details) < tt.expected {
				t.Errorf("expected at least %d details, got %d", tt.expected, len(details))
			}
		})
	}
}
