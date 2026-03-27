# Implementation Plan: Fork/Rewind from Checkpoints

## Objective

Add `wave fork` and `wave rewind` CLI commands that enable non-destructive branching from any completed step in a prior run, and destructive rewind to replay from a checkpoint. This requires enriching the state DB with checkpoint data (workspace commit SHAs, artifact snapshots) at each step boundary.

## Approach

The implementation builds on the existing `ResumeManager` infrastructure. The key difference: resume works from a *failed* run going forward; fork works from a *completed* (or partially completed) run, creating an entirely new run. Rewind modifies an existing run's state destructively.

### Three layers:

1. **Checkpoint enrichment** — record workspace git commit SHA and artifact file hashes at each step completion in a new `checkpoint` table
2. **Fork executor** — new `ForkManager` in `internal/pipeline/` that creates a new run, copies checkpoint state (artifacts, workspace), and executes from the step after the fork point
3. **Rewind** — resets state DB records and workspace to a checkpoint, leaving the run ready for `wave resume`

## File Mapping

### New files

| Path | Purpose |
|------|---------|
| `cmd/wave/commands/fork.go` | `wave fork` CLI command |
| `cmd/wave/commands/rewind.go` | `wave rewind` CLI command |
| `internal/pipeline/fork.go` | `ForkManager` — fork execution logic |
| `internal/pipeline/fork_test.go` | Unit tests for ForkManager |
| `internal/pipeline/checkpoint.go` | Checkpoint recording and retrieval |
| `internal/pipeline/checkpoint_test.go` | Unit tests for checkpoint logic |
| `cmd/wave/commands/fork_test.go` | CLI integration tests for fork |
| `cmd/wave/commands/rewind_test.go` | CLI integration tests for rewind |

### Modified files

| Path | Change |
|------|--------|
| `cmd/wave/main.go` | Register `NewForkCmd()` and `NewRewindCmd()` |
| `internal/state/store.go` | Add checkpoint CRUD methods to `StateStore` interface |
| `internal/state/schema.sql` | Add `checkpoint` table |
| `internal/state/migration_definitions.go` | Add migration version 12 for checkpoint table |
| `internal/state/types.go` | Add `CheckpointRecord` type |
| `internal/pipeline/executor.go` | Call checkpoint recording after each step completion |

## Architecture Decisions

### 1. Checkpoint storage: SQLite table, not git refs

The issue suggests workspace commit SHAs. Rather than creating git tags/refs per checkpoint (which pollutes the git namespace), store checkpoint metadata in a new `checkpoint` table:

```sql
CREATE TABLE IF NOT EXISTS checkpoint (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    step_index INTEGER NOT NULL,
    workspace_path TEXT NOT NULL,
    workspace_commit_sha TEXT DEFAULT '',
    artifact_snapshot TEXT NOT NULL,  -- JSON: {"stepID:name": "path", ...}
    created_at INTEGER NOT NULL,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);
```

The `workspace_commit_sha` captures the git HEAD of worktree-based workspaces at step completion. For non-worktree workspaces, it's empty and workspace state relies on the directory still existing.

The `artifact_snapshot` is a JSON-serialized map of all artifact paths available at that checkpoint (cumulative from step 1 through the checkpoint step).

### 2. Fork creates a new run, copies artifacts, delegates to ResumeManager

Fork doesn't need a new executor — it prepares state and delegates to the existing resume infrastructure:

1. Create new run record via `store.CreateRun()`
2. Load checkpoint for the source run at the fork point
3. Copy artifact files to new workspace
4. If worktree workspace: create new worktree from checkpoint commit SHA
5. Call `ResumeManager.ResumeFromStep()` with the new run context

### 3. Rewind deletes forward state, doesn't touch artifacts

Rewind operates on the state DB only:
1. Delete step attempts, events, artifacts, and performance metrics for steps after the rewind point
2. Update run status back to `failed` (so `wave resume` can pick it up)
3. Workspace directories for steps after the rewind point are optionally cleaned up
4. The run can then be resumed with `wave resume <run-id>`

### 4. Step index resolution

The `--from-step` flag accepts both step IDs (e.g., `plan`) and numeric indices (e.g., `3`). This matches the existing resume behavior. For fork, `--from-step plan` means "fork from after the plan step completed" — the new run starts with the step *after* plan.

### 5. No nested fork tracking (first iteration)

The issue mentions nested forks as a missing info item. First iteration treats forked runs as regular runs — they can be forked again, but there's no parent-child lineage tracking. A `forked_from_run_id` field on `pipeline_run` provides basic lineage.

## Risks

| Risk | Mitigation |
|------|------------|
| Workspace directories cleaned up before fork | Check workspace exists before attempting fork; error with clear message |
| Worktree commit SHA not captured for mount-based workspaces | Only record SHA for worktree workspaces; mount-based fork copies directory contents |
| Large artifact copies slow down fork | Copy is file-based and sequential; acceptable for prototype phase |
| Rewind deletes data that can't be recovered | Require `--confirm` flag or print warning with 5s countdown |
| Checkpoint table grows unbounded | Add cleanup in `wave clean` to remove checkpoints for deleted runs |
| Race condition: fork while run is still executing | Check run status; refuse to fork a running run |

## Testing Strategy

### Unit tests

- `internal/pipeline/checkpoint_test.go`: checkpoint record/retrieve round-trip, cumulative artifact snapshot building
- `internal/pipeline/fork_test.go`: fork state preparation, artifact copying, delegation to resume, error cases (missing run, running run, invalid step)
- State store: checkpoint CRUD operations

### Integration tests

- Fork a completed mock run, verify new run starts from correct step with correct artifacts
- Rewind a failed run, verify state is reset, verify resume works after rewind
- Fork `--list` output for completed and partially-completed runs
- Error cases: fork a running run, fork from non-existent step, rewind to non-existent step

### Manual validation

- `go test ./...` passes
- `go test -race ./...` passes
- `golangci-lint run ./...` passes
