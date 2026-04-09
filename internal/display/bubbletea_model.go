package display

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/recinq/wave/internal/pathfmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Unified color palette — semantic names for consistent theming
var (
	colorPrimary     = lipgloss.Color("14")  // Bright cyan — active/accent
	colorSuccess     = lipgloss.Color("10")  // Bright green — completed
	colorError       = lipgloss.Color("9")   // Bright red — failed
	colorMuted       = lipgloss.Color("244") // Medium gray — metadata/inactive
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
	header := m.renderHeader()

	// Current step with spacing
	currentStep := m.renderCurrentStep()

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

// shimmerPosition calculates a ping-pong sweep position across the logo width.
// It returns a float64 in [0, logoWidth] that bounces back and forth over cycleMs.
func shimmerPosition(logoWidth int, cycleMs int64) float64 { //nolint:unparam // logoWidth kept for flexibility
	now := time.Now().UnixMilli()
	halfCycle := cycleMs / 2
	phase := now % cycleMs
	// Sweep from -3 to logoWidth+2 so all shimmer bands (core + bold fringe)
	// fully disappear off both edges before the bounce
	overflow := 3.0
	sweepRange := float64(logoWidth-1) + 2*overflow
	var pos float64
	if phase < halfCycle {
		pos = float64(phase) / float64(halfCycle) * sweepRange
	} else {
		pos = float64(cycleMs-phase) / float64(halfCycle) * sweepRange
	}
	return pos - overflow
}

// shimmerColorForChar returns a style for a single logo character based on
// its distance from the current shimmer center position.
func shimmerColorForChar(charPos int, shimmerCenter float64) lipgloss.Style {
	distance := math.Abs(float64(charPos) - shimmerCenter)
	switch {
	case distance < 1.0:
		return lipgloss.NewStyle().Foreground(colorShimmerCore).Bold(true)
	case distance < 2.5:
		return lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	case distance < 4.0:
		return lipgloss.NewStyle().Foreground(colorShimmerMid)
	default:
		return lipgloss.NewStyle().Foreground(colorShimmerBase)
	}
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
	elapsed := time.Since(pipelineStart)

	pipelineLabel := m.ctx.PipelineName
	if m.ctx.PipelineID != "" && m.ctx.PipelineID != m.ctx.PipelineName {
		pipelineLabel = m.ctx.PipelineID
	}
	// Build compact progress summary for third line
	progressLine := fmt.Sprintf("Progress: %d%% Step %d/%d", m.ctx.OverallProgress, m.ctx.CurrentStepNum, m.ctx.TotalSteps)
	{
		var parts []string
		for _, stepID := range m.ctx.StepOrder {
			if state, exists := m.ctx.StepStatuses[stepID]; exists && state == StateRunning {
				parts = append(parts, stepID) // just count
			}
		}
		runningCount := len(parts)
		var counts []string
		if runningCount > 0 {
			counts = append(counts, fmt.Sprintf("%d running", runningCount))
		}
		if m.ctx.CompletedSteps > 0 {
			counts = append(counts, fmt.Sprintf("%d ok", m.ctx.CompletedSteps))
		}
		if m.ctx.FailedSteps > 0 {
			requiredFails := m.ctx.FailedSteps - m.ctx.OptionalFailedSteps
			if requiredFails > 0 {
				counts = append(counts, fmt.Sprintf("%d fail", requiredFails))
			}
			if m.ctx.OptionalFailedSteps > 0 {
				counts = append(counts, fmt.Sprintf("%d optional-fail", m.ctx.OptionalFailedSteps))
			}
		}
		if len(counts) > 0 {
			progressLine += " (" + strings.Join(counts, ", ") + ")"
		}
	}

	projectLines := []string{
		fmt.Sprintf("Pipeline: %s", pipelineLabel),
		fmt.Sprintf("Elapsed:  %s", m.formatElapsedWithTokens(elapsed)),
		progressLine,
	}
	if m.ctx.EstimatedTimeMs > 0 {
		projectLines = append(projectLines, fmt.Sprintf("ETA:      %s", FormatDuration(m.ctx.EstimatedTimeMs)))
	}

	// Render logo with per-character shimmer animation
	shimmerCenter := shimmerPosition(15, 2500)
	var shimmerLines []string
	for _, line := range logo {
		var rendered strings.Builder
		for runeIdx, r := range []rune(line) {
			rendered.WriteString(shimmerColorForChar(runeIdx, shimmerCenter).Render(string(r)))
		}
		shimmerLines = append(shimmerLines, rendered.String())
	}
	logoColumn := lipgloss.JoinVertical(lipgloss.Left, shimmerLines...)

	projectColumn := lipgloss.NewStyle().
		Foreground(colorInfo).
		Render(lipgloss.JoinVertical(lipgloss.Left, projectLines...))

	// Join horizontally with spacing
	return lipgloss.JoinHorizontal(lipgloss.Top,
		logoColumn,
		lipgloss.NewStyle().Width(4).Render(""), // Spacer
		projectColumn,
	)
}

// renderCurrentStep shows detailed step information with loading indicators
func (m *ProgressModel) renderCurrentStep() string {
	var steps []string

	// Iterate ALL steps in pipeline definition order
	for _, stepID := range m.ctx.StepOrder {
		state, exists := m.ctx.StepStatuses[stepID]
		if !exists {
			state = StateNotStarted
		}

		// Look up persona for this step
		persona := ""
		if m.ctx.StepPersonas != nil {
			persona = m.ctx.StepPersonas[stepID]
		}

		switch state {
		case StateCompleted:
			// Completed: checkmark stepID (persona) [model] (duration, tokens)
			durationText := "0.0s"
			if m.ctx.StepDurations != nil {
				if durationMs, exists := m.ctx.StepDurations[stepID]; exists {
					durationText = fmt.Sprintf("%.1fs", float64(durationMs)/1000.0)
				}
			}
			stepLine := fmt.Sprintf("✓ %s", stepID)
			if persona != "" {
				stepLine += fmt.Sprintf(" (%s)", persona)
			}
			// Show model info for completed steps
			if m.ctx.StepModels != nil {
				if model, ok := m.ctx.StepModels[stepID]; ok && model != "" {
					stepLine += fmt.Sprintf(" [%s]", model)
				}
			}
			// Append tokens alongside duration if available
			switch {
			case m.ctx.StepTokensIn != nil:
				tIn := m.ctx.StepTokensIn[stepID]
				tOut := m.ctx.StepTokensOut[stepID]
				switch {
				case tIn > 0 || tOut > 0:
					stepLine += fmt.Sprintf(" (%s, %s in / %s out)", durationText, FormatTokenCount(tIn), FormatTokenCount(tOut))
				case m.ctx.StepTokens != nil:
					if tokens, ok := m.ctx.StepTokens[stepID]; ok && tokens > 0 {
						stepLine += fmt.Sprintf(" (%s, %s tokens)", durationText, FormatTokenCount(tokens))
					} else {
						stepLine += fmt.Sprintf(" (%s)", durationText)
					}
				default:
					stepLine += fmt.Sprintf(" (%s)", durationText)
				}
			case m.ctx.StepTokens != nil:
				if tokens, ok := m.ctx.StepTokens[stepID]; ok && tokens > 0 {
					stepLine += fmt.Sprintf(" (%s, %s tokens)", durationText, FormatTokenCount(tokens))
				} else {
					stepLine += fmt.Sprintf(" (%s)", durationText)
				}
			default:
				stepLine += fmt.Sprintf(" (%s)", durationText)
			}
			stepLine = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render(stepLine)
			steps = append(steps, stepLine)

			// Collect all metadata lines (deliverables + handover) for tree formatting
			var metadataLines []string

			// Deliverables
			if m.ctx.DeliverablesByStep != nil {
				if stepDeliverables, exists := m.ctx.DeliverablesByStep[stepID]; exists {
					metadataLines = append(metadataLines, stepDeliverables...)
				}
			}

			// Handover metadata (verbose mode only)
			if m.ctx.Verbose && m.ctx.HandoversByStep != nil {
				if info, exists := m.ctx.HandoversByStep[stepID]; exists {
					// Artifact lines
					for _, path := range info.ArtifactPaths {
						metadataLines = append(metadataLines, fmt.Sprintf("artifact: %s (written)", pathfmt.FileURI(path)))
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
						metadataLines = append(metadataLines, fmt.Sprintf("contract: %s %s", schema, status))
					}
					// Handover target line
					if info.TargetStep != "" {
						metadataLines = append(metadataLines, fmt.Sprintf("handover → %s", info.TargetStep))
					}
				}
			}

			// Render all metadata lines with correct tree connectors
			for i, line := range metadataLines {
				connector := "├─"
				if i == len(metadataLines)-1 {
					connector = "└─"
				}
				metaLine := fmt.Sprintf("   %s %s", connector, line)
				metaLine = lipgloss.NewStyle().Foreground(colorMuted).Render(metaLine)
				steps = append(steps, metaLine)
			}

		case StateRunning:
			// Running: spinner stepID (persona) [model via adapter] (elapsed) bullet action
			spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			now := time.Now().UnixMilli()
			frame := (now / 80) % int64(len(spinners))
			icon := spinners[frame]

			stepLine := fmt.Sprintf("%s %s", icon, stepID)
			if persona != "" {
				stepLine += fmt.Sprintf(" (%s)", persona)
			}
			// Show model/adapter info for running steps
			if m.ctx.StepModels != nil {
				if model, ok := m.ctx.StepModels[stepID]; ok && model != "" {
					modelInfo := model
					if m.ctx.StepAdapters != nil {
						if adpt, ok := m.ctx.StepAdapters[stepID]; ok && adpt != "" {
							modelInfo += " via " + adpt
						}
					}
					stepLine += fmt.Sprintf(" [%s]", modelInfo)
				}
			}
			// Per-step timing: use StepStartTimes if available, fall back to CurrentStepStart
			var stepElapsed time.Duration
			if m.ctx.StepStartTimes != nil {
				if startNano, ok := m.ctx.StepStartTimes[stepID]; ok {
					stepElapsed = time.Since(time.Unix(0, startNano))
				}
			}
			if stepElapsed == 0 {
				stepStart := time.Unix(0, m.ctx.CurrentStepStart)
				stepElapsed = time.Since(stepStart)
			}
			stepLine += fmt.Sprintf(" (%s)", formatElapsed(stepElapsed))

			if m.ctx.CurrentAction != "" && stepID == m.ctx.CurrentStepID {
				stepLine += fmt.Sprintf(" • %s", m.ctx.CurrentAction)
			}

			stepLine = lipgloss.NewStyle().Foreground(colorPrimary).Render(stepLine)
			steps = append(steps, stepLine)

			// Show per-step tool activity when available, fall back to global
			toolName := ""
			toolTarget := ""
			if m.ctx.StepToolActivity != nil {
				if ta, ok := m.ctx.StepToolActivity[stepID]; ok {
					toolName = ta[0]
					toolTarget = ta[1]
				}
			}
			if toolName == "" && stepID == m.ctx.CurrentStepID {
				toolName = m.ctx.LastToolName
				toolTarget = m.ctx.LastToolTarget
			}
			if toolName != "" {
				overhead := 6 + len(toolName)
				termWidth := getTerminalWidth()
				maxTarget := termWidth - overhead
				if maxTarget < 20 {
					maxTarget = 20
				}
				target := toolTarget
				if len(target) > maxTarget {
					target = target[:maxTarget-3] + "..."
				}
				toolLine := fmt.Sprintf("   %s → %s", toolName, target)
				toolLine = lipgloss.NewStyle().Foreground(colorMuted).Render(toolLine)
				steps = append(steps, toolLine)
			}

		case StateFailed:
			// Failed: cross stepID (persona) (duration)
			stepLine := fmt.Sprintf("✗ %s", stepID)
			if persona != "" {
				stepLine += fmt.Sprintf(" (%s)", persona)
			}
			if m.ctx.StepDurations != nil {
				if durationMs, exists := m.ctx.StepDurations[stepID]; exists {
					durationText := fmt.Sprintf("%.1fs", float64(durationMs)/1000.0)
					stepLine += fmt.Sprintf(" (%s)", durationText)
				}
			}
			stepLine = lipgloss.NewStyle().Foreground(colorError).Render(stepLine)
			steps = append(steps, stepLine)

		case StateSkipped:
			// Skipped: dash stepID (persona)
			stepLine := fmt.Sprintf("— %s", stepID)
			if persona != "" {
				stepLine += fmt.Sprintf(" (%s)", persona)
			}
			stepLine = lipgloss.NewStyle().Foreground(colorMuted).Render(stepLine)
			steps = append(steps, stepLine)

		case StateCancelled:
			// Cancelled: circled asterisk stepID (persona)
			stepLine := fmt.Sprintf("⊛ %s", stepID)
			if persona != "" {
				stepLine += fmt.Sprintf(" (%s)", persona)
			}
			stepLine = lipgloss.NewStyle().Foreground(colorMuted).Render(stepLine)
			steps = append(steps, stepLine)

		default:
			// Not started (pending): circle stepID (persona)
			stepLine := fmt.Sprintf("○ %s", stepID)
			if persona != "" {
				stepLine += fmt.Sprintf(" (%s)", persona)
			}
			stepLine = lipgloss.NewStyle().Foreground(colorMuted).Render(stepLine)
			steps = append(steps, stepLine)
		}
	}

	if len(steps) == 0 {
		return "Waiting for pipeline to start..."
	}

	return lipgloss.JoinVertical(lipgloss.Left, steps...)
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

// formatElapsedWithTokens formats elapsed time and optionally appends total token count.
func (m *ProgressModel) formatElapsedWithTokens(d time.Duration) string {
	elapsed := formatElapsed(d)
	if m.ctx.TotalTokensIn > 0 || m.ctx.TotalTokensOut > 0 {
		return fmt.Sprintf("%s • %s in / %s out", elapsed, FormatTokenCount(m.ctx.TotalTokensIn), FormatTokenCount(m.ctx.TotalTokensOut))
	}
	if m.ctx.TotalTokens > 0 {
		return fmt.Sprintf("%s • %s tokens", elapsed, FormatTokenCount(m.ctx.TotalTokens))
	}
	return elapsed
}
