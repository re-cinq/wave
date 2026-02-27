package contract

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type typeScriptValidator struct{}

// tscAvailability caches the result of tsc availability check
var (
	tscAvailable     bool
	tscVersion       string
	tscCheckOnce     sync.Once
	tscCheckErr      error
)

func (v *typeScriptValidator) Validate(cfg ContractConfig, workspacePath string) error {
	// Check if TypeScript compiler is available
	available, version := CheckTypeScriptAvailability()
	if !available {
		if cfg.MustPass {
			return &ValidationError{
				ContractType: "typescript_interface",
				Message:      "TypeScript compiler (tsc) not available",
				Details: []string{
					"tsc command not found in PATH",
					"must_pass requires tsc to be installed",
					"install with: npm install -g typescript",
				},
				Retryable: false,
			}
		}
		// Graceful degradation - skip validation if tsc not available and must_pass is false
		return nil
	}

	contractPath := cfg.SchemaPath
	if contractPath == "" {
		return &ValidationError{
			ContractType: "typescript_interface",
			Message:      "no contract file path provided",
			Details:      []string{"specify 'schemaPath' with the path to your TypeScript interface file"},
			Retryable:    false,
		}
	}

	if _, err := os.Stat(contractPath); os.IsNotExist(err) {
		return &ValidationError{
			ContractType: "typescript_interface",
			Message:      fmt.Sprintf("contract file does not exist: %s", contractPath),
			Details:      []string{"ensure the TypeScript interface file exists at the specified path"},
			Retryable:    false,
		}
	}

	cmd := exec.Command("tsc", "--noEmit", "--strict", "--skipLibCheck", contractPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			details := extractTypeScriptErrors(string(output))
			details = append(details, fmt.Sprintf("tsc version: %s", version))
			return &ValidationError{
				ContractType: "typescript_interface",
				Message:      fmt.Sprintf("TypeScript validation failed (exit code %d)", exitError.ExitCode()),
				Details:      details,
				Retryable:    true,
			}
		}
		return &ValidationError{
			ContractType: "typescript_interface",
			Message:      "TypeScript validation failed",
			Details:      []string{err.Error()},
			Retryable:    false,
		}
	}

	if len(output) > 0 {
		details := extractTypeScriptErrors(string(output))
		return &ValidationError{
			ContractType: "typescript_interface",
			Message:      "TypeScript reported errors",
			Details:      details,
			Retryable:    true,
		}
	}

	return nil
}

// CheckTypeScriptAvailability checks if tsc is available and returns its version.
// The result is cached for performance.
func CheckTypeScriptAvailability() (available bool, version string) {
	tscCheckOnce.Do(func() {
		cmd := exec.Command("tsc", "--version")
		output, err := cmd.Output()
		if err != nil {
			tscCheckErr = err
			tscAvailable = false
			return
		}
		tscAvailable = true
		tscVersion = strings.TrimSpace(string(output))
	})
	return tscAvailable, tscVersion
}

// ResetTypeScriptAvailabilityCache resets the cached tsc availability check.
// This is primarily useful for testing.
func ResetTypeScriptAvailabilityCache() {
	tscCheckOnce = sync.Once{}
	tscAvailable = false
	tscVersion = ""
	tscCheckErr = nil
}

// extractTypeScriptErrors parses tsc output and extracts individual error messages.
func extractTypeScriptErrors(output string) []string {
	lines := strings.Split(output, "\n")
	details := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			details = append(details, line)
		}
	}
	if len(details) == 0 && output != "" {
		details = append(details, strings.TrimSpace(output))
	}
	return details
}
