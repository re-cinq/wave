package security

import (
	"strings"
	"testing"
)

func TestNewInputValidationError(t *testing.T) {
	err := NewInputValidationError("schema_content", "exceeds size limit of 1048576 bytes")

	if err.Type != string(ViolationInputValidation) {
		t.Errorf("Type = %q, want %q", err.Type, string(ViolationInputValidation))
	}
	if !strings.Contains(err.Message, "schema_content") {
		t.Errorf("Message should mention input type, got: %s", err.Message)
	}
	if !strings.Contains(err.Message, "exceeds size limit") {
		t.Errorf("Message should contain reason, got: %s", err.Message)
	}
	if !err.Retryable {
		t.Error("input validation should be retryable")
	}
	if reason, ok := err.Details["reason"]; !ok || reason != "exceeds size limit of 1048576 bytes" {
		t.Errorf("Details reason = %v", reason)
	}
}
