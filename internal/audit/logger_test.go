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
