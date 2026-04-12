package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/adapter"
)

// TestVerdict represents the result of a single generated test case.
type TestVerdict struct {
	Name        string `json:"name"`        // Test case name
	Pass        bool   `json:"pass"`        // Whether the test passed
	Description string `json:"description"` // What the test checks
	Reason      string `json:"reason"`      // Reason for pass/fail
}

// SpecDerivedResult is the structured output from the test persona.
type SpecDerivedResult struct {
	Verdict string        `json:"verdict"` // "pass", "fail"
	Tests   []TestVerdict `json:"tests"`   // Individual test results
	Summary string        `json:"summary"` // One-sentence summary
}

// specDerivedTestResultSchema is injected into the test persona's prompt
// so the LLM knows exactly what JSON structure to produce.
const specDerivedTestResultSchema = `{
  "verdict": "pass" | "fail",
  "tests": [
    {"name": "<test name>", "pass": true|false, "description": "<what is tested>", "reason": "<why pass/fail>"}
  ],
  "summary": "<one-sentence summary>"
}`

// specDerivedValidator implements ContractValidator for the spec_derived_test type.
type specDerivedValidator struct{}

// Validate implements ContractValidator. For spec_derived_test, this is a no-op —
// callers must use ValidateSpecDerived instead, which provides the runner context.
func (v *specDerivedValidator) Validate(_ ContractConfig, _ string) error {
	return &ValidationError{
		ContractType: "spec_derived_test",
		Message:      "spec_derived_test contracts require an adapter runner — use ValidateSpecDerived()",
		Retryable:    false,
	}
}

// ValidateSpecDerived runs a spec_derived_test contract using the provided adapter runner.
// It loads the spec artifact, enforces persona separation, invokes the test persona
// to generate and run tests, then returns the structured result.
func ValidateSpecDerived(cfg ContractConfig, workspacePath string, runner adapter.AdapterRunner, manifest interface{}, implementerPersona string) (*SpecDerivedResult, error) {
	// Validate required config fields
	if err := validateSpecDerivedConfig(cfg); err != nil {
		return nil, err
	}

	// Enforce persona separation
	if err := checkPersonaSeparation(cfg.TestPersona, implementerPersona); err != nil {
		return nil, err
	}

	// Load spec artifact
	specContent, err := loadSpecArtifact(cfg.SpecArtifact, workspacePath)
	if err != nil {
		return nil, err
	}

	// Build test generation prompt
	prompt := buildSpecDerivedPrompt(specContent, cfg.ImplementationStep)

	// Resolve timeout
	timeout := 120 * time.Second
	if cfg.Timeout != "" {
		d, parseErr := time.ParseDuration(cfg.Timeout)
		if parseErr == nil {
			timeout = d
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	runCfg := adapter.AdapterRunConfig{
		Persona:       cfg.TestPersona,
		WorkspacePath: workspacePath,
		Prompt:        prompt,
		Model:         cfg.Model,
		Timeout:       timeout,
	}

	result, err := runner.Run(ctx, runCfg)
	if err != nil {
		return nil, &ValidationError{
			ContractType: "spec_derived_test",
			Message:      fmt.Sprintf("test persona failed: %v", err),
			Retryable:    true,
		}
	}

	// Read stdout
	var stdoutStr string
	if result.Stdout != nil {
		data, err := io.ReadAll(io.LimitReader(result.Stdout, 1<<20)) // 1MB cap
		if err != nil {
			return nil, &ValidationError{
				ContractType: "spec_derived_test",
				Message:      fmt.Sprintf("failed to read test persona output: %v", err),
				Retryable:    true,
			}
		}
		stdoutStr = string(data)
	}

	// Parse test results
	testResult, err := parseSpecDerivedResult(stdoutStr)
	if err != nil {
		return nil, err
	}

	return testResult, nil
}

// validateSpecDerivedConfig checks that all required config fields are present.
func validateSpecDerivedConfig(cfg ContractConfig) error {
	var missing []string
	if cfg.SpecArtifact == "" {
		missing = append(missing, "spec_artifact")
	}
	if cfg.TestPersona == "" {
		missing = append(missing, "test_persona")
	}
	if cfg.ImplementationStep == "" {
		missing = append(missing, "implementation_step")
	}
	if len(missing) > 0 {
		return &ValidationError{
			ContractType: "spec_derived_test",
			Message:      fmt.Sprintf("missing required field(s): %s", strings.Join(missing, ", ")),
			Details:      missing,
			Retryable:    false,
		}
	}
	return nil
}

// checkPersonaSeparation enforces that the test persona differs from the implementer.
func checkPersonaSeparation(testPersona, implementerPersona string) error {
	if testPersona == implementerPersona {
		return &ValidationError{
			ContractType: "spec_derived_test",
			Message:      fmt.Sprintf("test_persona %q must differ from implementer persona %q — persona separation is required", testPersona, implementerPersona),
			Retryable:    false,
		}
	}
	return nil
}

// loadSpecArtifact reads the spec artifact file with path traversal protection.
func loadSpecArtifact(specArtifact, workspacePath string) (string, error) {
	// Path traversal protection
	cleanPath := filepath.Clean(specArtifact)
	if strings.Contains(cleanPath, "..") {
		return "", &ValidationError{
			ContractType: "spec_derived_test",
			Message:      fmt.Sprintf("spec_artifact %q contains path traversal", specArtifact),
			Retryable:    false,
		}
	}

	// Resolve relative to workspace
	fullPath := cleanPath
	if !filepath.IsAbs(cleanPath) {
		fullPath = filepath.Join(workspacePath, cleanPath)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", &ValidationError{
			ContractType: "spec_derived_test",
			Message:      fmt.Sprintf("failed to read spec artifact %q: %v", specArtifact, err),
			Retryable:    false,
		}
	}

	if len(data) == 0 {
		return "", &ValidationError{
			ContractType: "spec_derived_test",
			Message:      fmt.Sprintf("spec artifact %q is empty", specArtifact),
			Retryable:    false,
		}
	}

	return string(data), nil
}

// buildSpecDerivedPrompt assembles the prompt for the test persona.
func buildSpecDerivedPrompt(specContent, implementationStep string) string {
	var b strings.Builder
	b.WriteString("## Specification\n\n")
	b.WriteString(specContent)
	b.WriteString("\n\n")
	b.WriteString("## Task\n\n")
	b.WriteString(fmt.Sprintf("You are a test author. Based on the specification above, generate test cases for the implementation in step %q.\n", implementationStep))
	b.WriteString("Independently derive tests from the specification — do NOT rely on the implementation code.\n")
	b.WriteString("Run the tests against the workspace and report results.\n\n")
	b.WriteString("## Required Output Format\n\n")
	b.WriteString("You MUST respond with a single JSON object matching this schema:\n\n")
	b.WriteString("```json\n")
	b.WriteString(specDerivedTestResultSchema)
	b.WriteString("\n```\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- verdict MUST be exactly \"pass\" or \"fail\"\n")
	b.WriteString("- tests MUST be an array of test results\n")
	b.WriteString("- Each test must have name, pass (bool), description, and reason\n")
	b.WriteString("- Return ONLY the JSON object, no other text outside the JSON block\n")
	return b.String()
}

// parseSpecDerivedResult extracts SpecDerivedResult from test persona stdout.
func parseSpecDerivedResult(stdout string) (*SpecDerivedResult, error) {
	if stdout == "" {
		return nil, &ValidationError{
			ContractType: "spec_derived_test",
			Message:      "test persona produced no output",
			Retryable:    true,
		}
	}

	cleaned := extractJSON(stdout)
	var result SpecDerivedResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, &ValidationError{
			ContractType: "spec_derived_test",
			Message:      "failed to parse SpecDerivedResult from test persona output",
			Details:      []string{err.Error(), stdout},
			Retryable:    true,
		}
	}

	// Validate verdict enum
	switch result.Verdict {
	case "pass", "fail":
		// valid
	default:
		return nil, &ValidationError{
			ContractType: "spec_derived_test",
			Message:      fmt.Sprintf("invalid verdict %q (must be pass or fail)", result.Verdict),
			Details:      []string{stdout},
			Retryable:    true,
		}
	}

	return &result, nil
}
