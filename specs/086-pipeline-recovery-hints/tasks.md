# Task Breakdown: Pipeline Recovery Hints on Failure

**Feature Branch**: `086-pipeline-recovery-hints`
**Generated**: 2026-02-13
**Total Tasks**: 15
**Spec**: `specs/086-pipeline-recovery-hints/spec.md`
**Plan**: `specs/086-pipeline-recovery-hints/plan.md`

## Phase 1: Setup — StepError Type & Recovery Package Scaffold

- [ ] T001 [P1] Define `StepError` struct in `internal/pipeline/errors.go` — new file with `StepError{StepID string, Err error}` implementing `error` and `Unwrap()` interfaces; produces same message as current `fmt.Errorf("step %q failed: %w", ...)` pattern
- [ ] T002 [P1] Replace `fmt.Errorf` with `StepError` at `internal/pipeline/executor.go:271` and `:1333` — change both error wrapping sites to `return &StepError{StepID: step.ID, Err: err}` so the step ID is programmatically extractable via `errors.As()`

## Phase 2: Foundational — Core Types, Shell Escaping, Error Classification

- [ ] T003 [P] [P1] Create `internal/recovery/recovery.go` — define `HintType` constants (`resume`, `force`, `workspace`, `debug`), `ErrorClass` constants (`contract_validation`, `security_violation`, `runtime_error`, `unknown`), `RecoveryHint` struct (`Label`, `Command`, `Type`), and `RecoveryBlock` struct (`PipelineName`, `StepID`, `Input`, `WorkspacePath`, `ErrorClass`, `Hints`)
- [ ] T004 [P] [P1] Create `internal/recovery/shell.go` — implement `ShellEscape(s string) string` using POSIX single-quote wrapping with interior single-quote escaping (`'it'\''s'` style); handle empty strings, strings with no special chars, and strings containing single quotes
- [ ] T005 [P] [P1] Create `internal/recovery/shell_test.go` — table-driven tests for `ShellEscape()` covering: empty string, simple string, string with spaces, single quotes, double quotes, ampersands, semicolons, backticks, dollar signs, glob characters, newlines
- [ ] T006 [P1] Create `internal/recovery/classify.go` — implement `ClassifyError(err error) ErrorClass` using `errors.As()`: `*contract.ValidationError` → `ClassContractValidation`, `*security.SecurityValidationError` → `ClassSecurityViolation`, non-empty message → `ClassRuntimeError`, empty/generic → `ClassUnknown`
- [ ] T007 [P1] Create `internal/recovery/classify_test.go` — table-driven tests for `ClassifyError()` covering: wrapped `contract.ValidationError`, wrapped `security.SecurityValidationError`, generic error with message, bare `errors.New("")`, nil error, multi-wrapped error chains

## Phase 3: US1+US2 — Resume & Force Hints (P1)

- [ ] T008 [P1] Implement `BuildRecoveryBlock()` in `internal/recovery/recovery.go` — function signature: `BuildRecoveryBlock(pipelineName, input, stepID, runID string, errClass ErrorClass) *RecoveryBlock`; always adds resume hint (`wave run <pipeline> <escaped-input> --from-step <step>`); adds force hint only when `errClass == ClassContractValidation` (`wave run <pipeline> <escaped-input> --from-step <step> --force` with label "Resume and skip validation checks"); always adds workspace hint (`ls .wave/workspaces/<runID>/<stepID>/`); adds debug hint when `errClass == ClassRuntimeError || ClassUnknown`; omit input arg when empty
- [ ] T009 [P1] Create `internal/recovery/recovery_test.go` — table-driven tests for `BuildRecoveryBlock()` covering: runtime error (resume+workspace+debug hints), contract validation error (resume+force+workspace hints, no debug), security error (resume+workspace hints only), unknown error (resume+workspace+debug), empty input (no input in commands), special characters in input (shell-escaped), single-step pipeline

## Phase 4: US3 — Workspace Path & Text Formatting (P2)

- [ ] T010 [P2] Create `internal/recovery/format.go` — implement `FormatRecoveryBlock(block *RecoveryBlock) string` rendering a labeled recovery block to stderr: header "Recovery options:", each hint as "  <label>:\n    <command>", total ≤ 8 lines excluding blank separators; add leading blank line separator
- [ ] T011 [P2] Create `internal/recovery/format_test.go` — tests for `FormatRecoveryBlock()` covering: block with all 4 hint types, block with resume-only, verify ≤ 8 content lines, verify header present, verify proper indentation

## Phase 5: US4 + JSON Mode — Debug Hints & Event Extension (P3)

- [ ] T012 [P] [P3] Add `RecoveryHintJSON` struct and `RecoveryHints` field to `internal/event/emitter.go` — define `RecoveryHintJSON struct { Label string; Command string; Type string }` with JSON tags; add `RecoveryHints []RecoveryHintJSON \`json:"recovery_hints,omitempty"\`` field to `Event` struct
- [ ] T013 [P] [P1] Refactor `isContractValidationError()` in `internal/pipeline/executor_enhanced.go:210-230` — replace string-matching implementation with `errors.As(err, &contract.ValidationError{})` (use pointer: `var ve *contract.ValidationError; errors.As(err, &ve)`)

## Phase 6: Integration — Wire into CLI Error Path

- [ ] T014 [P1] Integrate recovery hints into `cmd/wave/commands/run.go` — in `runRun()` after `execErr != nil` (line 245): extract `StepError` via `errors.As()`; call `recovery.ClassifyError()` on underlying error; call `recovery.BuildRecoveryBlock()` with pipeline name, input, step ID, run ID, error class; for text/auto/quiet modes: `fmt.Fprintf(os.Stderr, recovery.FormatRecoveryBlock(...))` after the error; for JSON mode: convert hints to `event.RecoveryHintJSON` slice, emit failure event with `RecoveryHints` populated; import `internal/recovery` package; always return the original wrapped error after printing hints

## Phase 7: Polish & Cross-Cutting

- [ ] T015 [P1] Run full test suite `go test ./...` and fix any compilation errors or test regressions — ensure all new files compile, all existing tests pass, verify with `-race` flag

## Dependency Graph

```
T001 → T002 (StepError must exist before executor uses it)
T003 ─┐
T004 ─┤ (parallel: independent package scaffolding)
T005 ─┘
T003 → T006 → T007 (classify depends on types)
T003 + T004 + T006 → T008 → T009 (builder depends on types, shell, classify)
T008 → T010 → T011 (formatter depends on builder types)
T012 ─┐ (parallel: independent Event change)
T013 ─┘ (parallel: independent refactor)
T002 + T008 + T010 + T012 + T013 → T014 (integration needs all pieces)
T014 → T015 (test suite after integration)
```

## Notes

- Tasks T003, T004, T005 are parallelizable (independent files in the new package)
- Tasks T012, T013 are parallelizable (independent modifications to different packages)
- T008 is the critical path task — it implements the core hint generation logic for US1+US2
- T014 is the integration task that wires everything together in the CLI layer
- The `--force` hint label must say "Resume and skip validation checks" (not just "contract validation") per C5
- Shell escaping uses POSIX single-quote style per C2 — no external dependency
