package pipeline

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/recinq/wave/internal/adapter"
	"github.com/recinq/wave/internal/manifest"
)

// defaultMaxTranscriptSize is the maximum number of characters per thread transcript.
// When exceeded, oldest entries are trimmed. ~25k tokens at 4 chars/token.
const defaultMaxTranscriptSize = 100_000

// compactTruncateLen is the max chars of content shown per entry in compact fidelity.
const compactTruncateLen = 500

// summaryTimeout bounds the single LLM call behind "summary" fidelity.
const summaryTimeout = 60 * time.Second

const (
	summarySystemPrompt = "Summarize the following conversation transcript from a multi-step pipeline execution. Focus on key decisions, actions taken, errors encountered, and current state. Be concise."
	summaryPrompt       = "Provide a structured summary of the conversation so far."
)

// TranscriptSummarizer produces an LLM-generated summary of a thread
// transcript for "summary" fidelity.
type TranscriptSummarizer interface {
	Summarize(ctx context.Context, transcript string) (string, error)
}

// adapterSummarizer runs the "summarizer" persona's adapter (falling back to
// the first declared adapter, then "claude") as a single-shot text run.
type adapterSummarizer struct {
	registry *adapter.AdapterRegistry
	manifest *manifest.Manifest
}

// newAdapterSummarizer returns nil when no registry is available, which makes
// summary fidelity fall back to compact.
func newAdapterSummarizer(registry *adapter.AdapterRegistry, m *manifest.Manifest) TranscriptSummarizer {
	if registry == nil {
		return nil
	}
	return &adapterSummarizer{registry: registry, manifest: m}
}

func (s *adapterSummarizer) Summarize(ctx context.Context, transcript string) (string, error) {
	adapterName := ""
	if s.manifest != nil {
		if p := s.manifest.GetPersona("summarizer"); p != nil {
			adapterName = p.Adapter
		}
		if adapterName == "" {
			for name := range s.manifest.Adapters {
				adapterName = name
				break
			}
		}
	}
	if adapterName == "" {
		adapterName = "claude"
	}
	result, err := s.registry.Resolve(adapterName).Run(ctx, adapter.AdapterRunConfig{
		Adapter:      adapterName,
		Persona:      "summarizer",
		Prompt:       fmt.Sprintf("%s\n\n---\n\nConversation history to summarize:\n%s", summaryPrompt, transcript),
		SystemPrompt: summarySystemPrompt,
		Timeout:      summaryTimeout,
		OutputFormat: "text",
	})
	if err != nil {
		return "", fmt.Errorf("summary adapter failed: %w", err)
	}
	return result.ResultContent, nil
}

// ThreadEntry represents one step's contribution to a thread transcript.
type ThreadEntry struct {
	StepID    string
	Timestamp time.Time
	Content   string
}

// ThreadManager manages per-thread-group conversation transcripts within a
// single pipeline execution. It stores step output transcripts and formats
// them according to the requested fidelity level.
type ThreadManager struct {
	mu                sync.RWMutex
	transcripts       map[string][]ThreadEntry // threadID -> ordered entries
	maxTranscriptSize int
	summarizer        TranscriptSummarizer // for summary fidelity (may be nil)
}

// NewThreadManager creates a ThreadManager with the given summarizer (used
// for "summary" fidelity). The summarizer may be nil - summary fidelity will
// fall back to compact.
func NewThreadManager(summarizer TranscriptSummarizer) *ThreadManager {
	return &ThreadManager{
		transcripts:       make(map[string][]ThreadEntry),
		maxTranscriptSize: defaultMaxTranscriptSize,
		summarizer:        summarizer,
	}
}

// AppendTranscript adds a step's output to a thread group's transcript.
// If the resulting transcript exceeds the size cap, oldest entries are trimmed.
func (tm *ThreadManager) AppendTranscript(threadID, stepID string, content string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	entry := ThreadEntry{
		StepID:    stepID,
		Timestamp: time.Now(),
		Content:   content,
	}
	tm.transcripts[threadID] = append(tm.transcripts[threadID], entry)
	tm.enforceCapLocked(threadID)
}

// GetTranscript returns the formatted transcript for a thread group at the
// given fidelity level. Returns empty string for unknown threads or fresh fidelity.
func (tm *ThreadManager) GetTranscript(ctx context.Context, threadID, fidelity string) string {
	if fidelity == FidelityFresh {
		return ""
	}

	tm.mu.RLock()
	entries := tm.transcripts[threadID]
	tm.mu.RUnlock()

	if len(entries) == 0 {
		return ""
	}

	switch fidelity {
	case FidelityFull:
		return tm.formatFull(entries)
	case FidelityCompact:
		return tm.formatCompact(entries)
	case FidelitySummary:
		return tm.formatSummary(ctx, entries)
	default:
		return tm.formatFull(entries)
	}
}

// formatFull returns verbatim transcript entries with step attribution headers.
func (tm *ThreadManager) formatFull(entries []ThreadEntry) string {
	var sb strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&sb, "## Step: %s at %s\n\n", entry.StepID, entry.Timestamp.Format(time.RFC3339))
		sb.WriteString(entry.Content)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

// formatCompact returns step ID + truncated content for each entry.
func (tm *ThreadManager) formatCompact(entries []ThreadEntry) string {
	var sb strings.Builder
	for _, entry := range entries {
		content := entry.Content
		if len(content) > compactTruncateLen {
			content = content[:compactTruncateLen] + "..."
		}
		fmt.Fprintf(&sb, "### %s (completed)\n%s\n\n", entry.StepID, content)
	}
	return sb.String()
}

// formatSummary delegates to the summarizer for an LLM-generated summary.
// Falls back to formatCompact on error or when no summarizer is set.
func (tm *ThreadManager) formatSummary(ctx context.Context, entries []ThreadEntry) string {
	if tm.summarizer == nil {
		return tm.formatCompact(entries)
	}
	result, err := tm.summarizer.Summarize(ctx, tm.formatFull(entries))
	if err != nil {
		return tm.formatCompact(entries)
	}
	return result
}

// enforceCapLocked trims the oldest entries from a thread when the total size
// exceeds maxTranscriptSize. Must be called with tm.mu held.
func (tm *ThreadManager) enforceCapLocked(threadID string) {
	entries := tm.transcripts[threadID]
	totalSize := 0
	for _, entry := range entries {
		totalSize += len(entry.Content)
	}

	for totalSize > tm.maxTranscriptSize && len(entries) > 1 {
		totalSize -= len(entries[0].Content)
		entries = entries[1:]
	}
	tm.transcripts[threadID] = entries
}
