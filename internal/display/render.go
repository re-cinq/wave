package display

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/recinq/wave/internal/pathfmt"

	"github.com/charmbracelet/lipgloss"
)

// shimmerPosition calculates a ping-pong sweep position across the logo width.
// It returns a float64 in [0, logoWidth] that bounces back and forth over cycleMs.
func shimmerPosition(logoWidth int, cycleMs int64) float64 {
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

// RenderPipelineHeader renders the shimmering WAVE logo + pipeline metadata.
// Used by both ProgressModel and MissionControlModel.
func RenderPipelineHeader(ctx *PipelineContext) string {
	// ASCII logo
	logo := []string{
		"╦ ╦╔═╗╦  ╦╔═╗",
		"║║║╠═╣╚╗╔╝║╣",
		"╚╩╝╩ ╩ ╚╝ ╚═╝",
	}

	// Project info with real-time elapsed time calculation
	pipelineStart := time.Unix(0, ctx.PipelineStartTime)
	elapsed := time.Since(pipelineStart)

	pipelineLabel := ctx.PipelineName
	if ctx.PipelineID != "" && ctx.PipelineID != ctx.PipelineName {
		pipelineLabel = ctx.PipelineID
	}
	// Build compact progress summary for third line
	progressLine := fmt.Sprintf("Progress: %d%% Step %d/%d", ctx.OverallProgress, ctx.CurrentStepNum, ctx.TotalSteps)
	{
		var parts []string
		for _, stepID := range ctx.StepOrder {
			if state, exists := ctx.StepStatuses[stepID]; exists && state == StateRunning {
				parts = append(parts, stepID)
			}
		}
		runningCount := len(parts)
		var counts []string
		if runningCount > 0 {
			counts = append(counts, fmt.Sprintf("%d running", runningCount))
		}
		if ctx.CompletedSteps > 0 {
			counts = append(counts, fmt.Sprintf("%d ok", ctx.CompletedSteps))
		}
		if ctx.FailedSteps > 0 {
			counts = append(counts, fmt.Sprintf("%d fail", ctx.FailedSteps))
		}
		if len(counts) > 0 {
			progressLine += " (" + strings.Join(counts, ", ") + ")"
		}
	}

	projectLines := []string{
		fmt.Sprintf("Pipeline: %s", pipelineLabel),
		fmt.Sprintf("Elapsed:  %s", formatElapsedWithTokens(ctx, elapsed)),
		progressLine,
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

// RenderPipelineSteps renders all steps with status icons, spinners,
// tool activity, deliverables, contracts, and handover tree connectors.
func RenderPipelineSteps(ctx *PipelineContext) string {
	var steps []string

	// Iterate ALL steps in pipeline definition order
	for _, stepID := range ctx.StepOrder {
		state, exists := ctx.StepStatuses[stepID]
		if !exists {
			state = StateNotStarted
		}

		// Look up persona for this step
		persona := ""
		if ctx.StepPersonas != nil {
			persona = ctx.StepPersonas[stepID]
		}

		switch state {
		case StateCompleted:
			steps = append(steps, renderCompletedStep(ctx, stepID, persona)...)

		case StateRunning:
			steps = append(steps, renderRunningStep(ctx, stepID, persona)...)

		case StateFailed:
			stepLine := fmt.Sprintf("✗ %s", stepID)
			if persona != "" {
				stepLine += fmt.Sprintf(" (%s)", persona)
			}
			if ctx.StepDurations != nil {
				if durationMs, exists := ctx.StepDurations[stepID]; exists {
					durationText := fmt.Sprintf("%.1fs", float64(durationMs)/1000.0)
					stepLine += fmt.Sprintf(" (%s)", durationText)
				}
			}
			stepLine = lipgloss.NewStyle().Foreground(colorError).Render(stepLine)
			steps = append(steps, stepLine)

		case StateSkipped:
			stepLine := fmt.Sprintf("— %s", stepID)
			if persona != "" {
				stepLine += fmt.Sprintf(" (%s)", persona)
			}
			stepLine = lipgloss.NewStyle().Foreground(colorMuted).Render(stepLine)
			steps = append(steps, stepLine)

		case StateCancelled:
			stepLine := fmt.Sprintf("⊘ %s", stepID)
			if persona != "" {
				stepLine += fmt.Sprintf(" (%s)", persona)
			}
			stepLine = lipgloss.NewStyle().Foreground(colorMuted).Render(stepLine)
			steps = append(steps, stepLine)

		default:
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

// RenderPipelineView combines header + steps into a complete pipeline view.
// This is what the "attached mode" in mission control calls.
func RenderPipelineView(ctx *PipelineContext) string {
	if ctx == nil {
		return "Initializing pipeline..."
	}

	header := RenderPipelineHeader(ctx)
	currentStep := RenderPipelineSteps(ctx)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		currentStep,
	)
}

// renderCompletedStep renders a completed step with metadata tree.
func renderCompletedStep(ctx *PipelineContext, stepID, persona string) []string {
	var lines []string

	durationText := "0.0s"
	if ctx.StepDurations != nil {
		if durationMs, exists := ctx.StepDurations[stepID]; exists {
			durationText = fmt.Sprintf("%.1fs", float64(durationMs)/1000.0)
		}
	}
	stepLine := fmt.Sprintf("✓ %s", stepID)
	if persona != "" {
		stepLine += fmt.Sprintf(" (%s)", persona)
	}
	if ctx.StepModels != nil {
		if model, ok := ctx.StepModels[stepID]; ok && model != "" {
			stepLine += fmt.Sprintf(" [%s]", model)
		}
	}
	// Append tokens alongside duration if available
	if ctx.StepTokensIn != nil {
		tIn := ctx.StepTokensIn[stepID]
		tOut := ctx.StepTokensOut[stepID]
		if tIn > 0 || tOut > 0 {
			stepLine += fmt.Sprintf(" (%s, %s in / %s out)", durationText, FormatTokenCount(tIn), FormatTokenCount(tOut))
		} else if ctx.StepTokens != nil {
			if tokens, ok := ctx.StepTokens[stepID]; ok && tokens > 0 {
				stepLine += fmt.Sprintf(" (%s, %s tokens)", durationText, FormatTokenCount(tokens))
			} else {
				stepLine += fmt.Sprintf(" (%s)", durationText)
			}
		} else {
			stepLine += fmt.Sprintf(" (%s)", durationText)
		}
	} else if ctx.StepTokens != nil {
		if tokens, ok := ctx.StepTokens[stepID]; ok && tokens > 0 {
			stepLine += fmt.Sprintf(" (%s, %s tokens)", durationText, FormatTokenCount(tokens))
		} else {
			stepLine += fmt.Sprintf(" (%s)", durationText)
		}
	} else {
		stepLine += fmt.Sprintf(" (%s)", durationText)
	}
	stepLine = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render(stepLine)
	lines = append(lines, stepLine)

	// Collect all metadata lines (deliverables + handover) for tree formatting
	var metadataLines []string

	if ctx.DeliverablesByStep != nil {
		if stepDeliverables, exists := ctx.DeliverablesByStep[stepID]; exists {
			metadataLines = append(metadataLines, stepDeliverables...)
		}
	}

	if ctx.Verbose && ctx.HandoversByStep != nil {
		if info, exists := ctx.HandoversByStep[stepID]; exists {
			for _, path := range info.ArtifactPaths {
				metadataLines = append(metadataLines, fmt.Sprintf("artifact: %s (written)", pathfmt.FileURI(path)))
			}
			if info.ContractStatus != "" {
				status := "✓ valid"
				if info.ContractStatus == "failed" {
					status = "✗ failed"
				} else if info.ContractStatus == "soft_failure" {
					status = "⚠ soft failure"
				}
				schema := info.ContractSchema
				if schema == "" {
					schema = "contract"
				}
				metadataLines = append(metadataLines, fmt.Sprintf("contract: %s %s", schema, status))
			}
			if info.TargetStep != "" {
				metadataLines = append(metadataLines, fmt.Sprintf("handover → %s", info.TargetStep))
			}
		}
	}

	for i, line := range metadataLines {
		connector := "├─"
		if i == len(metadataLines)-1 {
			connector = "└─"
		}
		metaLine := fmt.Sprintf("   %s %s", connector, line)
		metaLine = lipgloss.NewStyle().Foreground(colorMuted).Render(metaLine)
		lines = append(lines, metaLine)
	}

	return lines
}

// renderRunningStep renders a running step with spinner and tool activity.
func renderRunningStep(ctx *PipelineContext, stepID, persona string) []string {
	var lines []string

	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	now := time.Now().UnixMilli()
	frame := (now / 80) % int64(len(spinners))
	icon := spinners[frame]

	stepLine := fmt.Sprintf("%s %s", icon, stepID)
	if persona != "" {
		stepLine += fmt.Sprintf(" (%s)", persona)
	}
	if ctx.StepModels != nil {
		if model, ok := ctx.StepModels[stepID]; ok && model != "" {
			modelInfo := model
			if ctx.StepAdapters != nil {
				if adpt, ok := ctx.StepAdapters[stepID]; ok && adpt != "" {
					modelInfo += " via " + adpt
				}
			}
			stepLine += fmt.Sprintf(" [%s]", modelInfo)
		}
	}
	var stepElapsed time.Duration
	if ctx.StepStartTimes != nil {
		if startNano, ok := ctx.StepStartTimes[stepID]; ok {
			stepElapsed = time.Since(time.Unix(0, startNano))
		}
	}
	if stepElapsed == 0 {
		stepStart := time.Unix(0, ctx.CurrentStepStart)
		stepElapsed = time.Since(stepStart)
	}
	stepLine += fmt.Sprintf(" (%s)", formatElapsed(stepElapsed))

	if ctx.CurrentAction != "" && stepID == ctx.CurrentStepID {
		stepLine += fmt.Sprintf(" • %s", ctx.CurrentAction)
	}

	stepLine = lipgloss.NewStyle().Foreground(colorPrimary).Render(stepLine)
	lines = append(lines, stepLine)

	// Show per-step tool activity when available, fall back to global
	toolName := ""
	toolTarget := ""
	if ctx.StepToolActivity != nil {
		if ta, ok := ctx.StepToolActivity[stepID]; ok {
			toolName = ta[0]
			toolTarget = ta[1]
		}
	}
	if toolName == "" && stepID == ctx.CurrentStepID {
		toolName = ctx.LastToolName
		toolTarget = ctx.LastToolTarget
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
		lines = append(lines, toolLine)
	}

	return lines
}

// formatElapsedWithTokens formats elapsed time and optionally appends total token count.
func formatElapsedWithTokens(ctx *PipelineContext, d time.Duration) string {
	elapsed := formatElapsed(d)
	if ctx.TotalTokensIn > 0 || ctx.TotalTokensOut > 0 {
		return fmt.Sprintf("%s • %s in / %s out", elapsed, FormatTokenCount(ctx.TotalTokensIn), FormatTokenCount(ctx.TotalTokensOut))
	}
	if ctx.TotalTokens > 0 {
		return fmt.Sprintf("%s • %s tokens", elapsed, FormatTokenCount(ctx.TotalTokens))
	}
	return elapsed
}

// ShimmerPosition is the exported wrapper for shimmerPosition.
func ShimmerPosition(logoWidth int, cycleMs int64) float64 {
	return shimmerPosition(logoWidth, cycleMs)
}

// ShimmerColorForChar is the exported wrapper for shimmerColorForChar.
func ShimmerColorForChar(charPos int, shimmerCenter float64) lipgloss.Style {
	return shimmerColorForChar(charPos, shimmerCenter)
}
