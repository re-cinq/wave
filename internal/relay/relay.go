package relay

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/timeouts"
)

// Relay-specific errors
var (
	// ErrNoAdapter is returned when compaction is attempted without an adapter.
	ErrNoAdapter = errors.New("no adapter provided for compaction")

	// ErrCompactionFailed is returned when the compaction process fails.
	ErrCompactionFailed = errors.New("compaction failed")

	// ErrWriteCheckpointFailed is returned when writing the checkpoint file fails.
	ErrWriteCheckpointFailed = errors.New("failed to write checkpoint")
)

// CompactionAdapter is the interface for running compaction.
// It is designed to be compatible with adapter.AdapterRunner.
type CompactionAdapter interface {
	RunCompaction(ctx context.Context, cfg CompactionConfig) (string, error)
}

// CompactionConfig holds configuration for a compaction run.
type CompactionConfig struct {
	WorkspacePath string
	ChatHistory   string
	SystemPrompt  string
	CompactPrompt string
	Timeout       time.Duration
}

// RelayMonitorConfig holds configuration for the relay monitor.
type RelayMonitorConfig struct {
	DefaultThreshold   int           `json:"defaultThreshold"`
	MinTokensToCompact int           `json:"minTokensToCompact"`
	ContextWindow      int           `json:"contextWindow"`
	CompactionTimeout  time.Duration `json:"-"` // set from manifest.Timeouts.GetRelayCompaction()
}

// RelayMonitor monitors token usage and triggers compaction when needed.
type RelayMonitor struct {
	config  RelayMonitorConfig
	adapter CompactionAdapter
}

// NewRelayMonitor creates a new RelayMonitor with the given configuration.
func NewRelayMonitor(cfg RelayMonitorConfig, adapter CompactionAdapter) *RelayMonitor {
	if cfg.DefaultThreshold == 0 {
		cfg.DefaultThreshold = 70
	}
	if cfg.MinTokensToCompact == 0 {
		cfg.MinTokensToCompact = 1000
	}
	if cfg.ContextWindow == 0 {
		cfg.ContextWindow = 200000 // Default to 200k tokens (Claude's context window)
	}
	return &RelayMonitor{config: cfg, adapter: adapter}
}

// Adapter returns the underlying CompactionAdapter, or nil if none was provided.
func (m *RelayMonitor) Adapter() CompactionAdapter {
	return m.adapter
}

// ShouldCompact determines if compaction should be triggered based on token usage.
func (m *RelayMonitor) ShouldCompact(tokensUsed int, thresholdPercent int) bool {
	if thresholdPercent == 0 {
		thresholdPercent = m.config.DefaultThreshold
	}
	contextWindow := m.config.ContextWindow
	threshold := (contextWindow * thresholdPercent) / 100
	return tokensUsed >= threshold && tokensUsed >= m.config.MinTokensToCompact
}

// ShouldCompactWithWindow determines if compaction should be triggered, using a custom context window.
func (m *RelayMonitor) ShouldCompactWithWindow(tokensUsed int, contextWindow int, thresholdPercent int) bool {
	if thresholdPercent == 0 {
		thresholdPercent = m.config.DefaultThreshold
	}
	if contextWindow == 0 {
		contextWindow = m.config.ContextWindow
	}
	threshold := (contextWindow * thresholdPercent) / 100
	return tokensUsed >= threshold && tokensUsed >= m.config.MinTokensToCompact
}

// Compact triggers compaction using the configured adapter and writes a checkpoint.
// It returns the compacted summary and any error that occurred.
//
// Errors:
//   - ErrNoAdapter: if no adapter is configured
//   - ErrCompactionFailed: if the adapter fails to compact (wraps original error)
//   - ErrWriteCheckpointFailed: if writing the checkpoint file fails (wraps original error)
//   - context.Canceled/context.DeadlineExceeded: if the context is canceled or times out
func (m *RelayMonitor) Compact(ctx context.Context, chatHistory string, systemPrompt string, compactPrompt string, workspacePath string) (string, error) {
	// Validate context
	if ctx == nil {
		return "", fmt.Errorf("%w: nil context", ErrCompactionFailed)
	}

	// Validate adapter is available
	if m.adapter == nil {
		return "", ErrNoAdapter
	}

	// Check context before starting expensive operation
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("%w: %v", ErrCompactionFailed, ctx.Err())
	default:
	}

	if compactPrompt == "" {
		compactPrompt = "Summarize this conversation history concisely, preserving key context and decisions:"
	}

	cfg := CompactionConfig{
		WorkspacePath: workspacePath,
		ChatHistory:   chatHistory,
		SystemPrompt:  systemPrompt,
		CompactPrompt: compactPrompt,
		Timeout:       m.compactionTimeout(),
	}

	compacted, err := m.adapter.RunCompaction(ctx, cfg)
	if err != nil {
		// Wrap specific error types for better error handling upstream
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", fmt.Errorf("%w: %v", ErrCompactionFailed, err)
		}
		return "", fmt.Errorf("compaction failed: %w", err)
	}

	// Write checkpoint file
	checkpointPath := filepath.Join(workspacePath, CheckpointFilename)
	checkpointContent := fmt.Sprintf(`# Checkpoint

## Summary
%s

---
*Generated at checkpoint - previous context preserved*
`, compacted)

	if err := os.WriteFile(checkpointPath, []byte(checkpointContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write checkpoint: %w", err)
	}

	return compacted, nil
}

func (m *RelayMonitor) compactionTimeout() time.Duration {
	if m.config.CompactionTimeout > 0 {
		return m.config.CompactionTimeout
	}
	return timeouts.RelayCompaction
}

// getContextWindow returns the configured context window.
func (m *RelayMonitor) getContextWindow() int {
	return m.config.ContextWindow
}

func (m *RelayMonitor) getTokenCount(chatHistory string) int {
	return len(strings.Split(chatHistory, " "))
}
