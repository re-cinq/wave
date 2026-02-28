# Tasks

## Phase 1: Foundation — Onboarding State and Types

- [X] Task 1.1: Create `internal/onboarding/state.go` with `State` struct, `IsOnboarded()`, `MarkOnboarded()`, `ClearOnboarding()`, and `ReadState()` functions using `.wave/.onboarded` JSON marker file
- [X] Task 1.2: Create `internal/onboarding/state_test.go` with table-driven tests for state persistence (temp dirs, missing file, corrupt file, read/write round-trip)
- [X] Task 1.3: Create `internal/onboarding/onboarding.go` with `WizardConfig`, `WizardResult`, `WizardStep` interface, `StepResult` type, and `RunWizard()` orchestrator skeleton
- [X] Task 1.4: Add `Category string` field to `PipelineMetadata` in `internal/pipeline/types.go` (yaml tag: `category,omitempty`)

## Phase 2: Core Wizard Steps

- [X] Task 2.1: Implement `DependencyStep` in `internal/onboarding/steps.go` — uses `exec.LookPath` to check for adapter binaries and `gh` CLI, reports missing tools with install URLs [P]
- [X] Task 2.2: Implement `TestConfigStep` in `internal/onboarding/steps.go` — calls `detectProject()` logic, presents heuristic defaults in `huh.Input` fields for confirm/override [P]
- [X] Task 2.3: Implement `PipelineSelectionStep` in `internal/onboarding/steps.go` — discovers pipelines via `tui.DiscoverPipelines()`, presents `huh.MultiSelect` grouped by category and `metadata.release` [P]
- [X] Task 2.4: Implement `AdapterConfigStep` in `internal/onboarding/steps.go` — `huh.Select` for adapter choice, runs verification command to confirm binary existence [P]
- [X] Task 2.5: Implement `ModelSelectionStep` in `internal/onboarding/steps.go` — `huh.Select` for model options based on chosen adapter [P]
- [X] Task 2.6: Complete `RunWizard()` orchestrator — wire all steps, build manifest from results, handle `--yes`/non-interactive mode (apply defaults without prompts)

## Phase 3: Command Integration

- [X] Task 3.1: Add `checkOnboarding()` helper to `cmd/wave/commands/helpers.go` — checks `.wave/.onboarded` marker, returns descriptive error if missing; grandfathers existing projects (has `wave.yaml` but no marker)
- [X] Task 3.2: Integrate onboarding gate into `runRun()` in `cmd/wave/commands/run.go` — call `checkOnboarding()` before pipeline load; skip when `--force` is set
- [X] Task 3.3: Integrate onboarding gate into `runDo()` in `cmd/wave/commands/do.go` — call `checkOnboarding()` before execution
- [X] Task 3.4: Refactor `cmd/wave/commands/init.go` to call `RunWizard()` when interactive; add `--reconfigure` flag that clears onboarding state and re-runs wizard with existing manifest values as defaults
- [X] Task 3.5: Ensure existing `--yes`, `--force`, `--merge`, `--all` flags continue to work as before — `--yes` path uses non-interactive wizard mode

## Phase 4: Testing

- [X] Task 4.1: Write unit tests for `DependencyStep`, `TestConfigStep`, `AdapterConfigStep`, `ModelSelectionStep` in `internal/onboarding/steps_test.go` — mock `exec.LookPath` and filesystem [P]
- [X] Task 4.2: Write unit tests for `PipelineSelectionStep` in `internal/onboarding/steps_test.go` — test category grouping, release filtering [P]
- [X] Task 4.3: Write unit tests for `RunWizard()` orchestrator in `internal/onboarding/onboarding_test.go` — test non-interactive mode, reconfigure mode, step error handling [P]
- [X] Task 4.4: Write integration tests for onboarding gating in `cmd/wave/commands/run_test.go` — test blocked execution, `--force` bypass, grandfathered projects
- [X] Task 4.5: Write integration tests for `wave init` with wizard in `cmd/wave/commands/init_test.go` — test `--yes` produces valid manifest, `--reconfigure` preserves settings
- [X] Task 4.6: Run full test suite `go test ./...` and fix any regressions

## Phase 5: Polish

- [X] Task 5.1: Add progress indicator to wizard (step X of 5) using `huh.Group` titles or a custom header
- [X] Task 5.2: Ensure `wave init --reconfigure` pre-fills all fields from existing `wave.yaml`
- [X] Task 5.3: Verify edge cases — no TTY, corrupt state file, missing adapter binary, failed auth verification
- [X] Task 5.4: Run `go test -race ./...` to verify no race conditions
