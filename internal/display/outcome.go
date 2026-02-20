package display

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/recinq/wave/internal/deliverable"
	"github.com/recinq/wave/internal/event"
)

// PipelineOutcome is a read-only summary struct constructed after pipeline
// execution completes. It aggregates key results from the deliverable tracker
// into a structured format suitable for rendering.
type PipelineOutcome struct {
	// Identity
	PipelineName string
	RunID        string

	// Status
	Success  bool
	Duration time.Duration
	Tokens   int

	// Key outcomes (outcome-worthy deliverables)
	Branch       string       // Branch name (empty if no branch created)
	Pushed       bool         // Whether branch was pushed
	RemoteRef    string       // Remote reference (e.g., "origin/branch-name")
	PushError    string       // Push error message (empty if no error)
	PullRequests []OutcomeLink // PR URLs with labels
	Issues       []OutcomeLink // Issue URLs with labels
	Deployments  []OutcomeLink // Deployment URLs with labels

	// Reports and key files (top N outcome-worthy files)
	Reports []OutcomeFile

	// Artifact/contract summary (counts only in default mode)
	ArtifactCount   int
	ContractsPassed int
	ContractsFailed int
	ContractsTotal  int
	FailedContracts []ContractFailure // Always shown, even in default mode

	// Next steps
	NextSteps []NextStep

	// Workspace info
	WorkspacePath string

	// Verbose data (full lists, only rendered in verbose mode)
	AllDeliverables []*deliverable.Deliverable

	// Failed step tracking
	FailedStepIDs []string
}

// OutcomeLink represents a URL outcome (PR, issue, deployment).
type OutcomeLink struct {
	Label string // e.g., "Pull Request", "Issue #42"
	URL   string
}

// OutcomeFile represents a file outcome (report, key deliverable).
type OutcomeFile struct {
	Label string // e.g., "Spec Output", "Test Results"
	Path  string // Absolute or workspace-relative path
}

// ContractFailure captures a failed contract for prominent display.
type ContractFailure struct {
	StepID  string
	Type    string // Contract type (json_schema, test_suite, etc.)
	Message string // Failure reason
}

// NextStep represents a suggested follow-up action.
type NextStep struct {
	Label   string // e.g., "Review the pull request"
	Command string // Optional command (e.g., "gh pr view <url>")
	URL     string // Optional URL to open
}

// deliverableTypePriority returns a sort priority for deliverable types.
// Lower values = higher priority (shown first).
func deliverableTypePriority(t deliverable.DeliverableType) int {
	switch t {
	case deliverable.TypePR:
		return 0
	case deliverable.TypeIssue:
		return 1
	case deliverable.TypeBranch:
		return 2
	case deliverable.TypeDeployment:
		return 3
	case deliverable.TypeFile:
		return 4
	case deliverable.TypeURL:
		return 5
	case deliverable.TypeArtifact:
		return 6
	case deliverable.TypeContract:
		return 7
	case deliverable.TypeLog:
		return 8
	default:
		return 9
	}
}

// isOutcomeWorthy returns true if the deliverable type should appear in the
// outcome summary (as opposed to detail-level deliverables).
func isOutcomeWorthy(t deliverable.DeliverableType) bool {
	switch t {
	case deliverable.TypePR, deliverable.TypeIssue, deliverable.TypeBranch, deliverable.TypeDeployment:
		return true
	default:
		return false
	}
}

// BuildOutcome constructs a PipelineOutcome from tracker data.
func BuildOutcome(tracker *deliverable.Tracker, pipelineName, runID string, success bool, duration time.Duration, tokens int, workspacePath string, failedStepIDs []string) *PipelineOutcome {
	outcome := &PipelineOutcome{
		PipelineName:  pipelineName,
		RunID:         runID,
		Success:       success,
		Duration:      duration,
		Tokens:        tokens,
		WorkspacePath: workspacePath,
		FailedStepIDs: failedStepIDs,
	}

	if tracker == nil {
		return outcome
	}

	all := tracker.GetAll()
	outcome.AllDeliverables = all

	// Build a set of failed step IDs for quick lookup
	failedSet := make(map[string]bool, len(failedStepIDs))
	for _, id := range failedStepIDs {
		failedSet[id] = true
	}

	// Extract branch info (first branch deliverable wins)
	for _, d := range tracker.GetByType(deliverable.TypeBranch) {
		if outcome.Branch == "" {
			outcome.Branch = d.Name
			if d.Metadata != nil {
				if pushed, ok := d.Metadata["pushed"].(bool); ok {
					outcome.Pushed = pushed
				}
				if ref, ok := d.Metadata["remote_ref"].(string); ok {
					outcome.RemoteRef = ref
				}
				if pushErr, ok := d.Metadata["push_error"].(string); ok {
					outcome.PushError = pushErr
				}
			}
		}
	}

	// Extract PRs
	for _, d := range tracker.GetByType(deliverable.TypePR) {
		label := d.Name
		if label == "" {
			label = "Pull Request"
		}
		outcome.PullRequests = append(outcome.PullRequests, OutcomeLink{Label: label, URL: d.Path})
	}

	// Extract issues
	for _, d := range tracker.GetByType(deliverable.TypeIssue) {
		label := d.Name
		if label == "" {
			label = "Issue"
		}
		outcome.Issues = append(outcome.Issues, OutcomeLink{Label: label, URL: d.Path})
	}

	// Extract deployments
	for _, d := range tracker.GetByType(deliverable.TypeDeployment) {
		label := d.Name
		if label == "" {
			label = "Deployment"
		}
		outcome.Deployments = append(outcome.Deployments, OutcomeLink{Label: label, URL: d.Path})
	}

	// Count detail-level deliverables (non-outcome-worthy)
	for _, d := range all {
		if !isOutcomeWorthy(d.Type) {
			outcome.ArtifactCount++
		}
	}

	// Count contracts
	contracts := tracker.GetByType(deliverable.TypeContract)
	outcome.ContractsTotal = len(contracts)
	for _, c := range contracts {
		if c.Metadata != nil {
			if failed, ok := c.Metadata["failed"].(bool); ok && failed {
				outcome.ContractsFailed++
				outcome.FailedContracts = append(outcome.FailedContracts, ContractFailure{
					StepID:  c.StepID,
					Type:    c.Description,
					Message: fmt.Sprintf("%v", c.Metadata["error"]),
				})
				continue
			}
		}
		outcome.ContractsPassed++
	}

	// Generate next steps
	outcome.NextSteps = GenerateNextSteps(outcome)

	return outcome
}

// GenerateNextSteps produces contextual follow-up suggestions based on the
// pipeline outcome.
func GenerateNextSteps(outcome *PipelineOutcome) []NextStep {
	var steps []NextStep

	// If PR exists, suggest reviewing it
	for _, pr := range outcome.PullRequests {
		steps = append(steps, NextStep{
			Label: fmt.Sprintf("Review the pull request: %s", pr.Label),
			URL:   pr.URL,
		})
	}

	// If branch was pushed, suggest viewing on remote
	if outcome.Branch != "" && outcome.Pushed && outcome.PushError == "" {
		ref := outcome.RemoteRef
		if ref == "" {
			ref = "origin/" + outcome.Branch
		}
		steps = append(steps, NextStep{
			Label:   "View changes on remote",
			Command: fmt.Sprintf("git log %s", ref),
		})
	}

	// If workspace path is set, suggest inspection
	if outcome.WorkspacePath != "" {
		steps = append(steps, NextStep{
			Label:   fmt.Sprintf("Inspect workspace at %s", outcome.WorkspacePath),
			Command: fmt.Sprintf("ls %s", outcome.WorkspacePath),
		})
	}

	return steps
}

// maxDefaultDeliverables is the maximum number of deliverables shown in default mode.
const maxDefaultDeliverables = 5

// RenderOutcomeSummary formats a PipelineOutcome as a human-readable string.
// When verbose is false, only summary counts and top outcome-worthy items are shown.
// When verbose is true, full deliverable details are included.
func RenderOutcomeSummary(outcome *PipelineOutcome, verbose bool, formatter *Formatter) string {
	if outcome == nil {
		return ""
	}

	var b strings.Builder

	// Build a set of failed step IDs for marking
	failedSet := make(map[string]bool, len(outcome.FailedStepIDs))
	for _, id := range outcome.FailedStepIDs {
		failedSet[id] = true
	}

	// --- Outcomes section ---
	hasOutcomes := outcome.Branch != "" || len(outcome.PullRequests) > 0 ||
		len(outcome.Issues) > 0 || len(outcome.Deployments) > 0

	if hasOutcomes {
		b.WriteString(formatter.Bold("Outcomes"))
		b.WriteString("\n")

		// Branch
		if outcome.Branch != "" {
			if outcome.PushError != "" {
				b.WriteString(fmt.Sprintf("  %s Branch: %s %s\n",
					formatter.Warning("⚠"),
					outcome.Branch,
					formatter.Warning(fmt.Sprintf("(push failed: %s)", outcome.PushError))))
			} else if outcome.Pushed {
				ref := outcome.RemoteRef
				if ref == "" {
					ref = "origin/" + outcome.Branch
				}
				b.WriteString(fmt.Sprintf("  %s Branch: %s → %s\n",
					formatter.Success("✓"),
					outcome.Branch,
					formatter.Muted(ref)))
			} else {
				b.WriteString(fmt.Sprintf("  %s Branch: %s %s\n",
					formatter.Success("✓"),
					outcome.Branch,
					formatter.Muted("(local only)")))
			}
		}

		// Pull Requests
		for _, pr := range outcome.PullRequests {
			b.WriteString(fmt.Sprintf("  %s %s: %s\n",
				formatter.Success("✓"),
				pr.Label,
				formatter.Primary(pr.URL)))
		}

		// Issues
		for _, issue := range outcome.Issues {
			b.WriteString(fmt.Sprintf("  %s %s: %s\n",
				formatter.Success("✓"),
				issue.Label,
				formatter.Primary(issue.URL)))
		}

		// Deployments
		for _, dep := range outcome.Deployments {
			b.WriteString(fmt.Sprintf("  %s %s: %s\n",
				formatter.Success("✓"),
				dep.Label,
				formatter.Primary(dep.URL)))
		}

		b.WriteString("\n")
	}

	// --- Artifact summary ---
	if outcome.ArtifactCount > 0 {
		b.WriteString(fmt.Sprintf("  %s\n", formatter.Muted(fmt.Sprintf("%d artifacts produced", outcome.ArtifactCount))))
	}

	// --- Contract summary ---
	if outcome.ContractsTotal > 0 {
		if outcome.ContractsFailed > 0 {
			b.WriteString(fmt.Sprintf("  %s\n",
				formatter.Error(fmt.Sprintf("%d/%d contracts passed", outcome.ContractsPassed, outcome.ContractsTotal))))
			for _, cf := range outcome.FailedContracts {
				prefix := ""
				if failedSet[cf.StepID] {
					prefix = "[FAILED] "
				}
				b.WriteString(fmt.Sprintf("    %s%s %s: %s\n",
					prefix,
					formatter.Error("✗"),
					cf.StepID,
					cf.Message))
			}
		} else {
			b.WriteString(fmt.Sprintf("  %s\n",
				formatter.Success(fmt.Sprintf("%d/%d contracts passed", outcome.ContractsPassed, outcome.ContractsTotal))))
		}
	}

	// --- Verbose: full deliverable list ---
	if verbose && len(outcome.AllDeliverables) > 0 {
		b.WriteString("\n")
		b.WriteString(formatter.Muted("  Deliverables:"))
		b.WriteString("\n")

		// Sort by type priority, then by creation time (newest first)
		sorted := make([]*deliverable.Deliverable, len(outcome.AllDeliverables))
		copy(sorted, outcome.AllDeliverables)
		sort.Slice(sorted, func(i, j int) bool {
			pi := deliverableTypePriority(sorted[i].Type)
			pj := deliverableTypePriority(sorted[j].Type)
			if pi != pj {
				return pi < pj
			}
			return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
		})

		for _, d := range sorted {
			prefix := ""
			if failedSet[d.StepID] {
				prefix = formatter.Warning("[FAILED] ")
			}
			b.WriteString(fmt.Sprintf("    %s%s\n", prefix, d.String()))
		}
	} else if !verbose && outcome.ArtifactCount > maxDefaultDeliverables {
		// Show top N by priority
		sorted := make([]*deliverable.Deliverable, 0)
		for _, d := range outcome.AllDeliverables {
			if !isOutcomeWorthy(d.Type) {
				sorted = append(sorted, d)
			}
		}
		sort.Slice(sorted, func(i, j int) bool {
			pi := deliverableTypePriority(sorted[i].Type)
			pj := deliverableTypePriority(sorted[j].Type)
			if pi != pj {
				return pi < pj
			}
			return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
		})

		shown := maxDefaultDeliverables
		if shown > len(sorted) {
			shown = len(sorted)
		}
		if shown > 0 {
			b.WriteString("\n")
			for _, d := range sorted[:shown] {
				prefix := ""
				if failedSet[d.StepID] {
					prefix = formatter.Warning("[FAILED] ")
				}
				b.WriteString(fmt.Sprintf("    %s%s\n", prefix, d.String()))
			}
			remaining := len(sorted) - shown
			if remaining > 0 {
				b.WriteString(fmt.Sprintf("    %s\n", formatter.Muted(fmt.Sprintf("... and %d more", remaining))))
			}
		}
	}

	// --- Next Steps ---
	if len(outcome.NextSteps) > 0 {
		b.WriteString("\n")
		b.WriteString(formatter.Bold("Next Steps"))
		b.WriteString("\n")
		for _, step := range outcome.NextSteps {
			if step.URL != "" {
				b.WriteString(fmt.Sprintf("  → %s\n    %s\n", step.Label, formatter.Primary(step.URL)))
			} else if step.Command != "" {
				b.WriteString(fmt.Sprintf("  → %s\n    %s\n", step.Label, formatter.Muted(step.Command)))
			} else {
				b.WriteString(fmt.Sprintf("  → %s\n", step.Label))
			}
		}
	}

	return b.String()
}

// ToOutcomesJSON converts a PipelineOutcome to the JSON-serializable OutcomesJSON
// format used in the final completion event.
func (o *PipelineOutcome) ToOutcomesJSON() *event.OutcomesJSON {
	if o == nil {
		return nil
	}

	result := &event.OutcomesJSON{
		Branch:    o.Branch,
		Pushed:    o.Pushed,
		RemoteRef: o.RemoteRef,
		PushError: o.PushError,
	}

	// Convert pull requests
	result.PullRequests = make([]event.OutcomeLinkJSON, len(o.PullRequests))
	for i, pr := range o.PullRequests {
		result.PullRequests[i] = event.OutcomeLinkJSON{Label: pr.Label, URL: pr.URL}
	}

	// Convert issues
	result.Issues = make([]event.OutcomeLinkJSON, len(o.Issues))
	for i, issue := range o.Issues {
		result.Issues[i] = event.OutcomeLinkJSON{Label: issue.Label, URL: issue.URL}
	}

	// Convert deployments
	result.Deployments = make([]event.OutcomeLinkJSON, len(o.Deployments))
	for i, dep := range o.Deployments {
		result.Deployments[i] = event.OutcomeLinkJSON{Label: dep.Label, URL: dep.URL}
	}

	// Convert all deliverables
	result.Deliverables = make([]event.DeliverableJSON, len(o.AllDeliverables))
	for i, d := range o.AllDeliverables {
		result.Deliverables[i] = event.DeliverableJSON{
			Type:        string(d.Type),
			Name:        d.Name,
			Path:        d.Path,
			Description: d.Description,
			StepID:      d.StepID,
		}
	}

	return result
}
