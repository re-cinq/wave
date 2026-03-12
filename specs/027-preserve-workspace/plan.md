# Implementation Plan: --preserve-workspace flag

## Objective

Add a `--preserve-workspace` CLI flag to `wave run` that skips the `os.RemoveAll` workspace cleanup at pipeline start, allowing developers to inspect intermediate artifacts from prior runs when debugging failed pipelines.

## Approach

Thread a boolean flag from Cobra CLI registration through `RunOptions` → `ExecutorOption` → `DefaultPipelineExecutor`, where the `Execute()` method gates the `os.RemoveAll(pipelineWsPath)` call on the flag value. Emit a stderr warning when the flag is active.

## File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `cmd/wave/commands/run.go` | modify | Add `PreserveWorkspace` to `RunOptions`, register `--preserve-workspace` Cobra flag, pass `WithPreserveWorkspace()` option to executor |
| `internal/pipeline/executor.go` | modify | Add `preserveWorkspace` field to `DefaultPipelineExecutor`, add `WithPreserveWorkspace()` option func, gate `os.RemoveAll` in `Execute()` on the field, emit warning event |
| `internal/pipeline/executor_test.go` | modify | Add tests for workspace preservation: verify directory survives when flag set, verify cleanup when flag unset, verify warning emission |

## Architecture Decisions

1. **ExecutorOption pattern**: Follow the existing `WithDebug`, `WithModelOverride`, etc. pattern. A new `WithPreserveWorkspace(bool)` option sets a field on `DefaultPipelineExecutor`. This is consistent with how all other CLI flags reach the executor.

2. **Warning via stderr, not event**: The warning should be printed to stderr (like the `--from-step` input recovery message on line 214 of run.go) for immediate developer visibility. Additionally emit a `"warning"` event so structured output consumers also see it.

3. **No workspace manager changes**: The `os.RemoveAll` call in `executor.go:309` is inline in the `Execute()` method — it doesn't go through `WorkspaceManager.CleanAll()`. The fix is localized to the executor. The `workspace.go` package needs no changes.

4. **Resume path unaffected**: `ResumeWithValidation` → `ResumeFromStep` creates a subpipeline and calls `Execute()` on it, which would clean workspaces. The `--preserve-workspace` flag naturally flows through since it's set on the executor instance, so the resume path also respects it.

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Stale artifacts cause confusing failures | Medium | Clear stderr warning when flag is active |
| Users forget flag is set in scripts | Low | Warning is unconditional; flag has no env-var equivalent |
| Interaction with worktree workspaces | Low | Worktree workspaces have separate lifecycle management (not affected by the pipeline-level `os.RemoveAll`) |

## Testing Strategy

1. **Unit test: flag registration** — Verify `--preserve-workspace` flag exists on the `run` command and defaults to `false`
2. **Unit test: workspace preserved** — Create a temp workspace directory with a marker file, run `Execute()` with `WithPreserveWorkspace(true)`, verify marker file survives
3. **Unit test: workspace cleaned without flag** — Same setup but without the option, verify marker file is removed
4. **Unit test: warning event emitted** — Use a capturing event emitter to verify a warning event is emitted when the flag is set
