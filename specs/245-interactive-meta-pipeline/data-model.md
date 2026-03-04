# Data Model: Interactive Meta-Pipeline Orchestrator

**Feature**: `245-interactive-meta-pipeline`  
**Date**: 2026-03-04  
**Phase**: 1 — Design & Contracts

## Entity Definitions

### HealthReport

Aggregated results from all parallel health check jobs. This is the primary output of Phase 1 (System & Codebase Health Check).

```go
// Package: internal/meta

// HealthReport aggregates results from all parallel health check jobs.
type HealthReport struct {
    Timestamp     time.Time           `json:"timestamp"`
    Duration      time.Duration       `json:"duration_ms"`
    Init          InitCheckResult     `json:"init"`
    Dependencies  DependencyReport    `json:"dependencies"`
    Codebase      CodebaseMetrics     `json:"codebase"`
    Platform      PlatformProfile     `json:"platform"`
    Errors        []HealthCheckError  `json:"errors,omitempty"`
}

// InitCheckResult holds the result of the initialization check.
type InitCheckResult struct {
    ManifestFound   bool      `json:"manifest_found"`
    ManifestValid   bool      `json:"manifest_valid"`
    WaveVersion     string    `json:"wave_version"`
    LastConfigDate  time.Time `json:"last_config_date,omitempty"`
    Error           string    `json:"error,omitempty"`
}

// DependencyReport summarizes tool and skill availability.
type DependencyReport struct {
    Tools  []DependencyStatus `json:"tools"`
    Skills []DependencyStatus `json:"skills"`
}

// DependencyStatus represents the availability of a single dependency.
type DependencyStatus struct {
    Name            string `json:"name"`
    Kind            string `json:"kind"`     // "tool" or "skill"
    Available       bool   `json:"available"`
    AutoInstallable bool   `json:"auto_installable"`
    Message         string `json:"message,omitempty"`
}

// CodebaseMetrics holds repository activity and health indicators.
type CodebaseMetrics struct {
    RecentCommits   int               `json:"recent_commits"`     // Commits in last 7 days
    OpenIssueCount  int               `json:"open_issue_count"`
    OpenPRCount     int               `json:"open_pr_count"`
    PRsByStatus     map[string]int    `json:"prs_by_status"`      // "open", "merged", "changes_requested"
    BranchCount     int               `json:"branch_count"`
    LastCommitDate  time.Time         `json:"last_commit_date"`
    APIAvailable    bool              `json:"api_available"`      // true if platform API was reachable
    Source          string            `json:"source"`             // "github_api", "git_local"
}

// HealthCheckError represents a non-fatal error from a specific check.
type HealthCheckError struct {
    Check   string `json:"check"`   // "init", "dependencies", "codebase", "platform"
    Message string `json:"message"`
    Timeout bool   `json:"timeout"` // true if the check timed out
}
```

### PlatformProfile

Detected hosting platform identity with available API endpoints.

```go
// Package: internal/platform

// PlatformType enumerates supported hosting platforms.
type PlatformType string

const (
    PlatformGitHub    PlatformType = "github"
    PlatformGitLab    PlatformType = "gitlab"
    PlatformBitbucket PlatformType = "bitbucket"
    PlatformGitea     PlatformType = "gitea"
    PlatformUnknown   PlatformType = "unknown"
)

// PlatformProfile represents a detected hosting platform.
type PlatformProfile struct {
    Type           PlatformType `json:"type"`
    Owner          string       `json:"owner"`
    Repo           string       `json:"repo"`
    APIURL         string       `json:"api_url,omitempty"`
    CLIURL         string       `json:"cli_tool,omitempty"`     // "gh", "glab", etc.
    PipelineFamily string       `json:"pipeline_family"`        // "gh", "gl", "bb", "gt"
    AdditionalRemotes []RemoteInfo `json:"additional_remotes,omitempty"`
}

// RemoteInfo describes a git remote.
type RemoteInfo struct {
    Name     string       `json:"name"`
    URL      string       `json:"url"`
    Platform PlatformType `json:"platform"`
}
```

### PipelineProposal

A recommended pipeline run or sequence with rationale and pre-filled input.

```go
// Package: internal/meta

// ProposalType distinguishes between single pipelines, parallel sets, and sequences.
type ProposalType string

const (
    ProposalSingle   ProposalType = "single"
    ProposalParallel ProposalType = "parallel"
    ProposalSequence ProposalType = "sequence"
)

// PipelineProposal represents a recommended pipeline run or sequence.
type PipelineProposal struct {
    ID           string       `json:"id"`            // Unique proposal ID (e.g., "p1", "p2")
    Type         ProposalType `json:"type"`
    Pipelines    []string     `json:"pipelines"`     // Pipeline name(s)
    Rationale    string       `json:"rationale"`     // Why this is recommended
    PrefilledInput string     `json:"prefilled_input"` // Suggested input string
    Priority     int          `json:"priority"`       // Lower = higher priority
    DepsReady    bool         `json:"deps_ready"`    // All dependencies satisfied
    MissingDeps  []string     `json:"missing_deps,omitempty"`
}
```

### ProposalSelection

User's choice from the interactive menu.

```go
// Package: internal/meta

// ProposalSelection represents the user's choice from the interactive menu.
type ProposalSelection struct {
    Proposals    []PipelineProposal `json:"proposals"`      // Selected proposal(s)
    ModifiedInputs map[string]string `json:"modified_inputs"` // Pipeline name → modified input
    ExecutionMode  ProposalType     `json:"execution_mode"`  // How to execute (single/parallel/sequence)
}
```

### CodebaseProfile (Phase 3)

Analysis of the repository's characteristics for auto-tuning.

```go
// Package: internal/meta

// SizeClass categorizes repository size.
type SizeClass string

const (
    SizeSmall    SizeClass = "small"    // < 10K LOC
    SizeMedium   SizeClass = "medium"   // 10K-100K LOC
    SizeLarge    SizeClass = "large"    // 100K-1M LOC
    SizeMonorepo SizeClass = "monorepo" // Multiple independent modules
)

// CodebaseProfile represents the analysis of a repository's characteristics.
type CodebaseProfile struct {
    Language     string    `json:"language"`       // Primary language
    Framework    string    `json:"framework"`      // Detected framework (if any)
    TestCommand  string    `json:"test_command"`   // Detected test command
    BuildCommand string    `json:"build_command"`  // Detected build command
    SourceGlob   string    `json:"source_glob"`    // Source file pattern
    Size         SizeClass `json:"size"`
    Structure    string    `json:"structure"`       // "single", "monorepo", "workspace"
    PackageCount int       `json:"package_count"`   // Number of packages/modules
}
```

### SequenceExecutor

Orchestrator for multi-pipeline sequences.

```go
// Package: internal/meta

// SequenceExecutor runs pipelines in order, managing cross-pipeline artifact handoff.
type SequenceExecutor struct {
    executorFactory func() *pipeline.DefaultPipelineExecutor
    emitter         event.EventEmitter
    wsRoot          string
}

// SequenceResult holds the outcome of a multi-pipeline sequence.
type SequenceResult struct {
    Pipelines      []SequencePipelineResult `json:"pipelines"`
    TotalDuration  time.Duration            `json:"total_duration_ms"`
    FailedAt       int                      `json:"failed_at"`  // -1 if all succeeded
}

// SequencePipelineResult holds the result of one pipeline in a sequence.
type SequencePipelineResult struct {
    PipelineName  string        `json:"pipeline_name"`
    Status        string        `json:"status"`   // "completed", "failed", "skipped"
    Duration      time.Duration `json:"duration_ms"`
    Error         string        `json:"error,omitempty"`
    ArtifactPaths map[string]string `json:"artifact_paths,omitempty"`
}
```

## Entity Relationships

```
HealthReport ─────────────────────────────────────────────────────┐
│  Contains:                                                       │
│  ├── InitCheckResult (1:1)                                       │
│  ├── DependencyReport (1:1) ──── DependencyStatus (1:N)         │
│  ├── CodebaseMetrics (1:1)                                       │
│  └── PlatformProfile (1:1) ──── RemoteInfo (1:N)                │
└──────────────────────────────────────────────────────────────────┘
                  │
                  │ feeds into
                  ▼
ProposalEngine ────────────────────────────────────────────────────┐
│  Produces:                                                       │
│  └── PipelineProposal (1:N) ── ranked by priority               │
└──────────────────────────────────────────────────────────────────┘
                  │
                  │ user selects from
                  ▼
ProposalSelection ─────────────────────────────────────────────────┐
│  Dispatches to:                                                  │
│  ├── DefaultPipelineExecutor (single/parallel)                   │
│  └── SequenceExecutor (sequence) ── SequenceResult               │
└──────────────────────────────────────────────────────────────────┘
```

## Package Organization

```
internal/
├── meta/                    # NEW — Meta-pipeline orchestrator
│   ├── health.go            # HealthReport, parallel health checks
│   ├── health_test.go
│   ├── proposal.go          # ProposalEngine, PipelineProposal
│   ├── proposal_test.go
│   ├── selection.go         # ProposalSelection, interactive dispatch
│   ├── selection_test.go
│   ├── sequence.go          # SequenceExecutor, artifact handoff
│   ├── sequence_test.go
│   ├── tuning.go            # CodebaseProfile (Phase 3)
│   └── tuning_test.go
├── platform/                # NEW — Platform detection
│   ├── detect.go            # Detect(), PlatformProfile
│   └── detect_test.go
├── tui/
│   ├── proposal_selector.go # NEW — RunProposalSelector()
│   └── health_report.go     # NEW — RenderHealthReport()
└── pipeline/
    ├── meta.go              # MODIFIED — remove extractYAMLLegacy
    ├── context.go           # MODIFIED — remove legacy template vars
    └── resume.go            # MODIFIED — remove legacy workspace lookup
```
