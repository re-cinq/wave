package display

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/deliverable"
	"github.com/recinq/wave/internal/event"
)

// BubbleTeaProgressDisplay implements ProgressEmitter using bubbletea for proper terminal handling.
type BubbleTeaProgressDisplay struct {
	mu                 sync.Mutex
	program            *tea.Program
	model              *ProgressModel
	ctx                context.Context
	cancel             context.CancelFunc
	pipelineID         string
	pipelineName       string
	totalSteps         int
	steps              map[string]*StepStatus
	stepOrder          []string
	stepDurations      map[string]int64      // Track step durations in milliseconds
	stepStartTimes     map[string]time.Time  // Track when each step started
	startTime          time.Time
	enabled            bool
	verbose            bool
	deliverableTracker *deliverable.Tracker
	currentStepID      string                   // Track primary running step (first in order)
	stepToolActivity   map[string][2]string      // stepID -> [toolName, toolTarget] per-step
	lastToolName       string                   // Most recent tool name (global fallback)
	lastToolTarget     string                   // Most recent tool target (global fallback)
	handoverInfo       map[string]*HandoverInfo // Per-step handover metadata
}

// NewBubbleTeaProgressDisplay creates a new bubbletea-based progress display.
func NewBubbleTeaProgressDisplay(pipelineID, pipelineName string, totalSteps int, tracker *deliverable.Tracker, verbose ...bool) *BubbleTeaProgressDisplay {
	termInfo := NewTerminalInfo()
	enabled := termInfo.IsTTY() && termInfo.SupportsANSI()

	isVerbose := len(verbose) > 0 && verbose[0]

	if !enabled {
		return &BubbleTeaProgressDisplay{
			enabled: false,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initial pipeline context
	initialCtx := &PipelineContext{
		PipelineName:      pipelineName,
		PipelineID:        pipelineID,
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
		ctx:                ctx,
		cancel:             cancel,
		pipelineID:         pipelineID,
		pipelineName:       pipelineName,
		totalSteps:         totalSteps,
		steps:              make(map[string]*StepStatus),
		stepOrder:          make([]string, 0, totalSteps),
		stepDurations:      make(map[string]int64),
		stepStartTimes:     make(map[string]time.Time),
		stepToolActivity:   make(map[string][2]string),
		handoverInfo:       make(map[string]*HandoverInfo),
		startTime:          time.Now(),
		enabled:            true,
		verbose:            isVerbose,
		model:              model,
		deliverableTracker: tracker,
		currentStepID:      "",
	}

	// Create the bubbletea program (no alt screen to avoid terminal corruption)
	display.program = tea.NewProgram(model,
		tea.WithContext(ctx),    // Cancellation context only
		tea.WithInput(os.Stdin), // Enable keyboard input
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

	btpd.addStepLocked(stepID, stepName, persona)
}

// addStepLocked creates a step entry without acquiring the mutex.
// The caller must already hold btpd.mu.
func (btpd *BubbleTeaProgressDisplay) addStepLocked(stepID, stepName, persona string) {
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

// Clear stops the bubbletea program and resets terminal state.
func (btpd *BubbleTeaProgressDisplay) Clear() {
	if !btpd.enabled {
		return
	}

	btpd.cancel()
	if btpd.program != nil {
		btpd.program.Quit()
		btpd.program.Wait() // Wait for program to fully exit
	}

	// Simple cleanup - just ensure cursor is visible
	fmt.Print("\033[?25h") // Show cursor
	fmt.Print("\033[0m")   // Reset formatting
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

// SetDeliverableTracker sets the deliverable tracker after construction
func (btpd *BubbleTeaProgressDisplay) SetDeliverableTracker(tracker *deliverable.Tracker) {
	if !btpd.enabled {
		return
	}
	btpd.mu.Lock()
	defer btpd.mu.Unlock()
	btpd.deliverableTracker = tracker
}

// updateFromEvent updates internal state based on an event.
func (btpd *BubbleTeaProgressDisplay) updateFromEvent(evt event.Event) {
	if evt.StepID == "" {
		return
	}

	// Ensure step exists
	if _, exists := btpd.steps[evt.StepID]; !exists {
		btpd.addStepLocked(evt.StepID, evt.StepID, "")
	}

	step := btpd.steps[evt.StepID]

	// Update step state based on event
	switch evt.State {
	case "started", "running":
		// Track when step starts if it wasn't already running
		if step.State != StateRunning {
			btpd.stepStartTimes[evt.StepID] = time.Now()
		}
		step.State = StateRunning
	case "completed":
		step.State = StateCompleted
		step.Progress = 100
		// Clean up per-step tool activity and clear stale global activity
		// to prevent leakage to the next step in shared-worktree pipelines.
		delete(btpd.stepToolActivity, evt.StepID)
		if btpd.currentStepID == evt.StepID {
			btpd.lastToolName = ""
			btpd.lastToolTarget = ""
		}
		// Capture step duration for display
		if evt.DurationMs > 0 {
			btpd.stepDurations[evt.StepID] = evt.DurationMs
		}
	case "failed":
		step.State = StateFailed
		delete(btpd.stepToolActivity, evt.StepID)
	case "skipped":
		step.State = StateSkipped
		delete(btpd.stepToolActivity, evt.StepID)
	case "retrying":
		step.State = StateRunning // Treat retrying as running
	case "warning":
		step.Message = evt.Message
	}

	// Update progress if provided
	if evt.Progress > 0 {
		step.Progress = evt.Progress
	}

	// Capture tool activity for verbose mode (per-step).
	// Guard: drop stream_activity for steps that are already completed or not yet started.
	// This prevents phantom activity from shared-worktree steps leaking to wrong steps.
	if btpd.verbose && evt.State == "stream_activity" && evt.ToolName != "" {
		if step.State == StateRunning {
			btpd.stepToolActivity[evt.StepID] = [2]string{evt.ToolName, evt.ToolTarget}
			btpd.lastToolName = evt.ToolName
			btpd.lastToolTarget = evt.ToolTarget
		}
	}

	// Capture handover metadata for verbose mode
	switch evt.State {
	case "validating":
		if _, exists := btpd.handoverInfo[evt.StepID]; !exists {
			btpd.handoverInfo[evt.StepID] = &HandoverInfo{}
		}
		btpd.handoverInfo[evt.StepID].ContractSchema = evt.ValidationPhase
	case "contract_passed":
		if _, exists := btpd.handoverInfo[evt.StepID]; !exists {
			btpd.handoverInfo[evt.StepID] = &HandoverInfo{}
		}
		btpd.handoverInfo[evt.StepID].ContractStatus = "passed"
	case "contract_failed":
		if _, exists := btpd.handoverInfo[evt.StepID]; !exists {
			btpd.handoverInfo[evt.StepID] = &HandoverInfo{}
		}
		btpd.handoverInfo[evt.StepID].ContractStatus = "failed"
	case "contract_soft_failure":
		if _, exists := btpd.handoverInfo[evt.StepID]; !exists {
			btpd.handoverInfo[evt.StepID] = &HandoverInfo{}
		}
		btpd.handoverInfo[evt.StepID].ContractStatus = "soft_failure"
	case "completed":
		if len(evt.Artifacts) > 0 {
			if _, exists := btpd.handoverInfo[evt.StepID]; !exists {
				btpd.handoverInfo[evt.StepID] = &HandoverInfo{}
			}
			btpd.handoverInfo[evt.StepID].ArtifactPaths = evt.Artifacts
		}
	}

	// Recompute currentStepID: first running step in order
	btpd.currentStepID = ""
	for _, sid := range btpd.stepOrder {
		if s, exists := btpd.steps[sid]; exists && s.State == StateRunning {
			btpd.currentStepID = sid
			break
		}
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

	running := 0
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
				running++
				if currentStepID == "" {
					// First running step is the "primary" for backward compat
					currentStepID = stepID
					currentPersona = step.Persona
					currentStepIdx = i
				}
			}
		}
	}

	// Calculate weighted overall progress using ProgressCalculator approach
	overallProgress := 0
	if btpd.totalSteps > 0 {
		// Sum progress across all running steps
		runningProgressSum := 0
		for _, stepID := range btpd.stepOrder {
			if step, exists := btpd.steps[stepID]; exists && step.State == StateRunning {
				runningProgressSum += step.Progress
			}
		}

		// Use weighted calculation: completed steps + partial running step progress
		completedWeight := float64(completed) / float64(btpd.totalSteps)
		currentWeight := 0.0
		if running > 0 {
			currentWeight = (float64(runningProgressSum) / 100.0) / float64(btpd.totalSteps)
		}
		weightedProgress := (completedWeight + currentWeight) * 100.0

		// Clamp to [0, 100] range
		if weightedProgress < 0 {
			overallProgress = 0
		} else if weightedProgress > 100 {
			overallProgress = 100
		} else {
			overallProgress = int(weightedProgress)
		}
	}

	// Convert step statuses
	stepStatuses := make(map[string]ProgressState)
	for stepID, step := range btpd.steps {
		stepStatuses[stepID] = step.State
	}

	// Build step personas mapping
	stepPersonas := make(map[string]string)
	for stepID, step := range btpd.steps {
		if step.Persona != "" {
			stepPersonas[stepID] = step.Persona
		}
	}

	// Get deliverables by step
	deliverablesByStep := make(map[string][]string)
	if btpd.deliverableTracker != nil {
		stepDeliverables := btpd.deliverableTracker.FormatByStep()
		for stepID, deliverables := range stepDeliverables {
			deliverablesByStep[stepID] = deliverables
		}
	}

	// Calculate current step start time - use actual step start if available
	currentStepStart := btpd.startTime.UnixNano() // Default to pipeline start
	if currentStepID != "" {
		if stepStartTime, exists := btpd.stepStartTimes[currentStepID]; exists {
			currentStepStart = stepStartTime.UnixNano()
		}
	}

	// Build per-step start times map (Unix nanoseconds)
	stepStartTimesNano := make(map[string]int64, len(btpd.stepStartTimes))
	for sid, t := range btpd.stepStartTimes {
		stepStartTimesNano[sid] = t.UnixNano()
	}

	// Copy per-step tool activity
	stepToolActivity := make(map[string][2]string, len(btpd.stepToolActivity))
	for sid, ta := range btpd.stepToolActivity {
		stepToolActivity[sid] = ta
	}


	// Build handover info with target steps
	handoversByStep := make(map[string]*HandoverInfo, len(btpd.handoverInfo))
	for stepID, info := range btpd.handoverInfo {
		copied := *info
		// Determine handover target: next step in order
		for i, sid := range btpd.stepOrder {
			if sid == stepID && i+1 < len(btpd.stepOrder) {
				copied.TargetStep = btpd.stepOrder[i+1]
				break
			}
		}
		handoversByStep[stepID] = &copied
	}
	return &PipelineContext{
		PipelineName:       btpd.pipelineName,
		PipelineID:         btpd.pipelineID,
		OverallProgress:    overallProgress,
		CurrentStepNum:     currentStepIdx + 1, // 1-indexed for display
		TotalSteps:         btpd.totalSteps,
		CurrentStepID:      currentStepID,
		CurrentPersona:     currentPersona,
		CompletedSteps:     completed,
		FailedSteps:        failed,
		SkippedSteps:       skipped,
		StepStatuses:       stepStatuses,
		StepOrder:          btpd.stepOrder,
		StepDurations:      btpd.stepDurations,
		StepPersonas:       stepPersonas,
		DeliverablesByStep: deliverablesByStep,
		ElapsedTimeMs:      elapsedMs,
		EstimatedTimeMs:    0, // Not calculated
		ManifestPath:       "wave.yaml",
		WorkspacePath:      ".wave/workspaces",
		CurrentAction:      "",
		CurrentStepName:    currentStepID,
		PipelineStartTime:  btpd.startTime.UnixNano(),
		CurrentStepStart:   currentStepStart, // Now uses actual step start time
		StepStartTimes:     stepStartTimesNano,
		StepToolActivity:   stepToolActivity,
		LastToolName:       btpd.lastToolName,
		LastToolTarget:     btpd.lastToolTarget,
		HandoversByStep:   handoversByStep,
		Verbose:           btpd.verbose,
	}
}