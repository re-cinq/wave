package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ComposeDetailModel is the right pane model for the compose mode artifact
// flow visualization. It displays per-boundary artifact compatibility between
// adjacent pipelines in a sequence.
type ComposeDetailModel struct {
	width      int
	height     int
	focused    bool
	viewport   viewport.Model
	validation CompatibilityResult
	focusedIdx int // Which boundary is focused (-1 for overview)
	parallel   bool
	stages     [][]int // Stage groupings for parallel rendering
}

// NewComposeDetailModel creates a new compose detail model.
func NewComposeDetailModel() ComposeDetailModel {
	return ComposeDetailModel{
		focusedIdx: -1,
		viewport:   viewport.New(0, 0),
	}
}

// Init returns nil.
func (m ComposeDetailModel) Init() tea.Cmd {
	return nil
}

// Update handles messages to update the compose detail model state.
func (m ComposeDetailModel) Update(msg tea.Msg) (ComposeDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ComposeSequenceChangedMsg:
		m.validation = msg.Validation
		m.parallel = msg.Parallel
		m.stages = msg.Stages
		content := renderArtifactFlow(m.validation, m.width)
		if m.parallel && len(m.stages) > 0 {
			content += "\n" + renderExecutionPlan(msg.Sequence, m.stages, m.width)
		}
		m.viewport.SetContent(content)
		m.viewport.GotoTop()
		return m, nil

	case tea.KeyMsg:
		if m.focused {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// View renders the compose detail pane.
func (m ComposeDetailModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	if len(m.validation.Flows) == 0 {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Add pipelines to see artifact flow")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	if m.focused {
		borderStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("2")).
			Width(m.width - 2).
			Height(m.height - 2)
		return borderStyle.Render(m.viewport.View())
	}

	return m.viewport.View()
}

// SetSize updates the model dimensions and re-renders content.
func (m *ComposeDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	// Re-render content at new width
	content := renderArtifactFlow(m.validation, w)
	if m.parallel && len(m.stages) > 0 {
		// We don't have the sequence here, so just re-render artifact flow.
		// Full plan rendering happens on ComposeSequenceChangedMsg.
	}
	m.viewport.SetContent(content)
}

// SetFocused updates the focused state.
func (m *ComposeDetailModel) SetFocused(focused bool) {
	m.focused = focused
}

// renderArtifactFlow renders the artifact flow visualization for the given
// compatibility result. It uses compact mode (text-only) for narrow terminals
// and full mode (box-drawing) for wider terminals.
func renderArtifactFlow(result CompatibilityResult, width int) string {
	if len(result.Flows) == 0 {
		return "Add pipelines to see artifact flow"
	}

	var sb strings.Builder

	if width < 120 {
		renderArtifactFlowCompact(&sb, result)
	} else {
		renderArtifactFlowFull(&sb, result)
	}

	// Status summary
	sb.WriteString("\n")
	renderStatusSummary(&sb, result)

	return sb.String()
}

// renderArtifactFlowCompact renders a text-only summary of artifact flows.
func renderArtifactFlowCompact(sb *strings.Builder, result CompatibilityResult) {
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	headerStyle := lipgloss.NewStyle().Bold(true)

	for i, flow := range result.Flows {
		if i > 0 {
			sb.WriteString("\n")
		}

		sb.WriteString(headerStyle.Render(fmt.Sprintf("%s → %s", flow.SourcePipeline, flow.TargetPipeline)))
		sb.WriteString("\n")

		for _, match := range flow.Matches {
			switch match.Status {
			case MatchCompatible:
				inputName := match.InputName
				if match.InputAs != "" {
					inputName = match.InputAs
				}
				sb.WriteString(fmt.Sprintf("  %s %s → %s (compatible)\n",
					greenStyle.Render("✓"),
					match.OutputName,
					inputName,
				))
			case MatchMissing:
				inputName := match.InputName
				if match.InputAs != "" {
					inputName = match.InputAs
				}
				if match.Optional {
					sb.WriteString(fmt.Sprintf("  %s %s (optional — no matching output)\n",
						yellowStyle.Render("⚠"),
						inputName,
					))
				} else {
					sb.WriteString(fmt.Sprintf("  %s %s (missing — no matching output)\n",
						redStyle.Render("✗"),
						inputName,
					))
				}
			case MatchUnmatched:
				sb.WriteString(fmt.Sprintf("  %s %s (not consumed)\n",
					dimStyle.Render("·"),
					match.OutputName,
				))
			}
		}
	}
}

// renderArtifactFlowFull renders a structured box-drawing visualization of
// artifact flows for wider terminals (>= 120 columns).
func renderArtifactFlowFull(sb *strings.Builder, result CompatibilityResult) {
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	// Collect all unique pipeline names in order for rendering boxes
	pipelineNames := make([]string, 0)
	seen := make(map[string]bool)
	for _, flow := range result.Flows {
		if !seen[flow.SourcePipeline] {
			pipelineNames = append(pipelineNames, flow.SourcePipeline)
			seen[flow.SourcePipeline] = true
		}
		if !seen[flow.TargetPipeline] {
			pipelineNames = append(pipelineNames, flow.TargetPipeline)
			seen[flow.TargetPipeline] = true
		}
	}

	// Build a map from source pipeline to its flow for quick lookup
	flowBySource := make(map[string]*ArtifactFlow)
	for i := range result.Flows {
		flowBySource[result.Flows[i].SourcePipeline] = &result.Flows[i]
	}

	for i, name := range pipelineNames {
		// Find outputs and inputs for this pipeline
		var outputs []string
		var inputs []string

		// Outputs: from the flow where this pipeline is the source
		if flow, ok := flowBySource[name]; ok {
			for _, out := range flow.Outputs {
				outputs = append(outputs, out.Name)
			}
		}

		// Inputs: from the flow where this pipeline is the target
		for _, flow := range result.Flows {
			if flow.TargetPipeline == name {
				for _, inp := range flow.Inputs {
					if inp.As != "" {
						inputs = append(inputs, inp.As)
					} else {
						inputs = append(inputs, inp.Artifact)
					}
				}
				break
			}
		}

		// Render pipeline box
		boxWidth := 40
		sb.WriteString("┌" + strings.Repeat("─", boxWidth) + "┐\n")
		sb.WriteString("│ " + titleStyle.Render(padRight(name, boxWidth-2)) + " │\n")

		if len(inputs) > 0 {
			sb.WriteString("│ " + labelStyle.Render(padRight("inputs: "+strings.Join(inputs, ", "), boxWidth-2)) + " │\n")
		}
		if len(outputs) > 0 {
			sb.WriteString("│ " + labelStyle.Render(padRight("outputs: "+strings.Join(outputs, ", "), boxWidth-2)) + " │\n")
		}

		sb.WriteString("└" + strings.Repeat("─", boxWidth) + "┘\n")

		// Render the flow matches between this pipeline and the next
		if flow, ok := flowBySource[name]; ok && i < len(pipelineNames)-1 {
			sb.WriteString("           │\n")

			for _, match := range flow.Matches {
				switch match.Status {
				case MatchCompatible:
					inputName := match.InputName
					if match.InputAs != "" {
						inputName = match.InputAs
					}
					sb.WriteString(fmt.Sprintf("  %s %s → %s\n",
						greenStyle.Render("✓"),
						match.OutputName,
						inputName,
					))
				case MatchMissing:
					inputName := match.InputName
					if match.InputAs != "" {
						inputName = match.InputAs
					}
					if match.Optional {
						sb.WriteString(fmt.Sprintf("  %s %s (optional — no matching output)\n",
							yellowStyle.Render("⚠"),
							inputName,
						))
					} else {
						sb.WriteString(fmt.Sprintf("  %s %s (missing — no matching output)\n",
							redStyle.Render("✗"),
							inputName,
						))
					}
				case MatchUnmatched:
					sb.WriteString(fmt.Sprintf("  %s %s (not consumed)\n",
						dimStyle.Render("·"),
						match.OutputName,
					))
				}
			}

			sb.WriteString("           │\n")
		}
	}
}

// renderStatusSummary appends a status summary line to the builder.
func renderStatusSummary(sb *strings.Builder, result CompatibilityResult) {
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	switch result.Status {
	case CompatibilityValid:
		sb.WriteString(greenStyle.Render("✓ All artifact flows compatible"))
		sb.WriteString("\n")
	case CompatibilityWarning:
		warnings := countDiagnosticsByType(result, "optional")
		sb.WriteString(yellowStyle.Render(fmt.Sprintf("⚠ %d optional input(s) unmatched", warnings)))
		sb.WriteString("\n")
	case CompatibilityError:
		errors := countDiagnosticsByType(result, "missing")
		warnings := countDiagnosticsByType(result, "optional")
		sb.WriteString(redStyle.Render(fmt.Sprintf("✗ %d error(s), %d warning(s) found", errors, warnings)))
		sb.WriteString("\n")
	}
}

// countDiagnosticsByType counts diagnostics containing the given keyword.
func countDiagnosticsByType(result CompatibilityResult, keyword string) int {
	count := 0
	for _, d := range result.Diagnostics {
		if strings.Contains(d, keyword) {
			count++
		}
	}
	return count
}

// renderExecutionPlan renders a DAG visualization showing parallel stage groups.
func renderExecutionPlan(seq Sequence, stages [][]int, width int) string {
	if len(stages) == 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Execution Plan"))
	sb.WriteString("\n\n")

	for i, stage := range stages {
		isParallel := len(stage) > 1
		mode := "sequential"
		if isParallel {
			mode = "parallel"
		}
		sb.WriteString(mutedStyle.Render(fmt.Sprintf("Stage %d (%s):", i+1, mode)))
		sb.WriteString("\n")

		for j, idx := range stage {
			name := ""
			if idx < seq.Len() {
				name = seq.Entries[idx].PipelineName
			}
			if isParallel {
				if j == 0 {
					sb.WriteString(fmt.Sprintf("┌─ %s\n", name))
				} else if j == len(stage)-1 {
					sb.WriteString(fmt.Sprintf("└─ %s\n", name))
				} else {
					sb.WriteString(fmt.Sprintf("├─ %s\n", name))
				}
			} else {
				sb.WriteString(fmt.Sprintf("   %s\n", name))
			}
		}

		// Connector between stages
		if i < len(stages)-1 {
			sb.WriteString("       │\n")
		}
	}

	return sb.String()
}

// padRight pads a string with spaces to the given width, truncating if necessary.
func padRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
