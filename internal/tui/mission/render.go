package mission

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/meta"
)

// fleetStats aggregates run counts for the header.
type fleetStats struct {
	running   int
	completed int
	failed    int
	stale     int
}

// computeFleetStats computes fleet stats from run snapshots.
func computeFleetStats(runs []RunSnapshot) fleetStats {
	var s fleetStats
	for _, r := range runs {
		switch r.Status {
		case "running":
			s.running++
		case "completed":
			s.completed++
		case "failed":
			s.failed++
		case "stale":
			s.stale++
		}
	}
	return s
}

// spinnerFrames are braille spinner characters for animation.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// currentSpinnerFrame returns the current spinner frame based on time.
func currentSpinnerFrame() string {
	idx := int(time.Now().UnixMilli()/80) % len(spinnerFrames)
	return spinnerFrames[idx]
}

// renderBrandTag renders a compact single-line brand tag.
func renderBrandTag() string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		Render("WAVE")
}

// HealthCheckStatus represents the status of a single health check category.
type HealthCheckStatus struct {
	Name    string // "Wave initialized", "Dependencies verified", etc.
	Done    bool
	Success bool
	Detail  string // "last updated 2h ago", "claude-code, gh, 4 skills", etc.
}

// buildInlineHealthSummary creates a compact one-liner from health check statuses.
func buildInlineHealthSummary(checks []HealthCheckStatus, loading bool) string {
	if loading || len(checks) == 0 {
		return styleMuted.Render(currentSpinnerFrame() + " Loading...")
	}

	allDone := true
	for _, c := range checks {
		if !c.Done {
			allDone = false
			break
		}
	}
	if !allDone {
		return styleMuted.Render(currentSpinnerFrame() + " Loading...")
	}

	var parts []string
	for _, c := range checks {
		var glyph string
		var style lipgloss.Style
		if c.Success {
			glyph = "✓"
			style = styleStatusCompleted
		} else {
			glyph = "✗"
			style = styleStatusFailed
		}
		label := c.Detail
		if label == "" {
			label = c.Name
		}
		parts = append(parts, style.Render(glyph)+" "+label)
	}
	return strings.Join(parts, "  ")
}

// --- Health Phase View ---

// renderHealthPhaseView renders the full-screen health check progress view.
func renderHealthPhaseView(checks []HealthCheckStatus, loading bool, width, height int) string {
	var b strings.Builder

	// Brand header with status
	b.WriteString(renderBrandTag())
	if loading && len(checks) == 0 {
		b.WriteString(styleMuted.Render("         Analyzing system health..."))
	} else if loading {
		b.WriteString(styleMuted.Render("         Analyzing system health..."))
	} else {
		b.WriteString(styleMuted.Render("         Health analysis complete"))
	}
	b.WriteString("\n\n")

	if len(checks) == 0 && loading {
		b.WriteString("  " + currentSpinnerFrame() + " " + styleMuted.Render("Starting health checks..."))
		b.WriteByte('\n')
		return b.String()
	}

	for _, c := range checks {
		var glyph string
		if !c.Done {
			glyph = styleStatusRunning.Render(currentSpinnerFrame())
		} else if c.Success {
			glyph = styleStatusCompleted.Render("✓")
		} else {
			glyph = styleStatusFailed.Render("✗")
		}

		line := fmt.Sprintf("  %s %s", glyph, c.Name)
		if c.Detail != "" {
			line += styleMuted.Render(" — " + c.Detail)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}

	return b.String()
}

// --- Proposals View (full-screen, not overlay) ---

// renderProposalsView renders proposals as a full-screen view.
func renderProposalsView(proposals []meta.PipelineProposal, cursor int, selected map[int]bool, skipped map[int]bool, pipelineNames []string, healthSummary string, width, height int) string {
	var b strings.Builder

	// Brand header
	b.WriteString(renderBrandTag())
	selectedCount := 0
	for i, sel := range selected {
		if sel && !skipped[i] {
			selectedCount++
		}
	}
	if selectedCount > 0 {
		b.WriteString(styleMuted.Render(fmt.Sprintf("         %d proposals — %d selected", len(proposals), selectedCount)))
	} else {
		b.WriteString(styleMuted.Render(fmt.Sprintf("         %d proposals from health analysis", len(proposals))))
	}
	b.WriteString("\n\n")

	// Inline health summary
	if healthSummary != "" {
		b.WriteString("  Health: " + healthSummary)
		b.WriteString("\n\n")
	}

	if len(proposals) == 0 {
		b.WriteString(styleMuted.Render("  No proposals available."))
		b.WriteByte('\n')
		b.WriteByte('\n')
		if len(pipelineNames) > 0 {
			b.WriteString(styleMuted.Render("  Available pipelines:"))
			b.WriteByte('\n')
			for _, name := range pipelineNames {
				b.WriteString(styleMuted.Render("  • " + name))
				b.WriteByte('\n')
			}
			b.WriteByte('\n')
		}
		b.WriteString(styleMuted.Render("  n: launch any pipeline    Tab: fleet view"))
		b.WriteByte('\n')
		return b.String()
	}

	// Proposal list
	for i, p := range proposals {
		if skipped[i] {
			b.WriteString(styleMuted.Render(fmt.Sprintf("  ── %s ── skipped ──", strings.Join(p.Pipelines, ", "))))
			b.WriteByte('\n')
			b.WriteByte('\n')
			continue
		}

		// Cursor prefix
		prefix := "  "
		if i == cursor {
			prefix = styleCursor.Render("▸ ")
		}

		// Selection checkbox
		checkbox := "○"
		if selected[i] {
			checkbox = styleStatusCompleted.Render("●")
		}

		// Type tag
		var typeTag string
		switch p.Type {
		case meta.ProposalSingle:
			typeTag = styleStatusRunning.Render("[single]")
		case meta.ProposalSequence:
			typeTag = styleStatusQueued.Render("[sequence]")
		case meta.ProposalParallel:
			typeTag = styleStatusCompleted.Render("[parallel]")
		}

		// Pipeline names
		names := strings.Join(p.Pipelines, " → ")
		if p.Type == meta.ProposalParallel {
			names = strings.Join(p.Pipelines, " | ")
		}

		// Priority badge
		var priorityBadge string
		switch {
		case p.Priority <= 1:
			priorityBadge = styleWarning.Render("high")
		case p.Priority <= 3:
			priorityBadge = styleStatusQueued.Render("medium")
		default:
			priorityBadge = styleMuted.Render("low")
		}

		b.WriteString(fmt.Sprintf("%s%s %s %s  Priority: %s\n",
			prefix, checkbox, typeTag, names, priorityBadge))

		// Rationale
		if p.Rationale != "" {
			b.WriteString(styleMuted.Render(fmt.Sprintf("   %s\n", p.Rationale)))
		}

		// Missing deps warning
		if !p.DepsReady && len(p.MissingDeps) > 0 {
			b.WriteString(styleStatusFailed.Render(fmt.Sprintf("   ⚠ Missing: %s\n", strings.Join(p.MissingDeps, ", "))))
		}

		b.WriteByte('\n')
	}

	// DAG preview for selected proposals (or cursor proposal if none selected)
	dag := renderDAGPreview(proposals, cursor, selected, skipped)
	if dag != "" {
		b.WriteString(styleMuted.Render("─── DAG Preview ───"))
		b.WriteByte('\n')
		b.WriteString(dag)
		b.WriteByte('\n')
	}

	// Execution plan for multiple selected proposals
	plan := renderExecutionPlan(proposals, selected, skipped)
	if plan != "" {
		b.WriteString(plan)
		b.WriteByte('\n')
	}

	return b.String()
}

// renderExecutionPlan renders a composed execution plan when multiple proposals are selected.
func renderExecutionPlan(proposals []meta.PipelineProposal, selected map[int]bool, skipped map[int]bool) string {
	// Only show when 2+ proposals are selected
	var active []meta.PipelineProposal
	for i, p := range proposals {
		if selected[i] && !skipped[i] {
			active = append(active, p)
		}
	}
	if len(active) < 2 {
		return ""
	}

	var b strings.Builder
	b.WriteString(styleMuted.Render("─── Selected Execution Plan ───"))
	b.WriteByte('\n')

	totalPipelines := 0
	for step, p := range active {
		totalPipelines += len(p.Pipelines)
		var label string
		switch p.Type {
		case meta.ProposalSequence:
			label = strings.Join(p.Pipelines, " → ")
		case meta.ProposalParallel:
			label = strings.Join(p.Pipelines, " | ")
		case meta.ProposalSingle:
			label = p.Pipelines[0]
		}
		b.WriteString(fmt.Sprintf("%d. %s", step+1, label))
		b.WriteByte('\n')
	}

	b.WriteString(styleMuted.Render(fmt.Sprintf("Total: %d proposals, %d pipelines", len(active), totalPipelines)))
	b.WriteByte('\n')

	return b.String()
}

// renderDAGPreview renders a text-based DAG showing execution order.
func renderDAGPreview(proposals []meta.PipelineProposal, cursor int, selected map[int]bool, skipped map[int]bool) string {
	// Collect proposals to show: selected ones, or cursor if none selected
	var active []meta.PipelineProposal
	hasSelected := false
	for i := range selected {
		if selected[i] && !skipped[i] {
			hasSelected = true
			break
		}
	}
	if hasSelected {
		for i, p := range proposals {
			if selected[i] && !skipped[i] {
				active = append(active, p)
			}
		}
	} else if cursor >= 0 && cursor < len(proposals) && !skipped[cursor] {
		active = append(active, proposals[cursor])
	}

	if len(active) == 0 {
		return ""
	}

	var b strings.Builder
	for _, p := range active {
		switch p.Type {
		case meta.ProposalSequence:
			// Show: pipeline1 ──→ pipeline2 ──→ pipeline3
			b.WriteString(strings.Join(p.Pipelines, " ──→ "))
			b.WriteByte('\n')
			// Show artifact flow between sequential pipelines
			for i := 0; i < len(p.Pipelines)-1; i++ {
				padding := 0
				for j := 0; j <= i; j++ {
					padding += len(p.Pipelines[j]) + 5 // " ──→ " = 5
				}
				b.WriteString(strings.Repeat(" ", padding))
				b.WriteString(styleMuted.Render("↑ artifacts"))
				b.WriteByte('\n')
			}

		case meta.ProposalParallel:
			// Show fork/join
			for i, name := range p.Pipelines {
				var connector string
				switch {
				case i == 0:
					connector = "┌─"
				case i == len(p.Pipelines)-1:
					connector = "└─"
				default:
					connector = "├─"
				}
				label := fmt.Sprintf("%s %s", connector, name)
				if i == 0 {
					label += styleMuted.Render("      (parallel)")
				}
				b.WriteString(label)
				b.WriteByte('\n')
			}

		case meta.ProposalSingle:
			b.WriteString(p.Pipelines[0])
			b.WriteByte('\n')
		}
	}

	return b.String()
}

// --- Form Overlay ---

// renderFormOverlay renders an embedded huh form inside a centered modal overlay.
func renderFormOverlay(formView, title string, width, height int) string {
	return renderOverlay(formView, title, width, height)
}

// --- Fleet rendering (list + preview + two-pane) ---

// renderListPane renders the left pane: brand + health + run list + actions.
func renderListPane(runs []RunSnapshot, cursor int, scrollOff int, filter string, filterMode bool, healthSummary string, proposalCount int, width, height int) string {
	var b strings.Builder

	// Brand line
	b.WriteString(renderBrandTag())
	b.WriteByte('\n')
	// Health summary
	if healthSummary != "" {
		b.WriteString("  " + healthSummary)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	// Filter visible runs
	visible := filterRuns(runs, filter)

	// Count active runs
	activeCount := 0
	for _, r := range visible {
		if r.isActive() {
			activeCount++
		}
	}

	idle := activeCount == 0 && !filterMode

	if len(visible) == 0 {
		// Empty state — no runs at all
		b.WriteString(styleMuted.Render("  No pipeline runs yet."))
		b.WriteByte('\n')
		b.WriteByte('\n')
		b.WriteString(styleHeader.Render("  Get started"))
		b.WriteByte('\n')
		b.WriteString(styleCursor.Render("  n") + styleMuted.Render("  Launch a pipeline"))
		b.WriteByte('\n')
		if proposalCount > 0 {
			b.WriteString(styleCursor.Render("  p") + styleMuted.Render(fmt.Sprintf("  View %d proposals", proposalCount)))
			b.WriteByte('\n')
		}
		b.WriteString(styleCursor.Render("  h") + styleMuted.Render("  Health report"))
		b.WriteByte('\n')
		b.WriteString(styleCursor.Render("  ?") + styleMuted.Render("  All keybindings"))
		b.WriteByte('\n')
	} else {
		// Idle call to action when nothing is running
		if idle {
			var actions []string
			actions = append(actions, styleCursor.Render("n")+styleMuted.Render(":launch"))
			if proposalCount > 0 {
				actions = append(actions, styleCursor.Render("p")+styleMuted.Render(fmt.Sprintf(":%d proposals", proposalCount)))
			}
			actions = append(actions, styleCursor.Render("?")+styleMuted.Render(":help"))
			b.WriteString("  " + strings.Join(actions, "  "))
			b.WriteByte('\n')
			b.WriteByte('\n')
		}

		// Run list
		headerLines := 4 // brand + health + blank + (idle actions or not)
		if idle {
			headerLines = 6 // + actions + blank
		}
		footerLines := 4 // blank + stats + filter + margin
		maxRows := height - headerLines - footerLines
		if maxRows < 1 {
			maxRows = 1
		}

		maxArchivedVisible := 10
		if idle {
			maxArchivedVisible = 5 // Show fewer archived runs when idle
		}

		end := scrollOff + maxRows
		if end > len(visible) {
			end = len(visible)
		}

		wroteArchiveDivider := false
		archivedCount := 0
		for i := scrollOff; i < end; i++ {
			r := visible[i]

			// Archive divider
			if !wroteArchiveDivider && !r.isActive() {
				hasActiveBefore := false
				for j := 0; j < i; j++ {
					if visible[j].isActive() {
						hasActiveBefore = true
						break
					}
				}
				if hasActiveBefore || idle {
					label := "Archive"
					if idle {
						label = "Recent"
					}
					b.WriteString(styleMuted.Render("  ── " + label + " ──"))
					b.WriteByte('\n')
				}
				wroteArchiveDivider = true
			}

			// Limit archived runs
			if !r.isActive() {
				archivedCount++
				if archivedCount > maxArchivedVisible {
					remaining := 0
					for j := i; j < end; j++ {
						if !visible[j].isActive() {
							remaining++
						}
					}
					if remaining > 0 {
						b.WriteString(styleMuted.Render(fmt.Sprintf("  … %d more archived runs", remaining)))
						b.WriteByte('\n')
					}
					break
				}
			}

			glyph, style := statusGlyph(r.Status)

			name := r.PipelineName
			if name == "" {
				name = r.RunID
			}
			maxElapsed := 7
			maxName := width - 5 - maxElapsed
			if maxName < 8 {
				maxName = 8
			}
			if len(name) > maxName {
				name = name[:maxName-3] + "..."
			}

			elapsed := "—"
			if r.Elapsed > 0 {
				elapsed = formatDuration(r.Elapsed)
			}

			prefix := "  "
			if i == cursor {
				prefix = styleCursor.Render("▸ ")
			}

			elapsedPad := strings.Repeat(" ", maxElapsed-len(elapsed))

			line := fmt.Sprintf("%s%s %-*s %s",
				prefix,
				style.Render(glyph),
				maxName, name,
				elapsedPad+styleMuted.Render(elapsed),
			)
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}

	// Stats footer
	stats := computeFleetStats(runs)
	var statParts []string
	if stats.running > 0 {
		statParts = append(statParts, styleStatusRunning.Render(fmt.Sprintf("%d running", stats.running)))
	}
	if stats.completed > 0 {
		statParts = append(statParts, styleStatusCompleted.Render(fmt.Sprintf("%d ok", stats.completed)))
	}
	if stats.failed > 0 {
		statParts = append(statParts, styleStatusFailed.Render(fmt.Sprintf("%d fail", stats.failed)))
	}
	if stats.stale > 0 {
		statParts = append(statParts, styleMuted.Render(fmt.Sprintf("%d stale", stats.stale)))
	}
	if len(statParts) > 0 {
		total := len(runs)
		header := styleMuted.Render(fmt.Sprintf("%d runs:", total))
		b.WriteByte('\n')
		b.WriteString("  " + header + "  " + strings.Join(statParts, "  "))
		b.WriteByte('\n')
	}

	// Filter indicator
	if filterMode || filter != "" {
		b.WriteString(styleMuted.Render(fmt.Sprintf("  /:filter %s▌", filter)))
		b.WriteByte('\n')
	}

	return b.String()
}


// renderPreviewPane renders the right pane: selected run detail.
func renderPreviewPane(rc *RunContext, run *RunSnapshot, width, height int) string {
	if run == nil {
		return styleMuted.Render("\n  Select a run to preview")
	}

	var b strings.Builder

	// Pipeline header
	header := fmt.Sprintf("Pipeline: %s", run.PipelineName)
	b.WriteString(styleHeader.Render(header))
	b.WriteByte('\n')

	// Run ID for finished runs — needed for wave chat
	if !run.isActive() && run.Status != "stale" {
		b.WriteString(styleMuted.Render("Run:      " + run.RunID))
		b.WriteByte('\n')
	}

	// Stale run message
	if run.Status == "stale" {
		age := formatDuration(time.Since(run.StartedAt))
		b.WriteString(styleWarning.Render("⚠ This run appears stale"))
		b.WriteByte('\n')
		b.WriteString(styleMuted.Render(fmt.Sprintf("  Started %s ago with no recent updates.", age)))
		b.WriteByte('\n')
		b.WriteString(styleMuted.Render("  It may be from a crashed session."))
		b.WriteByte('\n')
		b.WriteByte('\n')
		b.WriteString(styleMuted.Render("  Actions:"))
		b.WriteByte('\n')
		b.WriteString(styleMuted.Render("  • This run will be cleaned up automatically"))
		b.WriteByte('\n')
		b.WriteString(styleMuted.Render("  • Press n to launch a fresh run"))
		b.WriteByte('\n')
		return b.String()
	}

	// Elapsed + tokens
	elapsed := "—"
	if run.Elapsed > 0 {
		elapsed = formatDuration(run.Elapsed)
	}
	tokenStr := ""
	if run.TokensIn > 0 || run.TokensOut > 0 {
		tokenStr = fmt.Sprintf(" • %s in / %s out",
			display.FormatTokenCount(run.TokensIn),
			display.FormatTokenCount(run.TokensOut),
		)
	} else if run.TotalTokens > 0 {
		tokenStr = fmt.Sprintf(" • %s tokens", display.FormatTokenCount(run.TotalTokens))
	}
	b.WriteString(styleMuted.Render(fmt.Sprintf("Elapsed:  %s%s", elapsed, tokenStr)))
	b.WriteByte('\n')

	// Progress
	if run.TotalSteps > 0 {
		b.WriteString(styleMuted.Render(fmt.Sprintf("Progress: %d%% Step %d/%d", run.Progress, run.CompletedSteps, run.TotalSteps)))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	// Step detail via RunContext if available and has step data
	if rc != nil && len(rc.Ctx.StepOrder) > 0 {
		stepView := display.RenderPipelineSteps(rc.Ctx)
		b.WriteString(stepView)
	} else {
		// Fallback for archived runs or runs without step data
		glyph, style := statusGlyph(run.Status)
		b.WriteString(fmt.Sprintf("  Status:  %s %s", style.Render(glyph), run.Status))
		b.WriteByte('\n')
		if run.CurrentStep != "" {
			b.WriteString(styleMuted.Render(fmt.Sprintf("  Step:    %s", run.CurrentStep)))
			b.WriteByte('\n')
		}
		if run.TotalSteps > 0 {
			b.WriteString(styleMuted.Render(fmt.Sprintf("  Steps:   %d/%d", run.CompletedSteps, run.TotalSteps)))
			b.WriteByte('\n')
		}
		if run.ErrorMessage != "" {
			b.WriteByte('\n')
			b.WriteString(styleStatusFailed.Render("  Error: " + run.ErrorMessage))
			b.WriteByte('\n')
		}
		if run.Status == "completed" || run.Status == "failed" || run.Status == "cancelled" {
			b.WriteByte('\n')
			b.WriteString(styleMuted.Render("  (Step detail not available for this run)"))
		}
	}

	return b.String()
}

// renderTwoPaneLayout joins list and preview panes side-by-side.
func renderTwoPaneLayout(list, preview string, totalWidth, totalHeight int) string {
	if totalWidth < 50 {
		// Narrow: list only
		return list
	}

	var listWidth, previewWidth int
	if totalWidth < 80 {
		listWidth = int(float64(totalWidth) * 0.40)
		previewWidth = totalWidth - listWidth - 1 // 1 for divider
	} else {
		listWidth = int(float64(totalWidth) * 0.35)
		previewWidth = totalWidth - listWidth - 1
	}

	listPane := lipgloss.NewStyle().
		Width(listWidth).
		Height(totalHeight).
		Render(list)

	divider := lipgloss.NewStyle().
		Foreground(colorMuted).
		Width(1).
		Height(totalHeight).
		Render(strings.Repeat("│\n", totalHeight))

	previewPane := lipgloss.NewStyle().
		Width(previewWidth).
		Height(totalHeight).
		PaddingLeft(1).
		Render(preview)

	return lipgloss.JoinHorizontal(lipgloss.Top, listPane, divider, previewPane)
}

// renderOverlay renders a centered modal overlay.
func renderOverlay(content, title string, width, height int) string {
	maxW := width - 8
	if maxW > 80 {
		maxW = 80
	}
	if maxW < 30 {
		maxW = 30
	}

	maxH := height - 6
	if maxH < 5 {
		maxH = 5
	}

	// Truncate content to fit
	lines := strings.Split(content, "\n")
	if len(lines) > maxH {
		lines = lines[:maxH]
		lines = append(lines, styleMuted.Render("  ↓ more..."))
	}
	body := strings.Join(lines, "\n")

	box := styleOverlayBorder.
		Width(maxW).
		Render(styleHeader.Render(title) + "\n\n" + body)

	// Center the overlay
	boxWidth := lipgloss.Width(box)
	boxHeight := lipgloss.Height(box)

	padLeft := (width - boxWidth) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	padTop := (height - boxHeight) / 3
	if padTop < 0 {
		padTop = 0
	}

	var b strings.Builder
	for range padTop {
		b.WriteByte('\n')
	}
	for _, line := range strings.Split(box, "\n") {
		b.WriteString(strings.Repeat(" ", padLeft))
		b.WriteString(line)
		b.WriteByte('\n')
	}

	return b.String()
}

// renderHealthOverlay renders the health report as an overlay.
func renderHealthOverlay(content string, scrollOff int, width, height int) string {
	if content == "" {
		return renderOverlay(
			styleMuted.Render("No health data yet.\nPress R to run health checks."),
			"System Health",
			width, height,
		)
	}
	lines := strings.Split(content, "\n")
	if scrollOff >= len(lines) {
		scrollOff = max(0, len(lines)-1)
	}
	visible := lines[scrollOff:]
	return renderOverlay(strings.Join(visible, "\n"), "System Health", width, height)
}

// renderHelpOverlay renders a full keybinding reference overlay.
func renderHelpOverlay(width, height int) string {
	var b strings.Builder

	sections := []struct {
		title string
		keys  []struct{ key, desc string }
	}{
		{
			title: "Health Phase",
			keys: []struct{ key, desc string }{
				{"q", "Quit"},
			},
		},
		{
			title: "Proposals View",
			keys: []struct{ key, desc string }{
				{"j / k", "Navigate proposals"},
				{"Space", "Toggle selection"},
				{"Enter", "Launch selected"},
				{"s", "Skip proposal"},
				{"m", "Modify input"},
				{"n", "Launch any pipeline"},
				{"Tab", "Switch to fleet view"},
			},
		},
		{
			title: "Fleet View",
			keys: []struct{ key, desc string }{
				{"j / k", "Navigate runs"},
				{"Enter", "Attach to selected run"},
				{"n", "Launch new pipeline"},
				{"c", "Cancel selected run"},
				{"r", "Retry failed run"},
				{"o", "Open chat for finished run"},
				{"/", "Filter runs by name"},
				{"p / Tab", "Switch to proposals"},
				{"h", "Open health report"},
			},
		},
		{
			title: "Attached View",
			keys: []struct{ key, desc string }{
				{"Esc", "Detach (back to fleet)"},
				{"c", "Cancel run"},
				{"o", "Open chat for this run"},
			},
		},
		{
			title: "Global",
			keys: []struct{ key, desc string }{
				{"?", "Toggle this help"},
				{"q", "Quit"},
				{"Ctrl+C", "Force quit"},
			},
		},
	}

	for i, section := range sections {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(styleHeader.Render(section.title))
		b.WriteByte('\n')
		for _, k := range section.keys {
			b.WriteString(fmt.Sprintf("  %-12s %s\n", styleCursor.Render(k.key), k.desc))
		}
	}

	return renderOverlay(b.String(), "Keybindings", width, height)
}

// renderHelpBar renders the bottom help bar.
func renderHelpBar(view ViewID, overlay OverlayID, activeForm bool, width int) string {
	var help string
	if activeForm {
		help = overlayHelp(OverlayForm)
	} else if overlay != OverlayNone {
		help = overlayHelp(overlay)
	} else {
		help = viewHelp(view)
	}
	return styleHelp.Render(" " + help)
}

// filterRuns returns runs matching the filter string.
func filterRuns(runs []RunSnapshot, filter string) []RunSnapshot {
	if filter == "" {
		return runs
	}
	var filtered []RunSnapshot
	for _, r := range runs {
		if strings.Contains(strings.ToLower(r.PipelineName), strings.ToLower(filter)) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// renderProgressBar renders a small progress bar.
func renderProgressBar(pct int, width int) string {
	if pct <= 0 {
		return strings.Repeat("░", width) + "  0%"
	}
	filled := pct * width / 100
	if filled > width {
		filled = width
	}
	empty := width - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return fmt.Sprintf("%s %3d%%", bar, pct)
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm%ds", m, s)
}

// statusLabel returns a short status label.
func statusLabel(status string) string {
	switch status {
	case "running":
		return "run"
	case "completed":
		return "done"
	case "failed":
		return "fail"
	case "cancelled":
		return "stop"
	case "queued", "pending":
		return "wait"
	default:
		return status
	}
}
