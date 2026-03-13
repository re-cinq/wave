# Tasks

## Phase 1: Type Extensions

- [X] Task 1.1: Add `ReworkStep` field to `RetryConfig` in `internal/pipeline/types.go`
  - Add `ReworkStep string \`yaml:"rework_step,omitempty"\`` field
  - Add `StateReworking = "reworking"` constant alongside other state constants
- [X] Task 1.2: Extend `AttemptContext` in `internal/pipeline/types.go`
  - Add `ContractErrors []string` field for structured contract validation errors
  - Add `StepDuration time.Duration` field for how long the step ran
  - Add `PartialArtifacts map[string]string` field for partial artifact paths
  - Add `FailedStepID string` field for rework context
- [X] Task 1.3: Add `RetryConfig` validation method
  - Validate `rework_step` is non-empty when `on_failure` is `rework`
  - Return error if `rework_step` is set but `on_failure` is not `rework`

## Phase 2: DAG Validation

- [X] Task 2.1: Add rework target validation to `internal/pipeline/dag.go`
  - Verify `rework_step` references an existing step ID
  - Verify rework target is not a direct or transitive dependency of the failing step
  - Verify rework target does not create a cycle (rework target cannot have the failing step as a dependency)
- [X] Task 2.2: Integrate rework validation into `DAGValidator.ValidateDAG`
  - Call rework validation for each step that has `on_failure: rework`

## Phase 3: Core Executor Changes

- [X] Task 3.1: Add `case "rework"` to on_failure switch in `executor.go:executeStep`
  - Mark failed step as `StateFailed`
  - Build enhanced `AttemptContext` with failure details (error, stdout, duration, artifacts)
  - Look up rework target step from pipeline
  - Execute rework step via `runStepExecution`
  - If rework succeeds, re-register artifacts under original step ID
  - If rework fails, return rework step's error
- [X] Task 3.2: Implement artifact path replacement after rework success
  - Copy rework step's workspace path into `execution.WorkspacePaths` for failed step
  - Copy rework step's artifact paths into `execution.ArtifactPaths` for failed step
  - Emit event for rework completion with artifact mapping
- [X] Task 3.3: Emit rework-specific events
  - `StateReworking` when rework step starts
  - Completion event with rework context when rework succeeds
  - Include rework source step in event metadata

## Phase 4: Enhanced Failure Context

- [X] Task 4.1: Populate enhanced `AttemptContext` fields during retry
  - Set `StepDuration` from attempt timing in `executeStep`
  - Set `ContractErrors` when contract validation fails (parse contract error message)
  - Set `PartialArtifacts` by scanning workspace for output artifacts
  - Set `FailedStepID` when entering rework
- [X] Task 4.2: Inject enhanced failure context into rework step's prompt
  - Extend `buildStepPrompt` to include rework-specific context when `AttemptContexts` contains the step

## Phase 5: Resume Support

- [X] Task 5.1: Track rework transitions in resume state
  - Add `ReworkTransitions map[string]string` to `ResumeState` (failedStepID -> reworkStepID)
  - Record rework transitions when `on_failure: rework` triggers
- [ ] Task 5.2: Handle rework step completion during resume
  - When loading resume state, check if rework step completed for a failed step
  - If rework completed, register rework artifacts under original step ID
  - Skip rework step if already completed in prior run

## Phase 6: Schema Updates

- [X] Task 6.1: Update `wave-pipeline.schema.json` [P]
  - Add `RetryConfig` definition (currently missing from schema)
  - Include `on_failure` enum with `fail`, `skip`, `continue`, `rework`
  - Add `rework_step` string property
  - Add `retry` property reference to Step definition
  - Add `optional` boolean to Step definition (also currently missing)
- [X] Task 6.2: Add `StateReworking` to `internal/state/store.go` StepState constants [P]

## Phase 7: Testing

- [X] Task 7.1: Unit tests for `RetryConfig` validation (`internal/pipeline/retry_test.go`)
  - Test `rework_step` required when `on_failure: rework`
  - Test error when `rework_step` set without `on_failure: rework`
  - Test valid rework config passes validation
- [X] Task 7.2: Unit tests for DAG rework validation (`internal/pipeline/dag_test.go`) [P]
  - Test rework target exists
  - Test rework target not upstream dependency
  - Test rework target cycle detection
  - Test valid rework passes validation
- [X] Task 7.3: Unit tests for executor rework (`internal/pipeline/executor_test.go`) [P]
  - Test rework triggers after retry exhaustion
  - Test rework step executes with failure context
  - Test rework step artifacts replace failed step artifacts
  - Test rework step failure propagates correctly
  - Test existing on_failure behaviors unchanged (regression)
- [X] Task 7.4: Unit tests for enhanced `AttemptContext` [P]
  - Test `ContractErrors` populated on contract failure
  - Test `StepDuration` set correctly
  - Test `PartialArtifacts` detected from workspace
  - Test `FailedStepID` set during rework
- [ ] Task 7.5: Integration test for resume with rework [P]
  - Test resume correctly skips completed rework steps
  - Test resume carries rework failure context

## Phase 8: Documentation

- [X] Task 8.1: Update `docs/reference/pipeline-schema.md`
  - Document `on_failure: rework` behavior
  - Document `rework_step` field
  - Add example YAML snippet
- [ ] Task 8.2: Update `docs/guide/pipelines.md` if rework branching is relevant
