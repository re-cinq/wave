# Feature Specification: Pipeline Recovery Hints on Failure

**Feature Branch**: `086-pipeline-recovery-hints`
**Created**: 2026-02-13
**Status**: Draft
**Input**: GitHub Issue #85 — feat(cli): show contextual recovery hints on pipeline failure

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Resume Hint After Step Failure (Priority: P1)

A user runs a long multi-step pipeline. One of the intermediate steps fails due to a runtime error (e.g., adapter crash, timeout, or code generation failure). The user sees a clear, copy-pasteable `wave run` command that resumes execution from the failed step, including the original pipeline name and input string so no information is lost.

**Why this priority**: This is the most common recovery action. Every pipeline failure should provide at least one actionable recovery command. Without it, users must remember or reconstruct the correct `--from-step` incantation manually, which is error-prone and frustrating — especially for long-running pipelines.

**Independent Test**: Can be fully tested by running any pipeline with a step that fails (e.g., a mocked adapter returning exit code 1) and verifying that the stderr output contains a valid, copy-pasteable `wave run ... --from-step <step>` command with the correct pipeline name, input, and step ID.

**Acceptance Scenarios**:

1. **Given** a pipeline `feature` with steps [navigate, implement, review] is executed with input `"add auth"`, **When** the `implement` step fails with a runtime error, **Then** the CLI prints a recovery block to stderr containing the command: `wave run feature "add auth" --from-step implement`
2. **Given** a pipeline is executed with input that contains special characters (e.g., quotes, ampersands), **When** a step fails, **Then** the recovery command properly quotes/escapes the input so it is directly pasteable into a shell.
3. **Given** a pipeline is executed and the first step fails, **When** the recovery hint is shown, **Then** it still includes `--from-step` pointing to the first step (rather than omitting it).

---

### User Story 2 - Force Hint After Contract Validation Failure (Priority: P1)

A user's pipeline step produces output that fails contract validation (e.g., JSON schema mismatch). In addition to the standard resume hint, the CLI also suggests `--force` to skip contract validation on retry, since the user may want to inspect the output and proceed despite the validation failure.

**Why this priority**: Contract validation failures are a distinct error class with a specific, well-known workaround (`--force`). Showing this option only when relevant avoids noise and provides targeted guidance.

**Independent Test**: Can be tested by running a pipeline with a step whose output intentionally fails JSON schema validation (e.g., missing a required field), and verifying the output includes both the `--from-step` and `--from-step --force` commands.

**Acceptance Scenarios**:

1. **Given** a pipeline step fails due to contract validation (the error wraps a `contract.ValidationError`), **When** the recovery hints are shown, **Then** the output includes a `--force` variant of the resume command with an explanatory label such as "Resume and skip validation checks" (note: `--force` skips phase validation and stale artifact checks in addition to contract validation, per the existing `ResumeWithValidation` implementation — the label must accurately reflect this broader scope).
2. **Given** a pipeline step fails due to a runtime error (not a contract failure), **When** the recovery hints are shown, **Then** the `--force` variant is NOT shown (to avoid confusion).

---

### User Story 3 - Workspace Path for Artifact Inspection (Priority: P2)

After a failure, the user wants to inspect the workspace artifacts (e.g., the generated code, adapter output, partial results). The CLI prints the actual filesystem path to the failed step's workspace directory so the user can navigate directly to it.

**Why this priority**: Post-mortem artifact inspection is the second most common recovery action. The workspace path is already deterministic, but users should not have to guess or reconstruct it.

**Independent Test**: Can be tested by running a pipeline that fails and verifying the output contains an `ls` command (or plain path) pointing to the correct `.wave/workspaces/<pipelineID>/<stepID>/` directory that actually exists on disk.

**Acceptance Scenarios**:

1. **Given** a pipeline step fails, **When** the recovery hints are shown, **Then** the workspace path to the failed step is printed as a navigable path (e.g., `ls .wave/workspaces/<pipelineID>/<stepID>/`).
2. **Given** a pipeline step fails and the workspace directory exists, **When** the user copies and runs the printed path command, **Then** the command succeeds and shows the workspace contents.

---

### User Story 4 - Debug Suggestion for Ambiguous Errors (Priority: P3)

When a step fails with an ambiguous or unclear error (e.g., non-zero exit code with no meaningful error message), the CLI suggests using the `--debug` flag to get more detailed execution output on the next run.

**Why this priority**: This is a quality-of-life enhancement for difficult debugging scenarios. It is less critical than the core resume and artifact-inspection hints but helps users who encounter opaque failures.

**Independent Test**: Can be tested by triggering a failure with a generic error (e.g., exit code 1 with empty stderr) and verifying the output suggests re-running with `--debug`.

**Acceptance Scenarios**:

1. **Given** a pipeline step fails with an error that lacks a specific error type (i.e., not a contract validation error or security error), **When** the recovery hints are shown, **Then** the output includes a suggestion to re-run with `--debug` for more information.
2. **Given** a pipeline step fails with a clearly typed error (contract or security), **When** the recovery hints are shown, **Then** the `--debug` suggestion may still be shown, but is presented after the more specific hints.

---

### Edge Cases

- What happens when the pipeline name or input is empty? Recovery hints should still render valid commands (omitting empty input rather than printing empty quotes).
- What happens when the pipeline is invoked via `--pipeline` and `--input` flags instead of positional arguments? The recovery hint should reconstruct the command using positional arguments for brevity, or match the original invocation style.
- What happens when `--from-step` is already being used (i.e., the user resumed and the resumed step fails again)? The hint should show the same `--from-step` for the currently-failed step, not the originally requested step.
- What happens when the workspace directory has been cleaned up before the hint is printed? The workspace path hint should still be printed, but should not claim the directory exists.
- What happens in `--output json` mode? Recovery hints should be included as structured fields in the JSON event (e.g., a `recovery_hints` array), not as stderr text.
- What happens in `--output quiet` mode? Recovery hints should still be printed, since they are actionable error output relevant to the failure.
- What happens when the pipeline has only one step? The hint block should still be shown (the user may not realize `--from-step` is available even for single-step pipelines).

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: On any pipeline step failure, the CLI MUST print at least one recovery command to stderr (in text/auto output modes) or include recovery hints in the failure event (in JSON output mode).
- **FR-002**: The recovery block MUST include a `--from-step` command containing the pipeline name, the original input string (properly shell-escaped), and the failed step ID.
- **FR-003**: When the failure is caused by contract validation (detected via `errors.As()` unwrapping to `*contract.ValidationError`, replacing the existing string-matching approach in `isContractValidationError()`), the recovery block MUST include an additional `--from-step ... --force` command variant.
- **FR-004**: The recovery block MUST include the filesystem path to the failed step's workspace directory.
- **FR-005**: When the error is ambiguous (not a typed contract or security error), the recovery block SHOULD include a suggestion to re-run with `--debug`.
- **FR-006**: The recovery block MUST be concise — no more than 8 lines of output (excluding blank separator lines).
- **FR-007**: Recovery hints MUST be printed after the error message, visually separated, and clearly labeled (e.g., with a header like "Recovery options:").
- **FR-008**: In JSON output mode (`-o json`), recovery hints MUST be included as structured data in the failure event, not as stderr text. This is achieved by adding a `RecoveryHints []RecoveryHint` field to the existing `event.Event` struct (with `json:"recovery_hints,omitempty"`). The field is only populated on failure events, keeping the Event struct backward-compatible since the field is omitted when empty.
- **FR-009**: The input string in recovery commands MUST be properly shell-escaped using single-quote wrapping with interior single-quote escaping (`'it'\''s'` style), implemented as a pure-Go helper function in the recovery hints package — no external shell-escaping library required. This is the POSIX-standard approach and produces the most portable output.
- **FR-010**: Recovery hints MUST be generated from context already available at the call site in the command layer (`cmd/wave/commands/run.go`). The `RecoveryBlock` is constructed after the executor returns an error, using the pipeline name, input string, failed step ID, workspace path pattern, and error classification. No additional I/O or state lookups are required. The existing `ErrorMessageProvider` in `internal/pipeline/validation.go` is NOT modified — recovery hints are a separate, CLI-layer concern.

### Key Entities

- **RecoveryHint**: A single suggested action comprising a label (human-readable description), a command (the shell command to execute), and a condition (when to show this hint).
- **RecoveryBlock**: An ordered collection of RecoveryHints generated for a specific step failure, along with metadata (pipeline name, step ID, workspace path, error classification).
- **ErrorClassification**: The categorization of a failure as one of: `contract_validation`, `security_violation`, `runtime_error`, or `unknown`. Classification is performed using Go's `errors.As()` to unwrap the error chain: `*contract.ValidationError` → `contract_validation`, `*security.SecurityValidationError` → `security_violation`. Any error not matching these types is classified as `runtime_error` if it has a non-empty message, or `unknown` if the error message is empty or generic (e.g., bare "exit status 1").

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Every pipeline step failure produces at least one recovery hint visible to the user (100% coverage of failure paths).
- **SC-002**: A user can copy-paste any printed recovery command directly into their shell and have it execute successfully (no manual editing required for escaping or quoting).
- **SC-003**: The `--force` hint appears if and only if the failure was caused by contract validation — false positive rate of 0%.
- **SC-004**: Recovery hints add no more than 8 lines to the failure output, keeping the signal-to-noise ratio high.
- **SC-005**: All existing tests continue to pass — no regressions in error handling, output formatting, or pipeline execution.
- **SC-006**: Recovery hints are covered by unit tests for each error classification (contract, security, runtime, unknown) and for edge cases (empty input, special characters, JSON output mode).

## Clarifications

The following ambiguities were identified and resolved during spec review. Each decision is informed by codebase analysis and established patterns.

### C1: Error classification mechanism — type assertion vs. string matching

**Ambiguity**: The spec referenced `contract.ValidationError` for error detection, but the existing codebase (`isContractValidationError()` in `internal/pipeline/executor_enhanced.go`) uses fragile string-matching against error messages. The spec did not specify which approach to use.

**Resolution**: Use `errors.As()` type assertion against `*contract.ValidationError` and `*security.SecurityValidationError`. This is the idiomatic Go approach, avoids false positives from error message changes, and both error types are already well-defined structs in the codebase. The existing `isContractValidationError()` string-matching function should be refactored to use `errors.As()` as part of this feature.

**Sections updated**: FR-003, Key Entities (ErrorClassification)

### C2: Shell escaping strategy for recovery commands

**Ambiguity**: FR-009 required "properly shell-escaped" commands but did not specify the escaping algorithm or whether to use an external library.

**Resolution**: Use POSIX single-quote wrapping with interior single-quote escaping (`'it'\''s'` style), implemented as a pure-Go helper. This is the most portable approach, avoids adding a dependency, and handles all special characters including spaces, quotes, ampersands, semicolons, and glob characters. The helper should be a small function in the recovery hints package (~15 lines).

**Sections updated**: FR-009

### C3: Architectural placement — executor layer vs. command layer

**Ambiguity**: FR-010 stated hints should be generated from "context already available in the executor" but did not specify whether the `RecoveryBlock` is constructed in `internal/pipeline/` (executor layer) or `cmd/wave/commands/` (command layer). The existing `ErrorMessageProvider` in the pipeline layer already produces verbose troubleshooting output.

**Resolution**: Recovery hints are a CLI-layer concern, constructed in `cmd/wave/commands/run.go` after the executor returns an error. The command layer already has access to pipeline name, input string, output mode, and debug flag. The executor error is unwrapped to classify it, then the `RecoveryBlock` is built and rendered. The existing `ErrorMessageProvider` is not modified — it serves a different purpose (detailed troubleshooting guidance during execution).

**Sections updated**: FR-010

### C4: JSON output mode — extending the Event struct

**Ambiguity**: FR-008 required structured recovery hints in JSON failure events, but the `event.Event` struct had no field for this. The spec did not specify whether to extend the existing struct or create a new event type.

**Resolution**: Add a `RecoveryHints []RecoveryHint` field (with `json:"recovery_hints,omitempty"`) to the existing `event.Event` struct. This keeps the single-event-type model, maintains backward compatibility (omitempty means no change for non-failure events), and avoids introducing a parallel event hierarchy. The `RecoveryHint` struct has three fields: `Label string`, `Command string`, and `Type string` (matching the hint types: "resume", "force", "workspace", "debug").

**Sections updated**: FR-008

### C5: `--force` flag scope in hint labeling

**Ambiguity**: User Story 2 described `--force` as skipping "contract validation", but the actual implementation in `ResumeWithValidation()` skips phase validation AND stale artifact checks in addition to contract validation. The hint label could mislead users about what `--force` actually does.

**Resolution**: The `--force` hint label must accurately reflect the broader scope: "Resume and skip validation checks" rather than "Resume and skip contract validation". This prevents user surprise and aligns the hint with the actual `--force` behavior. The hint may include a parenthetical clarification, e.g., "(skips contract, phase, and stale-artifact validation)".
