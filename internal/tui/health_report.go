package tui

import (
	"fmt"
	"time"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/recinq/wave/internal/meta"
	"github.com/recinq/wave/internal/platform"
)

// Style palette — matches theme.go colors.
var (
	hrCyan  = lipgloss.Color("14")  // Wave primary
	hrWhite = lipgloss.Color("7")   // Primary text
	hrMuted = lipgloss.Color("244") // Secondary/description text
	hrRed   = lipgloss.Color("9")   // Errors
	hrGreen = lipgloss.Color("10")  // Success indicators
)

// Reusable lipgloss styles.
var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(hrCyan)
	labelStyle  = lipgloss.NewStyle().Foreground(hrMuted)
	valueStyle  = lipgloss.NewStyle().Foreground(hrWhite)
	errorStyle  = lipgloss.NewStyle().Foreground(hrRed)
	okStyle     = lipgloss.NewStyle().Foreground(hrGreen)
)

// Status indicators.
const (
	indicatorOK              = "✓"
	indicatorFail            = "✗"
	indicatorAutoInstallable = "⬡"
)

// RenderHealthReport formats a HealthReport for terminal display.
// The returned string is a styled, multi-section text block suitable
// for printing to stdout. It is not interactive (not a Bubble Tea model).
func RenderHealthReport(report *meta.HealthReport) string {
	if report == nil {
		return errorStyle.Render("No health report data available.")
	}

	var sections []string

	sections = append(sections, renderInitSection(report.Init))
	sections = append(sections, renderDependenciesSection(report.Dependencies))
	sections = append(sections, renderCodebaseSection(report.Codebase))
	sections = append(sections, renderPlatformSection(report.Platform))

	if len(report.Errors) > 0 {
		sections = append(sections, renderErrorsSection(report.Errors))
	}

	// Footer: duration.
	footer := labelStyle.Render(fmt.Sprintf("Completed in %s", report.Duration.Round(time.Millisecond)))
	sections = append(sections, footer)

	return strings.Join(sections, "\n\n")
}

// renderInitSection produces the initialization status block.
func renderInitSection(init meta.InitCheckResult) string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("Init"))
	b.WriteByte('\n')

	b.WriteString(formatBoolRow("Manifest found", init.ManifestFound))
	b.WriteByte('\n')
	b.WriteString(formatBoolRow("Manifest valid", init.ManifestValid))
	b.WriteByte('\n')
	b.WriteString(formatRow("Wave version", init.WaveVersion))

	if !init.LastConfigDate.IsZero() {
		b.WriteByte('\n')
		b.WriteString(formatRow("Last config", init.LastConfigDate.Format("2006-01-02 15:04:05")))
	}

	if init.Error != "" {
		b.WriteByte('\n')
		b.WriteString(fmt.Sprintf("  %s %s", errorStyle.Render(indicatorFail), errorStyle.Render(init.Error)))
	}

	return b.String()
}

// renderDependenciesSection produces the tool/skill dependency table.
func renderDependenciesSection(deps meta.DependencyReport) string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("Dependencies"))

	if len(deps.Tools) > 0 {
		b.WriteByte('\n')
		b.WriteString(labelStyle.Render("  Tools"))
		for _, t := range deps.Tools {
			b.WriteByte('\n')
			b.WriteString(formatDepRow(t))
		}
	}

	if len(deps.Skills) > 0 {
		b.WriteByte('\n')
		b.WriteString(labelStyle.Render("  Skills"))
		for _, s := range deps.Skills {
			b.WriteByte('\n')
			b.WriteString(formatDepRow(s))
		}
	}

	if len(deps.Tools) == 0 && len(deps.Skills) == 0 {
		b.WriteByte('\n')
		b.WriteString(labelStyle.Render("  No dependencies detected"))
	}

	return b.String()
}

// renderCodebaseSection produces the codebase metrics block.
func renderCodebaseSection(cb meta.CodebaseMetrics) string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("Codebase"))
	b.WriteByte('\n')

	b.WriteString(formatRow("Recent commits (30d)", fmt.Sprintf("%d", cb.RecentCommits)))
	b.WriteByte('\n')
	b.WriteString(formatRow("Open issues", fmt.Sprintf("%d", cb.OpenIssueCount)))
	b.WriteByte('\n')
	b.WriteString(formatRow("Open PRs", fmt.Sprintf("%d", cb.OpenPRCount)))
	b.WriteByte('\n')
	b.WriteString(formatRow("Branches", fmt.Sprintf("%d", cb.BranchCount)))

	if !cb.LastCommitDate.IsZero() {
		b.WriteByte('\n')
		b.WriteString(formatRow("Last commit", cb.LastCommitDate.Format("2006-01-02 15:04:05")))
	}

	b.WriteByte('\n')
	source := cb.Source
	if source == "" {
		source = "none"
	}
	b.WriteString(formatRow("Data source", source))

	return b.String()
}

// renderPlatformSection produces the platform detection block.
func renderPlatformSection(prof platform.PlatformProfile) string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("Platform"))
	b.WriteByte('\n')

	platformType := string(prof.Type)
	if platformType == "" {
		platformType = "unknown"
	}
	b.WriteString(formatRow("Type", platformType))

	if prof.Owner != "" || prof.Repo != "" {
		b.WriteByte('\n')
		b.WriteString(formatRow("Repository", fmt.Sprintf("%s/%s", prof.Owner, prof.Repo)))
	}

	if prof.PipelineFamily != "" {
		b.WriteByte('\n')
		b.WriteString(formatRow("Pipeline family", prof.PipelineFamily))
	}

	if prof.CLITool != "" {
		b.WriteByte('\n')
		b.WriteString(formatRow("CLI tool", prof.CLITool))
	}

	return b.String()
}

// renderErrorsSection produces the errors block.
func renderErrorsSection(errors []meta.HealthCheckError) string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("Errors"))

	for _, e := range errors {
		b.WriteByte('\n')
		prefix := indicatorFail
		if e.Timeout {
			prefix = "⏱"
		}
		b.WriteString(fmt.Sprintf("  %s %s: %s",
			errorStyle.Render(prefix),
			errorStyle.Render(e.Check),
			valueStyle.Render(e.Message),
		))
	}

	return b.String()
}

// formatRow renders a label: value pair with consistent indentation and styling.
func formatRow(label, value string) string {
	return fmt.Sprintf("  %s %s", labelStyle.Render(fmt.Sprintf("%-22s", label+":")), valueStyle.Render(value))
}

// formatBoolRow renders a boolean status row with check/cross indicators.
func formatBoolRow(label string, ok bool) string {
	if ok {
		return fmt.Sprintf("  %s %s", okStyle.Render(indicatorOK), valueStyle.Render(label))
	}
	return fmt.Sprintf("  %s %s", errorStyle.Render(indicatorFail), valueStyle.Render(label))
}

// formatDepRow renders a single dependency status line.
func formatDepRow(dep meta.DependencyStatus) string {
	var indicator string
	var style lipgloss.Style

	switch {
	case dep.Available:
		indicator = indicatorOK
		style = okStyle
	case dep.AutoInstallable:
		indicator = indicatorAutoInstallable
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	default:
		indicator = indicatorFail
		style = errorStyle
	}

	line := fmt.Sprintf("    %s %s", style.Render(indicator), valueStyle.Render(dep.Name))
	if dep.Message != "" {
		line += " " + labelStyle.Render(dep.Message)
	}
	return line
}
