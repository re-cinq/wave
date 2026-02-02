package relay

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// Concurrency and Race Condition Tests
// =============================================================================

func TestRelayMonitor_ConcurrentShouldCompact(t *testing.T) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{
		DefaultThreshold:   80,
		MinTokensToCompact: 1000,
		ContextWindow:      200000,
	}, adapter)

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines calling ShouldCompact concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				tokensUsed := 150000 + (goroutineID*1000 + j) // Vary tokens to test different scenarios
				result := m.ShouldCompact(tokensUsed, 80)
				// Basic sanity check - the result should be consistent
				if tokensUsed >= 160000 && !result {
					t.Errorf("goroutine %d: expected true for %d tokens, got false", goroutineID, tokensUsed)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestRelayMonitor_ConcurrentGetMethods(t *testing.T) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{
		ContextWindow: 100000,
	}, adapter)

	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // Two methods being tested

	// Test concurrent GetContextWindow calls
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				contextWindow := m.GetContextWindow()
				if contextWindow != 100000 {
					t.Errorf("expected 100000, got %d", contextWindow)
				}
			}
		}()
	}

	// Test concurrent GetTokenCount calls
	testText := "This is a test string with multiple words for token counting."
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				tokenCount := m.GetTokenCount(testText)
				if tokenCount <= 0 {
					t.Errorf("expected positive token count, got %d", tokenCount)
				}
			}
		}()
	}

	wg.Wait()
}

func TestRelayMonitor_ConcurrentCompaction(t *testing.T) {
	// Use a thread-safe mock that tracks call count
	callCount := int32(0)
	adapter := &mockCompactionAdapter{
		runFunc: func(ctx context.Context, cfg CompactionConfig) (string, error) {
			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)
			return "compacted result", nil
		},
	}

	m := NewRelayMonitor(RelayMonitorConfig{}, adapter)

	const numGoroutines = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines calling Compact concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			workspacePath := t.TempDir()

			_, err := m.Compact(context.Background(), "test history", "system", "compact", workspacePath)
			if err != nil {
				t.Errorf("goroutine %d: unexpected error: %v", goroutineID, err)
			}

			// Verify checkpoint file was created
			checkpointPath := workspacePath + "/checkpoint.md"
			if _, statErr := os.Stat(checkpointPath); os.IsNotExist(statErr) {
				t.Errorf("goroutine %d: checkpoint file was not created", goroutineID)
			}
		}(i)
	}

	wg.Wait()
}

func TestAdapterRunnerWrapper_ConcurrentRunCompaction(t *testing.T) {
	mockRunner := &mockAdapterRunner{
		runFunc: func(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error) {
			// Simulate some processing time
			time.Sleep(5 * time.Millisecond)
			return &AdapterResult{
				ExitCode:   0,
				Stdout:     &mockReader{data: []byte("concurrent result")},
				TokensUsed: 100,
			}, nil
		},
	}

	wrapper := &AdapterRunnerWrapper{
		Runner:      mockRunner,
		AdapterName: "claude",
		PersonaName: "summarizer",
	}

	const numGoroutines = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			result, err := wrapper.RunCompaction(context.Background(), CompactionConfig{
				WorkspacePath: t.TempDir(),
				ChatHistory:   "test chat history",
				SystemPrompt:  "system prompt",
				CompactPrompt: "summarize this",
			})

			if err != nil {
				t.Errorf("goroutine %d: unexpected error: %v", goroutineID, err)
				return
			}

			if result != "concurrent result" {
				t.Errorf("goroutine %d: expected 'concurrent result', got '%s'", goroutineID, result)
			}
		}(i)
	}

	wg.Wait()
}

func TestConcurrentCheckpointOperations(t *testing.T) {
	// Test concurrent checkpoint parsing and injection
	workspacePath := t.TempDir()
	checkpointContent := `# Checkpoint

## Summary
Concurrent test checkpoint summary.

## Decisions
Decision for concurrent testing

---
*Generated at concurrent test*
`

	checkpointPath := workspacePath + "/checkpoint.md"
	if err := os.WriteFile(checkpointPath, []byte(checkpointContent), 0644); err != nil {
		t.Fatalf("failed to write test checkpoint: %v", err)
	}

	const numGoroutines = 30

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // ParseCheckpoint and InjectCheckpointPrompt

	// Test concurrent ParseCheckpoint calls
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			checkpoint, err := ParseCheckpoint(workspacePath)
			if err != nil {
				t.Errorf("goroutine %d: ParseCheckpoint failed: %v", goroutineID, err)
				return
			}
			if checkpoint.Summary != "Concurrent test checkpoint summary." {
				t.Errorf("goroutine %d: unexpected summary: %s", goroutineID, checkpoint.Summary)
			}
		}(i)
	}

	// Test concurrent InjectCheckpointPrompt calls
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			prompt, err := InjectCheckpointPrompt(workspacePath)
			if err != nil {
				t.Errorf("goroutine %d: InjectCheckpointPrompt failed: %v", goroutineID, err)
				return
			}
			if !strings.Contains(prompt, "Concurrent test checkpoint summary") {
				t.Errorf("goroutine %d: prompt should contain summary", goroutineID)
			}
		}(i)
	}

	wg.Wait()
}

func TestValidationFunctions_ConcurrentCalls(t *testing.T) {
	// Test concurrent validation function calls

	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // Three validation functions

	// Test concurrent ValidateConfig calls
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			cfg := RelayMonitorConfig{
				DefaultThreshold:   80,
				MinTokensToCompact: 1000,
				ContextWindow:      200000,
			}
			err := ValidateConfig(cfg)
			if err != nil {
				t.Errorf("ValidateConfig failed: %v", err)
			}
		}()
	}

	// Test concurrent ValidateCheckpointFormat calls
	validContent := `# Checkpoint

## Summary
Valid content for concurrent testing.

---
*Generated*
`
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			err := ValidateCheckpointFormat(validContent)
			if err != nil {
				t.Errorf("ValidateCheckpointFormat failed: %v", err)
			}
		}()
	}

	// Test concurrent error checking function calls
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			testErr := ErrCompactionFailed
			if !IsCompactionError(testErr) {
				t.Error("IsCompactionError should return true")
			}
			if IsCheckpointError(testErr) {
				t.Error("IsCheckpointError should return false")
			}
		}()
	}

	wg.Wait()
}

func TestRelayMonitor_StressTest(t *testing.T) {
	// High-frequency calls to test for race conditions
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{
		DefaultThreshold:   80,
		MinTokensToCompact: 1000,
		ContextWindow:      200000,
	}, adapter)

	const numGoroutines = 100
	const operationsPerGoroutine = 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				// Mix different operations to stress test
				switch j % 4 {
				case 0:
					m.ShouldCompact(160000+j, 80)
				case 1:
					m.ShouldCompactWithWindow(160000+j, 200000, 80)
				case 2:
					m.GetContextWindow()
				case 3:
					m.GetTokenCount("test string")
				}
			}
		}(i)
	}

	wg.Wait()
}