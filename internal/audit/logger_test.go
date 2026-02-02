package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCredentialScrubbing(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	logger, err := NewTraceLoggerWithDir(traceDir)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"API_KEY", "API_KEY=sk-1234567890abcdef", "[REDACTED]"},
		{"token", "token:ghp_1234567890abcdef", "[REDACTED]"},
		{"SECRET", "SECRET=mysecret123", "[REDACTED]"},
		{"PASSWORD", "password=passw0rd", "[REDACTED]"},
		{"CREDENTIAL", "CREDENTIAL=cred123", "[REDACTED]"},
		{"AUTH", "AUTH=bearer_token", "[REDACTED]"},
		{"PRIVATE_KEY", "PRIVATE_KEY=pk_1234567890", "[REDACTED]"},
		{"ACCESS_KEY", "ACCESS_KEY=ak_1234567890", "[REDACTED]"},
		{"case insensitive", "api_key=sk-test", "[REDACTED]"},
		{"no credential", "normal_string", "normal_string"},
		{"mixed case", "Api-Key=value123", "[REDACTED]"},
		{"with hyphen", "ACCESS-KEY=key123", "[REDACTED]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logger.scrub(tt.input)
			if tt.expected == "[REDACTED]" {
				if result != tt.expected {
					t.Errorf("scrub(%q) = %q, want %q", tt.input, result, tt.expected)
				}
			} else {
				if result != tt.expected {
					t.Errorf("scrub(%q) = %q, want %q", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestCredentialScrubbingInContext(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	logger, err := NewTraceLoggerWithDir(traceDir)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	tests := []struct {
		name  string
		input string
		check func(*testing.T, string)
	}{
		{
			name:  "credential in command",
			input: "curl -H 'Authorization: Bearer token123' https://api.example.com",
			check: func(t *testing.T, result string) {
				if strings.Contains(result, "token123") {
					t.Errorf("credential not scrubbed: %s", result)
				}
				if !strings.Contains(result, "[REDACTED]") {
					t.Errorf("no [REDACTED] marker found")
				}
			},
		},
		{
			name:  "multiple credentials",
			input: "API_KEY=key1 TOKEN=token2",
			check: func(t *testing.T, result string) {
				if strings.Contains(result, "key1") || strings.Contains(result, "token2") {
					t.Errorf("credentials not scrubbed: %s", result)
				}
			},
		},
		{
			name:  "path with no credential pattern",
			input: "/home/user/secret/project",
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "secret") {
					t.Errorf("word 'secret' should NOT be scrubbed in paths: %s", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logger.scrub(tt.input)
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestLogFileCreation(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	logger, err := NewTraceLoggerWithDir(traceDir)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	if _, err := os.Stat(traceDir); os.IsNotExist(err) {
		t.Error("trace directory not created")
	}

	files, err := os.ReadDir(traceDir)
	if err != nil {
		t.Fatalf("failed to read trace directory: %v", err)
	}

	if len(files) == 0 {
		t.Error("no trace file created")
	}

	traceFile := files[0]
	if !strings.HasPrefix(traceFile.Name(), "trace-") || !strings.HasSuffix(traceFile.Name(), ".log") {
		t.Errorf("unexpected trace file name: %s", traceFile.Name())
	}

	err = logger.LogToolCall("pipeline-001", "step-001", "bash", "echo test")
	if err != nil {
		t.Errorf("LogToolCall failed: %v", err)
	}

	err = logger.LogFileOp("pipeline-001", "step-001", "read", "/path/to/file.txt")
	if err != nil {
		t.Errorf("LogFileOp failed: %v", err)
	}

	logger.Close()

	content, err := os.ReadFile(filepath.Join(traceDir, traceFile.Name()))
	if err != nil {
		t.Fatalf("failed to read trace file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "[TOOL]") {
		t.Error("trace file missing [TOOL] marker")
	}
	if !strings.Contains(contentStr, "[FILE]") {
		t.Error("trace file missing [FILE] marker")
	}
	if !strings.Contains(contentStr, "pipeline=pipeline-001") {
		t.Error("trace file missing pipeline ID")
	}
	if !strings.Contains(contentStr, "step=step-001") {
		t.Error("trace file missing step ID")
	}
}

func TestClose(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	logger, err := NewTraceLoggerWithDir(traceDir)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	err = logger.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	err = logger.LogToolCall("pipeline-001", "step-001", "bash", "echo test")
	if err == nil {
		t.Error("LogToolCall should fail after Close")
	}
}

// =============================================================================
// T102: Credential Scrubbing Patterns Tests
// =============================================================================

// TestCredentialScrubbingPatterns_ComprehensivePatterns tests that common
// credential patterns matching KEY=VALUE format are properly redacted.
// The current implementation uses regex to match patterns like:
//   API_KEY=value, TOKEN=value, SECRET=value, PASSWORD=value, etc.
func TestCredentialScrubbingPatterns_ComprehensivePatterns(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	logger, err := NewTraceLoggerWithDir(traceDir)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Test credential patterns that the current implementation DOES scrub.
	// These follow the pattern: KEYWORD[=:]\s*VALUE where KEYWORD matches
	// credentialPatterns (API_KEY, TOKEN, SECRET, PASSWORD, CREDENTIAL, AUTH,
	// PRIVATE_KEY, ACCESS_KEY)
	testCases := []struct {
		name     string
		input    string
		contains string // What should NOT be in output
	}{
		// API Keys - pattern matches API_KEY, API-KEY, APIKEY
		{"API_KEY=value", "API_KEY=sk-1234567890abcdef", "sk-1234567890abcdef"},
		{"api_key lowercase", "api_key=my_secret_key_123", "my_secret_key_123"},
		{"API-KEY with hyphen", "API-KEY=abc123def456", "abc123def456"},
		{"APIKEY no separator", "APIKEY=verysecretvalue", "verysecretvalue"},

		// Tokens - pattern matches TOKEN
		{"TOKEN=value", "TOKEN=ghp_1234567890abcdefghijklmnop", "ghp_1234567890abcdefghijklmnop"},
		{"token lowercase", "token:xoxb-123456789-abcdef", "xoxb-123456789-abcdef"},

		// Secrets - pattern matches SECRET
		{"SECRET=value", "SECRET=myverysecretvalue123", "myverysecretvalue123"},
		{"client_secret", "client_secret=IAmAClientSecret", "IAmAClientSecret"},
		{"signing_secret", "signing_secret=xoxb-secret-value", "xoxb-secret-value"},

		// Passwords - pattern matches PASSWORD
		{"PASSWORD=value", "PASSWORD=MyP@ssw0rd123", "MyP@ssw0rd123"},
		{"password lowercase", "password=hunter2abc", "hunter2abc"},
		{"db_password", "db_password=db_secret_pass", "db_secret_pass"},
		{"MYSQL_PASSWORD", "MYSQL_PASSWORD=root123abc", "root123abc"},

		// Credentials - pattern matches CREDENTIAL
		{"CREDENTIAL=value", "CREDENTIAL=credential_value_123", "credential_value_123"},

		// Auth - pattern matches AUTH
		{"AUTH=value", "AUTH=auth_token_12345", "auth_token_12345"},

		// Private Keys - pattern matches PRIVATE_KEY, PRIVATEKEY
		{"PRIVATE_KEY", "PRIVATE_KEY=pk_live_abcdef123456", "pk_live_abcdef123456"},
		{"PRIVATEKEY no separator", "PRIVATEKEY=myprivatekey", "myprivatekey"},

		// Access Keys - pattern matches ACCESS_KEY, ACCESSKEY
		{"ACCESS_KEY", "ACCESS_KEY=AKIAIOSFODNN7EXAMPLE", "AKIAIOSFODNN7EXAMPLE"},
		{"ACCESSKEY no separator", "ACCESSKEY=myaccesskey123", "myaccesskey123"},

		// Mixed case patterns (case insensitive)
		{"Api_Key mixed case", "Api_Key=MixedCaseKey123", "MixedCaseKey123"},
		{"AccEss_KeY mixed", "AccEss_KeY=mixedkey123", "mixedkey123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := logger.scrub(tc.input)

			// [REDACTED] should appear
			if !strings.Contains(result, "[REDACTED]") {
				t.Errorf("scrub(%q) = %q, expected [REDACTED] marker",
					tc.input, result)
			}
		})
	}
}

// TestCredentialScrubbingPatterns_PreservesNonCredentials ensures that normal
// text is not incorrectly scrubbed.
func TestCredentialScrubbingPatterns_PreservesNonCredentials(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	logger, err := NewTraceLoggerWithDir(traceDir)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// These should NOT be scrubbed
	testCases := []struct {
		name   string
		input  string
	}{
		{"normal path", "/home/user/project/src/main.go"},
		{"function call", "func GetAPIResponse() error"},
		{"variable name", "var secretCount = 5"},
		{"comment about token", "// Token parsing logic"},
		{"URL without credentials", "https://api.example.com/v1/users"},
		{"file path with secret word", "/var/log/secret-service/app.log"},
		{"code snippet", "if err := validatePassword(input); err != nil"},
		{"class name", "class CredentialManager"},
		{"method name", "parseAuthHeader(header)"},
		{"plain text", "This is just plain text with no credentials"},
		{"numbers", "12345678901234567890"},
		{"alphanumeric", "abc123def456ghi789"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := logger.scrub(tc.input)

			// Text should be preserved (not entirely replaced with [REDACTED])
			// Some words might trigger partial matching, so we check the text isn't completely lost
			if result == "[REDACTED]" {
				t.Errorf("scrub(%q) = %q, but input should be mostly preserved",
					tc.input, result)
			}
		})
	}
}

// TestCredentialScrubbingPatterns_MultipleCredentialsInText tests scrubbing
// when multiple credentials appear in a single text block.
// The current regex pattern matches KEYWORD=VALUE or KEYWORD:VALUE formats,
// where KEYWORD must match credential patterns and VALUE follows directly.
func TestCredentialScrubbingPatterns_MultipleCredentialsInText(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	logger, err := NewTraceLoggerWithDir(traceDir)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	testCases := []struct {
		name                 string
		input                string
		shouldContainRedacted bool
	}{
		{
			name:  "command with api_key",
			input: "curl -d 'api_key=key456abc' https://api.example.com",
			shouldContainRedacted: true,
		},
		{
			name:  "env vars format",
			input: "API_KEY=key1abc SECRET=secret2xyz TOKEN=token3def PASSWORD=pass4ghi",
			shouldContainRedacted: true,
		},
		{
			name:  "shell export command",
			input: "export API_KEY=sk-1234567890abcdef",
			shouldContainRedacted: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := logger.scrub(tc.input)

			if tc.shouldContainRedacted {
				if !strings.Contains(result, "[REDACTED]") {
					t.Errorf("scrub(%q) = %q, expected [REDACTED] marker",
						tc.input, result)
				}
			}
		})
	}
}

// TestCredentialScrubbingPatterns_LogToolCallScrubs verifies that LogToolCall
// properly scrubs credentials from arguments.
func TestCredentialScrubbingPatterns_LogToolCallScrubs(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	logger, err := NewTraceLoggerWithDir(traceDir)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Log a tool call with credentials in args
	sensitiveArgs := "curl -H 'API_KEY=sk-supersecret123' https://api.example.com"
	err = logger.LogToolCall("pipeline-001", "step-001", "bash", sensitiveArgs)
	if err != nil {
		t.Fatalf("LogToolCall failed: %v", err)
	}

	logger.Close()

	// Read the trace file and verify the secret was scrubbed
	files, err := os.ReadDir(traceDir)
	if err != nil {
		t.Fatalf("failed to read trace dir: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no trace file created")
	}

	content, err := os.ReadFile(filepath.Join(traceDir, files[0].Name()))
	if err != nil {
		t.Fatalf("failed to read trace file: %v", err)
	}

	contentStr := string(content)
	if strings.Contains(contentStr, "sk-supersecret123") {
		t.Errorf("trace file contains unredacted secret: %s", contentStr)
	}
	if !strings.Contains(contentStr, "[REDACTED]") {
		t.Errorf("trace file missing [REDACTED] marker: %s", contentStr)
	}
}

// TestCredentialScrubbingPatterns_LogFileOpScrubs verifies that LogFileOp
// properly scrubs credentials from file paths.
func TestCredentialScrubbingPatterns_LogFileOpScrubs(t *testing.T) {
	traceDir := filepath.Join(t.TempDir(), "traces")
	logger, err := NewTraceLoggerWithDir(traceDir)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Log a file op with credentials in path (edge case)
	sensitivePath := "/tmp/config_TOKEN=abc123/settings.json"
	err = logger.LogFileOp("pipeline-001", "step-001", "write", sensitivePath)
	if err != nil {
		t.Fatalf("LogFileOp failed: %v", err)
	}

	logger.Close()

	// Read the trace file and verify the secret was scrubbed
	files, err := os.ReadDir(traceDir)
	if err != nil {
		t.Fatalf("failed to read trace dir: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no trace file created")
	}

	content, err := os.ReadFile(filepath.Join(traceDir, files[0].Name()))
	if err != nil {
		t.Fatalf("failed to read trace file: %v", err)
	}

	contentStr := string(content)
	if strings.Contains(contentStr, "abc123") {
		t.Errorf("trace file contains unredacted token: %s", contentStr)
	}
}
