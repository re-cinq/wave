package tui

import (
	"errors"
	"fmt"
)

// ProposalDecision represents the user's decision on a proposed pipeline.
type ProposalDecision int

const (
	// Accept indicates the user accepted the proposal as-is.
	Accept ProposalDecision = iota
	// Modify indicates the user accepted with modifications.
	Modify
	// Skip indicates the user chose to skip this proposal.
	Skip
)

// String returns the human-readable name of the decision.
func (d ProposalDecision) String() string {
	switch d {
	case Accept:
		return "accept"
	case Modify:
		return "modify"
	case Skip:
		return "skip"
	default:
		return fmt.Sprintf("ProposalDecision(%d)", int(d))
	}
}

// Proposal represents a recommended pipeline for the user to review.
type Proposal struct {
	Pipeline      string
	Reason        string
	Dependencies  []string
	ParallelGroup string
	Input         string
	Priority      int
}

// ProposalResult holds the outcome of an interactive proposal review session.
type ProposalResult struct {
	Pipelines []AcceptedPipeline
	Aborted   bool
}

// AcceptedPipeline represents a pipeline the user accepted for execution.
type AcceptedPipeline struct {
	Pipeline string
	Input    string
	Flags    []string
}

// ValidateProposals checks that the proposals slice is well-formed.
// It returns an error if the slice is empty, any proposal has an empty
// Pipeline name, or circular dependencies exist among proposals.
func ValidateProposals(proposals []Proposal) error {
	if len(proposals) == 0 {
		return errors.New("proposals must not be empty")
	}

	names := make(map[string]struct{}, len(proposals))
	for _, p := range proposals {
		if p.Pipeline == "" {
			return errors.New("proposal has empty pipeline name")
		}
		names[p.Pipeline] = struct{}{}
	}

	// Build adjacency list from dependencies.
	deps := make(map[string][]string, len(proposals))
	for _, p := range proposals {
		for _, d := range p.Dependencies {
			if _, ok := names[d]; ok {
				deps[p.Pipeline] = append(deps[p.Pipeline], d)
			}
		}
	}

	// DFS-based cycle detection.
	const (
		white = 0 // unvisited
		gray  = 1 // in current path
		black = 2 // fully explored
	)

	color := make(map[string]int, len(proposals))
	for name := range names {
		color[name] = white
	}

	var visit func(string) error
	visit = func(node string) error {
		color[node] = gray
		for _, dep := range deps[node] {
			switch color[dep] {
			case gray:
				return fmt.Errorf("circular dependency detected: %s -> %s", node, dep)
			case white:
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		color[node] = black
		return nil
	}

	for name := range names {
		if color[name] == white {
			if err := visit(name); err != nil {
				return err
			}
		}
	}

	return nil
}
