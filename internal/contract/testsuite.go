package contract

import (
	"bytes"
	"fmt"
	"os/exec"
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

	cmd := exec.Command(command, args...)
	cmd.Dir = workspacePath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

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
				fmt.Sprintf("working directory: %s", workspacePath),
			},
			Retryable: false,
		}
	}

	// Note: Some test frameworks write to stderr even on success (e.g., progress output)
	// We only fail on non-zero exit code, not on stderr content
	return nil
}

// extractTestSuiteDetails formats test suite failure information.
func extractTestSuiteDetails(command string, args []string, stdout, stderr string) []string {
	details := make([]string, 0)

	details = append(details, fmt.Sprintf("command: %s %s", command, strings.Join(args, " ")))

	if stderr != "" {
		stderrLines := strings.Split(strings.TrimSpace(stderr), "\n")
		if len(stderrLines) > 10 {
			stderrLines = stderrLines[len(stderrLines)-10:]
			details = append(details, "stderr (last 10 lines):")
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
		if len(stdoutLines) > 10 {
			stdoutLines = stdoutLines[len(stdoutLines)-10:]
			details = append(details, "stdout (last 10 lines):")
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
