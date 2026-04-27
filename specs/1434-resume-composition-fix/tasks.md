# Work Items

## Phase 1: Setup

- [X] Item 1.1: Create feature branch `1434-resume-composition-fix` from current `main`.
- [X] Item 1.2: Confirm `internal/state/store.go` `RegisterArtifact` and `GetArtifacts` signatures (already verified: `RegisterArtifact(runID, stepID, name, path, type, sizeBytes)`).
- [X] Item 1.3: Confirm executor option pattern at `internal/pipeline/executor.go` (e.g. `WithRunID`, `WithDebug`).

## Phase 2: Core Implementation

### Bug 1 — Artifact registration in DAG executor

- [X] Item 2.1: In `internal/pipeline/executor.go` `executeAggregateInDAG` (~L5950, after the `outputPath` write succeeds), call `e.store.RegisterArtifact(execution.Status.ID, step.ID, artifactName, outputPath, "json", size)` guarded by `if e.store != nil`. `artifactName` is already derived at L5936. [P]
- [X] Item 2.2: In `internal/pipeline/executor.go` `collectIterateOutputs` (~L5890), after writing `<stepID>-collected.json`, call `e.store.RegisterArtifact(execution.Status.ID, step.ID, "collected-output", outputPath, "json", size)`. [P]

### Bug 1 — Resume artifact recovery via DB

- [X] Item 2.3: In `internal/pipeline/resume.go` `loadResumeState`, after the existing `for _, step := range p.Steps` loop, when `resolvedRunID != ""` and `r.executor.store != nil`, call `r.executor.store.GetArtifacts(resolvedRunID, "")` and merge each record into `state.ArtifactPaths` keyed `record.StepID + ":" + record.Name`. Skip override if key already populated by workspace walk.

### Bug 1 — Parallel correctness in legacy composition.go

- [X] Item 2.4: In `internal/pipeline/composition.go`, add `runID string` field on `CompositionExecutor` and a `WithRunID(string)` setter (or constructor parameter — pick whichever fits the existing `NewCompositionExecutor` signature). [P]
- [X] Item 2.5: In `executeAggregate` (~L477), call `c.store.RegisterArtifact(c.runID, step.ID, artifactName, outputPath, "json", size)` guarded by `c.store != nil && c.runID != ""`. [P]
- [X] Item 2.6: In `collectIterateOutputs` (~L319), if a future caller writes the array to disk, register `collected-output`. (Currently writes to template context only — defer if no on-disk path; document the gap with a TODO referencing the executor.go path.) [P]

### Bug 2 — Workspace path preservation

- [X] Item 2.7: In `internal/pipeline/executor.go`, add `workspaceOverride string` field on `DefaultPipelineExecutor` and `WithWorkspaceOverride(path string) ExecutorOption` returning `func(ex *DefaultPipelineExecutor) { ex.workspaceOverride = path }`.
- [X] Item 2.8: In `Execute()` (~L825-851), when `e.workspaceOverride != ""`, set `pipelineWsPath := e.workspaceOverride` and skip both the `os.RemoveAll(pipelineWsPath)` and `os.MkdirAll(pipelineWsPath, 0755)` calls (assume override exists from prior run).
- [X] Item 2.9: Apply the same gate at the second `pipelineWsPath` site in `executor.go` (~L1268-1272).
- [X] Item 2.10: In `cmd/wave/commands/resume.go` after `wsRoot` is resolved (~L201), compute `originalWsPath := filepath.Join(wsRoot, opts.RunID)` and append `pipeline.WithWorkspaceOverride(originalWsPath)` to `execOpts`.

## Phase 3: Testing

- [X] Item 3.1: Add `TestExecuteAggregateInDAG_RegistersArtifact` in `internal/pipeline/executor_test.go` — uses `MockStateStore` to capture `RegisterArtifact` calls; constructs a synthetic aggregate step; asserts call with `(stepID, "merged", path, "json", >0)`. [P]
- [X] Item 3.2: Add `TestCollectIterateOutputs_RegistersCollectedOutput` in `internal/pipeline/executor_test.go` — same shape, asserts `name == "collected-output"`. [P]
- [X] Item 3.3: Add `TestLoadResumeState_MergesDBArtifacts` in `internal/pipeline/resume_test.go` — pre-populate store with `RegisterArtifact`; assert `state.ArtifactPaths` is populated. [P]
- [X] Item 3.4: Add `TestWithWorkspaceOverride_PreservesPath` in `internal/pipeline/executor_test.go` — pre-create a marker file in the override dir; run `Execute` with a no-op pipeline; assert marker still present. [P]
- [X] Item 3.5: Add `TestResume_CompositionPipeline_E2E` in `internal/pipeline/resume_test.go` — stages original workspace + DB artifact records; calls `ResumeFromStep` with new run ID + `WithWorkspaceOverride`; asserts the resumed step's `inject_artifacts` lookup finds the aggregate output. (Larger integration test — covers acceptance criteria.)
- [X] Item 3.6: Run `go test -race ./internal/pipeline/...` to confirm no regressions in existing resume/executor/composition tests.

## Phase 4: Polish

- [X] Item 4.1: Run `go vet ./...` and `golangci-lint run ./...` — fix any new warnings.
- [X] Item 4.2: Run `go test ./...` (full suite).
- [X] Item 4.3: Verify manually: build binary (`go build -o /tmp/wave ./cmd/wave`), trigger a small composition pipeline (e.g. one with a single `aggregate` step), kill mid-flight, `wave resume <run-id>`, confirm aggregate output is found.
- [X] Item 4.4: Update `docs/guides/state-resumption.md` if it documents composition-pipeline resume behaviour. Otherwise add a brief note that aggregate/iterate outputs are now recoverable via DB.
- [X] Item 4.5: Self-review diff against acceptance criteria in `spec.md` before opening PR.
