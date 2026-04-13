package contract

import (
	"fmt"
	"strings"
	"time"
)

// ContractConfig defines the configuration for contract validation.
// It is the canonical type for contract configuration used by both the
// pipeline (via YAML parsing) and the contract validation engine (via JSON/runtime).
type ContractConfig struct {
	Type        string   `json:"type"                    yaml:"type"`
	Source      string   `json:"source,omitempty"        yaml:"source,omitempty"`
	Schema      string   `json:"schema,omitempty"        yaml:"schema,omitempty"`
	SchemaPath  string   `json:"schemaPath,omitempty"    yaml:"schema_path,omitempty"`
	Validate    bool     `json:"validate,omitempty"      yaml:"validate,omitempty"`
	Command     string   `json:"command,omitempty"       yaml:"command,omitempty"`
	CommandArgs []string `json:"commandArgs,omitempty"   yaml:"command_args,omitempty"`
	Dir         string   `json:"dir,omitempty"           yaml:"dir,omitempty"`        // Working directory: "project_root", absolute path, or empty for workspace
	MustPass    bool     `json:"must_pass,omitempty"     yaml:"must_pass,omitempty"`  // Determines if validation failure blocks pipeline
	MaxRetries  int      `json:"maxRetries,omitempty"    yaml:"max_retries,omitempty"`

	// Progressive validation settings
	ProgressiveValidation bool   `json:"progressive_validation,omitempty" yaml:"progressive_validation,omitempty"` // Enable progressive validation with warnings
	RecoveryLevel         string `json:"recovery_level,omitempty"         yaml:"recovery_level,omitempty"`         // "conservative", "progressive", or "aggressive"
	AllowRecovery         bool   `json:"allow_recovery,omitempty"         yaml:"allow_recovery,omitempty"`         // Enable automatic JSON recovery
	WarnOnRecovery        bool   `json:"warn_on_recovery,omitempty"       yaml:"warn_on_recovery,omitempty"`       // Generate warnings instead of errors for recoverable issues

	// Wrapper detection settings
	DisableWrapperDetection bool `json:"disable_wrapper_detection,omitempty" yaml:"disable_wrapper_detection,omitempty"` // Disable error wrapper detection (default: false, detection enabled)
	DebugMode               bool `json:"debug_mode,omitempty"                yaml:"debug_mode,omitempty"`                // Enable debug logging for wrapper detection

	// LLM judge settings
	Model     string   `json:"model,omitempty"     yaml:"model,omitempty"`     // LLM model for judge evaluation; accepts tier names (cheapest, balanced, strongest) or literal model IDs
	Criteria  []string `json:"criteria,omitempty"  yaml:"criteria,omitempty"`  // Evaluation criteria for LLM judge
	Threshold float64  `json:"threshold,omitempty" yaml:"threshold,omitempty"` // Pass threshold (0.0-1.0), default 1.0

	// Convergence tracking for rework loops
	ConvergenceWindow         int     `json:"convergence_window,omitempty"          yaml:"convergence_window,omitempty"`          // Number of rounds to compare for stall detection (default 3)
	ConvergenceMinImprovement float64 `json:"convergence_min_improvement,omitempty" yaml:"convergence_min_improvement,omitempty"` // Minimum score improvement to consider progress (default 0.05 = 5%)

	// Agent review settings
	Persona      string                `json:"persona,omitempty"      yaml:"persona,omitempty"`       // Reviewer persona name
	CriteriaPath string                `json:"criteriaPath,omitempty" yaml:"criteria_path,omitempty"` // Path to review criteria markdown
	Context      []ReviewContextSource `json:"context,omitempty"      yaml:"context,omitempty"`       // Context sources for the reviewer
	TokenBudget  int                   `json:"tokenBudget,omitempty"  yaml:"token_budget,omitempty"`  // Max tokens for review agent
	Timeout      string                `json:"timeout,omitempty"      yaml:"timeout,omitempty"`       // Duration string for review timeout
	ReworkStep   string                `json:"reworkStep,omitempty"   yaml:"rework_step,omitempty"`   // Step ID for on_failure: rework
	OnFailure    string                `json:"onFailure,omitempty"    yaml:"on_failure,omitempty"`    // "fail", "skip", "continue", "rework"
	// ArtifactPaths provides artifact name→path mappings for artifact context sources.
	// This is populated by the executor at validation time, not from YAML.
	ArtifactPaths map[string]string `json:"artifactPaths,omitempty" yaml:"-"`

	// source_diff contract fields
	Glob     string   `json:"glob,omitempty"       yaml:"glob,omitempty"`      // Glob pattern for qualifying source files
	Exclude  []string `json:"exclude,omitempty"    yaml:"exclude,omitempty"`   // Glob patterns for files to exclude
	MinFiles int      `json:"min_files,omitempty"  yaml:"min_files,omitempty"` // Minimum number of qualifying changed files required (default 1)

	// event_contains contract fields — validated by executor (needs event store access)
	Events []EventPattern `json:"events,omitempty" yaml:"events,omitempty"` // Expected event patterns to match against the step's event log

	// spec_derived_test contract fields
	SpecArtifact       string `json:"spec_artifact,omitempty"        yaml:"spec_artifact,omitempty"`        // Path to the specification artifact file
	TestPersona        string `json:"test_persona,omitempty"         yaml:"test_persona,omitempty"`         // Persona that generates tests (must differ from implementer)
	ImplementationStep string `json:"implementation_step,omitempty"  yaml:"implementation_step,omitempty"` // Step ID of the implementation to validate
}

// ValidationError provides detailed information about contract validation failures.
type ValidationError struct {
	ContractType string
	Message      string
	Details      []string
	Retryable    bool
	Attempt      int
	MaxRetries   int
}

func (e *ValidationError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("contract validation failed [%s]", e.ContractType))

	if e.MaxRetries > 0 {
		sb.WriteString(fmt.Sprintf(" (attempt %d/%d)", e.Attempt, e.MaxRetries))
	}

	sb.WriteString(": ")
	sb.WriteString(e.Message)

	if len(e.Details) > 0 {
		sb.WriteString("\n  Details:")
		for _, detail := range e.Details {
			sb.WriteString("\n    - ")
			sb.WriteString(detail)
		}
	}

	return sb.String()
}

// ContractValidator defines the interface for contract validation.
type ContractValidator interface {
	Validate(cfg ContractConfig, workspacePath string) error
}

// NewValidator creates a new contract validator based on the configuration type.
func NewValidator(cfg ContractConfig) ContractValidator {
	switch cfg.Type {
	case "json_schema":
		return &jsonSchemaValidator{}
	case "typescript_interface":
		return &typeScriptValidator{}
	case "test_suite":
		return &testSuiteValidator{}
	case "markdown_spec":
		return &markdownSpecValidator{}
	case "format":
		return &FormatValidator{}
	case "non_empty_file":
		return &nonEmptyFileValidator{}
	case "llm_judge":
		return &llmJudgeValidator{}
	case "source_diff":
		return &sourceDiffValidator{}
	case "agent_review":
		// agent_review requires an adapter runner — NewValidator returns nil.
		// The executor uses ValidateWithRunner() instead for this type.
		return nil
	case "spec_derived_test":
		// spec_derived_test requires an adapter runner — NewValidator returns nil.
		// The executor uses ValidateSpecDerived() instead for this type.
		return nil
	default:
		return nil
	}
}

// Validate runs the appropriate validator for the given configuration.
func Validate(cfg ContractConfig, workspacePath string) error {
	validator := NewValidator(cfg)
	if validator != nil {
		if err := validator.Validate(cfg, workspacePath); err != nil {
			return err
		}
	}

	return nil
}

// ValidateWithRetries runs validation with retry logic.
// It returns a ValidationError when max retries are exhausted.
func ValidateWithRetries(cfg ContractConfig, workspacePath string) error {
	validator := NewValidator(cfg)
	if validator == nil {
		return nil
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := validator.Validate(cfg, workspacePath)
		if err == nil {
			return nil
		}
		lastErr = err

		// If this is the last attempt, return a detailed error
		if attempt == maxRetries {
			return &ValidationError{
				ContractType: cfg.Type,
				Message:      fmt.Sprintf("validation failed after %d attempt(s)", maxRetries),
				Details:      []string{lastErr.Error()},
				Retryable:    false,
				Attempt:      attempt,
				MaxRetries:   maxRetries,
			}
		}
	}

	return lastErr
}

// ValidateWithAdaptiveRetry runs validation with intelligent retry strategy.
// It analyzes failures and generates targeted repair prompts for the AI.
func ValidateWithAdaptiveRetry(cfg ContractConfig, workspacePath string) (*RetryResult, error) {
	strategy := NewAdaptiveRetryStrategy(cfg.MaxRetries)
	if cfg.MaxRetries <= 0 {
		strategy.MaxRetries = 3 // Default to 3 retries
	}

	result := &RetryResult{
		Attempts:     0,
		FailureTypes: make([]FailureType, 0),
	}

	startTime := time.Now()

	for attempt := 1; attempt <= strategy.MaxRetries; attempt++ {
		result.Attempts = attempt

		// Run validation
		err := Validate(cfg, workspacePath)

		if err == nil {
			// Success
			result.Success = true
			result.TotalDuration = time.Since(startTime)
			return result, nil
		}

		// Classify the failure
		classified := strategy.Classifier.Classify(err)
		if classified != nil {
			result.FailureTypes = append(result.FailureTypes, classified.Type)
		}

		// Check if we should retry
		if !strategy.ShouldRetry(attempt, err) {
			result.FinalError = err
			result.TotalDuration = time.Since(startTime)
			return result, err
		}

		// If not the last attempt, wait before retrying
		if attempt < strategy.MaxRetries {
			delay := strategy.GetRetryDelay(attempt)
			time.Sleep(delay)
		}
	}

	// All retries exhausted
	result.Success = false
	result.TotalDuration = time.Since(startTime)
	return result, result.FinalError
}

// GetRepairGuidance generates targeted guidance for fixing validation failures.
// This is intended to be injected into the AI prompt for retry attempts.
func GetRepairGuidance(err error, attempt int, maxRetries int) string {
	strategy := NewAdaptiveRetryStrategy(maxRetries)
	return strategy.GenerateRepairPrompt(err, attempt)
}

// WrapValidationError wraps a regular error into a ValidationError with details.
func WrapValidationError(contractType string, err error, details ...string) *ValidationError {
	return &ValidationError{
		ContractType: contractType,
		Message:      err.Error(),
		Details:      details,
		Retryable:    true,
	}
}
