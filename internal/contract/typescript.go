package contract

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type typeScriptValidator struct{}

func (v *typeScriptValidator) Validate(cfg ContractConfig, workspacePath string) error {
	// Check if TypeScript compiler is available
	if !IsTypeScriptAvailable() {
		if cfg.StrictMode {
			return fmt.Errorf("TypeScript compiler (tsc) not available in PATH and strict mode is enabled")
		}
		// Graceful degradation - skip validation if tsc not available and not in strict mode
		return nil
	}

	contractPath := cfg.SchemaPath
	if contractPath == "" {
		return fmt.Errorf("no contract file path provided")
	}

	if _, err := os.Stat(contractPath); os.IsNotExist(err) {
		return fmt.Errorf("contract file does not exist: %s", contractPath)
	}

	cmd := exec.Command("tsc", "--noEmit", "--strict", "--skipLibCheck", contractPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("TypeScript validation failed (exit code %d): %s", exitError.ExitCode(), strings.TrimSpace(string(output)))
		}
		return fmt.Errorf("TypeScript validation failed: %w", err)
	}

	if len(output) > 0 {
		return fmt.Errorf("TypeScript errors: %s", strings.TrimSpace(string(output)))
	}

	return nil
}

func IsTypeScriptAvailable() bool {
	cmd := exec.Command("tsc", "--version")
	return cmd.Run() == nil
}
