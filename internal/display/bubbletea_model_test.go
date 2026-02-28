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

func TestProgressModel_View_HandoverMetadata_VerboseMode(t *testing.T) {
	ctx := &PipelineContext{
		PipelineName:      "test-pipeline",
		TotalSteps:        3,
		CurrentStepNum:    2,
		OverallProgress:   33,
		ManifestPath:      "wave.yaml",
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
		StepStatuses: map[string]ProgressState{
			"analyst":     StateCompleted,
			"implementer": StateRunning,
			"reviewer":    StateNotStarted,
		},
		StepOrder: []string{"analyst", "implementer", "reviewer"},
		StepPersonas: map[string]string{
			"analyst":     "analyst",
			"implementer": "implementer",
			"reviewer":    "reviewer",
		},
		StepDurations: map[string]int64{
			"analyst": 45200,
		},
		StepStartTimes: map[string]int64{
			"implementer": time.Now().UnixNano(),
		},
		Verbose: true,
		HandoversByStep: map[string]*HandoverInfo{
			"analyst": {
				ArtifactPaths:  []string{".wave/artifacts/analysis"},
				ContractStatus: "passed",
				ContractSchema: "json_schema",
				TargetStep:     "implementer",
			},
		},
	}
	model := NewProgressModel(ctx)
	view := model.View()

	// Should contain artifact line
	if !strings.Contains(view, "artifact: .wave/artifacts/analysis (written)") {
		t.Errorf("Verbose view should contain artifact path, got:\n%s", view)
	}

	// Should contain contract line
	if !strings.Contains(view, "contract: json_schema") {
		t.Errorf("Verbose view should contain contract schema, got:\n%s", view)
	}
	if !strings.Contains(view, "valid") {
		t.Errorf("Verbose view should contain 'valid' for passed contract, got:\n%s", view)
	}

	// Should contain handover target line
	if !strings.Contains(view, "handover") && !strings.Contains(view, "implementer") {
		t.Errorf("Verbose view should contain handover target, got:\n%s", view)
	}
}

func TestProgressModel_View_HandoverMetadata_NonVerboseMode(t *testing.T) {
	ctx := &PipelineContext{
		PipelineName:      "test-pipeline",
		TotalSteps:        2,
		CurrentStepNum:    2,
		OverallProgress:   50,
		ManifestPath:      "wave.yaml",
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
		StepStatuses: map[string]ProgressState{
			"analyst":     StateCompleted,
			"implementer": StateRunning,
		},
		StepOrder: []string{"analyst", "implementer"},
		StepPersonas: map[string]string{
			"analyst":     "analyst",
			"implementer": "implementer",
		},
		StepDurations: map[string]int64{
			"analyst": 30000,
		},
		StepStartTimes: map[string]int64{
			"implementer": time.Now().UnixNano(),
		},
		Verbose: false, // Not verbose
		HandoversByStep: map[string]*HandoverInfo{
			"analyst": {
				ArtifactPaths:  []string{".wave/artifacts/analysis"},
				ContractStatus: "passed",
				ContractSchema: "json_schema",
				TargetStep:     "implementer",
			},
		},
	}
	model := NewProgressModel(ctx)
	view := model.View()

	// Should NOT contain handover metadata when not verbose
	if strings.Contains(view, "artifact: .wave/artifacts/analysis") {
		t.Errorf("Non-verbose view should NOT contain artifact path, got:\n%s", view)
	}
	if strings.Contains(view, "contract: json_schema") {
		t.Errorf("Non-verbose view should NOT contain contract info, got:\n%s", view)
	}
	if strings.Contains(view, "handover") {
		t.Errorf("Non-verbose view should NOT contain handover target, got:\n%s", view)
	}
}

func TestProgressModel_View_HandoverMetadata_TreeConnectors(t *testing.T) {
	ctx := &PipelineContext{
		PipelineName:      "test-pipeline",
		TotalSteps:        2,
		CurrentStepNum:    2,
		OverallProgress:   50,
		ManifestPath:      "wave.yaml",
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
		StepStatuses: map[string]ProgressState{
			"analyst":     StateCompleted,
			"implementer": StateRunning,
		},
		StepOrder: []string{"analyst", "implementer"},
		StepPersonas: map[string]string{
			"analyst":     "analyst",
			"implementer": "implementer",
		},
		StepDurations: map[string]int64{
			"analyst": 30000,
		},
		StepStartTimes: map[string]int64{
			"implementer": time.Now().UnixNano(),
		},
		DeliverablesByStep: map[string][]string{
			"analyst": {"spec.md written"},
		},
		Verbose: true,
		HandoversByStep: map[string]*HandoverInfo{
			"analyst": {
				ArtifactPaths:  []string{".wave/artifacts/analysis"},
				ContractStatus: "passed",
				ContractSchema: "json_schema",
				TargetStep:     "implementer",
			},
		},
	}
	model := NewProgressModel(ctx)
	view := model.View()

	// Should contain deliverable
	if !strings.Contains(view, "spec.md written") {
		t.Errorf("View should contain deliverable, got:\n%s", view)
	}

	// Should contain handover metadata
	if !strings.Contains(view, "artifact: .wave/artifacts/analysis (written)") {
		t.Errorf("View should contain artifact path, got:\n%s", view)
	}

	// The last metadata line should use └─ connector
	if !strings.Contains(view, "└─") {
		t.Errorf("View should contain └─ connector for last metadata line, got:\n%s", view)
	}

	// Intermediate lines should use ├─ connector
	if !strings.Contains(view, "├─") {
		t.Errorf("View should contain ├─ connector for intermediate metadata lines, got:\n%s", view)
	}
}

func TestProgressModel_View_HandoverMetadata_FailedContract(t *testing.T) {
	ctx := &PipelineContext{
		PipelineName:      "test-pipeline",
		TotalSteps:        2,
		CurrentStepNum:    1,
		OverallProgress:   50,
		ManifestPath:      "wave.yaml",
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
		StepStatuses: map[string]ProgressState{
			"analyst":     StateCompleted,
			"implementer": StateNotStarted,
		},
		StepOrder: []string{"analyst", "implementer"},
		StepDurations: map[string]int64{
			"analyst": 30000,
		},
		Verbose: true,
		HandoversByStep: map[string]*HandoverInfo{
			"analyst": {
				ContractStatus: "failed",
				ContractSchema: "json_schema",
			},
		},
	}
	model := NewProgressModel(ctx)
	view := model.View()

	// Should show failed contract status
	if !strings.Contains(view, "failed") {
		t.Errorf("View should contain 'failed' for failed contract, got:\n%s", view)
	}
}

func TestProgressModel_View_HandoverMetadata_NoHandoverForLastStep(t *testing.T) {
	ctx := &PipelineContext{
		PipelineName:      "test-pipeline",
		TotalSteps:        1,
		CurrentStepNum:    1,
		OverallProgress:   100,
		ManifestPath:      "wave.yaml",
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
		StepStatuses: map[string]ProgressState{
			"reviewer": StateCompleted,
		},
		StepOrder: []string{"reviewer"},
		StepDurations: map[string]int64{
			"reviewer": 20000,
		},
		Verbose: true,
		HandoversByStep: map[string]*HandoverInfo{
			"reviewer": {
				ArtifactPaths: []string{".wave/artifacts/review"},
			},
		},
	}
	model := NewProgressModel(ctx)
	view := model.View()

	// Should contain artifact but no handover target (last step)
	if !strings.Contains(view, "artifact: .wave/artifacts/review (written)") {
		t.Errorf("View should contain artifact for last step, got:\n%s", view)
	}
	if strings.Contains(view, "handover") {
		t.Errorf("View should NOT contain handover line for last step (no next step), got:\n%s", view)
	}
}

// Task 4.1: Verify completed step lines include token count
func TestProgressModel_View_CompletedStepShowsTokens(t *testing.T) {
	ctx := &PipelineContext{
		PipelineName:      "test-pipeline",
		TotalSteps:        2,
		CurrentStepNum:    2,
		OverallProgress:   50,
		ManifestPath:      "wave.yaml",
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
		StepStatuses: map[string]ProgressState{
			"analyst":     StateCompleted,
			"implementer": StateRunning,
		},
		StepOrder: []string{"analyst", "implementer"},
		StepPersonas: map[string]string{
			"analyst":     "analyst",
			"implementer": "implementer",
		},
		StepDurations: map[string]int64{
			"analyst": 45200,
		},
		StepStartTimes: map[string]int64{
			"implementer": time.Now().UnixNano(),
		},
		StepTokens: map[string]int{
			"analyst": 149100,
		},
		TotalTokens: 149100,
	}
	model := NewProgressModel(ctx)
	view := model.View()

	// Should contain formatted token count for completed step
	if !strings.Contains(view, "149.1k tokens") {
		t.Errorf("Completed step should show '149.1k tokens', got:\n%s", view)
	}

	// Should contain duration alongside tokens
	if !strings.Contains(view, "45.2s") {
		t.Errorf("Completed step should show duration '45.2s', got:\n%s", view)
	}
}

// Task 4.1: Verify total tokens in header
func TestProgressModel_View_HeaderShowsTotalTokens(t *testing.T) {
	ctx := &PipelineContext{
		PipelineName:      "test-pipeline",
		TotalSteps:        2,
		CurrentStepNum:    2,
		OverallProgress:   50,
		ManifestPath:      "wave.yaml",
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
		StepStatuses: map[string]ProgressState{
			"step1": StateCompleted,
			"step2": StateRunning,
		},
		StepOrder: []string{"step1", "step2"},
		StepDurations: map[string]int64{
			"step1": 30000,
		},
		StepStartTimes: map[string]int64{
			"step2": time.Now().UnixNano(),
		},
		StepTokens: map[string]int{
			"step1": 50000,
		},
		TotalTokens: 50000,
	}
	model := NewProgressModel(ctx)
	view := model.View()

	// Header should contain total tokens
	if !strings.Contains(view, "50.0k tokens") {
		t.Errorf("Header should show total tokens '50.0k tokens', got:\n%s", view)
	}
}

// Task 4.2: Verify zero-token graceful degradation
func TestProgressModel_View_ZeroTokensHidden(t *testing.T) {
	ctx := &PipelineContext{
		PipelineName:      "test-pipeline",
		TotalSteps:        2,
		CurrentStepNum:    2,
		OverallProgress:   50,
		ManifestPath:      "wave.yaml",
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
		StepStatuses: map[string]ProgressState{
			"step1": StateCompleted,
			"step2": StateRunning,
		},
		StepOrder: []string{"step1", "step2"},
		StepDurations: map[string]int64{
			"step1": 30000,
		},
		StepStartTimes: map[string]int64{
			"step2": time.Now().UnixNano(),
		},
		// No StepTokens or TotalTokens set (zero values)
	}
	model := NewProgressModel(ctx)
	view := model.View()

	// Should NOT contain "tokens" anywhere when no tokens are set
	if strings.Contains(view, "tokens") {
		t.Errorf("View should not contain 'tokens' when no token data is set, got:\n%s", view)
	}
}

// Task 4.2: Verify zero-token with explicit zero map
func TestProgressModel_View_ExplicitZeroTokensHidden(t *testing.T) {
	ctx := &PipelineContext{
		PipelineName:      "test-pipeline",
		TotalSteps:        2,
		CurrentStepNum:    2,
		OverallProgress:   50,
		ManifestPath:      "wave.yaml",
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
		StepStatuses: map[string]ProgressState{
			"step1": StateCompleted,
			"step2": StateRunning,
		},
		StepOrder: []string{"step1", "step2"},
		StepDurations: map[string]int64{
			"step1": 30000,
		},
		StepStartTimes: map[string]int64{
			"step2": time.Now().UnixNano(),
		},
		StepTokens:  map[string]int{"step1": 0},
		TotalTokens: 0,
	}
	model := NewProgressModel(ctx)
	view := model.View()

	// Should NOT contain "tokens" when values are explicitly zero
	if strings.Contains(view, "tokens") {
		t.Errorf("View should not contain 'tokens' when token values are 0, got:\n%s", view)
	}
}
