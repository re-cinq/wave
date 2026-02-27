# Tasks

## Phase 1: Schema & Types
- [X] Task 1.1: Add `MaxConcurrentAgents int` field with `yaml:"max_concurrent_agents,omitempty"` to `Step` struct in `internal/pipeline/types.go`
- [X] Task 1.2: Add `MaxConcurrentAgents int` field to `AdapterRunConfig` struct in `internal/adapter/adapter.go`

## Phase 2: Validation
- [X] Task 2.1: Add validation rule in `internal/pipeline/validation.go` — reject `MaxConcurrentAgents < 0` or `> 10`
- [X] Task 2.2: Add table-driven test cases in `internal/pipeline/validation_test.go` for bounds checking (values: -1, 0, 1, 5, 10, 11)

## Phase 3: Core Implementation
- [X] Task 3.1: In `internal/pipeline/executor.go` `runStepExecution()`, pass `step.MaxConcurrentAgents` to `AdapterRunConfig` when building the config (~line 632)
- [X] Task 3.2: In `internal/adapter/claude.go` `prepareWorkspace()`, add concurrency section between contract prompt and restrictions. Build a `buildConcurrencySection(cfg AdapterRunConfig) string` function that returns a markdown section when `cfg.MaxConcurrentAgents > 1`, empty string otherwise [P]
- [X] Task 3.3: Wire the concurrency section into CLAUDE.md assembly in `prepareWorkspace()`, inserted after `cfg.ContractPrompt` and before `buildRestrictionSection(cfg)`

## Phase 4: Testing
- [X] Task 4.1: Add `TestBuildConcurrencySection` unit test in `internal/adapter/claude_test.go` — table-driven test for the builder function with values 0, 1, 3, 10 [P]
- [X] Task 4.2: Add `TestCLAUDEMDConcurrencySection` test in `internal/adapter/claude_test.go` — following existing `TestCLAUDEMDRestrictionSection` pattern, verify CLAUDE.md contains/doesn't contain concurrency hint [P]
- [X] Task 4.3: Add `TestMaxConcurrentAgentsPropagation` test in `internal/pipeline/executor_test.go` — verify the field is passed through to `AdapterRunConfig` [P]
- [X] Task 4.4: Run full test suite with `go test ./...` and verify no regressions

## Phase 5: Polish
- [X] Task 5.1: Run `go test -race ./...` to verify no race conditions
- [X] Task 5.2: Final review — verify all 5 acceptance criteria are met
