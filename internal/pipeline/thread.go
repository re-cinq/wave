package pipeline

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/recinq/wave/internal/relay"
)

// defaultMaxTranscriptSize is the maximum number of characters per thread transcript.
// When exceeded, oldest entries are trimmed. ~25k tokens at 4 chars/token.
const defaultMaxTranscriptSize = 100_000

// compactTruncateLen is the max chars of content shown per entry in compact fidelity.
const compactTruncateLen = 500

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
	compactionAdapter relay.CompactionAdapter // for summary fidelity (may be nil)
}

// NewThreadManager creates a ThreadManager with the given compaction adapter
// (used for "summary" fidelity). The adapter may be nil — summary fidelity
// will fall back to compact.
func NewThreadManager(adapter relay.CompactionAdapter) *ThreadManager {
	return &ThreadManager{
		transcripts:       make(map[string][]ThreadEntry),
		maxTranscriptSize: defaultMaxTranscriptSize,
		compactionAdapter: adapter,
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

// ThreadSize returns the total character count for a thread's transcript.
func (tm *ThreadManager) ThreadSize(threadID string) int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	total := 0
	for _, entry := range tm.transcripts[threadID] {
		total += len(entry.Content)
	}
	return total
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

// formatSummary delegates to the relay CompactionAdapter for LLM-generated
// summarization. Falls back to formatCompact on error or when no adapter is set.
func (tm *ThreadManager) formatSummary(ctx context.Context, entries []ThreadEntry) string {
	if tm.compactionAdapter == nil {
		return tm.formatCompact(entries)
	}

	// Build full transcript for compaction input
	fullTranscript := tm.formatFull(entries)

	result, err := tm.compactionAdapter.RunCompaction(ctx, relay.CompactionConfig{
		ChatHistory:   fullTranscript,
		SystemPrompt:  "Summarize the following conversation transcript from a multi-step pipeline execution. Focus on key decisions, actions taken, errors encountered, and current state. Be concise.",
		CompactPrompt: "Provide a structured summary of the conversation so far.",
		Timeout:       60 * time.Second,
	})
	if err != nil {
		// Fall back to compact on compaction failure
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
