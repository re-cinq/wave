package redact

import (
	"strings"
	"testing"
)

// TestRedact_PrecisePatterns covers each of the precise-regex secret shapes
// inherited from the former internal/webui/redact.go.
func TestRedact_PrecisePatterns(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		secret  string // substring that must NOT appear in the output
	}{
		{"AWS access key", "key=AKIAIOSFODNN7EXAMPLE", "AKIAIOSFODNN7EXAMPLE"},
		{"AWS secret key", "aws_secret_access_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "wJalrXUtnFEMI"},
		{"OpenAI key", "OPENAI_API_KEY=sk-abcdefghijklmnopqrstuvwxyz1234567890abcdefghijklmn", "sk-abcdefghijklmnopqrstuvwxyz1234567890abcdefghijklmn"},
		{"GitHub PAT", "token=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij", "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"},
		{"GitHub OAuth", "x=gho_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij", "gho_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"},
		{"GitHub fine-grained PAT", "x=github_pat_" + strings.Repeat("a", 82), "github_pat_" + strings.Repeat("a", 82)},
		{"GitLab PAT", "x=glpat-" + strings.Repeat("a", 20), "glpat-" + strings.Repeat("a", 20)},
		{"Slack token", "x=xoxb-1234567890-abcdefghij", "xoxb-1234567890-abcdefghij"},
		{"password inline", "password=mysecretpassword123", "mysecretpassword123"},
		{"bearer token", "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.abc", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"api_key generic", "api_key=somelongvalue", "somelongvalue"},
		{"long token generic", "token=abcdefghijklmnopqrstuvwxyz123", "abcdefghijklmnopqrstuvwxyz123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Redact(tt.input)
			if strings.Contains(got, tt.secret) {
				t.Errorf("Redact(%q) leaked secret %q: %s", tt.input, tt.secret, got)
			}
			if !strings.Contains(got, Placeholder) {
				t.Errorf("Redact(%q) = %q, expected %q marker", tt.input, got, Placeholder)
			}
		})
	}
}

// TestRedact_GenericKeywords covers the loose KEYWORD=value shapes inherited
// from internal/audit/logger.go that the precise patterns alone do not catch
// (e.g. CREDENTIAL=, AUTH=, PRIVATE_KEY=, generic ACCESS_KEY=).
func TestRedact_GenericKeywords(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"API_KEY", "API_KEY=sk-1234567890abcdef"},
		{"token short", "token:ghp_short"},
		{"SECRET", "SECRET=mysecret123"},
		{"PASSWORD", "password=passw0rd"},
		{"CREDENTIAL", "CREDENTIAL=cred123"},
		{"AUTH", "AUTH=auth_token_12345"},
		{"PRIVATE_KEY", "PRIVATE_KEY=pk_live_abcdef123456"},
		{"ACCESS_KEY", "ACCESS_KEY=AKIAIOSFODNN7EXAMPLE"},
		{"case insensitive", "api_key=sk-test"},
		{"mixed case", "Api-Key=value123"},
		{"with hyphen", "ACCESS-KEY=key123"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Redact(tc.input)
			if !strings.Contains(got, Placeholder) {
				t.Errorf("Redact(%q) = %q, expected %q marker", tc.input, got, Placeholder)
			}
		})
	}
}

// TestRedact_PreservesNonCredentials ensures plain text is not corrupted.
func TestRedact_PreservesNonCredentials(t *testing.T) {
	cases := []string{
		"This is normal text with no credentials at all.",
		"/home/user/project/src/main.go",
		"https://api.example.com/v1/users",
		"12345678901234567890",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			got := Redact(in)
			if got == Placeholder {
				t.Errorf("Redact(%q) over-redacted to bare placeholder", in)
			}
			if got != in && !strings.Contains(got, Placeholder) {
				t.Errorf("Redact(%q) altered text without redaction marker: %q", in, got)
			}
		})
	}
}

// TestRedact_LargeInputUnchanged verifies the 256 KB scanning cap.
func TestRedact_LargeInputUnchanged(t *testing.T) {
	big := strings.Repeat("a", maxRedactSize+1) + " AKIAIOSFODNN7EXAMPLE"
	got := Redact(big)
	if got != big {
		t.Errorf("Redact should return inputs over %d bytes unchanged", maxRedactSize)
	}
}
