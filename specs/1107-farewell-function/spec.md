# Feature Specification: Farewell Function

**Feature Branch**: `1107-farewell-function`
**Created**: 2026-04-21
**Status**: Draft
**Input**: User description: "add a Farewell function"

## Clarifications

### Session 2026-04-21

- Q: Should the farewell string be localized or English-only? → A: English-only. Rationale: rest of the Wave CLI/TUI/docs is English-only; no i18n framework exists in the codebase; introducing one for a farewell line is out of scope.
- Q: Should the farewell vary (random pool) or be a single fixed string? → A: Single fixed string. Rationale: satisfies SC-003 determinism without special-casing, keeps FR-008 "single source" trivially true, avoids adding a randomness seed/config surface.
- Q: Which quiet flag governs suppression (FR-005, AS-3.2)? → A: Reuse the existing global `--quiet` / non-interactive signal already honored by other CLI output; no new flag introduced.
- Q: Does "successful command" include read-only / list commands (e.g. `wave list`)? → A: Yes — all successful interactive command completions print the farewell; suppression rules (non-TTY, quiet, failure) still apply uniformly.
- Q: Where does the recipient name come from for CLI auto-display (AS-1.3)? → A: From `$USER` / OS username only; no Wave-specific profile lookup. Empty/unknown falls back to generic wording per edge case.

## User Scenarios & Testing _(mandatory)_

### User Story 1 - User sees farewell message on session end (Priority: P1)

When a user ends a Wave CLI session (exits a command, completes a pipeline run, or quits the TUI), the system displays a short, friendly farewell message so the user gets a clear signal that the session concluded cleanly.

**Why this priority**: Exit feedback is the minimal viable slice — without it the feature has no user-visible effect. A clean farewell also gives users confidence the process terminated normally rather than crashing silently.

**Independent Test**: Run any Wave command to completion (e.g., `wave run <pipeline>`) and observe that a farewell line is printed to stdout before the process exits with status 0. Can be demonstrated without any other story implemented.

**Acceptance Scenarios**:

1. **Given** a user runs a Wave command that completes successfully, **When** the command finishes, **Then** a farewell message is printed on its own line to stdout before process exit.
2. **Given** a user quits the Wave TUI via the standard quit key, **When** the TUI tears down, **Then** a farewell message is shown in the terminal after the UI clears.
3. **Given** a user has set a username or profile name, **When** the farewell is shown, **Then** the message addresses the user by that name when available.

---

### User Story 2 - Programmatic farewell for embedders (Priority: P2)

Developers embedding Wave or writing pipelines/personas can call a public `Farewell` function to produce a farewell string for inclusion in custom output (scripts, hooks, logs, notifications).

**Why this priority**: Enables reuse of the same farewell copy from multiple call sites (CLI, TUI, lifecycle hooks) without duplicating strings. Valuable but only after the user-visible message exists.

**Independent Test**: Call `Farewell` from a unit test and assert it returns a non-empty string matching the expected format.

**Acceptance Scenarios**:

1. **Given** a caller invokes `Farewell` with no arguments, **When** the call returns, **Then** a non-empty default farewell string is produced.
2. **Given** a caller invokes `Farewell` with a recipient name, **When** the call returns, **Then** the returned string contains that name verbatim.

---

### User Story 3 - Silent / scripting mode (Priority: P3)

Users running Wave non-interactively (CI, scripts, piped output) can suppress the farewell message so it does not pollute machine-readable output.

**Why this priority**: Needed for clean automation but the default interactive experience works without it.

**Independent Test**: Run a Wave command with stdout redirected to a file or a quiet flag set; assert the output contains no farewell line.

**Acceptance Scenarios**:

1. **Given** stdout is not a TTY, **When** a Wave command finishes, **Then** no farewell message is printed.
2. **Given** a user passes a quiet flag, **When** a Wave command finishes, **Then** no farewell message is printed regardless of TTY.

### Edge Cases

- User interrupts with Ctrl+C: farewell MAY be skipped since the process is aborting, and MUST NOT cause the signal handler to hang.
- Command fails with non-zero exit: error output takes precedence; farewell MUST NOT mask or suppress error messages.
- Output redirected to file or pipe: farewell MUST be suppressed to keep machine-readable output clean.
- Empty or unknown recipient name: fall back to a generic salutation; MUST NOT render an empty-name placeholder.
- Localization: English-only for now (see Clarifications 2026-04-21); no i18n layer is introduced.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST expose a `Farewell` function that returns a farewell message string.
- **FR-002**: `Farewell` MUST accept an optional recipient name and include it in the returned string when provided and non-empty.
- **FR-003**: `Farewell` MUST return a stable, non-empty default message when no name is provided.
- **FR-004**: The Wave CLI MUST print the farewell message at the end of successful interactive command execution.
- **FR-005**: The farewell output MUST be suppressed when stdout is not a TTY or when a quiet / non-interactive flag is set.
- **FR-006**: The farewell message MUST be written to stdout (not stderr) and MUST NOT alter the process exit code.
- **FR-007**: On command failure, the farewell MUST NOT be printed so that error messages remain the final user-visible output.
- **FR-008**: The farewell string MUST be sourced from a single place so CLI, TUI, and embedder call sites produce identical wording.
- **FR-009**: The farewell MUST be a single fixed English string template (optionally interpolated with a recipient name); no random-message pool, no localization table.
- **FR-010**: The CLI MUST resolve the auto-filled recipient name from the OS username (`$USER` or equivalent) only; if unset/empty, the generic default message is used.
- **FR-011**: Suppression (FR-005) MUST reuse the existing global quiet / non-interactive CLI signal; no new flag is introduced by this feature.

### Key Entities _(include if feature involves data)_

- **Farewell message**: a short human-readable string, optionally parameterized by recipient name. No persistence required.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: 100% of successful interactive Wave CLI command runs end with a visible farewell line before exit.
- **SC-002**: 0% of non-TTY or quiet runs emit the farewell line (verified by piping output to a file and asserting absence).
- **SC-003**: Calling `Farewell` with the same input produces the same output within a single build (determinism for fixed-message mode), verified by unit test.
- **SC-004**: Adding farewell output adds no more than 50 ms to total CLI wall-clock time on a baseline command.
