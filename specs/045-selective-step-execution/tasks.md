# Tasks

## Phase 1: Core Filter Type

- [X] Task 1.1: Create `internal/pipeline/filter.go` with `StepFilter` struct containing `Include []string`, `Exclude []string` fields and a `Mode()` method returning "include", "exclude", or "none"
- [X] Task 1.2: Implement `StepFilter.Validate(steps []*Step) error` — checks that all named steps exist in the pipeline, returns error listing available steps if any are invalid
- [X] Task 1.3: Implement `StepFilter.Apply(steps []*Step) ([]*Step, error)` — filters the topologically-sorted step list based on include/exclude mode
- [X] Task 1.4: Implement `StepFilter.ValidateDependencies(filtered []*Step, allSteps []*Step, artifactPaths map[string]string) error` — checks that filtered steps have their dependency artifacts available (either from another filtered step or from existing workspace artifacts)

## Phase 2: CLI Flag Integration

- [X] Task 2.1: Add `Steps []string` and `Exclude []string` fields to `RunOptions` struct in `cmd/wave/commands/run.go`
- [X] Task 2.2: Register `--steps` (`StringSliceVar`) and `-x`/`--exclude` (`StringSliceVar`) flags in `NewRunCmd()`
- [X] Task 2.3: Add mutual exclusivity validation in `RunE`: error if both `--steps` and `-x` are provided
- [X] Task 2.4: Add `--from-step` + `--steps` incompatibility check: error if both are provided
- [X] Task 2.5: Pass `StepFilter` through to executor via new `WithStepFilter(f StepFilter)` executor option

## Phase 3: Executor Integration

- [X] Task 3.1: Add `stepFilter StepFilter` field to `DefaultPipelineExecutor` and `WithStepFilter` option constructor
- [X] Task 3.2: In `Execute()`, apply filter after `TopologicalSort()` — call `stepFilter.Validate()` then `stepFilter.Apply()` on sorted steps, replacing `sortedSteps`
- [X] Task 3.3: Before execution loop, call `stepFilter.ValidateDependencies()` with filtered steps and any pre-existing artifact paths from workspace scanning
- [X] Task 3.4: Update `TotalSteps` in progress events to reflect filtered count (not full pipeline count)

## Phase 4: Resume + Exclude Integration

- [X] Task 4.1: In `ResumeManager.ResumeFromStep()`, propagate step filter from executor to the resumed pipeline execution
- [X] Task 4.2: In `executeResumedPipeline()`, apply exclusion filter to sorted steps after subpipeline creation
- [X] Task 4.3: Ensure excluded steps in resume mode still have their artifact paths loaded from workspace (reuse `loadResumeState` logic)

## Phase 5: Dry-Run Enhancement

- [X] Task 5.1: Update `performDryRun()` to accept `StepFilter` parameter
- [X] Task 5.2: When filter is active, show "[SKIP]" or "[RUN]" prefix for each step in the dry-run output
- [X] Task 5.3: For skipped steps, show artifact availability warnings if downstream steps need their outputs

## Phase 6: Unit Tests

- [X] Task 6.1: Write `filter_test.go` — `TestStepFilter_Validate` with valid/invalid step names [P]
- [X] Task 6.2: Write `filter_test.go` — `TestStepFilter_Apply_Include` with various include patterns [P]
- [X] Task 6.3: Write `filter_test.go` — `TestStepFilter_Apply_Exclude` with various exclude patterns [P]
- [X] Task 6.4: Write `filter_test.go` — `TestStepFilter_ValidateDependencies` with satisfied/unsatisfied deps [P]
- [X] Task 6.5: Write `filter_test.go` — `TestStepFilter_EmptyResult` error when all steps filtered out [P]
- [X] Task 6.6: Write `filter_test.go` — `TestStepFilter_MutualExclusivity` error when both include and exclude set [P]

## Phase 7: Integration Tests

- [X] Task 7.1: Write `run_filter_test.go` — `TestNewRunCmdFilterFlags` verifying `--steps` and `-x`/`--exclude` flag registration [P]
- [X] Task 7.2: Write `run_filter_test.go` — `TestRunStepsAndExcludeMutualExclusivity` error when both flags given [P]
- [X] Task 7.3: Write `run_filter_test.go` — `TestRunFromStepAndStepsIncompatibility` error when both flags given [P]
- [X] Task 7.4: Write `run_filter_test.go` — `TestRunDryRunWithFilter` dry-run output shows skip/run status [P]

## Phase 8: Validation

- [X] Task 8.1: Run `go test ./...` and verify all existing tests still pass
- [X] Task 8.2: Run `go test -race ./...` to check for race conditions
- [X] Task 8.3: Manual verification: `go build ./cmd/wave` succeeds
