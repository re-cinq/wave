package tui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// listTestPipelineProvider is a mock PipelineDataProvider scoped to
// pipeline_list_test.go to avoid collisions with mocks in other test files.
type listTestPipelineProvider struct {
	running   []RunningPipeline
	finished  []FinishedPipeline
	available []PipelineInfo
}

func (m *listTestPipelineProvider) FetchRunningPipelines() ([]RunningPipeline, error) {
	return m.running, nil
}

func (m *listTestPipelineProvider) FetchFinishedPipelines(limit int) ([]FinishedPipeline, error) {
	return m.finished, nil
}

func (m *listTestPipelineProvider) FetchAvailablePipelines() ([]PipelineInfo, error) {
	return m.available, nil
}

// newTestListModel creates a PipelineListModel pre-loaded with the given data.
// It bypasses async commands by directly injecting a PipelineDataMsg.
func newTestListModel(running []RunningPipeline, finished []FinishedPipeline, available []PipelineInfo) PipelineListModel {
	provider := &listTestPipelineProvider{running: running, finished: finished, available: available}
	m := NewPipelineListModel(provider)
	m.SetSize(40, 20)
	// Simulate data arrival
	m, _ = m.Update(PipelineDataMsg{Running: running, Finished: finished, Available: available})
	return m
}

var listAnsiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func listStripAnsi(s string) string {
	return listAnsiRegex.ReplaceAllString(s, "")
}

// sendKey is a convenience wrapper to send a key event and return the updated model.
func sendKey(m PipelineListModel, keyType tea.KeyType) (PipelineListModel, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: keyType})
}

// sendRune sends a rune key event (e.g. '/' or 's').
func sendRune(m PipelineListModel, r rune) (PipelineListModel, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
}

// extractSelectionMsg executes the tea.Cmd returned by Update and returns
// the PipelineSelectedMsg if one was emitted.
func extractSelectionMsg(cmd tea.Cmd) *PipelineSelectedMsg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	// Direct message (tea.Batch with 1 cmd returns it directly)
	if sel, ok := msg.(PipelineSelectedMsg); ok {
		return &sel
	}
	// tea.Batch with 2+ cmds returns tea.BatchMsg ([]tea.Cmd)
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			innerMsg := c()
			if sel, ok := innerMsg.(PipelineSelectedMsg); ok {
				return &sel
			}
		}
	}
	return nil
}

// extractRunningCountMsg executes the tea.Cmd and returns the RunningCountMsg
// if one was emitted, checking both direct and batched forms.
func extractRunningCountMsg(cmd tea.Cmd) *RunningCountMsg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	// Direct message
	if rcm, ok := msg.(RunningCountMsg); ok {
		return &rcm
	}
	// Batched
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			innerMsg := c()
			if rcm, ok := innerMsg.(RunningCountMsg); ok {
				return &rcm
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Sample data factories
// ---------------------------------------------------------------------------

func sampleRunning(n int) []RunningPipeline {
	out := make([]RunningPipeline, n)
	for i := range n {
		out[i] = RunningPipeline{
			RunID:      "run-" + string(rune('A'+i)),
			Name:       "running-" + string(rune('a'+i)),
			BranchName: "branch-" + string(rune('a'+i)),
			StartedAt:  time.Now().Add(-time.Duration(i+1) * time.Minute),
		}
	}
	return out
}

func sampleFinished(n int) []FinishedPipeline {
	statuses := []string{"completed", "failed", "cancelled"}
	out := make([]FinishedPipeline, n)
	for i := range n {
		out[i] = FinishedPipeline{
			RunID:       "frun-" + string(rune('A'+i)),
			Name:        "finished-" + string(rune('a'+i)),
			BranchName:  "fbranch-" + string(rune('a'+i)),
			Status:      statuses[i%len(statuses)],
			StartedAt:   time.Now().Add(-time.Duration(i+2) * time.Minute),
			CompletedAt: time.Now().Add(-time.Duration(i+1) * time.Minute),
			Duration:    time.Duration(i+1) * time.Minute,
		}
	}
	return out
}

func sampleAvailable(n int) []PipelineInfo {
	out := make([]PipelineInfo, n)
	for i := range n {
		out[i] = PipelineInfo{
			Name:        "avail-" + string(rune('a'+i)),
			Description: "desc " + string(rune('a'+i)),
			StepCount:   i + 1,
		}
	}
	return out
}

// ===========================================================================
// T012: Section Rendering Tests
// ===========================================================================

func TestPipelineListModel_View_AllSectionsRender(t *testing.T) {
	m := newTestListModel(sampleRunning(2), sampleFinished(3), sampleAvailable(2))
	view := listStripAnsi(m.View())

	assert.Contains(t, view, "Running (2)")
	assert.Contains(t, view, "Finished (3)")
	assert.Contains(t, view, "Available (2)")
}

func TestPipelineListModel_View_RunningItemsShowElapsedTime(t *testing.T) {
	running := []RunningPipeline{{
		RunID:     "r1",
		Name:      "long-running",
		StartedAt: time.Now().Add(-150 * time.Second),
	}}
	m := newTestListModel(running, nil, nil)
	view := listStripAnsi(m.View())

	// 150 seconds = 02:30 (formatElapsed produces MM:SS)
	assert.Contains(t, view, "02:30")
}

func TestPipelineListModel_View_FinishedItemsShowStatusAndDuration(t *testing.T) {
	finished := []FinishedPipeline{
		{RunID: "f1", Name: "success-pipe", Status: "completed", Duration: 5 * time.Minute},
		{RunID: "f2", Name: "fail-pipe", Status: "failed", Duration: 3 * time.Minute},
		{RunID: "f3", Name: "cancel-pipe", Status: "cancelled", Duration: 1 * time.Minute},
	}
	m := newTestListModel(nil, finished, nil)
	m.collapsed[2] = false // Expand Finished (collapsed by default)
	m.buildNavigableItems()
	view := listStripAnsi(m.View())

	// Completed shows checkmark
	assert.Contains(t, view, "✓")
	assert.Contains(t, view, "completed")
	assert.Contains(t, view, "5m 0s")

	// Failed shows cross
	assert.Contains(t, view, "✗")
	assert.Contains(t, view, "failed")
	assert.Contains(t, view, "3m 0s")

	// Cancelled shows cross
	assert.Contains(t, view, "cancelled")
	assert.Contains(t, view, "1m 0s")
}

func TestPipelineListModel_View_AvailableItemsShowNameOnly(t *testing.T) {
	avail := []PipelineInfo{{Name: "speckit-flow", Description: "A pipeline"}}
	m := newTestListModel(nil, nil, avail)
	view := listStripAnsi(m.View())

	assert.Contains(t, view, "speckit-flow")
	// Should not contain status markers or durations
	assert.NotContains(t, view, "✓")
	assert.NotContains(t, view, "✗")
}

func TestPipelineListModel_View_EmptySectionsShowZeroCount(t *testing.T) {
	m := newTestListModel(nil, nil, sampleAvailable(1))
	view := listStripAnsi(m.View())

	assert.Contains(t, view, "Running (0)")
	assert.Contains(t, view, "Finished (0)")
	assert.Contains(t, view, "Available (1)")
}

func TestPipelineListModel_View_AllSectionsEmpty(t *testing.T) {
	// When all sections have zero items, headers with "(0)" are still shown
	// because the empty-state message ("No pipelines found") only appears
	// when there are zero navigable items (i.e., no headers either).
	// With no filter active and empty data, section headers are still built.
	// Verify that the headers render correctly with zero counts.
	m := newTestListModel(nil, nil, nil)
	view := listStripAnsi(m.View())

	assert.Contains(t, view, "Running (0)")
	assert.Contains(t, view, "Finished (0)")
	assert.Contains(t, view, "Available (0)")
}

func TestPipelineListModel_View_NoMatchingPipelines_ShowsEmptyMessage(t *testing.T) {
	// The "No pipelines found" / "No matching pipelines" message appears
	// when a filter removes ALL items (including headers).
	avail := []PipelineInfo{{Name: "speckit-flow"}}
	m := newTestListModel(nil, nil, avail)

	// Activate filter with a query that matches nothing
	m, _ = sendRune(m, '/')
	for _, ch := range "zzzzzzz" {
		m, _ = sendRune(m, ch)
	}

	view := listStripAnsi(m.View())
	assert.Contains(t, view, "No matching pipelines")
}

func TestPipelineListModel_View_LongNamesTruncated(t *testing.T) {
	longName := strings.Repeat("x", 50)
	avail := []PipelineInfo{{Name: longName}}
	m := newTestListModel(nil, nil, avail)
	m.SetSize(40, 20)
	// Re-inject data so View uses updated size
	m, _ = m.Update(PipelineDataMsg{Available: avail})
	view := listStripAnsi(m.View())

	// Should contain the truncation marker
	assert.Contains(t, view, "…")
	// Should NOT contain the full name
	assert.NotContains(t, view, longName)
}

// ===========================================================================
// T015: Navigation Tests
// ===========================================================================

func TestPipelineListModel_Navigation_DownMovesCursor(t *testing.T) {
	m := newTestListModel(sampleRunning(2), nil, nil)
	require.Equal(t, 0, m.cursor)

	m, _ = sendKey(m, tea.KeyDown)
	assert.Equal(t, 1, m.cursor)
}

func TestPipelineListModel_Navigation_UpAtTopStays(t *testing.T) {
	m := newTestListModel(sampleRunning(2), nil, nil)
	require.Equal(t, 0, m.cursor)

	m, _ = sendKey(m, tea.KeyUp)
	assert.Equal(t, 0, m.cursor)
}

func TestPipelineListModel_Navigation_DownAtBottomStays(t *testing.T) {
	m := newTestListModel(sampleRunning(1), nil, nil)
	lastIdx := len(m.navigable) - 1

	// Move cursor to last item
	for range m.navigable {
		m, _ = sendKey(m, tea.KeyDown)
	}
	assert.Equal(t, lastIdx, m.cursor)

	// Try moving past the end
	m, _ = sendKey(m, tea.KeyDown)
	assert.Equal(t, lastIdx, m.cursor)
}

func TestPipelineListModel_Navigation_CrossSectionTraversal(t *testing.T) {
	m := newTestListModel(sampleRunning(1), sampleFinished(1), nil)
	m.collapsed[2] = false // Expand Finished (collapsed by default)
	m.buildNavigableItems()

	// navigable should include (order: Running, Available, Finished):
	// 0: Running (1) header
	// 1: running-a
	// 2: Available (0) header  (empty sections still get headers)
	// 3: Finished (1) header
	// 4: finished-a
	require.GreaterOrEqual(t, len(m.navigable), 5)

	// Navigate from top to finished item
	m, _ = sendKey(m, tea.KeyDown) // cursor=1 running item
	m, _ = sendKey(m, tea.KeyDown) // cursor=2 available header (empty)
	m, _ = sendKey(m, tea.KeyDown) // cursor=3 finished header
	m, _ = sendKey(m, tea.KeyDown) // cursor=4 finished item
	assert.Equal(t, 4, m.cursor)
	assert.Equal(t, itemKindFinished, m.navigable[m.cursor].kind)
}

func TestPipelineListModel_Navigation_SelectionMsgOnPipelineItem(t *testing.T) {
	m := newTestListModel(sampleRunning(1), nil, nil)

	// Move to the running item (index 1)
	m, cmd := sendKey(m, tea.KeyDown)
	assert.Equal(t, 1, m.cursor)
	assert.Equal(t, itemKindRunning, m.navigable[m.cursor].kind)

	sel := extractSelectionMsg(cmd)
	require.NotNil(t, sel, "should emit PipelineSelectedMsg on pipeline item")
}

func TestPipelineListModel_Navigation_NoSelectionMsgOnHeader(t *testing.T) {
	m := newTestListModel(sampleRunning(1), nil, nil)

	// Cursor starts at 0, which is the section header
	assert.Equal(t, 0, m.cursor)
	assert.Equal(t, itemKindSectionHeader, m.navigable[m.cursor].kind)

	// Move up (stays at 0, header)
	_, cmd := sendKey(m, tea.KeyUp)
	sel := extractSelectionMsg(cmd)
	assert.Nil(t, sel, "should NOT emit PipelineSelectedMsg on section header")
}

func TestPipelineListModel_Navigation_RunningItemIncludesRunID(t *testing.T) {
	running := []RunningPipeline{{
		RunID:      "run-xyz",
		Name:       "my-pipeline",
		BranchName: "feat/branch",
		StartedAt:  time.Now(),
	}}
	m := newTestListModel(running, nil, nil)

	// Move to running item
	m, cmd := sendKey(m, tea.KeyDown)
	sel := extractSelectionMsg(cmd)
	require.NotNil(t, sel)

	assert.Equal(t, "run-xyz", sel.RunID)
	assert.Equal(t, "feat/branch", sel.BranchName)
}

func TestPipelineListModel_Navigation_AvailableItemHasEmptyRunID(t *testing.T) {
	avail := []PipelineInfo{{Name: "speckit-flow"}}
	m := newTestListModel(nil, nil, avail)

	// Move cursor to the available item
	for i := range m.navigable {
		if m.navigable[i].kind == itemKindAvailable {
			m.cursor = i
			break
		}
	}

	// Emit selection for current cursor position
	cmd := m.emitSelectionMsg()
	sel := extractSelectionMsg(cmd)
	require.NotNil(t, sel)

	assert.Equal(t, "", sel.RunID)
	assert.Equal(t, "", sel.BranchName)
}

// ===========================================================================
// T018: Filter Tests
// ===========================================================================

func TestPipelineListModel_Filter_SlashActivates(t *testing.T) {
	m := newTestListModel(sampleRunning(1), nil, nil)
	require.False(t, m.filtering)

	m, _ = sendRune(m, '/')
	assert.True(t, m.filtering)
}

func TestPipelineListModel_Filter_MatchesSubstring(t *testing.T) {
	avail := []PipelineInfo{
		{Name: "speckit-flow"},
		{Name: "wave-evolve"},
		{Name: "speckit-debug"},
	}
	m := newTestListModel(nil, nil, avail)

	// Activate filter
	m, _ = sendRune(m, '/')

	// Type "spec" character by character
	for _, ch := range "spec" {
		m, _ = sendRune(m, ch)
	}

	view := listStripAnsi(m.View())
	assert.Contains(t, view, "speckit-flow")
	assert.Contains(t, view, "speckit-debug")
	assert.NotContains(t, view, "wave-evolve")
}

func TestPipelineListModel_Filter_AcrossAllSections(t *testing.T) {
	running := []RunningPipeline{
		{RunID: "r1", Name: "speckit-run", StartedAt: time.Now()},
		{RunID: "r2", Name: "wave-run", StartedAt: time.Now()},
	}
	finished := []FinishedPipeline{
		{RunID: "f1", Name: "speckit-done", Status: "completed", Duration: time.Minute},
	}
	avail := []PipelineInfo{
		{Name: "wave-evolve"},
	}
	m := newTestListModel(running, finished, avail)

	m, _ = sendRune(m, '/')
	for _, ch := range "speckit" {
		m, _ = sendRune(m, ch)
	}

	view := listStripAnsi(m.View())
	assert.Contains(t, view, "speckit-run")
	assert.Contains(t, view, "speckit-done")
	assert.NotContains(t, view, "wave-run")
	assert.NotContains(t, view, "wave-evolve")
}

func TestPipelineListModel_Filter_EscapeDismisses(t *testing.T) {
	avail := []PipelineInfo{
		{Name: "speckit-flow"},
		{Name: "wave-evolve"},
	}
	m := newTestListModel(nil, nil, avail)

	// Activate filter and type something
	m, _ = sendRune(m, '/')
	for _, ch := range "spec" {
		m, _ = sendRune(m, ch)
	}
	require.True(t, m.filtering)

	// Press escape to dismiss
	m, _ = sendKey(m, tea.KeyEscape)
	assert.False(t, m.filtering)

	// All items should be visible again
	view := listStripAnsi(m.View())
	assert.Contains(t, view, "speckit-flow")
	assert.Contains(t, view, "wave-evolve")
}

func TestPipelineListModel_Filter_ZeroMatchesMessage(t *testing.T) {
	avail := []PipelineInfo{{Name: "speckit-flow"}}
	m := newTestListModel(nil, nil, avail)

	m, _ = sendRune(m, '/')
	for _, ch := range "zzzzzzz" {
		m, _ = sendRune(m, ch)
	}

	view := listStripAnsi(m.View())
	assert.Contains(t, view, "No matching pipelines")
}

func TestPipelineListModel_Filter_NavigationInFilteredResults(t *testing.T) {
	avail := []PipelineInfo{
		{Name: "alpha-pipe"},
		{Name: "alpha-debug"},
		{Name: "beta-pipe"},
	}
	m := newTestListModel(nil, nil, avail)

	// Filter to "alpha" items
	m, _ = sendRune(m, '/')
	for _, ch := range "alpha" {
		m, _ = sendRune(m, ch)
	}

	// Navigate within filtered results — should not panic or go out of bounds
	startCursor := m.cursor
	m, _ = sendKey(m, tea.KeyDown)
	assert.GreaterOrEqual(t, m.cursor, startCursor)

	m, _ = sendKey(m, tea.KeyDown)
	m, _ = sendKey(m, tea.KeyDown)
	m, _ = sendKey(m, tea.KeyDown)
	// Should be clamped to last navigable item
	assert.Less(t, m.cursor, len(m.navigable))

	m, _ = sendKey(m, tea.KeyUp)
	assert.GreaterOrEqual(t, m.cursor, 0)
}

func TestPipelineListModel_Filter_CursorClampedAfterNarrow(t *testing.T) {
	// Start with many available pipelines, navigate cursor to a high index,
	// then filter so that fewer items remain — cursor must be clamped.
	avail := []PipelineInfo{
		{Name: "alpha-pipeline"},
		{Name: "beta-pipeline"},
		{Name: "gamma-pipeline"},
		{Name: "delta-pipeline"},
		{Name: "epsilon-pipeline"},
	}
	m := newTestListModel(nil, nil, avail)

	// Navigate cursor deep into the list (past all available items)
	for range 6 {
		m, _ = sendKey(m, tea.KeyDown)
	}
	require.GreaterOrEqual(t, m.cursor, 4, "cursor should be deep in the list")

	// Activate filter and type something that matches only one item
	m, _ = sendRune(m, '/')
	for _, ch := range "epsilon" {
		m, _ = sendRune(m, ch)
	}

	// Cursor must be clamped to valid range (not out of bounds)
	assert.Less(t, m.cursor, len(m.navigable),
		"cursor must be clamped to navigable bounds after filter narrows results")
	assert.GreaterOrEqual(t, m.cursor, 0, "cursor must not be negative")
}

func TestPipelineListModel_Filter_EnterWithZeroResults_StaysInFilterMode(t *testing.T) {
	avail := []PipelineInfo{{Name: "speckit-flow"}}
	m := newTestListModel(nil, nil, avail)

	// Activate filter and type a query that matches nothing
	m, _ = sendRune(m, '/')
	for _, ch := range "zzzzzzz" {
		m, _ = sendRune(m, ch)
	}
	require.True(t, m.filtering)
	require.Equal(t, 0, len(m.navigable))

	// Press Enter — should NOT deactivate filter mode
	m, _ = sendKey(m, tea.KeyEnter)
	assert.True(t, m.filtering, "filter should remain active when no results match")

	// User can still press Escape to dismiss filter and restore list
	m, _ = sendKey(m, tea.KeyEscape)
	assert.False(t, m.filtering)
	assert.Greater(t, len(m.navigable), 0, "all items should be restored after Escape")
}

func TestPipelineListModel_Filter_SlashRestoresListAfterConfirmedFilter(t *testing.T) {
	avail := []PipelineInfo{
		{Name: "speckit-flow"},
		{Name: "wave-evolve"},
	}
	m := newTestListModel(nil, nil, avail)

	// Activate filter, type "spec", then confirm with Enter
	m, _ = sendRune(m, '/')
	for _, ch := range "spec" {
		m, _ = sendRune(m, ch)
	}
	m, _ = sendKey(m, tea.KeyEnter)
	require.False(t, m.filtering)

	// Only speckit-flow should be visible
	view := listStripAnsi(m.View())
	assert.Contains(t, view, "speckit-flow")
	assert.NotContains(t, view, "wave-evolve")

	// Press '/' to start new filter — should restore full list immediately
	m, _ = sendRune(m, '/')
	assert.True(t, m.filtering)
	assert.Equal(t, "", m.filterQuery)

	view = listStripAnsi(m.View())
	assert.Contains(t, view, "speckit-flow")
	assert.Contains(t, view, "wave-evolve")
}

// ===========================================================================
// T020: Scrolling Tests
// ===========================================================================

func TestPipelineListModel_Scroll_CursorBelowViewportScrollsDown(t *testing.T) {
	// Create many items so they exceed the viewport
	running := sampleRunning(3)
	finished := sampleFinished(5)
	avail := sampleAvailable(5)
	m := newTestListModel(running, finished, avail)
	m.SetSize(40, 5)
	m, _ = m.Update(PipelineDataMsg{Running: running, Finished: finished, Available: avail})

	// Navigate down past the viewport (7 presses)
	for range 7 {
		m, _ = sendKey(m, tea.KeyDown)
	}

	// Verify cursor is beyond viewport height
	assert.Greater(t, m.cursor, 4)

	// The View method adjusts scroll internally. Verify that the first
	// navigable item (Running header at index 0) is NOT in the rendered
	// output because the viewport has scrolled past it.
	view := listStripAnsi(m.View())
	lines := strings.Split(view, "\n")
	// The first visible line should NOT be the Running header
	require.GreaterOrEqual(t, len(lines), 1)
	assert.NotContains(t, lines[0], "Running (3)",
		"viewport should have scrolled past the Running header")
}

func TestPipelineListModel_Scroll_ScrollBackUp(t *testing.T) {
	running := sampleRunning(3)
	finished := sampleFinished(5)
	avail := sampleAvailable(5)
	m := newTestListModel(running, finished, avail)
	m.SetSize(40, 5)
	m, _ = m.Update(PipelineDataMsg{Running: running, Finished: finished, Available: avail})

	// Navigate down past the viewport
	for range 8 {
		m, _ = sendKey(m, tea.KeyDown)
	}

	// Verify we scrolled down by checking the view content
	viewDown := listStripAnsi(m.View())
	assert.NotContains(t, strings.Split(viewDown, "\n")[0], "Running (3)")

	// Navigate back up to the top
	for range 8 {
		m, _ = sendKey(m, tea.KeyUp)
	}

	// The first line should again be the Running header
	viewUp := listStripAnsi(m.View())
	lines := strings.Split(viewUp, "\n")
	require.GreaterOrEqual(t, len(lines), 1)
	assert.Contains(t, lines[0], "Running (3)",
		"scrolling back up should show the Running header again")
}

func TestPipelineListModel_Scroll_CursorAtTopNoScroll(t *testing.T) {
	m := newTestListModel(sampleRunning(2), nil, nil)
	m.SetSize(40, 20) // viewport bigger than item count
	assert.Equal(t, 0, m.cursor)

	view := listStripAnsi(m.View())
	lines := strings.Split(view, "\n")
	require.GreaterOrEqual(t, len(lines), 1)
	assert.Contains(t, lines[0], "Running",
		"no scroll needed when cursor at top and viewport fits all items")
}

// ===========================================================================
// T022: Collapse/Expand Tests
// ===========================================================================

func TestPipelineListModel_Collapse_EnterOnHeaderCollapses(t *testing.T) {
	m := newTestListModel(sampleRunning(2), nil, nil)

	// Cursor starts on Running header
	require.Equal(t, 0, m.cursor)
	require.Equal(t, itemKindSectionHeader, m.navigable[0].kind)

	// Count items before collapse
	countBefore := len(m.navigable)

	// Press Enter to collapse
	m, _ = sendKey(m, tea.KeyEnter)

	// Items should be hidden — fewer navigable items
	assert.Less(t, len(m.navigable), countBefore, "collapsing should reduce navigable items")
	assert.True(t, m.collapsed[0], "Running section should be collapsed")

	// Verify the running items are gone from the view
	view := listStripAnsi(m.View())
	assert.NotContains(t, view, "running-a")
	assert.NotContains(t, view, "running-b")
}

func TestPipelineListModel_Collapse_EnterAgainExpands(t *testing.T) {
	m := newTestListModel(sampleRunning(2), nil, nil)

	// Collapse
	m, _ = sendKey(m, tea.KeyEnter)
	require.True(t, m.collapsed[0])
	collapsedCount := len(m.navigable)

	// Expand
	m, _ = sendKey(m, tea.KeyEnter)
	assert.False(t, m.collapsed[0])
	assert.Greater(t, len(m.navigable), collapsedCount, "expanding should restore items")

	// Items should reappear
	view := listStripAnsi(m.View())
	assert.Contains(t, view, "running-a")
}

func TestPipelineListModel_Collapse_CursorSkipsHiddenItems(t *testing.T) {
	m := newTestListModel(sampleRunning(2), sampleFinished(1), nil)

	// Navigable before collapse includes headers for empty Available section too.
	// Find the index of the Finished header for reference.
	var finishedHeaderIdx int
	for i, item := range m.navigable {
		if item.kind == itemKindSectionHeader && item.sectionIndex == 1 {
			finishedHeaderIdx = i
			break
		}
	}
	require.Equal(t, 3, finishedHeaderIdx, "Finished header should be at index 3 before collapse")

	// Collapse Running section (cursor is on header at 0)
	m, _ = sendKey(m, tea.KeyEnter)

	// After collapse, running items are hidden but all headers remain.
	// Navigate down from collapsed Running header
	m, _ = sendKey(m, tea.KeyDown)
	assert.Equal(t, 1, m.cursor)
	// The item at cursor 1 should be the Finished section header
	assert.Equal(t, itemKindSectionHeader, m.navigable[m.cursor].kind)
	assert.Equal(t, 1, m.navigable[m.cursor].sectionIndex, "should jump to Finished section header")
}

func TestPipelineListModel_Collapse_IndicatorRendering(t *testing.T) {
	m := newTestListModel(sampleRunning(1), nil, nil)

	// Expanded: should show ▾
	view := listStripAnsi(m.View())
	assert.Contains(t, view, "▼")

	// Collapse
	m, _ = sendKey(m, tea.KeyEnter)
	view = listStripAnsi(m.View())
	assert.Contains(t, view, "▶")
}

// ===========================================================================
// Additional Tests
// ===========================================================================

func TestPipelineListModel_Init_ReturnsBatchCmd(t *testing.T) {
	provider := &listTestPipelineProvider{}
	m := NewPipelineListModel(provider)
	cmd := m.Init()
	assert.NotNil(t, cmd, "Init should return a non-nil batch command")
}

func TestPipelineListModel_Update_DataMsgUpdatesState(t *testing.T) {
	provider := &listTestPipelineProvider{}
	m := NewPipelineListModel(provider)
	m.SetSize(40, 20)

	running := sampleRunning(2)
	finished := sampleFinished(1)
	avail := sampleAvailable(3)

	m, _ = m.Update(PipelineDataMsg{Running: running, Finished: finished, Available: avail})

	assert.Equal(t, running, m.running)
	assert.Equal(t, finished, m.finished)
	assert.Equal(t, avail, m.available)
	assert.Greater(t, len(m.navigable), 0, "navigable items should be built")
}

func TestPipelineListModel_Update_DataMsgEmitsRunningCount(t *testing.T) {
	provider := &listTestPipelineProvider{}
	m := NewPipelineListModel(provider)
	m.SetSize(40, 20)

	running := sampleRunning(3)
	_, cmd := m.Update(PipelineDataMsg{Running: running})

	rcm := extractRunningCountMsg(cmd)
	require.NotNil(t, rcm, "should emit RunningCountMsg")
	assert.Equal(t, 3, rcm.Count)
}

func TestPipelineListModel_SetSize(t *testing.T) {
	provider := &listTestPipelineProvider{}
	m := NewPipelineListModel(provider)

	m.SetSize(80, 40)
	assert.Equal(t, 80, m.width)
	assert.Equal(t, 40, m.height)
}

func TestPipelineListModel_View_ZeroDimensions(t *testing.T) {
	provider := &listTestPipelineProvider{}
	m := NewPipelineListModel(provider)
	// width and height default to 0
	view := m.View()
	assert.Equal(t, "", view)
}

// ===========================================================================
// T014: List integration tests for pipeline launch flow
// ===========================================================================

func TestPipelineListModel_PipelineLaunchedMsg_PrependsRunningEntry(t *testing.T) {
	m := newTestListModel(nil, nil, sampleAvailable(2))

	require.Equal(t, 0, len(m.running))

	// Send PipelineLaunchedMsg
	launchedMsg := PipelineLaunchedMsg{RunID: "run-new", PipelineName: "launched-pipe"}
	m, _ = m.Update(launchedMsg)

	require.Equal(t, 1, len(m.running))
	assert.Equal(t, "run-new", m.running[0].RunID)
	assert.Equal(t, "launched-pipe", m.running[0].Name)
}

func TestPipelineListModel_PipelineLaunchedMsg_RebuildNavigableItems(t *testing.T) {
	m := newTestListModel(nil, nil, sampleAvailable(1))
	navBefore := len(m.navigable)

	launchedMsg := PipelineLaunchedMsg{RunID: "run-new", PipelineName: "launched-pipe"}
	m, _ = m.Update(launchedMsg)

	// Should have more navigable items now (a running item was added)
	assert.Greater(t, len(m.navigable), navBefore)

	// Verify the new running item is in navigable
	foundRunning := false
	for _, item := range m.navigable {
		if item.kind == itemKindRunning && item.label == "launched-pipe" {
			foundRunning = true
			break
		}
	}
	assert.True(t, foundRunning, "navigable should include the new running item")
}

func TestPipelineListModel_PipelineLaunchedMsg_MovesCursorToRunningEntry(t *testing.T) {
	m := newTestListModel(nil, nil, sampleAvailable(2))

	launchedMsg := PipelineLaunchedMsg{RunID: "run-new", PipelineName: "launched-pipe"}
	m, _ = m.Update(launchedMsg)

	// Cursor should be on the first running item
	require.Less(t, m.cursor, len(m.navigable))
	assert.Equal(t, itemKindRunning, m.navigable[m.cursor].kind)
}

func TestPipelineListModel_PipelineLaunchedMsg_EmitsRunningCount(t *testing.T) {
	m := newTestListModel(nil, nil, sampleAvailable(1))

	launchedMsg := PipelineLaunchedMsg{RunID: "run-new", PipelineName: "launched-pipe"}
	_, cmd := m.Update(launchedMsg)

	rcm := extractRunningCountMsg(cmd)
	require.NotNil(t, rcm, "should emit RunningCountMsg")
	assert.Equal(t, 1, rcm.Count)
}

func TestPipelineListModel_PipelineLaunchedMsg_PreservesExistingRunning(t *testing.T) {
	existing := sampleRunning(2)
	m := newTestListModel(existing, nil, sampleAvailable(1))
	require.Equal(t, 2, len(m.running))

	launchedMsg := PipelineLaunchedMsg{RunID: "run-new", PipelineName: "launched-pipe"}
	m, cmd := m.Update(launchedMsg)

	// Should have 3 running entries (2 existing + 1 new)
	assert.Equal(t, 3, len(m.running))
	// New entry is at index 0 (prepended)
	assert.Equal(t, "run-new", m.running[0].RunID)
	// Existing entries are still there
	assert.Equal(t, existing[0].RunID, m.running[1].RunID)
	assert.Equal(t, existing[1].RunID, m.running[2].RunID)

	// Running count should reflect all 3
	rcm := extractRunningCountMsg(cmd)
	require.NotNil(t, rcm)
	assert.Equal(t, 3, rcm.Count)
}
