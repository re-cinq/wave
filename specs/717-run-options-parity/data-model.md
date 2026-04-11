# Data Model: Run Options Parity

**Feature**: #717 — Run Options Parity Across All Surfaces  
**Date**: 2026-04-11

## Entity Map

### 1. RunOptions (Canonical — CLI)

**Source**: `cmd/wave/commands/run.go:36-61`  
**Role**: Single source of truth for all pipeline run configuration. Every surface
ultimately maps its configuration to CLI flags consumed by this struct.

```go
type RunOptions struct {
    // Tier 1 — Essential (always visible)
    Pipeline          string  // Pipeline name
    Input             string  // Free-text input for the pipeline
    Adapter           string  // LLM backend override (claude, gemini, opencode, codex)
    Model             string  // Model override (tier name or literal)

    // Tier 2 — Execution (collapsible "Advanced")
    FromStep          string  // Resume from specific step
    Force             bool    // Skip validation when using --from-step
    DryRun            bool    // Show execution plan without running
    Detach            bool    // Run as background process
    Steps             string  // Comma-separated step names to include
    Exclude           string  // Comma-separated step names to exclude
    Timeout           int     // Minutes (0 = infinite)
    OnFailure         string  // "halt" (default) | "skip"

    // Tier 3 — Continuous
    Continuous        bool    // Enable continuous mode
    Source            string  // Work item source URI
    MaxIterations     int     // 0 = unlimited
    Delay             string  // Duration between iterations (e.g. "5s")

    // Tier 4 — Dev/Debug (API + CLI only)
    Mock              bool    // Use mock adapter
    PreserveWorkspace bool    // Keep workspace from previous run
    AutoApprove       bool    // Auto-approve approval gates
    NoRetro           bool    // Skip retrospective generation
    ForceModel        bool    // Override all step/persona model tiers

    // Internal
    Manifest          string  // Path to manifest file
    RunID             string  // Resume from specific run ID
    Output            OutputConfig
}
```

**No schema changes needed** — this struct is already complete.

---

### 2. webui.RunOptions (Server-Side)

**Source**: `internal/webui/handlers_control.go:30-37`  
**Role**: Carries CLI-parity options from WebUI API handlers to `spawnDetachedRun()`.

**Current state** (6 fields):
```go
type RunOptions struct {
    Model   string
    Adapter string
    DryRun  bool
    Timeout int
    Steps   string
    Exclude string
}
```

**Target state** (add Tier 2–4 fields):
```go
type RunOptions struct {
    // Tier 1
    Model       string
    Adapter     string

    // Tier 2
    DryRun      bool
    FromStep    string
    Force       bool
    Detach      bool
    Timeout     int
    Steps       string
    Exclude     string
    OnFailure   string

    // Tier 3
    Continuous    bool
    Source        string
    MaxIterations int
    Delay         string

    // Tier 4
    Mock              bool
    PreserveWorkspace bool
    AutoApprove       bool
    NoRetro           bool
    ForceModel        bool
}
```

**Impact**: `spawnDetachedRun()` must wire all new fields to subprocess flags.
`launchPipelineExecution()` callers (`handleStartPipeline`, `handleSubmitRun`,
`handleAPIStartFromIssue`, new `handleAPIStartFromPR`) must populate the struct
from their respective request types.

---

### 3. StartPipelineRequest (API — Pipeline Detail)

**Source**: `internal/webui/types.go:205-213`  
**Role**: JSON request body for `POST /api/pipelines/{name}/start`.

**Current state** (6 fields): Input, Model, Adapter, DryRun, Timeout, Steps, Exclude

**Target state** — add all Tier 1–4 fields:
```go
type StartPipelineRequest struct {
    Input         string `json:"input"`
    Model         string `json:"model,omitempty"`
    Adapter       string `json:"adapter,omitempty"`
    DryRun        bool   `json:"dry_run,omitempty"`
    FromStep      string `json:"from_step,omitempty"`
    Force         bool   `json:"force,omitempty"`
    Detach        bool   `json:"detach,omitempty"`
    Timeout       int    `json:"timeout,omitempty"`
    Steps         string `json:"steps,omitempty"`
    Exclude       string `json:"exclude,omitempty"`
    OnFailure     string `json:"on_failure,omitempty"`
    Continuous    bool   `json:"continuous,omitempty"`
    Source        string `json:"source,omitempty"`
    MaxIterations int    `json:"max_iterations,omitempty"`
    Delay         string `json:"delay,omitempty"`
    // Tier 4
    Mock              bool `json:"mock,omitempty"`
    PreserveWorkspace bool `json:"preserve_workspace,omitempty"`
    AutoApprove       bool `json:"auto_approve,omitempty"`
    NoRetro           bool `json:"no_retro,omitempty"`
    ForceModel        bool `json:"force_model,omitempty"`
}
```

---

### 4. SubmitRunRequest (API — Runs Page)

**Source**: `internal/webui/types.go:540-549`  
**Role**: JSON request body for `POST /api/runs`.

**Target state**: Same field set as `StartPipelineRequest` plus `Pipeline string`.

---

### 5. StartIssueRequest (API — Issue Page) [NEW]

**Source**: Currently inline anonymous struct in `handlers_issues.go:48-51`  
**Role**: JSON request body for `POST /api/issues/start`.

**Target state** — named type with Tier 1–3:
```go
type StartIssueRequest struct {
    IssueURL      string `json:"issue_url"`
    PipelineName  string `json:"pipeline_name"`
    // Tier 1
    Model         string `json:"model,omitempty"`
    Adapter       string `json:"adapter,omitempty"`
    // Tier 2
    DryRun        bool   `json:"dry_run,omitempty"`
    FromStep      string `json:"from_step,omitempty"`
    Force         bool   `json:"force,omitempty"`
    Detach        bool   `json:"detach,omitempty"`
    Timeout       int    `json:"timeout,omitempty"`
    Steps         string `json:"steps,omitempty"`
    Exclude       string `json:"exclude,omitempty"`
    OnFailure     string `json:"on_failure,omitempty"`
    // Tier 3
    Continuous    bool   `json:"continuous,omitempty"`
    Source        string `json:"source,omitempty"`
    MaxIterations int    `json:"max_iterations,omitempty"`
    Delay         string `json:"delay,omitempty"`
}
```

---

### 6. StartPRRequest (API — PR Page) [NEW]

**Source**: Does not exist yet  
**Role**: JSON request body for `POST /api/prs/start`.

**Target state** — mirrors `StartIssueRequest` with PR-specific fields:
```go
type StartPRRequest struct {
    PRURL         string `json:"pr_url"`
    PipelineName  string `json:"pipeline_name"`
    // Tier 1
    Model         string `json:"model,omitempty"`
    Adapter       string `json:"adapter,omitempty"`
    // Tier 2
    DryRun        bool   `json:"dry_run,omitempty"`
    FromStep      string `json:"from_step,omitempty"`
    Force         bool   `json:"force,omitempty"`
    Detach        bool   `json:"detach,omitempty"`
    Timeout       int    `json:"timeout,omitempty"`
    Steps         string `json:"steps,omitempty"`
    Exclude       string `json:"exclude,omitempty"`
    OnFailure     string `json:"on_failure,omitempty"`
    // Tier 3
    Continuous    bool   `json:"continuous,omitempty"`
    Source        string `json:"source,omitempty"`
    MaxIterations int    `json:"max_iterations,omitempty"`
    Delay         string `json:"delay,omitempty"`
}
```

---

### 7. LaunchConfig (TUI)

**Source**: `internal/tui/pipeline_messages.go:46-54`  
**Role**: User's pipeline launch configuration from the TUI argument form.

**Current state** (5 fields): PipelineName, Input, ModelOverride, Flags, DryRun/Verbose/Debug

**Target state** — add typed fields for Tier 1–3:
```go
type LaunchConfig struct {
    PipelineName  string
    Input         string
    ModelOverride string
    Flags         []string // Legacy: --verbose, --debug, --output text/json, --mock
    DryRun        bool
    Verbose       bool
    Debug         bool
    // New typed fields (Tier 1–3)
    Adapter       string
    Timeout       int
    FromStep      string
    Steps         string
    Exclude       string
    Detach        bool
    OnFailure     string
}
```

**Impact**: `PipelineLauncher.Launch()` must map new typed fields to subprocess
flags instead of relying on `Flags []string` for them.

---

### 8. DefaultFlags (TUI)

**Source**: `internal/tui/run_selector.go:24-34`  
**Role**: Flags presented in the interactive run selector.

**Current state**: --verbose, --debug, --output text, --output json, --dry-run, --mock

**Target state**: Add --detach:
```go
func DefaultFlags() []Flag {
    return []Flag{
        {Name: "--verbose", Description: "Verbose output"},
        {Name: "--debug", Description: "Debug output"},
        {Name: "--output text", Description: "Show only pipeline output"},
        {Name: "--output json", Description: "Machine-readable JSON output"},
        {Name: "--dry-run", Description: "Validate without executing"},
        {Name: "--mock", Description: "Use mock adapter"},
        {Name: "--detach", Description: "Run in background"},
    }
}
```

---

## Validation Rules (Cross-Entity)

| Rule | Surfaces | Source |
|------|----------|--------|
| `continuous` + `from-step` mutually exclusive | All | `run.go:201-204` |
| `continuous` requires `source` | All | `run.go:207-210` |
| `from-step` with non-existent step → error | WebUI, TUI | Pipeline manifest steps |
| `steps` ∩ `exclude` non-empty → error | All | `pipeline.StepFilter.Validate()` |
| `on-failure` ∈ {`halt`, `skip`} | All | `run.go:182` |
| `force` only meaningful with `from-step` | WebUI (visibility), All (behavior) | `run.go:167` |
| `detach` + approval gates requires `auto-approve` | CLI, API | `run.go:289-293` |
| `timeout` ≥ 0 (integer minutes) | All | `run.go:168` |
