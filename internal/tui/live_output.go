package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/recinq/wave/internal/event"
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

// DisplayFlags tracks which event categories are visible in the live output.
type DisplayFlags struct {
	Verbose    bool
	Debug      bool
	OutputOnly bool
}

// shouldFormat determines whether an event should be formatted into the buffer
// based on the current display flags.
func shouldFormat(evt event.Event, flags DisplayFlags) bool {
	if flags.OutputOnly {
		return evt.State == event.StateCompleted || evt.State == event.StateFailed
	}
	switch evt.State {
	case event.StateStarted, event.StateRunning, event.StateCompleted,
		event.StateFailed, event.StateContractValidating:
		return true
	case event.StateStreamActivity:
		return flags.Verbose
	case event.StateStepProgress, event.StateETAUpdated, event.StateCompactionProgress:
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
				parts = append(parts, "persona: "+evt.Persona)
			}
			if evt.Model != "" {
				parts = append(parts, "model: "+evt.Model)
			}
			meta = " (" + strings.Join(parts, ", ") + ")"
		}
		return fmt.Sprintf("[%s] Starting...%s", stepID, meta)

	case event.StateCompleted:
		duration := ""
		if evt.DurationMs > 0 {
			d := time.Duration(evt.DurationMs) * time.Millisecond
			duration = fmt.Sprintf(" (%s)", formatCompactDuration(d))
		}
		if noColor() {
			return fmt.Sprintf("[%s] Completed%s", stepID, duration)
		}
		return fmt.Sprintf("[%s] ✓ Completed%s", stepID, duration)

	case event.StateFailed:
		if noColor() {
			return fmt.Sprintf("[%s] Failed: %s", stepID, evt.Message)
		}
		return fmt.Sprintf("[%s] ✗ Failed: %s", stepID, evt.Message)

	case event.StateRunning:
		if evt.Message != "" {
			return fmt.Sprintf("[%s] %s", stepID, evt.Message)
		}
		return fmt.Sprintf("[%s] Running...", stepID)

	case event.StateContractValidating:
		phase := evt.ValidationPhase
		if phase == "" {
			phase = "validating"
		}
		return fmt.Sprintf("[%s] Contract validation: %s", stepID, phase)

	case event.StateStreamActivity:
		target := evt.ToolTarget
		if len(target) > 60 {
			target = target[:57] + "..."
		}
		return fmt.Sprintf("[%s] %s %s", stepID, evt.ToolName, target)

	case event.StateStepProgress:
		if evt.TokensIn > 0 || evt.TokensOut > 0 {
			if noColor() {
				return fmt.Sprintf("[%s] heartbeat (tokens: %d/%d)", stepID, evt.TokensOut, evt.TokensIn)
			}
			return fmt.Sprintf("[%s] ♡ heartbeat (tokens: %d/%d)", stepID, evt.TokensOut, evt.TokensIn)
		}
		return fmt.Sprintf("[%s] progress: %d%%", stepID, evt.Progress)

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

// formatCompactDuration formats a duration as a compact string (e.g., "42s", "1m23s").
func formatCompactDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%02dm", h, m)
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
	width        int
	height       int

	buffer   *EventBuffer
	viewport viewport.Model

	autoScroll bool

	flags DisplayFlags

	currentStep string
	stepNumber  int
	totalSteps  int
	model       string
	startedAt   time.Time

	completed         bool
	completionPending bool
}

const (
	liveOutputHeaderLines = 3
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

// updateViewportContent refreshes the viewport content from the buffer.
func (m *LiveOutputModel) updateViewportContent() {
	lines := m.buffer.Lines()
	if len(lines) == 0 {
		m.viewport.SetContent("Waiting for events...")
		return
	}
	m.viewport.SetContent(strings.Join(lines, "\n"))
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
		if evt.State == event.StateStarted && evt.StepID != "" {
			m.currentStep = evt.StepID
			m.stepNumber++
			if evt.Model != "" {
				m.model = evt.Model
			}
			if evt.TotalSteps > 0 {
				m.totalSteps = evt.TotalSteps
			}
		}

		// Handle terminal events
		if evt.State == event.StateCompleted && evt.StepID == "" {
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
			m.updateViewportContent()
			if m.autoScroll {
				m.viewport.GotoBottom()
			}
		}

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "v":
			m.flags.Verbose = !m.flags.Verbose
			return m, nil
		case "d":
			m.flags.Debug = !m.flags.Debug
			return m, nil
		case "o":
			m.flags.OutputOnly = !m.flags.OutputOnly
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

	// Line 1: Pipeline name
	line1 := titleStyle.Render(m.pipelineName)

	// Line 2: Status with step progress
	var statusParts []string
	if m.completed {
		statusParts = append(statusParts, "Finished")
	} else if m.stepNumber > 0 {
		statusParts = append(statusParts, fmt.Sprintf("Running (step %d/%d: %s)", m.stepNumber, m.totalSteps, m.currentStep))
	} else {
		statusParts = append(statusParts, "Running")
	}
	elapsed := formatElapsed(time.Since(m.startedAt))
	statusParts = append(statusParts, elapsed)
	if m.model != "" {
		statusParts = append(statusParts, m.model)
	}

	line2 := labelStyle.Render(strings.Join(statusParts, "  "))

	// Line 3: separator
	line3 := labelStyle.Render(strings.Repeat("─", m.width))

	return lipgloss.JoinVertical(lipgloss.Left, line1, line2, line3)
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
	parts = append(parts, verboseFlag, debugFlag, outputFlag)

	flagsStr := strings.Join(parts, "  ")

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
