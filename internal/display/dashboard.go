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

// NewDashboardWithConfig creates a dashboard with specific configuration.
func NewDashboardWithConfig(colorMode string, asciiOnly bool) *Dashboard {
	return &Dashboard{
		codec:     NewANSICodecWithConfig(colorMode, asciiOnly),
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

// clearPreviousRender clears the previously rendered dashboard.
func (d *Dashboard) clearPreviousRender() {
	if !d.termInfo.SupportsANSI() {
		return
	}

	for i := 0; i < d.lastLines; i++ {
		fmt.Print(d.codec.CursorUp(1))
		fmt.Print(d.codec.ClearLine())
	}
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
		fmt.Sprintf("%.1fs • %s", elapsed, ctx.ManifestPath),
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

// renderProgressPanel displays compact pipeline progress.
func (d *Dashboard) renderProgressPanel(ctx *PipelineContext) string {
	var sb strings.Builder

	// Compact progress line: Progress bar + percentage + step counter
	progressBar := d.renderProgressBar(ctx.OverallProgress, 25)
	sb.WriteString(progressBar)
	sb.WriteString(fmt.Sprintf(" %d%% ", ctx.OverallProgress))
	sb.WriteString(fmt.Sprintf("Step %d/%d", ctx.CurrentStepNum, ctx.TotalSteps))

	// Add completion counts in same line
	if ctx.CompletedSteps > 0 || ctx.FailedSteps > 0 || ctx.SkippedSteps > 0 {
		sb.WriteString(" (")
		if ctx.CompletedSteps > 0 {
			sb.WriteString(fmt.Sprintf("%d ok", ctx.CompletedSteps))
		}
		if ctx.FailedSteps > 0 {
			if ctx.CompletedSteps > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%d fail", ctx.FailedSteps))
		}
		if ctx.SkippedSteps > 0 {
			if ctx.CompletedSteps > 0 || ctx.FailedSteps > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%d skip", ctx.SkippedSteps))
		}
		sb.WriteString(")")
	}
	sb.WriteString("\n")

	return sb.String()
}

// renderStepStatusPanel displays compact step status.
func (d *Dashboard) renderStepStatusPanel(ctx *PipelineContext) string {
	var sb strings.Builder

	if len(ctx.StepStatuses) == 0 {
		sb.WriteString("No steps")
		sb.WriteString("\n")
		return sb.String()
	}

	// Compact step display - only current step or recent activity
	stepNum := 1
	for stepID, state := range ctx.StepStatuses {
		// Show current step with pulsating effect, others just as status
		if stepID == ctx.CurrentStepID {
			icon := d.getStatusIcon(state)
			stepLabel := fmt.Sprintf("%s %s", icon, stepID)

			// Pulsating current step
			pulsatingLabel := d.renderPulsatingStep(stepLabel)
			if ctx.CurrentPersona != "" {
				pulsatingLabel += fmt.Sprintf(" (%s)", ctx.CurrentPersona)
			}
			sb.WriteString(pulsatingLabel)
			sb.WriteString("\n")
			break
		}
		stepNum++
	}

	return sb.String()
}

// renderProjectInfoPanel displays project metadata and workspace information.
func (d *Dashboard) renderProjectInfoPanel(ctx *PipelineContext) string {
	var sb strings.Builder

	// Panel header
	sb.WriteString(d.codec.Bold("Project Information"))
	sb.WriteString("\n")

	// Pipeline name
	if ctx.PipelineName != "" {
		sb.WriteString(fmt.Sprintf("  Pipeline: %s\n", d.codec.Primary(ctx.PipelineName)))
	}

	// Manifest path
	if ctx.ManifestPath != "" {
		sb.WriteString(fmt.Sprintf("  Manifest: %s\n", d.codec.Muted(ctx.ManifestPath)))
	}

	// Workspace path
	if ctx.WorkspacePath != "" {
		sb.WriteString(fmt.Sprintf("  Workspace: %s\n", d.codec.Muted(ctx.WorkspacePath)))
	}

	return sb.String()
}

// renderCurrentAction displays the current step's action if available.
func (d *Dashboard) renderCurrentAction(ctx *PipelineContext) string {
	var sb strings.Builder

	sb.WriteString(d.codec.Bold("Current Activity"))
	sb.WriteString("\n")

	if ctx.CurrentStepName != "" {
		sb.WriteString(fmt.Sprintf("  %s", ctx.CurrentStepName))
		sb.WriteString("\n")
	}

	if ctx.CurrentAction != "" {
		sb.WriteString(fmt.Sprintf("  %s %s", d.charSet.RightArrow, ctx.CurrentAction))
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
		return "-"
	case StateCancelled:
		return "X"
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

// RenderCompact provides a condensed single-line view for constrained terminals.
func (d *Dashboard) RenderCompact(ctx *PipelineContext) error {
	// Clear previous output
	if d.lastLines > 0 {
		d.clearPreviousRender()
	}

	// Build compact output
	var output strings.Builder

	// Pipeline name and progress
	output.WriteString(d.codec.Bold(ctx.PipelineName))
	output.WriteString(" ")
	output.WriteString(d.renderProgressBar(ctx.OverallProgress, 20))
	output.WriteString(fmt.Sprintf(" %d%% ", ctx.OverallProgress))

	// Step info
	output.WriteString(fmt.Sprintf("[%d/%d] ", ctx.CurrentStepNum, ctx.TotalSteps))

	// Current step
	if ctx.CurrentStepID != "" {
		output.WriteString(d.codec.Primary(ctx.CurrentStepID))
	}

	// ETA
	if ctx.EstimatedTimeMs > 0 {
		eta := formatDashboardDuration(ctx.EstimatedTimeMs)
		output.WriteString(fmt.Sprintf(" ETA: %s", d.codec.Muted(eta)))
	}

	output.WriteString("\n")

	d.lastLines = 1
	fmt.Print(output.String())

	return nil
}

// ShouldUseCompactMode determines if compact mode should be used based on terminal size.
func (d *Dashboard) ShouldUseCompactMode() bool {
	// Use compact mode if terminal height is less than 20 lines or width less than 60 columns
	return d.termInfo.GetHeight() < 20 || d.termInfo.GetWidth() < 60
}

// formatDashboardDuration converts milliseconds to a human-readable duration string.
func formatDashboardDuration(ms int64) string {
	duration := time.Duration(ms) * time.Millisecond

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}

	if duration < time.Hour {
		minutes := int(duration.Minutes())
		seconds := int(duration.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// RenderPerformanceMetricsPanel displays performance metrics with animated counters.
func (d *Dashboard) RenderPerformanceMetricsPanel(tokens int, files int, artifacts int, durationMs int64, burnRate float64) string {
	var sb strings.Builder

	// Panel header
	sb.WriteString(d.codec.Bold("Performance Metrics"))
	sb.WriteString("\n")

	// Token count
	if tokens > 0 {
		tokenStr := FormatTokenCount(tokens)
		sb.WriteString(fmt.Sprintf("  Tokens: %s\n", d.codec.Primary(tokenStr)))
	}

	// Files modified
	if files > 0 {
		sb.WriteString(fmt.Sprintf("  Files Modified: %s\n", d.codec.Muted(fmt.Sprintf("%d", files))))
	}

	// Artifacts generated
	if artifacts > 0 {
		sb.WriteString(fmt.Sprintf("  Artifacts: %s\n", d.codec.Muted(fmt.Sprintf("%d", artifacts))))
	}

	// Duration
	if durationMs > 0 {
		durationStr := formatDashboardDuration(durationMs)
		sb.WriteString(fmt.Sprintf("  Duration: %s\n", d.codec.Muted(durationStr)))
	}

	// Token burn rate
	if burnRate > 0 {
		burnRateStr := ""
		if burnRate < 1 {
			burnRateStr = "< 1 token/s"
		} else if burnRate < 1000 {
			burnRateStr = fmt.Sprintf("%.1f tokens/s", burnRate)
		} else {
			burnRateStr = fmt.Sprintf("%.2fk tokens/s", burnRate/1000.0)
		}
		sb.WriteString(fmt.Sprintf("  Burn Rate: %s\n", d.codec.Primary(burnRateStr)))
	}

	return sb.String()
}

// RenderPerformanceComparison displays performance comparison indicators.
func (d *Dashboard) RenderPerformanceComparison(currentDuration int64, avgDuration int64, threshold float64) string {
	if avgDuration == 0 || currentDuration == 0 {
		return ""
	}

	ratio := float64(currentDuration) / float64(avgDuration)

	if ratio > (1.0 + threshold) {
		// Significantly slower than average
		percentSlower := int((ratio - 1.0) * 100)
		return d.codec.Warning(fmt.Sprintf("  ⚠ %d%% slower than average", percentSlower))
	}

	if ratio < (1.0 - threshold) {
		// Significantly faster than average
		percentFaster := int((1.0 - ratio) * 100)
		return d.codec.Success(fmt.Sprintf("  ✓ %d%% faster than average", percentFaster))
	}

	// Within normal range
	return d.codec.Muted("  ≈ average performance")
}
