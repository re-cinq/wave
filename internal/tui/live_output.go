package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/humanize"
	"github.com/recinq/wave/internal/state"
)

// EventBuffer is a bounded ring buffer of formatted display lines.
type EventBuffer struct {
	lines    []string
	capacity int
	head     int // Write position (next slot to write to)
	count    int // Number of entries currently in buffer
}

func NewEventBuffer(capacity int) *EventBuffer {
	if capacity <= 0 {
		capacity = 1000
	}
	return &EventBuffer{
		lines:    make([]string, capacity),
		capacity: capacity,
	}
}

func (b *EventBuffer) Append(line string) {
	b.lines[b.head] = line
	b.head = (b.head + 1) % b.capacity
	if b.count < b.capacity {
		b.count++
	}
}

func (b *EventBuffer) Lines() []string {
	if b.count == 0 {
		return nil
	}
	result := make([]string, b.count)
	if b.count < b.capacity {
		// Buffer not full yet — head is at the end of data
		copy(result, b.lines[:b.count])
	} else {
		// Buffer is full — head points to the oldest entry
		// Copy from head to end, then from 0 to head
		firstPart := b.capacity - b.head
		copy(result[:firstPart], b.lines[b.head:])
		copy(result[firstPart:], b.lines[:b.head])
	}
	return result
}

func (b *EventBuffer) Len() int {
	return b.count
}

// stepDashState tracks per-step dashboard state for the structured progress view.
type stepDashState struct {
	stepID         string
	persona        string
	model          string
	adapter        string
	status         string // "not_started", "running", "completed", "failed", "retrying"
	startedAt      time.Time
	durationMs     int64
	tokensIn       int
	tokensOut      int
	tokensUsed     int
	lastToolName   string
	lastToolTarget string
	contractStatus string // "", "passed", "failed", "soft_failure"
	contractSchema string
	artifacts      []string
	message        string
}

// DisplayFlags tracks which event categories are visible in the live output.
type DisplayFlags struct {
	Verbose    bool
	Debug      bool
	OutputOnly bool
}

// shouldFormat determines whether an event should be formatted into the buffer
// based on the current display flags. Mirrors BasicProgressDisplay.EmitProgress filtering.
func shouldFormat(evt event.Event, flags DisplayFlags) bool {
	if flags.OutputOnly {
		return evt.State == event.StateCompleted || evt.State == event.StateFailed
	}
	switch evt.State {
	case event.StateStarted:
		// Skip duplicate pipeline-level started events (e.g. workspace root info).
		// The first started event carries TotalSteps; subsequent ones are info-only
		// and would render as a redundant "Starting..." line.
		if evt.StepID == "" && evt.TotalSteps == 0 {
			return false
		}
		return true
	case event.StateRunning, event.StateCompleted,
		event.StateFailed, event.StateRetrying, event.StateContractValidating,
		"warning", "preflight", "contract_passed", "contract_failed", "contract_soft_failure":
		return true
	case event.StateStreamActivity:
		return flags.Verbose
	case event.StateStepProgress:
		if !flags.Debug {
			return false
		}
		// Skip empty heartbeats — only show progress with actual data (matches CLI behavior)
		return evt.TokensIn > 0 || evt.TokensOut > 0 || evt.CurrentAction != "" || evt.Progress > 0
	case event.StateETAUpdated, event.StateCompactionProgress:
		return flags.Debug
	default:
		return false
	}
}

// shouldFormatRecord determines whether a stored LogRecord should be displayed
// based on the current display flags. Used for detached pipeline runs where
// events come from SQLite rather than in-memory event.Event objects.
func shouldFormatRecord(rec state.LogRecord, flags DisplayFlags) bool {
	if flags.OutputOnly {
		return rec.State == event.StateCompleted || rec.State == event.StateFailed
	}
	switch rec.State {
	case event.StateStarted:
		return rec.StepID != "" // Filter pipeline-level info-only events
	case event.StateRunning, event.StateCompleted,
		event.StateFailed, event.StateRetrying, event.StateContractValidating:
		return true
	case "warning", "preflight",
		"contract_passed", "contract_failed",
		"contract_soft_failure":
		return true
	case event.StateStreamActivity:
		return flags.Verbose
	case event.StateStepProgress:
		return flags.Debug
	case event.StateETAUpdated, event.StateCompactionProgress:
		return flags.Debug
	default:
		return false
	}
}

// noColor returns true when styled output should be suppressed.
func noColor() bool {
	_, ok := os.LookupEnv("NO_COLOR")
	return ok
}

// formatEventLine formats a single event into a display line.
// Mirrors the formatting in BasicProgressDisplay.EmitProgress for CLI parity.
func formatEventLine(evt event.Event) string {
	stepID := evt.StepID
	if stepID == "" {
		stepID = evt.PipelineID
	}

	switch evt.State {
	case event.StateStarted:
		meta := ""
		if evt.Persona != "" || evt.Model != "" {
			parts := []string{}
			if evt.Persona != "" {
				parts = append(parts, evt.Persona)
			}
			if evt.Model != "" {
				parts = append(parts, evt.Model)
			}
			meta = " (" + strings.Join(parts, ", ") + ")"
		}
		return fmt.Sprintf("[%s] Starting...%s", stepID, meta)

	case event.StateCompleted:
		suffix := ""
		if evt.DurationMs > 0 {
			d := time.Duration(evt.DurationMs) * time.Millisecond
			tokenInfo := ""
			if evt.TokensIn > 0 || evt.TokensOut > 0 {
				tokenInfo = fmt.Sprintf(", %s in / %s out", formatTokenCount(evt.TokensIn), formatTokenCount(evt.TokensOut))
			} else if evt.TokensUsed > 0 {
				tokenInfo = fmt.Sprintf(", %s tokens", formatTokenCount(evt.TokensUsed))
			}
			suffix = fmt.Sprintf(" (%s%s)", formatCompactDuration(d), tokenInfo)
		}
		if noColor() {
			return fmt.Sprintf("[%s] Completed%s", stepID, suffix)
		}
		return fmt.Sprintf("[%s] ✓ Completed%s", stepID, suffix)

	case event.StateFailed:
		if noColor() {
			return fmt.Sprintf("[%s] Failed: %s", stepID, evt.Message)
		}
		return fmt.Sprintf("[%s] ✗ Failed: %s", stepID, evt.Message)

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
		if noColor() {
			return fmt.Sprintf("[%s] Warning: %s", stepID, evt.Message)
		}
		return fmt.Sprintf("[%s] ⚠ %s", stepID, evt.Message)

	case event.StateContractValidating:
		phase := evt.ValidationPhase
		if phase == "" {
			phase = "validating"
		}
		return fmt.Sprintf("[%s] Contract: %s", stepID, phase)

	case "contract_passed":
		if noColor() {
			return fmt.Sprintf("[%s] Contract: passed", stepID)
		}
		return fmt.Sprintf("[%s] ✓ Contract: passed", stepID)

	case "contract_failed":
		if noColor() {
			return fmt.Sprintf("[%s] Contract: failed", stepID)
		}
		return fmt.Sprintf("[%s] ✗ Contract: failed", stepID)

	case "contract_soft_failure":
		return fmt.Sprintf("[%s] Contract: soft failure (continuing)", stepID)

	case event.StateStreamActivity:
		target := evt.ToolTarget
		if len(target) > 60 {
			target = target[:57] + "..."
		}
		return fmt.Sprintf("[%s] %s %s", stepID, evt.ToolName, target)

	case event.StateStepProgress:
		if evt.CurrentAction != "" {
			return fmt.Sprintf("[%s] %s", stepID, evt.CurrentAction)
		}
		if evt.TokensIn > 0 || evt.TokensOut > 0 {
			return fmt.Sprintf("[%s] tokens: %s in / %s out", stepID, formatTokenCount(evt.TokensIn), formatTokenCount(evt.TokensOut))
		}
		if evt.Progress > 0 {
			return fmt.Sprintf("[%s] progress: %d%%", stepID, evt.Progress)
		}
		return fmt.Sprintf("[%s] heartbeat", stepID)

	case event.StateETAUpdated:
		if evt.EstimatedTimeMs > 0 {
			d := time.Duration(evt.EstimatedTimeMs) * time.Millisecond
			return fmt.Sprintf("[%s] ETA: ~%s remaining", stepID, formatCompactDuration(d))
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

// formatTokenCount delegates to the shared display.FormatTokenCount formatter.
func formatTokenCount(tokens int) string {
	return display.FormatTokenCount(tokens)
}

// formatCompactDuration delegates to the shared humanize.Duration formatter.
func formatCompactDuration(d time.Duration) string {
	return humanize.Duration(d)
}

// formatErrorBlock formats a failure event as a multi-line error block.
func formatErrorBlock(evt event.Event) string {
	var sb strings.Builder

	if noColor() {
		sb.WriteString("Pipeline failed\n")
	} else {
		sb.WriteString("✗ Pipeline failed\n")
	}

	stepID := evt.StepID
	if stepID == "" {
		stepID = evt.PipelineID
	}
	persona := evt.Persona
	if persona != "" {
		sb.WriteString(fmt.Sprintf("  Step: %s (%s)\n", stepID, persona))
	} else {
		sb.WriteString(fmt.Sprintf("  Step: %s\n", stepID))
	}

	if evt.FailureReason != "" {
		sb.WriteString(fmt.Sprintf("  Reason: %s\n", evt.FailureReason))
	}

	if evt.Remediation != "" {
		sb.WriteString(fmt.Sprintf("  Remediation: %s\n", evt.Remediation))
	}

	if len(evt.RecoveryHints) > 0 {
		sb.WriteString("  Recovery hints:\n")
		for _, hint := range evt.RecoveryHints {
			sb.WriteString(fmt.Sprintf("    → %s\n", hint.Command))
		}
	}

	return sb.String()
}

// formatElapsed formats a duration as MM:SS or HH:MM:SS.
func formatElapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int(d.Seconds())
	if d < time.Hour {
		m := totalSeconds / 60
		s := totalSeconds % 60
		return fmt.Sprintf("%02d:%02d", m, s)
	}
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	s := totalSeconds % 60
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}

// LiveOutputModel renders real-time pipeline output in the right pane.
type LiveOutputModel struct {
	runID        string
	pipelineName string
	input        string
	width        int
	height       int

	buffer        *EventBuffer
	rawEvents     []event.Event
	storedRecords []state.LogRecord // For detached run rebuilds (SQLite-polled events)
	viewport      viewport.Model

	autoScroll bool

	flags DisplayFlags

	currentStep string
	stepNumber  int
	totalSteps  int
	model       string
	startedAt   time.Time

	completed         bool
	completionPending bool

	// Handover tracking for rich output (tree-formatted metadata on step completion)
	handoverInfo map[string]*display.HandoverInfo
	stepOrder    []string

	// Dashboard state
	dashSteps   []*stepDashState
	dashStepMap map[string]*stepDashState
	showLog     bool // true = event log, false = dashboard (default)

	// Persisted-event tailing indicator: true when events come from SQLite
	// polling rather than in-memory PipelineEventMsg (detached/previous-session runs).
	tailingPersisted bool
}

const (
	liveOutputHeaderLines = 7
	liveOutputFooterLines = 2
)

// NewLiveOutputModel creates a new live output model for a running pipeline.
func NewLiveOutputModel(runID, pipelineName string, buffer *EventBuffer, startedAt time.Time, totalSteps int) LiveOutputModel {
	return LiveOutputModel{
		runID:        runID,
		pipelineName: pipelineName,
		buffer:       buffer,
		viewport:     viewport.New(0, 0),
		autoScroll:   true,
		startedAt:    startedAt,
		totalSteps:   totalSteps,
		handoverInfo: make(map[string]*display.HandoverInfo),
		dashStepMap:  make(map[string]*stepDashState),
	}
}

// SetSize updates the viewport dimensions, reserving space for header and footer.
func (m *LiveOutputModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	vpHeight := h - liveOutputHeaderLines - liveOutputFooterLines
	if vpHeight < 0 {
		vpHeight = 0
	}
	m.viewport.Width = w
	m.viewport.Height = vpHeight
	m.updateViewportContent()
}

// updateViewportContent refreshes the viewport content from the buffer or dashboard.
func (m *LiveOutputModel) updateViewportContent() {
	if m.showLog {
		lines := m.buffer.Lines()
		if len(lines) == 0 {
			m.viewport.SetContent("Waiting for events...")
			return
		}
		m.viewport.SetContent(strings.Join(lines, "\n"))
	} else {
		m.viewport.SetContent(m.renderDashboard())
	}
}

// rebuildBuffer clears and rebuilds the display buffer from raw events using current flags.
func (m *LiveOutputModel) rebuildBuffer() {
	m.buffer.head = 0
	m.buffer.count = 0

	if len(m.rawEvents) > 0 {
		// In-process run: rebuild from raw events
		for _, evt := range m.rawEvents {
			if shouldFormat(evt, m.flags) {
				if evt.State == event.StateFailed {
					errorBlock := formatErrorBlock(evt)
					for _, line := range strings.Split(errorBlock, "\n") {
						if line != "" {
							m.buffer.Append(line)
						}
					}
				} else {
					m.buffer.Append(formatEventLine(evt))
				}
				// Re-inject handover tree lines after step completion
				if evt.State == event.StateCompleted && evt.StepID != "" {
					if info, exists := m.handoverInfo[evt.StepID]; exists {
						for _, hl := range display.BuildHandoverLines(evt.StepID, info, m.stepOrder) {
							m.buffer.Append("  " + hl)
						}
					}
				}
			}
		}
	} else if len(m.storedRecords) > 0 {
		// Detached run: rebuild from stored SQLite records
		for _, rec := range m.storedRecords {
			if shouldFormatRecord(rec, m.flags) {
				m.buffer.Append(formatStoredEvent(rec))
			}
		}
	}

	m.updateViewportContent()
}

// getOrCreateDashStep returns the dashboard state for a step, creating it if needed.
func (m *LiveOutputModel) getOrCreateDashStep(stepID string) *stepDashState {
	if s, ok := m.dashStepMap[stepID]; ok {
		return s
	}
	s := &stepDashState{stepID: stepID, status: "not_started"}
	m.dashStepMap[stepID] = s
	m.dashSteps = append(m.dashSteps, s)
	return s
}

// updateDashStepFromEvent updates dashboard state from a live event.Event.
func (m *LiveOutputModel) updateDashStepFromEvent(evt event.Event) {
	if evt.StepID == "" {
		return
	}
	s := m.getOrCreateDashStep(evt.StepID)
	switch evt.State {
	case event.StateStarted:
		s.status = "running"
		s.startedAt = time.Now()
		if evt.Persona != "" {
			s.persona = evt.Persona
		}
		if evt.Model != "" {
			s.model = evt.Model
		}
		if evt.Adapter != "" {
			s.adapter = evt.Adapter
		}
	case event.StateRunning:
		s.status = "running"
		if evt.Message != "" {
			s.message = evt.Message
		}
	case event.StateCompleted:
		s.status = "completed"
		s.durationMs = evt.DurationMs
		s.tokensIn = evt.TokensIn
		s.tokensOut = evt.TokensOut
		s.tokensUsed = evt.TokensUsed
		if len(evt.Artifacts) > 0 {
			s.artifacts = evt.Artifacts
		}
	case event.StateFailed:
		s.status = "failed"
		s.message = evt.Message
		s.durationMs = evt.DurationMs
	case event.StateRetrying:
		s.status = "retrying"
		s.message = evt.Message
	case event.StateStreamActivity:
		s.lastToolName = evt.ToolName
		s.lastToolTarget = evt.ToolTarget
	case event.StateContractValidating:
		s.contractSchema = evt.ValidationPhase
	case "contract_passed":
		s.contractStatus = "passed"
	case "contract_failed":
		s.contractStatus = "failed"
	case "contract_soft_failure":
		s.contractStatus = "soft_failure"
	}
}

// updateStepTrackingFromRecord updates step tracking fields (stepNumber,
// currentStep, totalSteps, stepOrder) from a stored LogRecord. This mirrors
// the step tracking logic in the PipelineEventMsg handler but for SQLite-polled
// records used with detached/previous-session runs.
func (m *LiveOutputModel) updateStepTrackingFromRecord(rec state.LogRecord) {
	if rec.State != event.StateStarted {
		return
	}
	if rec.StepID != "" {
		m.currentStep = rec.StepID
		m.stepNumber++
		// Track step order for handover target resolution
		found := false
		for _, sid := range m.stepOrder {
			if sid == rec.StepID {
				found = true
				break
			}
		}
		if !found {
			m.stepOrder = append(m.stepOrder, rec.StepID)
		}
	}
}

// updateDashStepFromRecord updates dashboard state from a stored LogRecord.
func (m *LiveOutputModel) updateDashStepFromRecord(rec state.LogRecord) {
	if rec.StepID == "" {
		return
	}
	s := m.getOrCreateDashStep(rec.StepID)
	switch rec.State {
	case event.StateStarted:
		s.status = "running"
		s.startedAt = rec.Timestamp
		if rec.Persona != "" {
			s.persona = rec.Persona
		}
	case event.StateRunning:
		s.status = "running"
		if rec.Message != "" {
			s.message = rec.Message
		}
	case event.StateCompleted:
		s.status = "completed"
		s.durationMs = rec.DurationMs
		s.tokensUsed = rec.TokensUsed
	case event.StateFailed:
		s.status = "failed"
		s.message = rec.Message
	case event.StateRetrying:
		s.status = "retrying"
	case event.StateContractValidating:
		s.contractSchema = rec.Message
	case "contract_passed":
		s.contractStatus = "passed"
	case "contract_failed":
		s.contractStatus = "failed"
	case "contract_soft_failure":
		s.contractStatus = "soft_failure"
	case event.StateStreamActivity:
		if rec.Message != "" {
			parts := strings.SplitN(rec.Message, " ", 2)
			s.lastToolName = parts[0]
			if len(parts) > 1 {
				s.lastToolTarget = parts[1]
			}
		}
	}
}

// renderDashboard renders the structured per-step dashboard view.
func (m *LiveOutputModel) renderDashboard() string {
	nc := noColor()

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	failedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	if nc {
		successStyle = lipgloss.NewStyle()
		activeStyle = lipgloss.NewStyle()
		failedStyle = lipgloss.NewStyle()
		mutedStyle = lipgloss.NewStyle()
	}

	var lines []string

	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	for _, s := range m.dashSteps {
		switch s.status {
		case "completed":
			line := fmt.Sprintf("✓ %s", s.stepID)
			if s.persona != "" {
				line += fmt.Sprintf(" (%s)", s.persona)
			}
			if s.model != "" {
				line += fmt.Sprintf(" [%s]", s.model)
			}
			durationText := formatCompactDuration(time.Duration(s.durationMs) * time.Millisecond)
			tokenInfo := ""
			if s.tokensIn > 0 || s.tokensOut > 0 {
				tokenInfo = fmt.Sprintf(", %s in / %s out", formatTokenCount(s.tokensIn), formatTokenCount(s.tokensOut))
			} else if s.tokensUsed > 0 {
				tokenInfo = fmt.Sprintf(", %s tokens", formatTokenCount(s.tokensUsed))
			}
			line += fmt.Sprintf(" (%s%s)", durationText, tokenInfo)
			if nc {
				line = strings.Replace(line, "✓", "*", 1)
			}
			lines = append(lines, successStyle.Render(line))

			// Verbose: show handover metadata
			if m.flags.Verbose {
				if info, exists := m.handoverInfo[s.stepID]; exists {
					for _, hl := range display.BuildHandoverLines(s.stepID, info, m.stepOrder) {
						lines = append(lines, mutedStyle.Render("   "+hl))
					}
				}
			}

		case "running", "retrying":
			frame := (time.Now().UnixMilli() / 80) % int64(len(spinners))
			icon := spinners[frame]
			if nc {
				icon = ">"
			}

			line := fmt.Sprintf("%s %s", icon, s.stepID)
			if s.persona != "" {
				line += fmt.Sprintf(" (%s)", s.persona)
			}
			if !s.startedAt.IsZero() {
				line += fmt.Sprintf(" (%s)", formatElapsed(time.Since(s.startedAt)))
			}
			lines = append(lines, activeStyle.Render(line))

			// Tool activity line
			if s.lastToolName != "" {
				target := s.lastToolTarget
				if len(target) > 60 {
					target = target[:57] + "..."
				}
				toolLine := fmt.Sprintf("   %s %s", s.lastToolName, target)
				lines = append(lines, mutedStyle.Render(toolLine))
			}

		case "failed":
			line := fmt.Sprintf("✗ %s", s.stepID)
			if s.persona != "" {
				line += fmt.Sprintf(" (%s)", s.persona)
			}
			if s.durationMs > 0 {
				line += fmt.Sprintf(" (%s)", formatCompactDuration(time.Duration(s.durationMs)*time.Millisecond))
			}
			if nc {
				line = strings.Replace(line, "✗", "x", 1)
			}
			lines = append(lines, failedStyle.Render(line))

		default: // not_started
			line := fmt.Sprintf("○ %s", s.stepID)
			if s.persona != "" {
				line += fmt.Sprintf(" (%s)", s.persona)
			}
			lines = append(lines, mutedStyle.Render(line))
		}
	}

	if len(lines) == 0 {
		return "Waiting for pipeline to start..."
	}

	return strings.Join(lines, "\n")
}

// Update handles messages for the live output model.
func (m LiveOutputModel) Update(msg tea.Msg) (LiveOutputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case PipelineEventMsg:
		if msg.RunID != m.runID {
			return m, nil
		}
		evt := msg.Event

		// Update step tracking on started events
		if evt.State == event.StateStarted {
			if evt.TotalSteps > 0 {
				m.totalSteps = evt.TotalSteps
			}
			if evt.StepID != "" {
				m.currentStep = evt.StepID
				m.stepNumber++
				if evt.Model != "" {
					m.model = evt.Model
				}
				// Track step order for handover target resolution
				found := false
				for _, sid := range m.stepOrder {
					if sid == evt.StepID {
						found = true
						break
					}
				}
				if !found {
					m.stepOrder = append(m.stepOrder, evt.StepID)
				}
			}
		}

		// Accumulate handover metadata from contract events
		if evt.StepID != "" {
			switch evt.State {
			case event.StateContractValidating:
				if _, exists := m.handoverInfo[evt.StepID]; !exists {
					m.handoverInfo[evt.StepID] = &display.HandoverInfo{}
				}
				m.handoverInfo[evt.StepID].ContractSchema = evt.ValidationPhase
			case "contract_passed":
				if _, exists := m.handoverInfo[evt.StepID]; !exists {
					m.handoverInfo[evt.StepID] = &display.HandoverInfo{}
				}
				m.handoverInfo[evt.StepID].ContractStatus = "passed"
			case "contract_failed":
				if _, exists := m.handoverInfo[evt.StepID]; !exists {
					m.handoverInfo[evt.StepID] = &display.HandoverInfo{}
				}
				m.handoverInfo[evt.StepID].ContractStatus = "failed"
			case "contract_soft_failure":
				if _, exists := m.handoverInfo[evt.StepID]; !exists {
					m.handoverInfo[evt.StepID] = &display.HandoverInfo{}
				}
				m.handoverInfo[evt.StepID].ContractStatus = "soft_failure"
			}
		}

		// Handle terminal events
		if evt.State == event.StateCompleted && evt.StepID == "" {
			m.rawEvents = append(m.rawEvents, evt)
			// Pipeline-level completion
			duration := time.Since(m.startedAt)
			var summaryLine string
			if noColor() {
				summaryLine = fmt.Sprintf("Pipeline completed in %s", formatElapsed(duration))
			} else {
				summaryLine = fmt.Sprintf("✓ Pipeline completed in %s", formatElapsed(duration))
			}
			m.buffer.Append(summaryLine)
			m.completed = true
			m.updateViewportContent()
			if m.autoScroll {
				m.viewport.GotoBottom()
				return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
					return TransitionTimerMsg{RunID: m.runID}
				})
			}
			m.completionPending = true
			return m, nil
		}

		if evt.State == event.StateFailed && evt.StepID == "" {
			m.rawEvents = append(m.rawEvents, evt)
			// Pipeline-level failure
			errorBlock := formatErrorBlock(evt)
			for _, line := range strings.Split(errorBlock, "\n") {
				if line != "" {
					m.buffer.Append(line)
				}
			}
			m.completed = true
			m.updateViewportContent()
			if m.autoScroll {
				m.viewport.GotoBottom()
				return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
					return TransitionTimerMsg{RunID: m.runID}
				})
			}
			m.completionPending = true
			return m, nil
		}

		// Store raw event for flag-toggle rebuilds
		m.rawEvents = append(m.rawEvents, evt)

		// Update dashboard state
		m.updateDashStepFromEvent(evt)

		// Format and append event line
		if shouldFormat(evt, m.flags) {
			if evt.State == event.StateFailed {
				errorBlock := formatErrorBlock(evt)
				for _, line := range strings.Split(errorBlock, "\n") {
					if line != "" {
						m.buffer.Append(line)
					}
				}
			} else {
				line := formatEventLine(evt)
				m.buffer.Append(line)
			}

			// On step completion, capture artifacts and render handover tree
			if evt.State == event.StateCompleted && evt.StepID != "" {
				if len(evt.Artifacts) > 0 {
					if _, exists := m.handoverInfo[evt.StepID]; !exists {
						m.handoverInfo[evt.StepID] = &display.HandoverInfo{}
					}
					m.handoverInfo[evt.StepID].ArtifactPaths = evt.Artifacts
				}
				if info, exists := m.handoverInfo[evt.StepID]; exists {
					for _, hl := range display.BuildHandoverLines(evt.StepID, info, m.stepOrder) {
						m.buffer.Append("  " + hl)
					}
				}
			}

			m.updateViewportContent()
			if m.autoScroll {
				m.viewport.GotoBottom()
			}
		}

		// In dashboard mode, start tick for live elapsed time
		if !m.showLog && !m.completed {
			return m, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
				return DashboardTickMsg{}
			})
		}
		return m, nil

	case DashboardTickMsg:
		if !m.showLog && !m.completed {
			m.updateViewportContent()
			return m, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
				return DashboardTickMsg{}
			})
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "l":
			m.showLog = !m.showLog
			if m.showLog {
				m.rebuildBuffer()
			}
			m.updateViewportContent()
			if m.autoScroll {
				m.viewport.GotoBottom()
			}
			return m, nil
		case "v":
			m.flags.Verbose = !m.flags.Verbose
			m.rebuildBuffer()
			if m.autoScroll {
				m.viewport.GotoBottom()
			}
			return m, nil
		case "d":
			m.flags.Debug = !m.flags.Debug
			m.rebuildBuffer()
			if m.autoScroll {
				m.viewport.GotoBottom()
			}
			return m, nil
		case "o":
			m.flags.OutputOnly = !m.flags.OutputOnly
			m.rebuildBuffer()
			if m.autoScroll {
				m.viewport.GotoBottom()
			}
			return m, nil
		case "up", "down", "pgup", "pgdown":
			m.autoScroll = false
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			if m.viewport.AtBottom() {
				m.autoScroll = true
				// If completion is pending, start the transition timer
				if m.completionPending {
					m.completionPending = false
					return m, tea.Batch(cmd, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
						return TransitionTimerMsg{RunID: m.runID}
					}))
				}
			}
			return m, cmd
		}
	}

	return m, nil
}

// View renders the live output view with header, viewport, and footer.
func (m LiveOutputModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	nc := noColor()

	// Header
	header := m.renderHeader(nc)

	// Viewport
	vpView := m.viewport.View()

	// Footer
	footer := m.renderFooter(nc)

	return lipgloss.JoinVertical(lipgloss.Left, header, vpView, footer)
}

func (m LiveOutputModel) renderHeader(nc bool) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	if nc {
		titleStyle = lipgloss.NewStyle()
		labelStyle = lipgloss.NewStyle()
	}

	// Line 1: Pipeline label
	line1 := fmt.Sprintf("%s %s", labelStyle.Render("Pipeline:"), titleStyle.Render(m.pipelineName))

	// Line 2: Status with step progress and completion counts
	var statusStr string
	switch {
	case m.completed:
		statusStr = "Finished"
	case m.tailingPersisted:
		statusStr = "▶ Tailing persisted events"
		if m.stepNumber > 0 {
			statusStr = fmt.Sprintf("▶ Tailing persisted events (step %d/%d: %s)", m.stepNumber, m.totalSteps, m.currentStep)
		}
	case m.stepNumber > 0:
		statusStr = fmt.Sprintf("▶ Running (step %d/%d: %s)", m.stepNumber, m.totalSteps, m.currentStep)
		var okCount, failCount int
		for _, s := range m.dashSteps {
			switch s.status {
			case "completed":
				okCount++
			case "failed":
				failCount++
			}
		}
		if okCount > 0 || failCount > 0 {
			counts := fmt.Sprintf("%d ok", okCount)
			if failCount > 0 {
				counts += fmt.Sprintf(", %d fail", failCount)
			}
			statusStr += fmt.Sprintf(" (%s)", counts)
		}
	default:
		statusStr = "▶ Running"
	}
	elapsed := formatElapsed(time.Since(m.startedAt))
	if m.model != "" {
		statusStr += "  " + m.model
	}
	line2 := fmt.Sprintf("%s %s", labelStyle.Render("Status:"), statusStr)

	// Build metadata lines (blank line after pipeline name, matching finished detail)
	var lines []string
	lines = append(lines, line1, "", line2)
	if m.input != "" {
		lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("Input:"), m.input))
	}
	lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("RunID:"), m.runID))
	if !m.startedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("%s %s  %s %s",
			labelStyle.Render("Started:"), m.startedAt.Format("2006-01-02 15:04:05"),
			labelStyle.Render("Elapsed:"), elapsed))
	}
	// Separator
	lines = append(lines, labelStyle.Render(strings.Repeat("─", m.width)))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m LiveOutputModel) renderFooter(nc bool) string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	if nc {
		labelStyle = lipgloss.NewStyle()
	}

	// Line 1: separator
	line1 := labelStyle.Render(strings.Repeat("─", m.width))

	// Line 2: display flags and auto-scroll status
	var parts []string

	verboseFlag := "[ ] verbose"
	if m.flags.Verbose {
		verboseFlag = "[v] verbose"
	}
	debugFlag := "[ ] debug"
	if m.flags.Debug {
		debugFlag = "[d] debug"
	}
	outputFlag := "[ ] output-only"
	if m.flags.OutputOnly {
		outputFlag = "[o] output-only"
	}
	logFlag := "[ ] log"
	if m.showLog {
		logFlag = "[l] log"
	}
	parts = append(parts, verboseFlag, debugFlag, outputFlag, logFlag)

	flagsStr := strings.Join(parts, "  ")

	if m.tailingPersisted && !m.completed {
		flagsStr += "  |  tailing from SQLite"
	}

	if !m.autoScroll {
		if nc {
			flagsStr += "  |  Scrolling paused -- scroll to bottom to resume"
		} else {
			flagsStr += "  |  ⏸ Scrolling paused — scroll to bottom to resume"
		}
	}

	line2 := labelStyle.Render(flagsStr)

	return lipgloss.JoinVertical(lipgloss.Left, line1, line2)
}
