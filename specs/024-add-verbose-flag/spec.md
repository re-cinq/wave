# Feature Specification: Add --verbose Flag to Wave CLI

**Feature Branch**: `024-add-verbose-flag`
**Created**: 2026-02-06
**Status**: Draft
**Input**: User description: "Add a --verbose flag to the wave CLI"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Global Verbose Output During Pipeline Execution (Priority: P1)

As a Wave user running pipelines, I want a `--verbose` flag that increases the detail of output across all commands so that I can see what Wave is doing at each stage without needing to switch to full `--debug` mode.

Currently the gap between normal output and `--debug` is too large — users either get minimal progress information or full debug dumps. A verbose mode provides a useful middle ground that shows operational context without overwhelming internal traces.

**Why this priority**: This is the core value proposition. Most users encounter the need for more output when running `wave run` and pipeline commands. A verbose mode bridges the gap between normal and debug output.

**Independent Test**: Can be fully tested by running `wave run --verbose <pipeline>` and verifying that additional operational details (step transitions, artifact paths, contract validation results, workspace locations) appear in the output while raw debug traces do not.

**Acceptance Scenarios**:

1. **Given** a user runs `wave run --verbose my-pipeline`, **When** the pipeline executes, **Then** each step displays its workspace path, injected artifacts, persona name, and contract validation outcome in addition to normal progress output.
2. **Given** a user runs `wave run my-pipeline` without `--verbose`, **When** the pipeline executes, **Then** the output is identical to current behavior (no regression).
3. **Given** a user runs `wave --verbose run my-pipeline` (flag before subcommand), **When** the pipeline executes, **Then** verbose output is enabled because the flag is a persistent global flag.

---

### User Story 2 - Verbose Output for Non-Pipeline Commands (Priority: P2)

As a Wave user running utility commands (validate, status, list, clean, artifacts), I want `--verbose` to provide additional context so that I can understand what the tool is checking or querying without resorting to debug mode.

**Why this priority**: Extends the verbose capability beyond pipeline execution to the full CLI surface. Valuable for troubleshooting configuration issues and understanding tool behavior, but secondary to the primary pipeline use case.

**Independent Test**: Can be tested by running `wave validate --verbose wave.yaml` and verifying that it shows which validators are invoked, which files are checked, and intermediate validation results rather than just the final summary.

**Acceptance Scenarios**:

1. **Given** a user runs `wave validate --verbose wave.yaml`, **When** validation completes, **Then** the output includes which validators ran, what was checked, and pass/fail per section — not just the final summary.
2. **Given** a user runs `wave status --verbose`, **When** the status is displayed, **Then** additional metadata such as database path, last state transition timestamps, and workspace locations are shown.
3. **Given** a user runs `wave clean --verbose`, **When** cleanup runs, **Then** each workspace being removed is listed with its size before deletion.

---

### User Story 3 - Verbose and Debug Interaction (Priority: P3)

As a Wave power user, I want `--verbose` and `--debug` to compose predictably so that I understand the output hierarchy and can choose the right level of detail for my situation.

**Why this priority**: Ensures consistent UX for users who may combine flags or upgrade from verbose to debug. Important for long-term usability but not blocking for initial release.

**Independent Test**: Can be tested by running commands with `--debug`, `--verbose`, and both flags simultaneously, then comparing output to verify the correct level is active.

**Acceptance Scenarios**:

1. **Given** a user runs `wave --debug --verbose run my-pipeline`, **When** the pipeline executes, **Then** debug output is shown (debug supersedes verbose as the higher detail level).
2. **Given** a user runs `wave --verbose run my-pipeline`, **When** the pipeline executes, **Then** verbose output is shown but internal debug traces (raw adapter commands, full environment variable dumps) are not included.

---

### Edge Cases

- What happens when `--verbose` is combined with `--quiet` (on commands that support it)? `--quiet` takes precedence and suppresses all output including verbose details. _(Future behavior: `--quiet` flag does not currently exist. This documents intended interaction for when it is added.)_
- What happens when `--verbose` is used with `--format json`? Verbose details are included as additional fields in the JSON output rather than free-text — structured output remains structured. _(Future behavior: `--format json` flag does not currently exist. This documents intended interaction for when it is added.)_
- What happens when `--verbose` is used with `--no-logs`? Verbose details appear in the progress display stream (stderr) since log output (stdout) is suppressed.
- What happens when `--verbose` is used in a non-TTY (piped) environment? Verbose output is still emitted as plain text to the appropriate stream, without ANSI formatting.
- What happens when the global `--verbose` is used alongside the existing `validate --verbose` local flag? They compose equivalently — either flag activates verbose mode for the validate command. Cobra's persistent vs. local flag resolution handles this: the local `-v` flag shadows the persistent `-v` on the `validate` subcommand, while the persistent `-v` applies to all other subcommands. Since both activate the same behavior, the user experience is identical regardless of which flag is resolved.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST accept a `--verbose` / `-v` persistent flag on the root command, making it available to all subcommands. The `-v` shorthand on the global persistent flag coexists with the existing `-v` shorthand on the `validate` subcommand's local `--verbose` flag. Cobra resolves this by letting the local flag shadow the persistent flag on that specific subcommand. Both activate verbose mode, so no behavioral conflict exists.
- **FR-002**: System MUST propagate the verbose setting to all subcommand execution contexts so that any command can check whether verbose mode is active. Implementation follows the existing `--debug` flag pattern: a `verbose` bool is read from root PersistentFlags, passed into command run functions, and threaded through executor options (e.g., `pipeline.WithVerbose(verbose)`). The effective verbosity is resolved at point of use: if debug is set, use debug level; else if verbose is set, use verbose level; else use normal.
- **FR-003**: When verbose mode is active during pipeline execution, the system MUST display: step workspace paths, injected artifact filenames, persona names per step, and contract validation pass/fail details per step.
- **FR-004**: When verbose mode is active during non-pipeline commands, the system MUST display additional operational context relevant to that command. Initial scope is limited to commands with explicit acceptance scenarios: `validate` (validator details), `status` (metadata), and `clean` (deletion details). Remaining commands (`init`, `do`, `meta`, `resume`, `logs`, `cancel`, `artifacts`, `list`, `migrate`) inherit the global flag but produce no additional verbose output in this iteration — they silently accept the flag with no behavioral change.
- **FR-005**: The existing `--debug` flag MUST continue to function unchanged. When both `--debug` and `--verbose` are active, the system MUST use the higher detail level (debug).
- **FR-006**: The `--verbose` flag MUST NOT change the default output behavior when omitted — all current output remains identical (zero regression).
- **FR-007**: The `--verbose` flag MUST NOT conflict with the existing `--verbose/-v` flag on the `validate` subcommand. The two flags should compose equivalently — either one activates verbose mode for that command.
- **FR-008**: Verbose output MUST respect the existing dual-stream model: machine-readable data to stdout, human-readable verbose details to stderr. Verbose information is emitted by extending the existing event system with verbose-enriched events (additional fields on existing event types or new verbose-specific event types). The NDJSON emitter includes verbose data in stdout events; the progress emitter renders verbose data on stderr. This ensures `--no-logs` interaction works automatically (progress emitter on stderr still shows verbose data when stdout NDJSON is suppressed).

### Key Entities

- **VerbosityLevel**: Represents the current output detail level. Three values: normal (default), verbose (activated by `--verbose`), debug (activated by `--debug`). When both flags are set, the effective level is debug.
- **VerboseOutput**: Additional contextual information emitted during command execution. Only visible when verbosity is elevated above normal. Content varies per command but always provides operational insight without exposing internal traces.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Running any Wave command with `--verbose` produces additional output lines compared to the same command without the flag, for commands that have verbose-eligible output.
- **SC-002**: Running any Wave command without `--verbose` produces output identical to the current behavior (zero regression).
- **SC-003**: `wave --help` documents the `--verbose` flag with a clear, concise description of its purpose.
- **SC-004**: All existing tests continue to pass after the change with no regressions.
- **SC-005**: The verbose flag is covered by unit tests that assert verbose-specific output is present when the flag is set and absent when it is not.

## Clarifications _(resolved)_

The following ambiguities were identified during specification review and resolved autonomously based on codebase analysis and industry best practices.

### CLR-001: `-v` Shorthand Conflict Between Global and Validate Local Flags

**Category**: Integration points / Constraints

**Question**: FR-001 specifies `--verbose / -v` as a global persistent flag. The `validate` subcommand already registers `-v` as a local shorthand for its own `--verbose` flag (`cmd/wave/commands/validate.go:35`). How should the shorthand conflict be handled?

**Resolution**: Keep both `-v` registrations. Cobra's flag resolution semantics handle this correctly — local flags shadow persistent flags of the same name on the specific subcommand where they are defined. Since both flags activate verbose mode, the user gets consistent behavior regardless of whether they write `wave validate -v` or `wave -v validate`. This matches FR-007's intent that the flags "compose equivalently."

**Rationale**: No breaking change to validate's existing interface. The edge case section and FR-001 have been updated to document this Cobra behavior explicitly.

---

### CLR-002: VerbosityLevel Propagation Mechanism

**Category**: Domain model / Integration points

**Question**: The spec defines a `VerbosityLevel` entity but does not specify where the type lives or how verbosity flows through the system. The existing `--debug` flag uses a boolean threaded through executor options (`cmd/wave/commands/run.go:69-70`, `pipeline.WithDebug(debug)`). Should a formal enum type be introduced or should the existing bool-threading pattern be followed?

**Resolution**: Add a `verbose` bool alongside the existing `debug` bool, following the identical propagation pattern: read from `cmd.Flags().GetBool("verbose")`, pass into `runRun()`, and thread via `pipeline.WithVerbose(verbose)`. The effective verbosity level is resolved at point of use with: `if debug { ... } else if verbose { ... } else { ... }`.

**Rationale**: Lowest-friction approach that matches existing codebase patterns exactly. A formal `VerbosityLevel` enum adds abstraction for what is effectively two independent boolean flags with a simple precedence rule. FR-002 has been updated to specify this pattern.

---

### CLR-003: Edge Cases Reference Non-Existent Flags (`--quiet`, `--format json`)

**Category**: Edge cases / Constraints

**Question**: The edge cases section defines behavior for `--verbose` combined with `--quiet` and `--format json`, but neither flag exists in the current codebase. Should these be implemented as part of this feature?

**Resolution**: Keep the edge cases as forward-looking design documentation but annotate them as future behavior. Do not implement `--quiet` or `--format json` in this feature — that would expand scope well beyond "add a `--verbose` flag." Implementers should not build or test these interactions.

**Rationale**: The edge cases serve as valuable design intent for future work without blocking current implementation. Both edge case entries have been annotated with _(Future behavior)_ markers.

---

### CLR-004: Scope of Non-Pipeline Command Verbose Output

**Category**: Functional scope

**Question**: FR-004 requires verbose output for non-pipeline commands, but the CLI has 12+ subcommands and only `validate`, `status`, and `clean` have explicit acceptance scenarios. Must all commands implement verbose output?

**Resolution**: Implement verbose output only for the 4 commands with explicit acceptance scenarios: `run` (P1), `validate`, `status`, and `clean` (P2). The remaining commands (`init`, `do`, `meta`, `resume`, `logs`, `cancel`, `artifacts`, `list`, `migrate`) inherit the global flag automatically via Cobra's persistent flag mechanism but produce no additional verbose output in this iteration.

**Rationale**: Limits scope to what is specified and testable. Inventing verbose output for unspecified commands risks misalignment with user expectations. FR-004 has been updated to explicitly scope the initial implementation.

---

### CLR-005: Verbose Output Stream Routing Architecture

**Category**: Interaction flows / Integration points

**Question**: The codebase has a complex event-driven dual-stream architecture (NDJSON to stdout via `NDJSONEmitter`, progress to stderr via `ProgressEmitter`). New verbose information (workspace paths, artifact filenames, contract results) is not currently emitted by any stream. How should verbose data be routed?

**Resolution**: Extend the existing event system with verbose-enriched events. Add verbose-relevant fields to existing event types (or introduce new verbose-specific event types) so that the NDJSON emitter includes verbose data in stdout JSON events and the progress emitter renders verbose details on stderr. This ensures:
- `--no-logs` works automatically (stderr progress still renders verbose data)
- Future `--format json` works automatically (verbose data already in structured events)
- FR-008 compliance (dual-stream model respected)

**Rationale**: Most architecturally consistent with the existing event-driven output system. Avoids introducing a parallel output path that would bypass the established stream routing logic. FR-008 has been updated to specify this approach.
