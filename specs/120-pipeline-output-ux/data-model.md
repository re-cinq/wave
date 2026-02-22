# Data Model: Pipeline Output UX — Surface Key Outcomes

**Date**: 2026-02-20
**Spec**: `specs/120-pipeline-output-ux/spec.md`

## Entity Overview

```
┌─────────────────────────┐
│   DefaultPipelineExecutor│
│   (executor.go)         │
│                         │
│  deliverableTracker ────┼──────► Tracker
│                         │         │
└─────────────────────────┘         │ GetAll() / GetByType()
                                    ▼
                              []*Deliverable
                                    │
                                    │ (post-execution in run.go)
                                    ▼
                            ┌───────────────────┐
                            │  PipelineOutcome   │
                            │  (display/outcome) │
                            │                    │
                            │  Branch            │
                            │  Pushed            │
                            │  PushError         │
                            │  PullRequests []   │
                            │  Issues       []   │
                            │  Deployments  []   │
                            │  Reports      []   │
                            │  ArtifactCount     │
                            │  ContractResults   │
                            │  TotalTokens       │
                            │  Duration          │
                            │  NextSteps    []   │
                            └───────┬───────────┘
                                    │
                      ┌─────────────┼─────────────┐
                      ▼             ▼             ▼
               OutcomeSummary  OutcomesJSON    (quiet: suppressed)
               (text/auto)    (json mode)
```

## New Types

### 1. DeliverableType Constants (extend existing)

**Package**: `internal/deliverable`
**File**: `types.go`

```go
// New type constants (added to existing const block)
const (
    TypeBranch DeliverableType = "branch"
    TypeIssue  DeliverableType = "issue"
)
```

These join the existing types: `file`, `url`, `pr`, `deployment`, `log`, `contract`, `artifact`, `other`.

### 2. Branch Deliverable Metadata Convention

Branch deliverables use the existing `Metadata map[string]any` field:

| Key | Type | Description |
|-----|------|-------------|
| `pushed` | `bool` | Whether branch was pushed to remote |
| `remote_ref` | `string` | Remote reference (e.g., `origin/feat/my-branch`) |
| `push_error` | `string` | Error message if push failed |

### 3. NewBranchDeliverable Constructor

**Package**: `internal/deliverable`
**File**: `types.go`

```go
func NewBranchDeliverable(stepID, branchName, worktreePath, description string) *Deliverable {
    return &Deliverable{
        Type:        TypeBranch,
        Name:        branchName,
        Path:        worktreePath,
        Description: description,
        StepID:      stepID,
        CreatedAt:   time.Now(),
        Metadata: map[string]any{
            "pushed": false,
        },
    }
}
```

### 4. NewIssueDeliverable Constructor

**Package**: `internal/deliverable`
**File**: `types.go`

```go
func NewIssueDeliverable(stepID, name, issueURL, description string) *Deliverable {
    return &Deliverable{
        Type:        TypeIssue,
        Name:        name,
        Path:        issueURL,
        Description: description,
        StepID:      stepID,
        CreatedAt:   time.Now(),
    }
}
```

### 5. PipelineOutcome

**Package**: `internal/display`
**File**: `outcome.go`

Read-only summary struct constructed in `run.go` after pipeline execution completes.

```go
type PipelineOutcome struct {
    // Identity
    PipelineName string
    RunID        string

    // Status
    Success  bool
    Duration time.Duration
    Tokens   int

    // Key outcomes (outcome-worthy deliverables)
    Branch       string   // Branch name (empty if no branch created)
    Pushed       bool     // Whether branch was pushed
    RemoteRef    string   // Remote reference (e.g., "origin/branch-name")
    PushError    string   // Push error message (empty if no error)
    PullRequests []OutcomeLink // PR URLs with labels
    Issues       []OutcomeLink // Issue URLs with labels
    Deployments  []OutcomeLink // Deployment URLs with labels

    // Reports and key files (top N outcome-worthy files)
    Reports []OutcomeFile

    // Artifact/contract summary (counts only in default mode)
    ArtifactCount     int
    ContractsPassed   int
    ContractsFailed   int
    ContractsTotal    int
    FailedContracts   []ContractFailure // Always shown, even in default mode

    // Next steps
    NextSteps []NextStep

    // Workspace info
    WorkspacePath string

    // Verbose data (full lists, only rendered in verbose mode)
    AllDeliverables []*Deliverable
}
```

### 6. Supporting Types

```go
// OutcomeLink represents a URL outcome (PR, issue, deployment)
type OutcomeLink struct {
    Label string // e.g., "Pull Request", "Issue #42"
    URL   string
}

// OutcomeFile represents a file outcome (report, key deliverable)
type OutcomeFile struct {
    Label string // e.g., "Spec Output", "Test Results"
    Path  string // Absolute or workspace-relative path
}

// ContractFailure captures a failed contract for prominent display
type ContractFailure struct {
    StepID  string
    Type    string // Contract type (json_schema, test_suite, etc.)
    Message string // Failure reason
}

// NextStep represents a suggested follow-up action
type NextStep struct {
    Label   string // e.g., "Review the pull request"
    Command string // Optional command (e.g., "gh pr view <url>")
    URL     string // Optional URL to open
}
```

### 7. OutcomesJSON (for --output json)

**Package**: `internal/event`
**File**: `emitter.go` (extend Event struct)

```go
// OutcomesJSON is the structured outcome data included in the final JSON completion event
type OutcomesJSON struct {
    Branch       string          `json:"branch,omitempty"`
    Pushed       bool            `json:"pushed"`
    RemoteRef    string          `json:"remote_ref,omitempty"`
    PushError    string          `json:"push_error,omitempty"`
    PullRequests []OutcomeLinkJSON `json:"pull_requests,omitempty"`
    Issues       []OutcomeLinkJSON `json:"issues,omitempty"`
    Deployments  []OutcomeLinkJSON `json:"deployments,omitempty"`
    Deliverables []DeliverableJSON `json:"deliverables,omitempty"`
}

type OutcomeLinkJSON struct {
    Label string `json:"label"`
    URL   string `json:"url"`
}

type DeliverableJSON struct {
    Type        string `json:"type"`
    Name        string `json:"name"`
    Path        string `json:"path"`
    Description string `json:"description,omitempty"`
    StepID      string `json:"step_id"`
}
```

The `Event` struct gets a new field:
```go
// In event.Event struct
Outcomes *OutcomesJSON `json:"outcomes,omitempty"`
```

## Modified Types

### Deliverable.String() Extension

The existing `String()` method in `types.go` needs two new cases in the icon switch:

```go
case TypeBranch:
    icon = "🌿" // nerd font / "⎇" // ASCII fallback
case TypeIssue:
    icon = "📌" // nerd font / "!" // ASCII fallback
```

### Tracker.AddBranch / AddIssue Convenience Methods

```go
func (t *Tracker) AddBranch(stepID, branchName, worktreePath, description string) {
    t.Add(NewBranchDeliverable(stepID, branchName, worktreePath, description))
}

func (t *Tracker) AddIssue(stepID, name, issueURL, description string) {
    t.Add(NewIssueDeliverable(stepID, name, issueURL, description))
}
```

### Tracker.UpdateMetadata (new method)

For updating branch push status after publish steps:

```go
func (t *Tracker) UpdateMetadata(deliverableType DeliverableType, name string, key string, value any) {
    t.mu.Lock()
    defer t.mu.Unlock()
    for _, d := range t.deliverables {
        if d.Type == deliverableType && d.Name == name {
            if d.Metadata == nil {
                d.Metadata = make(map[string]any)
            }
            d.Metadata[key] = value
            return
        }
    }
}
```

## Relationship to Existing Entities

| Existing Entity | Relationship | Change |
|----------------|-------------|--------|
| `deliverable.Tracker` | Source of truth | Add `TypeBranch`/`TypeIssue` constants, `AddBranch()`/`AddIssue()` methods, `UpdateMetadata()` |
| `deliverable.Deliverable` | Extended | Two new type constants, two new constructors, String() updated |
| `event.Event` | Extended | New `Outcomes *OutcomesJSON` field (omitempty) |
| `display.Formatter` | Used by | `OutcomeSummary` uses Formatter for styled rendering |
| `pipeline.DefaultPipelineExecutor` | Instrumented | Records branch deliverables on worktree creation |
| `commands.runRun()` | Modified | Constructs `PipelineOutcome`, renders via `OutcomeSummary` |

## Data Flow

1. **During execution**: `executor.go` records deliverables via `Tracker.Add*()`
   - Worktree creation → `Tracker.AddBranch(stepID, branchName, path, desc)`
   - Publish step completion → `Tracker.UpdateMetadata(TypeBranch, branchName, "pushed", true)`
   - PR creation → `Tracker.AddPR()` (existing)
   - Issue creation → `Tracker.AddIssue()` (new)

2. **Post-execution** (in `run.go`):
   - Construct `PipelineOutcome` from `Tracker.GetAll()` + step metadata
   - For auto/text modes: render via `OutcomeSummary.Render(outcome, verbose, formatter)`
   - For JSON mode: convert to `OutcomesJSON` and attach to final completion event
   - For quiet mode: suppress outcome summary (existing behavior)
