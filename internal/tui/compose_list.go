package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/recinq/wave/internal/pipeline"
)

// ComposeListModel is the Bubble Tea model for the sequence builder list
// (left pane in compose mode).
type ComposeListModel struct {
	width       int
	height      int
	focused     bool
	sequence    Sequence
	cursor      int
	picking     bool
	picker      *huh.Form
	pickerValue string
	available   []PipelineInfo
	validation  CompatibilityResult
	confirming  bool // T026: inline confirmation for incompatible sequences
}

// NewComposeListModel creates a new compose list model. The initial pipeline
// is added as the first entry in the sequence, and available provides the
// list of pipelines that can be appended via the picker.
func NewComposeListModel(initial PipelineInfo, initialPipeline *pipeline.Pipeline, available []PipelineInfo) ComposeListModel {
	m := ComposeListModel{
		available: available,
	}

	m.sequence.Add(initial.Name, initialPipeline)
	m.validation = ValidateSequence(m.sequence)

	return m
}

// Init implements tea.Model. No async init needed.
func (m ComposeListModel) Init() tea.Cmd {
	return nil
}

// Update handles messages to update compose list state.
func (m ComposeListModel) Update(msg tea.Msg) (ComposeListModel, tea.Cmd) {
	// When picking, forward ALL messages to the picker form first.
	if m.picking && m.picker != nil {
		model, cmd := m.picker.Update(msg)
		m.picker = model.(*huh.Form)

		if m.picker.State == huh.StateCompleted {
			// Find selected pipeline from available list.
			// We only have PipelineInfo metadata; store nil Pipeline.
			// The full Pipeline struct can be loaded externally if needed.
			m.sequence.Add(m.pickerValue, nil)
			m.picking = false
			m.picker = nil
			m.pickerValue = ""
			m.validation = ValidateSequence(m.sequence)
			return m, tea.Batch(cmd, m.emitSequenceChanged())
		}

		if m.picker.State == huh.StateAborted {
			m.picking = false
			m.picker = nil
			m.pickerValue = ""
			return m, cmd
		}

		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}
		return m.handleKeyMsg(msg)
	}

	return m, nil
}

// handleKeyMsg processes keyboard input when the model is focused.
func (m ComposeListModel) handleKeyMsg(msg tea.KeyMsg) (ComposeListModel, tea.Cmd) {
	// T026: When confirming, handle the confirmation prompt keys.
	if m.confirming {
		switch msg.Type {
		case tea.KeyEnter:
			m.confirming = false
			return m, func() tea.Msg {
				return ComposeStartMsg{Sequence: m.sequence}
			}
		case tea.KeyEscape:
			m.confirming = false
			return m, nil
		default:
			m.confirming = false
			return m, nil
		}
	}

	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case tea.KeyDown:
		if m.cursor < m.sequence.Len()-1 {
			m.cursor++
		}
		return m, nil

	case tea.KeyShiftUp:
		if m.cursor > 0 {
			m.sequence.MoveUp(m.cursor)
			m.cursor--
			m.validation = ValidateSequence(m.sequence)
			return m, m.emitSequenceChanged()
		}
		return m, nil

	case tea.KeyShiftDown:
		if m.cursor < m.sequence.Len()-1 {
			m.sequence.MoveDown(m.cursor)
			m.cursor++
			m.validation = ValidateSequence(m.sequence)
			return m, m.emitSequenceChanged()
		}
		return m, nil

	case tea.KeyEnter:
		if m.sequence.IsEmpty() {
			return m, nil
		}
		// T026: If sequence has incompatibilities, show confirmation prompt.
		if !m.validation.IsReady() {
			m.confirming = true
			return m, nil
		}
		return m, func() tea.Msg {
			return ComposeStartMsg{Sequence: m.sequence}
		}

	case tea.KeyEscape:
		return m, func() tea.Msg {
			return ComposeCancelMsg{}
		}

	default:
		switch msg.String() {
		case "a":
			if len(m.available) == 0 {
				return m, nil
			}
			return m.enterPickingMode()

		case "x":
			if m.sequence.IsEmpty() {
				return m, nil
			}
			m.sequence.Remove(m.cursor)
			// Adjust cursor if it now exceeds bounds.
			if m.cursor >= m.sequence.Len() && m.cursor > 0 {
				m.cursor = m.sequence.Len() - 1
			}
			m.validation = ValidateSequence(m.sequence)
			return m, m.emitSequenceChanged()
		}
	}

	return m, nil
}

// enterPickingMode creates a huh.Select picker form and starts picking mode.
func (m ComposeListModel) enterPickingMode() (ComposeListModel, tea.Cmd) {
	options := make([]huh.Option[string], len(m.available))
	for i, p := range m.available {
		options[i] = huh.NewOption(p.Name, p.Name)
	}

	m.pickerValue = ""
	m.picker = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Add pipeline").
				Options(options...).
				Value(&m.pickerValue),
		),
	).WithTheme(WaveTheme())

	initCmd := m.picker.Init()
	m.picking = true

	return m, initCmd
}

// emitSequenceChanged returns a command that emits ComposeSequenceChangedMsg.
func (m ComposeListModel) emitSequenceChanged() tea.Cmd {
	seq := m.sequence
	val := m.validation
	return func() tea.Msg {
		return ComposeSequenceChangedMsg{
			Sequence:   seq,
			Validation: val,
		}
	}
}

// View renders the compose list pane.
func (m ComposeListModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	normalStyle := lipgloss.NewStyle()
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

	var lines []string

	// Title line.
	lines = append(lines, titleStyle.Render("Compose Sequence"))

	if m.sequence.IsEmpty() {
		lines = append(lines, "")
		lines = append(lines, mutedStyle.Render("No pipelines in sequence. Press 'a' to add."))
	} else {
		// Build a map of boundary statuses keyed by target index.
		// Flows[i] represents the boundary between entry i and entry i+1,
		// so the status icon is shown next to entry i+1.
		boundaryStatus := make(map[int]CompatibilityStatus)
		for i, flow := range m.validation.Flows {
			targetIdx := i + 1
			hasError := false
			hasWarning := false
			for _, match := range flow.Matches {
				if match.Status == MatchMissing && !match.Optional {
					hasError = true
				} else if match.Status == MatchMissing && match.Optional {
					hasWarning = true
				}
			}
			if hasError {
				boundaryStatus[targetIdx] = CompatibilityError
			} else if hasWarning {
				boundaryStatus[targetIdx] = CompatibilityWarning
			} else {
				boundaryStatus[targetIdx] = CompatibilityValid
			}
		}

		// T025: Count pipeline name occurrences for duplicate detection.
		nameCounts := make(map[string]int)
		for _, entry := range m.sequence.Entries {
			nameCounts[entry.PipelineName]++
		}

		for i, entry := range m.sequence.Entries {
			isSelected := i == m.cursor
			prefix := "  "
			if isSelected {
				prefix = cursorStyle.Render("▸ ")
			}

			// Status icon for entries after the first (boundary indicator).
			statusIcon := ""
			if i > 0 {
				switch boundaryStatus[i] {
				case CompatibilityValid:
					statusIcon = " " + greenStyle.Render("✓")
				case CompatibilityWarning:
					statusIcon = " " + yellowStyle.Render("~")
				case CompatibilityError:
					statusIcon = " " + redStyle.Render("✗")
				}
			}

			// T025: Duplicate indicator when the same pipeline appears more than once.
			dupIndicator := ""
			if nameCounts[entry.PipelineName] > 1 {
				dupIndicator = " " + mutedStyle.Render("(duplicate)")
			}

			indexStr := fmt.Sprintf("%d. ", i+1)
			var line string
			if isSelected {
				line = prefix + cursorStyle.Render(indexStr+entry.PipelineName) + dupIndicator + statusIcon
			} else {
				line = prefix + normalStyle.Render(indexStr+entry.PipelineName) + dupIndicator + statusIcon
			}
			lines = append(lines, line)
		}
	}

	// Picker overlay when in picking mode.
	if m.picking && m.picker != nil {
		lines = append(lines, "")
		lines = append(lines, m.picker.View())
	}

	// T026: Inline confirmation prompt for incompatible sequences.
	if m.confirming {
		lines = append(lines, "")
		lines = append(lines, yellowStyle.Render("Sequence has incompatibilities. Press Enter again to confirm, or Esc to cancel."))
	}

	// Status line at bottom.
	lines = append(lines, "")
	lines = append(lines, m.renderStatusLine(mutedStyle, greenStyle, redStyle, yellowStyle))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderStatusLine renders the overall compatibility status.
func (m ComposeListModel) renderStatusLine(
	mutedStyle, greenStyle, redStyle, yellowStyle lipgloss.Style,
) string {
	if m.sequence.IsEmpty() {
		return mutedStyle.Render("Status: empty sequence")
	}

	if m.sequence.IsSingle() {
		return mutedStyle.Render("Status: single pipeline — add more to compose")
	}

	errorCount := 0
	warningCount := 0
	for _, diag := range m.validation.Diagnostics {
		if strings.Contains(diag, "missing required") {
			errorCount++
		} else {
			warningCount++
		}
	}

	switch m.validation.Status {
	case CompatibilityValid:
		return greenStyle.Render("Status: all flows compatible ✓")
	case CompatibilityWarning:
		return yellowStyle.Render(fmt.Sprintf("Status: %d warning(s)", warningCount))
	case CompatibilityError:
		msg := fmt.Sprintf("Status: %d error(s) found", errorCount)
		if warningCount > 0 {
			msg += fmt.Sprintf(", %d warning(s)", warningCount)
		}
		return redStyle.Render(msg)
	}

	return ""
}

// SetSize updates the model dimensions.
func (m *ComposeListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetFocused updates the focused state.
func (m *ComposeListModel) SetFocused(focused bool) {
	m.focused = focused
}
