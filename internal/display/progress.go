package display

import (
	"fmt"
	"io"
	"os"
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

// HasErrors returns true if any steps have failed.
func (ctx *PipelineContext) HasErrors() bool {
	return ctx.FailedSteps > 0 || ctx.Error != ""
}
