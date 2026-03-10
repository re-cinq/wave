# Data Model: Init Merge & Upgrade Workflow

**Feature**: #230 — Init Merge & Upgrade Workflow  
**Date**: 2026-03-04

## Key Entities

### 1. FileChangeEntry

Represents a single file's status in the merge change summary.

```go
// FileChangeEntry represents one file in the change summary.
type FileChangeEntry struct {
    // RelPath is the relative path from project root (e.g., ".wave/personas/navigator.md")
    RelPath  string
    // Category is the asset category: "persona", "pipeline", "contract", "prompt"
    Category string
    // Status is one of: "new", "preserved", "up_to_date"
    Status   FileStatus
}

type FileStatus string

const (
    FileStatusNew      FileStatus = "new"        // File does not exist, will be created
    FileStatusPreserved FileStatus = "preserved"  // File exists, differs from default (user-modified)
    FileStatusUpToDate FileStatus = "up_to_date"  // File exists, matches default byte-for-byte
)
```

**Lifecycle**: Created during `computeChangeSummary()`, consumed by `displayChangeSummary()` and `applyChanges()`. Ephemeral — not persisted.

**Invariants**:
- Every file that would be touched by init --merge MUST appear in the summary
- Status is determined by byte-for-byte comparison with embedded defaults
- Files with status "preserved" are NEVER written to

### 2. ManifestChangeEntry

Represents a single key-level change in the manifest deep-merge.

```go
// ManifestChangeEntry represents a change to a manifest key.
type ManifestChangeEntry struct {
    // KeyPath is the dot-separated path (e.g., "runtime.relay.token_threshold_percent")
    KeyPath string
    // Action is "added" (new key from defaults) or "preserved" (user value kept)
    Action  ManifestAction
}

type ManifestAction string

const (
    ManifestActionAdded    ManifestAction = "added"     // New key from defaults
    ManifestActionPreserved ManifestAction = "preserved" // User value takes precedence
)
```

**Lifecycle**: Created during manifest merge diff computation, consumed by the manifest change summary display. Ephemeral.

**Invariants**:
- User keys ALWAYS take precedence (action = "preserved")
- New default keys not present in user manifest get action = "added"
- No key is ever "removed" — merge only adds

### 3. ChangeSummary

Aggregates all changes for user review before mutation.

```go
// ChangeSummary holds the complete pre-mutation change report.
type ChangeSummary struct {
    // Files is the list of file-level changes (personas, pipelines, contracts, prompts)
    Files []FileChangeEntry
    // ManifestChanges is the list of key-level manifest changes
    ManifestChanges []ManifestChangeEntry
    // MergedManifest is the computed merged manifest (written only after confirmation)
    MergedManifest map[string]interface{}
    // Assets is the resolved asset set used for the merge
    Assets *initAssets
    // AlreadyUpToDate is true when there are zero changes needed
    AlreadyUpToDate bool
}
```

**Lifecycle**: Created by `computeChangeSummary()`, displayed by `displayChangeSummary()`, applied by `applyChanges()`. The entire pre-mutation → confirm → apply flow revolves around this struct.

**Invariants**:
- If `AlreadyUpToDate` is true, `Files` contains only `up_to_date` entries and `ManifestChanges` contains only `preserved` entries
- `MergedManifest` is always computed, even if not written (needed for diff display)

### 4. MigrationStatus (existing)

Already defined in `internal/state/types.go` and `internal/state/migrations.go`. No changes needed.

```go
// MigrationStatus (existing — internal/state/migration_runner.go)
type MigrationStatus struct {
    CurrentVersion    int
    AllMigrations     []Migration
    PendingMigrations []Migration
}
```

## Relationships

```
ChangeSummary
├── []FileChangeEntry (1:N — one per asset file)
├── []ManifestChangeEntry (1:N — one per manifest key diff)
├── MergedManifest (1:1 — computed merge result)
└── initAssets (1:1 — resolved defaults)

MigrationStatus (independent — used by `wave migrate` commands)
├── []Migration (1:N — all defined migrations)
└── []Migration (1:N — pending migrations)
```

## Storage

All entities in this feature are **ephemeral** — computed in-memory during `wave init --merge` execution. No new database tables or persistent storage required.

The only persistence is the final write:
- `wave.yaml` — merged manifest (existing file, overwritten atomically)
- `.wave/personas/*.md` — new persona files (only for status "new")
- `.wave/pipelines/*.yaml` — new pipeline files (only for status "new")
- `.wave/contracts/*.json` — new contract files (only for status "new")
- `.wave/prompts/**/*.md` — new prompt files (only for status "new")
