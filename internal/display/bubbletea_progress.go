package display

import (
	"context"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/event"
)

// BubbleTeaProgressDisplay implements ProgressEmitter using bubbletea for proper terminal handling.
type BubbleTeaProgressDisplay struct {
	mu           sync.Mutex
	program      *tea.Program
	model        *ProgressModel
	ctx          context.Context
	cancel       context.CancelFunc
	pipelineID   string
	pipelineName string
	totalSteps   int
	steps        map[string]*StepStatus
	stepOrder    []string
	startTime    time.Time
	enabled      bool
}

// NewBubbleTeaProgressDisplay creates a new bubbletea-based progress display.
func NewBubbleTeaProgressDisplay(pipelineID, pipelineName string, totalSteps int) *BubbleTeaProgressDisplay {
	termInfo := NewTerminalInfo()
	enabled := termInfo.IsTTY() && termInfo.SupportsANSI()

	if !enabled {
		return &BubbleTeaProgressDisplay{
			enabled: false,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initial pipeline context
	initialCtx := &PipelineContext{
		PipelineName:      pipelineName,
		OverallProgress:   0,
		CurrentStepNum:    1,
		TotalSteps:        totalSteps,
		CurrentStepID:     "",
		CurrentPersona:    "",
		CompletedSteps:    0,
		FailedSteps:       0,
		SkippedSteps:      0,
		StepStatuses:      make(map[string]ProgressState),
		ElapsedTimeMs:     0,
		EstimatedTimeMs:   0,
		ManifestPath:      "wave.yaml",
		WorkspacePath:     ".wave/workspaces",
		CurrentAction:     "",
		CurrentStepName:   "",
		PipelineStartTime: time.Now().UnixNano(),
		CurrentStepStart:  time.Now().UnixNano(),
	}

	model := NewProgressModel(initialCtx)

	display := &BubbleTeaProgressDisplay{
		ctx:          ctx,
		cancel:       cancel,
		pipelineID:   pipelineID,
		pipelineName: pipelineName,
		totalSteps:   totalSteps,
		steps:        make(map[string]*StepStatus),
		stepOrder:    make([]string, 0, totalSteps),
		startTime:    time.Now(),
		enabled:      true,
		model:        model,
	}

	// Create the bubbletea program (no alt screen to avoid terminal corruption)
	display.program = tea.NewProgram(model,
		tea.WithContext(ctx), // Cancellation context only
	)

	// Start the program in a goroutine
	go func() {
		if _, err := display.program.Run(); err != nil {
			// Handle error silently for now
		}
	}()

	return display
}

// EmitProgress implements the ProgressEmitter interface.
func (btpd *BubbleTeaProgressDisplay) EmitProgress(evt event.Event) error {
	if !btpd.enabled {
		return nil
	}

	btpd.mu.Lock()
	defer btpd.mu.Unlock()

	// Update internal state based on event
	btpd.updateFromEvent(evt)

	// Convert to PipelineContext and send update
	pipelineCtx := btpd.toPipelineContext()
	SendUpdate(btpd.program, pipelineCtx)

	return nil
}

// AddStep adds a step to track (implements same interface as ProgressDisplay).
func (btpd *BubbleTeaProgressDisplay) AddStep(stepID, stepName, persona string) {
	if !btpd.enabled {
		return
	}

	btpd.mu.Lock()
	defer btpd.mu.Unlock()

	if _, exists := btpd.steps[stepID]; !exists {
		btpd.steps[stepID] = &StepStatus{
			StepID:   stepID,
			Name:     stepName,
			Persona:  persona,
			State:    StateNotStarted,
			Progress: 0,
		}
		btpd.stepOrder = append(btpd.stepOrder, stepID)
	}
}

// Clear stops the bubbletea program.
func (btpd *BubbleTeaProgressDisplay) Clear() {
	if !btpd.enabled {
		return
	}

	btpd.cancel()
	if btpd.program != nil {
		btpd.program.Quit()
	}
}

// Finish stops the bubbletea program and shows completion.
func (btpd *BubbleTeaProgressDisplay) Finish() {
	if !btpd.enabled {
		return
	}

	// Mark current step as completed before finishing
	btpd.mu.Lock()
	for _, step := range btpd.steps {
		if step.State == StateRunning {
			step.State = StateCompleted
		}
	}

	// Send final update to show all steps completed
	pipelineCtx := btpd.toPipelineContext()
	SendUpdate(btpd.program, pipelineCtx)
	btpd.mu.Unlock()

	// Give a brief moment for the final render
	time.Sleep(100 * time.Millisecond)

	btpd.Clear()
}

// updateFromEvent updates internal state based on an event.
func (btpd *BubbleTeaProgressDisplay) updateFromEvent(evt event.Event) {
	if evt.StepID == "" {
		return
	}

	// Ensure step exists
	if _, exists := btpd.steps[evt.StepID]; !exists {
		btpd.AddStep(evt.StepID, evt.StepID, "")
	}

	step := btpd.steps[evt.StepID]

	// Update step state based on event
	switch evt.State {
	case "started", "running":
		step.State = StateRunning
	case "completed":
		step.State = StateCompleted
		step.Progress = 100
	case "failed":
		step.State = StateFailed
	case "skipped":
		step.State = StateSkipped
	case "retrying":
		step.State = StateRunning // Treat retrying as running
	}

	// Update progress if provided
	if evt.Progress > 0 {
		step.Progress = evt.Progress
	}
}

// toPipelineContext converts internal state to PipelineContext.
func (btpd *BubbleTeaProgressDisplay) toPipelineContext() *PipelineContext {
	elapsed := time.Since(btpd.startTime)
	elapsedMs := elapsed.Nanoseconds() / int64(time.Millisecond)

	// Calculate overall progress and counts
	completed := 0
	failed := 0
	skipped := 0
	currentStepIdx := 0
	currentStepID := ""
	currentPersona := ""

	for i, stepID := range btpd.stepOrder {
		if step, exists := btpd.steps[stepID]; exists {
			switch step.State {
			case StateCompleted:
				completed++
			case StateFailed:
				failed++
			case StateSkipped:
				skipped++
			case StateRunning:
				currentStepID = stepID
				currentPersona = step.Persona
				currentStepIdx = i
			}
		}
	}

	overallProgress := 0
	if btpd.totalSteps > 0 {
		overallProgress = (completed * 100) / btpd.totalSteps
	}

	// Convert step statuses
	stepStatuses := make(map[string]ProgressState)
	for stepID, step := range btpd.steps {
		stepStatuses[stepID] = step.State
	}

	return &PipelineContext{
		PipelineName:      btpd.pipelineName,
		OverallProgress:   overallProgress,
		CurrentStepNum:    currentStepIdx + 1, // 1-indexed for display
		TotalSteps:        btpd.totalSteps,
		CurrentStepID:     currentStepID,
		CurrentPersona:    currentPersona,
		CompletedSteps:    completed,
		FailedSteps:       failed,
		SkippedSteps:      skipped,
		StepStatuses:      stepStatuses,
		ElapsedTimeMs:     elapsedMs,
		EstimatedTimeMs:   0, // Not calculated
		ManifestPath:      "wave.yaml",
		WorkspacePath:     ".wave/workspaces",
		CurrentAction:     "",
		CurrentStepName:   currentStepID,
		PipelineStartTime: btpd.startTime.UnixNano(),
		CurrentStepStart:  btpd.startTime.UnixNano(), // Simplified
	}
}