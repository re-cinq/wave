package pipeline

import (
	"context"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/relay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestExecution() *PipelineExecution {
	return &PipelineExecution{
		ThreadTranscripts: make(map[string][]ThreadEntry),
	}
}

func TestThreadManager_AppendAndGetTranscript(t *testing.T) {
	exec := newTestExecution()
	tm := NewThreadManager(exec, nil)

	tm.AppendTranscript("impl", "step-a", "step A output")
	tm.AppendTranscript("impl", "step-b", "step B output")

	transcript, err := tm.GetTranscript(context.Background(), "impl", FidelityFull)
	require.NoError(t, err)

	assert.Contains(t, transcript, "### Step: step-a")
	assert.Contains(t, transcript, "step A output")
	assert.Contains(t, transcript, "### Step: step-b")
	assert.Contains(t, transcript, "step B output")
	assert.Contains(t, transcript, "## Prior Conversation Context (thread: impl)")
}

func TestThreadManager_EmptyThreadGroup(t *testing.T) {
	exec := newTestExecution()
	tm := NewThreadManager(exec, nil)

	transcript, err := tm.GetTranscript(context.Background(), "nonexistent", FidelityFull)
	require.NoError(t, err)
	assert.Empty(t, transcript)
}

func TestThreadManager_FidelityFresh(t *testing.T) {
	exec := newTestExecution()
	tm := NewThreadManager(exec, nil)

	tm.AppendTranscript("impl", "step-a", "step A output")

	transcript, err := tm.GetTranscript(context.Background(), "impl", FidelityFresh)
	require.NoError(t, err)
	assert.Empty(t, transcript)
}

func TestThreadManager_FidelityCompact(t *testing.T) {
	exec := newTestExecution()
	tm := NewThreadManager(exec, nil)

	tm.AppendTranscript("impl", "step-a", "short output")
	tm.AppendTranscript("impl", "step-b", strings.Repeat("x", 1000))

	transcript, err := tm.GetTranscript(context.Background(), "impl", FidelityCompact)
	require.NoError(t, err)

	assert.Contains(t, transcript, "[compact]")
	assert.Contains(t, transcript, "### Step: step-a")
	assert.Contains(t, transcript, "short output")
	// Long content should be truncated
	assert.Contains(t, transcript, "... (truncated)")
}

func TestThreadManager_FidelityDefaultsToFull(t *testing.T) {
	exec := newTestExecution()
	tm := NewThreadManager(exec, nil)

	tm.AppendTranscript("impl", "step-a", "output")

	transcript, err := tm.GetTranscript(context.Background(), "impl", "")
	require.NoError(t, err)
	assert.Contains(t, transcript, "## Prior Conversation Context (thread: impl)")
	assert.NotContains(t, transcript, "[compact]")
	assert.NotContains(t, transcript, "[summary]")
}

func TestThreadManager_UnknownFidelity(t *testing.T) {
	exec := newTestExecution()
	tm := NewThreadManager(exec, nil)

	tm.AppendTranscript("impl", "step-a", "output")

	_, err := tm.GetTranscript(context.Background(), "impl", "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown fidelity level")
}

func TestThreadManager_ThreadGroupIsolation(t *testing.T) {
	exec := newTestExecution()
	tm := NewThreadManager(exec, nil)

	tm.AppendTranscript("impl", "step-a", "impl output")
	tm.AppendTranscript("review", "step-b", "review output")

	implTranscript, err := tm.GetTranscript(context.Background(), "impl", FidelityFull)
	require.NoError(t, err)
	assert.Contains(t, implTranscript, "impl output")
	assert.NotContains(t, implTranscript, "review output")

	reviewTranscript, err := tm.GetTranscript(context.Background(), "review", FidelityFull)
	require.NoError(t, err)
	assert.Contains(t, reviewTranscript, "review output")
	assert.NotContains(t, reviewTranscript, "impl output")
}

func TestThreadManager_TranscriptSizeCap(t *testing.T) {
	exec := newTestExecution()
	tm := NewThreadManager(exec, nil)
	tm.maxTranscriptSize = 100 // very small cap

	// Add content that exceeds the cap
	tm.AppendTranscript("impl", "step-a", strings.Repeat("A", 60))
	tm.AppendTranscript("impl", "step-b", strings.Repeat("B", 60))

	// step-a should be truncated, step-b should remain
	exec.mu.Lock()
	entries := exec.ThreadTranscripts["impl"]
	exec.mu.Unlock()

	require.Len(t, entries, 1, "oldest entry should be truncated")
	assert.Equal(t, "step-b", entries[0].StepID)
}

func TestThreadManager_TranscriptSizeCap_PreservesLastEntry(t *testing.T) {
	exec := newTestExecution()
	tm := NewThreadManager(exec, nil)
	tm.maxTranscriptSize = 10

	// Add a single entry larger than the cap — it should still be preserved
	tm.AppendTranscript("impl", "step-a", strings.Repeat("X", 200))

	exec.mu.Lock()
	entries := exec.ThreadTranscripts["impl"]
	exec.mu.Unlock()

	require.Len(t, entries, 1, "single entry should never be removed")
	assert.Equal(t, "step-a", entries[0].StepID)
}

// mockCompactor implements relay.CompactionAdapter for testing summary fidelity.
type mockCompactor struct {
	result string
	err    error
}

func (m *mockCompactor) RunCompaction(_ context.Context, _ relay.CompactionConfig) (string, error) {
	return m.result, m.err
}

func TestThreadManager_FidelitySummary_WithCompactor(t *testing.T) {
	exec := newTestExecution()
	compactor := &mockCompactor{result: "Summary: steps completed successfully."}
	tm := NewThreadManager(exec, compactor)

	tm.AppendTranscript("impl", "step-a", "detailed output")

	transcript, err := tm.GetTranscript(context.Background(), "impl", FidelitySummary)
	require.NoError(t, err)
	assert.Contains(t, transcript, "[summary]")
	assert.Contains(t, transcript, "Summary: steps completed successfully.")
}

func TestThreadManager_FidelitySummary_FallsBackToCompact(t *testing.T) {
	exec := newTestExecution()
	// No compactor — should fall back to compact
	tm := NewThreadManager(exec, nil)

	tm.AppendTranscript("impl", "step-a", "output")

	transcript, err := tm.GetTranscript(context.Background(), "impl", FidelitySummary)
	require.NoError(t, err)
	assert.Contains(t, transcript, "[compact]", "should fall back to compact when no compactor")
}
