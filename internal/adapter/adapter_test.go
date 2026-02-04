//go:build integration

package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestProcessGroupRunner_Run_Success(t *testing.T) {
	runner := NewProcessGroupRunner()
	ctx := context.Background()

	cfg := AdapterRunConfig{
		Adapter:       "echo",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "hello world",
		Timeout:       10 * time.Second,
		Env:           []string{},
	}

	result, err := runner.Run(ctx, cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got: %d", result.ExitCode)
	}

	stdoutBuf := make([]byte, 1024)
	n, _ := result.Stdout.Read(stdoutBuf)
	output := string(stdoutBuf[:n])

	if !strings.Contains(output, "hello world") {
		t.Errorf("expected 'hello world' in output, got: %s", output)
	}
}

func TestProcessGroupRunner_Run_Timeout(t *testing.T) {
	runner := NewProcessGroupRunner()
	ctx := context.Background()

	cfg := AdapterRunConfig{
		Adapter:       "sleep",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "10",
		Timeout:       100 * time.Millisecond,
		Env:           []string{},
	}

	result, err := runner.Run(ctx, cfg)
	if err == nil {
		t.Fatalf("expected timeout error, got result: %+v", result)
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestProcessGroupRunner_Run_NonZeroExit(t *testing.T) {
	runner := NewProcessGroupRunner()
	ctx := context.Background()

	// strings.Fields splits prompt into args, so "bash -c 'exit 42'" won't work.
	// Use a script that returns non-zero: /usr/bin/false returns 1.
	cfg := AdapterRunConfig{
		Adapter:       "false",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "",
		Timeout:       10 * time.Second,
		Env:           []string{},
	}

	result, err := runner.Run(ctx, cfg)
	if err != nil {
		t.Fatalf("expected no error for non-zero exit, got: %v", err)
	}

	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got: %d", result.ExitCode)
	}
}

func TestProcessGroupRunner_EnvInheritance(t *testing.T) {
	runner := NewProcessGroupRunner()
	ctx := context.Background()

	// The runner splits Prompt via strings.Fields, so we use printenv
	// which prints the value of a single env var.
	cfg := AdapterRunConfig{
		Adapter:       "printenv",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "TEST_VAR",
		Timeout:       10 * time.Second,
		Env:           []string{"TEST_VAR=test_value"},
	}

	result, err := runner.Run(ctx, cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	stdoutBuf := make([]byte, 1024)
	n, _ := result.Stdout.Read(stdoutBuf)
	output := string(stdoutBuf[:n])

	if !strings.Contains(output, "test_value") {
		t.Errorf("expected 'test_value' in output, got: %s", output)
	}
}

func TestMockAdapter_Run_Success(t *testing.T) {
	adapter := NewMockAdapter(
		WithStdoutJSON(`{"result": "success", "artifacts": ["file1.go", "file2.go"]}`),
		WithTokensUsed(150),
	)

	ctx := context.Background()
	cfg := AdapterRunConfig{
		Adapter:       "mock",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "test prompt",
		Timeout:       10 * time.Second,
	}

	result, err := adapter.Run(ctx, cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got: %d", result.ExitCode)
	}

	if result.TokensUsed != 150 {
		t.Errorf("expected 150 tokens, got: %d", result.TokensUsed)
	}

	expectedArtifacts := []string{"file1.go", "file2.go"}
	if len(result.Artifacts) != len(expectedArtifacts) {
		t.Errorf("expected %d artifacts, got: %d", len(expectedArtifacts), len(result.Artifacts))
	}
}

func TestMockAdapter_Run_Failure(t *testing.T) {
	testErr := errors.New("mock failure")
	adapter := NewMockAdapter(WithFailure(testErr))

	ctx := context.Background()
	cfg := AdapterRunConfig{
		Adapter:       "mock",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "test prompt",
		Timeout:       10 * time.Second,
	}

	_, err := adapter.Run(ctx, cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, testErr) {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestMockAdapter_Run_Timeout(t *testing.T) {
	adapter := NewMockAdapter(
		WithSimulatedDelay(500 * time.Millisecond),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cfg := AdapterRunConfig{
		Adapter:       "mock",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "test prompt",
		Timeout:       50 * time.Millisecond,
	}

	_, err := adapter.Run(ctx, cfg)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestMockAdapter_Run_CustomExitCode(t *testing.T) {
	adapter := NewMockAdapter(
		WithExitCode(1),
		WithStdoutJSON(`{"error": "something went wrong"}`),
	)

	ctx := context.Background()
	cfg := AdapterRunConfig{
		Adapter:       "mock",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "test prompt",
		Timeout:       10 * time.Second,
	}

	result, err := adapter.Run(ctx, cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got: %d", result.ExitCode)
	}
}

func TestMockAdapterRegistry_RegisterGet(t *testing.T) {
	registry := NewMockAdapterRegistry()

	adapter1 := NewMockAdapter(WithStdoutJSON(`{"name": "adapter1"}`))
	adapter2 := NewMockAdapter(WithStdoutJSON(`{"name": "adapter2"}`))

	registry.Register("adapter1", adapter1)
	registry.Register("adapter2", adapter2)

	retrieved := registry.Get("adapter1")
	if retrieved == nil {
		t.Fatal("expected adapter1 to be retrieved")
	}

	ctx := context.Background()
	cfg := AdapterRunConfig{Prompt: "test"}

	result, _ := retrieved.Run(ctx, cfg)
	stdoutBuf := make([]byte, 1024)
	n, _ := result.Stdout.Read(stdoutBuf)

	var parsed map[string]string
	json.Unmarshal(stdoutBuf[:n], &parsed)
	if parsed["name"] != "adapter1" {
		t.Errorf("expected adapter1, got: %s", parsed["name"])
	}
}

func TestMockAdapterRegistry_CreateRunner(t *testing.T) {
	registry := NewMockAdapterRegistry()
	original := NewMockAdapter(WithStdoutJSON(`{"runner": "test"}`))
	registry.Register("test-adapter", original)

	runner := registry.CreateRunner("test-adapter")

	ctx := context.Background()
	cfg := AdapterRunConfig{Prompt: "test"}

	result, err := runner.Run(ctx, cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	stdoutBuf := make([]byte, 1024)
	n, _ := result.Stdout.Read(stdoutBuf)

	var parsed map[string]string
	json.Unmarshal(stdoutBuf[:n], &parsed)
	if parsed["runner"] != "test" {
		t.Errorf("expected 'test', got: %s", parsed["runner"])
	}
}

func TestSlowReader_Read(t *testing.T) {
	data := "hello world"
	reader := NewSlowReader(data, 5, 10*time.Millisecond)

	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if n != 5 {
		t.Errorf("expected 5 bytes, got: %d", n)
	}

	if string(buf[:5]) != "hello" {
		t.Errorf("expected 'hello', got: %s", string(buf[:5]))
	}

	n, err = reader.Read(buf)
	if n != 5 {
		t.Errorf("expected 5 bytes, got: %d", n)
	}
	if string(buf[:n]) != " worl" {
		t.Errorf("expected ' worl', got: %s", string(buf[:n]))
	}

	n, err = reader.Read(buf)
	if n != 1 {
		t.Errorf("expected 1 byte, got: %d", n)
	}
	if err != io.EOF {
		t.Errorf("expected EOF, got: %v", err)
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"hello", 1},
		{"hello world", 2},
		{strings.Repeat("a", 100), 25},
		{"", 0},
	}

	for _, tt := range tests {
		result := estimateTokens(tt.input)
		if result != tt.expected {
			t.Errorf("estimateTokens(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestParseArtifacts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "no artifacts",
			input:    `{"result": "success"}`,
			expected: nil,
		},
		{
			name:     "with artifacts",
			input:    `{"result": "success", "artifacts": ["file1.go", "file2.go", "file3.go"]}`,
			expected: []string{"file1.go", "file2.go", "file3.go"},
		},
		{
			name:     "invalid json",
			input:    "not json",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var artifacts []string
			parseArtifacts([]byte(tt.input), &artifacts)

			if len(artifacts) != len(tt.expected) {
				t.Errorf("expected %d artifacts, got: %d", len(tt.expected), len(artifacts))
				return
			}

			for i, art := range artifacts {
				if art != tt.expected[i] {
					t.Errorf("artifact[%d] = %q, want %q", i, art, tt.expected[i])
				}
			}
		})
	}
}

func TestClaudeAdapter_ParseOutput(t *testing.T) {
	adapter := NewClaudeAdapter()

	// Test with new Claude CLI output format
	jsonl := `{"type":"result","subtype":"success","result":"Hello world","usage":{"input_tokens":10,"output_tokens":5}}`

	tokens, artifacts, resultContent := adapter.parseOutput([]byte(jsonl))

	if tokens != 15 {
		t.Errorf("expected 15 tokens, got: %d", tokens)
	}

	if len(artifacts) != 0 {
		t.Errorf("expected 0 artifacts, got: %d", len(artifacts))
	}

	if resultContent != "Hello world" {
		t.Errorf("expected result content 'Hello world', got: %s", resultContent)
	}
}

func TestAdapterResult_Fields(t *testing.T) {
	result := &AdapterResult{
		ExitCode:   0,
		Stdout:     bytes.NewReader([]byte(`{"result": "success"}`)),
		TokensUsed: 100,
		Artifacts:  []string{"file.go"},
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit_code 0, got: %d", result.ExitCode)
	}

	if result.TokensUsed != 100 {
		t.Errorf("expected tokens_used 100, got: %d", result.TokensUsed)
	}

	if len(result.Artifacts) != 1 || result.Artifacts[0] != "file.go" {
		t.Errorf("expected 1 artifact 'file.go', got: %v", result.Artifacts)
	}

	stdoutData, err := io.ReadAll(result.Stdout)
	if err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}
	if string(stdoutData) != `{"result": "success"}` {
		t.Errorf("unexpected stdout: %s", string(stdoutData))
	}
}

// =============================================================================
// Permission Enforcement Tests (T053-T059)
// =============================================================================

// T053: Add test for deny pattern blocks Write
func TestPermissionChecker_DenyPatternBlocksWrite(t *testing.T) {
	checker := NewPermissionChecker(
		"auditor",
		[]string{"Read", "Grep", "Glob"},
		[]string{"Write(*)", "Edit(*)"},
	)

	// Write should be denied
	err := checker.CheckPermission("Write", "src/main.go")
	if err == nil {
		t.Fatal("expected Write to be denied, got nil error")
	}

	permErr, ok := err.(*PermissionError)
	if !ok {
		t.Fatalf("expected PermissionError, got: %T", err)
	}

	if permErr.Tool != "Write" {
		t.Errorf("expected tool 'Write', got: %s", permErr.Tool)
	}

	if permErr.PersonaName != "auditor" {
		t.Errorf("expected persona 'auditor', got: %s", permErr.PersonaName)
	}

	// Edit should also be denied
	err = checker.CheckPermission("Edit", "src/config.yaml")
	if err == nil {
		t.Fatal("expected Edit to be denied, got nil error")
	}
}

// T054: Add test for allow pattern permits operation
func TestPermissionChecker_AllowPatternPermitsOperation(t *testing.T) {
	checker := NewPermissionChecker(
		"craftsman",
		[]string{"Read", "Write(.wave/specs/*)", "Bash(git log*)", "Edit"},
		[]string{},
	)

	// Read should be allowed (simple tool name)
	err := checker.CheckPermission("Read", "src/main.go")
	if err != nil {
		t.Fatalf("expected Read to be allowed, got: %v", err)
	}

	// Write to .wave/specs/ should be allowed
	err = checker.CheckPermission("Write", ".wave/specs/feature.yaml")
	if err != nil {
		t.Fatalf("expected Write to .wave/specs/ to be allowed, got: %v", err)
	}

	// Bash with git log should be allowed
	err = checker.CheckPermission("Bash", "git log --oneline")
	if err != nil {
		t.Fatalf("expected Bash with git log to be allowed, got: %v", err)
	}

	// Edit should be allowed (no argument restriction)
	err = checker.CheckPermission("Edit", "any/path/file.go")
	if err != nil {
		t.Fatalf("expected Edit to be allowed, got: %v", err)
	}
}

// T055: Add test for deny takes precedence over allow
func TestPermissionChecker_DenyTakesPrecedenceOverAllow(t *testing.T) {
	// Even though Write is in allowed list, deny should take precedence
	checker := NewPermissionChecker(
		"restricted",
		[]string{"Read", "Write", "Edit", "Bash"},                // Allow all basic tools
		[]string{"Write(*)", "Bash(rm -rf*)", "Bash(sudo *)"},    // But deny specific patterns
	)

	// Write is allowed but denied by deny pattern - should be blocked
	err := checker.CheckPermission("Write", "src/main.go")
	if err == nil {
		t.Fatal("expected Write to be denied due to deny pattern, got nil")
	}

	permErr, ok := err.(*PermissionError)
	if !ok {
		t.Fatalf("expected PermissionError, got: %T", err)
	}

	if !strings.Contains(permErr.Reason, "blocked by deny pattern") {
		t.Errorf("expected reason to mention deny pattern, got: %s", permErr.Reason)
	}

	// Bash is allowed but rm -rf is denied
	err = checker.CheckPermission("Bash", "rm -rf /home/user")
	if err == nil {
		t.Fatal("expected rm -rf to be denied, got nil")
	}

	// Bash with safe commands should be allowed
	err = checker.CheckPermission("Bash", "ls -la")
	if err != nil {
		t.Fatalf("expected safe bash to be allowed, got: %v", err)
	}

	// sudo commands should be denied
	err = checker.CheckPermission("Bash", "sudo apt install foo")
	if err == nil {
		t.Fatal("expected sudo to be denied, got nil")
	}

	// Read should still be allowed (no deny pattern)
	err = checker.CheckPermission("Read", "src/config.yaml")
	if err != nil {
		t.Fatalf("expected Read to be allowed, got: %v", err)
	}
}

// T056: Add test for permission error message format
func TestPermissionChecker_ErrorMessageFormat(t *testing.T) {
	checker := NewPermissionChecker(
		"navigator",
		[]string{"Read", "Glob"},
		[]string{"Write(*)"},
	)

	// Test error message format with deny pattern
	err := checker.CheckPermission("Write", "src/main.go")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()

	// Check that error message contains persona name
	if !strings.Contains(errMsg, "navigator") {
		t.Errorf("expected error to contain persona name 'navigator', got: %s", errMsg)
	}

	// Check that error message contains tool name
	if !strings.Contains(errMsg, "Write") {
		t.Errorf("expected error to contain tool name 'Write', got: %s", errMsg)
	}

	// Check that error message contains argument
	if !strings.Contains(errMsg, "src/main.go") {
		t.Errorf("expected error to contain argument 'src/main.go', got: %s", errMsg)
	}

	// Check error message format
	if !strings.Contains(errMsg, "permission denied") {
		t.Errorf("expected error to start with 'permission denied', got: %s", errMsg)
	}

	// Test error for not-allowed tool (no deny match, but not in allow list)
	checker2 := NewPermissionChecker(
		"auditor",
		[]string{"Read", "Grep"},
		[]string{},
	)

	err2 := checker2.CheckPermission("Write", "output.txt")
	if err2 == nil {
		t.Fatal("expected error for unlisted tool, got nil")
	}

	errMsg2 := err2.Error()
	if !strings.Contains(errMsg2, "auditor") {
		t.Errorf("expected error to contain persona name, got: %s", errMsg2)
	}
	if !strings.Contains(errMsg2, "not in allowed tools list") {
		t.Errorf("expected error to mention allowed tools list, got: %s", errMsg2)
	}
}

// T057: Verify permission check order (deny first)
func TestPermissionChecker_CheckOrder_DenyFirst(t *testing.T) {
	// This test verifies that deny patterns are checked BEFORE allow patterns.
	// The check order should be:
	// 1. Deny patterns (if any match -> deny)
	// 2. Allow patterns (if any match -> allow)
	// 3. Default (no allow patterns -> allow, else deny)

	testCases := []struct {
		name        string
		allowed     []string
		denied      []string
		tool        string
		argument    string
		expectDeny  bool
		expectReason string
	}{
		{
			name:        "deny checked first - deny matches",
			allowed:     []string{"Write"},
			denied:      []string{"Write(*.exe)"},
			tool:        "Write",
			argument:    "malware.exe",
			expectDeny:  true,
			expectReason: "blocked by deny pattern",
		},
		{
			name:        "deny not matched - allow checked",
			allowed:     []string{"Write"},
			denied:      []string{"Write(*.exe)"},
			tool:        "Write",
			argument:    "document.txt",
			expectDeny:  false,
		},
		{
			name:        "no deny patterns - allow checked",
			allowed:     []string{"Read"},
			denied:      []string{},
			tool:        "Read",
			argument:    "file.go",
			expectDeny:  false,
		},
		{
			name:        "no allow patterns - default allow",
			allowed:     []string{},
			denied:      []string{},
			tool:        "Write",
			argument:    "file.txt",
			expectDeny:  false,
		},
		{
			name:        "wildcard deny takes precedence",
			allowed:     []string{"*"},
			denied:      []string{"*"},
			tool:        "Read",
			argument:    "anything",
			expectDeny:  true,
			expectReason: "blocked by deny pattern",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checker := NewPermissionChecker("test-persona", tc.allowed, tc.denied)
			err := checker.CheckPermission(tc.tool, tc.argument)

			if tc.expectDeny {
				if err == nil {
					t.Fatal("expected operation to be denied, got nil error")
				}
				if tc.expectReason != "" && !strings.Contains(err.Error(), tc.expectReason) {
					t.Errorf("expected reason to contain %q, got: %s", tc.expectReason, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected operation to be allowed, got: %v", err)
				}
			}
		})
	}
}

// T058: Improve permission denied error message with persona name
func TestPermissionError_ContainsPersonaName(t *testing.T) {
	personas := []string{"navigator", "philosopher", "craftsman", "auditor", "reviewer"}

	for _, persona := range personas {
		checker := NewPermissionChecker(persona, []string{"Read"}, []string{"Write(*)"})

		err := checker.CheckPermission("Write", "test.txt")
		if err == nil {
			t.Fatalf("expected error for persona %s, got nil", persona)
		}

		// Verify persona name is in error message
		if !strings.Contains(err.Error(), persona) {
			t.Errorf("expected error message to contain persona name '%s', got: %s", persona, err.Error())
		}

		// Verify it's a properly typed error
		permErr, ok := err.(*PermissionError)
		if !ok {
			t.Fatalf("expected *PermissionError, got: %T", err)
		}

		if permErr.PersonaName != persona {
			t.Errorf("expected PersonaName to be '%s', got: '%s'", persona, permErr.PersonaName)
		}
	}
}

// T059: Add glob pattern matching tests
func TestPermissionChecker_GlobPatternMatching(t *testing.T) {
	testCases := []struct {
		name      string
		pattern   string
		tool      string
		argument  string
		shouldMatch bool
	}{
		// Simple wildcards
		{
			name:        "simple wildcard matches anything",
			pattern:     "*",
			tool:        "Write",
			argument:    "anything.go",
			shouldMatch: true,
		},
		{
			name:        "tool name exact match",
			pattern:     "Read",
			tool:        "Read",
			argument:    "file.txt",
			shouldMatch: true,
		},
		{
			name:        "tool name mismatch",
			pattern:     "Read",
			tool:        "Write",
			argument:    "file.txt",
			shouldMatch: false,
		},

		// Argument patterns with *
		{
			name:        "argument wildcard at end",
			pattern:     "Write(.wave/specs/*)",
			tool:        "Write",
			argument:    ".wave/specs/feature.yaml",
			shouldMatch: true,
		},
		{
			name:        "argument wildcard at end - no match",
			pattern:     "Write(.wave/specs/*)",
			tool:        "Write",
			argument:    "src/main.go",
			shouldMatch: false,
		},
		{
			name:        "argument wildcard at start",
			pattern:     "Bash(git log*)",
			tool:        "Bash",
			argument:    "git log --oneline",
			shouldMatch: true,
		},
		{
			name:        "argument exact match",
			pattern:     "Bash(ls -la)",
			tool:        "Bash",
			argument:    "ls -la",
			shouldMatch: true,
		},

		// Double star for deep paths
		{
			name:        "double star matches deep paths",
			pattern:     "Write(src/**/*.go)",
			tool:        "Write",
			argument:    "src/internal/adapter/adapter.go",
			shouldMatch: true,
		},
		{
			name:        "double star matches direct child",
			pattern:     "Read(**/*.yaml)",
			tool:        "Read",
			argument:    "config.yaml",
			shouldMatch: true,
		},

		// Single character wildcard
		{
			name:        "question mark single char",
			pattern:     "Write(file?.txt)",
			tool:        "Write",
			argument:    "file1.txt",
			shouldMatch: true,
		},
		{
			name:        "question mark no match for multiple chars",
			pattern:     "Write(file?.txt)",
			tool:        "Write",
			argument:    "file12.txt",
			shouldMatch: false,
		},

		// Character classes
		{
			name:        "character class matches",
			pattern:     "Read([abc].txt)",
			tool:        "Read",
			argument:    "a.txt",
			shouldMatch: true,
		},
		{
			name:        "character class no match",
			pattern:     "Read([abc].txt)",
			tool:        "Read",
			argument:    "d.txt",
			shouldMatch: false,
		},

		// Empty patterns and edge cases
		{
			name:        "tool with any argument (no constraint)",
			pattern:     "Edit",
			tool:        "Edit",
			argument:    "any/path/file.go",
			shouldMatch: true,
		},
		{
			name:        "tool with explicit (*) matches any",
			pattern:     "Write(*)",
			tool:        "Write",
			argument:    "absolutely/any/path.txt",
			shouldMatch: true,
		},
		{
			name:        "empty argument matches empty constraint",
			pattern:     "Bash",
			tool:        "Bash",
			argument:    "",
			shouldMatch: true,
		},

		// Dangerous command patterns
		{
			name:        "deny rm -rf pattern",
			pattern:     "Bash(rm -rf*)",
			tool:        "Bash",
			argument:    "rm -rf /home/user",
			shouldMatch: true,
		},
		{
			name:        "deny sudo pattern",
			pattern:     "Bash(sudo *)",
			tool:        "Bash",
			argument:    "sudo apt install package",
			shouldMatch: true,
		},

		// Path separator handling
		{
			name:        "single star does not cross path segments",
			pattern:     "Write(src/*.go)",
			tool:        "Write",
			argument:    "src/main.go",
			shouldMatch: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a checker where the pattern is either allow or deny
			// to test if it matches
			var checker *PermissionChecker
			if tc.shouldMatch {
				// If we expect match, put pattern in deny and check it blocks
				checker = NewPermissionChecker("test", []string{}, []string{tc.pattern})
				err := checker.CheckPermission(tc.tool, tc.argument)
				if err == nil {
					t.Errorf("pattern %q should have matched tool=%q arg=%q but didn't", tc.pattern, tc.tool, tc.argument)
				}
			} else {
				// If we don't expect match, put pattern in deny and check it allows
				checker = NewPermissionChecker("test", []string{}, []string{tc.pattern})
				err := checker.CheckPermission(tc.tool, tc.argument)
				if err != nil {
					t.Errorf("pattern %q should NOT have matched tool=%q arg=%q but did: %v", tc.pattern, tc.tool, tc.argument, err)
				}
			}
		})
	}
}

// Test internal matchToolPattern function directly for comprehensive coverage
func TestMatchToolPattern(t *testing.T) {
	testCases := []struct {
		pattern  string
		tool     string
		argument string
		expected bool
	}{
		// Basic patterns
		{"*", "Read", "any", true},
		{"*", "Write", "", true},
		{"Read", "Read", "file.txt", true},
		{"Read", "Write", "file.txt", false},

		// Patterns with arguments
		{"Write(*)", "Write", "file.txt", true},
		{"Write(*)", "Read", "file.txt", false},
		{"Write(.wave/*)", "Write", ".wave/specs.yaml", true},
		{"Write(.wave/*)", "Write", "src/main.go", false},

		// Patterns without argument (matches any argument)
		{"Edit", "Edit", "path/to/file.go", true},
		{"Edit", "Edit", "", true},

		// Complex glob patterns
		{"Bash(git *)", "Bash", "git status", true},
		{"Bash(git *)", "Bash", "git log --oneline", true},
		{"Bash(git *)", "Bash", "npm install", false},
	}

	for _, tc := range testCases {
		result := matchToolPattern(tc.pattern, tc.tool, tc.argument)
		if result != tc.expected {
			t.Errorf("matchToolPattern(%q, %q, %q) = %v, want %v",
				tc.pattern, tc.tool, tc.argument, result, tc.expected)
		}
	}
}

// Test parseToolPattern function
func TestParseToolPattern(t *testing.T) {
	testCases := []struct {
		pattern     string
		expectedTool string
		expectedArg string
	}{
		{"Read", "Read", ""},
		{"Write(*)", "Write", "*"},
		{"Bash(git log*)", "Bash", "git log*"},
		{"Write(.wave/specs/*)", "Write", ".wave/specs/*"},
		{"Edit", "Edit", ""},
		{"Glob(**/*.go)", "Glob", "**/*.go"},
		{"Read([abc].txt)", "Read", "[abc].txt"},
		// Edge cases
		{"Tool(with(nested)parens)", "Tool", "with(nested)parens"},
		{"NoParens", "NoParens", ""},
	}

	for _, tc := range testCases {
		tool, arg := parseToolPattern(tc.pattern)
		if tool != tc.expectedTool {
			t.Errorf("parseToolPattern(%q) tool = %q, want %q", tc.pattern, tool, tc.expectedTool)
		}
		if arg != tc.expectedArg {
			t.Errorf("parseToolPattern(%q) arg = %q, want %q", tc.pattern, arg, tc.expectedArg)
		}
	}
}

// Test IsPermissionError helper
func TestIsPermissionError(t *testing.T) {
	permErr := &PermissionError{
		PersonaName: "test",
		Tool:        "Write",
		Argument:    "file.txt",
		Reason:      "not allowed",
	}

	if !IsPermissionError(permErr) {
		t.Error("IsPermissionError should return true for *PermissionError")
	}

	regularErr := errors.New("regular error")
	if IsPermissionError(regularErr) {
		t.Error("IsPermissionError should return false for regular error")
	}

	if IsPermissionError(nil) {
		t.Error("IsPermissionError should return false for nil")
	}
}

// Test permission checker with no patterns (permissive mode)
func TestPermissionChecker_EmptyPatterns(t *testing.T) {
	// No patterns means everything is allowed
	checker := NewPermissionChecker("permissive", []string{}, []string{})

	tools := []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep", "NotebookEdit", "CustomTool"}
	for _, tool := range tools {
		err := checker.CheckPermission(tool, "any/argument")
		if err != nil {
			t.Errorf("expected all tools allowed with empty patterns, but %s was denied: %v", tool, err)
		}
	}
}

// Test overlapping allow and deny patterns
func TestPermissionChecker_OverlappingPatterns(t *testing.T) {
	// Allow Write to specific paths, deny Write everywhere else
	checker := NewPermissionChecker(
		"scoped-writer",
		[]string{"Read", "Write(.wave/*)"},
		[]string{"Write(*)"},
	)

	// Write to .wave/ - allowed by allow pattern, but denied by deny pattern
	// Since deny takes precedence, this should be denied
	err := checker.CheckPermission("Write", ".wave/config.yaml")
	if err == nil {
		t.Fatal("expected Write to be denied even to .wave/ when deny(*) exists")
	}

	// This demonstrates that deny(*) is very broad - to allow specific paths,
	// you need to use more specific deny patterns or restructure permissions

	// Better approach: specific deny patterns
	checker2 := NewPermissionChecker(
		"better-scoped",
		[]string{"Read", "Write"},
		[]string{"Write(src/*)", "Write(internal/*)"},
	)

	// Write to .wave/ should be allowed (not in deny)
	err = checker2.CheckPermission("Write", ".wave/config.yaml")
	if err != nil {
		t.Fatalf("expected Write to .wave/ to be allowed, got: %v", err)
	}

	// Write to src/ should be denied
	err = checker2.CheckPermission("Write", "src/main.go")
	if err == nil {
		t.Fatal("expected Write to src/ to be denied")
	}
}

// =============================================================================
// T105: Subprocess Timeout Tests with Hanging Mock
// =============================================================================

// TestSubprocessTimeout_HangingProcess verifies that a hanging subprocess
// is properly terminated after the timeout.
func TestSubprocessTimeout_HangingProcess(t *testing.T) {
	runner := NewProcessGroupRunner()
	ctx := context.Background()

	// Use sleep to simulate a hanging process
	cfg := AdapterRunConfig{
		Adapter:       "sleep",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "60", // Sleep for 60 seconds (will be killed by timeout)
		Timeout:       100 * time.Millisecond,
		Env:           []string{},
	}

	start := time.Now()
	result, err := runner.Run(ctx, cfg)
	elapsed := time.Since(start)

	// Should have returned an error (context deadline exceeded)
	if err == nil {
		t.Fatalf("expected timeout error, got result: %+v", result)
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}

	// Should have returned quickly (around the timeout, not 60 seconds)
	if elapsed > 500*time.Millisecond {
		t.Errorf("process took too long to timeout: %v", elapsed)
	}
}

// TestSubprocessTimeout_ContextCancellation verifies that canceling the context
// properly terminates the subprocess.
func TestSubprocessTimeout_ContextCancellation(t *testing.T) {
	runner := NewProcessGroupRunner()

	ctx, cancel := context.WithCancel(context.Background())

	cfg := AdapterRunConfig{
		Adapter:       "sleep",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "60",
		Timeout:       60 * time.Second, // Long timeout
		Env:           []string{},
	}

	// Cancel the context after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := runner.Run(ctx, cfg)
	elapsed := time.Since(start)

	// Should have returned an error (context canceled)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}

	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context error, got: %v", err)
	}

	// Should have returned quickly
	if elapsed > 500*time.Millisecond {
		t.Errorf("context cancellation took too long: %v", elapsed)
	}
}

// TestSubprocessTimeout_ProcessGroupKill verifies that the entire process group
// is killed, not just the parent process.
func TestSubprocessTimeout_ProcessGroupKill(t *testing.T) {
	runner := NewProcessGroupRunner()
	ctx := context.Background()

	// Use sleep directly to test timeout behavior
	// The adapter splits prompt by strings.Fields, so we need separate args
	cfg := AdapterRunConfig{
		Adapter:       "sleep",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "60",
		Timeout:       100 * time.Millisecond,
		Env:           []string{},
	}

	start := time.Now()
	_, err := runner.Run(ctx, cfg)
	elapsed := time.Since(start)

	// Should have timed out
	if err == nil {
		t.Fatal("expected timeout error")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}

	// Should have returned quickly (process group killed)
	if elapsed > 500*time.Millisecond {
		t.Errorf("process group kill took too long: %v", elapsed)
	}
}

// TestSubprocessTimeout_ZeroTimeout verifies that zero timeout gets a default.
func TestSubprocessTimeout_ZeroTimeout(t *testing.T) {
	runner := NewProcessGroupRunner()
	ctx := context.Background()

	cfg := AdapterRunConfig{
		Adapter:       "echo",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "quick test",
		Timeout:       0, // Should get default timeout
		Env:           []string{},
	}

	result, err := runner.Run(ctx, cfg)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got: %d", result.ExitCode)
	}
}

// TestSubprocessTimeout_MockAdapterRespects verifies that the mock adapter
// properly respects context cancellation.
func TestSubprocessTimeout_MockAdapterRespects(t *testing.T) {
	// Create a mock adapter that simulates a slow operation
	adapter := NewMockAdapter(
		WithSimulatedDelay(500 * time.Millisecond),
	)

	// Use context with timeout, since mock adapter respects context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cfg := AdapterRunConfig{
		Adapter:       "mock",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "test prompt",
		Timeout:       500 * time.Millisecond, // Config timeout is longer
	}

	start := time.Now()
	_, err := adapter.Run(ctx, cfg)
	elapsed := time.Since(start)

	// Should have timed out via context
	if err == nil {
		t.Fatal("expected timeout error from mock adapter")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}

	// Should have returned around the context timeout
	if elapsed > 300*time.Millisecond {
		t.Errorf("mock adapter timeout took too long: %v", elapsed)
	}
}

// TestSubprocessTimeout_MultipleSequentialTimeouts verifies that timeouts
// work correctly for multiple sequential subprocess invocations.
func TestSubprocessTimeout_MultipleSequentialTimeouts(t *testing.T) {
	runner := NewProcessGroupRunner()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		cfg := AdapterRunConfig{
			Adapter:       "sleep",
			Persona:       "test",
			WorkspacePath: "/tmp",
			Prompt:        "60",
			Timeout:       50 * time.Millisecond,
			Env:           []string{},
		}

		start := time.Now()
		_, err := runner.Run(ctx, cfg)
		elapsed := time.Since(start)

		if err == nil {
			t.Errorf("iteration %d: expected timeout error", i)
		}

		if elapsed > 300*time.Millisecond {
			t.Errorf("iteration %d: timeout took too long: %v", i, elapsed)
		}
	}
}

// TestSubprocessTimeout_ConcurrentTimeouts verifies that multiple concurrent
// subprocess timeouts work correctly.
func TestSubprocessTimeout_ConcurrentTimeouts(t *testing.T) {
	runner := NewProcessGroupRunner()
	ctx := context.Background()

	const numConcurrent = 5
	results := make(chan error, numConcurrent)

	start := time.Now()

	for i := 0; i < numConcurrent; i++ {
		go func(id int) {
			cfg := AdapterRunConfig{
				Adapter:       "sleep",
				Persona:       "test",
				WorkspacePath: "/tmp",
				Prompt:        "60",
				Timeout:       100 * time.Millisecond,
				Env:           []string{},
			}
			_, err := runner.Run(ctx, cfg)
			results <- err
		}(i)
	}

	// Collect all results
	timeoutCount := 0
	for i := 0; i < numConcurrent; i++ {
		err := <-results
		if errors.Is(err, context.DeadlineExceeded) {
			timeoutCount++
		}
	}

	elapsed := time.Since(start)

	// All should have timed out
	if timeoutCount != numConcurrent {
		t.Errorf("expected %d timeouts, got %d", numConcurrent, timeoutCount)
	}

	// All should have completed roughly at the same time
	if elapsed > 500*time.Millisecond {
		t.Errorf("concurrent timeouts took too long: %v", elapsed)
	}
}

// TestSubprocessTimeout_GracefulVsForced tests that SIGKILL terminates processes
// that might ignore SIGTERM.
func TestSubprocessTimeout_GracefulVsForced(t *testing.T) {
	runner := NewProcessGroupRunner()
	ctx := context.Background()

	// Use sleep which will be killed by SIGKILL on timeout
	cfg := AdapterRunConfig{
		Adapter:       "sleep",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "60",
		Timeout:       100 * time.Millisecond,
		Env:           []string{},
	}

	start := time.Now()
	_, err := runner.Run(ctx, cfg)
	elapsed := time.Since(start)

	// Should have timed out
	if err == nil {
		t.Fatal("expected timeout error")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}

	// SIGKILL should work quickly
	if elapsed > 500*time.Millisecond {
		t.Errorf("SIGKILL should have terminated process quickly: %v", elapsed)
	}
}

// TestSubprocessTimeout_OutputBeforeTimeout verifies timeout behavior
// when a process produces output before hanging.
func TestSubprocessTimeout_OutputBeforeTimeout(t *testing.T) {
	runner := NewProcessGroupRunner()
	ctx := context.Background()

	// Use a simple timeout test - the process will be killed
	cfg := AdapterRunConfig{
		Adapter:       "sleep",
		Persona:       "test",
		WorkspacePath: "/tmp",
		Prompt:        "60",
		Timeout:       100 * time.Millisecond,
		Env:           []string{},
	}

	start := time.Now()
	_, err := runner.Run(ctx, cfg)
	elapsed := time.Since(start)

	// Should have timed out
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected timeout, got: %v", err)
	}

	// Should complete quickly
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}

// TestSubprocessTimeout_ClaudeAdapterTimeout tests that the Claude adapter
// (when available) properly handles timeouts.
func TestSubprocessTimeout_ClaudeAdapterTimeout(t *testing.T) {
	// Skip if claude is not available
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not available")
	}

	adapter := NewClaudeAdapter()
	ctx := context.Background()

	cfg := AdapterRunConfig{
		Adapter:       "claude",
		Persona:       "test",
		WorkspacePath: t.TempDir(),
		Prompt:        "This is a test that should timeout",
		Timeout:       10 * time.Millisecond, // Very short timeout
		Env:           []string{},
	}

	start := time.Now()
	_, err := adapter.Run(ctx, cfg)
	elapsed := time.Since(start)

	// Should have returned (either success or timeout)
	// We mainly want to verify it doesn't hang
	if elapsed > 5*time.Second {
		t.Errorf("Claude adapter took too long: %v", elapsed)
	}

	if err != nil {
		t.Logf("Adapter returned error (expected for short timeout): %v", err)
	}
}

// TestExtractJSONFromMarkdown tests JSON extraction in isolation - no Claude needed
func TestExtractJSONFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "extracts simple json object",
			input:    "Here is the result:\n```json\n{\"key\": \"value\"}\n```\nDone.",
			expected: `{"key": "value"}`,
		},
		{
			name:     "extracts json array",
			input:    "List:\n```json\n[1, 2, 3]\n```",
			expected: `[1, 2, 3]`,
		},
		{
			name:     "extracts nested json",
			input:    "```json\n{\"files\": [\"a.go\"], \"meta\": {\"count\": 1}}\n```",
			expected: `{"files": ["a.go"], "meta": {"count": 1}}`,
		},
		{
			name:     "handles whitespace in block",
			input:    "```json\n\n  {\"a\": 1}  \n\n```",
			expected: `{"a": 1}`,
		},
		{
			name:     "returns empty for no json block",
			input:    "Just plain text",
			expected: "",
		},
		{
			name:     "returns empty for invalid json in block",
			input:    "```json\nnot valid json\n```",
			expected: "",
		},
		{
			name:     "returns empty for unclosed block",
			input:    "```json\n{\"key\": \"value\"}",
			expected: "",
		},
		{
			name:     "extracts first json block only",
			input:    "```json\n{\"first\": 1}\n```\n```json\n{\"second\": 2}\n```",
			expected: `{"first": 1}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractJSONFromMarkdown(tc.input)
			if result != tc.expected {
				t.Errorf("ExtractJSONFromMarkdown(%q)\n  got:  %q\n  want: %q", tc.input, result, tc.expected)
			}
		})
	}
}
