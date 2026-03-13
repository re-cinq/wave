# Tasks

## Phase 1: Core Filter Type

- [X] Task 1.1: Create `internal/pipeline/stepfilter.go` with `StepFilter` struct
  - Fields: `Include []string`, `Exclude []string`
  - `IsActive() bool` — returns true if either include or exclude is non-empty
  - `Validate() error` — checks mutual exclusivity of include/exclude
  - `ValidateStepNames(steps []Step) error` — checks all named steps exist in pipeline
  - `Apply(sorted []*Step) []*Step` — filters the topologically-sorted step list
  - `ValidateWithFromStep(fromStep string) error` — rejects `--from-step` + `--steps`

- [X] Task 1.2: Create `internal/pipeline/stepfilter_test.go` with comprehensive unit tests
  - Test include filter: single, multiple, all steps
  - Test exclude filter: single, multiple steps
  - Test mutual exclusivity validation
  - Test invalid step name detection
  - Test `--from-step` + `--steps` rejection
  - Test `--from-step` + `-x` acceptance
  - Test empty/nil filter (no-op)
  - Test Apply preserves topological order

## Phase 2: CLI Integration

- [X] Task 2.1: Add `Steps` and `Exclude` fields to `RunOptions` in `cmd/wave/commands/run.go` [P]
  - Add `Steps []string` and `Exclude []string` to `RunOptions`
  - Register `--steps` flag as `StringSliceVar`
  - Register `-x`/`--exclude` flag as `StringSliceVar` with short form

- [X] Task 2.2: Add flag validation in `runRun()` [P]
  - Validate `--steps` and `-x` are mutually exclusive
  - Validate `--from-step` + `--steps` is invalid
  - Allow `--from-step` + `-x` combination
  - Produce clear error messages for all invalid combinations

## Phase 3: Executor Integration

- [X] Task 3.1: Add `WithStepFilter` executor option in `internal/pipeline/executor.go`
  - Add `stepFilter *StepFilter` field to `DefaultPipelineExecutor`
  - Create `WithStepFilter(f *StepFilter) ExecutorOption`
  - In `Execute()`: after `TopologicalSort`, call `stepFilter.Apply()` on the sorted steps
  - Validate step names against the pipeline before filtering

- [X] Task 3.2: Wire filter from CLI to executor in `cmd/wave/commands/run.go`
  - Build `StepFilter` from `RunOptions.Steps` and `RunOptions.Exclude`
  - Pass to executor via `WithStepFilter()` option
  - Only create filter when flags are non-empty

- [X] Task 3.3: Integrate filter with resume path in `internal/pipeline/resume.go`
  - Pass `StepFilter` through to `executeResumedPipeline`
  - Apply filter to the sorted steps in the resumed execution loop
  - This enables `--from-step` + `-x` combination

## Phase 4: Dry-Run Enhancement

- [X] Task 4.1: Enhance `performDryRun` to show filter status
  - Accept `StepFilter` parameter
  - Show `[SKIP]` / `[RUN]` annotations per step when filter is active
  - Show summary: "X of Y steps will execute"
  - Warn about skipped steps that produce artifacts needed by later steps

## Phase 5: Testing

- [X] Task 5.1: Write executor integration tests for step filtering [P]
  - Test `Execute()` with include filter — verify only named steps run via event collector
  - Test `Execute()` with exclude filter — verify excluded steps skipped
  - Test with mock adapter to verify step execution order
  - Test filter with concurrent-ready steps (batch execution)

- [X] Task 5.2: Write resume integration tests for `-x` + `--from-step` [P]
  - Test `ResumeWithValidation` with exclude filter
  - Verify excluded steps are skipped in resumed execution
  - Verify artifact injection still works for non-excluded steps

- [X] Task 5.3: Run full test suite and fix any regressions
  - `go test ./...`
  - `go test -race ./...`
  - Verify existing `--from-step` tests still pass unchanged
