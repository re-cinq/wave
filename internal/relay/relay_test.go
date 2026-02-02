package relay

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

// mockCompactionAdapter implements CompactionAdapter for testing.
type mockCompactionAdapter struct {
	runFunc func(ctx context.Context, cfg CompactionConfig) (string, error)
}

func (m *mockCompactionAdapter) RunCompaction(ctx context.Context, cfg CompactionConfig) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, cfg)
	}
	return "compacted summary", nil
}

func TestRelayMonitor_ShouldCompact(t *testing.T) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{
		DefaultThreshold:   80,
		MinTokensToCompact: 1000,
		ContextWindow:      4000,
	}, adapter)

	tests := []struct {
		name       string
		tokensUsed int
		threshold  int
		expected   bool
	}{
		{"below threshold", 500, 80, false},
		{"at threshold", 3200, 80, true},
		{"above threshold", 3500, 80, true},
		{"below min tokens", 500, 80, false},
		{"custom threshold", 2500, 60, true},
		{"zero threshold uses default", 3500, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.ShouldCompact(tt.tokensUsed, tt.threshold)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRelayMonitor_ShouldCompactWithWindow(t *testing.T) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{
		DefaultThreshold:   80,
		MinTokensToCompact: 1000,
		ContextWindow:      200000,
	}, adapter)

	tests := []struct {
		name          string
		tokensUsed    int
		contextWindow int
		threshold     int
		expected      bool
	}{
		{"below threshold with custom window", 500, 4000, 80, false},
		{"at threshold with custom window", 3200, 4000, 80, true},
		{"above threshold with custom window", 3500, 4000, 80, true},
		{"zero window uses default", 180000, 0, 80, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.ShouldCompactWithWindow(tt.tokensUsed, tt.contextWindow, tt.threshold)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRelayMonitor_Compact(t *testing.T) {
	adapter := &mockCompactionAdapter{
		runFunc: func(ctx context.Context, cfg CompactionConfig) (string, error) {
			return "test compaction result", nil
		},
	}

	m := NewRelayMonitor(RelayMonitorConfig{}, adapter)

	workspacePath := t.TempDir()

	_, err := m.Compact(context.Background(), "test history", "system prompt", "Compact this:", workspacePath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	checkpointPath := workspacePath + "/checkpoint.md"
	if _, err := os.Stat(checkpointPath); os.IsNotExist(err) {
		t.Error("checkpoint file was not created")
	}
}

func TestRelayMonitor_CompactNoAdapter(t *testing.T) {
	m := NewRelayMonitor(RelayMonitorConfig{}, nil)

	_, err := m.Compact(context.Background(), "test history", "", "", t.TempDir())
	if err == nil {
		t.Error("expected error when no adapter provided")
	}
}

func TestRelayMonitor_GetContextWindow(t *testing.T) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{ContextWindow: 100000}, adapter)

	if m.GetContextWindow() != 100000 {
		t.Errorf("expected 100000, got %d", m.GetContextWindow())
	}
}

func TestRelayMonitor_GetContextWindowDefault(t *testing.T) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{}, adapter)

	// Should default to 200000
	if m.GetContextWindow() != 200000 {
		t.Errorf("expected 200000, got %d", m.GetContextWindow())
	}
}

func TestParseCheckpoint(t *testing.T) {
	workspacePath := t.TempDir()
	checkpointContent := `# Checkpoint

## Summary
Test summary
line 2

## Decisions
Decision 2

---
*Generated at test 1
Decision*
`
	os.WriteFile(workspacePath+"/checkpoint.md", []byte(checkpointContent), 0644)

	checkpoint, err := ParseCheckpoint(workspacePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if checkpoint.Summary != "Test summary\nline 2" {
		t.Errorf("unexpected summary: %s", checkpoint.Summary)
	}

	if len(checkpoint.Decisions) != 2 {
		t.Errorf("expected 2 decisions, got %d", len(checkpoint.Decisions))
	}
}

func TestInjectCheckpointPrompt(t *testing.T) {
	workspacePath := t.TempDir()
	checkpointContent := `# Checkpoint

## Summary
Test summary

## Decisions
Decision 1

---
*Generated at test*
`
	os.WriteFile(workspacePath+"/checkpoint.md", []byte(checkpointContent), 0644)

	prompt, err := InjectCheckpointPrompt(workspacePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(prompt, "Test summary") {
		t.Error("prompt should contain summary")
	}
	if !strings.Contains(prompt, "Decision 1") {
		t.Error("prompt should contain decisions")
	}
	if !strings.Contains(prompt, "READ CHECKPOINT.MD FIRST") {
		t.Error("prompt should contain header")
	}
}

func TestAdapterRunnerWrapper_RunCompaction(t *testing.T) {
	// Create a mock adapter runner
	mockRunner := &mockAdapterRunner{
		runFunc: func(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error) {
			// Verify the config was set correctly
			if cfg.Temperature != 0.3 {
				t.Errorf("expected temperature 0.3, got %f", cfg.Temperature)
			}
			if len(cfg.AllowedTools) != 3 {
				t.Errorf("expected 3 allowed tools, got %d", len(cfg.AllowedTools))
			}
			return &AdapterResult{
				ExitCode:   0,
				Stdout:     &mockReader{data: []byte("summarized content")},
				TokensUsed: 100,
			}, nil
		},
	}

	wrapper := &AdapterRunnerWrapper{
		Runner:      mockRunner,
		AdapterName: "claude",
		PersonaName: "summarizer",
	}

	result, err := wrapper.RunCompaction(context.Background(), CompactionConfig{
		WorkspacePath: t.TempDir(),
		ChatHistory:   "test chat history",
		SystemPrompt:  "system prompt",
		CompactPrompt: "summarize this",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "summarized content" {
		t.Errorf("expected 'summarized content', got '%s'", result)
	}
}

// mockAdapterRunner implements AdapterRunner for testing.
type mockAdapterRunner struct {
	runFunc func(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error)
}

func (m *mockAdapterRunner) Run(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, cfg)
	}
	return &AdapterResult{
		ExitCode:   0,
		Stdout:     &mockReader{data: []byte("default output")},
		TokensUsed: 100,
	}, nil
}

// mockReader implements StringReader for testing.
type mockReader struct {
	data []byte
	pos  int
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	if m.pos >= len(m.data) {
		return n, io.EOF
	}
	return n, nil
}
