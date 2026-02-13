package recovery

import (
	"errors"
	"strings"

	"github.com/recinq/wave/internal/contract"
	"github.com/recinq/wave/internal/security"
)

// ClassifyError categorizes an error for recovery hint selection.
// It uses errors.As() to unwrap the error chain and identify typed errors.
// Generic/ambiguous messages (e.g. bare exit codes) map to ClassUnknown
// so the correct set of hints is shown.
func ClassifyError(err error) ErrorClass {
	if err == nil {
		return ClassUnknown
	}

	var ve *contract.ValidationError
	if errors.As(err, &ve) {
		return ClassContractValidation
	}

	var se *security.SecurityValidationError
	if errors.As(err, &se) {
		return ClassSecurityViolation
	}

	// Distinguish meaningful runtime messages from generic/empty ones.
	msg := err.Error()
	if msg == "" || isGenericErrorMessage(msg) {
		return ClassUnknown
	}

	return ClassRuntimeError
}

// isGenericErrorMessage returns true for error messages that carry no
// actionable context (bare exit codes, signal names, etc.).
func isGenericErrorMessage(msg string) bool {
	lower := strings.ToLower(msg)
	for _, prefix := range []string{
		"exit status ",
		"signal: ",
	} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}
