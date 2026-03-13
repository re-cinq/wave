# Tasks

## Phase 1: Core Filter Type

- [X] Task 1.1: Create `internal/pipeline/step_filter.go` with `StepFilter` struct containing `Include []string` and `Exclude []string` fields
- [X] Task 1.2: Implement `StepFilter.Validate(pipeline *Pipeline) error` — checks step names exist, ensures mutual exclusivity of Include/Exclude
- [X] Task 1.3: Implement `StepFilter.ValidateCombinations(fromStep string) error` — rejects `--from-step` + `--steps` combo, allows `--from-step` + `-x`
- [X] Task 1.4: Implement `StepFilter.Apply(steps []*Step) []*Step` — returns filtered step list based on Include or Exclude
- [X] Task 1.5: Implement `StepFilter.ValidateDependencies(filtered []*Step, pipeline *Pipeline) error` — checks that filtered steps have all required dependencies either in the filtered set or available as prior artifacts

## Phase 2: CLI Flag Integration

- [X] Task 2.1: Add `Steps string` and `Exclude string` fields to `RunOptions` struct in `cmd/wave/commands/run.go` [P]
- [X] Task 2.2: Register `--steps` flag (`StringVar`) and `-x`/`--exclude` flag (`StringVarP`) on the run command [P]
- [X] Task 2.3: Add flag combination validation in `runRun()` before pipeline execution — reject `--steps` + `-x`, reject `--from-step` + `--steps`
- [X] Task 2.4: Parse comma-separated step names and construct `StepFilter`, pass to executor via new `WithStepFilter()` option

## Phase 3: Executor Integration

- [X] Task 3.1: Add `stepFilter *StepFilter` field to `DefaultPipelineExecutor` and `WithStepFilter()` option constructor
- [X] Task 3.2: Integrate filter into `Execute()` — after `TopologicalSort()`, call `stepFilter.Validate()` then `stepFilter.Apply()` to get filtered step list
- [X] Task 3.3: Integrate filter into `ResumeFromStep()` in `resume.go` — apply exclusion filter after creating resume subpipeline (for `--from-step` + `-x` combo)
- [X] Task 3.4: Handle artifact injection for filtered execution — when a step's dependency was filtered out, look for existing workspace artifacts using `loadResumeState()` logic

## Phase 4: Dry-Run Enhancement

- [X] Task 4.1: Pass `StepFilter` to `performDryRun()` function
- [X] Task 4.2: Show `[SKIP]` or `[RUN]` status for each step when a filter is active
- [X] Task 4.3: Show artifact availability warnings for skipped steps with downstream dependencies

## Phase 5: Testing

- [X] Task 5.1: Write unit tests for `StepFilter.Validate()` — invalid names, mutual exclusivity [P]
- [X] Task 5.2: Write unit tests for `StepFilter.Apply()` — include filter, exclude filter, no-op cases [P]
- [X] Task 5.3: Write unit tests for `StepFilter.ValidateCombinations()` — all flag combos [P]
- [X] Task 5.4: Write unit tests for `StepFilter.ValidateDependencies()` — missing deps, satisfied deps [P]
- [X] Task 5.5: Write integration tests for CLI flag parsing and validation in `cmd/wave/commands/run_test.go`
- [X] Task 5.6: Write executor integration test — `Execute()` with filter, verify correct steps run [P]
- [X] Task 5.7: Write resume integration test — `ResumeFromStep()` + exclusion filter [P]
- [X] Task 5.8: Verify all existing tests pass with `go test -race ./...`

## Phase 6: Polish

- [X] Task 6.1: Update `wave run --help` examples to show new flags
- [X] Task 6.2: Ensure error messages include available step names for discoverability
- [X] Task 6.3: Final validation — run `go vet ./...` and `golangci-lint run ./...`
