package security

import (
	"strings"
	"testing"
)

func TestNewPathTraversalError(t *testing.T) {
	approved := []string{"/home/user", "/tmp"}
	err := NewPathTraversalError("../../etc/passwd", approved)

	if err.Type != string(ViolationPathTraversal) {
		t.Errorf("Type = %q, want %q", err.Type, string(ViolationPathTraversal))
	}
	if !strings.Contains(err.Message, "path traversal") {
		t.Errorf("Message should mention path traversal, got: %s", err.Message)
	}
	if err.Retryable {
		t.Error("path traversal should not be retryable")
	}
	if err.IsRetryable() {
		t.Error("IsRetryable() should return false")
	}
	if err.SuggestedFix == "" {
		t.Error("SuggestedFix should not be empty")
	}
	if err.Details == nil {
		t.Error("Details should not be nil")
	}
	if dirs, ok := err.Details["approved_directories"]; !ok || dirs == nil {
		t.Error("Details should contain approved_directories")
	}
	if pathLen, ok := err.Details["path_length"]; !ok || pathLen != len("../../etc/passwd") {
		t.Errorf("Details path_length = %v, want %d", pathLen, len("../../etc/passwd"))
	}

	// Test Error() string format
	errStr := err.Error()
	if !strings.Contains(errStr, "[path_traversal]") {
		t.Errorf("Error() should contain type, got: %s", errStr)
	}
	if !strings.Contains(errStr, "Fix:") {
		t.Errorf("Error() should contain fix hint, got: %s", errStr)
	}
}

func TestNewPromptInjectionError(t *testing.T) {
	patterns := []string{"ignore previous", "system prompt"}
	err := NewPromptInjectionError("user_input", patterns)

	if err.Type != string(ViolationPromptInjection) {
		t.Errorf("Type = %q, want %q", err.Type, string(ViolationPromptInjection))
	}
	if !strings.Contains(err.Message, "prompt injection") {
		t.Errorf("Message should mention prompt injection, got: %s", err.Message)
	}
	if !err.Retryable {
		t.Error("prompt injection should be retryable")
	}
	if !err.IsRetryable() {
		t.Error("IsRetryable() should return true")
	}
	if inputType, ok := err.Details["input_type"]; !ok || inputType != "user_input" {
		t.Errorf("Details input_type = %v, want 'user_input'", inputType)
	}
	if count, ok := err.Details["patterns_detected"]; !ok || count != 2 {
		t.Errorf("Details patterns_detected = %v, want 2", count)
	}
}

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
