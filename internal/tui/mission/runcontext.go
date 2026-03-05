package mission

import (
	"time"

	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/event"
)

// RunContext bridges EventBus events to a display.PipelineContext for rendering.
// Each active run gets its own RunContext so the preview pane can render
// the exact same view as the single-run ProgressModel.
type RunContext struct {
	RunID    string
	Pipeline string
	Ctx      *display.PipelineContext
}

// NewRunContext creates a RunContext with pre-populated step order from the
// pipeline definition. Pending steps appear immediately, not only after events.
func NewRunContext(runID, pipeline string, stepOrder []string) *RunContext {
	statuses := make(map[string]display.ProgressState, len(stepOrder))
	personas := make(map[string]string, len(stepOrder))
	for _, s := range stepOrder {
		statuses[s] = display.StateNotStarted
	}

	return &RunContext{
		RunID:    runID,
		Pipeline: pipeline,
		Ctx: &display.PipelineContext{
			PipelineName:      pipeline,
			PipelineStartTime: time.Now().UnixNano(),
			CurrentStepStart:  time.Now().UnixNano(),
			TotalSteps:        len(stepOrder),
			StepOrder:         stepOrder,
			StepStatuses:      statuses,
			StepPersonas:      personas,
			StepDurations:     make(map[string]int64),
			StepTokens:        make(map[string]int),
			StepTokensIn:      make(map[string]int),
			StepTokensOut:     make(map[string]int),
			StepModels:        make(map[string]string),
			StepAdapters:      make(map[string]string),
			StepTemperatures:  make(map[string]float64),
			StepStartTimes:    make(map[string]int64),
			StepToolActivity:  make(map[string][2]string),
			DeliverablesByStep: make(map[string][]string),
			HandoversByStep:   make(map[string]*display.HandoverInfo),
		},
	}
}

// ApplyEvent translates an event.Event into PipelineContext mutations.
func (rc *RunContext) ApplyEvent(evt event.Event) {
	ctx := rc.Ctx

	if evt.PipelineID != "" && ctx.PipelineID == "" {
		ctx.PipelineID = evt.PipelineID
	}

	// Update totals
	if evt.TotalSteps > 0 {
		ctx.TotalSteps = evt.TotalSteps
	}
	if evt.CompletedSteps > 0 {
		ctx.CompletedSteps = evt.CompletedSteps
		ctx.CurrentStepNum = evt.CompletedSteps + 1
		if ctx.CurrentStepNum > ctx.TotalSteps {
			ctx.CurrentStepNum = ctx.TotalSteps
		}
	}
	if evt.Progress > 0 {
		ctx.OverallProgress = evt.Progress
	}
	if evt.TokensIn > 0 {
		ctx.TotalTokensIn = evt.TokensIn
	}
	if evt.TokensOut > 0 {
		ctx.TotalTokensOut = evt.TokensOut
	}
	if evt.TokensUsed > 0 {
		ctx.TotalTokens = evt.TokensUsed
	}

	stepID := evt.StepID
	if stepID == "" {
		// Pipeline-level event (no step)
		switch evt.State {
		case event.StateCompleted:
			ctx.OverallProgress = 100
		case event.StateFailed:
			ctx.Error = evt.Message
		}
		return
	}

	// Ensure step is in our maps (handles steps not in initial order)
	if _, exists := ctx.StepStatuses[stepID]; !exists {
		ctx.StepOrder = append(ctx.StepOrder, stepID)
		ctx.StepStatuses[stepID] = display.StateNotStarted
		ctx.TotalSteps = len(ctx.StepOrder)
	}

	// Step-level metadata
	if evt.Persona != "" {
		ctx.StepPersonas[stepID] = evt.Persona
	}
	if evt.Model != "" {
		ctx.StepModels[stepID] = evt.Model
	}
	if evt.Adapter != "" {
		ctx.StepAdapters[stepID] = evt.Adapter
	}
	if evt.Temperature > 0 {
		ctx.StepTemperatures[stepID] = evt.Temperature
	}

	// State transitions
	switch evt.State {
	case event.StateStarted, event.StateRunning:
		ctx.StepStatuses[stepID] = display.StateRunning
		ctx.CurrentStepID = stepID
		if _, exists := ctx.StepStartTimes[stepID]; !exists {
			ctx.StepStartTimes[stepID] = time.Now().UnixNano()
		}
		ctx.CurrentStepStart = ctx.StepStartTimes[stepID]
		if evt.Persona != "" {
			ctx.CurrentPersona = evt.Persona
		}

	case event.StateCompleted:
		ctx.StepStatuses[stepID] = display.StateCompleted
		if evt.DurationMs > 0 {
			ctx.StepDurations[stepID] = evt.DurationMs
		} else if startNano, ok := ctx.StepStartTimes[stepID]; ok {
			ctx.StepDurations[stepID] = time.Since(time.Unix(0, startNano)).Milliseconds()
		}
		// Recount completed and failed
		rc.recount()

	case event.StateFailed:
		ctx.StepStatuses[stepID] = display.StateFailed
		ctx.Error = evt.Message
		if startNano, ok := ctx.StepStartTimes[stepID]; ok {
			ctx.StepDurations[stepID] = time.Since(time.Unix(0, startNano)).Milliseconds()
		}
		rc.recount()

	case event.StateStreamActivity:
		// Tool activity updates
		if evt.ToolName != "" {
			ctx.StepToolActivity[stepID] = [2]string{evt.ToolName, evt.ToolTarget}
			ctx.LastToolName = evt.ToolName
			ctx.LastToolTarget = evt.ToolTarget
		}
	}

	// Per-step token updates
	if evt.TokensIn > 0 {
		ctx.StepTokensIn[stepID] = evt.TokensIn
	}
	if evt.TokensOut > 0 {
		ctx.StepTokensOut[stepID] = evt.TokensOut
	}
	if evt.TokensUsed > 0 {
		ctx.StepTokens[stepID] = evt.TokensUsed
	}

	// Current action
	if evt.CurrentAction != "" {
		ctx.CurrentAction = evt.CurrentAction
	}
}

// recount updates completed/failed step counts from the status map.
func (rc *RunContext) recount() {
	ctx := rc.Ctx
	completed := 0
	failed := 0
	for _, state := range ctx.StepStatuses {
		switch state {
		case display.StateCompleted:
			completed++
		case display.StateFailed:
			failed++
		}
	}
	ctx.CompletedSteps = completed
	ctx.FailedSteps = failed
	if ctx.TotalSteps > 0 {
		ctx.OverallProgress = completed * 100 / ctx.TotalSteps
	}
	ctx.CurrentStepNum = completed + 1
	if ctx.CurrentStepNum > ctx.TotalSteps {
		ctx.CurrentStepNum = ctx.TotalSteps
	}
}
