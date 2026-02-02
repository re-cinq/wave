package relay

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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
		cfg.DefaultThreshold = 80
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
func (m *RelayMonitor) Compact(ctx context.Context, chatHistory string, systemPrompt string, compactPrompt string, workspacePath string) (string, error) {
	if m.adapter == nil {
		return "", fmt.Errorf("no adapter provided for compaction")
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
		return "", fmt.Errorf("compaction failed: %w", err)
	}

	checkpointPath := filepath.Join(workspacePath, "checkpoint.md")
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
func (w *AdapterRunnerWrapper) RunCompaction(ctx context.Context, cfg CompactionConfig) (string, error) {
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

	// Read all output from stdout
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 1024)
	for {
		n, readErr := result.Stdout.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if readErr != nil {
			break
		}
	}

	return string(buf), nil
}
