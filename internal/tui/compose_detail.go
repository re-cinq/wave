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
	sequence   Sequence // Stored for re-rendering on resize
	focusedIdx int      // Which boundary is focused (-1 for overview)
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
		m.sequence = msg.Sequence
		m.parallel = msg.Parallel
		m.stages = msg.Stages
		if m.parallel && len(m.stages) > 0 {
			m.validation = ValidateSequenceWithStages(msg.Sequence, msg.Stages)
		} else {
			m.validation = msg.Validation
		}
		content := renderArtifactFlow(m.validation, m.width, m.parallel, m.sequence, m.stages)
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

	hasContent := len(m.validation.Flows) > 0 || (m.parallel && len(m.stages) > 0)
	if !hasContent {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("Add pipelines to see artifact flow")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	if m.focused {
		borderStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("6")).
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
	content := renderArtifactFlow(m.validation, w, m.parallel, m.sequence, m.stages)
	m.viewport.SetContent(content)
}

// SetFocused updates the focused state.
func (m *ComposeDetailModel) SetFocused(focused bool) {
	m.focused = focused
}

// renderArtifactFlow renders the artifact flow visualization for the given
// compatibility result. It uses compact mode (text-only) for narrow terminals
// and full mode (box-drawing) for wider terminals. When parallel mode is active
// with stages, it dispatches to stage-aware rendering that integrates the
// execution plan structure into the artifact flow view.
func renderArtifactFlow(result CompatibilityResult, width int, parallel bool, seq Sequence, stages [][]int) string {
	if parallel && len(stages) > 0 {
		return renderArtifactFlowStageAware(result, width, seq, stages)
	}

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

// renderArtifactFlowStageAware renders the integrated stage-aware artifact flow
// visualization for parallel mode. It combines stage groupings with cross-stage
// artifact flows into a single unified view.
func renderArtifactFlowStageAware(result CompatibilityResult, width int, seq Sequence, stages [][]int) string {
	var sb strings.Builder

	if width < 120 {
		renderStageFlowCompact(&sb, result, seq, stages)
	} else {
		renderStageFlowFull(&sb, result, seq, stages)
	}

	if len(result.Flows) > 0 {
		sb.WriteString("\n")
		renderStatusSummary(&sb, result)
	}

	return sb.String()
}

// renderStageFlowCompact renders a compact text-only stage-aware artifact flow.
// Each stage is shown as a group with its pipelines listed, and cross-stage
// flows are shown between stage blocks.
func renderStageFlowCompact(sb *strings.Builder, result CompatibilityResult, seq Sequence, stages [][]int) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	// Build map from flow source label to flow
	flowBySource := make(map[string]*ArtifactFlow)
	for i := range result.Flows {
		flowBySource[result.Flows[i].SourcePipeline] = &result.Flows[i]
	}

	for i, stage := range stages {
		isParallel := len(stage) > 1
		mode := "sequential"
		if isParallel {
			mode = "parallel"
		}
		sb.WriteString(titleStyle.Render(fmt.Sprintf("Stage %d", i+1)))
		sb.WriteString(mutedStyle.Render(fmt.Sprintf(" (%s)", mode)))
		sb.WriteString("\n")

		for j, idx := range stage {
			name := ""
			if idx < seq.Len() {
				name = seq.Entries[idx].PipelineName
			}
			switch {
			case !isParallel:
				sb.WriteString(fmt.Sprintf("  %s\n", name))
			case j == 0:
				sb.WriteString(fmt.Sprintf("  ┌─ %s\n", name))
			case j == len(stage)-1:
				sb.WriteString(fmt.Sprintf("  └─ %s\n", name))
			default:
				sb.WriteString(fmt.Sprintf("  ├─ %s\n", name))
			}
		}

		// Show cross-stage flows between this stage and next
		stageLabel := fmt.Sprintf("Stage %d", i+1)
		if flow, ok := flowBySource[stageLabel]; ok {
			sb.WriteString("\n")
			renderFlowMatches(sb, flow.Matches, greenStyle, yellowStyle, redStyle, dimStyle)
			sb.WriteString("\n")
		} else if i < len(stages)-1 {
			sb.WriteString("\n")
		}
	}
}

// renderStageFlowFull renders a box-drawing stage-aware artifact flow for wider
// terminals. Stage blocks are rendered as grouped boxes with cross-stage flows
// between them.
func renderStageFlowFull(sb *strings.Builder, result CompatibilityResult, seq Sequence, stages [][]int) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	// Build map from flow source label to flow
	flowBySource := make(map[string]*ArtifactFlow)
	for i := range result.Flows {
		flowBySource[result.Flows[i].SourcePipeline] = &result.Flows[i]
	}

	boxWidth := 40

	for i, stage := range stages {
		isParallel := len(stage) > 1
		mode := "sequential"
		if isParallel {
			mode = "parallel"
		}

		stageLabel := fmt.Sprintf("Stage %d (%s)", i+1, mode)
		sb.WriteString("┌" + strings.Repeat("─", boxWidth) + "┐\n")
		sb.WriteString("│ " + titleStyle.Render(padRight(stageLabel, boxWidth-2)) + " │\n")

		// List pipelines in this stage
		for _, idx := range stage {
			name := ""
			if idx < seq.Len() {
				name = seq.Entries[idx].PipelineName
			}
			if isParallel {
				sb.WriteString("│ " + mutedStyle.Render(padRight("  ║ "+name, boxWidth-2)) + " │\n")
			} else {
				sb.WriteString("│ " + mutedStyle.Render(padRight("  "+name, boxWidth-2)) + " │\n")
			}
		}

		// Show outputs if this stage is a source for a cross-stage flow
		stageLabelKey := fmt.Sprintf("Stage %d", i+1)
		if flow, ok := flowBySource[stageLabelKey]; ok && len(flow.Outputs) > 0 {
			outNames := make([]string, len(flow.Outputs))
			for j, out := range flow.Outputs {
				outNames[j] = out.Name
			}
			sb.WriteString("│ " + labelStyle.Render(padRight("outputs: "+strings.Join(outNames, ", "), boxWidth-2)) + " │\n")
		}

		// Show inputs if this stage is a target for a cross-stage flow
		for _, flow := range result.Flows {
			if flow.TargetPipeline == stageLabelKey && len(flow.Inputs) > 0 {
				inNames := make([]string, len(flow.Inputs))
				for j, inp := range flow.Inputs {
					if inp.As != "" {
						inNames[j] = inp.As
					} else {
						inNames[j] = inp.Artifact
					}
				}
				sb.WriteString("│ " + labelStyle.Render(padRight("inputs: "+strings.Join(inNames, ", "), boxWidth-2)) + " │\n")
				break
			}
		}

		sb.WriteString("└" + strings.Repeat("─", boxWidth) + "┘\n")

		// Render cross-stage flow matches
		if flow, ok := flowBySource[stageLabelKey]; ok {
			sb.WriteString("           │\n")
			renderFlowMatches(sb, flow.Matches, greenStyle, yellowStyle, redStyle, dimStyle)
			sb.WriteString("           │\n")
		}
	}
}

// renderFlowMatches renders individual flow match entries with appropriate styling.
func renderFlowMatches(sb *strings.Builder, matches []FlowMatch, greenStyle, yellowStyle, redStyle, dimStyle lipgloss.Style) {
	for _, match := range matches {
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
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
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
func renderExecutionPlan(seq Sequence, stages [][]int, _ int) string {
	if len(stages) == 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
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
				switch {
				case j == 0:
					sb.WriteString(fmt.Sprintf("┌─ %s\n", name))
				case j == len(stage)-1:
					sb.WriteString(fmt.Sprintf("└─ %s\n", name))
				default:
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
