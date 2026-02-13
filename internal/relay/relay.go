package relay

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Relay-specific errors
var (
	// ErrNoAdapter is returned when compaction is attempted without an adapter.
	ErrNoAdapter = errors.New("no adapter provided for compaction")

	// ErrCompactionFailed is returned when the compaction process fails.
	ErrCompactionFailed = errors.New("compaction failed")

	// ErrWriteCheckpointFailed is returned when writing the checkpoint file fails.
	ErrWriteCheckpointFailed = errors.New("failed to write checkpoint")

	// ErrInvalidThreshold is returned when an invalid threshold is provided.
	ErrInvalidThreshold = errors.New("invalid threshold: must be between 0 and 100")

	// ErrInvalidContextWindow is returned when an invalid context window is provided.
	ErrInvalidContextWindow = errors.New("invalid context window: must be positive")

	// ErrAdapterRunFailed is returned when the adapter runner fails.
	ErrAdapterRunFailed = errors.New("adapter run failed")
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
	DefaultThreshold   int `json:"defaultThreshold"`
	MinTokensToCompact int `json:"minTokensToCompact"`
	ContextWindow      int `json:"contextWindow"`
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
		Timeout:       5 * time.Minute,
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

// ValidateConfig validates the relay monitor configuration.
// Returns nil if the configuration is valid, or an error describing the issue.
func ValidateConfig(cfg RelayMonitorConfig) error {
	if cfg.DefaultThreshold < 0 || cfg.DefaultThreshold > 100 {
		return fmt.Errorf("%w: got %d", ErrInvalidThreshold, cfg.DefaultThreshold)
	}
	if cfg.ContextWindow < 0 {
		return fmt.Errorf("%w: got %d", ErrInvalidContextWindow, cfg.ContextWindow)
	}
	if cfg.MinTokensToCompact < 0 {
		return fmt.Errorf("invalid min tokens: must be non-negative, got %d", cfg.MinTokensToCompact)
	}
	return nil
}

// GetContextWindow returns the configured context window.
func (m *RelayMonitor) GetContextWindow() int {
	return m.config.ContextWindow
}

func (m *RelayMonitor) GetTokenCount(chatHistory string) int {
	return len(strings.Split(chatHistory, " "))
}

// AdapterRunnerWrapper wraps an adapter runner to implement CompactionAdapter.
// This allows reusing the existing adapter infrastructure for compaction.
type AdapterRunnerWrapper struct {
	Runner      AdapterRunner
	AdapterName string
	PersonaName string
}

// AdapterRunner is a subset of adapter.AdapterRunner for compaction purposes.
type AdapterRunner interface {
	Run(ctx context.Context, cfg AdapterRunnerConfig) (*AdapterResult, error)
}

// AdapterRunnerConfig mirrors the config needed for adapter runs.
type AdapterRunnerConfig struct {
	Adapter       string
	Persona       string
	WorkspacePath string
	Prompt        string
	SystemPrompt  string
	Timeout       time.Duration
	Temperature   float64
	AllowedTools  []string
	DenyTools     []string
	OutputFormat  string
}

// AdapterResult mirrors the adapter result structure.
type AdapterResult struct {
	ExitCode   int
	Stdout     StringReader
	TokensUsed int
	Artifacts  []string
}

// StringReader is a minimal interface for reading stdout.
type StringReader interface {
	Read(p []byte) (n int, err error)
}

// RunCompaction implements CompactionAdapter by running the adapter with a compaction prompt.
// It returns the summarized content and any error that occurred.
//
// Errors:
//   - ErrAdapterRunFailed: if the adapter fails to run (wraps original error)
//   - context errors: if the context is canceled or times out
func (w *AdapterRunnerWrapper) RunCompaction(ctx context.Context, cfg CompactionConfig) (string, error) {
	// Validate context
	if ctx == nil {
		return "", fmt.Errorf("%w: nil context", ErrAdapterRunFailed)
	}

	// Validate runner
	if w.Runner == nil {
		return "", fmt.Errorf("%w: nil runner", ErrAdapterRunFailed)
	}

	// Check context before starting
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("%w: %v", ErrAdapterRunFailed, ctx.Err())
	default:
	}

	// Build the compaction prompt combining the chat history and compact instruction
	prompt := fmt.Sprintf("%s\n\n---\n\nConversation history to summarize:\n%s", cfg.CompactPrompt, cfg.ChatHistory)

	runCfg := AdapterRunnerConfig{
		Adapter:       w.AdapterName,
		Persona:       w.PersonaName,
		WorkspacePath: cfg.WorkspacePath,
		Prompt:        prompt,
		SystemPrompt:  cfg.SystemPrompt,
		Timeout:       cfg.Timeout,
		Temperature:   0.3, // Lower temperature for summarization
		AllowedTools:  []string{"Read", "Glob", "Grep"}, // Read-only tools for compaction
		OutputFormat:  "text",
	}

	result, err := w.Runner.Run(ctx, runCfg)
	if err != nil {
		return "", fmt.Errorf("adapter run failed: %w", err)
	}

	// Validate result
	if result == nil {
		return "", fmt.Errorf("%w: nil result returned", ErrAdapterRunFailed)
	}

	if result.Stdout == nil {
		return "", nil // No output, but not an error
	}

	// Read all output from stdout with size limit to prevent memory issues
	const maxOutputSize = 1024 * 1024 // 1MB limit
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 1024)
	for {
		n, readErr := result.Stdout.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			// Enforce size limit
			if len(buf) > maxOutputSize {
				buf = buf[:maxOutputSize]
				break
			}
		}
		if readErr != nil {
			break
		}
	}

	return string(buf), nil
}

// IsCompactionError returns true if the error is related to compaction failure.
func IsCompactionError(err error) bool {
	return errors.Is(err, ErrCompactionFailed) || errors.Is(err, ErrAdapterRunFailed)
}

// IsCheckpointError returns true if the error is related to checkpoint operations.
func IsCheckpointError(err error) bool {
	return errors.Is(err, ErrWriteCheckpointFailed) ||
		errors.Is(err, ErrCheckpointNotFound) ||
		errors.Is(err, ErrInvalidCheckpoint)
}
