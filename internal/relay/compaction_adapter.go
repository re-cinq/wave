package relay

import (
	"context"
	"fmt"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
)

// AdapterCompactionRunner bridges adapter.AdapterRegistry to the
// relay.CompactionAdapter interface. It resolves the persona named
// "summarizer" from the manifest (falling back to the first declared
// adapter, then to "claude") and dispatches a single-shot adapter run
// whose ResultContent becomes the compacted summary.
//
// Pure orchestration — no state, safe for concurrent reuse so long as
// the underlying registry and manifest are concurrency-safe.
type AdapterCompactionRunner struct {
	Registry *adapter.AdapterRegistry
	Manifest *manifest.Manifest
}

// NewAdapterCompactionRunner constructs an AdapterCompactionRunner.
// Both registry and m may be nil; nil values are handled defensively
// at run time but make compaction a no-op error.
func NewAdapterCompactionRunner(registry *adapter.AdapterRegistry, m *manifest.Manifest) *AdapterCompactionRunner {
	return &AdapterCompactionRunner{Registry: registry, Manifest: m}
}

// RunCompaction implements relay.CompactionAdapter.
func (a *AdapterCompactionRunner) RunCompaction(ctx context.Context, cfg CompactionConfig) (string, error) {
	prompt := cfg.CompactPrompt
	if cfg.ChatHistory != "" {
		prompt = fmt.Sprintf("%s\n\n---\n\nConversation history to summarize:\n%s", cfg.CompactPrompt, cfg.ChatHistory)
	}

	adapterName := ""
	if a.Manifest != nil {
		if p := a.Manifest.GetPersona("summarizer"); p != nil {
			adapterName = p.Adapter
		}
		if adapterName == "" {
			for name := range a.Manifest.Adapters {
				adapterName = name
				break
			}
		}
	}
	if adapterName == "" {
		adapterName = "claude"
	}

	if a.Registry == nil {
		return "", fmt.Errorf("compaction adapter: registry is nil")
	}
	compactionRunner := a.Registry.Resolve(adapterName)
	result, err := compactionRunner.Run(ctx, adapter.AdapterRunConfig{
		Adapter:       adapterName,
		Persona:       "summarizer",
		WorkspacePath: cfg.WorkspacePath,
		Prompt:        prompt,
		SystemPrompt:  cfg.SystemPrompt,
		Timeout:       cfg.Timeout,
		OutputFormat:  "text",
	})
	if err != nil {
		return "", fmt.Errorf("compaction adapter failed: %w", err)
	}

	return result.ResultContent, nil
}
