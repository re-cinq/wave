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
		return fmt.Errorf("no command configured for test suite validation")
	}

	var args []string
	if len(cfg.CommandArgs) > 0 {
		args = cfg.CommandArgs
	}

	cmd := exec.Command(cfg.Command, args...)
	cmd.Dir = workspacePath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("test suite failed (exit code %d): %s", exitError.ExitCode(), strings.TrimSpace(stderr.String()))
		}
		return fmt.Errorf("test suite execution failed: %w", err)
	}

	if stderr.Len() > 0 {
		return fmt.Errorf("test suite stderr: %s", strings.TrimSpace(stderr.String()))
	}

	return nil
}
