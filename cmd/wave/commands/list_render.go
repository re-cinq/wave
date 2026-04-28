package commands

import (
	"fmt"
	"strings"

	"github.com/recinq/wave/internal/display"
	"github.com/recinq/wave/internal/listing"
)

// renderPipelinesTable prints pipelines in the standard tabular form.
func renderPipelinesTable(pipelines []listing.PipelineInfo) {
	f := display.NewFormatter()
	sectionHeader(f, "Pipelines")

	if len(pipelines) == 0 {
		fmt.Printf("  %s\n", f.Muted("(none found in "+listing.DefaultPipelineDir+"/)"))
		return
	}

	for _, p := range pipelines {
		stepBadge := f.Muted(fmt.Sprintf("[%d steps]", len(p.Steps)))
		fmt.Printf("\n  %s %s\n", f.Primary(p.Name), stepBadge)

		if p.Description != "" {
			fmt.Printf("    %s\n", f.Muted(p.Description))
		}

		if len(p.Steps) > 0 {
			fmt.Printf("    %s\n", formatStepsFlow(p.Steps, f))
		}
	}

	fmt.Println()
}

// formatStepsFlow renders pipeline steps as a visual flow with arrows.
func formatStepsFlow(steps []string, f *display.Formatter) string {
	if len(steps) == 0 {
		return ""
	}

	var parts []string
	for i, step := range steps {
		if i == 0 {
			parts = append(parts, f.Success("○")+f.Muted(" "+step))
		} else {
			parts = append(parts, f.Muted("→ "+step))
		}
	}

	return strings.Join(parts, " ")
}

// renderPersonasTable prints personas in the standard tabular form.
func renderPersonasTable(personas []listing.PersonaInfo) {
	f := display.NewFormatter()
	sectionHeader(f, "Personas")

	if len(personas) == 0 {
		fmt.Printf("  %s\n", f.Muted("(none defined)"))
		return
	}

	for _, p := range personas {
		fmt.Printf("\n  %s\n", f.Primary(p.Name))

		metaParts := []string{
			fmt.Sprintf("adapter: %s", p.Adapter),
			fmt.Sprintf("temp: %.1f", p.Temperature),
		}
		if perm := formatPermissionSummary(p.AllowedTools, p.DeniedTools); perm != "" {
			metaParts = append(metaParts, perm)
		}
		fmt.Printf("    %s\n", f.Muted(strings.Join(metaParts, " • ")))

		if p.Description != "" {
			fmt.Printf("    %s\n", p.Description)
		}
	}

	fmt.Println()
}

// formatPermissionSummary creates a concise summary of persona permissions.
func formatPermissionSummary(allowed, denied []string) string {
	allowCount := len(allowed)
	denyCount := len(denied)

	if allowCount == 0 && denyCount == 0 {
		return "tools:(default)"
	}

	parts := []string{}
	if allowCount > 0 {
		parts = append(parts, fmt.Sprintf("allow:%d", allowCount))
	}
	if denyCount > 0 {
		parts = append(parts, fmt.Sprintf("deny:%d", denyCount))
	}

	return strings.Join(parts, " ")
}

// renderAdaptersTable prints adapters in the standard tabular form.
func renderAdaptersTable(adapters []listing.AdapterInfo) {
	f := display.NewFormatter()
	sectionHeader(f, "Adapters")

	if len(adapters) == 0 {
		fmt.Printf("  %s\n", f.Muted("(none defined)"))
		return
	}

	for _, a := range adapters {
		var statusIcon string
		if a.Available {
			statusIcon = f.Success("✓")
		} else {
			statusIcon = f.Error("✗")
		}

		fmt.Printf("\n  %s %s\n", statusIcon, f.Primary(a.Name))

		metaParts := []string{
			fmt.Sprintf("binary: %s", a.Binary),
			fmt.Sprintf("mode: %s", a.Mode),
			fmt.Sprintf("format: %s", a.OutputFormat),
		}
		fmt.Printf("    %s\n", f.Muted(strings.Join(metaParts, " • ")))

		if !a.Available {
			fmt.Printf("    %s\n", f.Error("binary not found in PATH"))
		}
	}

	fmt.Println()
}

// renderRunsTable displays run information in table format.
func renderRunsTable(runs []listing.RunInfo) {
	f := display.NewFormatter()
	termWidth := display.GetTerminalWidth()

	sepWidth := termWidth
	if sepWidth < 40 {
		sepWidth = 40
	}

	fmt.Println()
	fmt.Printf("%s\n", f.Colorize("Recent Pipeline Runs", "\033[1;37m"))
	fmt.Printf("%s\n", f.Muted(strings.Repeat("─", sepWidth)))

	if len(runs) == 0 {
		fmt.Printf("  %s\n\n", f.Muted("(no runs found)"))
		return
	}

	const (
		statusWidth   = 12
		startedWidth  = 20
		durationWidth = 10
		indent        = 2
		gaps          = 8 // 4 gaps x 2 chars each
	)

	fixedWidth := indent + statusWidth + startedWidth + durationWidth + gaps
	remaining := termWidth - fixedWidth
	if remaining < 20 {
		remaining = 20
	}

	runIDWidth := remaining * 60 / 100
	pipelineWidth := remaining - runIDWidth
	if runIDWidth < 10 {
		runIDWidth = 10
	}
	if pipelineWidth < 8 {
		pipelineWidth = 8
	}

	fmt.Printf("  %s  %s  %s  %s  %s\n",
		f.Muted(fmt.Sprintf("%-*s", runIDWidth, "RUN_ID")),
		f.Muted(fmt.Sprintf("%-*s", pipelineWidth, "PIPELINE")),
		f.Muted(fmt.Sprintf("%-*s", statusWidth, "STATUS")),
		f.Muted(fmt.Sprintf("%-*s", startedWidth, "STARTED")),
		f.Muted("DURATION"),
	)

	for _, run := range runs {
		runID := run.RunID
		pipeline := run.Pipeline

		if len(runID) > runIDWidth && runIDWidth > 3 {
			runID = runID[:runIDWidth-3] + "..."
		}
		if len(pipeline) > pipelineWidth && pipelineWidth > 3 {
			pipeline = pipeline[:pipelineWidth-3] + "..."
		}

		var statusStr string
		switch strings.ToLower(run.Status) {
		case "completed":
			statusStr = f.Success(fmt.Sprintf("%-*s", statusWidth, run.Status))
		case "failed":
			statusStr = f.Error(fmt.Sprintf("%-*s", statusWidth, run.Status))
		case "running":
			statusStr = f.Primary(fmt.Sprintf("%-*s", statusWidth, run.Status))
		case "cancelled":
			statusStr = f.Warning(fmt.Sprintf("%-*s", statusWidth, run.Status))
		default:
			statusStr = f.Muted(fmt.Sprintf("%-*s", statusWidth, run.Status))
		}

		fmt.Printf("  %-*s  %-*s  %s  %-*s  %s\n",
			runIDWidth, runID, pipelineWidth, pipeline, statusStr,
			startedWidth, run.StartedAt, f.Muted(run.Duration))
	}

	fmt.Println()
}

// renderContractsTable displays contract information in table format.
func renderContractsTable(contracts []listing.ContractInfo) {
	f := display.NewFormatter()
	sectionHeader(f, "Contracts")

	if len(contracts) == 0 {
		fmt.Printf("  %s\n", f.Muted("(none found in "+listing.DefaultContractDir+"/)"))
		fmt.Println()
		return
	}

	for _, contract := range contracts {
		typeBadge := f.Muted(fmt.Sprintf("[%s]", contract.Type))
		fmt.Printf("\n  %s %s\n", f.Primary(contract.Name), typeBadge)

		if len(contract.UsedBy) == 0 {
			fmt.Printf("    %s\n", f.Muted("(unused)"))
			continue
		}
		fmt.Printf("    %s\n", f.Muted("used by:"))
		for _, usage := range contract.UsedBy {
			usageStr := fmt.Sprintf("%s → %s", usage.Pipeline, usage.Step)
			if usage.Persona != "" {
				usageStr += fmt.Sprintf(" (%s)", usage.Persona)
			}
			fmt.Printf("      %s %s\n", f.Success("•"), usageStr)
		}
	}

	fmt.Println()
}

// renderSkillsTable displays skill information in table format.
func renderSkillsTable(skills []listing.SkillInfo) {
	f := display.NewFormatter()
	sectionHeader(f, "Skills")

	if len(skills) == 0 {
		fmt.Printf("  %s\n", f.Muted("(none defined)"))
		fmt.Println()
		return
	}

	for _, skill := range skills {
		var statusIcon string
		if skill.Installed {
			statusIcon = f.Success("✓")
		} else {
			statusIcon = f.Error("✗")
		}

		fmt.Printf("\n  %s %s\n", statusIcon, f.Primary(skill.Name))

		metaParts := []string{}
		if skill.Check != "" {
			metaParts = append(metaParts, fmt.Sprintf("check: %s", skill.Check))
		}
		if skill.Install != "" {
			metaParts = append(metaParts, fmt.Sprintf("install: %s", skill.Install))
		}
		if len(metaParts) > 0 {
			fmt.Printf("    %s\n", f.Muted(strings.Join(metaParts, " • ")))
		}

		if len(skill.UsedBy) > 0 {
			fmt.Printf("    %s %s\n", f.Muted("used by:"), strings.Join(skill.UsedBy, ", "))
		}
	}

	fmt.Println()
}

// renderCompositionsTable displays composition pipelines in table format.
func renderCompositionsTable(compositions []listing.CompositionInfo) {
	if len(compositions) == 0 {
		fmt.Println("No composition pipelines found.")
		return
	}

	f := display.NewFormatter()
	sectionHeader(f, "Composition Pipelines")

	for _, c := range compositions {
		fmt.Printf("\n  %s\n", f.Primary(c.Name))
		if c.Description != "" {
			fmt.Printf("    %s\n", f.Muted(c.Description))
		}
		if len(c.SubPipelines) > 0 {
			fmt.Printf("    %s %s\n", f.Muted("sub-pipelines:"), strings.Join(c.SubPipelines, ", "))
		}
		if len(c.StepTypes) > 0 {
			fmt.Printf("    %s %s\n", f.Muted("step types:"), strings.Join(c.StepTypes, ", "))
		}
	}

	fmt.Println()
}
