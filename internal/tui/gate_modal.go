package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/recinq/wave/internal/pipeline"
)

// GatePromptMsg is sent to the Bubble Tea program when a gate needs user input.
// The TUI renders the gate modal and sends the decision back through the response channel.
type GatePromptMsg struct {
	Gate     *pipeline.GateConfig
	Response chan<- gateResponse
}

// GateDismissMsg signals the TUI to close the gate modal.
type GateDismissMsg struct{}

// gateResponse carries the user's decision or an error back to the blocking handler.
type gateResponse struct {
	decision *pipeline.GateDecision
	err      error
}

// TUIGateHandler implements pipeline.GateHandler by bridging to the Bubble Tea
// event loop. When Prompt is called (from the executor goroutine), it sends a
// GatePromptMsg to the TUI program and blocks until the user makes a choice.
// The TUI renders the gate modal and delivers the decision through a channel.
type TUIGateHandler struct {
	program *tea.Program
}

// NewTUIGateHandler creates a gate handler that sends prompts to the given
// Bubble Tea program. The program must handle GatePromptMsg in its Update loop.
func NewTUIGateHandler(program *tea.Program) *TUIGateHandler {
	return &TUIGateHandler{program: program}
}

// Prompt sends the gate to the TUI for display and blocks until the user
// responds or the context is cancelled.
func (h *TUIGateHandler) Prompt(ctx context.Context, gate *pipeline.GateConfig) (*pipeline.GateDecision, error) {
	if h.program == nil {
		return nil, fmt.Errorf("TUI gate handler has no program reference")
	}

	ch := make(chan gateResponse, 1)

	h.program.Send(GatePromptMsg{
		Gate:     gate,
		Response: ch,
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-ch:
		return resp.decision, resp.err
	}
}

// GateModalModel is a Bubble Tea component that renders a gate approval modal.
// It displays the gate prompt, choices, and optional freeform input, then sends
// the decision back through the response channel.
type GateModalModel struct {
	active   bool
	gate     *pipeline.GateConfig
	response chan<- gateResponse

	cursor   int    // Index into gate.Choices
	freeform string // Freeform text input buffer
	editing  bool   // Whether freeform input is active
	width    int
	height   int
}

// NewGateModalModel creates an inactive gate modal. Activate it by calling
// Show with a GatePromptMsg.
func NewGateModalModel() GateModalModel {
	return GateModalModel{}
}

// Active returns whether the modal is currently displayed.
func (m GateModalModel) Active() bool {
	return m.active
}

// Show activates the modal with the given gate prompt.
func (m *GateModalModel) Show(msg GatePromptMsg) {
	m.active = true
	m.gate = msg.Gate
	m.response = msg.Response
	m.cursor = 0
	m.freeform = ""
	m.editing = false

	// Pre-select the default choice if one is configured.
	if m.gate.Default != "" {
		for i, c := range m.gate.Choices {
			if c.Key == m.gate.Default {
				m.cursor = i
				break
			}
		}
	}
}

// SetSize updates the modal's available dimensions for centering.
func (m *GateModalModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles key events for the gate modal. Returns the updated model and
// an optional command. When the user confirms a choice, a GateDismissMsg is
// returned so the parent can deactivate the modal.
func (m GateModalModel) Update(msg tea.Msg) (GateModalModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		if m.editing {
			return m.updateFreeformInput(msg)
		}

		switch msg.Type {
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.gate.Choices)-1 {
				m.cursor++
			}
		case tea.KeyEnter:
			return m.confirmChoice()
		case tea.KeyEsc:
			return m.cancel()
		case tea.KeyRunes:
			key := string(msg.Runes)
			for i, c := range m.gate.Choices {
				if c.Key == key {
					m.cursor = i
					return m.confirmChoice()
				}
			}
		}
	}

	return m, nil
}

// updateFreeformInput handles key events during freeform text editing.
func (m GateModalModel) updateFreeformInput(msg tea.KeyMsg) (GateModalModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Submit the freeform text and finalize the choice.
		return m.finalize()
	case tea.KeyEsc:
		// Cancel freeform editing, return to choice selection.
		m.editing = false
		m.freeform = ""
		return m, nil
	case tea.KeyBackspace:
		if len(m.freeform) > 0 {
			m.freeform = m.freeform[:len(m.freeform)-1]
		}
	case tea.KeyRunes:
		m.freeform += string(msg.Runes)
	case tea.KeySpace:
		m.freeform += " "
	}
	return m, nil
}

// confirmChoice handles Enter on a selected choice. If freeform is enabled,
// transitions to freeform input mode; otherwise finalizes immediately.
func (m GateModalModel) confirmChoice() (GateModalModel, tea.Cmd) {
	if m.gate == nil {
		return m.cancel()
	}
	if m.gate.Freeform {
		m.editing = true
		return m, nil
	}
	return m.finalize()
}

// cancel aborts the gate interaction and sends a cancellation error.
func (m GateModalModel) cancel() (GateModalModel, tea.Cmd) {
	if m.response != nil {
		m.response <- gateResponse{
			err: fmt.Errorf("gate cancelled by user"),
		}
	}
	m.active = false
	m.gate = nil
	m.response = nil
	return m, func() tea.Msg { return GateDismissMsg{} }
}

// finalize sends the selected choice (and optional freeform text) back through
// the response channel and deactivates the modal.
func (m GateModalModel) finalize() (GateModalModel, tea.Cmd) {
	if m.gate == nil || m.cursor >= len(m.gate.Choices) {
		return m.cancel()
	}

	choice := m.gate.Choices[m.cursor]

	if m.response != nil {
		m.response <- gateResponse{
			decision: &pipeline.GateDecision{
				Choice:    choice.Key,
				Label:     choice.Label,
				Text:      strings.TrimSpace(m.freeform),
				Timestamp: time.Now(),
				Target:    choice.Target,
			},
		}
	}

	m.active = false
	m.gate = nil
	m.response = nil
	return m, func() tea.Msg { return GateDismissMsg{} }
}

// View renders the gate modal as a centered overlay.
func (m GateModalModel) View() string {
	if !m.active || m.gate == nil {
		return ""
	}

	var (
		cyan  = lipgloss.Color("6")
		white = lipgloss.Color("7")
		muted = lipgloss.Color("244")
		red   = lipgloss.Color("1")
	)

	titleStyle := lipgloss.NewStyle().
		Foreground(cyan).
		Bold(true).
		MarginBottom(1)

	promptStyle := lipgloss.NewStyle().
		Foreground(white).
		MarginBottom(1)

	choiceStyle := lipgloss.NewStyle().
		Foreground(white)

	selectedStyle := lipgloss.NewStyle().
		Foreground(cyan).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(muted)

	abortStyle := lipgloss.NewStyle().
		Foreground(red)

	hintStyle := lipgloss.NewStyle().
		Foreground(muted).
		MarginTop(1)

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("Gate Approval"))
	b.WriteString("\n")

	// Prompt message
	prompt := m.gate.Prompt
	if prompt == "" {
		prompt = m.gate.Message
	}
	if prompt != "" {
		b.WriteString(promptStyle.Render(prompt))
		b.WriteString("\n")
	}

	// Choices
	for i, c := range m.gate.Choices {
		prefix := "  "
		style := choiceStyle
		if i == m.cursor {
			prefix = "> "
			style = selectedStyle
		}

		label := fmt.Sprintf("%s[%s] %s", prefix, c.Key, c.Label)
		if c.Target == "_fail" {
			label = fmt.Sprintf("%s[%s] %s", prefix, c.Key, abortStyle.Render(c.Label+" (abort)"))
			if i == m.cursor {
				label = fmt.Sprintf("%s[%s] %s", prefix, c.Key, abortStyle.Bold(true).Render(c.Label+" (abort)"))
			}
		} else {
			label = style.Render(label)
		}

		b.WriteString(label)
		b.WriteString("\n")
	}

	// Freeform input area
	if m.editing {
		b.WriteString("\n")
		inputLabel := lipgloss.NewStyle().Foreground(cyan).Render("Notes: ")
		cursor := lipgloss.NewStyle().Foreground(cyan).Render("_")
		b.WriteString(inputLabel + m.freeform + cursor)
		b.WriteString("\n")
		b.WriteString(hintStyle.Render("Enter: submit  Esc: cancel"))
	} else {
		// Key hints
		keys := make([]string, 0, len(m.gate.Choices))
		for _, c := range m.gate.Choices {
			keys = append(keys, c.Key)
		}
		b.WriteString(hintStyle.Render(
			fmt.Sprintf("↑↓: select  Enter: confirm  %s: quick-select  Esc: cancel",
				keyStyle.Render(strings.Join(keys, "/"))),
		))
	}

	content := b.String()

	// Wrap in a bordered box
	modalWidth := 50
	if m.width > 0 && m.width < modalWidth+4 {
		modalWidth = m.width - 4
	}
	if modalWidth < 30 {
		modalWidth = 30
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyan).
		Padding(1, 2).
		Width(modalWidth)

	rendered := boxStyle.Render(content)

	// Center the modal horizontally and vertically if dimensions are known.
	if m.width > 0 && m.height > 0 {
		rendered = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, rendered)
	}

	return rendered
}
