package contract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// T074: Test for test suite validation

// TestTestSuiteValidator_TableDriven tests various test suite validation scenarios.
func TestTestSuiteValidator_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		cfg           ContractConfig
		expectError   bool
		errorContains string
	}{
		{
			name: "missing command",
			cfg: ContractConfig{
				Type:    "test_suite",
				Command: "",
			},
			expectError:   true,
			errorContains: "no command configured",
		},
		{
			name: "successful echo command",
			cfg: ContractConfig{
				Type:        "test_suite",
				Command:     "echo",
				CommandArgs: []string{"hello", "world"},
			},
			expectError: false,
		},
		{
			name: "successful true command",
			cfg: ContractConfig{
				Type:    "test_suite",
				Command: "true",
			},
			expectError: false,
		},
		{
			name: "failing false command",
			cfg: ContractConfig{
				Type:    "test_suite",
				Command: "false",
			},
			expectError:   true,
			errorContains: "test suite failed",
		},
		{
			name: "nonexistent command",
			cfg: ContractConfig{
				Type:    "test_suite",
				Command: "nonexistent_command_xyz_123",
			},
			expectError:   true,
			errorContains: "execution failed",
		},
		{
			name: "command with exit code 1",
			cfg: ContractConfig{
				Type:        "test_suite",
				Command:     "sh",
				CommandArgs: []string{"-c", "exit 1"},
			},
			expectError:   true,
			errorContains: "exit code 1",
		},
		{
			name: "command with exit code 2",
			cfg: ContractConfig{
				Type:        "test_suite",
				Command:     "sh",
				CommandArgs: []string{"-c", "exit 2"},
			},
			expectError:   true,
			errorContains: "exit code 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &testSuiteValidator{}
			workspacePath := t.TempDir()

			err := v.Validate(tt.cfg, workspacePath)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
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

// TestTestSuiteValidator_ValidationErrorDetails tests that validation errors include proper details.
func TestTestSuiteValidator_ValidationErrorDetails(t *testing.T) {
	tests := []struct {
		name           string
		cfg            ContractConfig
		expectedFields []string
	}{
		{
			name: "missing command error",
			cfg: ContractConfig{
				Type: "test_suite",
			},
			expectedFields: []string{"test_suite", "no command"},
		},
		{
			name: "command failure with stderr",
			cfg: ContractConfig{
				Type:        "test_suite",
				Command:     "sh",
				CommandArgs: []string{"-c", "echo 'test error' >&2; exit 1"},
			},
			expectedFields: []string{"test_suite", "test suite failed", "exit code 1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &testSuiteValidator{}
			workspacePath := t.TempDir()

			err := v.Validate(tt.cfg, workspacePath)
			if err == nil {
				t.Fatal("expected error but got none")
			}

			errStr := err.Error()
			for _, field := range tt.expectedFields {
				if !strings.Contains(errStr, field) {
					t.Errorf("error should contain %q, got: %s", field, errStr)
				}
			}

			// Verify it's a ValidationError
			validErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}

			if validErr.ContractType != "test_suite" {
				t.Errorf("expected contract type test_suite, got %s", validErr.ContractType)
			}
		})
	}
}

// TestTestSuiteValidator_WorkingDirectory tests that commands run in the correct working directory.
func TestTestSuiteValidator_WorkingDirectory(t *testing.T) {
	v := &testSuiteValidator{}
	workspacePath := t.TempDir()

	// Create a marker file in the workspace
	markerFile := filepath.Join(workspacePath, "marker.txt")
	if err := os.WriteFile(markerFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create marker file: %v", err)
	}

	// Command that checks for the marker file
	cfg := ContractConfig{
		Type:        "test_suite",
		Command:     "sh",
		CommandArgs: []string{"-c", "test -f marker.txt"},
	}

	err := v.Validate(cfg, workspacePath)
	if err != nil {
		t.Errorf("command should find marker.txt in working directory: %v", err)
	}
}

// TestTestSuiteValidator_CommandWithStdout tests that stdout output is captured.
func TestTestSuiteValidator_CommandWithStdout(t *testing.T) {
	v := &testSuiteValidator{}
	workspacePath := t.TempDir()

	// Command that outputs to stdout and fails
	cfg := ContractConfig{
		Type:        "test_suite",
		Command:     "sh",
		CommandArgs: []string{"-c", "echo 'stdout message'; exit 1"},
	}

	err := v.Validate(cfg, workspacePath)
	if err == nil {
		t.Fatal("expected error for failing command")
	}

	validErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	// Check that stdout was captured in details
	foundStdout := false
	for _, detail := range validErr.Details {
		if strings.Contains(detail, "stdout") {
			foundStdout = true
			break
		}
	}
	if !foundStdout {
		t.Error("expected stdout to be captured in error details")
	}
}

// TestTestSuiteValidator_CommandWithStderr tests that stderr output is captured.
func TestTestSuiteValidator_CommandWithStderr(t *testing.T) {
	v := &testSuiteValidator{}
	workspacePath := t.TempDir()

	// Command that outputs to stderr and fails
	cfg := ContractConfig{
		Type:        "test_suite",
		Command:     "sh",
		CommandArgs: []string{"-c", "echo 'error message' >&2; exit 1"},
	}

	err := v.Validate(cfg, workspacePath)
	if err == nil {
		t.Fatal("expected error for failing command")
	}

	validErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	// Check that stderr was captured in details
	foundStderr := false
	for _, detail := range validErr.Details {
		if strings.Contains(detail, "stderr") || strings.Contains(detail, "error message") {
			foundStderr = true
			break
		}
	}
	if !foundStderr {
		t.Error("expected stderr to be captured in error details")
	}
}

// TestTestSuiteValidator_ScriptExecution tests running a script file.
func TestTestSuiteValidator_ScriptExecution(t *testing.T) {
	v := &testSuiteValidator{}
	workspacePath := t.TempDir()

	// Create a test script
	scriptPath := filepath.Join(workspacePath, "test.sh")
	scriptContent := `#!/bin/sh
echo "Running tests..."
# Simulate some test output
echo "Test 1: PASS"
echo "Test 2: PASS"
exit 0
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to create test script: %v", err)
	}

	cfg := ContractConfig{
		Type:        "test_suite",
		Command:     "sh",
		CommandArgs: []string{"test.sh"},
	}

	err := v.Validate(cfg, workspacePath)
	if err != nil {
		t.Errorf("script execution should succeed: %v", err)
	}
}

// TestTestSuiteValidator_FailingScript tests a failing script file.
func TestTestSuiteValidator_FailingScript(t *testing.T) {
	v := &testSuiteValidator{}
	workspacePath := t.TempDir()

	// Create a failing test script
	scriptPath := filepath.Join(workspacePath, "test.sh")
	scriptContent := `#!/bin/sh
echo "Running tests..."
echo "Test 1: PASS"
echo "Test 2: FAIL" >&2
exit 1
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to create test script: %v", err)
	}

	cfg := ContractConfig{
		Type:        "test_suite",
		Command:     "sh",
		CommandArgs: []string{"test.sh"},
	}

	err := v.Validate(cfg, workspacePath)
	if err == nil {
		t.Fatal("expected error for failing script")
	}

	validErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if !validErr.Retryable {
		t.Error("test suite failures should be retryable")
	}
}

// TestTestSuiteValidator_LongOutput tests handling of long output.
func TestTestSuiteValidator_LongOutput(t *testing.T) {
	v := &testSuiteValidator{}
	workspacePath := t.TempDir()

	// Command that generates many lines of output
	cfg := ContractConfig{
		Type:        "test_suite",
		Command:     "sh",
		CommandArgs: []string{"-c", "for i in $(seq 1 50); do echo \"Line $i\"; done; exit 1"},
	}

	err := v.Validate(cfg, workspacePath)
	if err == nil {
		t.Fatal("expected error for failing command")
	}

	validErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	// Should have truncated output (last 10 lines)
	lineCount := 0
	for _, detail := range validErr.Details {
		if strings.HasPrefix(detail, "  Line") {
			lineCount++
		}
	}
	// Should not have all 50 lines
	if lineCount > 15 {
		t.Errorf("expected truncated output, got %d lines", lineCount)
	}
}

// TestExtractTestSuiteDetails tests the detail extraction helper.
func TestExtractTestSuiteDetails(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		args      []string
		stdout    string
		stderr    string
		minFields int
	}{
		{
			name:      "empty output",
			command:   "test",
			args:      []string{},
			stdout:    "",
			stderr:    "",
			minFields: 1, // At least command
		},
		{
			name:      "with stdout",
			command:   "npm",
			args:      []string{"test"},
			stdout:    "Test output line 1\nTest output line 2",
			stderr:    "",
			minFields: 3, // command + stdout header + lines
		},
		{
			name:      "with stderr",
			command:   "pytest",
			args:      []string{"-v"},
			stdout:    "",
			stderr:    "Error: test failed\nAssertionError",
			minFields: 3, // command + stderr header + lines
		},
		{
			name:      "with both",
			command:   "go",
			args:      []string{"test", "./..."},
			stdout:    "=== RUN TestExample\n--- PASS",
			stderr:    "warning: something",
			minFields: 5, // command + both headers + lines
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details := extractTestSuiteDetails(tt.command, tt.args, tt.stdout, tt.stderr)
			if len(details) < tt.minFields {
				t.Errorf("expected at least %d fields, got %d: %v", tt.minFields, len(details), details)
			}

			// First detail should contain the command
			if !strings.Contains(details[0], tt.command) {
				t.Errorf("first detail should contain command %q, got: %s", tt.command, details[0])
			}
		})
	}
}

// TestTestSuiteValidator_EnvironmentVariables tests that environment variables are available.
func TestTestSuiteValidator_EnvironmentVariables(t *testing.T) {
	v := &testSuiteValidator{}
	workspacePath := t.TempDir()

	// Set an environment variable and check it's available
	os.Setenv("TEST_SUITE_VAR", "test_value")
	defer os.Unsetenv("TEST_SUITE_VAR")

	cfg := ContractConfig{
		Type:        "test_suite",
		Command:     "sh",
		CommandArgs: []string{"-c", "test \"$TEST_SUITE_VAR\" = \"test_value\""},
	}

	err := v.Validate(cfg, workspacePath)
	if err != nil {
		t.Errorf("command should have access to environment variables: %v", err)
	}
}
