package display

import (
	"fmt"
	"strings"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/humanize"
)

// EventLineProfile selects which framing/layout the canonical event-line
// formatter applies.
type EventLineProfile int

const (
	// ProfileLiveTUI renders for the in-process live TUI / stored-event view.
	// Layout: "[stepID] [symbol ]Verb body" with title-case verbs and a fixed
	// 60-char stream-activity truncation. Drives both internal/tui
	// formatEventLine and formatStoredEvent.
	ProfileLiveTUI EventLineProfile = iota

	// ProfileBasicCLI renders for the non-TTY stderr CLI display.
	// Layout: "[HH:MM:SS] symbol stepID verb extra" with lowercase verbs and
	// terminal-width-aware stream truncation. Drives
	// display.BasicProgressDisplay.EmitProgress.
	ProfileBasicCLI
)

// EventLineOpts configures the canonical EventLine formatter.
//
// Fields prefixed Live* apply only to ProfileLiveTUI; Basic* apply only to
// ProfileBasicCLI. Cross-profile fields are ignored by the unused profile.
type EventLineOpts struct {
	Profile EventLineProfile

	// LiveColor enables ✓/✗/⚠ symbols and capitalized verb fallthroughs in
	// the live-TUI profile. When false, output degrades to plain "Verb:" form.
	LiveColor bool

	// BasicTimestamp is the pre-formatted "HH:MM:SS" string used as the line
	// prefix in the basic-CLI profile. Callers typically pass
	// ev.Timestamp.Format("15:04:05").
	BasicTimestamp string

	// BasicTermInfo is consulted for terminal-width-aware truncation of
	// stream_activity tool target strings in the basic-CLI profile.
	BasicTermInfo *TerminalInfo

	// BasicVerbose gates emission of stream_activity lines in the basic-CLI
	// profile.
	BasicVerbose bool
}

// LiveTUIProfile returns options for the live-TUI / stored-event renderer.
func LiveTUIProfile(color bool) EventLineOpts {
	return EventLineOpts{Profile: ProfileLiveTUI, LiveColor: color}
}

// BasicCLIProfile returns options for the basic-CLI stderr renderer.
func BasicCLIProfile(timestamp string, termInfo *TerminalInfo, verbose bool) EventLineOpts {
	return EventLineOpts{
		Profile:        ProfileBasicCLI,
		BasicTimestamp: timestamp,
		BasicTermInfo:  termInfo,
		BasicVerbose:   verbose,
	}
}

// EventLine formats a single event as a one-line display string.
//
// The returned emit flag is true when the event should produce a visible line.
// The basic-CLI profile suppresses some events (e.g. started without persona,
// non-verbose stream activity, contract_passed which is metadata-only); the
// live-TUI profile always emits.
func EventLine(evt event.Event, opts EventLineOpts) (line string, emit bool) {
	switch opts.Profile {
	case ProfileBasicCLI:
		return eventLineBasicCLI(evt, opts)
	default:
		return eventLineLive(evt, opts), true
	}
}

// stepIDOrPipeline returns evt.StepID, falling back to evt.PipelineID, then
// to "pipeline" so callers always have a non-empty bracket label.
func stepIDOrPipeline(evt event.Event) string {
	if evt.StepID != "" {
		return evt.StepID
	}
	if evt.PipelineID != "" {
		return evt.PipelineID
	}
	return "pipeline"
}

// eventLineLive renders an event in live-TUI / stored-event style.
func eventLineLive(evt event.Event, opts EventLineOpts) string {
	stepID := stepIDOrPipeline(evt)
	color := opts.LiveColor

	switch evt.State {
	case event.StateStarted:
		meta := ""
		var parts []string
		if evt.Persona != "" {
			parts = append(parts, evt.Persona)
		}
		if evt.Model != "" {
			parts = append(parts, evt.Model)
		}
		if len(parts) > 0 {
			meta = " (" + strings.Join(parts, ", ") + ")"
		}
		return fmt.Sprintf("[%s] Starting...%s", stepID, meta)

	case event.StateCompleted:
		suffix := ""
		if evt.DurationMs > 0 {
			d := time.Duration(evt.DurationMs) * time.Millisecond
			tokenInfo := ""
			switch {
			case evt.TokensIn > 0 || evt.TokensOut > 0:
				tokenInfo = fmt.Sprintf(", %s in / %s out", FormatTokenCount(evt.TokensIn), FormatTokenCount(evt.TokensOut))
			case evt.TokensUsed > 0:
				tokenInfo = fmt.Sprintf(", %s tokens", FormatTokenCount(evt.TokensUsed))
			}
			suffix = fmt.Sprintf(" (%s%s)", humanize.DurationMs(d.Milliseconds()), tokenInfo)
		}
		if color {
			return fmt.Sprintf("[%s] ✓ Completed%s", stepID, suffix)
		}
		return fmt.Sprintf("[%s] Completed%s", stepID, suffix)

	case event.StateFailed:
		msg := evt.Message
		if msg == "" {
			msg = "unknown error"
		}
		if color {
			return fmt.Sprintf("[%s] ✗ Failed: %s", stepID, msg)
		}
		return fmt.Sprintf("[%s] Failed: %s", stepID, msg)

	case event.StateRetrying:
		if evt.Message != "" {
			return fmt.Sprintf("[%s] Retrying: %s", stepID, evt.Message)
		}
		return fmt.Sprintf("[%s] Retrying...", stepID)

	case event.StateRunning:
		if evt.Message != "" {
			return fmt.Sprintf("[%s] %s", stepID, evt.Message)
		}
		return fmt.Sprintf("[%s] Running...", stepID)

	case "warning":
		if color {
			return fmt.Sprintf("[%s] ⚠ %s", stepID, evt.Message)
		}
		return fmt.Sprintf("[%s] Warning: %s", stepID, evt.Message)

	case event.StateContractValidating:
		phase := evt.ValidationPhase
		if phase == "" {
			phase = "validating"
		}
		return fmt.Sprintf("[%s] Contract: %s", stepID, phase)

	case "contract_passed":
		if color {
			return fmt.Sprintf("[%s] ✓ Contract: passed", stepID)
		}
		return fmt.Sprintf("[%s] Contract: passed", stepID)

	case "contract_failed":
		if color {
			return fmt.Sprintf("[%s] ✗ Contract: failed", stepID)
		}
		return fmt.Sprintf("[%s] Contract: failed", stepID)

	case "contract_soft_failure":
		return fmt.Sprintf("[%s] Contract: soft failure (continuing)", stepID)

	case event.StateStreamActivity:
		// Stored-events path: ToolName/ToolTarget are absent; the LogRecord
		// adapter packs the original "ToolName ToolTarget" into Message. Use
		// Message verbatim when ToolName is empty so both inputs match the
		// pre-refactor output byte-for-byte.
		if evt.ToolName == "" && evt.ToolTarget == "" {
			return fmt.Sprintf("[%s] %s", stepID, evt.Message)
		}
		target := evt.ToolTarget
		if len(target) > 60 {
			target = target[:57] + "..."
		}
		return fmt.Sprintf("[%s] %s %s", stepID, evt.ToolName, target)

	case event.StateStepProgress:
		switch {
		case evt.CurrentAction != "":
			return fmt.Sprintf("[%s] %s", stepID, evt.CurrentAction)
		case evt.TokensIn > 0 || evt.TokensOut > 0:
			return fmt.Sprintf("[%s] tokens: %s in / %s out", stepID, FormatTokenCount(evt.TokensIn), FormatTokenCount(evt.TokensOut))
		case evt.Progress > 0:
			return fmt.Sprintf("[%s] progress: %d%%", stepID, evt.Progress)
		case evt.Message != "":
			// Stored-events path: step_progress with action arrives via Message.
			return fmt.Sprintf("[%s] %s", stepID, evt.Message)
		}
		return fmt.Sprintf("[%s] heartbeat", stepID)

	case event.StateETAUpdated:
		if evt.EstimatedTimeMs > 0 {
			d := time.Duration(evt.EstimatedTimeMs) * time.Millisecond
			return fmt.Sprintf("[%s] ETA: ~%s remaining", stepID, humanize.DurationMs(d.Milliseconds()))
		}
		return fmt.Sprintf("[%s] ETA: calculating...", stepID)

	case event.StateCompactionProgress:
		return fmt.Sprintf("[%s] Context compaction in progress...", stepID)

	default:
		if evt.Message != "" {
			return fmt.Sprintf("[%s] %s", stepID, evt.Message)
		}
		return fmt.Sprintf("[%s] %s", stepID, evt.State)
	}
}

// eventLineBasicCLI renders an event in basic-CLI stderr style.
//
// Returns emit=false for events the basic profile suppresses entirely:
//   - started/running with no persona (no contextual line)
//   - step_progress with no action
//   - non-verbose stream_activity, or stream_activity for non-running steps
//   - contract_passed/failed/soft_failure (handover metadata; rendered later by
//     renderHandoverMetadata, not as an inline event line)
//   - any state not handled by the original switch (e.g. eta_updated)
func eventLineBasicCLI(evt event.Event, opts EventLineOpts) (string, bool) {
	timestamp := opts.BasicTimestamp

	// Pipeline-level warnings (no StepID)
	if evt.StepID == "" {
		if evt.State == "warning" && evt.Message != "" {
			return fmt.Sprintf("[%s] ⚠ %s", timestamp, evt.Message), true
		}
		return "", false
	}

	switch evt.State {
	case "started", "running":
		if evt.Persona == "" {
			return "", false
		}
		line := fmt.Sprintf("[%s] → %s (%s)", timestamp, evt.StepID, evt.Persona)
		if evt.Model != "" {
			line += fmt.Sprintf(" [%s", evt.Model)
			if evt.Adapter != "" {
				line += fmt.Sprintf(" via %s", evt.Adapter)
			}
			line += "]"
		}
		return line, true

	case "completed":
		tokenInfo := FormatTokenCount(evt.TokensUsed) + " tokens"
		if evt.TokensIn > 0 || evt.TokensOut > 0 {
			tokenInfo = FormatTokenCount(evt.TokensIn) + " in / " + FormatTokenCount(evt.TokensOut) + " out"
		}
		return fmt.Sprintf("[%s] ✓ %s completed (%.1fs, %s)", timestamp, evt.StepID, float64(evt.DurationMs)/1000.0, tokenInfo), true

	case "failed":
		return fmt.Sprintf("[%s] ✗ %s failed: %s", timestamp, evt.StepID, evt.Message), true

	case "retrying":
		return fmt.Sprintf("[%s] ↻ %s retrying: %s", timestamp, evt.StepID, evt.Message), true

	case "step_progress":
		if evt.CurrentAction == "" {
			return "", false
		}
		return fmt.Sprintf("[%s]   %s %s", timestamp, evt.StepID, evt.CurrentAction), true

	case "warning":
		return fmt.Sprintf("[%s] ⚠ %s %s", timestamp, evt.StepID, evt.Message), true

	case "validating", "contract_validating":
		return fmt.Sprintf("[%s]   %s validating contract", timestamp, evt.StepID), true

	case "stream_activity":
		if !opts.BasicVerbose || evt.ToolName == "" {
			return "", false
		}
		// Compute available width: total - prefix overhead
		// Format: "[HH:MM:SS]   %-20s %s → " = 10 + 3 + 20 + 1 + len(toolName) + 3
		width := 80
		if opts.BasicTermInfo != nil {
			width = opts.BasicTermInfo.GetWidth()
		}
		overhead := 37 + len(evt.ToolName)
		maxTarget := width - overhead
		if maxTarget < 20 {
			maxTarget = 20
		}
		target := evt.ToolTarget
		if len(target) > maxTarget {
			target = target[:maxTarget-3] + "..."
		}
		return fmt.Sprintf("[%s]   %-20s %s → %s", timestamp, evt.StepID, evt.ToolName, target), true

	default:
		return "", false
	}
}
