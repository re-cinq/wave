package relay

import (
	"context"
	"os"
	"strings"
	"testing"
)

// =============================================================================
// Benchmarks for Performance-Critical Functions
// =============================================================================

func BenchmarkRelayMonitor_ShouldCompact(b *testing.B) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{
		DefaultThreshold:   80,
		MinTokensToCompact: 1000,
		ContextWindow:      200000,
	}, adapter)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.ShouldCompact(160000, 80)
	}
}

func BenchmarkRelayMonitor_ShouldCompactWithWindow(b *testing.B) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{
		DefaultThreshold:   80,
		MinTokensToCompact: 1000,
		ContextWindow:      200000,
	}, adapter)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.ShouldCompactWithWindow(160000, 200000, 80)
	}
}

func BenchmarkRelayMonitor_GetTokenCount_Small(b *testing.B) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{}, adapter)
	text := "This is a small text with just a few words."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.GetTokenCount(text)
	}
}

func BenchmarkRelayMonitor_GetTokenCount_Large(b *testing.B) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{}, adapter)
	// Generate a large text (10KB)
	text := strings.Repeat("This is a long conversation with many words that will be used for benchmarking token counting performance. ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.GetTokenCount(text)
	}
}

func BenchmarkRelayMonitor_GetTokenCount_Huge(b *testing.B) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{}, adapter)
	// Generate a huge text (1MB)
	text := strings.Repeat("This is a very long conversation with many words that will be used for benchmarking token counting performance. ", 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.GetTokenCount(text)
	}
}

func BenchmarkParseCheckpoint_Small(b *testing.B) {
	workspacePath := b.TempDir()
	checkpointContent := `# Checkpoint

## Summary
Small checkpoint content.

## Decisions
Simple decision

---
*Generated at test*
`
	writeTestCheckpoint(b, workspacePath, checkpointContent)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseCheckpoint(workspacePath)
	}
}

func BenchmarkParseCheckpoint_Large(b *testing.B) {
	workspacePath := b.TempDir()
	// Generate a large checkpoint
	summary := strings.Repeat("This is a line in the summary section. ", 100)
	decisions := make([]string, 50)
	for i := range decisions {
		decisions[i] = "Decision " + strings.Repeat("with details ", 10)
	}

	checkpointContent := `# Checkpoint

## Summary
` + summary + `

## Decisions
` + strings.Join(decisions, "\n") + `

---
*Generated at test*
`
	writeTestCheckpoint(b, workspacePath, checkpointContent)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseCheckpoint(workspacePath)
	}
}

func BenchmarkInjectCheckpointPrompt(b *testing.B) {
	workspacePath := b.TempDir()
	checkpointContent := `# Checkpoint

## Summary
This is a checkpoint summary that will be injected into prompts.

## Decisions
Decision about architecture
Decision about database
Decision about deployment

---
*Generated at benchmark test*
`
	writeTestCheckpoint(b, workspacePath, checkpointContent)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InjectCheckpointPrompt(workspacePath)
	}
}

func BenchmarkValidateConfig(b *testing.B) {
	cfg := RelayMonitorConfig{
		DefaultThreshold:   80,
		MinTokensToCompact: 1000,
		ContextWindow:      200000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateConfig(cfg)
	}
}

func BenchmarkValidateCheckpointFormat_Valid(b *testing.B) {
	content := `# Checkpoint

## Summary
Valid checkpoint content for benchmarking.

## Decisions
Benchmark decision

---
*Generated at test*
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateCheckpointFormat(content)
	}
}

func BenchmarkAdapterRunnerWrapper_RunCompaction(b *testing.B) {
	mockRunner := &mockAdapterRunner{
		runFunc: func(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error) {
			return &AdapterResult{
				ExitCode:   0,
				Stdout:     &mockReader{data: []byte("benchmark compaction result")},
				TokensUsed: 100,
			}, nil
		},
	}

	wrapper := &AdapterRunnerWrapper{
		Runner:      mockRunner,
		AdapterName: "claude",
		PersonaName: "summarizer",
	}

	compactionCfg := CompactionConfig{
		WorkspacePath: b.TempDir(),
		ChatHistory:   "benchmark chat history",
		SystemPrompt:  "system prompt",
		CompactPrompt: "summarize this",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrapper.RunCompaction(context.Background(), compactionCfg)
	}
}

// Helper function for benchmarks
func writeTestCheckpoint(b *testing.B, workspacePath, content string) {
	b.Helper()
	checkpointPath := workspacePath + "/checkpoint.md"
	if err := os.WriteFile(checkpointPath, []byte(content), 0644); err != nil {
		b.Fatalf("failed to write test checkpoint: %v", err)
	}
}