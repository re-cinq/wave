package meta

import (
	"fmt"
	"sort"
)

// ProposalType distinguishes between single pipelines, parallel sets, and sequences.
type ProposalType string

const (
	ProposalSingle   ProposalType = "single"
	ProposalParallel ProposalType = "parallel"
	ProposalSequence ProposalType = "sequence"
)

// PipelineProposal represents a recommended pipeline run or sequence.
type PipelineProposal struct {
	ID             string       `json:"id"`
	Type           ProposalType `json:"type"`
	Pipelines      []string     `json:"pipelines"`
	Rationale      string       `json:"rationale"`
	PrefilledInput string       `json:"prefilled_input"`
	Priority       int          `json:"priority"`
	DepsReady      bool         `json:"deps_ready"`
	MissingDeps    []string     `json:"missing_deps,omitempty"`
}

// ProposalSelection represents the user's choice from the interactive menu.
type ProposalSelection struct {
	Proposals      []PipelineProposal `json:"proposals"`
	ModifiedInputs map[string]string  `json:"modified_inputs"`
	ExecutionMode  ProposalType       `json:"execution_mode"`
}

// ProposalEngine generates pipeline proposals from health check results.
type ProposalEngine struct {
	report    *HealthReport
	pipelines []string // available pipeline names
}

// NewProposalEngine creates a new proposal engine.
func NewProposalEngine(report *HealthReport, pipelines []string) *ProposalEngine {
	return &ProposalEngine{
		report:    report,
		pipelines: pipelines,
	}
}

// GenerateProposals analyzes the health report and generates ranked proposals.
func (e *ProposalEngine) GenerateProposals() []PipelineProposal {
	var proposals []PipelineProposal
	counter := 0

	family := e.report.Platform.PipelineFamily

	// Rule 1: Open issues > 0 and <= 3 → propose {family}-implement
	if e.report.Codebase.OpenIssueCount > 0 && e.report.Codebase.OpenIssueCount <= 3 {
		name := family + "-implement"
		if e.pipelineExists(name) {
			counter++
			depsReady, missingDeps := e.checkDeps(name)
			proposals = append(proposals, PipelineProposal{
				ID:             fmt.Sprintf("p%d", counter),
				Type:           ProposalSingle,
				Pipelines:      []string{name},
				Rationale:      fmt.Sprintf("%d open issue(s) found — run implementation pipeline", e.report.Codebase.OpenIssueCount),
				PrefilledInput: fmt.Sprintf("Implement open issues (%d found)", e.report.Codebase.OpenIssueCount),
				Priority:       1,
				DepsReady:      depsReady,
				MissingDeps:    missingDeps,
			})
		}
	}

	// Rule 2: Open issues > 3 → propose {family}-implement-epic
	if e.report.Codebase.OpenIssueCount > 3 {
		name := family + "-implement-epic"
		if e.pipelineExists(name) {
			counter++
			depsReady, missingDeps := e.checkDeps(name)
			proposals = append(proposals, PipelineProposal{
				ID:             fmt.Sprintf("p%d", counter),
				Type:           ProposalSingle,
				Pipelines:      []string{name},
				Rationale:      fmt.Sprintf("%d open issues found — run epic implementation pipeline", e.report.Codebase.OpenIssueCount),
				PrefilledInput: fmt.Sprintf("Implement open issues (%d found)", e.report.Codebase.OpenIssueCount),
				Priority:       1,
				DepsReady:      depsReady,
				MissingDeps:    missingDeps,
			})
		}
	}

	// Rule 3: Open PRs > 0 → propose {family}-pr-review (only if pipeline exists)
	if e.report.Codebase.OpenPRCount > 0 {
		name := family + "-pr-review"
		if e.pipelineExists(name) {
			counter++
			depsReady, missingDeps := e.checkDeps(name)
			proposals = append(proposals, PipelineProposal{
				ID:             fmt.Sprintf("p%d", counter),
				Type:           ProposalSingle,
				Pipelines:      []string{name},
				Rationale:      fmt.Sprintf("%d open PR(s) awaiting review", e.report.Codebase.OpenPRCount),
				PrefilledInput: fmt.Sprintf("Review open pull requests (%d found)", e.report.Codebase.OpenPRCount),
				Priority:       2,
				DepsReady:      depsReady,
				MissingDeps:    missingDeps,
			})
		}
	}

	// Rule 4: Low recent commits (< 5 in last 30 days) → propose wave-evolve
	if e.report.Codebase.RecentCommits < 5 {
		name := "wave-evolve"
		if e.pipelineExists(name) {
			counter++
			depsReady, missingDeps := e.checkDeps(name)
			proposals = append(proposals, PipelineProposal{
				ID:             fmt.Sprintf("p%d", counter),
				Type:           ProposalSingle,
				Pipelines:      []string{name},
				Rationale:      fmt.Sprintf("Low recent activity (%d commits in last 30 days) — evolve the project", e.report.Codebase.RecentCommits),
				PrefilledInput: "Evolve project based on current state",
				Priority:       3,
				DepsReady:      depsReady,
				MissingDeps:    missingDeps,
			})
		}
	}

	// Rule 5: Open issues > 0 AND both research and implement pipelines exist → propose sequence
	if e.report.Codebase.OpenIssueCount > 0 {
		researchName := family + "-research"
		implementName := family + "-implement"
		if e.pipelineExists(researchName) && e.pipelineExists(implementName) {
			counter++
			depsReadyResearch, missingResearch := e.checkDeps(researchName)
			depsReadyImplement, missingImplement := e.checkDeps(implementName)
			depsReady := depsReadyResearch && depsReadyImplement
			missingDeps := mergeMissingDeps(missingResearch, missingImplement)
			proposals = append(proposals, PipelineProposal{
				ID:             fmt.Sprintf("p%d", counter),
				Type:           ProposalSequence,
				Pipelines:      []string{researchName, implementName},
				Rationale:      fmt.Sprintf("Research then implement open issues (%d found)", e.report.Codebase.OpenIssueCount),
				PrefilledInput: fmt.Sprintf("Research and implement open issues (%d found)", e.report.Codebase.OpenIssueCount),
				Priority:       2,
				DepsReady:      depsReady,
				MissingDeps:    missingDeps,
			})
		}
	}

	// Rule 6: No actionable signals → propose wave-evolve as default
	if len(proposals) == 0 {
		name := "wave-evolve"
		if e.pipelineExists(name) {
			counter++
			depsReady, missingDeps := e.checkDeps(name)
			proposals = append(proposals, PipelineProposal{
				ID:             fmt.Sprintf("p%d", counter),
				Type:           ProposalSingle,
				Pipelines:      []string{name},
				Rationale:      "No actionable signals detected — evolve the project",
				PrefilledInput: "Evolve project based on current state",
				Priority:       4,
				DepsReady:      depsReady,
				MissingDeps:    missingDeps,
			})
		}
	}

	// Sort by priority (lower number = higher priority).
	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].Priority < proposals[j].Priority
	})

	return proposals
}

// pipelineExists checks if a pipeline name is in the available list.
func (e *ProposalEngine) pipelineExists(name string) bool {
	for _, p := range e.pipelines {
		if p == name {
			return true
		}
	}
	return false
}

// checkDeps checks whether all tool dependencies are available for a pipeline.
// It returns true if all tools are available, along with any missing tool names.
func (e *ProposalEngine) checkDeps(pipeline string) (bool, []string) {
	_ = pipeline // Pipeline-specific dep resolution is future work; check all tools for now.

	var missing []string
	for _, tool := range e.report.Dependencies.Tools {
		if !tool.Available {
			missing = append(missing, tool.Name)
		}
	}
	for _, skill := range e.report.Dependencies.Skills {
		if !skill.Available {
			missing = append(missing, skill.Name)
		}
	}

	return len(missing) == 0, missing
}

// mergeMissingDeps combines two missing dependency slices, removing duplicates.
func mergeMissingDeps(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	var merged []string
	for _, dep := range a {
		if _, ok := seen[dep]; !ok {
			seen[dep] = struct{}{}
			merged = append(merged, dep)
		}
	}
	for _, dep := range b {
		if _, ok := seen[dep]; !ok {
			seen[dep] = struct{}{}
			merged = append(merged, dep)
		}
	}
	return merged
}
