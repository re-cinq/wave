# Implementation Plan: Pipeline Recovery Hints on Failure

**Branch**: `086-pipeline-recovery-hints` | **Date**: 2026-02-13 | **Spec**: `specs/086-pipeline-recovery-hints/spec.md`
**Input**: Feature specification from `/specs/086-pipeline-recovery-hints/spec.md`

## Summary

Add contextual recovery hints to pipeline failure output. When a pipeline step fails, the CLI prints concise, copy-pasteable commands to stderr (or structured data in JSON mode) showing how to resume, skip validation, inspect the workspace, or enable debug output. Recovery hints are generated in the CLI command layer using error classification via `errors.As()` type assertions against existing error types (`contract.ValidationError`, `security.SecurityValidationError`).

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `github.com/spf13/cobra` (CLI), `gopkg.in/yaml.v3` (config) — no new dependencies
**Storage**: N/A (no persistence changes)
**Testing**: `go test ./...` with `-race` flag
**Target Platform**: Linux/macOS (single binary)
**Project Type**: Single Go binary
**Performance Goals**: Recovery hint generation adds < 1ms to error path (pure in-memory string formatting)
**Constraints**: Recovery block ≤ 8 lines output (FR-006); no additional I/O on error path (FR-010)
**Scale/Scope**: 1 new package (~150 LOC), 3 modified files, ~50 LOC of test code per error class

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|---|---|---|
| P1: Single Binary | ✅ Pass | No new dependencies; pure Go implementation |
| P2: Manifest as SSoT | ✅ Pass | No manifest changes |
| P3: Persona-Scoped Execution | ✅ Pass | No persona changes |
| P4: Fresh Memory at Boundaries | ✅ Pass | No change to step isolation |
| P5: Navigator-First | ✅ Pass | No pipeline structure changes |
| P6: Contracts at Handovers | ✅ Pass | Contract validation unchanged; recovery hints are a CLI-layer concern |
| P7: Relay via Summarizer | ✅ Pass | No relay changes |
| P8: Ephemeral Workspaces | ✅ Pass | Workspaces referenced (read-only path) not modified |
| P9: Credentials Never Touch Disk | ✅ Pass | No credential handling |
| P10: Observable Progress | ✅ Pass | Recovery hints enhance observability by adding structured `recovery_hints` field to failure events |
| P11: Bounded Recursion | ✅ Pass | No recursion changes |
| P12: Minimal State Machine | ✅ Pass | No new states; hints attach to existing `failed` state |
| P13: Test Ownership | ✅ Pass | New tests for all error classifications; existing tests unmodified |

## Project Structure

### Documentation (this feature)

```
specs/086-pipeline-recovery-hints/
├── plan.md              # This file
├── research.md          # Phase 0 research and decisions
├── data-model.md        # Entity definitions and relationships
└── tasks.md             # Task breakdown (created by /speckit.tasks)
```

### Source Code (repository root)

```
internal/
├── recovery/                    # NEW: Recovery hint generation
│   ├── recovery.go              # RecoveryHint, RecoveryBlock, BuildRecoveryBlock()
│   ├── classify.go              # ClassifyError() — errors.As() based classification
│   ├── format.go                # FormatRecoveryBlock() — stderr text rendering
│   ├── shell.go                 # ShellEscape() — POSIX single-quote escaping
│   ├── recovery_test.go         # Unit tests for BuildRecoveryBlock()
│   ├── classify_test.go         # Unit tests for ClassifyError()
│   ├── format_test.go           # Unit tests for FormatRecoveryBlock()
│   └── shell_test.go            # Unit tests for ShellEscape()
├── pipeline/
│   ├── errors.go                # NEW: StepError type definition
│   ├── executor.go              # MODIFIED: Use StepError instead of fmt.Errorf
│   └── executor_enhanced.go     # MODIFIED: Refactor isContractValidationError() to errors.As()
├── event/
│   └── emitter.go               # MODIFIED: Add RecoveryHintJSON and RecoveryHints field to Event
└── ...

cmd/wave/commands/
└── run.go                       # MODIFIED: Add recovery hint generation on failure path
```

**Structure Decision**: New `internal/recovery/` package encapsulates all recovery hint logic. This follows the existing codebase pattern where each `internal/` subpackage has a focused responsibility. The recovery package depends on `internal/contract` and `internal/security` (for error type assertions) and `internal/event` (for `RecoveryHintJSON`). The `cmd/wave/commands/run.go` imports `internal/recovery` to build and render hints.

## Implementation Approach

### Step 1: Define StepError type in pipeline package

Create `internal/pipeline/errors.go` with `StepError` struct that implements `error` and `Unwrap()`.
Modify `executor.go:271` and `:1333` to return `&StepError{StepID: step.ID, Err: err}` instead of `fmt.Errorf("step %q failed: %w", step.ID, err)`.

**Files**: `internal/pipeline/errors.go` (new), `internal/pipeline/executor.go` (modify 2 lines)

### Step 2: Create recovery package with core types and shell escaping

Create `internal/recovery/` with:
- `recovery.go`: `HintType`, `ErrorClass`, `RecoveryHint`, `RecoveryBlock` types
- `shell.go`: `ShellEscape(s string) string` — POSIX single-quote escaping
- `shell_test.go`: Tests for empty string, simple string, single quotes, special chars, spaces

**Files**: `internal/recovery/recovery.go` (new), `internal/recovery/shell.go` (new), `internal/recovery/shell_test.go` (new)

### Step 3: Implement error classification

Create `internal/recovery/classify.go` with `ClassifyError(err error) ErrorClass`:
- `errors.As(err, &contract.ValidationError{})` → `ClassContractValidation`
- `errors.As(err, &security.SecurityValidationError{})` → `ClassSecurityViolation`
- Non-empty error message → `ClassRuntimeError`
- Empty/generic message → `ClassUnknown`

Also refactor `isContractValidationError()` in `executor_enhanced.go` to use `errors.As()`.

**Files**: `internal/recovery/classify.go` (new), `internal/recovery/classify_test.go` (new), `internal/pipeline/executor_enhanced.go` (modify)

### Step 4: Build recovery block builder

Create `internal/recovery/recovery.go` (extend) with `BuildRecoveryBlock()`:
- Always adds resume hint: `wave run <pipeline> <escaped-input> --from-step <step>`
- Adds force hint if `ClassContractValidation`: `wave run <pipeline> <escaped-input> --from-step <step> --force`
- Always adds workspace hint: `ls .wave/workspaces/<runID>/<stepID>/`
- Adds debug hint if `ClassRuntimeError` or `ClassUnknown`: append `--debug` suggestion

**Files**: `internal/recovery/recovery.go` (extend), `internal/recovery/recovery_test.go` (new)

### Step 5: Implement text formatter for stderr output

Create `internal/recovery/format.go` with `FormatRecoveryBlock(block *RecoveryBlock) string`:
- Header: `"Recovery options:"`
- Each hint: `"  <label>:"`  /  `"    <command>"`
- Total ≤ 8 lines (excluding blank separators)

**Files**: `internal/recovery/format.go` (new), `internal/recovery/format_test.go` (new)

### Step 6: Extend Event struct for JSON mode

Add `RecoveryHintJSON` struct and `RecoveryHints []RecoveryHintJSON` field to `event.Event`.

**Files**: `internal/event/emitter.go` (modify)

### Step 7: Integrate into run.go error path

Modify `runRun()` in `cmd/wave/commands/run.go`:
- After `execErr != nil`, extract `StepError` via `errors.As()`
- Call `recovery.ClassifyError()` on the underlying error
- Call `recovery.BuildRecoveryBlock()` with pipeline name, input, step ID, run ID, error class
- Text/auto/quiet modes: `recovery.FormatRecoveryBlock()` → `fmt.Fprintf(os.Stderr, ...)`
- JSON mode: emit a failure event with `RecoveryHints` field populated
- Return the original error after printing hints

**Files**: `cmd/wave/commands/run.go` (modify)

### Step 8: Handle edge cases

- Empty input: omit input argument from recovery command
- Special characters in input: handled by `ShellEscape()`
- `--from-step` already in use: hint shows `--from-step` for the *currently* failed step
- Missing workspace directory: print path regardless
- Single-step pipeline: hints still shown

**Files**: Covered by tests in `internal/recovery/recovery_test.go`

## Complexity Tracking

_No constitution violations to justify._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| (none) | — | — |
