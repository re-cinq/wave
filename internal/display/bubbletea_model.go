package display

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Unified color palette — semantic names for consistent theming
var (
	colorPrimary     = lipgloss.Color("14")  // Bright cyan — active/accent
	colorSuccess     = lipgloss.Color("10")  // Bright green — completed
	colorError       = lipgloss.Color("9")   // Bright red — failed
	colorMuted       = lipgloss.Color("244") // Medium gray — metadata/inactive
	colorDim         = lipgloss.Color("240") // Dark gray — empty/not-started
	colorInfo        = lipgloss.Color("7")   // Light gray — project info
	colorShimmerCore = lipgloss.Color("15")  // White — shimmer center
	colorShimmerMid  = lipgloss.Color("14")  // Bright cyan — shimmer fringe
	colorShimmerBase = lipgloss.Color("14")  // Bright cyan — shimmer ambient
)

// ProgressModel implements the bubbletea model for Wave progress display
type ProgressModel struct {
	ctx        *PipelineContext
	quit       bool
	lastUpdate time.Time
}

// TickMsg represents a regular update tick
type TickMsg time.Time

// UpdateContextMsg represents a context update from the pipeline
type UpdateContextMsg *PipelineContext

// NewProgressModel creates a new bubbletea model for progress display
func NewProgressModel(ctx *PipelineContext) *ProgressModel {
	return &ProgressModel{
		ctx:        ctx,
		quit:       false,
		lastUpdate: time.Now(),
	}
}

// Init implements tea.Model
func (m *ProgressModel) Init() tea.Cmd {
	return tea.Batch(tea.ClearScreen, tickCmd())
}

// Update implements tea.Model
func (m *ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		}

	case TickMsg:
		m.lastUpdate = time.Time(msg)
		return m, tickCmd()

	case UpdateContextMsg:
		m.ctx = msg
		return m, nil
	}

	return m, nil
}

// View implements tea.Model
func (m *ProgressModel) View() string {
	if m.ctx == nil {
		return "Initializing pipeline..."
	}

	// Header with logo and project info (has spacing built-in)
	header := RenderPipelineHeader(m.ctx)

	// Current step with spacing
	currentStep := RenderPipelineSteps(m.ctx)

	// Main content area
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"", // Empty line for spacing
		currentStep,
		"", // Empty line before buttons
		"", // Another empty line
	)

	// Bottom status line with readable colors
	statusLine := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render("Press: q=quit")

	// Combine content with bottom status and add margins
	fullContent := content + "\n" + statusLine

	// Add margins: 1 character on all sides
	return lipgloss.NewStyle().
		Margin(0, 1, 1, 1). // top, right, bottom, left
		Render(fullContent)
}

// tickCmd returns a command that ticks every 33ms for 30 FPS smooth updates (matching spinner rate)
func tickCmd() tea.Cmd {
	return tea.Tick(33*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// SendUpdate sends a context update to the bubbletea model
func SendUpdate(p *tea.Program, ctx *PipelineContext) {
	if p != nil {
		p.Send(UpdateContextMsg(ctx))
	}
}

// formatElapsed formats a duration for display (e.g., "2m 20s")
func formatElapsed(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}
