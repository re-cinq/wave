# Work Items

## Phase 1: Setup
- [X] Item 1.1: Confirm `e.store.RegisterArtifact` signature and the `event.Event` patterns used in `writeOutputArtifacts` (executor.go:4517-4523). Use as template.
- [X] Item 1.2: Confirm all `filepath.Join(wsRoot, pipelineID, ...)` call sites in `internal/pipeline/` (executor.go, concurrency.go, matrix.go) — produce a checklist before edits.

## Phase 2: Core Implementation
- [X] Item 2.1: Add `workspaceRunID string` field to `DefaultPipelineExecutor`; add `WithWorkspaceRunID` option; add `workspaceRunIDFor(pipelineID)` accessor (renamed from `effectiveWorkspaceRunID()` so fresh runs without `WithRunID` keep the existing `Status.ID`-based path).
- [X] Item 2.2: Patch `executeAggregateInDAG` to call `e.store.RegisterArtifact` after writing the output.
- [X] Item 2.3: Patch `collectIterateOutputs` to register the `<stepID>-collected.json` file as `collected-output` artifact.
- [X] Item 2.4: Switch step-workspace path computations at executor.go:3852, 3943, 6146 to `workspaceRunIDFor(pipelineID)`.
- [X] Item 2.5: Switch concurrency.go:204 and matrix.go:489 to `workspaceRunIDFor(pipelineID)`.
- [X] Item 2.6: Patch `composition.go` `executeAggregate` and `collectIterateOutputs` for parity (legacy path); plumb `runID` into `CompositionExecutor` via `SetRunID`. Registration uses an `artifactRegistrar` interface so the existing `state.RunStore`-typed field stays narrow.
- [X] Item 2.7: Update `cmd/wave/commands/resume.go` to pass `pipeline.WithWorkspaceRunID(opts.RunID)` to the executor.

## Phase 3: Testing
- [X] Item 3.1: `TestExecuteAggregateInDAG_RegistersArtifact` (executor_test.go) + `TestExecuteAggregateInDAG_NoStore` for nil-store ergonomics.
- [X] Item 3.2: `TestCollectIterateOutputs_RegistersArtifact` (executor_test.go).
- [X] Item 3.3: `TestWorkspaceRunIDFor` covers fallback + override (executor_test.go).
- [X] Item 3.4: `TestCreateStepWorkspace_UsesEffectiveWorkspaceRunID` verifies the resume override threads through to step-workspace path computation (executor_test.go).
- [ ] Item 3.5: `TestResume_AggregateStepResumesSuccessfully` deferred — full end-to-end resume harness requires substantial DB + filesystem fixture scaffolding beyond unit-level coverage. The path-resolution + artifact-registration unit tests above prove both fixes meet the acceptance criteria; manual regression on a real `ops-pr-respond` run is the remaining validation step.
- [X] Item 3.6: `TestCompositionExecutor_Aggregate_RegistersArtifact` + `TestCompositionExecutor_Aggregate_NoRegistrationWithoutRunID` (composition_test.go).
- [X] Item 3.7: `go test ./...` and `go test -race ./internal/pipeline/...` — all green.

## Phase 4: Polish
- [ ] Item 4.1: Manual regression on a real `ops-pr-respond` failure post-aggregate (deferred — out-of-scope for this worktree's environment; tracked for follow-up validation).
- [ ] Item 4.2: `golangci-lint run ./...` (binary not installed in this environment; `go vet ./...` passes).
- [ ] Item 4.3: Update CHANGELOG / release notes if the project keeps them; otherwise skip.
- [ ] Item 4.4: Open PR with `fix(pipeline):` prefix; reference #1434, #1401, #1412.
