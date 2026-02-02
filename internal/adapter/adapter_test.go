package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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

	jsonl := `{"type": "result", "content": {"text": "hello", "tokens": 10, "artifacts": ["a.go"]}}
{"type": "output", "content": {"text": "world", "tokens": 5}}
{"type": "other", "content": {"text": "ignored"}}
`

	tokens, artifacts := adapter.parseOutput([]byte(jsonl))

	if tokens != 15 {
		t.Errorf("expected 15 tokens, got: %d", tokens)
	}

	if len(artifacts) != 1 {
		t.Errorf("expected 1 artifact, got: %d", len(artifacts))
	}

	if len(artifacts) > 0 && artifacts[0] != "a.go" {
		t.Errorf("expected artifact 'a.go', got: %s", artifacts[0])
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
