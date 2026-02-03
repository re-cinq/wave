package display

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/recinq/wave/internal/event"
)

// ProgressBar represents a visual progress indicator with customizable styling.
type ProgressBar struct {
	width      int
	current    int
	total      int
	prefix     string
	suffix     string
	fillChar   string
	emptyChar  string
	leftCap    string
	rightCap   string
	showPercent bool
	codec      *ANSICodec
	charSet    UnicodeCharSet
}

// NewProgressBar creates a new progress bar with default styling.
func NewProgressBar(total int, width int) *ProgressBar {
	codec := NewANSICodec()
	charSet := GetUnicodeCharSet()

	return &ProgressBar{
		width:       width,
		total:       total,
		current:     0,
		prefix:      "",
		suffix:      "",
		fillChar:    charSet.Block,
		emptyChar:   charSet.LightBlock,
		leftCap:     "[",
		rightCap:    "]",
		showPercent: true,
		codec:       codec,
		charSet:     charSet,
	}
}

// SetProgress updates the current progress value.
func (pb *ProgressBar) SetProgress(current int) {
	if current > pb.total {
		current = pb.total
	}
	if current < 0 {
		current = 0
	}
	pb.current = current
}

// SetPrefix sets the text shown before the progress bar.
func (pb *ProgressBar) SetPrefix(prefix string) {
	pb.prefix = prefix
}

// SetSuffix sets the text shown after the progress bar.
func (pb *ProgressBar) SetSuffix(suffix string) {
	pb.suffix = suffix
}

// Render returns the formatted progress bar string.
func (pb *ProgressBar) Render() string {
	var sb strings.Builder

	// Add prefix if set
	if pb.prefix != "" {
		sb.WriteString(pb.prefix)
		sb.WriteString(" ")
	}

	// Calculate fill amount
	var fillWidth int
	if pb.total > 0 {
		fillWidth = (pb.current * pb.width) / pb.total
	}

	// Build progress bar
	sb.WriteString(pb.leftCap)
	for i := 0; i < pb.width; i++ {
		if i < fillWidth {
			sb.WriteString(pb.codec.Primary(pb.fillChar))
		} else {
			sb.WriteString(pb.codec.Muted(pb.emptyChar))
		}
	}
	sb.WriteString(pb.rightCap)

	// Add percentage if enabled
	if pb.showPercent && pb.total > 0 {
		percent := (pb.current * 100) / pb.total
		sb.WriteString(fmt.Sprintf(" %3d%%", percent))
	}

	// Add suffix if set
	if pb.suffix != "" {
		sb.WriteString(" ")
		sb.WriteString(pb.suffix)
	}

	return sb.String()
}

// StepStatus represents the display state for a single pipeline step.
type StepStatus struct {
	StepID        string
	Name          string
	State         ProgressState
	Persona       string
	Message       string
	Progress      int
	CurrentAction string
	StartTime     time.Time
	EndTime       *time.Time
	ElapsedMs     int64
	TokensUsed    int
	Spinner       *Spinner
}

// NewStepStatus creates a new step status display.
func NewStepStatus(stepID, name, persona string) *StepStatus {
	return &StepStatus{
		StepID:    stepID,
		Name:      name,
		State:     StateNotStarted,
		Persona:   persona,
		StartTime: time.Now(),
		Spinner:   NewSpinner(AnimationSpinner),
	}
}

// UpdateState transitions the step to a new state.
func (ss *StepStatus) UpdateState(newState ProgressState) {
	oldState := ss.State
	ss.State = newState

	// Handle state transitions
	if oldState != StateRunning && newState == StateRunning {
		ss.StartTime = time.Now()
		if ss.Spinner != nil {
			ss.Spinner.Start()
		}
	}

	if newState == StateCompleted || newState == StateFailed || newState == StateCancelled {
		now := time.Now()
		ss.EndTime = &now
		ss.ElapsedMs = now.Sub(ss.StartTime).Milliseconds()
		if ss.Spinner != nil {
			ss.Spinner.Stop()
		}
	}
}

// Render returns the formatted step status line.
func (ss *StepStatus) Render() string {
	codec := NewANSICodec()
	charSet := GetUnicodeCharSet()

	var sb strings.Builder

	// State icon
	switch ss.State {
	case StateCompleted:
		sb.WriteString(codec.Success(charSet.CheckMark))
	case StateFailed:
		sb.WriteString(codec.Error(charSet.CrossMark))
	case StateRunning:
		if ss.Spinner != nil {
			sb.WriteString(codec.Primary(ss.Spinner.Current()))
		} else {
			sb.WriteString(codec.Primary("⟳"))
		}
	case StateSkipped:
		sb.WriteString(codec.Muted("○"))
	case StateCancelled:
		sb.WriteString(codec.Warning("⊛"))
	default:
		sb.WriteString(codec.Muted("○"))
	}

	sb.WriteString(" ")

	// Step name and persona
	sb.WriteString(codec.Bold(ss.Name))
	if ss.Persona != "" {
		sb.WriteString(codec.Muted(fmt.Sprintf(" (%s)", ss.Persona)))
	}

	// Progress percentage
	if ss.Progress > 0 && ss.State == StateRunning {
		sb.WriteString(fmt.Sprintf(" %s%d%%%s", codec.Primary("["), ss.Progress, codec.Primary("]")))
	}

	// Current action
	if ss.CurrentAction != "" && ss.State == StateRunning {
		sb.WriteString(codec.Muted(fmt.Sprintf(" - %s", ss.CurrentAction)))
	}

	// Elapsed time
	if ss.State == StateRunning {
		elapsed := time.Since(ss.StartTime)
		sb.WriteString(codec.Muted(fmt.Sprintf(" (%s)", formatStepDuration(elapsed))))
	} else if ss.EndTime != nil {
		sb.WriteString(codec.Muted(fmt.Sprintf(" (%.1fs)", float64(ss.ElapsedMs)/1000.0)))
	}

	// Tokens used (for completed steps)
	if ss.TokensUsed > 0 && (ss.State == StateCompleted || ss.State == StateFailed) {
		sb.WriteString(codec.Muted(fmt.Sprintf(" • %dk tokens", ss.TokensUsed/1000)))
	}

	// Message
	if ss.Message != "" {
		sb.WriteString(codec.Muted(fmt.Sprintf(" • %s", ss.Message)))
	}

	return sb.String()
}

// ProgressDisplay manages the real-time display of pipeline progress.
type ProgressDisplay struct {
	mu             sync.Mutex
	writer         io.Writer
	termInfo       *TerminalInfo
	codec          *ANSICodec
	charSet        UnicodeCharSet
	pipelineID     string
	pipelineName   string
	totalSteps     int
	currentStepIdx int
	steps          map[string]*StepStatus
	stepOrder      []string
	overallBar     *ProgressBar
	startTime      time.Time
	lastRender     time.Time
	refreshRate    time.Duration
	enabled        bool
	linesRendered  int
}

// NewProgressDisplay creates a new progress display manager.
func NewProgressDisplay(pipelineID, pipelineName string, totalSteps int) *ProgressDisplay {
	termInfo := NewTerminalInfo()
	codec := NewANSICodec()
	charSet := GetUnicodeCharSet()

	// Create overall progress bar
	barWidth := 30
	if termInfo.GetWidth() > 80 {
		barWidth = 40
	}
	overallBar := NewProgressBar(totalSteps, barWidth)
	overallBar.SetPrefix(codec.Bold("Pipeline Progress"))

	pd := &ProgressDisplay{
		writer:        os.Stderr,
		termInfo:      termInfo,
		codec:         codec,
		charSet:       charSet,
		pipelineID:    pipelineID,
		pipelineName:  pipelineName,
		totalSteps:    totalSteps,
		steps:         make(map[string]*StepStatus),
		stepOrder:     make([]string, 0, totalSteps),
		overallBar:    overallBar,
		startTime:     time.Now(),
		refreshRate:   100 * time.Millisecond,
		enabled:       termInfo.IsTTY() && termInfo.SupportsANSI(),
		linesRendered: 0,
	}

	return pd
}

// AddStep registers a new step for tracking.
func (pd *ProgressDisplay) AddStep(stepID, name, persona string) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if _, exists := pd.steps[stepID]; !exists {
		pd.steps[stepID] = NewStepStatus(stepID, name, persona)
		pd.stepOrder = append(pd.stepOrder, stepID)
	}
}

// UpdateStep updates the state of a specific step.
func (pd *ProgressDisplay) UpdateStep(stepID string, state ProgressState, message string, progress int) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	step, exists := pd.steps[stepID]
	if !exists {
		// Auto-add step if not registered
		step = NewStepStatus(stepID, stepID, "")
		pd.steps[stepID] = step
		pd.stepOrder = append(pd.stepOrder, stepID)
	}

	step.UpdateState(state)
	step.Message = message
	step.Progress = progress

	// Update overall progress
	completedSteps := 0
	for _, s := range pd.steps {
		if s.State == StateCompleted {
			completedSteps++
		}
	}
	pd.overallBar.SetProgress(completedSteps)

	pd.render()
}

// UpdateStepAction updates the current action for a running step.
func (pd *ProgressDisplay) UpdateStepAction(stepID, action string) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if step, exists := pd.steps[stepID]; exists {
		step.CurrentAction = action
		pd.render()
	}
}

// UpdateStepTokens updates the token usage for a step.
func (pd *ProgressDisplay) UpdateStepTokens(stepID string, tokens int) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if step, exists := pd.steps[stepID]; exists {
		step.TokensUsed = tokens
	}
}

// EmitProgress processes an event and updates the display.
func (pd *ProgressDisplay) EmitProgress(ev event.Event) error {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	// Handle step-level events
	if ev.StepID != "" {
		step, exists := pd.steps[ev.StepID]
		if !exists {
			// Auto-register step
			step = NewStepStatus(ev.StepID, ev.StepID, ev.Persona)
			pd.steps[ev.StepID] = step
			pd.stepOrder = append(pd.stepOrder, ev.StepID)
		}

		// Update step based on event state
		switch ev.State {
		case "started", "running":
			step.UpdateState(StateRunning)
			step.Message = ev.Message
		case "completed":
			step.UpdateState(StateCompleted)
			step.TokensUsed = ev.TokensUsed
			step.Message = ev.Message
		case "failed":
			step.UpdateState(StateFailed)
			step.Message = ev.Message
		case "retrying":
			step.UpdateState(StateRunning)
			step.Message = ev.Message
		case "step_progress":
			step.Progress = ev.Progress
			step.CurrentAction = ev.CurrentAction
		case "validating", "contract_validating":
			step.CurrentAction = "Validating contract"
		case "compacting", "compaction_progress":
			step.CurrentAction = "Compacting context"
		}

		pd.render()
	}

	return nil
}

// render updates the terminal display.
func (pd *ProgressDisplay) render() {
	if !pd.enabled {
		return
	}

	// Throttle updates
	now := time.Now()
	if now.Sub(pd.lastRender) < pd.refreshRate {
		return
	}
	pd.lastRender = now

	// Clear previous render
	if pd.linesRendered > 0 {
		for i := 0; i < pd.linesRendered; i++ {
			fmt.Fprint(pd.writer, pd.codec.CursorUp(1))
			fmt.Fprint(pd.writer, pd.codec.ClearLine())
		}
	}

	var lines []string

	// Pipeline header
	elapsed := time.Since(pd.startTime)
	header := fmt.Sprintf("%s %s %s",
		pd.codec.Bold(pd.pipelineName),
		pd.codec.Muted("•"),
		pd.codec.Muted(formatStepDuration(elapsed)))
	lines = append(lines, header)

	// Overall progress bar
	lines = append(lines, pd.overallBar.Render())
	lines = append(lines, "")

	// Step status lines
	for _, stepID := range pd.stepOrder {
		if step, exists := pd.steps[stepID]; exists {
			lines = append(lines, step.Render())
		}
	}

	// Write all lines
	for _, line := range lines {
		fmt.Fprintln(pd.writer, line)
	}

	pd.linesRendered = len(lines)
}

// Clear removes the progress display from the terminal.
func (pd *ProgressDisplay) Clear() {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if !pd.enabled || pd.linesRendered == 0 {
		return
	}

	for i := 0; i < pd.linesRendered; i++ {
		fmt.Fprint(pd.writer, pd.codec.CursorUp(1))
		fmt.Fprint(pd.writer, pd.codec.ClearLine())
	}

	pd.linesRendered = 0
}

// Finish completes the progress display and shows a summary.
func (pd *ProgressDisplay) Finish() {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if !pd.enabled {
		return
	}

	// Stop all spinners
	for _, step := range pd.steps {
		if step.Spinner != nil {
			step.Spinner.Stop()
		}
	}

	// Final render
	pd.render()
}

// BasicProgressDisplay provides simple text-based progress for non-TTY environments.
type BasicProgressDisplay struct {
	mu     sync.Mutex
	writer io.Writer
}

// NewBasicProgressDisplay creates a fallback progress display.
func NewBasicProgressDisplay() *BasicProgressDisplay {
	return &BasicProgressDisplay{
		writer: os.Stderr,
	}
}

// EmitProgress outputs simple text-based progress updates.
func (bpd *BasicProgressDisplay) EmitProgress(ev event.Event) error {
	bpd.mu.Lock()
	defer bpd.mu.Unlock()

	timestamp := ev.Timestamp.Format("15:04:05")

	if ev.StepID != "" {
		switch ev.State {
		case "started", "running":
			fmt.Fprintf(bpd.writer, "[%s] → %s (%s)\n", timestamp, ev.StepID, ev.Persona)
		case "completed":
			fmt.Fprintf(bpd.writer, "[%s] ✓ %s completed (%.1fs, %dk tokens)\n",
				timestamp, ev.StepID, float64(ev.DurationMs)/1000.0, ev.TokensUsed/1000)
		case "failed":
			fmt.Fprintf(bpd.writer, "[%s] ✗ %s failed: %s\n", timestamp, ev.StepID, ev.Message)
		case "step_progress":
			if ev.Progress > 0 {
				fmt.Fprintf(bpd.writer, "[%s]   %s: %d%%\n", timestamp, ev.StepID, ev.Progress)
			}
		case "validating", "contract_validating":
			fmt.Fprintf(bpd.writer, "[%s]   %s: validating contract\n", timestamp, ev.StepID)
		}
	}

	return nil
}

// formatStepDuration formats a duration for display.
func formatStepDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

// ProgressCalculator provides utilities for calculating pipeline progress metrics,
// including overall completion percentage and estimated time to completion (ETA).
type ProgressCalculator struct {
	// No state needed for now - all calculations are based on input parameters
}

// NewProgressCalculator creates a new progress calculator.
func NewProgressCalculator() *ProgressCalculator {
	return &ProgressCalculator{}
}

// CalculateOverallProgress computes the overall pipeline completion percentage.
// It uses a weighted approach that considers:
// - Completed steps (full weight)
// - Current step progress (partial weight)
// - Remaining steps (no weight)
func (pc *ProgressCalculator) CalculateOverallProgress(
	totalSteps int,
	completedSteps int,
	currentStepProgress int,
) int {
	if totalSteps <= 0 {
		return 0
	}

	// Calculate progress as percentage
	// Each completed step contributes (100 / totalSteps) percent
	// Current step contributes its partial progress
	completedWeight := float64(completedSteps) / float64(totalSteps)
	currentWeight := (float64(currentStepProgress) / 100.0) / float64(totalSteps)

	overallProgress := (completedWeight + currentWeight) * 100.0

	// Clamp to [0, 100] range
	if overallProgress < 0 {
		return 0
	}
	if overallProgress > 100 {
		return 100
	}

	return int(overallProgress)
}

// CalculateETA estimates the time remaining for pipeline completion.
// It uses historical average step duration and applies it to remaining steps.
// Returns ETA in milliseconds.
func (pc *ProgressCalculator) CalculateETA(
	totalSteps int,
	completedSteps int,
	averageStepTimeMs int64,
	currentStepElapsedMs int64,
	currentStepProgress int,
) int64 {
	if totalSteps <= 0 || completedSteps >= totalSteps {
		return 0
	}

	// Calculate remaining time for current step
	var currentStepRemainingMs int64
	if currentStepProgress > 0 && currentStepProgress < 100 {
		// Estimate total time for current step based on progress
		estimatedCurrentStepTotal := (currentStepElapsedMs * 100) / int64(currentStepProgress)
		currentStepRemainingMs = estimatedCurrentStepTotal - currentStepElapsedMs
	} else if averageStepTimeMs > 0 {
		// Use average if no progress available
		currentStepRemainingMs = averageStepTimeMs - currentStepElapsedMs
	}

	// Ensure non-negative
	if currentStepRemainingMs < 0 {
		currentStepRemainingMs = 0
	}

	// Calculate remaining steps after current one
	remainingSteps := totalSteps - completedSteps - 1
	if remainingSteps < 0 {
		remainingSteps = 0
	}

	// Calculate ETA for remaining steps
	var remainingStepsETA int64
	if averageStepTimeMs > 0 {
		remainingStepsETA = averageStepTimeMs * int64(remainingSteps)
	}

	totalETA := currentStepRemainingMs + remainingStepsETA
	return totalETA
}

// UpdatePipelineContext updates a PipelineContext with calculated progress metrics.
// This is a convenience method that combines progress and ETA calculations.
func (pc *ProgressCalculator) UpdatePipelineContext(
	ctx *PipelineContext,
	currentStepProgress int,
) {
	// Calculate overall progress
	ctx.OverallProgress = pc.CalculateOverallProgress(
		ctx.TotalSteps,
		ctx.CompletedSteps,
		currentStepProgress,
	)

	// Calculate elapsed time
	if ctx.PipelineStartTime > 0 {
		now := time.Now().UnixNano()
		ctx.ElapsedTimeMs = (now - ctx.PipelineStartTime) / int64(time.Millisecond)
	}

	// Calculate average step time from completed steps
	if ctx.CompletedSteps > 0 && ctx.ElapsedTimeMs > 0 {
		ctx.AverageStepTimeMs = ctx.ElapsedTimeMs / int64(ctx.CompletedSteps)
	}

	// Calculate current step elapsed time
	var currentStepElapsedMs int64
	if ctx.CurrentStepStart > 0 {
		now := time.Now().UnixNano()
		currentStepElapsedMs = (now - ctx.CurrentStepStart) / int64(time.Millisecond)
	}

	// Calculate ETA
	ctx.EstimatedTimeMs = pc.CalculateETA(
		ctx.TotalSteps,
		ctx.CompletedSteps,
		ctx.AverageStepTimeMs,
		currentStepElapsedMs,
		currentStepProgress,
	)
}

// CreatePipelineContext creates a new PipelineContext from pipeline metadata.
func CreatePipelineContext(
	manifestPath string,
	pipelineName string,
	workspacePath string,
	totalSteps int,
	stepIDs []string,
) *PipelineContext {
	ctx := &PipelineContext{
		ManifestPath:      manifestPath,
		PipelineName:      pipelineName,
		WorkspacePath:     workspacePath,
		TotalSteps:        totalSteps,
		CurrentStepNum:    0,
		CompletedSteps:    0,
		FailedSteps:       0,
		SkippedSteps:      0,
		OverallProgress:   0,
		EstimatedTimeMs:   0,
		PipelineStartTime: time.Now().UnixNano(),
		StepStatuses:      make(map[string]ProgressState),
	}

	// Initialize step statuses
	for _, stepID := range stepIDs {
		ctx.StepStatuses[stepID] = StateNotStarted
	}

	return ctx
}

// UpdateStepStatus updates the status of a specific step in the context.
func (ctx *PipelineContext) UpdateStepStatus(stepID string, state ProgressState) {
	if ctx.StepStatuses == nil {
		ctx.StepStatuses = make(map[string]ProgressState)
	}

	oldState := ctx.StepStatuses[stepID]
	ctx.StepStatuses[stepID] = state

	// Update counters based on state transitions
	switch state {
	case StateCompleted:
		if oldState != StateCompleted {
			ctx.CompletedSteps++
		}
	case StateFailed:
		if oldState != StateFailed {
			ctx.FailedSteps++
		}
	case StateSkipped:
		if oldState != StateSkipped {
			ctx.SkippedSteps++
		}
	case StateRunning:
		ctx.CurrentStepID = stepID
		ctx.CurrentStepStart = time.Now().UnixNano()
	}
}

// SetCurrentStep updates the current step information in the context.
func (ctx *PipelineContext) SetCurrentStep(stepNum int, stepID string, stepName string, persona string) {
	ctx.CurrentStepNum = stepNum
	ctx.CurrentStepID = stepID
	ctx.CurrentStepName = stepName
	ctx.CurrentPersona = persona
	ctx.CurrentStepStart = time.Now().UnixNano()

	// Update step status to running
	ctx.UpdateStepStatus(stepID, StateRunning)
}

// SetCurrentAction updates the current action being performed.
func (ctx *PipelineContext) SetCurrentAction(action string) {
	ctx.CurrentAction = action
}

// MarkStepCompleted marks a step as completed and updates counters.
func (ctx *PipelineContext) MarkStepCompleted(stepID string) {
	ctx.UpdateStepStatus(stepID, StateCompleted)
}

// MarkStepFailed marks a step as failed and updates counters.
func (ctx *PipelineContext) MarkStepFailed(stepID string, errorMsg string) {
	ctx.UpdateStepStatus(stepID, StateFailed)
	ctx.Error = errorMsg
}

// MarkStepSkipped marks a step as skipped and updates counters.
func (ctx *PipelineContext) MarkStepSkipped(stepID string) {
	ctx.UpdateStepStatus(stepID, StateSkipped)
}

// GetCompletionPercentage returns the overall completion percentage (0-100).
func (ctx *PipelineContext) GetCompletionPercentage() int {
	return ctx.OverallProgress
}

// GetCurrentStepNumber returns the 1-based index of the current step.
func (ctx *PipelineContext) GetCurrentStepNumber() int {
	return ctx.CurrentStepNum
}

// IsComplete returns true if all steps are completed or the pipeline has finished.
func (ctx *PipelineContext) IsComplete() bool {
	return ctx.CompletedSteps >= ctx.TotalSteps
}

// HasErrors returns true if any steps have failed.
func (ctx *PipelineContext) HasErrors() bool {
	return ctx.FailedSteps > 0 || ctx.Error != ""
}
