# Tasks

## Phase 1: Step Filter Module

- [X] Task 1.1: Create `internal/pipeline/stepfilter.go` with `StepFilterConfig` type (Include/Exclude string slices) and `ApplyStepFilter(steps []*Step, config StepFilterConfig) ([]*Step, []string, error)` that returns filtered steps, skipped step IDs, and any validation errors
- [X] Task 1.2: Implement `ValidateStepNames(names []string, pipeline *Pipeline) error` to validate step names against the pipeline's available steps, producing clear error messages listing available steps
- [X] Task 1.3: Implement `ValidateFilterCombination(config StepFilterConfig, fromStep string) error` to enforce mutual exclusivity rules: `--steps` + `-x` = error, `--from-step` + `--steps` = error, `--from-step` + `-x` = ok
- [X] Task 1.4: Implement artifact dependency validation in filter — when a step is skipped, check if any remaining step depends on it via `inject_artifacts` and no prior workspace artifacts exist

## Phase 2: CLI Flag Registration

- [X] Task 2.1: Add `Steps string` and `Exclude string` fields to `RunOptions` struct in `cmd/wave/commands/run.go`
- [X] Task 2.2: Register `--steps` flag (`StringVar`) and `-x`/`--exclude` flag (`StringVarP`) on the Cobra command in `NewRunCmd()`
- [X] Task 2.3: Add flag combination validation in the `RunE` handler before `runRun()` — call `ValidateFilterCombination` with parsed values and `FromStep`

## Phase 3: Executor Integration

- [X] Task 3.1: Add `WithStepFilter(config StepFilterConfig)` executor option and `stepFilter StepFilterConfig` field on `DefaultPipelineExecutor`
- [X] Task 3.2: In `Execute()`, after `TopologicalSort()` and before the execution loop, apply `ApplyStepFilter()` to the sorted steps. Mark skipped steps as `StateSkipped` in execution state. Emit skip events for filtered-out steps
- [X] Task 3.3: In `ResumeFromStep()` / `executeResumedPipeline()`, apply the exclude filter (if set) after topological sort of the subpipeline. The `--steps` filter is not applicable here (already validated as incompatible with `--from-step`)
- [X] Task 3.4: Pass `StepFilterConfig` from `runRun()` into the executor via `WithStepFilter()` — parse comma-separated strings into slices, trim whitespace, filter empty strings

## Phase 4: Dry-Run Enhancement

- [X] Task 4.1: Update `performDryRun()` to accept a `StepFilterConfig` parameter. For each step, show `[RUN]`, `[SKIP]`, or `[EXCLUDE]` status based on the filter
- [X] Task 4.2: In dry-run output, if a step is skipped and downstream steps depend on its artifacts, show an artifact availability warning (check if prior workspace artifacts exist on disk)

## Phase 5: Testing

- [X] Task 5.1: Write unit tests in `internal/pipeline/stepfilter_test.go` — table-driven tests for `ApplyStepFilter`, `ValidateStepNames`, `ValidateFilterCombination`, artifact dependency validation [P]
- [X] Task 5.2: Write integration tests in `cmd/wave/commands/run_test.go` — flag existence tests for `--steps` and `-x`/`--exclude`, flag combination validation [P]
- [X] Task 5.3: Write executor integration tests — `Execute()` with `WithStepFilter` include filter, exclude filter, empty filter (no change), all-excluded error [P]
- [X] Task 5.4: Write resume integration test — `ResumeWithValidation()` with exclude filter applied correctly [P]
- [X] Task 5.5: Run `go test ./...` and `go test -race ./...` to verify no regressions

## Phase 6: Polish

- [X] Task 6.1: Update command examples in `NewRunCmd` Long/Example strings to include `--steps` and `-x` usage
- [X] Task 6.2: Final validation — verify all 10 acceptance criteria from the issue are satisfied
