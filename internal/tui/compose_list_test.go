package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/pipelinecatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var composeAnsiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func composeStripAnsi(s string) string {
	return composeAnsiRegex.ReplaceAllString(s, "")
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newTestComposeList creates a ComposeListModel with the given number of
// entries. The first entry is always "pipeline-1" (added by the constructor).
// Additional entries are added via sequence.Add.
func newTestComposeList(entries int) ComposeListModel {
	initial := pipelinecatalog.PipelineInfo{Name: "pipeline-1", Description: "First", StepCount: 2}
	initialPipeline := testPipeline("pipeline-1",
		[]pipeline.ArtifactDef{{Name: "output1"}}, nil)

	available := []pipelinecatalog.PipelineInfo{
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

// drainCmds simulates Bubble Tea's event loop by executing commands and feeding
// resulting messages back through the model's Update, up to 20 iterations.
func drainCmds(m *ComposeListModel, cmd tea.Cmd) {
	for i := 0; i < 20 && cmd != nil; i++ {
		msg := cmd()
		if msg == nil {
			return
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				if c != nil {
					drainCmds(m, c)
				}
			}
			return
		}
		*m, cmd = m.Update(msg)
	}
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

	t.Run("p toggles parallel mode", func(t *testing.T) {
		m := newTestComposeList(2)
		assert.False(t, m.parallel)

		m, _ = composeListSendRune(m, 'p')
		assert.True(t, m.parallel, "p should toggle parallel on")

		m, _ = composeListSendRune(m, 'p')
		assert.False(t, m.parallel, "p should toggle parallel off")
	})

	t.Run("d toggles stage break", func(t *testing.T) {
		m := newTestComposeList(3)
		m.cursor = 0
		assert.Empty(t, m.breaks)

		m, _ = composeListSendRune(m, 'd')
		assert.True(t, m.breaks[0], "d should add break after cursor 0")

		m, _ = composeListSendRune(m, 'd')
		assert.False(t, m.breaks[0], "d again should remove break")
	})

	t.Run("d on last entry is no-op", func(t *testing.T) {
		m := newTestComposeList(3)
		m.cursor = 2 // last entry

		m, _ = composeListSendRune(m, 'd')
		assert.Empty(t, m.breaks, "d on last entry should not add break")
	})

	t.Run("buildStages with no breaks returns single stage", func(t *testing.T) {
		m := newTestComposeList(3)
		stages := m.buildStages()
		assert.Equal(t, 1, len(stages))
		assert.Equal(t, []int{0, 1, 2}, stages[0])
	})

	t.Run("buildStages with break splits into two stages", func(t *testing.T) {
		m := newTestComposeList(3)
		m.breaks = map[int]bool{0: true}
		stages := m.buildStages()
		assert.Equal(t, 2, len(stages))
		assert.Equal(t, []int{0}, stages[0])
		assert.Equal(t, []int{1, 2}, stages[1])
	})

	t.Run("Enter emits ComposeStartMsg with parallel flag", func(t *testing.T) {
		m := newTestComposeList(2)
		m.parallel = true
		m.breaks = map[int]bool{0: true}

		_, cmd := composeListSendKey(m, tea.KeyEnter)
		start := extractMsg[ComposeStartMsg](cmd)
		require.NotNil(t, start)
		assert.True(t, start.Parallel, "ComposeStartMsg should have Parallel=true")
		assert.Equal(t, 2, len(start.Stages), "should have 2 stages")
	})

	t.Run("scroll: first items disappear when cursor moves past viewport", func(t *testing.T) {
		// Create a list with 10 entries and height=8 (visible=5 entries: height 8 - overhead 3)
		m := newTestComposeList(10)
		m.SetSize(40, 8)

		// Move cursor down past the visible window (5 entries)
		for i := 0; i < 6; i++ {
			m, _ = composeListSendKey(m, tea.KeyDown)
		}
		assert.Equal(t, 6, m.cursor)

		view := composeStripAnsi(m.View())
		// "pipeline-1" (entry 0) should have scrolled out of view
		assert.NotContains(t, view, "1. pipeline-1",
			"first entry should scroll out when cursor is past viewport")
		// Current entry should be visible
		assert.Contains(t, view, "7. pipeline-7",
			"cursor entry should be visible")
	})

	t.Run("scroll: scrolling back up restores first items", func(t *testing.T) {
		m := newTestComposeList(10)
		m.SetSize(40, 8) // visible = 5 entries

		// Scroll down
		for i := 0; i < 6; i++ {
			m, _ = composeListSendKey(m, tea.KeyDown)
		}
		// Scroll back up
		for i := 0; i < 6; i++ {
			m, _ = composeListSendKey(m, tea.KeyUp)
		}
		assert.Equal(t, 0, m.cursor)

		view := composeStripAnsi(m.View())
		assert.Contains(t, view, "1. pipeline-1",
			"first entry should be visible after scrolling back up")
	})

	t.Run("scroll: all entries visible when height is sufficient", func(t *testing.T) {
		m := newTestComposeList(3)
		m.SetSize(40, 20) // plenty of room

		view := composeStripAnsi(m.View())
		assert.Contains(t, view, "1. pipeline-1")
		assert.Contains(t, view, "2. pipeline-2")
		assert.Contains(t, view, "3. pipeline-3")
	})

	t.Run("scroll: empty list renders correctly", func(t *testing.T) {
		m := newTestComposeList(1)
		m.cursor = 0
		m, _ = composeListSendRune(m, 'x') // remove only entry
		require.True(t, m.sequence.IsEmpty())

		view := composeStripAnsi(m.View())
		assert.Contains(t, view, "No pipelines in sequence")
	})

	t.Run("scroll: single-item list does not scroll", func(t *testing.T) {
		m := newTestComposeList(1)
		m.SetSize(40, 8)

		view := composeStripAnsi(m.View())
		assert.Contains(t, view, "1. pipeline-1")
		// Status should indicate single pipeline
		assert.Contains(t, view, "single pipeline")
	})

	t.Run("scroll: height=0 returns empty string", func(t *testing.T) {
		m := newTestComposeList(3)
		m.SetSize(40, 0)
		view := m.View()
		assert.Equal(t, "", view)
	})

	t.Run("scroll: visible entries count matches available space", func(t *testing.T) {
		m := newTestComposeList(10)
		m.SetSize(40, 6) // visible = 6 - 3 overhead = 3 entries

		view := composeStripAnsi(m.View())
		// Count how many "pipeline-" entries appear
		count := strings.Count(view, ". pipeline-")
		assert.Equal(t, 3, count,
			"should show exactly 3 entries for height=6 (overhead=3)")
	})

	t.Run("a enters picking mode", func(t *testing.T) {
		m := newTestComposeList(1)
		require.False(t, m.picking)

		m, cmd := composeListSendRune(m, 'a')
		assert.True(t, m.picking, "should enter picking mode")
		assert.NotNil(t, m.picker, "picker form should exist")
		assert.NotNil(t, m.pickerTarget, "picker target should be allocated")
		assert.NotNil(t, cmd, "should return init command")
	})

	t.Run("picker processes init then Enter completes selection", func(t *testing.T) {
		m := newTestComposeList(1)
		require.Equal(t, 1, m.sequence.Len())

		// Enter picking mode
		m, _ = composeListSendRune(m, 'a')
		require.True(t, m.picking)
		require.NotNil(t, m.picker)

		// Send Enter — first round: Select returns NextField cmd
		m, cmd := composeListSendKey(m, tea.KeyEnter)

		// The form isn't complete yet — it needs the NextField message
		// to be fed back (Bubble Tea does this in its event loop).
		// Drain all returned commands back through the model.
		drainCmds(&m, cmd)

		assert.False(t, m.picking, "picking should be done after Enter")
		assert.Nil(t, m.picker, "picker should be cleared")
		assert.Equal(t, 2, m.sequence.Len(), "selected pipeline should be added")
	})

	t.Run("picker Escape aborts without adding", func(t *testing.T) {
		m := newTestComposeList(1)
		require.Equal(t, 1, m.sequence.Len())

		m, _ = composeListSendRune(m, 'a')
		require.True(t, m.picking)

		m, cmd := composeListSendKey(m, tea.KeyEscape)
		drainCmds(&m, cmd)

		assert.False(t, m.picking, "picking should end on Escape")
		assert.Nil(t, m.picker, "picker should be cleared")
		assert.Equal(t, 1, m.sequence.Len(), "sequence should be unchanged")
	})
}
