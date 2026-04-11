package contract

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type testSuiteValidator struct{}

func (v *testSuiteValidator) Validate(cfg ContractConfig, workspacePath string) error {
	if cfg.Command == "" {
		return &ValidationError{
			ContractType: "test_suite",
			Message:      "no command configured for test suite validation",
			Details:      []string{"specify 'command' with the test runner to execute"},
			Retryable:    false,
		}
	}

	// Detect unresolved template variables (e.g. {{ project.test_command }}).
	// These indicate missing project configuration in wave.yaml.
	if strings.Contains(cfg.Command, "{{ ") || strings.Contains(cfg.Command, "{{") {
		return &ValidationError{
			ContractType: "test_suite",
			Message:      "unresolved template variable in contract command — configure project section in wave.yaml",
			Details: []string{
				fmt.Sprintf("command: %s", cfg.Command),
				"add the missing variable to the project section of your wave.yaml",
			},
			Retryable: false,
		}
	}

	var command string
	var args []string

	if len(cfg.CommandArgs) > 0 {
		// Explicit command and args specified
		command = cfg.Command
		args = cfg.CommandArgs
	} else {
		// Parse command string into command and args
		// This allows users to write "go test ./... -v" as the command
		parts := strings.Fields(cfg.Command)
		if len(parts) == 0 {
			return &ValidationError{
				ContractType: "test_suite",
				Message:      "empty command for test suite validation",
				Details:      []string{"specify 'command' with the test runner to execute"},
				Retryable:    false,
			}
		}
		command = parts[0]
		if len(parts) > 1 {
			args = parts[1:]
		}
	}

	// Resolve working directory — default to project_root for test_suite
	// since tests almost always need the actual project context (go.mod, package.json, etc.)
	contractDir := cfg.Dir
	if contractDir == "" {
		contractDir = "project_root"
	}
	dir, err := resolveContractDir(contractDir, workspacePath)
	if err != nil {
		return &ValidationError{
			ContractType: "test_suite",
			Message:      fmt.Sprintf("failed to resolve working directory: %v", err),
			Details:      []string{fmt.Sprintf("dir: %s", cfg.Dir), fmt.Sprintf("workspace: %s", workspacePath)},
			Retryable:    false,
		}
	}

	cmd := exec.Command(command, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			details := extractTestSuiteDetails(command, args, stdout.String(), stderr.String())
			return &ValidationError{
				ContractType: "test_suite",
				Message:      fmt.Sprintf("test suite failed (exit code %d)", exitError.ExitCode()),
				Details:      details,
				Retryable:    true,
			}
		}
		return &ValidationError{
			ContractType: "test_suite",
			Message:      "test suite execution failed",
			Details: []string{
				err.Error(),
				fmt.Sprintf("command: %s %s", command, strings.Join(args, " ")),
				fmt.Sprintf("working directory: %s", dir),
			},
			Retryable: false,
		}
	}

	// Note: Some test frameworks write to stderr even on success (e.g., progress output)
	// We only fail on non-zero exit code, not on stderr content
	return nil
}

// resolveContractDir resolves the working directory for contract command execution.
//   - Empty: use workspacePath
//   - "project_root": resolve via git rev-parse --show-toplevel
//   - Absolute path: use as-is
//   - Relative path: resolve relative to workspacePath
func resolveContractDir(dir, workspacePath string) (string, error) {
	if dir == "" {
		return workspacePath, nil
	}

	if dir == "project_root" {
		// Walk up from workspacePath to find the real project root.
		// Workspace dirs often have their own git init (for Claude Code path anchoring),
		// so git rev-parse may return the workspace dir instead of the actual project root.
		// Look for project markers (go.mod, package.json, etc.) to find the real root.
		projectMarkers := []string{"go.mod", "package.json", "Cargo.toml", "pyproject.toml", ".git"}
		candidate := workspacePath
		for {
			for _, marker := range projectMarkers {
				if _, err := os.Stat(filepath.Join(candidate, marker)); err == nil {
					return candidate, nil
				}
			}
			parent := filepath.Dir(candidate)
			if parent == candidate {
				break // reached filesystem root
			}
			candidate = parent
		}
		// Fallback: try git rev-parse (works when project has no marker files)
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		cmd.Dir = workspacePath
		if out, err := cmd.Output(); err == nil {
			return strings.TrimSpace(string(out)), nil
		}
		// Last resort: use process CWD
		if cwd, err := os.Getwd(); err == nil {
			return cwd, nil
		}
		return "", fmt.Errorf("failed to resolve project root: no project markers or git repo found")
	}

	if filepath.IsAbs(dir) {
		return dir, nil
	}

	return filepath.Join(workspacePath, dir), nil
}

// extractTestSuiteDetails formats test suite failure information.
func extractTestSuiteDetails(command string, args []string, stdout, stderr string) []string {
	details := make([]string, 0)

	details = append(details, fmt.Sprintf("command: %s %s", command, strings.Join(args, " ")))

	if stderr != "" {
		stderrLines := strings.Split(strings.TrimSpace(stderr), "\n")
		if len(stderrLines) > 50 {
			stderrLines = stderrLines[len(stderrLines)-50:]
			details = append(details, "stderr (last 50 lines):")
		} else {
			details = append(details, "stderr:")
		}
		for _, line := range stderrLines {
			if line = strings.TrimSpace(line); line != "" {
				details = append(details, "  "+line)
			}
		}
	}

	if stdout != "" {
		stdoutLines := strings.Split(strings.TrimSpace(stdout), "\n")
		if len(stdoutLines) > 50 {
			stdoutLines = stdoutLines[len(stdoutLines)-50:]
			details = append(details, "stdout (last 50 lines):")
		} else {
			details = append(details, "stdout:")
		}
		for _, line := range stdoutLines {
			if line = strings.TrimSpace(line); line != "" {
				details = append(details, "  "+line)
			}
		}
	}

	return details
}
