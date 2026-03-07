package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newTestComposeList creates a ComposeListModel with the given number of
// entries. The first entry is always "pipeline-1" (added by the constructor).
// Additional entries are added via sequence.Add.
func newTestComposeList(entries int) ComposeListModel {
	initial := PipelineInfo{Name: "pipeline-1", Description: "First", StepCount: 2}
	initialPipeline := testPipeline("pipeline-1",
		[]pipeline.ArtifactDef{{Name: "output1"}}, nil)

	available := []PipelineInfo{
		{Name: "pipeline-1", Description: "First", StepCount: 2},
		{Name: "pipeline-2", Description: "Second", StepCount: 3},
		{Name: "pipeline-3", Description: "Third", StepCount: 1},
	}

	m := NewComposeListModel(initial, initialPipeline, available)
	m.SetSize(40, 20)
	m.SetFocused(true)

	// Add more entries if requested
	for i := 1; i < entries; i++ {
		name := fmt.Sprintf("pipeline-%d", i+1)
		p := testPipeline(name, []pipeline.ArtifactDef{{Name: fmt.Sprintf("output%d", i+1)}}, nil)
		m.sequence.Add(name, p)
		m.validation = ValidateSequence(m.sequence)
	}

	return m
}

// composeListSendKey sends a key event to a ComposeListModel and returns the
// updated model and cmd.
func composeListSendKey(m ComposeListModel, keyType tea.KeyType) (ComposeListModel, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: keyType})
}

// composeListSendRune sends a rune key event to a ComposeListModel.
func composeListSendRune(m ComposeListModel, r rune) (ComposeListModel, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
}

// extractMsg executes a tea.Cmd and returns a pointer to the message of type T
// if found, checking both direct and batched forms. Returns nil otherwise.
func extractMsg[T any](cmd tea.Cmd) *T {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	if m, ok := msg.(T); ok {
		return &m
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			inner := c()
			if m, ok := inner.(T); ok {
				return &m
			}
		}
	}
	return nil
}

// ===========================================================================
// ComposeListModel tests
// ===========================================================================

func TestComposeListModel(t *testing.T) {
	t.Run("cursor navigation down", func(t *testing.T) {
		m := newTestComposeList(3)
		require.Equal(t, 0, m.cursor)

		m, _ = composeListSendKey(m, tea.KeyDown)
		assert.Equal(t, 1, m.cursor)
	})

	t.Run("cursor navigation up", func(t *testing.T) {
		m := newTestComposeList(3)
		m.cursor = 1

		m, _ = composeListSendKey(m, tea.KeyUp)
		assert.Equal(t, 0, m.cursor)
	})

	t.Run("cursor stays in bounds", func(t *testing.T) {
		m := newTestComposeList(3)

		// At index 0, KeyUp should stay at 0
		require.Equal(t, 0, m.cursor)
		m, _ = composeListSendKey(m, tea.KeyUp)
		assert.Equal(t, 0, m.cursor, "cursor should not go below 0")

		// Move to last index
		lastIdx := m.sequence.Len() - 1
		m.cursor = lastIdx

		// At last index, KeyDown should stay at last
		m, _ = composeListSendKey(m, tea.KeyDown)
		assert.Equal(t, lastIdx, m.cursor, "cursor should not exceed last index")
	})

	t.Run("reorder shift+down", func(t *testing.T) {
		m := newTestComposeList(3)
		require.Equal(t, 0, m.cursor)

		// Before: pipeline-1, pipeline-2, pipeline-3
		assert.Equal(t, "pipeline-1", m.sequence.Entries[0].PipelineName)
		assert.Equal(t, "pipeline-2", m.sequence.Entries[1].PipelineName)
		assert.Equal(t, "pipeline-3", m.sequence.Entries[2].PipelineName)

		m, cmd := composeListSendKey(m, tea.KeyShiftDown)

		// After: pipeline-2, pipeline-1, pipeline-3
		assert.Equal(t, "pipeline-2", m.sequence.Entries[0].PipelineName)
		assert.Equal(t, "pipeline-1", m.sequence.Entries[1].PipelineName)
		assert.Equal(t, "pipeline-3", m.sequence.Entries[2].PipelineName)
		assert.Equal(t, 1, m.cursor, "cursor should move to 1 after shift+down")

		// Should emit ComposeSequenceChangedMsg
		changed := extractMsg[ComposeSequenceChangedMsg](cmd)
		assert.NotNil(t, changed, "shift+down should emit ComposeSequenceChangedMsg")
	})

	t.Run("reorder shift+up", func(t *testing.T) {
		m := newTestComposeList(3)
		m.cursor = 2

		// Before: pipeline-1, pipeline-2, pipeline-3
		assert.Equal(t, "pipeline-1", m.sequence.Entries[0].PipelineName)
		assert.Equal(t, "pipeline-2", m.sequence.Entries[1].PipelineName)
		assert.Equal(t, "pipeline-3", m.sequence.Entries[2].PipelineName)

		m, cmd := composeListSendKey(m, tea.KeyShiftUp)

		// After: pipeline-1, pipeline-3, pipeline-2
		assert.Equal(t, "pipeline-1", m.sequence.Entries[0].PipelineName)
		assert.Equal(t, "pipeline-3", m.sequence.Entries[1].PipelineName)
		assert.Equal(t, "pipeline-2", m.sequence.Entries[2].PipelineName)
		assert.Equal(t, 1, m.cursor, "cursor should move to 1 after shift+up")

		// Should emit ComposeSequenceChangedMsg
		changed := extractMsg[ComposeSequenceChangedMsg](cmd)
		assert.NotNil(t, changed, "shift+up should emit ComposeSequenceChangedMsg")
	})

	t.Run("remove entry", func(t *testing.T) {
		m := newTestComposeList(3)
		m.cursor = 1

		// Before: 3 entries
		require.Equal(t, 3, m.sequence.Len())
		removedName := m.sequence.Entries[1].PipelineName

		m, cmd := composeListSendRune(m, 'x')

		assert.Equal(t, 2, m.sequence.Len(), "sequence should have one fewer entry")
		// Verify the removed entry is gone
		for _, e := range m.sequence.Entries {
			assert.NotEqual(t, removedName, e.PipelineName)
		}

		// Should emit ComposeSequenceChangedMsg
		changed := extractMsg[ComposeSequenceChangedMsg](cmd)
		assert.NotNil(t, changed, "remove should emit ComposeSequenceChangedMsg")
	})

	t.Run("remove last entry adjusts cursor", func(t *testing.T) {
		m := newTestComposeList(2)
		m.cursor = 1

		// Remove the entry at cursor 1 (last position)
		m, _ = composeListSendRune(m, 'x')

		assert.Equal(t, 1, m.sequence.Len())
		assert.Equal(t, 0, m.cursor, "cursor should adjust to 0 when last entry is removed")
	})

	t.Run("Esc emits ComposeCancelMsg", func(t *testing.T) {
		m := newTestComposeList(1)

		_, cmd := composeListSendKey(m, tea.KeyEscape)

		cancel := extractMsg[ComposeCancelMsg](cmd)
		require.NotNil(t, cancel, "Esc should emit ComposeCancelMsg")
	})

	t.Run("Enter on non-empty sequence emits ComposeStartMsg", func(t *testing.T) {
		m := newTestComposeList(2)
		require.False(t, m.sequence.IsEmpty())

		_, cmd := composeListSendKey(m, tea.KeyEnter)

		start := extractMsg[ComposeStartMsg](cmd)
		require.NotNil(t, start, "Enter on non-empty sequence should emit ComposeStartMsg")
		assert.Equal(t, m.sequence.Len(), start.Sequence.Len(),
			"ComposeStartMsg should carry the current sequence")
	})

	t.Run("Enter on empty sequence is no-op", func(t *testing.T) {
		m := newTestComposeList(1)
		// Remove the only entry to get an empty sequence
		m.cursor = 0
		m, _ = composeListSendRune(m, 'x')
		require.True(t, m.sequence.IsEmpty())

		_, cmd := composeListSendKey(m, tea.KeyEnter)

		start := extractMsg[ComposeStartMsg](cmd)
		assert.Nil(t, start, "Enter on empty sequence should not emit ComposeStartMsg")
	})

	t.Run("duplicate pipeline allowed", func(t *testing.T) {
		m := newTestComposeList(1)
		require.Equal(t, 1, m.sequence.Len())

		// Add the same pipeline again directly via sequence
		p := testPipeline("pipeline-1",
			[]pipeline.ArtifactDef{{Name: "output1"}}, nil)
		m.sequence.Add("pipeline-1", p)

		assert.Equal(t, 2, m.sequence.Len(), "duplicate pipeline should be allowed")
		assert.Equal(t, "pipeline-1", m.sequence.Entries[0].PipelineName)
		assert.Equal(t, "pipeline-1", m.sequence.Entries[1].PipelineName)
	})
}
