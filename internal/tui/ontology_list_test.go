package tui

import (
	"errors"
	"regexp"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock OntologyDataProvider for list/detail tests.
// ---------------------------------------------------------------------------

type mockOntologyProvider struct {
	overview *OntologyOverview
	err      error
}

func (m *mockOntologyProvider) FetchOntology() (*OntologyOverview, error) {
	return m.overview, m.err
}

// ---------------------------------------------------------------------------
// ANSI stripping helper (prefixed to avoid collision with listStripAnsi).
// ---------------------------------------------------------------------------

var ontologyAnsiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func ontologyStripAnsi(s string) string {
	return ontologyAnsiRegex.ReplaceAllString(s, "")
}

// ---------------------------------------------------------------------------
// Helpers for sending keys to OntologyListModel.
// ---------------------------------------------------------------------------

func ontologySendKey(m OntologyListModel, keyType tea.KeyType) (OntologyListModel, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: keyType})
}

// newTestOntologyListModel creates an OntologyListModel pre-loaded with the
// given overview by simulating an OntologyDataMsg.
func newTestOntologyListModel(overview *OntologyOverview) OntologyListModel {
	p := &mockOntologyProvider{overview: overview}
	m := NewOntologyListModel(p)
	m.SetSize(60, 20)
	m, _ = m.Update(OntologyDataMsg{Overview: overview, Err: nil})
	return m
}

// sampleOntologyOverview builds a simple overview with n named contexts.
func sampleOntologyOverview(n int) *OntologyOverview {
	ctx := make([]OntologyInfo, n)
	names := []string{"alpha", "billing", "catalog", "delivery", "events"}
	for i := range n {
		name := names[i%len(names)]
		if i >= len(names) {
			name = names[i%len(names)] + "-extra"
		}
		ctx[i] = OntologyInfo{
			Name:        name,
			Description: "desc " + name,
		}
	}
	return &OntologyOverview{
		Telos:    "Test telos",
		Contexts: ctx,
	}
}

// ---------------------------------------------------------------------------
// Constructor and Init
// ---------------------------------------------------------------------------

// TestNewOntologyListModel_InitialState verifies that NewOntologyListModel
// returns a model with the provider wired and focused=true by default.
func TestNewOntologyListModel_InitialState(t *testing.T) {
	p := &mockOntologyProvider{}
	m := NewOntologyListModel(p)

	assert.True(t, m.focused, "model should be focused by default")
	assert.Equal(t, 0, m.cursor)
	assert.False(t, m.loaded)
	assert.Nil(t, m.navigable)
}

// TestOntologyListModel_Init_ReturnsFetchCmd verifies that Init returns a
// non-nil command (the fetchOntologyData Cmd).
func TestOntologyListModel_Init_ReturnsFetchCmd(t *testing.T) {
	p := &mockOntologyProvider{overview: &OntologyOverview{Telos: "hello"}}
	m := NewOntologyListModel(p)

	cmd := m.Init()
	require.NotNil(t, cmd, "Init should return fetchOntologyData as a Cmd")

	// Execute the cmd and verify it returns an OntologyDataMsg.
	msg := cmd()
	dataMsg, ok := msg.(OntologyDataMsg)
	require.True(t, ok, "cmd should return OntologyDataMsg")
	assert.Equal(t, "hello", dataMsg.Overview.Telos)
}

// ---------------------------------------------------------------------------
// SetSize and SetFocused
// ---------------------------------------------------------------------------

// TestOntologyListModel_SetSize_UpdatesDimensions verifies that SetSize stores
// the given width and height.
func TestOntologyListModel_SetSize_UpdatesDimensions(t *testing.T) {
	p := &mockOntologyProvider{}
	m := NewOntologyListModel(p)

	m.SetSize(80, 40)
	assert.Equal(t, 80, m.width)
	assert.Equal(t, 40, m.height)
}

// TestOntologyListModel_SetFocused_TogglesFocus verifies that SetFocused changes
// the focused field.
func TestOntologyListModel_SetFocused_TogglesFocus(t *testing.T) {
	p := &mockOntologyProvider{}
	m := NewOntologyListModel(p)
	require.True(t, m.focused)

	m.SetFocused(false)
	assert.False(t, m.focused)

	m.SetFocused(true)
	assert.True(t, m.focused)
}

// ---------------------------------------------------------------------------
// Update — OntologyDataMsg
// ---------------------------------------------------------------------------

// TestOntologyListModel_Update_DataMsg_LoadsContexts verifies that receiving
// an OntologyDataMsg without error populates items and sets loaded=true.
func TestOntologyListModel_Update_DataMsg_LoadsContexts(t *testing.T) {
	p := &mockOntologyProvider{}
	m := NewOntologyListModel(p)
	m.SetSize(60, 20)

	overview := sampleOntologyOverview(3)
	m, _ = m.Update(OntologyDataMsg{Overview: overview, Err: nil})

	assert.True(t, m.loaded)
	assert.Len(t, m.items, 3)
	assert.Equal(t, overview.Telos, m.telos)
}

// TestOntologyListModel_Update_DataMsg_WithError_ModelUnchanged verifies that
// an OntologyDataMsg with a non-nil error leaves the model unloaded.
func TestOntologyListModel_Update_DataMsg_WithError_ModelUnchanged(t *testing.T) {
	p := &mockOntologyProvider{}
	m := NewOntologyListModel(p)
	m.SetSize(60, 20)

	m, _ = m.Update(OntologyDataMsg{Err: errors.New("fetch failed")})

	assert.False(t, m.loaded, "error message should not mark model as loaded")
	assert.Empty(t, m.items)
}

// TestOntologyListModel_Update_DataMsg_NilOverview_LoadsEmpty verifies that an
// OntologyDataMsg with nil Overview (no error) marks the model loaded with
// zero contexts.
func TestOntologyListModel_Update_DataMsg_NilOverview_LoadsEmpty(t *testing.T) {
	p := &mockOntologyProvider{}
	m := NewOntologyListModel(p)
	m.SetSize(60, 20)

	m, _ = m.Update(OntologyDataMsg{Overview: nil, Err: nil})

	assert.True(t, m.loaded)
	assert.Empty(t, m.items)
}

// TestOntologyListModel_Update_StaleOverview_SetsStale verifies that an
// overview with Stale=true propagates to the model.
func TestOntologyListModel_Update_StaleOverview_SetsStale(t *testing.T) {
	m := newTestOntologyListModel(&OntologyOverview{Stale: true, Telos: "t"})
	assert.True(t, m.stale)
}

// ---------------------------------------------------------------------------
// Update — key messages
// ---------------------------------------------------------------------------

// TestOntologyListModel_Update_KeyMsg_Unfocused_Ignored verifies that key
// messages are ignored when the model is not focused.
func TestOntologyListModel_Update_KeyMsg_Unfocused_Ignored(t *testing.T) {
	m := newTestOntologyListModel(sampleOntologyOverview(3))
	m.SetFocused(false)
	startCursor := m.cursor

	m, _ = ontologySendKey(m, tea.KeyDown)
	assert.Equal(t, startCursor, m.cursor, "cursor should not move when unfocused")
}

// TestOntologyListModel_Update_KeyDown_MovesCursor verifies that pressing Down
// advances the cursor.
func TestOntologyListModel_Update_KeyDown_MovesCursor(t *testing.T) {
	m := newTestOntologyListModel(sampleOntologyOverview(3))
	require.Greater(t, len(m.navigable), 1)
	require.Equal(t, 0, m.cursor)

	m, _ = ontologySendKey(m, tea.KeyDown)
	assert.Equal(t, 1, m.cursor)
}

// TestOntologyListModel_Update_KeyUp_AtTop_StaysCursor verifies that pressing
// Up when the cursor is at the top does not move it below 0.
func TestOntologyListModel_Update_KeyUp_AtTop_StaysCursor(t *testing.T) {
	m := newTestOntologyListModel(sampleOntologyOverview(3))
	require.Equal(t, 0, m.cursor)

	m, _ = ontologySendKey(m, tea.KeyUp)
	assert.Equal(t, 0, m.cursor)
}

// TestOntologyListModel_Update_KeyDown_AtBottom_StaysCursor verifies that Down
// at the last item does not advance beyond the end.
func TestOntologyListModel_Update_KeyDown_AtBottom_StaysCursor(t *testing.T) {
	m := newTestOntologyListModel(sampleOntologyOverview(2))
	require.Greater(t, len(m.navigable), 0)
	lastIdx := len(m.navigable) - 1

	for range len(m.navigable) {
		m, _ = ontologySendKey(m, tea.KeyDown)
	}
	assert.Equal(t, lastIdx, m.cursor)
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// TestOntologyListModel_View_ZeroDimensions_ReturnsEmpty verifies that View
// returns "" when width or height is zero (not yet sized).
func TestOntologyListModel_View_ZeroDimensions_ReturnsEmpty(t *testing.T) {
	p := &mockOntologyProvider{}
	m := NewOntologyListModel(p)
	// width and height default to 0

	assert.Equal(t, "", m.View())
}

// TestOntologyListModel_View_EmptyContexts_ShowsNoContextsDefined verifies that
// when the model is loaded with no contexts the view shows a placeholder.
func TestOntologyListModel_View_EmptyContexts_ShowsNoContextsDefined(t *testing.T) {
	m := newTestOntologyListModel(&OntologyOverview{})

	view := ontologyStripAnsi(m.View())
	assert.Contains(t, view, "No contexts defined")
}

// TestOntologyListModel_View_WithContexts_ShowsContextNames verifies that
// context names appear in the rendered view.
func TestOntologyListModel_View_WithContexts_ShowsContextNames(t *testing.T) {
	overview := &OntologyOverview{
		Telos: "Test telos",
		Contexts: []OntologyInfo{
			{Name: "billing"},
			{Name: "auth"},
		},
	}
	m := newTestOntologyListModel(overview)

	view := ontologyStripAnsi(m.View())
	assert.Contains(t, view, "billing")
	assert.Contains(t, view, "auth")
}

// TestOntologyListModel_View_StaleWarning verifies that when stale=true the
// view shows the staleness warning.
func TestOntologyListModel_View_StaleWarning(t *testing.T) {
	m := newTestOntologyListModel(&OntologyOverview{
		Stale:    true,
		Telos:    "t",
		Contexts: []OntologyInfo{{Name: "ctx"}},
	})

	view := ontologyStripAnsi(m.View())
	assert.Contains(t, view, "stale")
}

// ---------------------------------------------------------------------------
// adjustScrollOffset
// ---------------------------------------------------------------------------

// TestOntologyListModel_AdjustScrollOffset_CursorAtTop_OffsetStaysZero verifies
// that when the cursor is at position 0 the scroll offset remains 0.
func TestOntologyListModel_AdjustScrollOffset_CursorAtTop_OffsetStaysZero(t *testing.T) {
	m := newTestOntologyListModel(sampleOntologyOverview(5))
	m.cursor = 0
	m.scrollOffset = 0

	m.adjustScrollOffset(10) // viewport larger than list
	assert.Equal(t, 0, m.scrollOffset)
}

// TestOntologyListModel_AdjustScrollOffset_CursorBelowViewport_OffsetAdvances
// verifies that when the cursor is below the visible viewport the offset
// advances to keep the cursor in view.
func TestOntologyListModel_AdjustScrollOffset_CursorBelowViewport_OffsetAdvances(t *testing.T) {
	m := newTestOntologyListModel(sampleOntologyOverview(5))
	require.GreaterOrEqual(t, len(m.navigable), 5)

	m.cursor = 4
	m.scrollOffset = 0

	m.adjustScrollOffset(3) // viewport shows only 3 items
	assert.Greater(t, m.scrollOffset, 0, "scroll offset should advance when cursor is below viewport")
}

// TestOntologyListModel_AdjustScrollOffset_CursorAboveOffset_OffsetDecreases
// verifies that when the cursor moves above the current scrollOffset the offset
// decreases to match the cursor.
func TestOntologyListModel_AdjustScrollOffset_CursorAboveOffset_OffsetDecreases(t *testing.T) {
	m := newTestOntologyListModel(sampleOntologyOverview(5))
	require.GreaterOrEqual(t, len(m.navigable), 5)

	m.cursor = 0
	m.scrollOffset = 3 // cursor is above the current view

	m.adjustScrollOffset(3)
	assert.Equal(t, 0, m.scrollOffset, "offset should match cursor when cursor is above viewport")
}

// ---------------------------------------------------------------------------
// buildNavigableItems
// ---------------------------------------------------------------------------

// TestOntologyListModel_BuildNavigableItems_NoFilter_AllIncluded verifies that
// with an empty filter query all items are included in navigable.
func TestOntologyListModel_BuildNavigableItems_NoFilter_AllIncluded(t *testing.T) {
	m := newTestOntologyListModel(sampleOntologyOverview(3))

	m.filterQuery = ""
	m.buildNavigableItems()

	assert.Len(t, m.navigable, 3)
}

// TestOntologyListModel_BuildNavigableItems_FilterMatch verifies that the filter
// query narrows the navigable list to matching items.
func TestOntologyListModel_BuildNavigableItems_FilterMatch(t *testing.T) {
	overview := &OntologyOverview{
		Contexts: []OntologyInfo{
			{Name: "billing"},
			{Name: "auth"},
			{Name: "analytics"},
		},
	}
	m := newTestOntologyListModel(overview)

	m.filterQuery = "an" // matches "analytics"
	m.buildNavigableItems()

	require.Len(t, m.navigable, 1)
	assert.Equal(t, "analytics", m.navigable[0].Name)
}

// ---------------------------------------------------------------------------
// fetchOntologyData (the tea.Cmd function)
// ---------------------------------------------------------------------------

// TestOntologyListModel_FetchOntologyData_NilProvider_ReturnsEmptyMsg verifies
// that when provider is nil fetchOntologyData returns an OntologyDataMsg with
// no error and nil overview.
func TestOntologyListModel_FetchOntologyData_NilProvider_ReturnsEmptyMsg(t *testing.T) {
	m := NewOntologyListModel(nil)

	msg := m.fetchOntologyData()
	dataMsg, ok := msg.(OntologyDataMsg)
	require.True(t, ok)
	assert.Nil(t, dataMsg.Err)
	assert.Nil(t, dataMsg.Overview)
}

// TestOntologyListModel_FetchOntologyData_WithProvider_ReturnsOverview verifies
// that fetchOntologyData calls the provider and wraps the result.
func TestOntologyListModel_FetchOntologyData_WithProvider_ReturnsOverview(t *testing.T) {
	expected := &OntologyOverview{Telos: "shipped"}
	p := &mockOntologyProvider{overview: expected}
	m := NewOntologyListModel(p)

	msg := m.fetchOntologyData()
	dataMsg, ok := msg.(OntologyDataMsg)
	require.True(t, ok)
	assert.Nil(t, dataMsg.Err)
	assert.Equal(t, "shipped", dataMsg.Overview.Telos)
}

// TestOntologyListModel_FetchOntologyData_ProviderError_ReturnsErrMsg verifies
// that a provider error is surfaced in the OntologyDataMsg.Err field.
func TestOntologyListModel_FetchOntologyData_ProviderError_ReturnsErrMsg(t *testing.T) {
	p := &mockOntologyProvider{err: errors.New("network failure")}
	m := NewOntologyListModel(p)

	msg := m.fetchOntologyData()
	dataMsg, ok := msg.(OntologyDataMsg)
	require.True(t, ok)
	require.Error(t, dataMsg.Err)
	assert.Contains(t, dataMsg.Err.Error(), "network failure")
}

// ---------------------------------------------------------------------------
// emitSelectionMsg
// ---------------------------------------------------------------------------

// TestOntologyListModel_EmitSelectionMsg_EmptyNavigable_ReturnsNil verifies
// that emitSelectionMsg returns nil when there are no navigable items.
func TestOntologyListModel_EmitSelectionMsg_EmptyNavigable_ReturnsNil(t *testing.T) {
	m := NewOntologyListModel(nil)
	m.navigable = nil

	cmd := m.emitSelectionMsg()
	assert.Nil(t, cmd)
}

// TestOntologyListModel_EmitSelectionMsg_WithItems_ReturnsSelectedMsg verifies
// that emitSelectionMsg emits an OntologySelectedMsg with the current context's
// Name and the cursor index.
func TestOntologyListModel_EmitSelectionMsg_WithItems_ReturnsSelectedMsg(t *testing.T) {
	m := newTestOntologyListModel(&OntologyOverview{
		Contexts: []OntologyInfo{
			{Name: "alpha"},
			{Name: "beta"},
		},
	})
	require.GreaterOrEqual(t, len(m.navigable), 2)
	m.cursor = 1

	cmd := m.emitSelectionMsg()
	require.NotNil(t, cmd)

	msg := cmd()
	selMsg, ok := msg.(OntologySelectedMsg)
	require.True(t, ok)
	assert.Equal(t, "beta", selMsg.Name)
	assert.Equal(t, 1, selMsg.Index)
}

// ---------------------------------------------------------------------------
// formatAge (package-level function)
// ---------------------------------------------------------------------------

// TestFormatAge_Various verifies the human-readable age formatting without an
// "ago" suffix (distinct from webui.formatTimeAgo).
func TestFormatAge_Various(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"less than a minute", 30 * time.Second, "just now"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours", 3 * time.Hour, "3h"},
		{"days", 48 * time.Hour, "2d"},
		{"zero duration", 0, "just now"},
		{"exactly one minute", 60 * time.Second, "1m"},
		{"exactly one hour", 60 * time.Minute, "1h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAge(tt.d)
			assert.Equal(t, tt.want, got)
		})
	}
}
