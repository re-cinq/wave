package display

import (
	"fmt"
	"strings"
	"time"
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

// Render displays the complete dashboard with all panels.
func (d *Dashboard) Render(ctx *PipelineContext) error {
	// Clear previous output if rendered before
	if d.lastLines > 0 {
		d.clearPreviousRender()
	}

	// Build dashboard content
	var output strings.Builder

	// Render Wave logo and header
	output.WriteString(d.renderHeader())
	output.WriteString("\n")

	// Render overall pipeline progress panel
	output.WriteString(d.renderProgressPanel(ctx))
	output.WriteString("\n")

	// Render step status panel
	output.WriteString(d.renderStepStatusPanel(ctx))
	output.WriteString("\n")

	// Render project information panel
	output.WriteString(d.renderProjectInfoPanel(ctx))
	output.WriteString("\n")

	// Render current action if available
	if ctx.CurrentAction != "" {
		output.WriteString(d.renderCurrentAction(ctx))
		output.WriteString("\n")
	}

	// Count lines for next clear
	content := output.String()
	d.lastLines = strings.Count(content, "\n") + 1

	// Print to output
	fmt.Print(content)

	return nil
}

// Clear removes the dashboard display.
func (d *Dashboard) Clear() {
	if d.lastLines > 0 {
		d.clearPreviousRender()
		d.lastLines = 0
	}
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

// renderHeader displays the Wave ASCII logo and title.
func (d *Dashboard) renderHeader() string {
	var sb strings.Builder

	// Wave ASCII logo (exact format as specified)
	logo := []string{
		"╦ ╦╔═╗╦  ╦╔═╗",
		"║║║╠═╣╚╗╔╝║╣",
		"╚╩╝╩ ╩ ╚╝ ╚═╝",
	}

	// Render logo with primary color
	for _, line := range logo {
		sb.WriteString(d.codec.Primary(line))
		sb.WriteString("\n")
	}

	// Add subtitle
	subtitle := "Multi-Agent Pipeline Orchestrator"
	sb.WriteString(d.codec.Muted(subtitle))
	sb.WriteString("\n")

	return sb.String()
}

// renderProgressPanel displays overall pipeline progress with progress bar and ETA.
func (d *Dashboard) renderProgressPanel(ctx *PipelineContext) string {
	var sb strings.Builder

	// Panel header
	sb.WriteString(d.codec.Bold("Pipeline Progress"))
	sb.WriteString("\n")

	// Progress bar
	progressBar := d.renderProgressBar(ctx.OverallProgress, 40)
	sb.WriteString(progressBar)
	sb.WriteString(fmt.Sprintf(" %d%%\n", ctx.OverallProgress))

	// Step counter: "Step X of Y"
	stepInfo := fmt.Sprintf("Step %d of %d", ctx.CurrentStepNum, ctx.TotalSteps)
	sb.WriteString(d.codec.Primary(stepInfo))

	// Add completion counts if any steps completed or failed
	if ctx.CompletedSteps > 0 || ctx.FailedSteps > 0 || ctx.SkippedSteps > 0 {
		counts := fmt.Sprintf(" (%s%d completed%s", d.codec.Success(""), ctx.CompletedSteps, "\033[0m")
		if ctx.FailedSteps > 0 {
			counts += fmt.Sprintf(", %s%d failed%s", d.codec.Error(""), ctx.FailedSteps, "\033[0m")
		}
		if ctx.SkippedSteps > 0 {
			counts += fmt.Sprintf(", %s%d skipped%s", d.codec.Muted(""), ctx.SkippedSteps, "\033[0m")
		}
		counts += ")"
		sb.WriteString(counts)
	}
	sb.WriteString("\n")

	// ETA and elapsed time
	if ctx.EstimatedTimeMs > 0 {
		eta := formatDashboardDuration(ctx.EstimatedTimeMs)
		sb.WriteString(fmt.Sprintf("ETA: %s", d.codec.Muted(eta)))
		sb.WriteString(" ")
	}

	if ctx.ElapsedTimeMs > 0 {
		elapsed := formatDashboardDuration(ctx.ElapsedTimeMs)
		sb.WriteString(fmt.Sprintf("Elapsed: %s", d.codec.Muted(elapsed)))
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderStepStatusPanel displays the status of all pipeline steps with completion indicators.
func (d *Dashboard) renderStepStatusPanel(ctx *PipelineContext) string {
	var sb strings.Builder

	// Panel header
	sb.WriteString(d.codec.Bold("Step Status"))
	sb.WriteString("\n")

	// Render step statuses
	if len(ctx.StepStatuses) == 0 {
		sb.WriteString(d.codec.Muted("  No steps available"))
		sb.WriteString("\n")
		return sb.String()
	}

	// Display each step with appropriate status indicator
	stepNum := 1
	for stepID, state := range ctx.StepStatuses {
		icon := d.getStatusIcon(state)
		stepLabel := fmt.Sprintf("  %s Step %d: %s", icon, stepNum, stepID)

		// Highlight current step
		if stepID == ctx.CurrentStepID {
			stepLabel = d.codec.Bold(stepLabel)
			if ctx.CurrentPersona != "" {
				stepLabel += fmt.Sprintf(" (%s)", d.codec.Primary(ctx.CurrentPersona))
			}
		} else {
			// Color based on state
			switch state {
			case StateCompleted:
				stepLabel = d.codec.Success(stepLabel)
			case StateFailed:
				stepLabel = d.codec.Error(stepLabel)
			case StateSkipped:
				stepLabel = d.codec.Muted(stepLabel)
			}
		}

		sb.WriteString(stepLabel)
		sb.WriteString("\n")
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

// renderProgressBar creates a visual progress bar.
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

	// Filled portion
	filled := strings.Repeat(d.charSet.Block, filledWidth)
	bar.WriteString(d.codec.Success(filled))

	// Empty portion
	empty := strings.Repeat(d.charSet.LightBlock, emptyWidth)
	bar.WriteString(d.codec.Muted(empty))

	bar.WriteString("]")
	return bar.String()
}

// getStatusIcon returns the appropriate icon for a step state.
func (d *Dashboard) getStatusIcon(state ProgressState) string {
	switch state {
	case StateCompleted:
		return d.codec.Success(d.charSet.CheckMark)
	case StateFailed:
		return d.codec.Error(d.charSet.CrossMark)
	case StateRunning:
		return d.codec.Primary("⏳") // Hourglass for running
	case StateSkipped:
		return d.codec.Muted("⊘") // Empty circle with slash
	case StateCancelled:
		return d.codec.Warning("⊛") // Circled X
	case StateNotStarted:
		return d.codec.Muted("○") // Empty circle
	default:
		return d.codec.Muted("○")
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
		return fmt.Sprintf("%.0fs", duration.Seconds())
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
