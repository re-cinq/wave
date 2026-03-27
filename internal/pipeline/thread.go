package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/recinq/wave/internal/relay"
)

// DefaultMaxTranscriptSize is the maximum number of characters stored per thread group.
// When exceeded, oldest entries are truncated first.
const DefaultMaxTranscriptSize = 100_000

// ThreadEntry holds a single step's contribution to a thread conversation.
type ThreadEntry struct {
	StepID    string    `json:"step_id"`
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
}

// ThreadManager manages conversation transcripts for thread groups within a pipeline execution.
type ThreadManager struct {
	execution        *PipelineExecution
	maxTranscriptSize int
	compactor        relay.CompactionAdapter
}

// NewThreadManager creates a ThreadManager for the given execution.
// compactor may be nil — summary fidelity falls back to compact if unavailable.
func NewThreadManager(execution *PipelineExecution, compactor relay.CompactionAdapter) *ThreadManager {
	return &ThreadManager{
		execution:        execution,
		maxTranscriptSize: DefaultMaxTranscriptSize,
		compactor:        compactor,
	}
}

// AppendTranscript adds a step's output to the thread group transcript.
// If the total transcript size exceeds the cap, oldest entries are removed.
func (tm *ThreadManager) AppendTranscript(threadGroup, stepID, content string) {
	tm.execution.mu.Lock()
	defer tm.execution.mu.Unlock()

	entry := ThreadEntry{
		StepID:    stepID,
		Timestamp: time.Now(),
		Content:   content,
	}

	tm.execution.ThreadTranscripts[threadGroup] = append(
		tm.execution.ThreadTranscripts[threadGroup], entry,
	)

	// Enforce size cap — remove oldest entries until under limit
	tm.truncateLocked(threadGroup)
}

// GetTranscript returns the formatted transcript for a thread group based on fidelity level.
// Returns empty string if the thread group has no entries or fidelity is "fresh".
func (tm *ThreadManager) GetTranscript(ctx context.Context, threadGroup, fidelity string) (string, error) {
	tm.execution.mu.Lock()
	entries := tm.execution.ThreadTranscripts[threadGroup]
	// Copy entries under lock to avoid holding lock during formatting
	entriesCopy := make([]ThreadEntry, len(entries))
	copy(entriesCopy, entries)
	tm.execution.mu.Unlock()

	if len(entriesCopy) == 0 {
		return "", nil
	}

	// Default to full fidelity when thread is set but fidelity is empty
	if fidelity == "" {
		fidelity = FidelityFull
	}

	return tm.FormatPreamble(ctx, threadGroup, entriesCopy, fidelity)
}

// FormatPreamble transforms transcript entries into a fidelity-appropriate preamble string.
func (tm *ThreadManager) FormatPreamble(ctx context.Context, threadGroup string, entries []ThreadEntry, fidelity string) (string, error) {
	switch fidelity {
	case FidelityFresh:
		return "", nil

	case FidelityCompact:
		return tm.formatCompact(threadGroup, entries), nil

	case FidelitySummary:
		return tm.formatSummary(ctx, threadGroup, entries)

	case FidelityFull, "":
		return tm.formatFull(threadGroup, entries), nil

	default:
		return "", fmt.Errorf("unknown fidelity level: %q", fidelity)
	}
}

// formatFull returns all transcript entries verbatim with step attribution headers.
func (tm *ThreadManager) formatFull(threadGroup string, entries []ThreadEntry) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Prior Conversation Context (thread: %s)\n\n", threadGroup))
	for _, entry := range entries {
		b.WriteString(fmt.Sprintf("### Step: %s\n\n", entry.StepID))
		b.WriteString(entry.Content)
		b.WriteString("\n\n")
	}
	return b.String()
}

// formatCompact returns a structured summary with step ID and truncated content.
func (tm *ThreadManager) formatCompact(threadGroup string, entries []ThreadEntry) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Prior Conversation Context (thread: %s) [compact]\n\n", threadGroup))
	for _, entry := range entries {
		b.WriteString(fmt.Sprintf("### Step: %s\n\n", entry.StepID))
		// Truncate content to first 500 chars for compact view
		content := entry.Content
		if len(content) > 500 {
			content = content[:500] + "\n... (truncated)"
		}
		b.WriteString(content)
		b.WriteString("\n\n")
	}
	return b.String()
}

// formatSummary uses relay CompactionAdapter to LLM-summarize the transcript.
// Falls back to compact if no compactor is available.
func (tm *ThreadManager) formatSummary(ctx context.Context, threadGroup string, entries []ThreadEntry) (string, error) {
	if tm.compactor == nil {
		return tm.formatCompact(threadGroup, entries), nil
	}

	// Build full transcript for summarization
	fullText := tm.formatFull(threadGroup, entries)

	summary, err := tm.compactor.RunCompaction(ctx, relay.CompactionConfig{
		ChatHistory:   fullText,
		CompactPrompt: "Summarize the following conversation transcript from a multi-step pipeline execution. Focus on: what was done, key decisions, outcomes, and any errors encountered. Be concise.",
	})
	if err != nil {
		// Fall back to compact on compaction failure
		return tm.formatCompact(threadGroup, entries), nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Prior Conversation Context (thread: %s) [summary]\n\n", threadGroup))
	b.WriteString(summary)
	b.WriteString("\n\n")
	return b.String(), nil
}

// truncateLocked removes oldest entries from a thread group until the total content size
// is within the max transcript size. Must be called with execution.mu held.
func (tm *ThreadManager) truncateLocked(threadGroup string) {
	entries := tm.execution.ThreadTranscripts[threadGroup]
	totalSize := 0
	for _, e := range entries {
		totalSize += len(e.Content)
	}

	for totalSize > tm.maxTranscriptSize && len(entries) > 1 {
		totalSize -= len(entries[0].Content)
		entries = entries[1:]
	}

	tm.execution.ThreadTranscripts[threadGroup] = entries
}
