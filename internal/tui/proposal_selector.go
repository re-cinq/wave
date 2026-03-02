package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// proposalDecisionResult holds the user's decision and any modifications for a
// single proposal.
type proposalDecisionResult struct {
	Decision ProposalDecision
	Input    string
	Flags    []string
}

// proposalGroup holds a set of proposals that share a ParallelGroup. Proposals
// with an empty ParallelGroup are each placed into their own singleton group.
type proposalGroup struct {
	GroupName string
	Proposals []Proposal
}

// RunProposalSelector presents an interactive TUI flow for reviewing pipeline
// proposals. It validates the proposals, walks through each proposal (or
// parallel group) with accept/modify/skip, previews the DAG, and asks for
// final confirmation.
func RunProposalSelector(proposals []Proposal) (*ProposalResult, error) {
	if err := ValidateProposals(proposals); err != nil {
		return nil, fmt.Errorf("invalid proposals: %w", err)
	}

	fmt.Println(WaveLogo())

	groups := groupProposalsByParallel(proposals)
	decisions := make(map[string]proposalDecisionResult, len(proposals))

	for _, g := range groups {
		if len(g.Proposals) > 1 {
			if err := runParallelGroupReview(g, decisions); err != nil {
				return nil, err
			}
		} else {
			if err := runSingleProposalReview(g.Proposals[0], decisions); err != nil {
				return nil, err
			}
		}
	}

	// Check if all proposals were skipped.
	allSkipped := true
	for _, d := range decisions {
		if d.Decision != Skip {
			allSkipped = false
			break
		}
	}
	if allSkipped {
		return &ProposalResult{Aborted: true}, nil
	}

	// Build the accepted list in dependency order.
	accepted := buildAcceptedPipelines(proposals, decisions)

	// Build the filtered proposal list for the DAG preview.
	var acceptedProposals []Proposal
	acceptedSet := make(map[string]struct{}, len(accepted))
	for _, a := range accepted {
		acceptedSet[a.Pipeline] = struct{}{}
	}
	for _, p := range proposals {
		if _, ok := acceptedSet[p.Pipeline]; ok {
			acceptedProposals = append(acceptedProposals, p)
		}
	}

	// Show DAG preview.
	dagPreview := RenderDAGPreview(acceptedProposals, getTermWidth())
	if dagPreview != "" {
		fmt.Println()
		fmt.Println(dagPreview)
		fmt.Println()
	}

	// Confirm execution.
	var confirmed bool
	confirm := huh.NewConfirm().
		Title("Run these pipelines?").
		Description(formatAcceptedSummary(accepted)).
		Affirmative("Run").
		Negative("Cancel").
		Value(&confirmed)

	confirmForm := huh.NewForm(huh.NewGroup(confirm)).
		WithTheme(WaveTheme())

	if err := confirmForm.Run(); err != nil {
		return nil, err
	}

	if !confirmed {
		return &ProposalResult{Aborted: true}, nil
	}

	return &ProposalResult{
		Pipelines: accepted,
		Aborted:   false,
	}, nil
}

// runSingleProposalReview shows an accept/modify/skip selector for a single
// proposal and records the decision.
func runSingleProposalReview(p Proposal, decisions map[string]proposalDecisionResult) error {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	title := fmt.Sprintf("%s  %s", p.Pipeline, dimStyle.Render(p.Reason))

	var decision string
	selectField := huh.NewSelect[string]().
		Title(title).
		Options(
			huh.NewOption("Accept", "accept"),
			huh.NewOption("Modify", "modify"),
			huh.NewOption("Skip", "skip"),
		).
		Value(&decision)

	form := huh.NewForm(huh.NewGroup(selectField)).
		WithTheme(WaveTheme())

	if err := form.Run(); err != nil {
		return err
	}

	switch decision {
	case "accept":
		decisions[p.Pipeline] = proposalDecisionResult{
			Decision: Accept,
			Input:    p.Input,
		}
	case "modify":
		input, flags, err := runModifyForm(p)
		if err != nil {
			return err
		}
		decisions[p.Pipeline] = proposalDecisionResult{
			Decision: Modify,
			Input:    input,
			Flags:    flags,
		}
	case "skip":
		decisions[p.Pipeline] = proposalDecisionResult{
			Decision: Skip,
		}
	}

	return nil
}

// runParallelGroupReview shows a multi-select for a parallel group of
// proposals, allowing the user to batch-select which pipelines to accept.
// Unselected pipelines within the group are marked as skipped.
func runParallelGroupReview(g proposalGroup, decisions map[string]proposalDecisionResult) error {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	title := fmt.Sprintf("Parallel group: %s", g.GroupName)

	// Build options for each proposal in the group.
	options := make([]huh.Option[string], len(g.Proposals))
	for i, p := range g.Proposals {
		label := p.Pipeline
		if p.Reason != "" {
			label = fmt.Sprintf("%-20s %s", p.Pipeline, dimStyle.Render(p.Reason))
		}
		options[i] = huh.NewOption(label, p.Pipeline)
	}

	// Pre-select all proposals (accept-all default).
	preSelected := make([]string, len(g.Proposals))
	for i, p := range g.Proposals {
		preSelected[i] = p.Pipeline
	}

	var selected []string
	multiSelect := huh.NewMultiSelect[string]().
		Title(title).
		Options(options...).
		Value(&selected)

	// Set the initial selected values to all proposals.
	selected = preSelected

	form := huh.NewForm(huh.NewGroup(multiSelect)).
		WithTheme(WaveTheme())

	if err := form.Run(); err != nil {
		return err
	}

	selectedSet := make(map[string]struct{}, len(selected))
	for _, s := range selected {
		selectedSet[s] = struct{}{}
	}

	for _, p := range g.Proposals {
		if _, ok := selectedSet[p.Pipeline]; ok {
			decisions[p.Pipeline] = proposalDecisionResult{
				Decision: Accept,
				Input:    p.Input,
			}
		} else {
			decisions[p.Pipeline] = proposalDecisionResult{
				Decision: Skip,
			}
		}
	}

	return nil
}

// runModifyForm shows a sub-form for modifying a proposal's input and flags.
func runModifyForm(p Proposal) (string, []string, error) {
	var input string
	var selectedFlags []string

	input = p.Input

	flags := DefaultFlags()
	flagOptions := buildFlagOptions(flags)

	inputField := huh.NewInput().
		Title("Input").
		Placeholder(p.Pipeline).
		Value(&input)

	multiSelect := huh.NewMultiSelect[string]().
		Title("Options").
		Options(flagOptions...).
		Value(&selectedFlags)

	form := huh.NewForm(huh.NewGroup(inputField, multiSelect)).
		WithTheme(WaveTheme())

	if err := form.Run(); err != nil {
		return "", nil, err
	}

	return input, selectedFlags, nil
}

// sortProposalsByPriority returns a copy of proposals sorted by Priority
// (ascending).
func sortProposalsByPriority(proposals []Proposal) []Proposal {
	sorted := make([]Proposal, len(proposals))
	copy(sorted, proposals)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})
	return sorted
}

// groupProposalsByParallel groups proposals by ParallelGroup, preserving order.
// Ungrouped proposals (empty ParallelGroup) are each treated as their own
// group. Within each group, proposals are sorted by Priority.
func groupProposalsByParallel(proposals []Proposal) []proposalGroup {
	sorted := sortProposalsByPriority(proposals)

	var order []string
	groups := make(map[string][]Proposal)

	for _, p := range sorted {
		key := p.ParallelGroup
		if key == "" {
			// Each ungrouped proposal forms its own group with a unique key.
			uniqueKey := "\x00" + p.Pipeline
			order = append(order, uniqueKey)
			groups[uniqueKey] = []Proposal{p}
			continue
		}
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], p)
	}

	result := make([]proposalGroup, 0, len(order))
	for _, key := range order {
		groupName := ""
		if key[0] != '\x00' {
			groupName = key
		}
		result = append(result, proposalGroup{
			GroupName: groupName,
			Proposals: groups[key],
		})
	}
	return result
}

// buildAcceptedPipelines collects accepted proposals into the result,
// respecting dependency ordering via topological sort.
func buildAcceptedPipelines(proposals []Proposal, decisions map[string]proposalDecisionResult) []AcceptedPipeline {
	// Build accepted set.
	acceptedSet := make(map[string]struct{})
	for _, p := range proposals {
		d, ok := decisions[p.Pipeline]
		if ok && d.Decision != Skip {
			acceptedSet[p.Pipeline] = struct{}{}
		}
	}

	// Build adjacency for topological sort (only among accepted proposals).
	proposalMap := make(map[string]Proposal, len(proposals))
	for _, p := range proposals {
		proposalMap[p.Pipeline] = p
	}

	adj := make(map[string][]string)
	inDeg := make(map[string]int)
	for name := range acceptedSet {
		inDeg[name] = 0
	}
	for name := range acceptedSet {
		p := proposalMap[name]
		for _, dep := range p.Dependencies {
			if _, ok := acceptedSet[dep]; ok {
				adj[dep] = append(adj[dep], name)
				inDeg[name]++
			}
		}
	}

	// Kahn's algorithm to produce topological order.
	// Collect accepted proposals in insertion order for deterministic seeding.
	ordered := make([]string, 0, len(acceptedSet))
	for _, p := range proposals {
		if _, ok := acceptedSet[p.Pipeline]; ok {
			ordered = append(ordered, p.Pipeline)
		}
	}

	var queue []string
	for _, name := range ordered {
		if inDeg[name] == 0 {
			queue = append(queue, name)
		}
	}

	var topoOrder []string
	for len(queue) > 0 {
		sort.Strings(queue)
		var nextQueue []string
		for _, name := range queue {
			topoOrder = append(topoOrder, name)
			for _, child := range adj[name] {
				inDeg[child]--
				if inDeg[child] == 0 {
					nextQueue = append(nextQueue, child)
				}
			}
		}
		queue = nextQueue
	}

	// Build the result.
	result := make([]AcceptedPipeline, 0, len(topoOrder))
	for _, name := range topoOrder {
		d := decisions[name]
		result = append(result, AcceptedPipeline{
			Pipeline: name,
			Input:    d.Input,
			Flags:    d.Flags,
		})
	}
	return result
}

// formatAcceptedSummary builds a compact summary of accepted pipelines for the
// confirmation dialog description.
func formatAcceptedSummary(accepted []AcceptedPipeline) string {
	if len(accepted) == 0 {
		return "No pipelines selected"
	}
	names := make([]string, len(accepted))
	for i, a := range accepted {
		names[i] = a.Pipeline
	}
	return strings.Join(names, " -> ")
}

// getTermWidth returns the current terminal width, defaulting to 80 if it
// cannot be determined.
func getTermWidth() int {
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
		return width
	}
	return 80
}
