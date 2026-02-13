//go:build webui

package webui

import (
	"strings"
	"testing"
)

func TestRedactCredentials(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // should NOT be in output
	}{
		{
			name:     "AWS access key",
			input:    "key=AKIAIOSFODNN7EXAMPLE",
			contains: "AKIAIOSFODNN7EXAMPLE",
		},
		{
			name:     "OpenAI key",
			input:    "OPENAI_API_KEY=sk-abcdefghijklmnopqrstuvwxyz1234567890abcdefghijklmn",
			contains: "sk-abcdefghijklmnopqrstuvwxyz1234567890abcdefghijklmn",
		},
		{
			name:     "GitHub PAT",
			input:    "token=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij",
			contains: "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij",
		},
		{
			name:     "password inline",
			input:    "password=mysecretpassword123",
			contains: "mysecretpassword123",
		},
		{
			name:     "bearer token",
			input:    "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.abc",
			contains: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactCredentials(tt.input)
			if strings.Contains(result, tt.contains) {
				t.Errorf("RedactCredentials() should have redacted %q but output was: %s", tt.contains, result)
			}
			if !strings.Contains(result, redactedPlaceholder) {
				t.Errorf("RedactCredentials() output should contain %q, got: %s", redactedPlaceholder, result)
			}
		})
	}
}

func TestRedactCredentials_NoCredentials(t *testing.T) {
	input := "This is normal text with no credentials at all."
	result := RedactCredentials(input)
	if result != input {
		t.Errorf("RedactCredentials() should not modify text without credentials, got: %s", result)
	}
}
