package relay

import (
	"context"
	"os"
	"path/filepath"
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

func TestRelayMonitor_ConcurrentCompaction(t *testing.T) {
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
			checkpointPath := filepath.Join(workspacePath, CheckpointFilename)
			if _, statErr := os.Stat(checkpointPath); os.IsNotExist(statErr) {
				t.Errorf("goroutine %d: checkpoint file was not created", goroutineID)
			}
		}(i)
	}

	wg.Wait()
}
