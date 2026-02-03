package display

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProgressModel implements the bubbletea model for Wave progress display
type ProgressModel struct {
	ctx        *PipelineContext
	paused     bool
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
		paused:     false,
		quit:       false,
		lastUpdate: time.Now(),
	}
}

// Init implements tea.Model
func (m *ProgressModel) Init() tea.Cmd {
	return tickCmd() // Just start ticking, no alt screen
}

// Update implements tea.Model
func (m *ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		case "p":
			m.paused = !m.paused
			return m, nil
		}

	case TickMsg:
		m.lastUpdate = time.Time(msg)
		// Don't continue ticking when paused
		if m.paused {
			return m, nil
		}
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
	header := m.renderHeader()

	// Progress line with spacing
	progress := m.renderProgress()

	// Current step with spacing
	currentStep := m.renderCurrentStep()

	// Main content area
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"", // Empty line for spacing
		progress,
		"", // Empty line for spacing
		currentStep,
		"", // Empty line before buttons
		"", // Another empty line
	)

	// Bottom status line with readable colors
	var statusLine string
	if m.paused {
		statusLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Bright yellow for paused
			Render("PAUSED - Press 'p' to resume, 'q' to quit")
	} else {
		statusLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")). // Medium gray for buttons
			Render("Press: p=pause q=quit")
	}

	// Combine content with bottom status and add margins
	fullContent := content + "\n" + statusLine

	// Add margins: 1 character on all sides
	return lipgloss.NewStyle().
		Margin(1, 1, 1, 1). // top, right, bottom, left
		Render(fullContent)
}

// renderHeader creates the header with proper spacing and alignment
func (m *ProgressModel) renderHeader() string {
	// ASCII logo
	logo := []string{
		"╦ ╦╔═╗╦  ╦╔═╗",
		"║║║╠═╣╚╗╔╝║╣",
		"╚╩╝╩ ╩ ╚╝ ╚═╝",
	}

	// Project info with real-time elapsed time calculation
	pipelineStart := time.Unix(0, m.ctx.PipelineStartTime)
	elapsed := time.Since(pipelineStart).Seconds()

	projectLines := []string{
		fmt.Sprintf("Pipeline: %s", m.ctx.PipelineName),
		fmt.Sprintf("Config:   %s", m.ctx.ManifestPath),
		fmt.Sprintf("Elapsed:  %.1fs", elapsed), // Real-time elapsed time
	}

	// Create columns with proper spacing and bright colors
	logoColumn := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")). // Bright cyan for logo
		Render(lipgloss.JoinVertical(lipgloss.Left, logo...))

	projectColumn := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")). // Light gray for project info
		Render(lipgloss.JoinVertical(lipgloss.Left, projectLines...))

	// Join horizontally with spacing
	return lipgloss.JoinHorizontal(lipgloss.Top,
		logoColumn,
		lipgloss.NewStyle().Width(4).Render(""), // Spacer
		projectColumn,
	) + "\n"
}

// renderProgress creates the progress bar and step info
func (m *ProgressModel) renderProgress() string {
	// Progress bar (25 chars wide)
	width := 25
	filled := (m.ctx.OverallProgress * width) / 100
	empty := width - filled

	// Create progress bar with proper block characters
	var progressBar string
	progressBar = "["
	for i := 0; i < filled; i++ {
		progressBar += "█" // Filled block
	}
	for i := 0; i < empty; i++ {
		progressBar += "░" // Empty block
	}
	progressBar += "]"

	stepInfo := fmt.Sprintf(" %d%% Step %d/%d",
		m.ctx.OverallProgress, m.ctx.CurrentStepNum, m.ctx.TotalSteps)

	// Add completion counts
	if m.ctx.CompletedSteps > 0 || m.ctx.FailedSteps > 0 || m.ctx.SkippedSteps > 0 {
		counts := " ("
		if m.ctx.CompletedSteps > 0 {
			counts += fmt.Sprintf("%d ok", m.ctx.CompletedSteps)
		}
		if m.ctx.FailedSteps > 0 {
			if m.ctx.CompletedSteps > 0 {
				counts += ", "
			}
			counts += fmt.Sprintf("%d fail", m.ctx.FailedSteps)
		}
		if m.ctx.SkippedSteps > 0 {
			if m.ctx.CompletedSteps > 0 || m.ctx.FailedSteps > 0 {
				counts += ", "
			}
			counts += fmt.Sprintf("%d skip", m.ctx.SkippedSteps)
		}
		counts += ")"
		stepInfo += counts
	}

	return progressBar + stepInfo
}

// renderCurrentStep shows detailed step information with loading indicators
func (m *ProgressModel) renderCurrentStep() string {
	var steps []string

	// Show all completed steps first, then current step
	// We need to iterate in proper order, not random map order
	completedSteps := []string{}
	currentStep := ""

	// Collect completed and current steps
	for stepID, stepState := range m.ctx.StepStatuses {
		if stepState == StateCompleted {
			completedSteps = append(completedSteps, stepID)
		} else if stepState == StateRunning && stepID == m.ctx.CurrentStepID {
			currentStep = stepID
		}
	}

	// Show completed steps with deliverables
	for _, stepID := range completedSteps {
		// Show completed step with actual duration
		durationText := "0.0s"
		if m.ctx.StepDurations != nil {
			if durationMs, exists := m.ctx.StepDurations[stepID]; exists {
				durationText = fmt.Sprintf("%.1fs", float64(durationMs)/1000.0)
			}
		}
		stepLine := fmt.Sprintf("✓ %s (%s)", stepID, durationText)
		stepLine = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true).Render(stepLine) // Bright cyan like logo
		steps = append(steps, stepLine)

		// Show deliverables for this step in tree format
		if m.ctx.DeliverablesByStep != nil {
			if stepDeliverables, exists := m.ctx.DeliverablesByStep[stepID]; exists {
				for _, deliverable := range stepDeliverables {
					deliverableLine := fmt.Sprintf("  ├─ %s", deliverable)
					deliverableLine = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(deliverableLine) // Medium gray
					steps = append(steps, deliverableLine)
				}
			}
		}
	}

	// Show current running step
	if currentStep != "" {
		// Show current running step with spinner
		spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		now := time.Now().UnixMilli()
		frame := (now / 80) % int64(len(spinners)) // 80ms per frame
		icon := spinners[frame]

		stepLine := fmt.Sprintf("%s %s", icon, currentStep)
		if m.ctx.CurrentPersona != "" {
			stepLine += fmt.Sprintf(" (%s)", m.ctx.CurrentPersona)
		}
		// Real-time step timing in seconds with one decimal
		stepStart := time.Unix(0, m.ctx.CurrentStepStart)
		stepElapsed := time.Since(stepStart).Seconds()
		stepLine += fmt.Sprintf(" (%.1fs)", stepElapsed)

		if m.ctx.CurrentAction != "" {
			stepLine += fmt.Sprintf(" • %s", m.ctx.CurrentAction)
		}

		stepLine = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(stepLine) // Bright yellow
		steps = append(steps, stepLine)
	}

	if len(steps) == 0 {
		return "Waiting for pipeline to start..."
	}

	return lipgloss.JoinVertical(lipgloss.Left, steps...)
}

// tickCmd returns a command that ticks every 200ms for smooth but not overwhelming updates
func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// SendUpdate sends a context update to the bubbletea model
func SendUpdate(p *tea.Program, ctx *PipelineContext) {
	if p != nil {
		p.Send(UpdateContextMsg(ctx))
	}
}