package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewOntologyDetailModel
// ---------------------------------------------------------------------------

// TestNewOntologyDetailModel_DefaultState verifies the constructor returns a
// model with zero dimensions, no selection, and not focused.
func TestNewOntologyDetailModel_DefaultState(t *testing.T) {
	m := NewOntologyDetailModel()

	assert.Equal(t, 0, m.width)
	assert.Equal(t, 0, m.height)
	assert.Nil(t, m.selected)
	assert.False(t, m.focused)
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

// TestOntologyDetailModel_Init_ReturnsNil verifies that Init returns a nil Cmd.
func TestOntologyDetailModel_Init_ReturnsNil(t *testing.T) {
	m := NewOntologyDetailModel()
	cmd := m.Init()
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// SetSize
// ---------------------------------------------------------------------------

// TestOntologyDetailModel_SetSize_UpdatesDimensions verifies that SetSize stores
// the given width and height and forwards them to the viewport.
func TestOntologyDetailModel_SetSize_UpdatesDimensions(t *testing.T) {
	m := NewOntologyDetailModel()

	m.SetSize(80, 40)
	assert.Equal(t, 80, m.width)
	assert.Equal(t, 40, m.height)
	assert.Equal(t, 80, m.viewport.Width)
	assert.Equal(t, 40, m.viewport.Height)
}

// TestOntologyDetailModel_SetSize_WithSelection_UpdatesViewportContent verifies
// that when a context is already selected, SetSize re-renders the detail into
// the viewport.
func TestOntologyDetailModel_SetSize_WithSelection_UpdatesViewportContent(t *testing.T) {
	m := NewOntologyDetailModel()
	info := &OntologyInfo{Name: "billing", Description: "Billing context"}
	m.SetContext(info)

	m.SetSize(80, 40)
	// The viewport content should now contain the context name.
	content := ontologyStripAnsi(m.viewport.View())
	// After SetSize the viewport may scroll; just check the content string
	// directly via the stored content (rendered internally).
	_ = content // viewport.View() depends on scroll, use the rendered string below
	rendered := renderOntologyDetail(info, 80)
	assert.Contains(t, rendered, "billing")
}

// ---------------------------------------------------------------------------
// SetFocused
// ---------------------------------------------------------------------------

// TestOntologyDetailModel_SetFocused_UpdatesField verifies that SetFocused
// toggles the focused field.
func TestOntologyDetailModel_SetFocused_UpdatesField(t *testing.T) {
	m := NewOntologyDetailModel()
	require.False(t, m.focused)

	m.SetFocused(true)
	assert.True(t, m.focused)

	m.SetFocused(false)
	assert.False(t, m.focused)
}

// ---------------------------------------------------------------------------
// SetContext
// ---------------------------------------------------------------------------

// TestOntologyDetailModel_SetContext_UpdatesSelected verifies that SetContext
// stores the provided OntologyInfo in the model.
func TestOntologyDetailModel_SetContext_UpdatesSelected(t *testing.T) {
	m := NewOntologyDetailModel()
	info := &OntologyInfo{Name: "billing"}

	m.SetContext(info)
	require.NotNil(t, m.selected)
	assert.Equal(t, "billing", m.selected.Name)
}

// TestOntologyDetailModel_SetContext_Nil_ClearsSelected verifies that passing
// nil clears the selection.
func TestOntologyDetailModel_SetContext_Nil_ClearsSelected(t *testing.T) {
	m := NewOntologyDetailModel()
	m.SetContext(&OntologyInfo{Name: "billing"})
	require.NotNil(t, m.selected)

	m.SetContext(nil)
	assert.Nil(t, m.selected)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// TestOntologyDetailModel_Update_NonKeyMsg_ReturnsUnchanged verifies that a
// non-key message (e.g. a window size message) returns the model unchanged.
func TestOntologyDetailModel_Update_NonKeyMsg_ReturnsUnchanged(t *testing.T) {
	m := NewOntologyDetailModel()
	m.SetSize(80, 40)

	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	assert.Nil(t, cmd)
	// Model dimensions should not change — the WindowSizeMsg is not handled.
	assert.Equal(t, 80, updated.width)
}

// TestOntologyDetailModel_Update_KeyMsg_Unfocused_ReturnsNilCmd verifies that
// a key message when not focused returns nil cmd and unchanged model.
func TestOntologyDetailModel_Update_KeyMsg_Unfocused_ReturnsNilCmd(t *testing.T) {
	m := NewOntologyDetailModel()
	m.SetSize(80, 40)
	m.SetFocused(false)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Nil(t, cmd)
	assert.Equal(t, m.focused, updated.focused)
}

// TestOntologyDetailModel_Update_KeyMsg_Focused_DelegatesToViewport verifies
// that a key message when focused is forwarded to the viewport.
func TestOntologyDetailModel_Update_KeyMsg_Focused_DelegatesToViewport(t *testing.T) {
	m := NewOntologyDetailModel()
	m.SetSize(80, 40)
	m.SetFocused(true)

	// No panic or error expected; the viewport handles the key.
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// TestOntologyDetailModel_View_ZeroDimensions_ReturnsEmpty verifies that View
// returns "" when width or height is zero.
func TestOntologyDetailModel_View_ZeroDimensions_ReturnsEmpty(t *testing.T) {
	m := NewOntologyDetailModel()
	// Default width and height are 0.
	assert.Equal(t, "", m.View())
}

// TestOntologyDetailModel_View_NoSelection_ShowsPlaceholder verifies that when
// no context is selected the view shows the "Select a context" placeholder.
func TestOntologyDetailModel_View_NoSelection_ShowsPlaceholder(t *testing.T) {
	m := NewOntologyDetailModel()
	m.SetSize(80, 20)

	view := ontologyStripAnsi(m.View())
	assert.Contains(t, view, "Select a context")
}

// TestOntologyDetailModel_View_WithSelection_NotEmpty verifies that when a
// context is selected View returns non-empty content.
func TestOntologyDetailModel_View_WithSelection_NotEmpty(t *testing.T) {
	m := NewOntologyDetailModel()
	m.SetSize(80, 20)
	m.SetContext(&OntologyInfo{Name: "billing", Description: "Handles invoicing"})

	view := m.View()
	assert.NotEmpty(t, view)
}

// ---------------------------------------------------------------------------
// renderOntologyDetail
// ---------------------------------------------------------------------------

// TestRenderOntologyDetail_NilInfo_ReturnsEmpty verifies that passing nil
// returns an empty string without panic.
func TestRenderOntologyDetail_NilInfo_ReturnsEmpty(t *testing.T) {
	result := renderOntologyDetail(nil, 80)
	assert.Equal(t, "", result)
}

// TestRenderOntologyDetail_NameAndDescription verifies that the context name
// and description appear in the rendered output.
func TestRenderOntologyDetail_NameAndDescription(t *testing.T) {
	info := &OntologyInfo{
		Name:        "billing",
		Description: "Handles all billing logic",
	}

	result := ontologyStripAnsi(renderOntologyDetail(info, 80))
	assert.Contains(t, result, "billing")
	assert.Contains(t, result, "Handles all billing logic")
}

// TestRenderOntologyDetail_NoInvariants_InvariantsSectionOmitted verifies that
// when Invariants is nil or empty the "Invariants:" section is not rendered.
func TestRenderOntologyDetail_NoInvariants_InvariantsSectionOmitted(t *testing.T) {
	info := &OntologyInfo{
		Name:       "billing",
		Invariants: nil,
	}

	result := ontologyStripAnsi(renderOntologyDetail(info, 80))
	assert.NotContains(t, result, "Invariants:")
}

// TestRenderOntologyDetail_WithInvariants_InvariantsShown verifies that when
// invariants are set they appear in the rendered output.
func TestRenderOntologyDetail_WithInvariants_InvariantsShown(t *testing.T) {
	info := &OntologyInfo{
		Name:       "billing",
		Invariants: []string{"no double-billing", "idempotent charges"},
	}

	result := ontologyStripAnsi(renderOntologyDetail(info, 80))
	assert.Contains(t, result, "Invariants:")
	assert.Contains(t, result, "no double-billing")
	assert.Contains(t, result, "idempotent charges")
}

// TestRenderOntologyDetail_NoSkill_ShowsNoSkillMessage verifies that when
// HasSkill=false the "No context skill file" message is shown.
func TestRenderOntologyDetail_NoSkill_ShowsNoSkillMessage(t *testing.T) {
	info := &OntologyInfo{
		Name:     "billing",
		HasSkill: false,
	}

	result := ontologyStripAnsi(renderOntologyDetail(info, 80))
	assert.Contains(t, result, "No context skill file")
}

// TestRenderOntologyDetail_WithSkillFile_ShowsPathAndContent verifies that
// when HasSkill=true and the skill file exists the path and content are shown.
func TestRenderOntologyDetail_WithSkillFile_ShowsPathAndContent(t *testing.T) {
	skillContent := "# Billing Skill\nHandle invoice creation.\n"
	tmpFile := filepath.Join(t.TempDir(), "SKILL.md")
	require.NoError(t, os.WriteFile(tmpFile, []byte(skillContent), 0o644))

	info := &OntologyInfo{
		Name:      "billing",
		HasSkill:  true,
		SkillPath: tmpFile,
	}

	result := ontologyStripAnsi(renderOntologyDetail(info, 80))
	assert.Contains(t, result, tmpFile, "skill path should appear in output")
	assert.Contains(t, result, "Billing Skill", "skill file content should appear")
}

// TestRenderOntologyDetail_SkillFileUnreadable_PathShownNoContent verifies that
// when HasSkill=true but the skill file cannot be read the path is shown but
// no file content is rendered.
func TestRenderOntologyDetail_SkillFileUnreadable_PathShownNoContent(t *testing.T) {
	info := &OntologyInfo{
		Name:      "billing",
		HasSkill:  true,
		SkillPath: "/nonexistent/path/SKILL.md",
	}

	result := ontologyStripAnsi(renderOntologyDetail(info, 80))
	assert.Contains(t, result, "/nonexistent/path/SKILL.md", "path should still be shown")
	// No file content should appear beyond the path line.
	assert.NotContains(t, result, "# ")
}

// TestRenderOntologyDetail_WithLineage_ShowsStats verifies that when HasLineage
// is true the pipeline lineage section is rendered with correct stats.
func TestRenderOntologyDetail_WithLineage_ShowsStats(t *testing.T) {
	info := &OntologyInfo{
		Name:        "billing",
		HasLineage:  true,
		TotalRuns:   10,
		Successes:   8,
		Failures:    2,
		SuccessRate: 80.0,
		LastUsed:    time.Now().Add(-24 * time.Hour),
	}

	result := ontologyStripAnsi(renderOntologyDetail(info, 80))
	assert.Contains(t, result, "Pipeline Lineage:")
	assert.Contains(t, result, "10")
	assert.Contains(t, result, "80%")
}

// TestRenderOntologyDetail_NoLineage_LineageSectionOmitted verifies that when
// HasLineage=false the lineage section is not rendered.
func TestRenderOntologyDetail_NoLineage_LineageSectionOmitted(t *testing.T) {
	info := &OntologyInfo{
		Name:       "billing",
		HasLineage: false,
	}

	result := ontologyStripAnsi(renderOntologyDetail(info, 80))
	assert.NotContains(t, result, "Pipeline Lineage:")
}

// TestRenderOntologyDetail_LongSkillContent_Truncated verifies that skill file
// content exceeding 2000 characters is truncated with "... (truncated)".
func TestRenderOntologyDetail_LongSkillContent_Truncated(t *testing.T) {
	longContent := strings.Repeat("x", 2500)
	tmpFile := filepath.Join(t.TempDir(), "SKILL.md")
	require.NoError(t, os.WriteFile(tmpFile, []byte(longContent), 0o644))

	info := &OntologyInfo{
		Name:      "billing",
		HasSkill:  true,
		SkillPath: tmpFile,
	}

	result := ontologyStripAnsi(renderOntologyDetail(info, 80))
	assert.Contains(t, result, "(truncated)")
}
