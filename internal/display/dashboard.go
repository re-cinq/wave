package display

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Dashboard provides a comprehensive pipeline execution overview with panel-based layout.
// It integrates the Wave ASCII logo and displays overall progress, step tracking, and project info.
type Dashboard struct {
	codec     *ANSICodec
	charSet   UnicodeCharSet
	termInfo  *TerminalInfo
	lastLines int // Track number of lines for clearing
}

// NewDashboard creates a new dashboard with detected terminal capabilities.
func NewDashboard() *Dashboard {
	return &Dashboard{
		codec:     NewANSICodec(),
		charSet:   GetUnicodeCharSet(),
		termInfo:  NewTerminalInfo(),
		lastLines: 0,
	}
}

// Render displays the compact dashboard with no clearing.
func (d *Dashboard) Render(ctx *PipelineContext) error {
	// No clearing at all - just output content
	var output strings.Builder

	// Compact header with logo and project info
	output.WriteString(d.renderHeader(ctx))

	// Compact progress line
	output.WriteString(d.renderProgressPanel(ctx))

	// Current step only (compact)
	output.WriteString(d.renderStepStatusPanel(ctx))

	// Current action if available (compact)
	if ctx.CurrentAction != "" {
		output.WriteString(fmt.Sprintf("Action: %s\n", ctx.CurrentAction))
	}

	// Just print to output - no clearing needed
	fmt.Print(output.String())

	return nil
}

// Clear removes the dashboard display (no-op since we don't clear).
func (d *Dashboard) Clear() {
	// No clearing needed
}

// renderHeader displays the Wave ASCII logo with project info on the right.
func (d *Dashboard) renderHeader(ctx *PipelineContext) string {
	var sb strings.Builder

	// Wave ASCII logo
	logo := []string{
		"╦ ╦╔═╗╦  ╦╔═╗",
		"║║║╠═╣╚╗╔╝║╣",
		"╚╩╝╩ ╩ ╚╝ ╚═╝",
	}

	// Project info for right side
	elapsed := float64(ctx.ElapsedTimeMs) / 1000.0
	pipelineLabel := ctx.PipelineName
	if ctx.PipelineID != "" && ctx.PipelineID != ctx.PipelineName {
		pipelineLabel = ctx.PipelineID
	}
	projectInfo := []string{
		pipelineLabel,
		d.formatElapsedInfo(elapsed, ctx),
		" Press: q=quit",
	}

	// Render logo with project info aligned to the right
	for i, logoLine := range logo {
		sb.WriteString(d.codec.Primary(logoLine))
		if i < len(projectInfo) {
			// Add spacing and right-aligned project info
			sb.WriteString("  ")
			sb.WriteString(d.codec.Muted(projectInfo[i]))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatElapsedInfo formats the elapsed time info line, optionally including total tokens.
func (d *Dashboard) formatElapsedInfo(elapsed float64, ctx *PipelineContext) string {
	info := fmt.Sprintf("%.1fs", elapsed)
	if ctx.TotalTokensIn > 0 || ctx.TotalTokensOut > 0 {
		info += fmt.Sprintf(" • %s in / %s out", FormatTokenCount(ctx.TotalTokensIn), FormatTokenCount(ctx.TotalTokensOut))
	} else if ctx.TotalTokens > 0 {
		info += fmt.Sprintf(" • %s tokens", FormatTokenCount(ctx.TotalTokens))
	}
	return info
}

// renderProgressPanel displays compact pipeline progress.
func (d *Dashboard) renderProgressPanel(ctx *PipelineContext) string {
	var sb strings.Builder

	// Compact progress line: Progress bar + percentage + step counter
	progressBar := d.renderProgressBar(ctx.OverallProgress, 25)
	sb.WriteString(progressBar)
	sb.WriteString(fmt.Sprintf(" %d%% ", ctx.OverallProgress))
	sb.WriteString(fmt.Sprintf("Step %d/%d", ctx.CurrentStepNum, ctx.TotalSteps))

	// Add completion counts in same line
	if ctx.CompletedSteps > 0 || ctx.FailedSteps > 0 || ctx.SkippedSteps > 0 || ctx.OptionalFailedSteps > 0 {
		sb.WriteString(" (")
		if ctx.CompletedSteps > 0 {
			sb.WriteString(fmt.Sprintf("%d ok", ctx.CompletedSteps))
		}
		requiredFails := ctx.FailedSteps - ctx.OptionalFailedSteps
		if requiredFails > 0 {
			if ctx.CompletedSteps > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%d fail", requiredFails))
		}
		if ctx.OptionalFailedSteps > 0 {
			if ctx.CompletedSteps > 0 || requiredFails > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%d optional-fail", ctx.OptionalFailedSteps))
		}
		if ctx.SkippedSteps > 0 {
			if ctx.CompletedSteps > 0 || ctx.FailedSteps > 0 || ctx.OptionalFailedSteps > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%d skip", ctx.SkippedSteps))
		}
		sb.WriteString(")")
	}
	sb.WriteString("\n")

	return sb.String()
}

// renderStepStatusPanel displays all pipeline steps with their status.
func (d *Dashboard) renderStepStatusPanel(ctx *PipelineContext) string {
	var sb strings.Builder

	if len(ctx.StepStatuses) == 0 {
		sb.WriteString("No steps")
		sb.WriteString("\n")
		return sb.String()
	}

	// Iterate all steps in pipeline definition order
	for _, stepID := range ctx.StepOrder {
		state, exists := ctx.StepStatuses[stepID]
		if !exists {
			state = StateNotStarted
		}

		icon := d.getStatusIcon(state)

		// Look up persona
		persona := ""
		if ctx.StepPersonas != nil {
			persona = ctx.StepPersonas[stepID]
		}

		if state == StateRunning {
			// Pulsating effect for running step
			stepLabel := fmt.Sprintf("%s %s", icon, stepID)
			pulsatingLabel := d.renderPulsatingStep(stepLabel)
			if persona != "" {
				pulsatingLabel += fmt.Sprintf(" (%s)", persona)
			}
			// Show adapter/model for running step
			if model, ok := ctx.StepModels[stepID]; ok && model != "" {
				adapter := ctx.StepAdapters[stepID]
				if adapter != "" {
					pulsatingLabel += fmt.Sprintf(" [%s/%s]", adapter, model)
				} else {
					pulsatingLabel += fmt.Sprintf(" [%s]", model)
				}
			}
			sb.WriteString(pulsatingLabel)
		} else {
			// Static rendering for non-running steps
			stepLine := fmt.Sprintf("%s %s", icon, stepID)
			if persona != "" {
				stepLine += fmt.Sprintf(" (%s)", persona)
			}
			// Show adapter/model for non-running steps
			if model, ok := ctx.StepModels[stepID]; ok && model != "" {
				adapter := ctx.StepAdapters[stepID]
				if adapter != "" {
					stepLine += fmt.Sprintf(" [%s/%s]", adapter, model)
				} else {
					stepLine += fmt.Sprintf(" [%s]", model)
				}
			}
			// Show duration and tokens for completed and failed steps
			if state == StateCompleted || state == StateFailed {
				durationText := ""
				if ctx.StepDurations != nil {
					if durationMs, exists := ctx.StepDurations[stepID]; exists {
						durationText = fmt.Sprintf("%.1fs", float64(durationMs)/1000.0)
					}
				}
				tokenText := ""
				if ctx.StepTokensIn != nil {
					tIn := ctx.StepTokensIn[stepID]
					tOut := ctx.StepTokensOut[stepID]
					if tIn > 0 || tOut > 0 {
						tokenText = fmt.Sprintf("%s in / %s out", FormatTokenCount(tIn), FormatTokenCount(tOut))
					}
				}
				if tokenText == "" && ctx.StepTokens != nil {
					if tokens, exists := ctx.StepTokens[stepID]; exists && tokens > 0 {
						tokenText = fmt.Sprintf("%s tokens", FormatTokenCount(tokens))
					}
				}
				switch {
				case durationText != "" && tokenText != "":
					stepLine += fmt.Sprintf(" (%s, %s)", durationText, tokenText)
				case durationText != "":
					stepLine += fmt.Sprintf(" (%s)", durationText)
				case tokenText != "":
					stepLine += fmt.Sprintf(" (%s)", tokenText)
				}
			}
			sb.WriteString(stepLine)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderProgressBar creates a visual progress bar with pulsing wave animation.
func (d *Dashboard) renderProgressBar(progress int, width int) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

	filledWidth := (progress * width) / 100
	emptyWidth := width - filledWidth

	var bar strings.Builder
	bar.WriteString("[")

	// Calculate pulse position (moves across the empty area)
	now := time.Now().UnixMilli()
	pulseInterval := int64(2000) // 2 second cycle for full pulse wave
	pulseCycle := now % pulseInterval

	// Pulse moves across the entire empty area in one cycle
	pulsePos := -1 // -1 means no pulse
	if emptyWidth > 0 {
		// Pulse position within empty area (0 to emptyWidth-1)
		pulsePos = int((pulseCycle * int64(emptyWidth)) / pulseInterval)
	}

	// Render filled portion - Wave cyan color (matches logo)
	for i := 0; i < filledWidth; i++ {
		filledChar := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render(d.charSet.Block)
		bar.WriteString(filledChar)
	}

	// Render empty portion with pulsing wave
	for i := 0; i < emptyWidth; i++ {
		var char string
		var style lipgloss.Style

		if i == pulsePos {
			// Pulse character - medium shade, brighter color
			char = "▓"
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("14")) // Wave cyan for pulse
		} else {
			// Normal empty character - light shade, muted color
			char = d.charSet.LightBlock
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Dark gray
		}

		styledChar := style.Render(char)
		bar.WriteString(styledChar)
	}

	bar.WriteString("]")
	return bar.String()
}

// getStatusIcon returns the appropriate icon for a step state.
func (d *Dashboard) getStatusIcon(state ProgressState) string {
	switch state {
	case StateCompleted:
		return d.charSet.CheckMark
	case StateFailed:
		return d.charSet.CrossMark
	case StateRunning:
		return ">"
	case StateSkipped:
		return "—"
	case StateCancelled:
		return "⊛"
	case StateNotStarted:
		return "○"
	default:
		return "○"
	}
}

// renderPulsatingStep creates a pulsating visual effect for the currently running step.
func (d *Dashboard) renderPulsatingStep(stepLabel string) string {
	// Calculate pulsating state based on time (roughly every second)
	// Use millisecond timing to create smooth pulsing effect
	now := time.Now().UnixMilli()
	pulseInterval := int64(1000) // 1 second pulse cycle

	// Create 3-phase pulsating: dim -> normal -> bright
	phase := (now / (pulseInterval / 3)) % 3

	switch phase {
	case 0:
		// Dim phase - muted color
		return d.codec.Muted(stepLabel)
	case 1:
		// Normal phase - primary color
		return d.codec.Primary(stepLabel)
	case 2:
		// Bright phase - bold primary
		return d.codec.Bold(d.codec.Primary(stepLabel))
	default:
		return d.codec.Primary(stepLabel)
	}
}
