# Tasks

## Phase 1: Type Definitions and Validation

- [X] Task 1.1: Add `SubPipelineConfig` struct to `internal/pipeline/types.go` with fields for artifact inject/extract, timeout, max_cycles, stop_condition
- [X] Task 1.2: Add `Config *SubPipelineConfig` field to `Step` struct in `internal/pipeline/types.go`
- [X] Task 1.3: Add `Validate()` method to `SubPipelineConfig` (timeout parsing, required field checks)
- [X] Task 1.4: Add circular sub-pipeline detection to `internal/pipeline/dag.go`
- [X] Task 1.5: Update dry run validator in `internal/pipeline/dryrun.go` to validate new config and artifacts fields

## Phase 2: State Nesting

- [X] Task 2.1: Add `ParentRunID` and `ParentStepID` fields to `RunRecord` in `internal/state/types.go`
- [X] Task 2.2: Add `SetParentRun(childRunID, parentRunID, stepID)` and `GetChildRuns(parentRunID)` to `StateStore` interface in `internal/state/store.go`
- [X] Task 2.3: Implement parent-child run linkage in SQLite store (`internal/state/store.go`) — add nullable columns, implement new methods

## Phase 3: Core Implementation

- [X] Task 3.1: Create `internal/pipeline/subpipeline.go` with artifact inject function — copies parent artifacts into child workspace [P]
- [X] Task 3.2: Add artifact extract function to `subpipeline.go` — copies child artifacts back to parent execution state [P]
- [X] Task 3.3: Add `MergeFrom()` method to `PipelineContext` in `internal/pipeline/context.go` for child->parent context merge [P]
- [X] Task 3.4: Enhance `executeCompositionStep()` in `internal/pipeline/executor.go` — when `step.Config` is set, apply artifact inject before child execution, artifact extract + context merge after
- [X] Task 3.5: Add lifecycle enforcement — wrap child executor context with `context.WithTimeout()`, propagate `max_cycles` to child loop config, evaluate `stop_condition` template
- [X] Task 3.6: Wire parent-child state linkage — call `SetParentRun()` after creating child run, pass parent run ID through executor options
- [X] Task 3.7: Update `CompositionExecutor.executeSubPipeline()` in `composition.go` to handle new config (timeout enforcement)
- [X] Task 3.8: Implement `workspace.ref: parent` support — workspace path passed to child executor via existing WorkspaceConfig.Ref

## Phase 4: Testing

- [X] Task 4.1: Write unit tests for `SubPipelineConfig.Validate()` (invalid timeout, missing fields, valid configs) [P]
- [X] Task 4.2: Write unit tests for artifact inject/extract functions (happy path, missing artifacts, optional artifacts) [P]
- [X] Task 4.3: Write unit tests for `PipelineContext.MergeFrom()` (key overwrite, namespace isolation) [P]
- [X] Task 4.4: Write unit tests for lifecycle enforcement (timeout, max_cycles, stop_condition) [P]
- [X] Task 4.5: Write integration tests for end-to-end sub-pipeline execution with artifact flow
- [X] Task 4.6: Write integration tests for state nesting (parent-child run linkage, GetChildRuns)
- [X] Task 4.7: Update existing composition tests to verify backward compatibility
- [X] Task 4.8: Update dry run tests for new config validation

## Phase 5: Polish

- [X] Task 5.1: Run `go test ./...` and fix any failures
- [X] Task 5.2: Run `go vet ./...` — clean
- [X] Task 5.3: All existing tests pass
