package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/recinq/wave/internal/meta"
)

// Proposal selector styles — matches theme.go and health_report.go palette.
var (
	psTypeStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))   // cyan for type tags
	psRationaleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244")) // gray for rationale
	psDepsWarnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))   // yellow for missing deps
)

// RunProposalSelector presents proposals to the user and returns their selection.
// For interactive terminals, it uses huh.NewSelect for single selection.
// Returns nil selection and nil error if no proposals are available.
func RunProposalSelector(proposals []meta.PipelineProposal) (*meta.ProposalSelection, error) {
	if len(proposals) == 0 {
		fmt.Println(psRationaleStyle.Render("No runnable pipelines available"))
		return nil, nil
	}

	fmt.Println(WaveLogo())

	// Build select options from proposals.
	options := buildProposalOptions(proposals)

	var selectedID string
	selectField := huh.NewSelect[string]().
		Title("Select pipeline").
		Options(options...).
		Height(min(len(proposals)+2, 10)).
		Value(&selectedID)

	selectForm := huh.NewForm(
		huh.NewGroup(selectField),
	).WithTheme(WaveTheme())

	if err := selectForm.Run(); err != nil {
		return nil, err
	}

	// Find the selected proposal.
	var selected *meta.PipelineProposal
	for i := range proposals {
		if proposals[i].ID == selectedID {
			selected = &proposals[i]
			break
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("selected proposal %q not found", selectedID)
	}

	// Show input field pre-filled with the proposal's prefilled input.
	modifiedInput := selected.PrefilledInput
	inputField := huh.NewInput().
		Title("Input").
		Value(&modifiedInput).
		Placeholder("Describe what to do...")

	inputForm := huh.NewForm(
		huh.NewGroup(inputField),
	).WithTheme(WaveTheme())

	if err := inputForm.Run(); err != nil {
		return nil, err
	}

	return buildProposalSelection(*selected, modifiedInput), nil
}

// buildProposalOptions creates huh select options from pipeline proposals.
func buildProposalOptions(proposals []meta.PipelineProposal) []huh.Option[string] {
	options := make([]huh.Option[string], len(proposals))
	for i, p := range proposals {
		options[i] = huh.NewOption(formatProposalOption(p), p.ID)
	}
	return options
}

// formatProposalOption formats a single proposal for display in the selector.
// Format: "pipeline-name          [type] rationale text"
// For sequences: "pipeline-a → pipeline-b  [sequence] rationale text"
// For parallel:  "pipeline-a | pipeline-b  [parallel] rationale text"
func formatProposalOption(p meta.PipelineProposal) string {
	name := formatPipelineNames(p.Pipelines, p.Type)
	typeTag := psTypeStyle.Render(fmt.Sprintf("[%s]", p.Type))
	rationale := psRationaleStyle.Render(p.Rationale)

	label := fmt.Sprintf("%-24s %s %s", name, typeTag, rationale)

	if !p.DepsReady && len(p.MissingDeps) > 0 {
		warn := psDepsWarnStyle.Render(fmt.Sprintf("(missing: %s)", strings.Join(p.MissingDeps, ", ")))
		label += " " + warn
	}

	return label
}

// formatPipelineNames joins pipeline names with the appropriate separator for the proposal type.
func formatPipelineNames(pipelines []string, proposalType meta.ProposalType) string {
	switch proposalType {
	case meta.ProposalSequence:
		return strings.Join(pipelines, " → ")
	case meta.ProposalParallel:
		return strings.Join(pipelines, " | ")
	default:
		return strings.Join(pipelines, ", ")
	}
}

// buildProposalSelection creates a ProposalSelection from a selected proposal and modified input.
func buildProposalSelection(selected meta.PipelineProposal, modifiedInput string) *meta.ProposalSelection {
	inputs := make(map[string]string, len(selected.Pipelines))
	for _, pipeline := range selected.Pipelines {
		inputs[pipeline] = modifiedInput
	}

	return &meta.ProposalSelection{
		Proposals:      []meta.PipelineProposal{selected},
		ModifiedInputs: inputs,
		ExecutionMode:  selected.Type,
	}
}
