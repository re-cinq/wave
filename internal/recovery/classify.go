package recovery

import (
	"errors"

	"github.com/recinq/wave/internal/contract"
	"github.com/recinq/wave/internal/security"
)

// ClassifyError categorizes an error for recovery hint selection.
// It uses errors.As() to unwrap the error chain and identify typed errors.
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

	if err.Error() != "" {
		return ClassRuntimeError
	}

	return ClassUnknown
}
