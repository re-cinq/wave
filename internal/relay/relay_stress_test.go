package relay

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Stress Tests for Large Payloads and Edge Cases
// =============================================================================

func TestRelayMonitor_LargeTokenCounts(t *testing.T) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{
		DefaultThreshold:   80,
		MinTokensToCompact: 1000,
		ContextWindow:      1000000, // 1M context window
	}, adapter)

	testCases := []struct {
		name       string
		tokensUsed int
		expected   bool
	}{
		{"very large token count", 950000, true},    // 95% of 1M
		{"maximum token count", 1000000, true},      // 100% of 1M
		{"over context window", 1500000, true},      // 150% of 1M
		{"just under threshold", 799999, false},     // Just under 80%
		{"exactly at threshold", 800000, true},      // Exactly 80%
		{"extreme token count", 10000000, true},     // 10M tokens
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := m.ShouldCompact(tc.tokensUsed, 80)
			if result != tc.expected {
				t.Errorf("expected %v for %d tokens, got %v", tc.expected, tc.tokensUsed, result)
			}
		})
	}
}

func TestRelayMonitor_GetTokenCount_LargeTexts(t *testing.T) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{}, adapter)

	testCases := []struct {
		name        string
		textSize    int
		description string
	}{
		{"1KB text", 1024, "Small document"},
		{"10KB text", 10 * 1024, "Medium document"},
		{"100KB text", 100 * 1024, "Large document"},
		{"1MB text", 1024 * 1024, "Very large document"},
		{"5MB text", 5 * 1024 * 1024, "Extremely large document"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate text of approximately the target size
			baseText := "This is a test sentence for measuring token counting performance. "
			repetitions := tc.textSize / len(baseText)
			if repetitions < 1 {
				repetitions = 1
			}
			text := strings.Repeat(baseText, repetitions)

			start := time.Now()
			tokenCount := m.GetTokenCount(text)
			duration := time.Since(start)

			if tokenCount <= 0 {
				t.Errorf("expected positive token count, got %d", tokenCount)
			}

			t.Logf("%s: %d characters, %d tokens, took %v", tc.description, len(text), tokenCount, duration)

			// Performance check - should complete within reasonable time
			if duration > 100*time.Millisecond {
				t.Logf("WARNING: %s took %v to process, consider optimization", tc.description, duration)
			}
		})
	}
}

func TestParseCheckpoint_LargeCheckpoints(t *testing.T) {
	testCases := []struct {
		name           string
		summarySize    int
		decisionsCount int
	}{
		{"small checkpoint", 100, 5},
		{"medium checkpoint", 1000, 25},
		{"large checkpoint", 10000, 100},
		{"very large checkpoint", 50000, 500},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workspacePath := t.TempDir()

			// Generate large summary
			summaryLine := "This is a line in the summary section with meaningful content. "
			summaryLines := tc.summarySize / len(summaryLine)
			if summaryLines < 1 {
				summaryLines = 1
			}
			summary := strings.Repeat(summaryLine, summaryLines)

			// Generate many decisions
			var decisions []string
			for i := 0; i < tc.decisionsCount; i++ {
				decision := fmt.Sprintf("Decision %d: This is a detailed decision with context and reasoning.", i+1)
				decisions = append(decisions, decision)
			}

			checkpointContent := fmt.Sprintf(`# Checkpoint

## Summary
%s

## Decisions
%s

---
*Generated at stress test*
`, summary, strings.Join(decisions, "\n"))

			checkpointPath := workspacePath + "/checkpoint.md"
			if err := os.WriteFile(checkpointPath, []byte(checkpointContent), 0644); err != nil {
				t.Fatalf("failed to write large checkpoint: %v", err)
			}

			start := time.Now()
			checkpoint, err := ParseCheckpoint(workspacePath)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("failed to parse large checkpoint: %v", err)
			}

			if len(checkpoint.Decisions) != tc.decisionsCount {
				t.Errorf("expected %d decisions, got %d", tc.decisionsCount, len(checkpoint.Decisions))
			}

			t.Logf("%s: parsed %d decisions, took %v", tc.name, len(checkpoint.Decisions), duration)
		})
	}
}

func TestInjectCheckpointPrompt_LargeCheckpoints(t *testing.T) {
	workspacePath := t.TempDir()

	// Create a very large checkpoint
	summaryLines := make([]string, 1000)
	for i := range summaryLines {
		summaryLines[i] = fmt.Sprintf("Summary line %d with detailed information about the conversation state.", i+1)
	}

	decisions := make([]string, 200)
	for i := range decisions {
		decisions[i] = fmt.Sprintf("Decision %d: Detailed decision with context and implications for the system.", i+1)
	}

	checkpointContent := fmt.Sprintf(`# Checkpoint

## Summary
%s

## Decisions
%s

---
*Generated at large checkpoint test*
`, strings.Join(summaryLines, "\n"), strings.Join(decisions, "\n"))

	checkpointPath := workspacePath + "/checkpoint.md"
	if err := os.WriteFile(checkpointPath, []byte(checkpointContent), 0644); err != nil {
		t.Fatalf("failed to write large checkpoint: %v", err)
	}

	start := time.Now()
	prompt, err := InjectCheckpointPrompt(workspacePath)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("failed to inject large checkpoint: %v", err)
	}

	if !strings.Contains(prompt, "Summary line 500") {
		t.Error("prompt should contain middle summary lines")
	}

	if !strings.Contains(prompt, "Decision 100") {
		t.Error("prompt should contain middle decisions")
	}

	t.Logf("Large checkpoint injection took %v, generated %d character prompt", duration, len(prompt))
}

func TestAdapterRunnerWrapper_LargePayloads(t *testing.T) {
	// Test with large chat history and outputs
	largeOutput := strings.Repeat("This is a large compaction result with many details. ", 10000)

	mockRunner := &mockAdapterRunner{
		runFunc: func(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error) {
			// Simulate processing time proportional to input size
			time.Sleep(time.Duration(len(cfg.Prompt)/1000) * time.Microsecond)
			return &AdapterResult{
				ExitCode:   0,
				Stdout:     &mockReader{data: []byte(largeOutput)},
				TokensUsed: len(cfg.Prompt) / 4, // Approximate token estimation
			}, nil
		},
	}

	wrapper := &AdapterRunnerWrapper{
		Runner:      mockRunner,
		AdapterName: "claude",
		PersonaName: "summarizer",
	}

	testCases := []struct {
		name            string
		chatHistorySize int
	}{
		{"small history", 1024},
		{"medium history", 10 * 1024},
		{"large history", 100 * 1024},
		{"very large history", 500 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate large chat history
			baseHistory := "User: This is a conversation message with details. Assistant: This is a response with information. "
			repetitions := tc.chatHistorySize / len(baseHistory)
			if repetitions < 1 {
				repetitions = 1
			}
			chatHistory := strings.Repeat(baseHistory, repetitions)

			start := time.Now()
			result, err := wrapper.RunCompaction(context.Background(), CompactionConfig{
				WorkspacePath: t.TempDir(),
				ChatHistory:   chatHistory,
				SystemPrompt:  "system prompt",
				CompactPrompt: "summarize this conversation",
			})
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("unexpected error with large payload: %v", err)
			}

			if len(result) != len(largeOutput) {
				t.Errorf("expected output length %d, got %d", len(largeOutput), len(result))
			}

			t.Logf("%s: input %d chars, output %d chars, took %v", tc.name, len(chatHistory), len(result), duration)
		})
	}
}

func TestAdapterRunnerWrapper_OutputSizeLimit(t *testing.T) {
	// Test the 1MB output size limit
	hugeMockOutput := make([]byte, 2*1024*1024) // 2MB output
	for i := range hugeMockOutput {
		hugeMockOutput[i] = byte('A' + (i % 26))
	}

	mockRunner := &mockAdapterRunner{
		runFunc: func(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error) {
			return &AdapterResult{
				ExitCode:   0,
				Stdout:     &mockReader{data: hugeMockOutput},
				TokensUsed: 100000,
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
		ChatHistory:   "test",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be limited to 1MB
	const maxOutputSize = 1024 * 1024
	if len(result) > maxOutputSize {
		t.Errorf("output should be limited to %d bytes, got %d", maxOutputSize, len(result))
	}

	if len(result) != maxOutputSize {
		t.Errorf("expected exactly %d bytes (size limit), got %d", maxOutputSize, len(result))
	}
}

