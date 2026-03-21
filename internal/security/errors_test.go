package security

import (
	"fmt"
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

func TestNewInvalidPersonaError(t *testing.T) {
	available := []string{"navigator", "implementer", "reviewer"}
	err := NewInvalidPersonaError("hacker", available)

	if err.Type != string(ViolationInvalidPersona) {
		t.Errorf("Type = %q, want %q", err.Type, string(ViolationInvalidPersona))
	}
	if !strings.Contains(err.Message, "hacker") {
		t.Errorf("Message should mention requested persona, got: %s", err.Message)
	}
	if !err.Retryable {
		t.Error("invalid persona should be retryable")
	}
	if count, ok := err.Details["available_count"]; !ok || count != 3 {
		t.Errorf("Details available_count = %v, want 3", count)
	}
}

func TestNewMalformedJSONError(t *testing.T) {
	err := NewMalformedJSONError("trailing comma at line 5")

	if err.Type != string(ViolationMalformedJSON) {
		t.Errorf("Type = %q, want %q", err.Type, string(ViolationMalformedJSON))
	}
	if !strings.Contains(err.Message, "trailing comma") {
		t.Errorf("Message should contain the issue description, got: %s", err.Message)
	}
	if !err.Retryable {
		t.Error("malformed JSON should be retryable")
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

func TestNewConfigurationError(t *testing.T) {
	err := NewConfigurationError("max_input_length", "must be positive")

	if err.Type != "configuration_error" {
		t.Errorf("Type = %q, want 'configuration_error'", err.Type)
	}
	if !strings.Contains(err.Message, "max_input_length") {
		t.Errorf("Message should mention setting, got: %s", err.Message)
	}
	if err.Retryable {
		t.Error("configuration error should not be retryable")
	}
	if err.IsRetryable() {
		t.Error("IsRetryable() should return false for configuration error")
	}
}

func TestIsSecurityError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"security error", NewMalformedJSONError("test"), true},
		{"path traversal error", NewPathTraversalError("/etc", nil), true},
		{"regular error", fmt.Errorf("a regular error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSecurityError(tt.err)
			if got != tt.want {
				t.Errorf("IsSecurityError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSecurityError(t *testing.T) {
	secErr := NewMalformedJSONError("test")
	extracted, ok := GetSecurityError(secErr)
	if !ok {
		t.Error("GetSecurityError should return true for SecurityValidationError")
	}
	if extracted.Type != string(ViolationMalformedJSON) {
		t.Errorf("extracted Type = %q, want %q", extracted.Type, string(ViolationMalformedJSON))
	}
}
