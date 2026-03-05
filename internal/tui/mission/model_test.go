package mission

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/meta"
)

// --- Model creation & defaults ---

func TestNewMissionControlModel(t *testing.T) {
	model := NewMissionControlModel(Options{
		ManifestPath: "wave.yaml",
	}, nil)

	if model.activeView != ViewHealthPhase {
		t.Errorf("expected default view to be ViewHealthPhase, got %d", model.activeView)
	}
	if model.overlay != OverlayNone {
		t.Errorf("expected no overlay, got %d", model.overlay)
	}
	if model.eventBus == nil {
		t.Error("expected eventBus to be initialized")
	}
	if model.runManager == nil {
		t.Error("expected runManager to be initialized")
	}
	if model.healthCache == nil {
		t.Error("expected healthCache to be initialized")
	}
	if model.runContexts == nil {
		t.Error("expected runContexts map to be initialized")
	}
	if len(model.healthChecks) != 0 {
		t.Errorf("expected 0 health checks initially, got %d", len(model.healthChecks))
	}
}

// --- Run events ---

func TestRunEventUpdatesSnapshot(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet

	msg := RunEventMsg{RunEvent{
		RunID: "test-run-1",
		Event: event.Event{
			Timestamp:  time.Now(),
			PipelineID: "test-pipeline",
			State:      "started",
			TotalSteps: 3,
		},
	}}

	updated, _ := model.Update(msg)
	m := updated.(MissionControlModel)

	if len(m.runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(m.runs))
	}
	if m.runs[0].RunID != "test-run-1" {
		t.Errorf("expected run ID 'test-run-1', got %q", m.runs[0].RunID)
	}
	if m.runs[0].Status != "running" {
		t.Errorf("expected status 'running', got %q", m.runs[0].Status)
	}
	if m.runs[0].PipelineName != "test-pipeline" {
		t.Errorf("expected pipeline 'test-pipeline', got %q", m.runs[0].PipelineName)
	}
}

func TestRunEventCreatesRunContext(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40

	msg := RunEventMsg{RunEvent{
		RunID: "run-1",
		Event: event.Event{
			PipelineID: "my-pipeline",
			State:      "started",
		},
	}}

	updated, _ := model.Update(msg)
	m := updated.(MissionControlModel)

	rc, exists := m.runContexts["run-1"]
	if !exists {
		t.Fatal("expected RunContext to be created for run-1")
	}
	if rc.Pipeline != "my-pipeline" {
		t.Errorf("expected pipeline 'my-pipeline', got %q", rc.Pipeline)
	}
}

func TestRunEventStepUpdatesContext(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.applyRunEvent("run-1", event.Event{PipelineID: "p1", State: "started"})
	model.applyRunEvent("run-1", event.Event{StepID: "analyze", State: "started", Persona: "navigator", Model: "opus"})

	rc := model.runContexts["run-1"]
	if rc == nil {
		t.Fatal("expected RunContext")
	}

	status := rc.Ctx.StepStatuses["analyze"]
	if string(status) != "running" {
		t.Errorf("expected step status 'running', got %q", status)
	}
	if rc.Ctx.StepPersonas["analyze"] != "navigator" {
		t.Errorf("expected persona 'navigator', got %q", rc.Ctx.StepPersonas["analyze"])
	}
	if rc.Ctx.StepModels["analyze"] != "opus" {
		t.Errorf("expected model 'opus', got %q", rc.Ctx.StepModels["analyze"])
	}
}

// --- View rendering ---

func TestFleetViewRendersRunningPipeline(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet

	msg := RunEventMsg{RunEvent{
		RunID: "test-1",
		Event: event.Event{
			PipelineID: "test-pipeline",
			State:      "started",
		},
	}}
	updated, _ := model.Update(msg)
	m := updated.(MissionControlModel)

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(view, "test-pipeline") {
		t.Error("expected view to contain 'test-pipeline'")
	}
	if !strings.Contains(view, glyphRunning) {
		t.Error("expected view to contain running glyph")
	}
}

func TestFleetViewEmpty(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet

	view := model.View()
	if !strings.Contains(view, "No pipeline runs") {
		t.Error("expected empty state message")
	}
}

func TestFleetViewShowsHealthLoading(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet

	view := model.View()
	if !strings.Contains(view, "Loading") {
		t.Error("expected health loading indicator in fleet view")
	}
}

// --- Navigation ---

func TestFleetNavigation(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet

	model.applyRunEvent("run-1", event.Event{PipelineID: "pipeline-a", State: "started"})
	model.applyRunEvent("run-2", event.Event{PipelineID: "pipeline-b", State: "started"})

	if model.cursor != 0 {
		t.Errorf("expected initial cursor at 0, got %d", model.cursor)
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m := updated.(MissionControlModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after j, got %d", m.cursor)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(MissionControlModel)
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0 after k, got %d", m.cursor)
	}
}

// --- Attach / Detach ---

func TestEnterAttachesToRun(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet
	model.applyRunEvent("run-1", event.Event{PipelineID: "test-pipe", State: "started"})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := updated.(MissionControlModel)
	if m.activeView != ViewAttached {
		t.Errorf("expected ViewAttached after Enter, got %d", m.activeView)
	}
	if m.attachedRunID != "run-1" {
		t.Errorf("expected attachedRunID 'run-1', got %q", m.attachedRunID)
	}
}

func TestEscDetachesFromRun(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewAttached
	model.attachedRunID = "run-1"

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m := updated.(MissionControlModel)
	if m.activeView != ViewFleet {
		t.Errorf("expected ViewFleet after Esc from attached, got %d", m.activeView)
	}
	if m.attachedRunID != "" {
		t.Errorf("expected empty attachedRunID, got %q", m.attachedRunID)
	}
}

// --- Proposals view ---

func TestProposalsViewFromFleet(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet

	// Press p to switch to proposals view
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m := updated.(MissionControlModel)
	if m.activeView != ViewProposals {
		t.Errorf("expected ViewProposals after 'p', got %d", m.activeView)
	}
}

func TestTabTogglesBetweenProposalsAndFleet(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewProposals

	// Tab from proposals -> fleet
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	m := updated.(MissionControlModel)
	if m.activeView != ViewFleet {
		t.Errorf("expected ViewFleet after Tab from proposals, got %d", m.activeView)
	}

	// Tab from fleet -> proposals
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(MissionControlModel)
	if m.activeView != ViewProposals {
		t.Errorf("expected ViewProposals after Tab from fleet, got %d", m.activeView)
	}
}

func TestProposalNavigation(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewProposals
	model.proposals = []meta.PipelineProposal{
		{ID: "p1", Type: meta.ProposalSingle, Pipelines: []string{"gh-implement"}},
		{ID: "p2", Type: meta.ProposalSingle, Pipelines: []string{"gh-pr-review"}},
	}

	// Navigate down
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m := updated.(MissionControlModel)
	if m.proposalCursor != 1 {
		t.Errorf("expected proposalCursor=1, got %d", m.proposalCursor)
	}

	// Navigate up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(MissionControlModel)
	if m.proposalCursor != 0 {
		t.Errorf("expected proposalCursor=0, got %d", m.proposalCursor)
	}
}

func TestProposalMultiSelect(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewProposals
	model.proposals = []meta.PipelineProposal{
		{ID: "p1", Type: meta.ProposalSingle, Pipelines: []string{"gh-implement"}},
		{ID: "p2", Type: meta.ProposalSingle, Pipelines: []string{"gh-pr-review"}},
	}

	// Space to toggle select
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m := updated.(MissionControlModel)
	if !m.proposalSelect[0] {
		t.Error("expected proposal 0 to be selected after Space")
	}

	// Space again to deselect
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = updated.(MissionControlModel)
	if m.proposalSelect[0] {
		t.Error("expected proposal 0 to be deselected after second Space")
	}
}

func TestProposalSkip(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewProposals
	model.proposals = []meta.PipelineProposal{
		{ID: "p1", Type: meta.ProposalSingle, Pipelines: []string{"gh-implement"}},
		{ID: "p2", Type: meta.ProposalSingle, Pipelines: []string{"gh-pr-review"}},
	}

	// Skip proposal
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m := updated.(MissionControlModel)
	if !m.proposalSkipped[0] {
		t.Error("expected proposal 0 to be skipped")
	}
}

func TestProposalSkipAllTransitionsToFleet(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewProposals
	model.proposals = []meta.PipelineProposal{
		{ID: "p1", Type: meta.ProposalSingle, Pipelines: []string{"gh-implement"}},
	}

	// Skip the only proposal
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m := updated.(MissionControlModel)
	if m.activeView != ViewFleet {
		t.Errorf("expected ViewFleet when all proposals skipped, got %d", m.activeView)
	}
}

// --- Health overlay ---

func TestHealthOverlayFromFleet(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m := updated.(MissionControlModel)
	if m.overlay != OverlayHealth {
		t.Errorf("expected OverlayHealth, got %d", m.overlay)
	}

	// Scrolling in overlay
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(MissionControlModel)
	if m.healthScrollOff != 1 {
		t.Errorf("expected healthScrollOff=1, got %d", m.healthScrollOff)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(MissionControlModel)
	if m.healthScrollOff != 0 {
		t.Errorf("expected healthScrollOff=0, got %d", m.healthScrollOff)
	}

	// Close
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(MissionControlModel)
	if m.overlay != OverlayNone {
		t.Errorf("expected OverlayNone, got %d", m.overlay)
	}
}

// --- Health auto-transition ---

func TestHealthAutoTransitionsToProposals(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40

	if model.activeView != ViewHealthPhase {
		t.Fatalf("expected ViewHealthPhase, got %d", model.activeView)
	}

	// Simulate health check completion
	msg := HealthCacheMsg{
		Report: &meta.HealthReport{
			Init: meta.InitCheckResult{ManifestFound: true, WaveVersion: "dev"},
		},
	}
	updated, _ := model.Update(msg)
	m := updated.(MissionControlModel)

	// Should auto-transition to proposals view
	if m.activeView != ViewProposals {
		t.Errorf("expected ViewProposals after health completes from health phase, got %d", m.activeView)
	}
	if !m.healthLoaded {
		t.Error("expected healthLoaded=true")
	}
	if len(m.healthChecks) < 4 {
		t.Errorf("expected at least 4 health checks populated, got %d", len(m.healthChecks))
	}
}

func TestHealthNoTransitionWhenAlreadyOnFleet(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet // User already tabbed to fleet

	msg := HealthCacheMsg{
		Report: &meta.HealthReport{
			Init: meta.InitCheckResult{ManifestFound: true, WaveVersion: "dev"},
		},
	}
	updated, _ := model.Update(msg)
	m := updated.(MissionControlModel)

	// Should stay on fleet — no forced transition
	if m.activeView != ViewFleet {
		t.Errorf("expected ViewFleet (no transition when already on fleet), got %d", m.activeView)
	}
}

func TestHealthPhaseAutoTransition(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40

	// Starts on health phase
	if model.activeView != ViewHealthPhase {
		t.Fatalf("expected ViewHealthPhase initially, got %d", model.activeView)
	}

	// Health view renders while loading
	view := model.View()
	if !strings.Contains(view, "health") || !strings.Contains(view, "WAVE") {
		t.Error("expected health phase to render WAVE brand and health reference")
	}
}

// --- Health phase only allows quit ---

func TestHealthPhaseOnlyAllowsQuit(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	// Starts on health phase

	// Random keys should not change view
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m := updated.(MissionControlModel)
	if m.activeView != ViewHealthPhase {
		t.Errorf("expected ViewHealthPhase after j, got %d", m.activeView)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(MissionControlModel)
	if m.activeView != ViewHealthPhase {
		t.Errorf("expected ViewHealthPhase after n, got %d", m.activeView)
	}

	// Tab allows skipping to fleet
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(MissionControlModel)
	if m.activeView != ViewFleet {
		t.Errorf("expected ViewFleet after Tab from health phase, got %d", m.activeView)
	}
}

// --- Embedded form blocks keys ---

func TestEmbeddedFormBlocksKeys(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet

	// Simulate having an active form
	model.activeForm = &mockHuhForm
	model.formKind = "pipeline-select"
	model.overlay = OverlayForm

	// 'q' should NOT quit when form is active (it should go to form)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m := updated.(MissionControlModel)
	if m.quitting {
		t.Error("expected q to not quit when form is active")
	}
	// But ctrl+c should quit
	_ = cmd
}

// mockHuhForm is a minimal huh.Form for testing form-active detection.
// We just need a non-nil pointer; the form's Update won't be called in tests
// that only check whether keys are blocked.
var mockHuhForm = *newMockForm()

func newMockForm() *huh.Form {
	var s string
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("test").Value(&s),
		),
	)
}

// --- Filter mode ---

func TestFilterMode(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet

	model.applyRunEvent("run-1", event.Event{PipelineID: "alpha-pipeline", State: "started"})
	model.applyRunEvent("run-2", event.Event{PipelineID: "beta-pipeline", State: "started"})

	// Enter filter mode
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m := updated.(MissionControlModel)
	if !m.filterMode {
		t.Error("expected filter mode to be active")
	}

	// Type filter characters
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(MissionControlModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(MissionControlModel)

	visible := m.visibleRuns()
	if len(visible) != 1 {
		t.Fatalf("expected 1 visible run with filter 'al', got %d", len(visible))
	}
	if visible[0].PipelineName != "alpha-pipeline" {
		t.Errorf("expected 'alpha-pipeline', got %q", visible[0].PipelineName)
	}

	// Confirm filter
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\n'}})
	m = updated.(MissionControlModel)

	// Clear filter with Esc
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(MissionControlModel)
	if m.filter != "" {
		t.Errorf("expected empty filter after Esc, got %q", m.filter)
	}
}

// --- EventBus ---

func TestEventBus(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	sent := bus.Send(RunEvent{
		RunID: "test-1",
		Event: event.Event{State: "started"},
	})
	if !sent {
		t.Error("expected Send to return true")
	}

	select {
	case evt := <-bus.ch:
		if evt.RunID != "test-1" {
			t.Errorf("expected run ID 'test-1', got %q", evt.RunID)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for event")
	}
}

func TestEventBusClose(t *testing.T) {
	bus := NewEventBus()
	bus.Close()

	sent := bus.Send(RunEvent{RunID: "test"})
	if sent {
		t.Error("expected Send to return false after Close")
	}

	bus.Close() // double close should not panic
}

func TestBusEmitter(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	emitter := NewBusEmitter(bus, "run-42")
	err := emitter.EmitProgress(event.Event{State: "started", PipelineID: "test"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	select {
	case evt := <-bus.ch:
		if evt.RunID != "run-42" {
			t.Errorf("expected run ID 'run-42', got %q", evt.RunID)
		}
		if evt.Event.State != "started" {
			t.Errorf("expected state 'started', got %q", evt.Event.State)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for event")
	}
}

// --- Utility functions ---

func TestFleetStatsComputation(t *testing.T) {
	runs := []RunSnapshot{
		{Status: "running"},
		{Status: "running"},
		{Status: "completed"},
		{Status: "failed"},
	}
	stats := computeFleetStats(runs)
	if stats.running != 2 {
		t.Errorf("expected 2 running, got %d", stats.running)
	}
	if stats.completed != 1 {
		t.Errorf("expected 1 completed, got %d", stats.completed)
	}
	if stats.failed != 1 {
		t.Errorf("expected 1 failed, got %d", stats.failed)
	}
}

func TestStatusGlyph(t *testing.T) {
	tests := []struct {
		status        string
		expectedGlyph string
	}{
		{"running", glyphRunning},
		{"completed", glyphCompleted},
		{"failed", glyphFailed},
		{"cancelled", glyphCancelled},
		{"queued", glyphQueued},
		{"pending", glyphQueued},
		{"stale", glyphStale},
	}

	for _, tt := range tests {
		glyph, _ := statusGlyph(tt.status)
		if glyph != tt.expectedGlyph {
			t.Errorf("statusGlyph(%q) = %q, want %q", tt.status, glyph, tt.expectedGlyph)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{2 * time.Minute, "2m"},
		{5*time.Minute + 15*time.Second, "5m15s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		pct  int
		want string
	}{
		{0, "░░░░░░░░░░  0%"},
		{50, "█████░░░░░  50%"},
		{100, "██████████ 100%"},
	}

	for _, tt := range tests {
		got := renderProgressBar(tt.pct, 10)
		if got != tt.want {
			t.Errorf("renderProgressBar(%d, 10) = %q, want %q", tt.pct, got, tt.want)
		}
	}
}

// --- Store merge ---

func TestMergeFromStore(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.applyRunEvent("local-1", event.Event{PipelineID: "pipeline-a", State: "started"})

	records := []storeRecord{
		{RunID: "local-1", PipelineName: "pipeline-a", Status: "running"},
		{RunID: "external-1", PipelineName: "pipeline-b", Status: "completed", StartedAt: time.Now().Add(-5 * time.Minute)},
	}
	model.mergeFromStore(records)

	if len(model.runs) != 2 {
		t.Fatalf("expected 2 runs after merge, got %d", len(model.runs))
	}

	found := false
	for _, r := range model.runs {
		if r.RunID == "external-1" {
			found = true
			if r.Local {
				t.Error("expected external run to have Local=false")
			}
			if r.Status != "completed" {
				t.Errorf("expected status 'completed', got %q", r.Status)
			}
		}
	}
	if !found {
		t.Error("expected external-1 to be in runs")
	}
}

// --- Stale run detection ---

func TestStaleRunDetection(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)

	// Add a "running" run from the store that started 2 hours ago (non-local)
	records := []storeRecord{
		{
			RunID:        "old-run",
			PipelineName: "stale-pipeline",
			Status:       "running",
			StartedAt:    time.Now().Add(-2 * time.Hour),
		},
	}
	model.mergeFromStore(records)

	// Find the run and check it's marked stale
	found := false
	for _, r := range model.runs {
		if r.RunID == "old-run" {
			found = true
			if r.Status != "stale" {
				t.Errorf("expected stale status for old non-local running run, got %q", r.Status)
			}
		}
	}
	if !found {
		t.Error("expected old-run to be in runs")
	}
}

func TestRecentRunNotMarkedStale(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)

	// Add a "running" run from the store that started 5 minutes ago (non-local)
	records := []storeRecord{
		{
			RunID:        "recent-run",
			PipelineName: "recent-pipeline",
			Status:       "running",
			StartedAt:    time.Now().Add(-5 * time.Minute),
		},
	}
	model.mergeFromStore(records)

	for _, r := range model.runs {
		if r.RunID == "recent-run" {
			if r.Status != "running" {
				t.Errorf("expected recent run to still be 'running', got %q", r.Status)
			}
		}
	}
}

// --- RunContext ---

func TestRunContextApplyEvent(t *testing.T) {
	rc := NewRunContext("run-1", "test-pipe", []string{"analyze", "implement", "review"})

	rc.ApplyEvent(event.Event{
		StepID:  "analyze",
		State:   "started",
		Persona: "navigator",
		Model:   "opus",
	})

	if string(rc.Ctx.StepStatuses["analyze"]) != "running" {
		t.Errorf("expected analyze=running, got %q", rc.Ctx.StepStatuses["analyze"])
	}
	if rc.Ctx.StepPersonas["analyze"] != "navigator" {
		t.Errorf("expected persona=navigator, got %q", rc.Ctx.StepPersonas["analyze"])
	}
	if rc.Ctx.CurrentStepID != "analyze" {
		t.Errorf("expected currentStepID=analyze, got %q", rc.Ctx.CurrentStepID)
	}

	rc.ApplyEvent(event.Event{
		StepID:     "analyze",
		State:      "completed",
		DurationMs: 5000,
	})

	if string(rc.Ctx.StepStatuses["analyze"]) != "completed" {
		t.Errorf("expected analyze=completed, got %q", rc.Ctx.StepStatuses["analyze"])
	}
	if rc.Ctx.StepDurations["analyze"] != 5000 {
		t.Errorf("expected duration=5000, got %d", rc.Ctx.StepDurations["analyze"])
	}
	if rc.Ctx.CompletedSteps != 1 {
		t.Errorf("expected completedSteps=1, got %d", rc.Ctx.CompletedSteps)
	}
	if rc.Ctx.OverallProgress != 33 {
		t.Errorf("expected progress=33, got %d", rc.Ctx.OverallProgress)
	}
}

func TestRunContextToolActivity(t *testing.T) {
	rc := NewRunContext("run-1", "test-pipe", []string{"implement"})
	rc.ApplyEvent(event.Event{StepID: "implement", State: "started"})
	rc.ApplyEvent(event.Event{
		StepID:     "implement",
		State:      "stream_activity",
		ToolName:   "Write",
		ToolTarget: "src/main.go",
	})

	ta, ok := rc.Ctx.StepToolActivity["implement"]
	if !ok {
		t.Fatal("expected tool activity for implement")
	}
	if ta[0] != "Write" || ta[1] != "src/main.go" {
		t.Errorf("expected [Write, src/main.go], got %v", ta)
	}
}

// --- Sort ---

func TestRunsSortByDate(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	now := time.Now()

	model.runs = []RunSnapshot{
		{RunID: "old", StartedAt: now.Add(-10 * time.Minute), Status: "running"},
		{RunID: "new", StartedAt: now.Add(-1 * time.Minute), Status: "running"},
		{RunID: "mid", StartedAt: now.Add(-5 * time.Minute), Status: "running"},
	}
	model.sortRuns()

	if model.runs[0].RunID != "new" {
		t.Errorf("expected newest first, got %q", model.runs[0].RunID)
	}
	if model.runs[1].RunID != "mid" {
		t.Errorf("expected middle second, got %q", model.runs[1].RunID)
	}
	if model.runs[2].RunID != "old" {
		t.Errorf("expected oldest last, got %q", model.runs[2].RunID)
	}
}

// --- Overlay blocks fleet keys ---

func TestOverlayBlocksFleetKeys(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet
	model.overlay = OverlayHealth

	// 'n' should not open pipeline selector when overlay is active
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := updated.(MissionControlModel)
	if m.overlay != OverlayHealth {
		t.Error("overlay should still be active")
	}
	if cmd != nil {
		t.Error("should not launch pipeline selector when overlay is active")
	}
}

// --- Attached view ---

func TestAttachedViewShowsPipelineView(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.applyRunEvent("run-1", event.Event{PipelineID: "test-pipe", State: "started"})
	model.applyRunEvent("run-1", event.Event{StepID: "analyze", State: "started", Persona: "navigator"})
	model.activeView = ViewAttached
	model.attachedRunID = "run-1"

	view := model.View()
	if !strings.Contains(view, "analyze") {
		t.Error("attached view should contain step 'analyze'")
	}
}

// --- Health cache ---

func TestHealthCacheNoData(t *testing.T) {
	cache := NewHealthCache("wave.yaml", "dev")
	report := cache.Report()
	if report != nil {
		t.Error("expected nil report initially")
	}
}

// --- Compute elapsed ---

func TestComputeElapsed(t *testing.T) {
	now := time.Now()
	completed := now.Add(-2 * time.Minute)

	tests := []struct {
		name     string
		rec      storeRecord
		wantZero bool
	}{
		{
			name:     "empty start",
			rec:      storeRecord{},
			wantZero: true,
		},
		{
			name: "completed with end time",
			rec: storeRecord{
				Status:      "completed",
				StartedAt:   now.Add(-5 * time.Minute),
				CompletedAt: &completed,
			},
			wantZero: false,
		},
		{
			name: "running",
			rec: storeRecord{
				Status:    "running",
				StartedAt: now.Add(-1 * time.Minute),
			},
			wantZero: false,
		},
	}

	for _, tt := range tests {
		elapsed := computeElapsed(tt.rec)
		if tt.wantZero && elapsed != 0 {
			t.Errorf("%s: expected 0 elapsed, got %v", tt.name, elapsed)
		}
		if !tt.wantZero && elapsed == 0 {
			t.Errorf("%s: expected non-zero elapsed", tt.name)
		}
	}
}

// --- Stale detection for queued/pending ---

func TestStaleQueuedRunDetection(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)

	records := []storeRecord{
		{
			RunID:        "old-queued",
			PipelineName: "stale-queued-pipeline",
			Status:       "queued",
			StartedAt:    time.Now().Add(-2 * time.Hour),
		},
		{
			RunID:        "old-pending",
			PipelineName: "stale-pending-pipeline",
			Status:       "pending",
			StartedAt:    time.Now().Add(-2 * time.Hour),
		},
	}
	model.mergeFromStore(records)

	for _, r := range model.runs {
		if r.RunID == "old-queued" && r.Status != "stale" {
			t.Errorf("expected stale for old queued run, got %q", r.Status)
		}
		if r.RunID == "old-pending" && r.Status != "stale" {
			t.Errorf("expected stale for old pending run, got %q", r.Status)
		}
	}
}

// --- ctrl+c in overlay quits ---

func TestCtrlCInOverlayQuits(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet
	model.overlay = OverlayHealth

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m := updated.(MissionControlModel)
	if !m.quitting {
		t.Error("expected quitting=true after ctrl+c in overlay")
	}
	if cmd == nil {
		t.Error("expected tea.Quit command")
	}
}

func TestCtrlCInFilterQuits(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet
	model.filterMode = true

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m := updated.(MissionControlModel)
	if !m.quitting {
		t.Error("expected quitting=true after ctrl+c in filter mode")
	}
	if cmd == nil {
		t.Error("expected tea.Quit command")
	}
}

// --- q in overlay quits ---

func TestQInOverlayQuits(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet
	model.overlay = OverlayHealth

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m := updated.(MissionControlModel)
	if !m.quitting {
		t.Error("expected quitting=true after q in overlay")
	}
	if cmd == nil {
		t.Error("expected tea.Quit command")
	}
}

// --- Filter cursor reset ---

func TestFilterCursorResetOnTextChange(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet
	model.applyRunEvent("run-1", event.Event{PipelineID: "alpha", State: "started"})
	model.applyRunEvent("run-2", event.Event{PipelineID: "beta", State: "started"})
	model.cursor = 1

	// Enter filter mode and type
	model.filterMode = true
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m := updated.(MissionControlModel)
	if m.cursor != 0 {
		t.Errorf("expected cursor reset to 0 after filter text change, got %d", m.cursor)
	}
}

// --- Help overlay ---

func TestHelpOverlayFromFleet(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m := updated.(MissionControlModel)
	if m.overlay != OverlayHelp {
		t.Errorf("expected OverlayHelp after ?, got %d", m.overlay)
	}
}

func TestHelpOverlayFromAttached(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewAttached

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m := updated.(MissionControlModel)
	if m.overlay != OverlayHelp {
		t.Errorf("expected OverlayHelp after ? in attached view, got %d", m.overlay)
	}
}

func TestHelpOverlayRendersKeybindings(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet
	model.overlay = OverlayHelp

	view := model.View()
	if !strings.Contains(view, "Keybindings") {
		t.Error("expected help overlay to contain 'Keybindings'")
	}
	if !strings.Contains(view, "Fleet View") {
		t.Error("expected help overlay to contain 'Fleet View' section")
	}
}

// --- Health scroll reset on refresh ---

func TestHealthScrollResetOnRefresh(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet
	model.overlay = OverlayHealth
	model.healthScrollOff = 5

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	m := updated.(MissionControlModel)
	if m.healthScrollOff != 0 {
		t.Errorf("expected healthScrollOff reset to 0 on R, got %d", m.healthScrollOff)
	}
}

// --- Stale preview pane ---

func TestStaleRunPreviewPaneContent(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet

	// Add a stale run
	model.runs = append(model.runs, RunSnapshot{
		RunID:        "stale-1",
		PipelineName: "stale-pipeline",
		Status:       "stale",
		StartedAt:    time.Now().Add(-2 * time.Hour),
	})

	view := model.View()
	if !strings.Contains(view, "stale") {
		t.Error("expected stale run preview to mention stale")
	}
}

func TestArchivedRunPreviewFallback(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet

	// Add a completed run without step data in RunContext
	model.runs = append(model.runs, RunSnapshot{
		RunID:        "done-1",
		PipelineName: "test-pipeline",
		Status:       "completed",
		CurrentStep:  "final-step",
		StartedAt:    time.Now().Add(-1 * time.Hour),
		Elapsed:      45 * time.Minute,
	})
	// Create a RunContext with empty StepOrder (simulates store-loaded run without event data)
	model.runContexts["done-1"] = NewRunContext("done-1", "test-pipeline", nil)

	view := model.View()
	if strings.Contains(view, "Waiting for pipeline to start") {
		t.Error("completed run without step data should not show 'Waiting for pipeline to start'")
	}
	if !strings.Contains(view, "completed") {
		t.Error("expected fallback to show status 'completed'")
	}
}

func TestFailedRunPreviewShowsError(t *testing.T) {
	model := NewMissionControlModel(Options{ManifestPath: "wave.yaml"}, nil)
	model.width = 120
	model.height = 40
	model.activeView = ViewFleet

	model.runs = append(model.runs, RunSnapshot{
		RunID:        "fail-1",
		PipelineName: "broken-pipeline",
		Status:       "failed",
		ErrorMessage: "step timeout exceeded",
		StartedAt:    time.Now().Add(-30 * time.Minute),
		Elapsed:      20 * time.Minute,
	})

	view := model.View()
	if strings.Contains(view, "Waiting for pipeline to start") {
		t.Error("failed run should not show 'Waiting for pipeline to start'")
	}
	if !strings.Contains(view, "step timeout exceeded") {
		t.Error("expected error message in preview pane")
	}
}
