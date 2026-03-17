package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusBarModel_View_ContainsHints(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)
	view := sb.View()

	assert.Contains(t, view, "q: quit")
	assert.Contains(t, view, "Tab/Shift+Tab: views")
}

func TestStatusBarModel_View_ContainsContextLabel(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)
	view := sb.View()

	assert.Contains(t, view, "Pipelines")
}

func TestStatusBarModel_SetWidth(t *testing.T) {
	sb := NewStatusBarModel()
	assert.Equal(t, 0, sb.width)

	sb.SetWidth(80)
	assert.Equal(t, 80, sb.width)
}

func TestStatusBarModel_DefaultContextLabel(t *testing.T) {
	sb := NewStatusBarModel()
	assert.Equal(t, "Pipelines", sb.contextLabel)
}

func TestStatusBarModel_View_DefaultLeftPaneHints(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)
	view := sb.View()

	assert.Contains(t, view, "Enter: view")
	assert.Contains(t, view, "↑↓: navigate")
	assert.Contains(t, view, "/: filter")
}

func TestStatusBarModel_Update_FocusChangedToRight(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneRight})
	view := sb.View()

	assert.Contains(t, view, "↑↓: scroll")
	assert.Contains(t, view, "Esc: back")
	assert.NotContains(t, view, "/: filter")
}

func TestStatusBarModel_Update_FocusChangedBackToLeft(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	// Switch to right
	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneRight})
	// Switch back to left
	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneLeft})
	view := sb.View()

	assert.Contains(t, view, "↑↓: navigate")
	assert.Contains(t, view, "Enter: view")
	assert.Contains(t, view, "/: filter")
}

// ===========================================================================
// T019: Status bar form hint tests
// ===========================================================================

func TestStatusBarModel_FormActiveMsg_SetsFormActive(t *testing.T) {
	sb := NewStatusBarModel()

	sb, _ = sb.Update(FormActiveMsg{Active: true})
	assert.True(t, sb.formActive)
}

func TestStatusBarModel_FormActive_RightPane_ShowsFormHints(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	// Set form active and focus to right pane
	sb, _ = sb.Update(FormActiveMsg{Active: true})
	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneRight})

	view := sb.View()
	assert.Contains(t, view, "Tab: next")
	assert.Contains(t, view, "Enter: launch")
	assert.Contains(t, view, "Esc: cancel")
}

func TestStatusBarModel_FormInactive_RevertsToDefault(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	// Activate form
	sb, _ = sb.Update(FormActiveMsg{Active: true})
	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneRight})
	assert.True(t, sb.formActive)

	// Deactivate form and switch back to left
	sb, _ = sb.Update(FormActiveMsg{Active: false})
	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneLeft})

	view := sb.View()
	assert.Contains(t, view, "Enter: view")
	assert.Contains(t, view, "/: filter")
}

// ===========================================================================
// T037: Status bar finished detail hint tests
// ===========================================================================

func TestStatusBarModel_FinishedDetailActiveMsg_SetsField(t *testing.T) {
	sb := NewStatusBarModel()

	sb, _ = sb.Update(FinishedDetailActiveMsg{Active: true})
	assert.True(t, sb.finishedDetailActive)

	sb, _ = sb.Update(FinishedDetailActiveMsg{Active: false})
	assert.False(t, sb.finishedDetailActive)
}

func TestStatusBarModel_FinishedDetailActive_RightPane_ShowsFinishedHints(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	sb, _ = sb.Update(FinishedDetailActiveMsg{Active: true})
	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneRight})

	view := sb.View()
	assert.Contains(t, view, "[Enter] Chat")
	assert.Contains(t, view, "[b] Branch")
	assert.Contains(t, view, "[d] Diff")
	assert.Contains(t, view, "[Esc] Back")
}

func TestStatusBarModel_HintPriority_FormOverFinishedDetail(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	// Both form and finished detail active
	sb, _ = sb.Update(FormActiveMsg{Active: true})
	sb, _ = sb.Update(FinishedDetailActiveMsg{Active: true})
	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneRight})

	view := sb.View()
	// Form hints should take priority
	assert.Contains(t, view, "Tab: next")
	assert.Contains(t, view, "Enter: launch")
	assert.NotContains(t, view, "[Enter] Chat")
}

func TestStatusBarModel_HintPriority_LiveOutputOverFinishedDetail(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	// Both live output and finished detail active
	sb, _ = sb.Update(LiveOutputActiveMsg{Active: true})
	sb, _ = sb.Update(FinishedDetailActiveMsg{Active: true})
	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneRight})

	view := sb.View()
	// Live output hints should take priority
	assert.Contains(t, view, "v: verbose")
	assert.NotContains(t, view, "[Enter] Chat")
}

func TestStatusBarModel_FinishedDetailInactive_RevertsToGeneric(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	// Activate finished detail
	sb, _ = sb.Update(FinishedDetailActiveMsg{Active: true})
	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneRight})
	view := sb.View()
	assert.Contains(t, view, "[Enter] Chat")

	// Deactivate
	sb, _ = sb.Update(FinishedDetailActiveMsg{Active: false})
	view = sb.View()
	assert.Contains(t, view, "↑↓: scroll")
	assert.NotContains(t, view, "[Enter] Chat")
}

// ===========================================================================
// T020: Status bar compose mode hint tests
// ===========================================================================

func TestStatusBarModel_ComposeActiveMsg_True_ShowsComposeHints(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	sb, _ = sb.Update(ComposeActiveMsg{Active: true})

	view := sb.View()
	assert.Contains(t, view, "add", "compose hints should mention add")
	assert.Contains(t, view, "remove", "compose hints should mention remove")
	assert.Contains(t, view, "reorder", "compose hints should mention reorder")
	assert.Contains(t, view, "Esc", "compose hints should mention Esc")
}

func TestStatusBarModel_ComposeActiveMsg_False_RestoresDefaultHints(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	// Activate compose mode
	sb, _ = sb.Update(ComposeActiveMsg{Active: true})
	view := sb.View()
	assert.Contains(t, view, "reorder")

	// Deactivate compose mode
	sb, _ = sb.Update(ComposeActiveMsg{Active: false})
	view = sb.View()
	assert.Contains(t, view, "navigate", "default hints should contain navigate")
	assert.NotContains(t, view, "reorder", "compose hints should be gone after deactivation")
}

func TestStatusBarModel_ComposeHints_ContainExpectedKeybindings(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	sb, _ = sb.Update(ComposeActiveMsg{Active: true})

	view := sb.View()
	assert.Contains(t, view, "a: add", "compose hints should contain 'a: add'")
	assert.Contains(t, view, "x: remove", "compose hints should contain 'x: remove'")
	assert.Contains(t, view, "Shift+↑↓: reorder", "compose hints should contain 'Shift+↑↓: reorder'")
	assert.Contains(t, view, "Enter: start", "compose hints should contain 'Enter: start'")
	assert.Contains(t, view, "Esc: cancel", "compose hints should contain 'Esc: cancel'")
}

// ===========================================================================
// Running info and event log hint tests
// ===========================================================================

func TestStatusBarModel_RunningInfoActiveMsg_SetsField(t *testing.T) {
	sb := NewStatusBarModel()

	sb, _ = sb.Update(RunningInfoActiveMsg{Active: true})
	assert.True(t, sb.runningInfoActive)

	sb, _ = sb.Update(RunningInfoActiveMsg{Active: false})
	assert.False(t, sb.runningInfoActive)
}

func TestStatusBarModel_RunningInfoActive_RightPane_ShowsDismissHint(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	sb, _ = sb.Update(RunningInfoActiveMsg{Active: true})
	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneRight})

	view := sb.View()
	assert.Contains(t, view, "c: dismiss")
	assert.Contains(t, view, "l: logs")
}

func TestStatusBarModel_FinishedDetailHints_IncludesLogs(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	sb, _ = sb.Update(FinishedDetailActiveMsg{Active: true})
	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneRight})

	view := sb.View()
	assert.Contains(t, view, "[l] Logs")
}

func TestStatusBarModel_HintPriority_LiveOutputOverRunningInfo(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)

	sb, _ = sb.Update(LiveOutputActiveMsg{Active: true})
	sb, _ = sb.Update(RunningInfoActiveMsg{Active: true})
	sb, _ = sb.Update(FocusChangedMsg{Pane: FocusPaneRight})

	view := sb.View()
	// Live output hints should take priority
	assert.Contains(t, view, "v: verbose")
	assert.NotContains(t, view, "c: dismiss")
}

func TestStatusBarModel_NonGuidedMode_HealthView_ShowsStandardHints(t *testing.T) {
	sb := NewStatusBarModel()
	sb.SetWidth(120)
	sb, _ = sb.Update(ViewChangedMsg{View: ViewHealth})

	view := sb.View()
	assert.Contains(t, view, "r: recheck")
}
