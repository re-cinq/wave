# Implementation Plan: --preserve-workspace flag

## Objective

Add a `--preserve-workspace` boolean flag to the `wave run` command that skips the `os.RemoveAll` workspace cleanup at pipeline start, allowing developers to debug failed pipeline steps by inspecting preserved intermediate artifacts.

## Approach

Thread a `PreserveWorkspace` boolean through three layers:

1. **CLI flag** → `RunOptions` struct in `cmd/wave/commands/run.go`
2. **Executor option** → `DefaultPipelineExecutor` in `internal/pipeline/executor.go`
3. **Conditional cleanup** → gate the `os.RemoveAll` call on the new flag

The flag is purely additive — it only suppresses cleanup. No changes to workspace creation, artifact injection, or contract validation are needed. When active, a warning is emitted to stderr so users are aware that stale state may cause non-reproducible results.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `cmd/wave/commands/run.go` | modify | Add `PreserveWorkspace` field to `RunOptions`, register `--preserve-workspace` flag, pass to executor via `WithPreserveWorkspace` option |
| `internal/pipeline/executor.go` | modify | Add `preserveWorkspace` field to `DefaultPipelineExecutor`, add `WithPreserveWorkspace` option func, gate `os.RemoveAll` call on field value, emit warning event |
| `internal/pipeline/executor_test.go` | modify | Add test verifying workspace preservation when flag is set |

## Architecture Decisions

1. **Executor option pattern**: Follow the existing `ExecutorOption` functional option pattern (`WithDebug`, `WithModelOverride`, etc.) for consistency. No new interfaces or types needed.

2. **Warning via event emitter**: Use the existing `e.emit()` mechanism to emit a `"warning"` state event when preserve-workspace is active, consistent with how other warnings (e.g., failed cleanup) are already emitted.

3. **No workspace manager changes**: The `os.RemoveAll` call is directly in `executor.Execute()` (line ~318), not in the `WorkspaceManager` interface. The gate applies at the same level — no changes to the workspace package.

4. **Complementary with `--from-step`**: The flags operate independently — `--from-step` controls *which steps* run, `--preserve-workspace` controls *whether cleanup happens*. No special interaction logic is needed.

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Stale workspace state causes confusing failures | Medium | Warning message clearly communicates the risk to users |
| Flag forgotten in CI/automated contexts | Low | Flag defaults to `false`; only active when explicitly passed |
| Workspace path collision with new run ID | Low | The `os.MkdirAll` call after the gated `RemoveAll` still ensures the directory exists; existing contents are simply preserved |

## Testing Strategy

1. **Unit test in `executor_test.go`**: Create a workspace directory with test content, run executor with `WithPreserveWorkspace(true)`, verify the content is still present after execution starts. Verify that without the flag, the content is removed.

2. **Integration with existing tests**: Ensure `go test ./...` passes — no existing behavior changes when flag is not set (default `false`).

3. **Manual verification**: `wave run --preserve-workspace <pipeline>` should show the warning and preserve prior workspace contents.
