package display

import (
	"testing"
	"time"

	"github.com/recinq/wave/internal/event"
)

func newTestBubbleTeaDisplay() *BubbleTeaProgressDisplay {
	return &BubbleTeaProgressDisplay{
		steps:            make(map[string]*StepStatus),
		stepOrder:        make([]string, 0),
		stepDurations:    make(map[string]int64),
		stepTokens:       make(map[string]int),
		stepStartTimes:   make(map[string]time.Time),
		stepToolActivity: make(map[string][2]string),
		handoverInfo:     make(map[string]*HandoverInfo),
		stepModels:       make(map[string]string),
		stepAdapters:     make(map[string]string),
		stepTemperatures: make(map[string]float64),
		stepTokensIn:     make(map[string]int),
		stepTokensOut:    make(map[string]int),
		verbose:          true,
		enabled:          true,
	}
}

func TestUpdateFromEvent_StreamActivityGuard(t *testing.T) {
	tests := []struct {
		name         string
		setupSteps   func(d *BubbleTeaProgressDisplay)
		event        event.Event
		wantActivity bool   // should stepToolActivity have an entry after event
		wantToolName string // expected lastToolName after event
	}{
		{
			name: "stream_activity for completed step is dropped",
			setupSteps: func(d *BubbleTeaProgressDisplay) {
				d.steps["step-1"] = &StepStatus{
					StepID: "step-1",
					State:  StateCompleted,
				}
				d.stepOrder = []string{"step-1"}
			},
			event: event.Event{
				StepID:     "step-1",
				State:      "stream_activity",
				ToolName:   "Bash",
				ToolTarget: "git status",
				Timestamp:  time.Now(),
			},
			wantActivity: false,
			wantToolName: "",
		},
		{
			name: "stream_activity for not-started step is dropped",
			setupSteps: func(d *BubbleTeaProgressDisplay) {
				d.steps["step-1"] = &StepStatus{
					StepID: "step-1",
					State:  StateNotStarted,
				}
				d.stepOrder = []string{"step-1"}
			},
			event: event.Event{
				StepID:     "step-1",
				State:      "stream_activity",
				ToolName:   "Read",
				ToolTarget: "main.go",
				Timestamp:  time.Now(),
			},
			wantActivity: false,
			wantToolName: "",
		},
		{
			name: "stream_activity for running step is accepted",
			setupSteps: func(d *BubbleTeaProgressDisplay) {
				d.steps["step-1"] = &StepStatus{
					StepID: "step-1",
					State:  StateRunning,
				}
				d.stepOrder = []string{"step-1"}
				d.currentStepID = "step-1"
			},
			event: event.Event{
				StepID:     "step-1",
				State:      "stream_activity",
				ToolName:   "Write",
				ToolTarget: "output.go",
				Timestamp:  time.Now(),
			},
			wantActivity: true,
			wantToolName: "Write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestBubbleTeaDisplay()
			tt.setupSteps(d)

			d.updateFromEvent(tt.event)

			_, hasActivity := d.stepToolActivity[tt.event.StepID]
			if hasActivity != tt.wantActivity {
				t.Errorf("stepToolActivity[%q] exists = %v, want %v", tt.event.StepID, hasActivity, tt.wantActivity)
			}

			if d.lastToolName != tt.wantToolName {
				t.Errorf("lastToolName = %q, want %q", d.lastToolName, tt.wantToolName)
			}
		})
	}
}

func TestUpdateFromEvent_CompletionClearsStaleGlobalActivity(t *testing.T) {
	d := newTestBubbleTeaDisplay()

	// Set up a running step with global activity
	d.steps["step-1"] = &StepStatus{
		StepID: "step-1",
		State:  StateRunning,
	}
	d.stepOrder = []string{"step-1", "step-2"}
	d.steps["step-2"] = &StepStatus{
		StepID: "step-2",
		State:  StateNotStarted,
	}
	d.currentStepID = "step-1"
	d.lastToolName = "Bash"
	d.lastToolTarget = "go test ./..."
	d.stepToolActivity["step-1"] = [2]string{"Bash", "go test ./..."}

	// Complete step-1
	d.updateFromEvent(event.Event{
		StepID:    "step-1",
		State:     "completed",
		Timestamp: time.Now(),
	})

	// Global activity should be cleared
	if d.lastToolName != "" {
		t.Errorf("lastToolName should be cleared on completion, got %q", d.lastToolName)
	}
	if d.lastToolTarget != "" {
		t.Errorf("lastToolTarget should be cleared on completion, got %q", d.lastToolTarget)
	}

	// Per-step activity should be removed
	if _, exists := d.stepToolActivity["step-1"]; exists {
		t.Error("stepToolActivity[step-1] should be deleted after completion")
	}
}

func TestUpdateFromEvent_SyntheticCompletionMarksStepDone(t *testing.T) {
	d := newTestBubbleTeaDisplay()

	// Simulate the display registering all pipeline steps (as CreateEmitter does)
	d.AddStep("step-1", "step-1", "navigator")
	d.AddStep("step-2", "step-2", "auditor")
	d.AddStep("step-3", "step-3", "writer")

	// Verify initial state: all not started
	for _, sid := range []string{"step-1", "step-2", "step-3"} {
		if d.steps[sid].State != StateNotStarted {
			t.Fatalf("step %s should be not_started initially, got %s", sid, d.steps[sid].State)
		}
	}

	// Simulate synthetic completion events (as emitted by ResumeFromStep)
	d.updateFromEvent(event.Event{
		StepID:    "step-1",
		State:     "completed",
		Persona:   "navigator",
		Message:   "Completed in prior run",
		Timestamp: time.Now(),
	})
	d.updateFromEvent(event.Event{
		StepID:    "step-2",
		State:     "completed",
		Persona:   "auditor",
		Message:   "Completed in prior run",
		Timestamp: time.Now(),
	})

	// Verify step-1 and step-2 are now completed
	if d.steps["step-1"].State != StateCompleted {
		t.Errorf("step-1 should be completed, got %s", d.steps["step-1"].State)
	}
	if d.steps["step-2"].State != StateCompleted {
		t.Errorf("step-2 should be completed, got %s", d.steps["step-2"].State)
	}
	// step-3 should remain not started
	if d.steps["step-3"].State != StateNotStarted {
		t.Errorf("step-3 should be not_started, got %s", d.steps["step-3"].State)
	}

	// Verify pipeline context has correct counts
	ctx := d.toPipelineContext()
	if ctx.CompletedSteps != 2 {
		t.Errorf("expected 2 completed steps in context, got %d", ctx.CompletedSteps)
	}
}

func TestUpdateFromEvent_PersonaRefreshFromEvent(t *testing.T) {
	d := newTestBubbleTeaDisplay()

	// Register a step with an unresolved forge template persona
	d.AddStep("gather-context", "gather-context", "{{ forge.type }}-analyst")

	// Verify initial persona is the unresolved template
	if d.steps["gather-context"].Persona != "{{ forge.type }}-analyst" {
		t.Fatalf("initial persona = %q, want %q", d.steps["gather-context"].Persona, "{{ forge.type }}-analyst")
	}

	// Simulate a running event with the resolved persona (as the executor sends)
	d.updateFromEvent(event.Event{
		StepID:    "gather-context",
		State:     "running",
		Persona:   "github-analyst",
		Timestamp: time.Now(),
	})

	// Verify persona was updated to resolved value
	if d.steps["gather-context"].Persona != "github-analyst" {
		t.Errorf("persona after running event = %q, want %q", d.steps["gather-context"].Persona, "github-analyst")
	}

	// Verify toPipelineContext reflects the resolved persona
	ctx := d.toPipelineContext()
	if ctx.StepPersonas["gather-context"] != "github-analyst" {
		t.Errorf("StepPersonas[gather-context] = %q, want %q", ctx.StepPersonas["gather-context"], "github-analyst")
	}
}

func TestUpdateFromEvent_ModelAdapterTemperatureCapture(t *testing.T) {
	d := newTestBubbleTeaDisplay()

	d.AddStep("step-1", "step-1", "navigator")

	// Simulate a running event with model/adapter/temperature
	d.updateFromEvent(event.Event{
		StepID:      "step-1",
		State:       "running",
		Persona:     "navigator",
		Model:       "opus",
		Adapter:     "claude",
		Temperature: 0.7,
		Timestamp:   time.Now(),
	})

	// Verify model/adapter/temperature were captured
	if d.stepModels["step-1"] != "opus" {
		t.Errorf("stepModels[step-1] = %q, want %q", d.stepModels["step-1"], "opus")
	}
	if d.stepAdapters["step-1"] != "claude" {
		t.Errorf("stepAdapters[step-1] = %q, want %q", d.stepAdapters["step-1"], "claude")
	}
	if d.stepTemperatures["step-1"] != 0.7 {
		t.Errorf("stepTemperatures[step-1] = %f, want %f", d.stepTemperatures["step-1"], 0.7)
	}

	// Verify toPipelineContext includes the new fields
	ctx := d.toPipelineContext()
	if ctx.StepModels["step-1"] != "opus" {
		t.Errorf("PipelineContext.StepModels[step-1] = %q, want %q", ctx.StepModels["step-1"], "opus")
	}
	if ctx.StepAdapters["step-1"] != "claude" {
		t.Errorf("PipelineContext.StepAdapters[step-1] = %q, want %q", ctx.StepAdapters["step-1"], "claude")
	}
	if ctx.StepTemperatures["step-1"] != 0.7 {
		t.Errorf("PipelineContext.StepTemperatures[step-1] = %f, want %f", ctx.StepTemperatures["step-1"], 0.7)
	}
}

func TestUpdateFromEvent_TokensInOutCapture(t *testing.T) {
	d := newTestBubbleTeaDisplay()

	d.AddStep("step-1", "step-1", "navigator")
	d.AddStep("step-2", "step-2", "auditor")

	// Start step-1
	d.updateFromEvent(event.Event{
		StepID:    "step-1",
		State:     "running",
		Timestamp: time.Now(),
	})

	// Complete step-1 with token breakdown
	d.updateFromEvent(event.Event{
		StepID:     "step-1",
		State:      "completed",
		TokensUsed: 15000,
		TokensIn:   12000,
		TokensOut:  3000,
		DurationMs: 5000,
		Timestamp:  time.Now(),
	})

	// Start and complete step-2
	d.updateFromEvent(event.Event{
		StepID:    "step-2",
		State:     "running",
		Timestamp: time.Now(),
	})
	d.updateFromEvent(event.Event{
		StepID:     "step-2",
		State:      "completed",
		TokensUsed: 8000,
		TokensIn:   6000,
		TokensOut:  2000,
		DurationMs: 3000,
		Timestamp:  time.Now(),
	})

	// Verify per-step token breakdown
	if d.stepTokensIn["step-1"] != 12000 {
		t.Errorf("stepTokensIn[step-1] = %d, want %d", d.stepTokensIn["step-1"], 12000)
	}
	if d.stepTokensOut["step-1"] != 3000 {
		t.Errorf("stepTokensOut[step-1] = %d, want %d", d.stepTokensOut["step-1"], 3000)
	}

	// Verify toPipelineContext totals
	ctx := d.toPipelineContext()
	if ctx.StepTokensIn["step-1"] != 12000 {
		t.Errorf("PipelineContext.StepTokensIn[step-1] = %d, want %d", ctx.StepTokensIn["step-1"], 12000)
	}
	if ctx.StepTokensOut["step-2"] != 2000 {
		t.Errorf("PipelineContext.StepTokensOut[step-2] = %d, want %d", ctx.StepTokensOut["step-2"], 2000)
	}
	if ctx.TotalTokensIn != 18000 {
		t.Errorf("PipelineContext.TotalTokensIn = %d, want %d", ctx.TotalTokensIn, 18000)
	}
	if ctx.TotalTokensOut != 5000 {
		t.Errorf("PipelineContext.TotalTokensOut = %d, want %d", ctx.TotalTokensOut, 5000)
	}
}
