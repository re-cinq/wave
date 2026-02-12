package display

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestProgressModel_Update_PKeyIgnored(t *testing.T) {
	ctx := &PipelineContext{
		PipelineName:      "test-pipeline",
		TotalSteps:        2,
		CurrentStepNum:    1,
		OverallProgress:   50,
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
		StepStatuses:      map[string]ProgressState{},
	}
	model := NewProgressModel(ctx)

	// Send a 'p' keypress — should be ignored (no state change, no command)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	updatedModel, cmd := model.Update(msg)

	if cmd != nil {
		t.Error("pressing 'p' should not produce a command")
	}

	pm := updatedModel.(*ProgressModel)
	if pm.quit {
		t.Error("pressing 'p' should not set quit")
	}
}

func TestProgressModel_Update_QKeyQuits(t *testing.T) {
	ctx := &PipelineContext{
		PipelineName:      "test-pipeline",
		TotalSteps:        1,
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
		StepStatuses:      map[string]ProgressState{},
	}
	model := NewProgressModel(ctx)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedModel, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("pressing 'q' should produce a quit command")
	}

	pm := updatedModel.(*ProgressModel)
	if !pm.quit {
		t.Error("pressing 'q' should set quit to true")
	}
}

func TestProgressModel_Update_TickAlwaysContinues(t *testing.T) {
	ctx := &PipelineContext{
		PipelineName:      "test-pipeline",
		TotalSteps:        1,
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
		StepStatuses:      map[string]ProgressState{},
	}
	model := NewProgressModel(ctx)

	// Send a tick — should always produce the next tick command
	msg := TickMsg(time.Now())
	_, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("tick should always produce the next tick command")
	}
}

func TestProgressModel_View_StatusLineShowsOnlyQuit(t *testing.T) {
	ctx := &PipelineContext{
		PipelineName:      "test-pipeline",
		TotalSteps:        2,
		CurrentStepNum:    1,
		OverallProgress:   50,
		ManifestPath:      "wave.yaml",
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
		StepStatuses:      map[string]ProgressState{},
	}
	model := NewProgressModel(ctx)

	view := model.View()

	if strings.Contains(view, "p=pause") {
		t.Error("View should not contain 'p=pause'")
	}
	if strings.Contains(view, "PAUSED") {
		t.Error("View should not contain 'PAUSED'")
	}
	if !strings.Contains(view, "q=quit") {
		t.Error("View should contain 'q=quit'")
	}
}

func TestProgressModel_NoPausedField(t *testing.T) {
	ctx := &PipelineContext{
		PipelineName:      "test-pipeline",
		TotalSteps:        1,
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
		StepStatuses:      map[string]ProgressState{},
	}
	model := NewProgressModel(ctx)

	// Verify the model has no paused state by sending multiple 'p' keys
	// and verifying tick behavior remains consistent
	pMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	model.Update(pMsg)
	model.Update(pMsg)

	// Tick should still produce next tick command
	tickMsg := TickMsg(time.Now())
	_, cmd := model.Update(tickMsg)
	if cmd == nil {
		t.Error("tick should still produce next tick command after 'p' key presses")
	}
}
