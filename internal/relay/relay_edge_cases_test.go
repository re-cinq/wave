package relay

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Edge Cases and Additional Error Handling Tests
// =============================================================================

func TestNewRelayMonitor_ConfigDefaults(t *testing.T) {
	testCases := []struct {
		name           string
		config         RelayMonitorConfig
		expectedConfig RelayMonitorConfig
	}{
		{
			name:   "all zeros - should use defaults",
			config: RelayMonitorConfig{},
			expectedConfig: RelayMonitorConfig{
				DefaultThreshold:   80,
				MinTokensToCompact: 1000,
				ContextWindow:      200000,
			},
		},
		{
			name: "partial config - should use defaults for zero values",
			config: RelayMonitorConfig{
				DefaultThreshold: 70,
			},
			expectedConfig: RelayMonitorConfig{
				DefaultThreshold:   70,
				MinTokensToCompact: 1000,
				ContextWindow:      200000,
			},
		},
		{
			name: "full config - should use provided values",
			config: RelayMonitorConfig{
				DefaultThreshold:   90,
				MinTokensToCompact: 2000,
				ContextWindow:      100000,
			},
			expectedConfig: RelayMonitorConfig{
				DefaultThreshold:   90,
				MinTokensToCompact: 2000,
				ContextWindow:      100000,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &mockCompactionAdapter{}
			m := NewRelayMonitor(tc.config, adapter)

			if m.config.DefaultThreshold != tc.expectedConfig.DefaultThreshold {
				t.Errorf("DefaultThreshold: expected %d, got %d", tc.expectedConfig.DefaultThreshold, m.config.DefaultThreshold)
			}
			if m.config.MinTokensToCompact != tc.expectedConfig.MinTokensToCompact {
				t.Errorf("MinTokensToCompact: expected %d, got %d", tc.expectedConfig.MinTokensToCompact, m.config.MinTokensToCompact)
			}
			if m.config.ContextWindow != tc.expectedConfig.ContextWindow {
				t.Errorf("ContextWindow: expected %d, got %d", tc.expectedConfig.ContextWindow, m.config.ContextWindow)
			}
		})
	}
}

func TestRelayMonitor_ShouldCompact_EdgeCases(t *testing.T) {
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
			name: "negative tokens used",
			config: RelayMonitorConfig{
				DefaultThreshold:   80,
				MinTokensToCompact: 1000,
				ContextWindow:      10000,
			},
			tokensUsed:       -100,
			thresholdPercent: 80,
			expected:         false,
			description:      "negative tokens should not trigger compaction",
		},
		{
			name: "negative threshold percent",
			config: RelayMonitorConfig{
				DefaultThreshold:   80,
				MinTokensToCompact: 1000,
				ContextWindow:      10000,
			},
			tokensUsed:       9000,
			thresholdPercent: -10,
			expected:         true, // Should use default threshold
			description:      "negative threshold should use default",
		},
		{
			name: "threshold over 100",
			config: RelayMonitorConfig{
				DefaultThreshold:   80,
				MinTokensToCompact: 1000,
				ContextWindow:      10000,
			},
			tokensUsed:       9500,
			thresholdPercent: 150,
			expected:         false, // 150% of 10000 = 15000, 9500 < 15000
			description:      "threshold over 100% should work mathematically",
		},
		{
			name: "zero context window in config",
			config: RelayMonitorConfig{
				DefaultThreshold:   80,
				MinTokensToCompact: 1000,
				ContextWindow:      0, // This triggers default
			},
			tokensUsed:       160000,
			thresholdPercent: 80,
			expected:         true, // 80% of 200000 (default) = 160000
			description:      "zero context window should use default",
		},
		{
			name: "very small context window",
			config: RelayMonitorConfig{
				DefaultThreshold:   80,
				MinTokensToCompact: 1000,
				ContextWindow:      100,
			},
			tokensUsed:       1000,
			thresholdPercent: 80,
			expected:         true, // 80% of 100 = 80, but 1000 >= minTokens
			description:      "small context window with sufficient tokens",
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

func TestRelayMonitor_Compact_EdgeCases(t *testing.T) {
	t.Run("empty chat history", func(t *testing.T) {
		adapter := &mockCompactionAdapter{
			runFunc: func(ctx context.Context, cfg CompactionConfig) (string, error) {
				if cfg.ChatHistory == "" {
					return "empty history summary", nil
				}
				return "regular summary", nil
			},
		}

		m := NewRelayMonitor(RelayMonitorConfig{}, adapter)
		result, err := m.Compact(context.Background(), "", "system", "compact", t.TempDir())

		if err != nil {
			t.Errorf("should handle empty chat history: %v", err)
		}
		if result != "empty history summary" {
			t.Errorf("expected 'empty history summary', got '%s'", result)
		}
	})

	t.Run("nil context", func(t *testing.T) {
		adapter := &mockCompactionAdapter{}
		m := NewRelayMonitor(RelayMonitorConfig{}, adapter)

		_, err := m.Compact(nil, "history", "", "", t.TempDir())
		if err == nil {
			t.Error("expected error with nil context")
		}
	})

	t.Run("empty compact prompt uses default", func(t *testing.T) {
		var receivedPrompt string
		adapter := &mockCompactionAdapter{
			runFunc: func(ctx context.Context, cfg CompactionConfig) (string, error) {
				receivedPrompt = cfg.CompactPrompt
				return "summary", nil
			},
		}

		m := NewRelayMonitor(RelayMonitorConfig{}, adapter)
		_, err := m.Compact(context.Background(), "history", "", "", t.TempDir())

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(receivedPrompt, "Summarize this conversation") {
			t.Errorf("should use default prompt, got: %s", receivedPrompt)
		}
	})

	t.Run("workspace path with spaces and special chars", func(t *testing.T) {
		adapter := &mockCompactionAdapter{
			runFunc: func(ctx context.Context, cfg CompactionConfig) (string, error) {
				return "summary", nil
			},
		}

		m := NewRelayMonitor(RelayMonitorConfig{}, adapter)

		// Create a workspace with spaces in the path
		baseDir := t.TempDir()
		workspacePath := filepath.Join(baseDir, "test workspace with spaces & chars!")
		err := os.MkdirAll(workspacePath, 0755)
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		_, err = m.Compact(context.Background(), "history", "", "", workspacePath)
		if err != nil {
			t.Errorf("should handle workspace paths with spaces: %v", err)
		}

		// Verify checkpoint was created
		checkpointPath := filepath.Join(workspacePath, "checkpoint.md")
		if _, err := os.Stat(checkpointPath); os.IsNotExist(err) {
			t.Error("checkpoint file should be created in special workspace path")
		}
	})
}

func TestAdapterRunnerWrapper_EdgeCases(t *testing.T) {
	t.Run("nil runner", func(t *testing.T) {
		wrapper := &AdapterRunnerWrapper{
			Runner:      nil,
			AdapterName: "claude",
			PersonaName: "summarizer",
		}

		_, err := wrapper.RunCompaction(context.Background(), CompactionConfig{
			WorkspacePath: t.TempDir(),
		})

		if err == nil {
			t.Error("expected error with nil runner")
		}
	})

	t.Run("empty adapter/persona names", func(t *testing.T) {
		var receivedConfig AdapterRunnerConfig
		mockRunner := &mockAdapterRunner{
			runFunc: func(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error) {
				receivedConfig = cfg
				return &AdapterResult{
					ExitCode: 0,
					Stdout:   &mockReader{data: []byte("result")},
				}, nil
			},
		}

		wrapper := &AdapterRunnerWrapper{
			Runner:      mockRunner,
			AdapterName: "",
			PersonaName: "",
		}

		_, err := wrapper.RunCompaction(context.Background(), CompactionConfig{
			WorkspacePath: t.TempDir(),
		})

		if err != nil {
			t.Errorf("should handle empty adapter/persona names: %v", err)
		}

		if receivedConfig.Adapter != "" || receivedConfig.Persona != "" {
			t.Errorf("expected empty adapter/persona, got adapter='%s', persona='%s'",
				receivedConfig.Adapter, receivedConfig.Persona)
		}
	})

	t.Run("very long chat history", func(t *testing.T) {
		longHistory := strings.Repeat("This is a very long conversation history. ", 10000)

		mockRunner := &mockAdapterRunner{
			runFunc: func(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error) {
				if len(cfg.Prompt) < len(longHistory) {
					t.Error("prompt should include the full chat history")
				}
				return &AdapterResult{
					ExitCode: 0,
					Stdout:   &mockReader{data: []byte("summary of long history")},
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
			ChatHistory:   longHistory,
		})

		if err != nil {
			t.Errorf("should handle long chat history: %v", err)
		}

		if result != "summary of long history" {
			t.Errorf("unexpected result: %s", result)
		}
	})

	t.Run("adapter returns zero exit code but error", func(t *testing.T) {
		mockRunner := &mockAdapterRunner{
			runFunc: func(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error) {
				return &AdapterResult{ExitCode: 0}, errors.New("runtime error")
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
			t.Error("expected error when adapter run fails")
		}
	})
}

func TestCheckpointParsing_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		shouldError bool
		description string
	}{
		{
			name: "checkpoint with only header",
			content: `# Checkpoint`,
			shouldError: false,
			description: "should handle minimal checkpoint",
		},
		{
			name: "checkpoint with multiple summary sections",
			content: `# Checkpoint

## Summary
First summary

## Summary
Second summary - this should not appear

## Decisions
Some decision
`,
			shouldError: false,
			description: "should handle multiple summary sections gracefully",
		},
		{
			name: "checkpoint with nested headers",
			content: `# Checkpoint

## Summary
Main summary

### Subsection
This should be ignored

## Decisions
Main decision
`,
			shouldError: false,
			description: "should ignore nested headers",
		},
		{
			name: "checkpoint with empty lines and whitespace",
			content: `# Checkpoint


## Summary


Summary with empty lines



## Decisions


Decision with empty lines


---
*Generated*


`,
			shouldError: false,
			description: "should handle extra whitespace gracefully",
		},
		{
			name: "checkpoint without sections",
			content: `# Checkpoint

This is just random content without proper sections.
It has multiple lines.
But no proper ## sections.
`,
			shouldError: false,
			description: "should handle malformed checkpoint without sections",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workspacePath := t.TempDir()
			checkpointPath := filepath.Join(workspacePath, CheckpointFilename)
			err := os.WriteFile(checkpointPath, []byte(tc.content), 0644)
			if err != nil {
				t.Fatalf("failed to write test checkpoint: %v", err)
			}

			checkpoint, err := ParseCheckpoint(workspacePath)

			if tc.shouldError && err == nil {
				t.Errorf("%s: expected error but got none", tc.description)
			} else if !tc.shouldError && err != nil {
				t.Errorf("%s: unexpected error: %v", tc.description, err)
			}

			if err == nil && checkpoint == nil {
				t.Errorf("%s: checkpoint should not be nil when no error", tc.description)
			}
		})
	}
}

func TestValidateConfig_ExtendedEdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		config      RelayMonitorConfig
		expectError bool
		description string
	}{
		{
			name: "threshold exactly 0",
			config: RelayMonitorConfig{
				DefaultThreshold: 0,
			},
			expectError: false,
			description: "0 is valid threshold (uses default)",
		},
		{
			name: "threshold exactly 100",
			config: RelayMonitorConfig{
				DefaultThreshold: 100,
			},
			expectError: false,
			description: "100 is valid threshold",
		},
		{
			name: "very large context window",
			config: RelayMonitorConfig{
				ContextWindow: 10000000, // 10M
			},
			expectError: false,
			description: "very large context window should be valid",
		},
		{
			name: "very large min tokens",
			config: RelayMonitorConfig{
				MinTokensToCompact: 1000000,
			},
			expectError: false,
			description: "very large min tokens should be valid",
		},
		{
			name: "all maximum values",
			config: RelayMonitorConfig{
				DefaultThreshold:   100,
				MinTokensToCompact: 1000000,
				ContextWindow:      10000000,
			},
			expectError: false,
			description: "all maximum valid values",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateConfig(tc.config)

			if tc.expectError && err == nil {
				t.Errorf("%s: expected error but got none", tc.description)
			} else if !tc.expectError && err != nil {
				t.Errorf("%s: unexpected error: %v", tc.description, err)
			}
		})
	}
}

func TestErrorClassification_ComplexCases(t *testing.T) {
	// Test deeply wrapped errors
	originalError := ErrCompactionFailed
	wrappedOnce := fmt.Errorf("level 1: %w", originalError)
	wrappedTwice := fmt.Errorf("level 2: %w", wrappedOnce)
	wrappedThrice := fmt.Errorf("level 3: %w", wrappedTwice)

	if !IsCompactionError(wrappedThrice) {
		t.Error("should detect deeply wrapped compaction error")
	}

	// Test error types that are both compaction and checkpoint errors
	mixedError := fmt.Errorf("mixed: %w and %w", ErrCompactionFailed, ErrWriteCheckpointFailed)

	if !IsCompactionError(mixedError) {
		t.Error("should detect compaction error in mixed error")
	}

	if !IsCheckpointError(mixedError) {
		t.Error("should detect checkpoint error in mixed error")
	}

	// Test nil cases
	if IsCompactionError(nil) {
		t.Error("nil should not be a compaction error")
	}

	if IsCheckpointError(nil) {
		t.Error("nil should not be a checkpoint error")
	}
}

func TestTimeout_EdgeCases(t *testing.T) {
	t.Run("very short timeout", func(t *testing.T) {
		adapter := &mockCompactionAdapter{
			runFunc: func(ctx context.Context, cfg CompactionConfig) (string, error) {
				// Simulate slow operation that respects context
				select {
				case <-time.After(100 * time.Millisecond):
					return "should not complete", nil
				case <-ctx.Done():
					return "", ctx.Err()
				}
			},
		}

		m := NewRelayMonitor(RelayMonitorConfig{}, adapter)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		_, err := m.Compact(ctx, "history", "", "", t.TempDir())
		if err == nil {
			t.Error("expected timeout error")
		}

		if !strings.Contains(err.Error(), "compaction failed") {
			t.Errorf("expected compaction failed error, got: %v", err)
		}
	})

	t.Run("cancelled context before start", func(t *testing.T) {
		adapter := &mockCompactionAdapter{}
		m := NewRelayMonitor(RelayMonitorConfig{}, adapter)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := m.Compact(ctx, "history", "", "", t.TempDir())
		if err == nil {
			t.Error("expected cancelled context error")
		}
	})
}