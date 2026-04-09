package commands

import (
	"encoding/json"
	"fmt"
	"io"
)

// Error code constants for machine-parseable error classification.
const (
	CodePipelineNotFound       = "pipeline_not_found"
	CodeManifestMissing        = "manifest_missing"
	CodeManifestInvalid        = "manifest_invalid"
	CodeContractViolation      = "contract_violation"
	CodeFlagConflict           = "flag_conflict"
	CodeOnboardingRequired     = "onboarding_required"
	CodePreflightFailed        = "preflight_failed"
	CodeInternalError          = "internal_error"
	CodeSecurityViolation      = "security_violation"
	CodeSkillNotFound          = "skill_not_found"
	CodeSkillSourceError       = "skill_source_error"
	CodeSkillDependencyMissing = "skill_dependency_missing"
	CodeInvalidArgs            = "invalid_args"
	CodeStateDBError           = "state_db_error"
	CodeRunNotFound            = "run_not_found"
	CodeMigrationFailed        = "migration_failed"
	CodeDatasetError           = "dataset_error"
	CodeValidationFailed       = "validation_failed"
	CodeSkillPublishFailed     = "skill_publish_failed"
	CodeSkillValidationFailed  = "skill_validation_failed"
	CodeSkillAlreadyExists     = "skill_already_exists"
)

// CLIError represents a structured error for CLI output.
// In JSON mode, this is rendered as a JSON object to stderr.
// In text mode, it renders as a human-readable error with suggestion.
type CLIError struct {
	Message    string `json:"error"`
	Code       string `json:"code"`
	Suggestion string `json:"suggestion"`
	Debug      string `json:"debug,omitempty"`
	Cause      error  `json:"-"`
}

// NewCLIError creates a new CLIError with the given code, message, and suggestion.
func NewCLIError(code, message, suggestion string) *CLIError {
	return &CLIError{
		Code:       code,
		Message:    message,
		Suggestion: suggestion,
	}
}

func (e *CLIError) Error() string {
	return e.Message
}

// Unwrap returns the underlying cause, supporting errors.Is/As chain inspection.
func (e *CLIError) Unwrap() error {
	return e.Cause
}

// WithCause returns a copy of the CLIError with the Cause field set.
func (e *CLIError) WithCause(err error) *CLIError {
	e.Cause = err
	return e
}

// RenderJSONError marshals an error as a JSON object to the writer.
// If the error is a *CLIError, it is serialized directly.
// Plain errors are wrapped as CLIError with code "internal_error".
func RenderJSONError(w io.Writer, err error, debug bool) {
	var cliErr *CLIError
	switch e := err.(type) {
	case *CLIError:
		cliErr = &CLIError{
			Message:    e.Message,
			Code:       e.Code,
			Suggestion: e.Suggestion,
			Debug:      e.Debug,
			Cause:      e.Cause,
		}
	default:
		cliErr = &CLIError{
			Message:    err.Error(),
			Code:       CodeInternalError,
			Suggestion: "",
		}
	}

	if !debug {
		cliErr.Debug = ""
	}

	data, marshalErr := json.Marshal(cliErr)
	if marshalErr != nil {
		fmt.Fprintf(w, `{"error":%q,"code":"internal_error","suggestion":""}`, err.Error())
		fmt.Fprintln(w)
		return
	}
	fmt.Fprintln(w, string(data))
}

// RenderTextError formats an error for human-readable text output.
// CLIErrors include suggestion lines; plain errors show just the message.
// Debug details are included only when debug=true.
func RenderTextError(w io.Writer, err error, debug bool) {
	switch e := err.(type) {
	case *CLIError:
		fmt.Fprintf(w, "Error: %s\n", e.Message)
		if e.Suggestion != "" {
			fmt.Fprintf(w, "  Suggestion: %s\n", e.Suggestion)
		}
		if debug && e.Debug != "" {
			fmt.Fprintf(w, "  Debug: %s\n", e.Debug)
		}
	default:
		fmt.Fprintf(w, "Error: %s\n", err.Error())
	}
}
