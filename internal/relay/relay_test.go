package relay

import (
	"context"
	"strings"
	"testing"
)

type mockAdapter struct {
	compactFunc func(ctx context.Context, chatHistory string, persona Persona) (string, error)
}

func (m *mockAdapter) Compact(ctx context.Context, chatHistory string, persona Persona) (string, error) {
	if m.compactFunc != nil {
		return m.compactFunc(ctx, chatHistory, persona)
	}
	return "compacted summary", nil
}

func TestRelayMonitor_ShouldCompact(t *testing.T) {
	m := NewRelayMonitor(relayConfig{DefaultThreshold: 80, MinTokensToCompact: 1000})

	tests := []struct {
		name          string
		tokensUsed    int
		contextWindow int
		threshold     int
		expected      bool
	}{
		{"below threshold", 500, 4000, 80, false},
		{"at threshold", 3200, 4000, 80, true},
		{"above threshold", 3500, 4000, 80, true},
		{"below min tokens", 500, 1000, 80, false},
		{"custom threshold", 2500, 4000, 60, true},
		{"zero threshold uses default", 3500, 4000, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.ShouldCompact(tt.tokensUsed, tt.contextWindow, tt.threshold)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRelayMonitor_Compact(t *testing.T) {
	m := NewRelayMonitor(relayConfig{})

	adapter := &mockAdapter{
		compactFunc: func(ctx context.Context, chatHistory string, persona Persona) (string, error) {
			return "test compaction result", nil
		},
	}

	persona := Persona{
		Name:          "test",
		CompactPrompt: "Compact this:",
	}

	workspacePath := t.TempDir()

	_, err := m.Compact(context.Background(), "test history", persona, adapter, workspacePath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	checkpointPath := workspacePath + "/checkpoint.md"
	if _, err := os.Stat(checkpointPath); os.IsNotExist(err) {
		t.Error("checkpoint file was not created")
	}
}

func TestRelayMonitor_CompactNoAdapter(t *testing.T) {
	m := NewRelayMonitor(relayConfig{})

	_, err := m.Compact(context.Background(), "test history", Persona{}, nil, t.TempDir())
	if err == nil {
		t.Error("expected error when no adapter provided")
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
