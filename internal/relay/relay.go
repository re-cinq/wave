package relay

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Persona struct {
	Name          string   `json:"name"`
	SystemPrompt  string   `json:"systemPrompt"`
	CompactPrompt string   `json:"compactPrompt"`
	Commands      []string `json:"commands,omitempty"`
	Adapter       string   `json:"adapter,omitempty"`
	MaxTokens     int      `json:"maxTokens,omitempty"`
}

type Adapter interface {
	Compact(ctx context.Context, chatHistory string, persona Persona) (string, error)
}

type relayConfig struct {
	DefaultThreshold   int `json:"defaultThreshold"`
	MinTokensToCompact int `json:"minTokensToCompact"`
}

type RelayMonitor struct {
	config relayConfig
}

func NewRelayMonitor(cfg relayConfig) *RelayMonitor {
	if cfg.DefaultThreshold == 0 {
		cfg.DefaultThreshold = 80
	}
	if cfg.MinTokensToCompact == 0 {
		cfg.MinTokensToCompact = 1000
	}
	return &RelayMonitor{config: cfg}
}

func (m *RelayMonitor) ShouldCompact(tokensUsed int, contextWindow int, thresholdPercent int) bool {
	if thresholdPercent == 0 {
		thresholdPercent = m.config.DefaultThreshold
	}
	threshold := (contextWindow * thresholdPercent) / 100
	return tokensUsed >= threshold && tokensUsed >= m.config.MinTokensToCompact
}

func (m *RelayMonitor) Compact(ctx context.Context, chatHistory string, summarizerPersona Persona, adapter Adapter, workspacePath string) (string, error) {
	if adapter == nil {
		return "", fmt.Errorf("no adapter provided for compaction")
	}

	compactPrompt := summarizerPersona.CompactPrompt
	if compactPrompt == "" {
		compactPrompt = "Summarize this conversation history concisely, preserving key context and decisions:"
	}

	compacted, err := adapter.Compact(ctx, chatHistory, summarizerPersona)
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

func (m *RelayMonitor) GetTokenCount(chatHistory string) int {
	return len(strings.Split(chatHistory, " "))
}
