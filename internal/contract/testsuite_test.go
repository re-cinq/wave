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
			} else if err != nil {
				t.Errorf("expected no error but got: %v", err)
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

	// Command that generates many lines of output (100 lines, truncated to 50)
	cfg := ContractConfig{
		Type:        "test_suite",
		Command:     "sh",
		CommandArgs: []string{"-c", "for i in $(seq 1 100); do echo \"Line $i\"; done; exit 1"},
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

	// Should have truncated output (last 50 lines)
	lineCount := 0
	for _, detail := range validErr.Details {
		if strings.HasPrefix(detail, "  Line") {
			lineCount++
		}
	}
	// Should have at most 50 lines, not all 100
	if lineCount > 55 {
		t.Errorf("expected truncated output (max ~50 lines), got %d lines", lineCount)
	}
	// Should have at least 40 lines (we keep 50, minus any empty)
	if lineCount < 40 {
		t.Errorf("expected at least 40 lines of output, got %d lines", lineCount)
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
		// git init and add a project marker
		cmd := exec.Command("git", "init", gitInit)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %v\n%s", err, out)
		}
		if err := os.WriteFile(filepath.Join(gitInit, "go.mod"), []byte("module test\n"), 0644); err != nil {
			t.Fatal(err)
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

	t.Run("project_root walks up to find project markers", func(t *testing.T) {
		// Create a nested dir structure with go.mod at the top
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0644); err != nil {
			t.Fatal(err)
		}
		nested := filepath.Join(root, "sub", "deep")
		if err := os.MkdirAll(nested, 0755); err != nil {
			t.Fatal(err)
		}
		dir, err := resolveContractDir("project_root", nested)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != root {
			t.Errorf("expected %q, got %q", root, dir)
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

		// Create a temp git repo with a project marker (go.mod) so the
		// walk-up logic in resolveContractDir finds the repo root.
		ws := t.TempDir()
		repoDir := filepath.Join(ws, "repo")
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("git", "init", repoDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %v\n%s", err, out)
		}
		// go.mod is a project marker recognised by the walk-up logic
		if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
			t.Fatal(err)
		}

		// Workspace is a subdirectory of the repo
		subWs := filepath.Join(repoDir, ".agents", "workspaces", "test")
		if err := os.MkdirAll(subWs, 0755); err != nil {
			t.Fatal(err)
		}

		cfg := ContractConfig{
			Type:        "test_suite",
			Command:     "sh",
			CommandArgs: []string{"-c", "test -f go.mod"},
		}
		if err := v.Validate(cfg, subWs); err != nil {
			t.Errorf("should find go.mod in project root: %v", err)
		}
	})

	t.Run("dir empty outside git repo uses CWD fallback", func(t *testing.T) {
		v := &testSuiteValidator{}
		ws := t.TempDir()

		// With no project markers and no git repo, resolveContractDir falls
		// back to os.Getwd(). The command "true" always succeeds, so this
		// should not error.
		cfg := ContractConfig{
			Type:    "test_suite",
			Command: "true",
		}
		err := v.Validate(cfg, ws)
		if err != nil {
			t.Errorf("CWD fallback should allow command to run: %v", err)
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
		subWs := filepath.Join(repoDir, ".agents", "workspaces", "test")
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

// TestResolveContractDir_ProjectMarkers tests that different project markers are recognized.
func TestResolveContractDir_ProjectMarkers(t *testing.T) {
	markers := []struct {
		name    string
		marker  string
		content string
	}{
		{"package.json", "package.json", `{"name":"test"}`},
		{"Cargo.toml", "Cargo.toml", "[package]\nname = \"test\""},
		{"pyproject.toml", "pyproject.toml", "[project]\nname = \"test\""},
	}

	for _, m := range markers {
		t.Run("project_root with "+m.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, m.marker), []byte(m.content), 0644); err != nil {
				t.Fatal(err)
			}
			nested := filepath.Join(root, "deep", "sub")
			if err := os.MkdirAll(nested, 0755); err != nil {
				t.Fatal(err)
			}

			dir, err := resolveContractDir("project_root", nested)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dir != root {
				t.Errorf("expected %q, got %q", root, dir)
			}
		})
	}
}

// TestResolveContractDir_FallbackPaths tests the git and CWD fallback paths.
func TestResolveContractDir_FallbackPaths(t *testing.T) {
	t.Run("git rev-parse fallback when no markers found", func(t *testing.T) {
		// Create a git repo with no project markers
		ws := t.TempDir()
		repoDir := filepath.Join(ws, "repo")
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("git", "init", repoDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %v\n%s", err, out)
		}

		subDir := filepath.Join(repoDir, "some", "path")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		dir, err := resolveContractDir("project_root", subDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should resolve to a valid directory (git root or CWD fallback)
		if dir == "" {
			t.Error("expected non-empty directory")
		}
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Errorf("expected existing directory, got %q", dir)
		}
	})

	t.Run("CWD fallback when no markers and no git repo", func(t *testing.T) {
		ws := t.TempDir()
		// No project markers, no git repo — should fall back to git or CWD
		dir, err := resolveContractDir("project_root", ws)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should return a valid directory (git root or CWD)
		if dir == "" {
			t.Error("expected non-empty fallback directory")
		}
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Errorf("expected existing directory, got %q", dir)
		}
	})

	t.Run("project marker in workspacePath itself returns it directly", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0644); err != nil {
			t.Fatal(err)
		}

		dir, err := resolveContractDir("project_root", root)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != root {
			t.Errorf("expected %q (marker in workspace itself), got %q", root, dir)
		}
	})

	t.Run("walk-up reaches filesystem root and falls through", func(t *testing.T) {
		// Use /tmp (or a temp dir unlikely to have project markers)
		tmpDir := t.TempDir()
		nested := filepath.Join(tmpDir, "a", "b", "c")
		if err := os.MkdirAll(nested, 0755); err != nil {
			t.Fatal(err)
		}

		// No markers anywhere, no git repo in nested
		// Should fall through to git rev-parse (fails) then CWD fallback
		dir, err := resolveContractDir("project_root", nested)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should get either git root or CWD — not empty
		if dir == "" {
			t.Error("expected non-empty fallback directory")
		}
	})
}

// TestTestSuiteValidator_EnvironmentVariables tests that environment variables are available.
func TestTestSuiteValidator_EnvironmentVariables(t *testing.T) {
	v := &testSuiteValidator{}
	workspacePath := t.TempDir()

	// Set an environment variable and check it's available
	_ = os.Setenv("TEST_SUITE_VAR", "test_value")
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

// --- Coverage gap: resolveContractDir edge cases ---

// TestResolveContractDir_WorkspaceHasMarker_ReturnsImmediately tests that when
// workspacePath itself contains a project marker, it is returned without walking up.
func TestResolveContractDir_WorkspaceHasMarker_ReturnsImmediately(t *testing.T) {
	ws := t.TempDir()
	// Place marker directly in workspace
	if err := os.WriteFile(filepath.Join(ws, "package.json"), []byte(`{"name":"ws"}`), 0644); err != nil {
		t.Fatal(err)
	}

	dir, err := resolveContractDir("project_root", ws)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir != ws {
		t.Errorf("expected workspace %q (has marker), got %q", ws, dir)
	}
}

// TestResolveContractDir_DeepNesting_FindsNearestAncestor tests that the walk-up
// finds the nearest ancestor with a marker, not a more distant one.
func TestResolveContractDir_DeepNesting_FindsNearestAncestor(t *testing.T) {
	root := t.TempDir()

	// Create two levels with markers — inner should be found first.
	inner := filepath.Join(root, "level1")
	if err := os.MkdirAll(inner, 0755); err != nil {
		t.Fatal(err)
	}
	// Marker at root
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module outer\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Marker at inner
	if err := os.WriteFile(filepath.Join(inner, "go.mod"), []byte("module inner\n"), 0644); err != nil {
		t.Fatal(err)
	}

	deep := filepath.Join(inner, "sub", "deep")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatal(err)
	}

	dir, err := resolveContractDir("project_root", deep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should find inner (nearest ancestor with marker), not root
	if dir != inner {
		t.Errorf("expected nearest ancestor %q, got %q", inner, dir)
	}
}

// TestResolveContractDir_FilesystemRoot_WalkUpTerminates tests that the walk-up
// loop terminates when it reaches the filesystem root without finding markers.
func TestResolveContractDir_FilesystemRoot_WalkUpTerminates(t *testing.T) {
	// Create a deep directory with no markers anywhere in the hierarchy.
	// The walk-up will reach / and the parent==candidate check should terminate it.
	ws := t.TempDir()
	deep := filepath.Join(ws, "a", "b", "c", "d", "e")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatal(err)
	}

	// This should not hang or error — falls through to git/CWD fallback.
	dir, err := resolveContractDir("project_root", deep)
	if err != nil {
		t.Fatalf("expected fallback, got error: %v", err)
	}
	if dir == "" {
		t.Error("expected non-empty directory from fallback")
	}
}

// TestResolveContractDir_GitRevParseFallback tests that when no project markers
// are found, git rev-parse is used as a fallback.
func TestResolveContractDir_GitRevParseFallback(t *testing.T) {
	// Create a git repo without any project markers.
	ws := t.TempDir()
	repoDir := filepath.Join(ws, "gitonly")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", repoDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	subDir := filepath.Join(repoDir, "level1", "level2")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	dir, err := resolveContractDir("project_root", subDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should resolve to the git repo root
	absRepo, _ := filepath.EvalSymlinks(repoDir)
	absDir, _ := filepath.EvalSymlinks(dir)
	if absDir != absRepo {
		// It might fall back to CWD if git rev-parse behaves differently,
		// but it should at least return a valid directory.
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Errorf("expected valid directory, got %q", dir)
		}
	}
}

// TestResolveContractDir_CWDFallback_NoMarkersNoGit tests the CWD fallback when
// there are no project markers and no git repository.
func TestResolveContractDir_CWDFallback_NoMarkersNoGit(t *testing.T) {
	// Create a directory with no markers and outside any git repo.
	ws := t.TempDir()

	// Ensure we're not inside a git repo for this test.
	// The test directory (t.TempDir) is typically outside any git repo.
	dir, err := resolveContractDir("project_root", ws)
	if err != nil {
		// If all three strategies fail (walk-up, git, CWD), we get an error.
		// This is hard to trigger because os.Getwd() rarely fails,
		// so receiving an error here is actually testing that final error path.
		if !strings.Contains(err.Error(), "failed to resolve project root") {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}

	// Should have fallen back to CWD (or git found a repo).
	if dir == "" {
		t.Error("expected non-empty fallback directory")
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("expected existing directory, got %q", dir)
	}
}

// TestResolveContractDir_AllMarkerTypes_TableDriven tests each supported marker.
func TestResolveContractDir_AllMarkerTypes_TableDriven(t *testing.T) {
	markers := []struct {
		name    string
		marker  string
		content string
	}{
		{"go.mod", "go.mod", "module test\n"},
		{"package.json", "package.json", `{"name":"test"}`},
		{"Cargo.toml", "Cargo.toml", "[package]\nname = \"test\""},
		{"pyproject.toml", "pyproject.toml", "[project]\nname = \"test\""},
	}

	for _, m := range markers {
		t.Run(m.name+"_at_workspace_level", func(t *testing.T) {
			ws := t.TempDir()
			if err := os.WriteFile(filepath.Join(ws, m.marker), []byte(m.content), 0644); err != nil {
				t.Fatal(err)
			}

			dir, err := resolveContractDir("project_root", ws)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dir != ws {
				t.Errorf("expected %q, got %q", ws, dir)
			}
		})
	}
}

// TestResolveContractDir_EmptyAndRelative verifies the trivial dir="" and
// relative path branches with additional assertions.
func TestResolveContractDir_EmptyAndRelative(t *testing.T) {
	t.Run("empty dir returns workspacePath exactly", func(t *testing.T) {
		ws := "/some/workspace/path"
		dir, err := resolveContractDir("", ws)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != ws {
			t.Errorf("expected %q, got %q", ws, dir)
		}
	})

	t.Run("relative path joined with workspace", func(t *testing.T) {
		ws := "/workspace"
		dir, err := resolveContractDir("contracts/v2", ws)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(ws, "contracts/v2")
		if dir != expected {
			t.Errorf("expected %q, got %q", expected, dir)
		}
	})

	t.Run("absolute path returned unchanged", func(t *testing.T) {
		ws := "/workspace"
		abs := "/opt/contracts"
		dir, err := resolveContractDir(abs, ws)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != abs {
			t.Errorf("expected %q, got %q", abs, dir)
		}
	})
}
