# Feature Specification: CLI Compliance Polish

**Feature Branch**: `260-tui-cli-compliance`  
**Created**: 2026-03-06  
**Status**: Draft  
**Input**: [GitHub Issue #260](https://github.com/re-cinq/wave/issues/260) — CLI compliance polish per clig.dev guidelines: `--json`, `NO_COLOR`, `--no-color`, `--quiet`, error messages, output stream discipline, flag consistency.

## Context

Wave already has substantial CLI infrastructure:
- Root persistent flags: `--output` (auto/json/text/quiet), `--debug`, `--verbose`, `--no-tui`
- Per-subcommand `--format` flags on `status`, `logs`, `list`, `artifacts`, `cancel`
- `NO_COLOR` env var support in `internal/display/capability.go`
- `clean --quiet` for scripting
- Progress emitters directing output to stderr in text/quiet modes
- NDJSON event emitter for `--output json`

This issue is about **standardizing and polishing** the existing CLI surface to comply with [clig.dev](https://clig.dev) guidelines by adding convenience aliases, enforcing consistency across all subcommands, and improving error message quality.

## User Scenarios & Testing _(mandatory)_

### User Story 1 — Consistent Flag Surface Across All Subcommands (Priority: P1)

A user who frequently works with different `wave` subcommands expects the same standard flags to be available everywhere. Currently, `--json` does not exist as a standalone flag (users must use `--output json`), `--quiet`/`-q` is only on `clean`, and `--no-color` exists only as the `NO_COLOR` env var. This story unifies the flag surface.

**Why this priority**: Flag consistency is the foundation — every other story depends on flags working uniformly. Without this, users must memorize different flag combinations per subcommand.

**Independent Test**: Run `wave <any-subcommand> --help` and verify standard flags (`-q`/`--quiet`, `--json`, `--no-color`, `--debug`, `--verbose`, `--no-tui`) appear in the output.

**Acceptance Scenarios**:

1. **Given** any wave subcommand, **When** the user runs `wave <cmd> --help`, **Then** the help output lists all standard persistent flags: `-v`/`--verbose`, `-q`/`--quiet`, `--debug`, `--json`, `--no-color`, `--no-tui`, and `-o`/`--output`.
2. **Given** the root command, **When** `--json` is passed, **Then** it behaves identically to `--output json` — producing JSON on stdout with no TUI, no progress spinners, and no color. For streaming commands (`run`), this means NDJSON (one event per line); for non-streaming commands (`status`, `list`, etc.), this means structured JSON output.
3. **Given** the root command, **When** `-q`/`--quiet` is passed, **Then** it behaves identically to `--output quiet` — suppressing non-essential output.
4. **Given** both `--json` and `--output text` are passed, **Then** the CLI reports a conflict error and exits with a non-zero code and actionable message.
5. **Given** a subcommand with a local `--format` flag (e.g. `status --format json`), **When** the root `--json` flag is also passed, **Then** the root `--json` flag takes precedence and the subcommand's `--format` value is ignored.

---

### User Story 2 — Machine-Readable JSON Output (Priority: P1)

A CI/CD pipeline or script author pipes `wave` output into `jq` or another JSON parser. They need valid, parseable JSON from every subcommand when `--json` is specified — not just from `wave run`.

**Why this priority**: Machine-readable output enables automation and is a core clig.dev compliance requirement. Tied with flag consistency as the most impactful change.

**Independent Test**: Run `wave status --json`, `wave list --json`, `wave artifacts --json`, and pipe each through `jq .` — all must succeed without parse errors.

**Acceptance Scenarios**:

1. **Given** `wave status --json`, **When** output is piped through `jq`, **Then** the output is a single valid JSON object (structured, not NDJSON).
2. **Given** `wave list pipelines --json`, **When** the pipeline list is rendered, **Then** it emits a JSON array of pipeline objects (not table rows).
3. **Given** `wave artifacts --json <run-id>`, **When** artifacts exist, **Then** the output is a JSON array of artifact metadata objects.
4. **Given** `wave run --json <pipeline>`, **When** the pipeline runs, **Then** stdout contains only NDJSON events (one JSON object per line) and stderr contains only error messages (no progress text).
5. **Given** `wave logs --json <run-id>`, **When** logs are retrieved, **Then** each log entry is a JSON object on its own line (NDJSON).
6. **Given** any subcommand with `--json`, **When** the command fails, **Then** stderr contains a JSON object with `"error"`, `"code"`, and `"suggestion"` fields.

---

### User Story 3 — Color and Styling Control (Priority: P2)

A user on a terminal that does not support ANSI colors (or who prefers plain text for readability/accessibility) wants to disable all color output via a flag or environment variable without affecting functionality.

**Why this priority**: Color control is a clig.dev and NO_COLOR.org standard. The `NO_COLOR` env var is already implemented; this story adds the `--no-color` flag as an explicit alternative.

**Independent Test**: Run `wave status --no-color` and verify the output contains zero ANSI escape sequences.

**Acceptance Scenarios**:

1. **Given** the `--no-color` flag is passed, **When** any output is rendered (CLI or TUI), **Then** no ANSI escape sequences are present in stdout or stderr.
2. **Given** the `NO_COLOR` env var is set (to any non-empty value), **When** any command runs, **Then** color is disabled identically to `--no-color`.
3. **Given** `--no-color` is passed alongside `--no-tui`, **When** `wave run` executes, **Then** progress text renders without color codes.
4. **Given** `--no-color` is NOT passed and `NO_COLOR` is unset, **When** the terminal supports ANSI, **Then** colored output is rendered as normal.
5. **Given** `--no-color` is passed, **When** the TUI is active, **Then** the TUI renders without color attributes but preserves structural formatting (borders, layout, spacing). Lipgloss styles are applied with empty color values, maintaining the TUI structure in monochrome.

---

### User Story 4 — Quiet Mode for Scripting (Priority: P2)

A script author embedding `wave` in a larger automation wants to suppress all non-essential output (spinners, progress bars, status lines) and only see errors and final results.

**Why this priority**: Quiet mode is essential for CI/CD integration and piping. The `-o quiet` mode exists but lacks the `-q`/`--quiet` shorthand expected by clig.dev.

**Independent Test**: Run `wave run -q <pipeline>` and verify that only the final result (or error) appears in stderr, with no intermediate progress.

**Acceptance Scenarios**:

1. **Given** `-q` or `--quiet` is passed, **When** a pipeline runs, **Then** only the final completion/failure summary is emitted to stderr.
2. **Given** `--quiet` is passed, **When** the TUI would normally launch, **Then** the TUI is NOT launched (quiet implies non-interactive).
3. **Given** `--quiet` and `--json` are both passed, **When** a pipeline runs, **Then** `--json` controls stdout format (NDJSON events for streaming commands, structured JSON for non-streaming) while `--quiet` suppresses all non-essential stderr output (progress, warnings). Only fatal errors appear on stderr.
4. **Given** `--quiet` is passed to `wave status`, **When** the status is retrieved, **Then** only the essential status line is emitted (no table headers, no decorations).

---

### User Story 5 — Actionable Error Messages (Priority: P3)

A user encounters a failure (missing manifest, invalid pipeline name, adapter not found, contract violation). Instead of a cryptic error, they receive a clear message explaining what went wrong and what to do next.

**Why this priority**: Good error messages reduce support burden and improve developer experience. Less urgent than output mode mechanics but critical for polish.

**Independent Test**: Deliberately trigger known error conditions and verify each error message includes a suggestion.

**Acceptance Scenarios**:

1. **Given** `wave run nonexistent-pipeline`, **When** the pipeline is not found, **Then** the error message includes the pipeline name, lists available pipelines, and suggests `wave list pipelines`.
2. **Given** `wave run --manifest missing.yaml`, **When** the manifest file does not exist, **Then** the error message includes the file path and suggests `wave init`.
3. **Given** a pipeline step fails contract validation, **When** the error is reported, **Then** the message includes the step name, contract type, and a description of what was expected.
4. **Given** any error, **When** `--debug` is NOT set, **Then** no stack traces or internal Go error chains are shown.
5. **Given** any error with `--debug` set, **When** the error is displayed, **Then** it includes the full error chain and relevant debug context.
6. **Given** any error with `--json` set, **When** the error is emitted, **Then** it is a valid JSON object on stderr with `"error"`, `"code"`, and `"suggestion"` fields.

---

### User Story 6 — Output Stream Discipline (Priority: P3)

A user pipes `wave run --json <pipeline>` to `jq` and expects only clean JSON on stdout. Progress indicators, warnings, and informational messages must not pollute stdout.

**Why this priority**: Correct stream separation is required for reliable piping and is a clig.dev mandate. Most of this is already implemented; this story ensures completeness.

**Independent Test**: Run `wave run --json <pipeline> 2>/dev/null | jq .` and verify all lines parse as valid JSON.

**Acceptance Scenarios**:

1. **Given** any command with `--json`, **When** output is produced, **Then** stdout contains ONLY valid JSON (one object per line for streaming commands, a single JSON document for non-streaming commands), with no interleaved text.
2. **Given** any command without `--json`, **When** progress spinners or bars render, **Then** they render to stderr, never stdout.
3. **Given** `wave run` completes, **When** the final summary is shown, **Then** it appears at the END of stderr output (not buried in the middle).
4. **Given** `--verbose` is active, **When** tool call activity is logged, **Then** it goes to stderr.

---

### Edge Cases

- **Conflicting flags**: `--json` + `--output text` should produce a clear conflict error, not undefined behavior.
- **Piped stdout with no flags**: When stdout is not a TTY and no explicit output flag is set, the auto mode should fall back to text (not TUI) — this is already handled by TTY detection.
- **`TERM=dumb`**: Should disable both color and TUI, behaving as `--no-color --no-tui`.
- **`--no-color` in TUI mode**: TUI should still launch but render in monochrome (no color attributes, but structural formatting preserved).
- **Empty pipeline list in `--json` mode**: Should output `[]`, not an empty string or nothing.
- **Subcommand `--format` vs root `--json`**: Root `--json` takes precedence when both are present. Subcommand `--format` flags are retained for backward compatibility but are overridden by root-level output flags.
- **`--quiet` + `--verbose`**: `--quiet` wins — suppresses verbose output. If the user passes both, log a warning to stderr.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The CLI MUST provide `--json` as a persistent root flag that is equivalent to `--output json`.
- **FR-002**: The CLI MUST provide `-q`/`--quiet` as a persistent root flag that is equivalent to `--output quiet`.
- **FR-003**: The CLI MUST provide `--no-color` as a persistent root flag that disables all ANSI color and styling.
- **FR-004**: The `--no-color` flag MUST have identical behavior to setting the `NO_COLOR` environment variable. Implementation: `--no-color` sets the internal `colorMode` to `"off"`, which is the same code path as `NO_COLOR` detection in `SelectColorPalette`.
- **FR-005**: When `--json` is passed, every subcommand MUST produce valid, parseable JSON output on stdout. For streaming commands (`run`), this is NDJSON (one JSON object per line). For non-streaming commands (`status`, `list`, `logs`, `artifacts`, `cancel`), this is a single structured JSON document.
- **FR-006**: Subcommands that currently use `--format json` for local table/JSON toggling MUST respect the root `--json` flag as an override. The subcommand `--format` flags are retained for backward compatibility but are superseded when `--json` (or any root `--output` variant) is set. Both root `--json` and subcommand `--format json` produce the same structured JSON for non-streaming commands.
- **FR-007**: When `--quiet` is passed, all non-essential output (progress indicators, spinners, decorative formatting) MUST be suppressed.
- **FR-008**: When `--quiet` is passed, the TUI MUST NOT launch (quiet implies non-interactive).
- **FR-009**: Error messages MUST be actionable — every error MUST include a suggested next step or remediation.
- **FR-010**: Stack traces and internal error chains MUST only appear when `--debug` is set.
- **FR-011**: Progress indicators (spinners, bars, live updates) MUST render to stderr, never stdout.
- **FR-012**: Important summary information (completion status, timing, token usage) MUST appear at the END of output, not buried in the middle.
- **FR-013**: When `--json` is set and an error occurs, the error MUST be emitted as a JSON object to stderr with `"error"`, `"code"`, and `"suggestion"` fields. The `"code"` field provides a machine-parseable error classification (e.g. `"pipeline_not_found"`, `"manifest_missing"`, `"contract_violation"`).
- **FR-014**: When conflicting flags are passed (e.g. `--json` + `--output text`), the CLI MUST report a clear conflict error and exit with non-zero status.
- **FR-015**: All standard flags MUST be consistent and visible in `--help` output for every subcommand.

### Key Entities

- **OutputMode**: The resolved output behavior (auto, json, text, quiet) after considering `--output`, `--json`, `--quiet`, and their interactions. Resolution order: (1) check for conflicts, (2) `--json` sets json, (3) `--quiet` sets quiet, (4) `--output` value, (5) default auto.
- **ColorMode**: The resolved color behavior (auto, on, off) after considering `--no-color`, `NO_COLOR` env var, and terminal capabilities. `--no-color` flag maps to `colorMode = "off"`, which follows the existing `SelectColorPalette` code path.
- **ErrorResponse**: A structured error containing the error message (`"error"`), a machine-parseable error code (`"code"`), a human-readable suggestion (`"suggestion"`), and optionally debug details (`"debug"`, only when `--debug` is set).

## Clarifications

The following ambiguities were identified and resolved during the clarify step:

### C1: JSON Output Format for Non-Streaming vs Streaming Commands

**Ambiguity**: The spec originally described `--json` as producing "NDJSON" uniformly, but `wave status`, `wave list`, etc. are request-response commands where NDJSON (one event per line) doesn't apply. The existing `--format json` on these commands produces structured JSON (e.g., `{"runs": [...]}` for status).

**Resolution**: `--json` produces **structured JSON** for non-streaming commands and **NDJSON** for streaming commands (`wave run`). This matches the existing `--format json` behavior on subcommands and aligns with clig.dev's recommendation that JSON output be parseable by `jq`. Streaming commands naturally produce line-delimited events; non-streaming commands produce a single JSON document.

**Rationale**: NDJSON for `wave status` would break `jq .` compatibility (which expects a single document). The clig.dev spec says "if the output is primarily for machines, JSON" — it doesn't mandate NDJSON for all commands.

### C2: Error Response `code` Field

**Ambiguity**: FR-013 originally specified only `"error"` + `"suggestion"` fields, but US5-S6 referenced `"error"` + `"code"` + `"suggestion"`. The `code` field was inconsistently specified.

**Resolution**: The `"code"` field is **required** in structured JSON error responses. It provides a machine-parseable error classification string (e.g., `"pipeline_not_found"`, `"manifest_missing"`, `"contract_violation"`, `"flag_conflict"`, `"adapter_not_found"`). FR-013 and the ErrorResponse entity definition have been updated to include `"code"`.

**Rationale**: Machine consumers need stable error codes to programmatically handle different failure modes. This aligns with clig.dev's guidance on structured error output and follows patterns in tools like `gh` and `kubectl`.

### C3: `--quiet` + `--json` Combination Semantics

**Ambiguity**: US4-S3 described `--quiet` + `--json` but the semantics were unclear — if `--json` produces output on stdout, what does "quiet" suppress?

**Resolution**: `--json` and `--quiet` control **orthogonal concerns**. `--json` controls the **stdout format** (JSON output). `--quiet` controls **stderr verbosity** (suppresses progress, warnings, informational messages). When combined: stdout gets JSON output as normal, stderr is silent except for fatal errors. This is **not** a conflict — the combination is valid and useful for CI/CD scripts that want structured output with minimal noise.

**Rationale**: This follows the Unix convention where stdout is for data and stderr is for diagnostics. `--quiet` reduces diagnostic noise; `--json` structures the data channel. Tools like `curl -s -o - | jq` demonstrate this pattern.

### C4: `--no-color` Effect on TUI Rendering (Monochrome Mode)

**Ambiguity**: US3-S5 said "no lipgloss color styles applied" but lipgloss handles both color attributes and structural formatting (borders, padding, alignment). Stripping all lipgloss styling would destroy the TUI layout.

**Resolution**: "Monochrome mode" means **no color attributes** but **preserved structural formatting**. Lipgloss styles continue to apply borders, padding, width, and alignment, but all `Foreground()`, `Background()`, and `ColorProfile()` color values are cleared/empty. This is implemented by passing `colorMode = "off"` through to lipgloss renderers, which use `lipgloss.NoColor{}` as the color profile.

**Rationale**: This is the standard approach used by Charm ecosystem tools (e.g., `glow`, `soft-serve`). The lipgloss `HasDarkBackground` detection and `ColorProfile()` already support this via `lipgloss.Ascii` profile. Destroying layout would make the TUI unusable.

### C5: Subcommand `--format` Flag Coexistence with Root Flags

**Ambiguity**: The spec said root `--json` "overrides" subcommand `--format`, but didn't specify whether `--format` should be deprecated, removed, or continue to coexist.

**Resolution**: Subcommand `--format` flags are **retained for backward compatibility** and continue to work as before. When a root-level output flag (`--json`, `--quiet`, `--output`) is explicitly set, it takes precedence over the subcommand's `--format`. When no root output flag is set (i.e., `--output` remains at its default `auto`), the subcommand's `--format` controls output as it does today. No deprecation warnings are added in this iteration.

**Rationale**: Removing `--format` would be a breaking change for existing scripts. The precedence rule is simple: root flags are "global policy" and subcommand flags are "local preference" — global wins when explicitly set. This is the same pattern used by `kubectl` (`-o json` vs subcommand-specific flags).

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Running `wave <subcommand> --help` for ALL subcommands shows the standard persistent flag set (`--json`, `-q`/`--quiet`, `--no-color`, `--debug`, `--verbose`, `--no-tui`, `--output`).
- **SC-002**: `wave status --json | jq .` succeeds for every subcommand that produces output (`status`, `list`, `logs`, `artifacts`, `cancel`, `run`).
- **SC-003**: `wave run --json <pipeline> 2>/dev/null | jq -c .` parses every line — zero non-JSON lines on stdout.
- **SC-004**: `wave status --no-color | grep -P '\x1b\[' | wc -l` returns 0 — no ANSI escape sequences present.
- **SC-005**: `wave run -q <pipeline>` produces no output on stdout and only a final summary line on stderr.
- **SC-006**: Every error path tested includes a `"suggestion"` or human-readable remediation hint.
- **SC-007**: All existing tests pass with no regressions (`go test -race ./...`).
- **SC-008**: Flag conflict detection (`--json` + `--output text`) produces a non-zero exit code and clear message.
