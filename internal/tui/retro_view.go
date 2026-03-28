package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// RetroViewModel holds the data needed to render a pipeline retrospective.
type RetroViewModel struct {
	RunID           string
	Pipeline        string
	Duration        time.Duration
	TotalSteps      int
	SuccessCount    int
	FailureCount    int
	TotalRetries    int
	TotalTokens     int
	Smoothness      string // "effortless", "smooth", "bumpy", "struggled", "failed"
	Intent          string
	Outcome         string
	FrictionPoints  []RetroFrictionPoint
	Learnings       []RetroLearning
	Recommendations []string
}

// RetroFrictionPoint describes a point of friction during pipeline execution.
type RetroFrictionPoint struct {
	Type   string // e.g. "retry", "contract_failure", "timeout"
	Step   string
	Detail string
}

// RetroLearning captures a lesson learned from the pipeline run.
type RetroLearning struct {
	Category string // e.g. "performance", "reliability", "quality"
	Detail   string
}

// RenderRetro renders a pipeline retrospective using lipgloss styles consistent
// with the existing TUI design language.
func RenderRetro(retro *RetroViewModel) string {
	if retro == nil {
		return ""
	}

	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var sb strings.Builder

	sb.WriteString(sectionStyle.Render("Retrospective:"))
	sb.WriteString("\n")

	// Smoothness with color coding
	if retro.Smoothness != "" {
		smoothnessDisplay := smoothnessStyled(retro.Smoothness)
		sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Smoothness:"), smoothnessDisplay))
	}

	// Quantitative metrics grid
	sb.WriteString(fmt.Sprintf("  %s %d/%d succeeded",
		labelStyle.Render("Steps:"),
		retro.SuccessCount,
		retro.TotalSteps,
	))
	if retro.FailureCount > 0 {
		sb.WriteString(fmt.Sprintf(", %d failed", retro.FailureCount))
	}
	sb.WriteString("\n")

	if retro.TotalRetries > 0 {
		sb.WriteString(fmt.Sprintf("  %s %d\n", labelStyle.Render("Retries:"), retro.TotalRetries))
	}
	if retro.TotalTokens > 0 {
		sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Tokens:"), formatTokenCount(retro.TotalTokens)))
	}

	// Intent / Outcome narrative
	if retro.Intent != "" {
		sb.WriteString(fmt.Sprintf("\n  %s %s\n", labelStyle.Render("Intent:"), retro.Intent))
	}
	if retro.Outcome != "" {
		sb.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Outcome:"), retro.Outcome))
	}

	// Friction points
	if len(retro.FrictionPoints) > 0 {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s\n", sectionStyle.Render("Friction:")))
		for _, fp := range retro.FrictionPoints {
			prefix := fp.Type
			if fp.Step != "" {
				prefix = fp.Step + " (" + fp.Type + ")"
			}
			sb.WriteString(fmt.Sprintf("    - %s: %s\n", prefix, fp.Detail))
		}
	}

	// Learnings
	if len(retro.Learnings) > 0 {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s\n", sectionStyle.Render("Learnings:")))
		for _, l := range retro.Learnings {
			sb.WriteString(fmt.Sprintf("    - [%s] %s\n", l.Category, l.Detail))
		}
	}

	// Recommendations
	if len(retro.Recommendations) > 0 {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s\n", sectionStyle.Render("Recommendations:")))
		for _, r := range retro.Recommendations {
			sb.WriteString(fmt.Sprintf("    - %s\n", r))
		}
	}

	return sb.String()
}

// smoothnessStyled returns the smoothness label with appropriate color coding.
func smoothnessStyled(smoothness string) string {
	switch smoothness {
	case "effortless", "smooth":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(smoothness)
	case "bumpy":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(smoothness)
	case "struggled", "failed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(smoothness)
	default:
		return smoothness
	}
}

// NOTE: formatTokenCount is declared in live_output.go and reused here via package scope.
