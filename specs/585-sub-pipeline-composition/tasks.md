# Tasks

## Phase 1: Type Definitions and Validation

- [ ] Task 1.1: Add `SubPipelineConfig` struct to `internal/pipeline/types.go` with fields: `Inject []string`, `Extract []string`, `Timeout string`, `MaxCycles int`, `StopCondition string`
- [ ] Task 1.2: Add `Config *SubPipelineConfig` field to `Step` struct in `internal/pipeline/types.go` (near line 293, alongside `SubPipeline`)
- [ ] Task 1.3: Add `Validate()` method to `SubPipelineConfig` (timeout parsing, field consistency checks)
- [ ] Task 1.4: Add circular sub-pipeline detection function in `internal/pipeline/subpipeline.go` -- build directed graph of pipeline->sub-pipeline references, detect cycles via DFS
- [ ] Task 1.5: Update dry run validator in `internal/pipeline/dryrun.go` to validate new `Config` fields when step has `SubPipeline`

## Phase 2: State Nesting

- [ ] Task 2.1: Add `ParentRunID` and `ParentStepID` string fields to `RunRecord` in `internal/state/types.go`
- [ ] Task 2.2: Add `SetParentRun(childRunID, parentRunID, stepID string) error` and `GetChildRuns(parentRunID string) ([]RunRecord, error)` to `StateStore` interface in `internal/state/store.go`
- [ ] Task 2.3: Implement parent-child run linkage in SQLite store (`internal/state/sqlite.go`) -- add nullable `parent_run_id` and `parent_step_id` columns to runs table, implement new methods

## Phase 3: Core Implementation

- [ ] Task 3.1: Create `internal/pipeline/subpipeline.go` with `injectParentArtifacts()` function -- copies named artifacts from parent execution's `ArtifactPaths` into child workspace's `.wave/artifacts/` [P]
- [ ] Task 3.2: Add `extractChildArtifacts()` function to `subpipeline.go` -- after child completion, copies named artifacts from child execution back to parent's `ArtifactPaths` and registers in parent `PipelineContext` [P]
- [ ] Task 3.3: Add `MergeFrom(child *PipelineContext, namespace string)` method to `PipelineContext` in `internal/pipeline/context.go` -- merges child custom variables and artifact paths into parent, namespaced by child pipeline name [P]
- [ ] Task 3.4: Enhance `executeCompositionStep()` in `internal/pipeline/executor.go` (line 3951) -- when `step.Config` is non-nil: call `injectParentArtifacts()` before child execution, call `extractChildArtifacts()` + `MergeFrom()` after child completion
- [ ] Task 3.5: Add lifecycle enforcement in `executeCompositionStep()` -- wrap child executor context with `context.WithTimeout()` when `Config.Timeout` is set; propagate `Config.MaxCycles` to child pipeline's loop config; evaluate `Config.StopCondition` template after child completes
- [ ] Task 3.6: Wire parent-child state linkage -- after creating child run ID, call `store.SetParentRun(childRunID, parentRunID, stepID)`. Add `WithParentRunID(id string)` executor option to propagate parent run identity
- [ ] Task 3.7: Update `CompositionExecutor.executeSubPipeline()` in `composition.go` (line 438) and `runSubPipeline()` (line 447) to respect `SubPipelineConfig` when present -- apply timeout, pass artifact inject/extract config
- [ ] Task 3.8: Implement `workspace.ref: parent` support -- when child step's `WorkspaceConfig.Ref` is "parent", pass parent step's resolved workspace path to child executor

## Phase 4: Testing

- [ ] Task 4.1: Write unit tests for `SubPipelineConfig.Validate()` (invalid timeout, missing required fields, valid configs) [P]
- [ ] Task 4.2: Write unit tests for `injectParentArtifacts()` and `extractChildArtifacts()` (happy path, missing artifacts, optional artifacts) [P]
- [ ] Task 4.3: Write unit tests for `PipelineContext.MergeFrom()` (key overwrite, namespace isolation, concurrent safety) [P]
- [ ] Task 4.4: Write unit tests for lifecycle enforcement (timeout cancellation, max_cycles propagation, stop_condition evaluation) [P]
- [ ] Task 4.5: Write unit tests for circular pipeline detection (no cycle, simple A->B->A, transitive A->B->C->A)
- [ ] Task 4.6: Write integration test for end-to-end sub-pipeline execution with artifact inject/extract flow
- [ ] Task 4.7: Write integration test for state nesting (parent-child run linkage, GetChildRuns query)
- [ ] Task 4.8: Verify all existing composition tests pass unchanged (backward compatibility)

## Phase 5: Polish

- [ ] Task 5.1: Run `go test ./...` and fix any failures
- [ ] Task 5.2: Run `go vet ./...` and fix any warnings
- [ ] Task 5.3: Run `golangci-lint run ./...` and fix any issues
- [ ] Task 5.4: Final review -- ensure no regressions in existing sub-pipeline, gate, iterate, branch, loop, aggregate functionality
