package display

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/pathfmt"
)

// ProgressBar represents a visual progress indicator with customizable styling.
type ProgressBar struct {
	width       int
	current     int
	total       int
	prefix      string
	suffix      string
	fillChar    string
	emptyChar   string
	leftCap     string
	rightCap    string
	showPercent bool
	codec       *ANSICodec
	charSet     UnicodeCharSet
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
	Model         string
	Adapter       string
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

	// Adapter and model
	if ss.Adapter != "" || ss.Model != "" {
		parts := []string{}
		if ss.Adapter != "" {
			parts = append(parts, ss.Adapter)
		}
		if ss.Model != "" {
			parts = append(parts, ss.Model)
		}
		sb.WriteString(codec.Muted(fmt.Sprintf(" [%s]", strings.Join(parts, "/"))))
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
		sb.WriteString(codec.Muted(fmt.Sprintf(" • %s tokens", FormatTokenCount(ss.TokensUsed))))
	}

	// Message
	if ss.Message != "" {
		sb.WriteString(codec.Muted(fmt.Sprintf(" • %s", ss.Message)))
	}

	return sb.String()
}

// ProgressDisplay manages the real-time display of pipeline progress.
type ProgressDisplay struct {
	mu              sync.Mutex
	writer          io.Writer
	termInfo        *TerminalInfo
	codec           *ANSICodec
	charSet         UnicodeCharSet
	dashboard       *Dashboard
	pipelineID      string
	pipelineName    string
	totalSteps      int
	currentStepIdx  int
	steps           map[string]*StepStatus
	stepOrder       []string
	overallBar      *ProgressBar
	startTime       time.Time
	lastRender      time.Time
	refreshRate     time.Duration
	enabled         bool
	linesRendered   int
	estimatedTimeMs int64 // Latest ETA from pipeline events
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
		dashboard:     NewDashboard(),
		pipelineID:    pipelineID,
		pipelineName:  pipelineName,
		totalSteps:    totalSteps,
		steps:         make(map[string]*StepStatus),
		stepOrder:     make([]string, 0, totalSteps),
		overallBar:    overallBar,
		startTime:     time.Now(),
		refreshRate:   200 * time.Millisecond, // 5 FPS for smooth progress updates without flickering
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

	// Capture ETA from events
	if ev.EstimatedTimeMs > 0 {
		pd.estimatedTimeMs = ev.EstimatedTimeMs
	}

	// Handle pipeline-level warnings (no StepID)
	if ev.StepID == "" && ev.State == "warning" && ev.Message != "" {
		pd.render()
		return nil
	}

	// Handle step-level events
	if ev.StepID != "" {
		step, exists := pd.steps[ev.StepID]
		if !exists {
			// Auto-register step
			step = NewStepStatus(ev.StepID, ev.StepID, ev.Persona)
			pd.steps[ev.StepID] = step
			pd.stepOrder = append(pd.stepOrder, ev.StepID)
		}

		// Update persona from resolved event value (defense-in-depth for forge template vars)
		if ev.Persona != "" {
			step.Persona = ev.Persona
		}

		// Update adapter and model from event (capture on first run)
		if ev.Adapter != "" && step.Adapter == "" {
			step.Adapter = ev.Adapter
		}
		if ev.Model != "" && step.Model == "" {
			step.Model = ev.Model
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
		case "warning":
			step.Message = ev.Message
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

	// No clearing needed - just append new content

	// Convert ProgressDisplay data to PipelineContext for dashboard rendering
	ctx := pd.toPipelineContext()

	// Use dashboard rendering (includes logo in header)
	if err := pd.dashboard.Render(ctx); err != nil {
		// Fallback to simple text on render error
		fmt.Fprintf(pd.writer, "Progress: %s (%d/%d steps)\n",
			pd.pipelineName, pd.currentStepIdx, pd.totalSteps)
		pd.linesRendered = 1
		return
	}

	// Dashboard handles its own line counting, so we estimate lines rendered
	// Approximate: header(4) + progress(5) + steps(N) + project(3) + spacing
	pd.linesRendered = 4 + 5 + len(pd.stepOrder) + 3 + 3
}

// Clear removes the progress display from the terminal (no-op since we don't clear).
func (pd *ProgressDisplay) Clear() {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	// No clearing needed
}

// toPipelineContext converts ProgressDisplay data to PipelineContext for dashboard rendering.
func (pd *ProgressDisplay) toPipelineContext() *PipelineContext {
	elapsed := time.Since(pd.startTime)
	elapsedMs := elapsed.Nanoseconds() / int64(time.Millisecond)

	// Calculate overall progress
	completed := 0
	failed := 0
	skipped := 0
	for _, step := range pd.steps {
		switch step.State {
		case StateCompleted:
			completed++
		case StateFailed:
			failed++
		case StateSkipped:
			skipped++
		}
	}

	overallProgress := 0
	if pd.totalSteps > 0 {
		overallProgress = (completed * 100) / pd.totalSteps
	}

	// Find current step
	currentStepID := ""
	currentPersona := ""
	for _, stepID := range pd.stepOrder {
		if step, exists := pd.steps[stepID]; exists && step.State == StateRunning {
			currentStepID = stepID
			// Extract persona if available (simplified)
			break
		}
	}

	// Convert step statuses
	stepStatuses := make(map[string]ProgressState)
	for stepID, step := range pd.steps {
		stepStatuses[stepID] = step.State
	}

	// Build step personas mapping
	stepPersonas := make(map[string]string)
	for stepID, step := range pd.steps {
		if step.Persona != "" {
			stepPersonas[stepID] = step.Persona
		}
	}

	// Compute per-step tokens and total
	stepTokens := make(map[string]int, len(pd.steps))
	totalTokens := 0
	for stepID, step := range pd.steps {
		if step.TokensUsed > 0 {
			stepTokens[stepID] = step.TokensUsed
			totalTokens += step.TokensUsed
		}
	}

	// Build step models and adapters mapping
	stepModels := make(map[string]string)
	stepAdapters := make(map[string]string)
	for stepID, step := range pd.steps {
		if step.Model != "" {
			stepModels[stepID] = step.Model
		}
		if step.Adapter != "" {
			stepAdapters[stepID] = step.Adapter
		}
	}

	return &PipelineContext{
		PipelineName:      pd.pipelineName,
		PipelineID:        pd.pipelineID,
		OverallProgress:   overallProgress,
		CurrentStepNum:    pd.currentStepIdx + 1, // 1-indexed for display
		TotalSteps:        pd.totalSteps,
		CurrentStepID:     currentStepID,
		CurrentPersona:    currentPersona,
		CompletedSteps:    completed,
		FailedSteps:       failed,
		SkippedSteps:      skipped,
		StepStatuses:      stepStatuses,
		StepOrder:         pd.stepOrder,
		StepPersonas:      stepPersonas,
		ElapsedTimeMs:     elapsedMs,
		EstimatedTimeMs:   pd.estimatedTimeMs,
		AverageStepTimeMs: pd.averageStepTimeMs(),
		ManifestPath:      "wave.yaml",
		WorkspacePath:     ".agents/workspaces",
		CurrentAction:     "", // Not tracked in ProgressDisplay
		CurrentStepName:   currentStepID,
		PipelineStartTime: pd.startTime.UnixNano(),
		CurrentStepStart:  pd.startTime.UnixNano(), // Simplified
		StepTokens:        stepTokens,
		TotalTokens:       totalTokens,
		StepModels:        stepModels,
		StepAdapters:      stepAdapters,
	}
}

// averageStepTimeMs computes the average duration of completed steps.
// Caller must hold pd.mu.
func (pd *ProgressDisplay) averageStepTimeMs() int64 {
	var total int64
	var count int64
	for _, step := range pd.steps {
		if step.State == StateCompleted && step.ElapsedMs > 0 {
			total += step.ElapsedMs
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / count
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
	mu           sync.Mutex
	writer       io.Writer
	verbose      bool
	termInfo     *TerminalInfo
	handoverInfo map[string]*HandoverInfo // Per-step handover metadata
	stepOrder    []string                 // Ordered list of step IDs (for target lookup)
	stepStates   map[string]string        // Per-step state tracking for activity guard
}

// NewBasicProgressDisplay creates a fallback progress display.
func NewBasicProgressDisplay() *BasicProgressDisplay {
	return &BasicProgressDisplay{
		writer:       os.Stderr,
		verbose:      false,
		termInfo:     NewTerminalInfo(),
		handoverInfo: make(map[string]*HandoverInfo),
		stepStates:   make(map[string]string),
	}
}

// NewBasicProgressDisplayWithVerbose creates a progress display with verbose tool activity.
func NewBasicProgressDisplayWithVerbose(verbose bool) *BasicProgressDisplay {
	return &BasicProgressDisplay{
		writer:       os.Stderr,
		verbose:      verbose,
		termInfo:     NewTerminalInfo(),
		handoverInfo: make(map[string]*HandoverInfo),
		stepStates:   make(map[string]string),
	}
}

// EmitProgress outputs simple text-based progress updates.
//
// State-tracking side effects (stepStates, stepOrder, handoverInfo) are kept
// here. Line emission delegates to the canonical EventLine formatter (see
// eventline.go) using the basic-CLI profile.
func (bpd *BasicProgressDisplay) EmitProgress(ev event.Event) error {
	bpd.mu.Lock()
	defer bpd.mu.Unlock()

	timestamp := ev.Timestamp.Format("15:04:05")

	// Update internal tracking first so downstream lookups (verbose handover
	// render, stream-activity guard) see the new state.
	bpd.updateTracking(ev)

	// stream_activity is suppressed when the step is not in "running" state.
	// The canonical formatter has no per-step state, so guard at the call site.
	if ev.State == "stream_activity" && ev.StepID != "" && bpd.stepStates[ev.StepID] != "running" {
		return nil
	}

	if line, emit := EventLine(ev, BasicCLIProfile(timestamp, bpd.termInfo, bpd.verbose)); emit {
		fmt.Fprintln(bpd.writer, line)
	}

	// On step completion in verbose mode, render handover metadata.
	if ev.State == "completed" && ev.StepID != "" && bpd.verbose {
		if info, exists := bpd.handoverInfo[ev.StepID]; exists {
			bpd.renderHandoverMetadata(timestamp, ev.StepID, info)
		}
	}

	return nil
}

// updateTracking maintains the per-step state, ordering, and handover metadata
// that the canonical formatter does not own. Caller must hold bpd.mu.
func (bpd *BasicProgressDisplay) updateTracking(ev event.Event) {
	if ev.StepID == "" {
		return
	}
	switch ev.State {
	case "started", "running":
		bpd.stepStates[ev.StepID] = "running"
		bpd.trackStepOrder(ev.StepID)
	case "completed":
		bpd.stepStates[ev.StepID] = "completed"
		if len(ev.Artifacts) > 0 {
			info := bpd.ensureHandoverInfo(ev.StepID)
			info.ArtifactPaths = ev.Artifacts
		}
	case "failed":
		bpd.stepStates[ev.StepID] = "failed"
	case "retrying":
		bpd.stepStates[ev.StepID] = "running"
	case "validating", "contract_validating":
		info := bpd.ensureHandoverInfo(ev.StepID)
		info.ContractSchema = ev.ValidationPhase
		bpd.trackStepOrder(ev.StepID)
	case "contract_passed":
		bpd.ensureHandoverInfo(ev.StepID).ContractStatus = "passed"
	case "contract_failed":
		bpd.ensureHandoverInfo(ev.StepID).ContractStatus = "failed"
	case "contract_soft_failure":
		bpd.ensureHandoverInfo(ev.StepID).ContractStatus = "soft_failure"
	}
}

func (bpd *BasicProgressDisplay) trackStepOrder(stepID string) {
	for _, sid := range bpd.stepOrder {
		if sid == stepID {
			return
		}
	}
	bpd.stepOrder = append(bpd.stepOrder, stepID)
}

func (bpd *BasicProgressDisplay) ensureHandoverInfo(stepID string) *HandoverInfo {
	if info, exists := bpd.handoverInfo[stepID]; exists {
		return info
	}
	info := &HandoverInfo{}
	bpd.handoverInfo[stepID] = info
	return info
}

// renderHandoverMetadata outputs handover metadata lines in tree format for a completed step.
func (bpd *BasicProgressDisplay) renderHandoverMetadata(timestamp, stepID string, info *HandoverInfo) {
	lines := bpd.buildHandoverLines(stepID, info)
	for _, line := range lines {
		fmt.Fprintf(bpd.writer, "[%s]   %s\n", timestamp, line)
	}
}

// BuildHandoverLines constructs tree-formatted handover metadata lines.
// stepID is the completing step, stepOrder is the ordered list of step IDs
// seen so far (used to resolve handover targets when info.TargetStep is empty).
func BuildHandoverLines(stepID string, info *HandoverInfo, stepOrder []string) []string {
	var items []string

	// Artifact lines
	for _, path := range info.ArtifactPaths {
		items = append(items, fmt.Sprintf("artifact: %s (written)", pathfmt.FileURI(path)))
	}

	// Contract line
	if info.ContractStatus != "" {
		status := "✓ valid"
		switch info.ContractStatus {
		case "failed":
			status = "✗ failed"
		case "soft_failure":
			status = "⚠ soft failure"
		}
		schema := info.ContractSchema
		if schema == "" {
			schema = "contract"
		}
		items = append(items, fmt.Sprintf("contract: %s %s", schema, status))
	}

	// Handover target line
	targetStep := info.TargetStep
	targetStepNum := 0
	if targetStep == "" {
		// Determine from step order
		for i, sid := range stepOrder {
			if sid == stepID && i+1 < len(stepOrder) {
				targetStep = stepOrder[i+1]
				targetStepNum = i + 2 // 1-based, and it's the next step
				break
			}
		}
	} else {
		// Find targetStep's position in stepOrder
		for i, sid := range stepOrder {
			if sid == targetStep {
				targetStepNum = i + 1 // 1-based
				break
			}
		}
	}
	if targetStep != "" {
		items = append(items, fmt.Sprintf("handover → step %d: %s", targetStepNum, targetStep))
	}

	// Format with tree connectors
	var lines []string
	for i, item := range items {
		connector := "├─"
		if i == len(items)-1 {
			connector = "└─"
		}
		lines = append(lines, fmt.Sprintf("%s %s", connector, item))
	}
	return lines
}

// buildHandoverLines delegates to the shared BuildHandoverLines function.
func (bpd *BasicProgressDisplay) buildHandoverLines(stepID string, info *HandoverInfo) []string {
	return BuildHandoverLines(stepID, info, bpd.stepOrder)
}

// QuietProgressDisplay only renders pipeline-level completed/failed events.
type QuietProgressDisplay struct {
	mu     sync.Mutex
	writer io.Writer
}

// NewQuietProgressDisplay creates a minimal progress display.
func NewQuietProgressDisplay() *QuietProgressDisplay {
	return &QuietProgressDisplay{
		writer: os.Stderr,
	}
}

// EmitProgress outputs only pipeline-level completion or failure.
func (qpd *QuietProgressDisplay) EmitProgress(ev event.Event) error {
	qpd.mu.Lock()
	defer qpd.mu.Unlock()

	// Only render pipeline-level completed/failed events (no step ID means pipeline event)
	if ev.StepID == "" {
		switch ev.State {
		case "completed":
			fmt.Fprintf(qpd.writer, "%s completed\n", ev.PipelineID)
		case "failed":
			fmt.Fprintf(qpd.writer, "%s failed: %s\n", ev.PipelineID, ev.Message)
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
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

// CreatePipelineContext creates a new PipelineContext from pipeline metadata.
func CreatePipelineContext(
	manifestPath string,
	pipelineName string,
	workspacePath string,
	totalSteps int,
	stepIDs []string,
	stepPersonas map[string]string,
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
		StepOrder:         stepIDs,
		StepPersonas:      stepPersonas,
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
	if ctx.StepDurations == nil {
		ctx.StepDurations = make(map[string]int64)
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

// UpdateStepDuration stores the duration for a completed step.
func (ctx *PipelineContext) UpdateStepDuration(stepID string, durationMs int64) {
	if ctx.StepDurations == nil {
		ctx.StepDurations = make(map[string]int64)
	}
	ctx.StepDurations[stepID] = durationMs
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
func (ctx *PipelineContext) MarkStepCompleted(stepID string, durationMs int64) {
	ctx.UpdateStepStatus(stepID, StateCompleted)
	ctx.UpdateStepDuration(stepID, durationMs)
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
