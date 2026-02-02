package relay

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
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

// =============================================================================
// T066: Threshold Detection Tests
// =============================================================================

func TestRelayMonitor_ThresholdDetection_EdgeCases(t *testing.T) {
	adapter := &mockCompactionAdapter{}

	testCases := []struct {
		name               string
		config             RelayMonitorConfig
		tokensUsed         int
		thresholdPercent   int
		expected           bool
		description        string
	}{
		{
			name: "exactly at threshold boundary",
			config: RelayMonitorConfig{
				DefaultThreshold:   80,
				MinTokensToCompact: 1000,
				ContextWindow:      10000,
			},
			tokensUsed:       8000, // exactly 80%
			thresholdPercent: 80,
			expected:         true,
			description:      "should trigger when exactly at threshold",
		},
		{
			name: "one token below threshold",
			config: RelayMonitorConfig{
				DefaultThreshold:   80,
				MinTokensToCompact: 1000,
				ContextWindow:      10000,
			},
			tokensUsed:       7999, // one below 80%
			thresholdPercent: 80,
			expected:         false,
			description:      "should not trigger when one below threshold",
		},
		{
			name: "meets threshold but below min tokens",
			config: RelayMonitorConfig{
				DefaultThreshold:   80,
				MinTokensToCompact: 10000,
				ContextWindow:      10000,
			},
			tokensUsed:       8000,
			thresholdPercent: 80,
			expected:         false,
			description:      "should not trigger when below min tokens even if at threshold",
		},
		{
			name: "very low threshold (10%)",
			config: RelayMonitorConfig{
				DefaultThreshold:   10,
				MinTokensToCompact: 100,
				ContextWindow:      10000,
			},
			tokensUsed:       1000,
			thresholdPercent: 10,
			expected:         true,
			description:      "should work with low threshold percentage",
		},
		{
			name: "100% threshold",
			config: RelayMonitorConfig{
				DefaultThreshold:   100,
				MinTokensToCompact: 1000,
				ContextWindow:      10000,
			},
			tokensUsed:       10000,
			thresholdPercent: 100,
			expected:         true,
			description:      "should trigger at 100% usage",
		},
		{
			name: "zero tokens used",
			config: RelayMonitorConfig{
				DefaultThreshold:   80,
				MinTokensToCompact: 1000,
				ContextWindow:      10000,
			},
			tokensUsed:       0,
			thresholdPercent: 80,
			expected:         false,
			description:      "should not trigger with zero tokens",
		},
		{
			name: "over 100% usage",
			config: RelayMonitorConfig{
				DefaultThreshold:   80,
				MinTokensToCompact: 1000,
				ContextWindow:      10000,
			},
			tokensUsed:       15000, // 150%
			thresholdPercent: 80,
			expected:         true,
			description:      "should trigger when over context window",
		},
		{
			name: "min tokens exactly met",
			config: RelayMonitorConfig{
				DefaultThreshold:   50,
				MinTokensToCompact: 5000,
				ContextWindow:      10000,
			},
			tokensUsed:       5000, // exactly 50% and exactly min
			thresholdPercent: 50,
			expected:         true,
			description:      "should trigger when both threshold and min tokens exactly met",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewRelayMonitor(tc.config, adapter)
			result := m.ShouldCompact(tc.tokensUsed, tc.thresholdPercent)
			if result != tc.expected {
				t.Errorf("%s: expected %v, got %v", tc.description, tc.expected, result)
			}
		})
	}
}

func TestRelayMonitor_ThresholdDetection_DefaultValues(t *testing.T) {
	adapter := &mockCompactionAdapter{}

	t.Run("uses default threshold when zero provided", func(t *testing.T) {
		m := NewRelayMonitor(RelayMonitorConfig{
			DefaultThreshold:   75,
			MinTokensToCompact: 1000,
			ContextWindow:      10000,
		}, adapter)

		// 75% of 10000 = 7500
		result := m.ShouldCompact(7500, 0) // 0 means use default
		if !result {
			t.Error("should use default threshold of 75% when 0 is provided")
		}

		result = m.ShouldCompact(7000, 0)
		if result {
			t.Error("should not trigger below default threshold")
		}
	})

	t.Run("uses default context window when zero in config", func(t *testing.T) {
		m := NewRelayMonitor(RelayMonitorConfig{
			DefaultThreshold:   80,
			MinTokensToCompact: 1000,
			// ContextWindow not set - should default to 200000
		}, adapter)

		// 80% of 200000 = 160000
		result := m.ShouldCompact(160000, 80)
		if !result {
			t.Error("should use default context window of 200000")
		}

		if m.GetContextWindow() != 200000 {
			t.Errorf("expected default context window 200000, got %d", m.GetContextWindow())
		}
	})

	t.Run("uses default min tokens when zero in config", func(t *testing.T) {
		m := NewRelayMonitor(RelayMonitorConfig{
			DefaultThreshold: 80,
			ContextWindow:    10000,
			// MinTokensToCompact not set - should default to 1000
		}, adapter)

		// 80% of 10000 = 8000, but need at least 1000 min tokens
		result := m.ShouldCompact(800, 10) // 10% of 10000 = 1000, but only 800 tokens
		if result {
			t.Error("should not trigger when below default min tokens of 1000")
		}
	})
}

// =============================================================================
// T069: Relay with Summarizer Failure Tests
// =============================================================================

func TestRelayMonitor_Compact_SummarizerFailure(t *testing.T) {
	testCases := []struct {
		name          string
		adapterError  error
		expectedError string
	}{
		{
			name:          "adapter returns generic error",
			adapterError:  errors.New("adapter execution failed"),
			expectedError: "compaction failed",
		},
		{
			name:          "adapter returns timeout error",
			adapterError:  context.DeadlineExceeded,
			expectedError: "compaction failed",
		},
		{
			name:          "adapter returns context canceled",
			adapterError:  context.Canceled,
			expectedError: "compaction failed",
		},
		{
			name:          "adapter returns network error",
			adapterError:  errors.New("connection refused"),
			expectedError: "compaction failed",
		},
		{
			name:          "adapter returns rate limit error",
			adapterError:  errors.New("rate limit exceeded"),
			expectedError: "compaction failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &mockCompactionAdapter{
				runFunc: func(ctx context.Context, cfg CompactionConfig) (string, error) {
					return "", tc.adapterError
				},
			}

			m := NewRelayMonitor(RelayMonitorConfig{}, adapter)
			workspacePath := t.TempDir()

			_, err := m.Compact(context.Background(), "test history", "system prompt", "compact this", workspacePath)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("expected error to contain %q, got: %v", tc.expectedError, err)
			}

			// Verify checkpoint was not written on failure
			checkpointPath := workspacePath + "/checkpoint.md"
			if _, err := os.Stat(checkpointPath); !os.IsNotExist(err) {
				t.Error("checkpoint should not be written on adapter failure")
			}
		})
	}
}

func TestRelayMonitor_Compact_ContextCancellation(t *testing.T) {
	adapter := &mockCompactionAdapter{
		runFunc: func(ctx context.Context, cfg CompactionConfig) (string, error) {
			// Simulate slow operation that respects context
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return "completed", nil
			}
		},
	}

	m := NewRelayMonitor(RelayMonitorConfig{}, adapter)

	t.Run("context canceled before completion", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := m.Compact(ctx, "test history", "", "", t.TempDir())
		if err == nil {
			t.Fatal("expected error when context is canceled")
		}
	})

	t.Run("context deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Give time for context to expire
		time.Sleep(10 * time.Millisecond)

		_, err := m.Compact(ctx, "test history", "", "", t.TempDir())
		if err == nil {
			t.Fatal("expected error when context deadline exceeded")
		}
	})
}

func TestRelayMonitor_Compact_WriteFailure(t *testing.T) {
	adapter := &mockCompactionAdapter{
		runFunc: func(ctx context.Context, cfg CompactionConfig) (string, error) {
			return "compacted result", nil
		},
	}

	m := NewRelayMonitor(RelayMonitorConfig{}, adapter)

	t.Run("fails when workspace is read-only", func(t *testing.T) {
		workspacePath := t.TempDir()
		// Make directory read-only
		os.Chmod(workspacePath, 0555)
		defer os.Chmod(workspacePath, 0755) // Restore for cleanup

		_, err := m.Compact(context.Background(), "test history", "", "", workspacePath)
		if err == nil {
			t.Fatal("expected error when cannot write checkpoint")
		}

		if !strings.Contains(err.Error(), "failed to write checkpoint") {
			t.Errorf("expected write checkpoint error, got: %v", err)
		}
	})

	t.Run("fails when workspace does not exist", func(t *testing.T) {
		_, err := m.Compact(context.Background(), "test history", "", "", "/nonexistent/path")
		if err == nil {
			t.Fatal("expected error when workspace does not exist")
		}
	})
}

func TestRelayMonitor_Compact_EmptyCompactionResult(t *testing.T) {
	adapter := &mockCompactionAdapter{
		runFunc: func(ctx context.Context, cfg CompactionConfig) (string, error) {
			return "", nil // Empty result but no error
		},
	}

	m := NewRelayMonitor(RelayMonitorConfig{}, adapter)
	workspacePath := t.TempDir()

	result, err := m.Compact(context.Background(), "test history", "", "", workspacePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "" {
		t.Errorf("expected empty result, got: %s", result)
	}

	// Checkpoint should still be written, just with empty summary
	checkpointPath := workspacePath + "/checkpoint.md"
	if _, err := os.Stat(checkpointPath); os.IsNotExist(err) {
		t.Error("checkpoint should be written even with empty compaction result")
	}
}

// =============================================================================
// Additional Relay Monitor Tests
// =============================================================================

func TestRelayMonitor_GetTokenCount(t *testing.T) {
	adapter := &mockCompactionAdapter{}
	m := NewRelayMonitor(RelayMonitorConfig{}, adapter)

	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 1, // strings.Split returns [""] for empty string
		},
		{
			name:     "single word",
			input:    "hello",
			expected: 1,
		},
		{
			name:     "multiple words",
			input:    "hello world how are you",
			expected: 5,
		},
		{
			name:     "words with punctuation",
			input:    "hello, world! how are you?",
			expected: 5,
		},
		{
			name:     "multiple spaces",
			input:    "hello  world",
			expected: 3, // Split on single space creates empty element
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := m.GetTokenCount(tc.input)
			if result != tc.expected {
				t.Errorf("expected %d, got %d for input: %q", tc.expected, result, tc.input)
			}
		})
	}
}

func TestAdapterRunnerWrapper_RunCompaction_ErrorHandling(t *testing.T) {
	t.Run("returns error from adapter", func(t *testing.T) {
		expectedErr := errors.New("adapter failed")
		mockRunner := &mockAdapterRunner{
			runFunc: func(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error) {
				return nil, expectedErr
			},
		}

		wrapper := &AdapterRunnerWrapper{
			Runner:      mockRunner,
			AdapterName: "claude",
			PersonaName: "summarizer",
		}

		_, err := wrapper.RunCompaction(context.Background(), CompactionConfig{
			WorkspacePath: t.TempDir(),
			ChatHistory:   "test",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !strings.Contains(err.Error(), "adapter run failed") {
			t.Errorf("expected 'adapter run failed' in error, got: %v", err)
		}
	})

	t.Run("handles empty stdout", func(t *testing.T) {
		mockRunner := &mockAdapterRunner{
			runFunc: func(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error) {
				return &AdapterResult{
					ExitCode: 0,
					Stdout:   &mockReader{data: []byte{}},
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
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "" {
			t.Errorf("expected empty result, got: %s", result)
		}
	})

	t.Run("handles nil stdout", func(t *testing.T) {
		mockRunner := &mockAdapterRunner{
			runFunc: func(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error) {
				return &AdapterResult{
					ExitCode: 0,
					Stdout:   nil,
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
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "" {
			t.Errorf("expected empty result for nil stdout, got: %s", result)
		}
	})

	t.Run("handles nil result", func(t *testing.T) {
		mockRunner := &mockAdapterRunner{
			runFunc: func(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error) {
				return nil, nil
			},
		}

		wrapper := &AdapterRunnerWrapper{
			Runner:      mockRunner,
			AdapterName: "claude",
			PersonaName: "summarizer",
		}

		_, err := wrapper.RunCompaction(context.Background(), CompactionConfig{
			WorkspacePath: t.TempDir(),
		})

		if err == nil {
			t.Fatal("expected error for nil result, got nil")
		}

		if !errors.Is(err, ErrAdapterRunFailed) {
			t.Errorf("expected ErrAdapterRunFailed, got: %v", err)
		}
	})
}

// =============================================================================
// T071: Improved Error Handling Tests
// =============================================================================

func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name        string
		config      RelayMonitorConfig
		expectError bool
		errorType   error
	}{
		{
			name: "valid config",
			config: RelayMonitorConfig{
				DefaultThreshold:   80,
				MinTokensToCompact: 1000,
				ContextWindow:      200000,
			},
			expectError: false,
		},
		{
			name: "valid config with zero values (uses defaults)",
			config: RelayMonitorConfig{
				DefaultThreshold:   0,
				MinTokensToCompact: 0,
				ContextWindow:      0,
			},
			expectError: false,
		},
		{
			name: "invalid threshold - too high",
			config: RelayMonitorConfig{
				DefaultThreshold: 101,
			},
			expectError: true,
			errorType:   ErrInvalidThreshold,
		},
		{
			name: "invalid threshold - negative",
			config: RelayMonitorConfig{
				DefaultThreshold: -1,
			},
			expectError: true,
			errorType:   ErrInvalidThreshold,
		},
		{
			name: "invalid context window - negative",
			config: RelayMonitorConfig{
				ContextWindow: -1000,
			},
			expectError: true,
			errorType:   ErrInvalidContextWindow,
		},
		{
			name: "invalid min tokens - negative",
			config: RelayMonitorConfig{
				MinTokensToCompact: -100,
			},
			expectError: true,
		},
		{
			name: "boundary: threshold at 100",
			config: RelayMonitorConfig{
				DefaultThreshold: 100,
			},
			expectError: false,
		},
		{
			name: "boundary: threshold at 0",
			config: RelayMonitorConfig{
				DefaultThreshold: 0,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateConfig(tc.config)

			if tc.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errorType != nil && !errors.Is(err, tc.errorType) {
					t.Errorf("expected error type %v, got: %v", tc.errorType, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestIsCompactionError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ErrCompactionFailed",
			err:      ErrCompactionFailed,
			expected: true,
		},
		{
			name:     "wrapped ErrCompactionFailed",
			err:      fmt.Errorf("wrapper: %w", ErrCompactionFailed),
			expected: true,
		},
		{
			name:     "ErrAdapterRunFailed",
			err:      ErrAdapterRunFailed,
			expected: true,
		},
		{
			name:     "wrapped ErrAdapterRunFailed",
			err:      fmt.Errorf("wrapper: %w", ErrAdapterRunFailed),
			expected: true,
		},
		{
			name:     "unrelated error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "ErrNoAdapter is not a compaction error",
			err:      ErrNoAdapter,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsCompactionError(tc.err)
			if result != tc.expected {
				t.Errorf("IsCompactionError(%v) = %v, want %v", tc.err, result, tc.expected)
			}
		})
	}
}

func TestIsCheckpointError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ErrWriteCheckpointFailed",
			err:      ErrWriteCheckpointFailed,
			expected: true,
		},
		{
			name:     "ErrCheckpointNotFound",
			err:      ErrCheckpointNotFound,
			expected: true,
		},
		{
			name:     "ErrInvalidCheckpoint",
			err:      ErrInvalidCheckpoint,
			expected: true,
		},
		{
			name:     "wrapped checkpoint error",
			err:      fmt.Errorf("context: %w", ErrCheckpointNotFound),
			expected: true,
		},
		{
			name:     "unrelated error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsCheckpointError(tc.err)
			if result != tc.expected {
				t.Errorf("IsCheckpointError(%v) = %v, want %v", tc.err, result, tc.expected)
			}
		})
	}
}

func TestRelayMonitor_ErrorTypes(t *testing.T) {
	t.Run("ErrNoAdapter when adapter is nil", func(t *testing.T) {
		m := NewRelayMonitor(RelayMonitorConfig{}, nil)
		_, err := m.Compact(context.Background(), "", "", "", t.TempDir())

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !errors.Is(err, ErrNoAdapter) {
			t.Errorf("expected ErrNoAdapter, got: %v", err)
		}
	})

	t.Run("wrapped errors preserve original", func(t *testing.T) {
		originalErr := errors.New("original error")
		adapter := &mockCompactionAdapter{
			runFunc: func(ctx context.Context, cfg CompactionConfig) (string, error) {
				return "", originalErr
			},
		}

		m := NewRelayMonitor(RelayMonitorConfig{}, adapter)
		_, err := m.Compact(context.Background(), "", "", "", t.TempDir())

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// The wrapped error should contain the original error message
		if !strings.Contains(err.Error(), "original error") {
			t.Errorf("error should contain original message, got: %v", err)
		}
	})
}

