# Tasks

## Phase 1: Configuration & Types

- [X] Task 1.1: Add `Stacked bool` field to `MatrixStrategy` in `internal/pipeline/types.go`
- [X] Task 1.2: Add `BranchName string` field to `MatrixResult` in `internal/pipeline/matrix.go`
- [X] Task 1.3: Add `stacked` property to `MatrixStrategy` definition in `.wave/schemas/wave-pipeline.schema.json`
- [X] Task 1.4: Add `baseBranchOverride` field to `DefaultPipelineExecutor` in `internal/pipeline/executor.go`

## Phase 2: Core Implementation

- [X] Task 2.1: Modify `createStepWorkspace` in `executor.go` to use `baseBranchOverride` when set — if `baseBranchOverride` is non-empty and the step has `workspace.base` configured, substitute the override value
- [X] Task 2.2: Modify `childPipelineWorker` in `matrix.go` to accept a base branch parameter and set `baseBranchOverride` on the child executor before running the child pipeline
- [X] Task 2.3: Capture branch name from child pipeline execution — after `childExecutor.Execute()` completes, extract the first branch from the child's `WorktreePaths` and store it in `MatrixResult.BranchName`
- [X] Task 2.4: Implement stacked branch propagation in `tieredExecution` — after each tier completes, collect `BranchName` from successful `MatrixResult` entries and build an `itemID → branchName` mapping
- [X] Task 2.5: Implement single-parent base resolution — when a tier N+1 item has exactly one parent dependency, look up the parent's branch in the mapping and pass it to the child pipeline worker as the base branch
- [X] Task 2.6: Implement multi-parent merge — when a tier N+1 item has multiple parent dependencies, create a temporary integration branch by merging all parent branches using `git merge`

## Phase 3: Testing

- [X] Task 3.1: Add unit test for `Stacked` field parsing in `MatrixStrategy` [P]
- [X] Task 3.2: Add unit test for `baseBranchOverride` behavior in `createStepWorkspace` [P]
- [X] Task 3.3: Add unit test for stacked single-parent tier execution in `tieredExecution`
- [X] Task 3.4: Add unit test for stacked multi-parent merge in `tieredExecution`
- [X] Task 3.5: Add unit test verifying default behavior unchanged when `Stacked` is false
- [X] Task 3.6: Add unit test for failure propagation with stacking enabled

## Phase 4: Pipeline Config & Polish

- [X] Task 4.1: Add `stacked: true` to `gh-implement-epic.yaml` implement-subissues strategy
- [X] Task 4.2: Run `go test ./...` and `go test -race ./...` to verify no regressions
- [X] Task 4.3: Run `golangci-lint run ./...` for static analysis
