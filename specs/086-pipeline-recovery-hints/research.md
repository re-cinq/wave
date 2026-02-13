# Research: Pipeline Recovery Hints on Failure

**Feature Branch**: `086-pipeline-recovery-hints`
**Date**: 2026-02-13

## Decision 1: Error Classification Mechanism

**Decision**: Use `errors.As()` type assertion against `*contract.ValidationError` and `*security.SecurityValidationError`.

**Rationale**: The existing `isContractValidationError()` function in `internal/pipeline/executor_enhanced.go:210-230` uses brittle string matching against keywords like `"contract validation failed"`, `"schema validation"`, `"artifact.json"`, and `"json_schema"`. This approach has false-positive risk (any error message containing these substrings triggers the match) and breaks when error messages change.

Both target error types are already well-defined structs:
- `contract.ValidationError` in `internal/contract/contract.go:35-42` — has `ContractType`, `Message`, `Details`, `Retryable`, `Attempt`, `MaxRetries` fields
- `security.SecurityValidationError` in `internal/security/errors.go:9-15` — has `Type`, `Message`, `Details`, `Retryable`, `SuggestedFix` fields

Using `errors.As()` is idiomatic Go and handles wrapped error chains correctly. This also allows refactoring the existing `isContractValidationError()` to use `errors.As()` as a side benefit.

**Alternatives Rejected**:
- String matching (current approach): Fragile, false-positive prone, breaks on message changes
- Custom error interface (e.g., `Classifiable` interface): Over-engineered for 2 types; would require modifying existing error types for no clear benefit

## Decision 2: Shell Escaping Strategy

**Decision**: Use POSIX single-quote wrapping with interior single-quote escaping (`'value'` → `'it'\''s'`), implemented as a pure-Go helper.

**Rationale**: POSIX single quoting is the most portable approach. Inside single quotes, all characters are literal (no expansion), with the one exception that single quotes themselves cannot appear. The standard workaround is to end the single-quoted segment, insert an escaped single quote (`\'`), and restart the single-quoted segment: `'it'\''s'`.

This is a ~15-line function. No external dependency needed.

**Alternatives Rejected**:
- Double-quote escaping: Requires escaping `$`, `` ` ``, `\`, `!`, `"`, which is more complex and error-prone
- `shellescape` Go library: Adds a dependency for trivial functionality
- `%q` formatting: Go string quoting is not shell-compatible

## Decision 3: Architectural Placement — CLI Layer

**Decision**: Build `RecoveryBlock` in `cmd/wave/commands/run.go` after the executor returns an error, not inside the executor.

**Rationale**: Recovery hints are a presentation concern. The executor's job is to run steps and report failures; the CLI layer's job is to format actionable user-facing output. All necessary information is already available in `runRun()`:
- `opts.Pipeline` — pipeline name
- `opts.Input` — original input string
- `opts.Output.Format` — output mode (auto/json/text/quiet)
- `opts.FromStep` / `opts.Force` — current invocation flags
- `runID` — for workspace path construction
- `execErr` — the error to classify

The existing `ErrorMessageProvider` in `internal/pipeline/validation.go` generates *troubleshooting guidance* during execution (verbose, multi-paragraph). Recovery hints are a separate concept: *concise, actionable commands* shown after failure. They should not be mixed.

**Alternatives Rejected**:
- Building hints in the executor: Violates separation of concerns; executor doesn't know about CLI flags or output modes
- Extending `ErrorMessageProvider`: Different purpose (troubleshooting vs. recovery commands); would bloat an already large error formatter

## Decision 4: Extracting Failed Step ID from Error Chain

**Decision**: Parse the step ID from the error message pattern `step %q failed:` using a regex or string parser, since the executor wraps errors with `fmt.Errorf("step %q failed: %w", step.ID, err)`.

**Rationale**: The executor at `internal/pipeline/executor.go:271` and `:1333` wraps step errors as `fmt.Errorf("step %q failed: %w", step.ID, err)`. Then `run.go:246` wraps again as `"pipeline execution failed: %w"`.

Two approaches were considered:
1. **Parse step ID from error string**: Simple regex `step "([^"]+)" failed:` extracts the step ID reliably since the format is controlled by Wave.
2. **Define a `StepError` type**: More robust but requires modifying the executor to return `&StepError{StepID: step.ID, Err: err}` instead of `fmt.Errorf(...)`.

The `StepError` approach is cleaner and enables `errors.As()` extraction. Since this is prototype phase (backward compatibility is not a constraint), introducing `StepError` is the right move. It also benefits the existing `isContractValidationError()` refactor by making the error chain cleaner.

**Alternatives Rejected**:
- Parsing error strings: Works but fragile; same string-matching anti-pattern the spec wants to eliminate
- Passing step ID separately (return value or struct): Requires changing the executor interface, which is more invasive

## Decision 5: JSON Output Mode — Extending Event Struct

**Decision**: Add `RecoveryHints []RecoveryHint` field to `event.Event` with `json:"recovery_hints,omitempty"`.

**Rationale**: The `event.Event` struct (`internal/event/emitter.go:11-38`) is the single event type for all pipeline events. Adding an `omitempty` field keeps backward compatibility — non-failure events emit no extra field. The `RecoveryHint` struct has `Label`, `Command`, and `Type` fields.

For JSON output mode, recovery hints are emitted as part of the failure event rather than printed to stderr. The `NDJSONEmitter` already handles JSON encoding; the new field just needs to be populated.

**Alternatives Rejected**:
- New event type (e.g., `RecoveryEvent`): Breaks single-event-type model; requires updating all consumers
- Separate JSON envelope: Over-engineered; `omitempty` achieves the same thing simply

## Decision 6: Output in --output quiet Mode

**Decision**: Recovery hints are printed to stderr in `--output quiet` mode.

**Rationale**: Per FR-001, recovery hints MUST appear on every failure. The `quiet` mode suppresses progress output but recovery hints are actionable error output — they help the user recover from the failure. Suppressing them would defeat the purpose.

**Alternatives Rejected**:
- Suppress in quiet mode: Violates FR-001; user loses actionable recovery info
- Print to stdout: Breaks convention; quiet mode should keep stdout clean for scripting

## Decision 7: Workspace Path Pattern

**Decision**: Use the deterministic pattern `.wave/workspaces/<runID>/<stepID>/` for workspace path hints.

**Rationale**: The workspace manager creates directories at `<wsRoot>/<pipelineID>/<stepID>/` where `wsRoot` defaults to `.wave/workspaces` (see `run.go:178`). The `runID` (which equals `pipelineID`) is generated by `pipeline.GenerateRunID()` and is available in `runRun()`. The step ID is extracted from the error chain.

The hint prints `ls .wave/workspaces/<runID>/<stepID>/` as a navigable command. Per the edge case in the spec, the hint is always printed even if the directory may have been cleaned up.

**Alternatives Rejected**:
- Checking directory existence before printing: Adds I/O to the error path; the hint is still useful as a reference even if the directory is gone
- Using absolute paths: Relative paths are shorter and work from the project root where users typically run `wave`
