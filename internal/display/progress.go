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
