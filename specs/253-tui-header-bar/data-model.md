# Data Model: TUI Header Bar

**Date**: 2026-03-05
**Branch**: `253-tui-header-bar`

## Entities

### HeaderModel (Bubble Tea Component)

Primary component in `internal/tui/header.go`. Replaces the current stub implementation.

```go
type HeaderModel struct {
    width        int
    metadata     HeaderMetadata
    logo         LogoAnimator
    provider     MetadataProvider
    refreshTimer time.Duration   // 30s periodic git refresh
}
```

**Relationships**: Composed into `AppModel` (already exists). Receives messages from `AppModel.Update()`.

### HeaderMetadata (Value Object)

Holds all displayable project metadata fields.

```go
type HeaderMetadata struct {
    // Git state (from git CLI)
    Branch       string   // Current branch or overridden pipeline branch
    CommitHash   string   // Abbreviated commit hash (7 chars)
    IsDirty      bool     // Working tree has uncommitted changes
    RemoteName   string   // e.g., "origin"

    // Manifest state (from wave.yaml)
    ProjectName  string   // From manifest.Metadata.Name
    RepoName     string   // From manifest.Metadata.Repo or git remote

    // Pipeline state (from state DB)
    RunningCount int      // Number of currently running pipelines
    HealthStatus HealthStatus // Aggregate health

    // GitHub state (from gh CLI)
    IssuesCount  int      // Open issues count
    GitHubState  GitHubAuthState // Auth state enum

    // Override state
    OverrideBranch string // Set when a finished pipeline is selected
}
```

### HealthStatus (Enum)

```go
type HealthStatus int

const (
    HealthOK   HealthStatus = iota // ● OK — no failures
    HealthWarn                      // ▲ WARN — soft failures
    HealthErr                       // ✗ ERR — hard failures
)
```

### GitHubAuthState (Enum)

```go
type GitHubAuthState int

const (
    GitHubNotConfigured GitHubAuthState = iota // No gh CLI or auth → "—"
    GitHubOffline                               // Auth exists, API unreachable → "[offline]"
    GitHubConnected                             // Working → show count
)
```

### LogoAnimator (Value Object)

Manages logo foreground color cycling.

```go
type LogoAnimator struct {
    palette    []lipgloss.Color // {"6", "4", "5"} — cyan, blue, magenta
    colorIndex int              // Current position in palette
    active     bool             // true when runningCount > 0
}
```

**Tick behavior**: When `active`, `tea.Tick(200ms, ...)` returns `LogoTickMsg{}`. On tick, `colorIndex = (colorIndex + 1) % len(palette)`. When `!active`, no ticks are scheduled and `colorIndex` resets to 0 (static cyan).

### MetadataProvider (Interface)

Abstraction for testability — decouples data fetching from rendering.

```go
type MetadataProvider interface {
    FetchGitState() (GitState, error)
    FetchManifestInfo() (ManifestInfo, error)
    FetchGitHubInfo(repo string) (GitHubInfo, error)
    FetchPipelineHealth() (HealthStatus, error)
}
```

### GitState (Value Object — Fetch Result)

```go
type GitState struct {
    Branch     string
    CommitHash string
    IsDirty    bool
    RemoteName string
}
```

### ManifestInfo (Value Object — Fetch Result)

```go
type ManifestInfo struct {
    ProjectName string
    RepoName    string // owner/repo format
}
```

### GitHubInfo (Value Object — Fetch Result)

```go
type GitHubInfo struct {
    AuthState   GitHubAuthState
    IssuesCount int
}
```

## Bubble Tea Messages

All inter-component communication uses typed messages:

```go
// Async fetch results
type GitStateMsg struct {
    State GitState
    Err   error
}

type ManifestInfoMsg struct {
    Info ManifestInfo
    Err  error
}

type GitHubInfoMsg struct {
    Info GitHubInfo
    Err  error
}

type PipelineHealthMsg struct {
    Health HealthStatus
    Err    error
}

// External state changes
type RunningCountMsg struct {
    Count int
}

type PipelineSelectedMsg struct {
    RunID      string
    BranchName string // Empty means no finished pipeline selected
}

// Internal timer messages
type LogoTickMsg struct{}
type GitRefreshTickMsg struct{}
```

## Schema Change (Prerequisite)

### Migration #7: Add `branch_name` to `pipeline_run`

```sql
-- Up
ALTER TABLE pipeline_run ADD COLUMN branch_name TEXT DEFAULT '';

-- Down (table recreation approach for SQLite < 3.35 compat)
-- Standard pattern from migration_definitions.go
```

### RunRecord Update

```go
type RunRecord struct {
    // ... existing fields ...
    BranchName   string    // NEW: Worktree branch for this run
}
```

### StateStore Interface Addition

```go
// Add to StateStore interface
UpdateRunBranch(runID string, branch string) error
```

## Column Priority Order (FR-009)

| Priority | Column       | Min Width | Placeholder | Error State      |
|----------|-------------|-----------|-------------|------------------|
| 1        | Logo        | 14        | (always)    | (always shown)   |
| 2        | Branch      | 10        | "…"         | "[no git]"       |
| 3        | Health      | 6         | "● …"       | "● OK" (default) |
| 4        | Repo/Project| 10        | "…"         | "[no project]"   |
| 5        | Clean/Dirty | 3         | "…"         | "…"              |
| 6        | Remote      | 8         | "…"         | "—"              |
| 7        | Issues      | 5         | "…"         | "—" or "[offline]"|
| 8        | Commit Hash | 9         | "…"         | "[no git]"       |
