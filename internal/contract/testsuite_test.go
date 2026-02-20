package contract

import (
	"os"
	"os/exec"
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
		{
			name: "command string parsing - echo hello world",
			cfg: ContractConfig{
				Type:    "test_suite",
				Command: "echo hello world",
			},
			expectError: false,
		},
		{
			name: "command string parsing - true with no args",
			cfg: ContractConfig{
				Type:    "test_suite",
				Command: "true",
			},
			expectError: false,
		},
		{
			name: "command string parsing - failing command",
			cfg: ContractConfig{
				Type:    "test_suite",
				Command: "sh -c exit\\ 1",
			},
			expectError:   true,
			errorContains: "test suite failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &testSuiteValidator{}
			workspacePath := t.TempDir()

			// Set Dir to workspace so tests don't need a git repo
			// (the default for test_suite is project_root)
			cfg := tt.cfg
			if cfg.Dir == "" && cfg.Command != "" {
				cfg.Dir = workspacePath
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

			// Set Dir to workspace so tests don't need a git repo
			cfg := tt.cfg
			if cfg.Dir == "" && cfg.Command != "" {
				cfg.Dir = workspacePath
			}

			err := v.Validate(cfg, workspacePath)
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

	// Command that checks for the marker file — use Dir to run in workspace
	cfg := ContractConfig{
		Type:        "test_suite",
		Command:     "sh",
		CommandArgs: []string{"-c", "test -f marker.txt"},
		Dir:         workspacePath,
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
		Dir:         workspacePath,
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
		Dir:         workspacePath,
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
		Dir:         workspacePath,
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
		Dir:         workspacePath,
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
		Dir:         workspacePath,
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

// TestResolveContractDir tests the working directory resolution logic.
func TestResolveContractDir(t *testing.T) {
	t.Run("empty dir uses workspace", func(t *testing.T) {
		ws := t.TempDir()
		dir, err := resolveContractDir("", ws)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != ws {
			t.Errorf("expected %q, got %q", ws, dir)
		}
	})

	t.Run("project_root resolves git toplevel", func(t *testing.T) {
		// Create a temp git repo for this test
		ws := t.TempDir()
		gitInit := filepath.Join(ws, "repo")
		if err := os.MkdirAll(gitInit, 0755); err != nil {
			t.Fatal(err)
		}
		// git init the temp dir
		cmd := exec.Command("git", "init", gitInit)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %v\n%s", err, out)
		}

		subDir := filepath.Join(gitInit, "sub", "dir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		dir, err := resolveContractDir("project_root", subDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != gitInit {
			t.Errorf("expected %q, got %q", gitInit, dir)
		}
	})

	t.Run("project_root fails outside git repo", func(t *testing.T) {
		ws := t.TempDir()
		_, err := resolveContractDir("project_root", ws)
		if err == nil {
			t.Error("expected error outside git repo")
		}
		if !strings.Contains(err.Error(), "git repo") {
			t.Errorf("error should mention git repo, got: %v", err)
		}
	})

	t.Run("absolute path used as-is", func(t *testing.T) {
		ws := t.TempDir()
		absDir := t.TempDir()
		dir, err := resolveContractDir(absDir, ws)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != absDir {
			t.Errorf("expected %q, got %q", absDir, dir)
		}
	})

	t.Run("relative path resolved against workspace", func(t *testing.T) {
		ws := t.TempDir()
		subDir := filepath.Join(ws, "subdir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}
		dir, err := resolveContractDir("subdir", ws)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != subDir {
			t.Errorf("expected %q, got %q", subDir, dir)
		}
	})
}

// TestTestSuiteValidator_DirField tests that the Dir config field controls where commands run.
func TestTestSuiteValidator_DirField(t *testing.T) {
	t.Run("dir empty defaults to project_root", func(t *testing.T) {
		v := &testSuiteValidator{}

		// Create a temp git repo with a marker file
		ws := t.TempDir()
		repoDir := filepath.Join(ws, "repo")
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("git", "init", repoDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %v\n%s", err, out)
		}
		if err := os.WriteFile(filepath.Join(repoDir, "marker.txt"), []byte("root"), 0644); err != nil {
			t.Fatal(err)
		}

		// Workspace is a subdirectory of the repo
		subWs := filepath.Join(repoDir, ".wave", "workspaces", "test")
		if err := os.MkdirAll(subWs, 0755); err != nil {
			t.Fatal(err)
		}

		cfg := ContractConfig{
			Type:        "test_suite",
			Command:     "sh",
			CommandArgs: []string{"-c", "test -f marker.txt"},
		}
		if err := v.Validate(cfg, subWs); err != nil {
			t.Errorf("should find marker in project root: %v", err)
		}
	})

	t.Run("dir empty fails outside git repo", func(t *testing.T) {
		v := &testSuiteValidator{}
		ws := t.TempDir()

		cfg := ContractConfig{
			Type:    "test_suite",
			Command: "true",
		}
		err := v.Validate(cfg, ws)
		if err == nil {
			t.Error("expected error outside git repo with empty dir")
		}
	})

	t.Run("dir absolute runs in specified dir", func(t *testing.T) {
		v := &testSuiteValidator{}
		ws := t.TempDir()
		targetDir := t.TempDir()

		// Create marker in target, NOT workspace
		if err := os.WriteFile(filepath.Join(targetDir, "target-marker.txt"), []byte("target"), 0644); err != nil {
			t.Fatal(err)
		}

		cfg := ContractConfig{
			Type:        "test_suite",
			Command:     "sh",
			CommandArgs: []string{"-c", "test -f target-marker.txt"},
			Dir:         targetDir,
		}
		if err := v.Validate(cfg, ws); err != nil {
			t.Errorf("should find marker in target dir: %v", err)
		}
	})

	t.Run("dir project_root resolves git root", func(t *testing.T) {
		v := &testSuiteValidator{}

		// Create a temp git repo with a marker file
		ws := t.TempDir()
		repoDir := filepath.Join(ws, "repo")
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("git", "init", repoDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %v\n%s", err, out)
		}
		if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module test"), 0644); err != nil {
			t.Fatal(err)
		}

		// Workspace is a subdirectory of the repo
		subWs := filepath.Join(repoDir, ".wave", "workspaces", "test")
		if err := os.MkdirAll(subWs, 0755); err != nil {
			t.Fatal(err)
		}

		cfg := ContractConfig{
			Type:        "test_suite",
			Command:     "sh",
			CommandArgs: []string{"-c", "test -f go.mod"},
			Dir:         "project_root",
		}
		if err := v.Validate(cfg, subWs); err != nil {
			t.Errorf("should find go.mod in project root: %v", err)
		}
	})
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
		Dir:         workspacePath,
	}

	err := v.Validate(cfg, workspacePath)
	if err != nil {
		t.Errorf("command should have access to environment variables: %v", err)
	}
}

// =============================================================================
// T020: User Story 7 - Test suite exit code test
// =============================================================================

// T020: Test that non-zero exit codes cause contract validation to fail
// This ensures that when a test command returns non-zero exit code, the contract fails.
func TestTestSuiteValidator_ExitCodeFailure(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		commandArgs    []string
		expectError    bool
		expectedExitCode int
		errorContains  string
	}{
		{
			name:             "false command exits with 1",
			command:          "false",
			commandArgs:      nil,
			expectError:      true,
			expectedExitCode: 1,
			errorContains:    "test suite failed",
		},
		{
			name:             "explicit exit 1",
			command:          "sh",
			commandArgs:      []string{"-c", "exit 1"},
			expectError:      true,
			expectedExitCode: 1,
			errorContains:    "exit code 1",
		},
		{
			name:             "explicit exit 2",
			command:          "sh",
			commandArgs:      []string{"-c", "exit 2"},
			expectError:      true,
			expectedExitCode: 2,
			errorContains:    "exit code 2",
		},
		{
			name:             "explicit exit 42",
			command:          "sh",
			commandArgs:      []string{"-c", "exit 42"},
			expectError:      true,
			expectedExitCode: 42,
			errorContains:    "exit code 42",
		},
		{
			name:             "explicit exit 127 (command not found convention)",
			command:          "sh",
			commandArgs:      []string{"-c", "exit 127"},
			expectError:      true,
			expectedExitCode: 127,
			errorContains:    "exit code 127",
		},
		{
			name:             "explicit exit 255",
			command:          "sh",
			commandArgs:      []string{"-c", "exit 255"},
			expectError:      true,
			expectedExitCode: 255,
			errorContains:    "exit code 255",
		},
		{
			name:        "true command exits with 0 (success)",
			command:     "true",
			commandArgs: nil,
			expectError: false,
		},
		{
			name:        "explicit exit 0 (success)",
			command:     "sh",
			commandArgs: []string{"-c", "exit 0"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &testSuiteValidator{}
			workspacePath := t.TempDir()

			cfg := ContractConfig{
				Type:        "test_suite",
				Command:     tt.command,
				CommandArgs: tt.commandArgs,
				Dir:         workspacePath,
			}

			err := v.Validate(cfg, workspacePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for exit code %d, but validation passed", tt.expectedExitCode)
					return
				}

				// Verify it's a ValidationError
				validErr, ok := err.(*ValidationError)
				if !ok {
					t.Errorf("expected ValidationError, got %T", err)
					return
				}

				// Verify contract type
				if validErr.ContractType != "test_suite" {
					t.Errorf("expected contract type test_suite, got %s", validErr.ContractType)
				}

				// Verify error message contains expected text
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error should contain %q, got: %v", tt.errorContains, err)
				}

				// Verify the error is marked as retryable (test failures can be fixed)
				if !validErr.Retryable {
					t.Error("test suite exit code failures should be retryable")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for exit code 0, but got: %v", err)
				}
			}
		})
	}
}

// TestTestSuiteValidator_ExitCodeWithOutput verifies that exit codes take precedence
// over output content - a failing command with "successful" looking output should still fail.
func TestTestSuiteValidator_ExitCodeWithOutput(t *testing.T) {
	v := &testSuiteValidator{}
	workspacePath := t.TempDir()

	// Command that prints "success" but exits with non-zero
	cfg := ContractConfig{
		Type:        "test_suite",
		Command:     "sh",
		CommandArgs: []string{"-c", "echo 'All tests passed!'; exit 1"},
		Dir:         workspacePath,
	}

	err := v.Validate(cfg, workspacePath)
	if err == nil {
		t.Error("expected error despite 'success' message in output")
	}

	validErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if !strings.Contains(validErr.Error(), "exit code 1") {
		t.Errorf("error should mention exit code, got: %v", validErr)
	}
}

// TestTestSuiteValidator_ExitCodeZeroWithErrorOutput verifies that exit code 0 succeeds
// even when stderr has content - some tools write to stderr for non-error output.
func TestTestSuiteValidator_ExitCodeZeroWithErrorOutput(t *testing.T) {
	v := &testSuiteValidator{}
	workspacePath := t.TempDir()

	// Command that writes to stderr but exits with 0
	cfg := ContractConfig{
		Type:        "test_suite",
		Command:     "sh",
		CommandArgs: []string{"-c", "echo 'Warning: deprecated feature' >&2; exit 0"},
		Dir:         workspacePath,
	}

	err := v.Validate(cfg, workspacePath)
	if err != nil {
		t.Errorf("exit code 0 should succeed even with stderr output: %v", err)
	}
}
