# Implementation Plan: CLI Compliance Polish

**Branch**: `260-tui-cli-compliance` | **Date**: 2026-03-06 | **Spec**: `specs/260-tui-cli-compliance/spec.md`
**Input**: Feature specification from `/specs/260-tui-cli-compliance/spec.md`

## Summary

Standardize Wave's CLI surface for clig.dev compliance by adding `--json`, `-q`/`--quiet`, and `--no-color` as persistent root flag aliases, implementing flag conflict detection in `PersistentPreRunE`, structured JSON error responses with error codes and suggestions, and enforcing output stream discipline (data→stdout, progress→stderr). This builds on the existing `--output` flag infrastructure and `NO_COLOR` env var support.

## Technical Context

**Language/Version**: Go 1.25+ (existing project)
**Primary Dependencies**: `github.com/spf13/cobra` (CLI), `gopkg.in/yaml.v3` (config) — all existing
**Storage**: SQLite via `modernc.org/sqlite` (existing, for error context)
**Testing**: `go test` with `testify/assert`, `testify/require`
**Target Platform**: Linux/macOS terminal
**Project Type**: Single Go binary — changes in `cmd/wave/` and `internal/display/`
**Constraints**: No new external dependencies; must not break existing tests (`go test -race ./...`)

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | ✅ Pass | No new runtime dependencies. All changes use existing stdlib and cobra. |
| P2: Manifest as SSOT | ✅ Pass | No new config files. Flag resolution is pure CLI logic. |
| P3: Persona-Scoped Execution | N/A | CLI flag handling, not pipeline execution. |
| P4: Fresh Memory at Step Boundary | N/A | CLI flag handling, not pipeline steps. |
| P5: Navigator-First Architecture | N/A | CLI command infrastructure, not pipeline. |
| P6: Contracts at Every Handover | N/A | No pipeline step handovers. |
| P7: Relay via Dedicated Summarizer | N/A | CLI command infrastructure. |
| P8: Ephemeral Workspaces | N/A | No workspace changes. |
| P9: Credentials Never Touch Disk | ✅ Pass | No credential handling. |
| P10: Observable Progress | ✅ Pass | Progress output correctly routed to stderr. JSON error output enhances observability. |
| P11: Bounded Recursion | N/A | No pipeline execution changes. |
| P12: Minimal Step State Machine | N/A | No step state changes. |
| P13: Test Ownership | ✅ Pass | All new code will have tests; existing tests must continue to pass. |

No violations. No complexity tracking entries needed.

## Project Structure

### Documentation (this feature)

```
specs/260-tui-cli-compliance/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
cmd/wave/
├── main.go                          # MODIFY — add PersistentPreRunE, JSON error handler
└── commands/
    ├── output.go                    # MODIFY — extend OutputConfig, add ResolveOutputConfig(), ResolveFormat()
    ├── output_test.go               # MODIFY — tests for conflict detection, resolution logic
    ├── errors.go                    # NEW — CLIError type, error code constants, formatters
    ├── errors_test.go               # NEW — tests for error formatting in JSON/text modes
    ├── run.go                       # MODIFY — use resolved config instead of GetOutputConfig()
    ├── status.go                    # MODIFY — use NO_COLOR-aware color, use ResolveFormat()
    ├── status_test.go               # MODIFY — add JSON format and --no-color tests
    ├── list.go                      # MODIFY — use ResolveFormat() for root flag precedence
    ├── list_test.go                 # MODIFY — add root --json override test
    ├── logs.go                      # MODIFY — use ResolveFormat(), move summary to stderr
    ├── logs_test.go                 # MODIFY — add format resolution tests
    ├── artifacts.go                 # MODIFY — use ResolveFormat()
    ├── artifacts_test.go            # MODIFY — add format resolution tests
    ├── cancel.go                    # MODIFY — use ResolveFormat()
    ├── cancel_test.go               # MODIFY — add format resolution tests
    ├── clean.go                     # MODIFY — move progress to stderr, check root --quiet
    └── clean_test.go                # MODIFY — add stderr output verification
```

**Structure Decision**: All changes are within existing `cmd/wave/` structure. One new file (`errors.go`) for the `CLIError` type and error code constants. No new packages.

## Implementation Approach

### Phase 1: Root Flag Registration and Conflict Detection

**Files**: `cmd/wave/main.go` (MODIFY), `cmd/wave/commands/output.go` (MODIFY), `cmd/wave/commands/output_test.go` (MODIFY)

1. **Register new persistent root flags** in `main.go init()`:
   - `--json` (bool): shorthand for `--output json`
   - `-q`/`--quiet` (bool): shorthand for `--output quiet`
   - `--no-color` (bool): disables all ANSI color and styling

2. **Add `PersistentPreRunE`** to `rootCmd` in `main.go`:
   - Call `commands.ResolveOutputConfig(cmd)` which:
     a. Reads all flag states using `cmd.Root().PersistentFlags()`
     b. Uses `.Changed()` to detect explicit flags
     c. Detects conflicts:
        - `--json` + `--output` (with non-default, non-json value) → `CLIError{Code: "flag_conflict"}`
        - `--quiet` + `--output` (with non-default, non-quiet value) → `CLIError{Code: "flag_conflict"}`
        - `--json` + `--quiet` → OK (orthogonal: json=stdout format, quiet=stderr verbosity)
        - `--quiet` + `--verbose` → log warning to stderr, quiet wins
     d. Resolves final `OutputConfig`:
        - If `--json` → Format = "json"
        - Else if `--quiet` → Format = "quiet"
        - Else → Format = `--output` value (default "auto")
     e. If `--no-color` → `os.Setenv("NO_COLOR", "1")`
     f. If `--json` or `--quiet` → set `shouldLaunchTUI` override (noTUI = true)
   - Store resolved config in cmd context (via `cmd.SetContext()`)

3. **Update `GetOutputConfig()`** in `output.go`:
   - Read resolved config from cmd context first
   - Fall back to existing flag reading if context is empty (for tests)

4. **Tests** (SC-008, FR-014):
   - `--json` alone → Format = "json"
   - `--quiet` alone → Format = "quiet"
   - `--json` + `--output text` → error with code "flag_conflict"
   - `--quiet` + `--output json` → error with code "flag_conflict"
   - `--json` + `--quiet` → no error, Format = "json" (json wins for format, quiet for stderr)
   - `--no-color` → `NO_COLOR` env var set
   - `--quiet` + `--verbose` → quiet wins with warning

### Phase 2: CLIError Type and JSON Error Rendering

**Files**: `cmd/wave/commands/errors.go` (NEW), `cmd/wave/commands/errors_test.go` (NEW), `cmd/wave/main.go` (MODIFY)

1. **Create `CLIError`** in `errors.go`:
   - Fields: `Message string`, `Code string`, `Suggestion string`, `Debug string`
   - Implements `error` interface
   - Constructor: `NewCLIError(code, message, suggestion string) *CLIError`
   - JSON rendering: `RenderJSONError(w io.Writer, err error, debug bool)` — handles both `*CLIError` and plain `error`
   - Error code constants matching data model

2. **Update main.go error handler**:
   - After `rootCmd.Execute()` returns error:
     a. Read resolved output format from root flags
     b. If JSON mode: call `RenderJSONError(os.Stderr, err, debug)`
     c. Else: existing behavior + append suggestion if `*CLIError`
   - Replace plain `fmt.Fprintln(os.Stderr, err)` with format-aware rendering

3. **Tests** (FR-013, SC-006):
   - `CLIError` renders as valid JSON with error/code/suggestion fields
   - Plain error wraps as `CLIError{Code: "internal_error"}`
   - Debug details included only when `--debug` is set
   - JSON output on stderr, not stdout

### Phase 3: Actionable Error Messages

**Files**: `cmd/wave/commands/run.go` (MODIFY), `cmd/wave/commands/status.go` (MODIFY), `cmd/wave/commands/list.go` (MODIFY), various command files

1. **Wrap key error paths** with `CLIError`:
   - `loadPipeline()`: pipeline not found → `CLIError{Code: "pipeline_not_found", Suggestion: "Run 'wave list pipelines' to see available pipelines"}`
   - Manifest read error → `CLIError{Code: "manifest_missing", Suggestion: "Run 'wave init' to create a manifest"}`
   - Manifest parse error → `CLIError{Code: "manifest_invalid", Suggestion: "Check wave.yaml syntax"}`
   - Onboarding check → `CLIError{Code: "onboarding_required", Suggestion: "Run 'wave init'"}`
   - `--from-step` with unknown step → `CLIError{Code: "step_not_found", Suggestion: "Run 'wave run <pipeline> --dry-run' to see available steps"}`

2. **Enhance recovery hints** to populate `CLIError.Suggestion` when in JSON mode (integrate with existing `recovery.BuildRecoveryBlock()`)

3. **Tests** (FR-009, FR-010, SC-006):
   - Each error path returns actionable suggestion
   - `--debug` includes error chain, without `--debug` it's hidden
   - JSON errors parse as valid JSON with required fields

### Phase 4: Subcommand Format Resolution and `--no-color` for Status

**Files**: `cmd/wave/commands/output.go` (MODIFY), `cmd/wave/commands/status.go` (MODIFY), `cmd/wave/commands/list.go` (MODIFY), `cmd/wave/commands/logs.go` (MODIFY), `cmd/wave/commands/artifacts.go` (MODIFY), `cmd/wave/commands/cancel.go` (MODIFY)

1. **Add `ResolveFormat()`** in `output.go`:
   ```go
   func ResolveFormat(cmd *cobra.Command, localFormat string) string
   ```
   - If root `--json` was explicitly set → return "json"
   - If root `--quiet` was explicitly set → return "quiet" (mapped to minimal output)
   - If root `--output` was explicitly set to non-default → return its value
   - Otherwise → return `localFormat` unchanged

2. **Update each subcommand** with local `--format`:
   - `status.go`: Replace `opts.Format` reads with `ResolveFormat(cmd, opts.Format)` call in RunE
   - `list.go`: Same pattern
   - `logs.go`: Same pattern
   - `artifacts.go`: Same pattern
   - `cancel.go`: Same pattern

3. **Fix `status.go` hardcoded ANSI colors**:
   - Replace `colorReset`, `colorRed`, etc. constants with calls that check `NO_COLOR` env var
   - Use existing `display.ANSICodec` or gate colors on `os.Getenv("NO_COLOR") == ""`
   - Move status table header to use conditional color

4. **Tests** (FR-006, SC-001, SC-004):
   - Root `--json` overrides subcommand `--format table`
   - Root `--quiet` overrides subcommand `--format json`
   - Default root → subcommand `--format` preserved
   - `--no-color` produces zero ANSI escapes in status output

### Phase 5: Output Stream Discipline

**Files**: `cmd/wave/commands/clean.go` (MODIFY), `cmd/wave/commands/logs.go` (MODIFY), `cmd/wave/commands/artifacts.go` (MODIFY), `cmd/wave/commands/status.go` (MODIFY)

1. **Route progress/informational messages to stderr**:
   - `clean.go`: Change `fmt.Printf("Nothing to clean\n")` → `fmt.Fprintf(os.Stderr, ...)`
   - `clean.go`: Change all progress messages (removed, failed, progress) → stderr
   - `logs.go`: Move performance summary rendering → stderr
   - `artifacts.go`: Move "No artifacts found", "Artifacts for run:" messages → stderr when not in JSON mode
   - `status.go`: Move "No pipelines found", "No running pipelines" → stderr

2. **Keep data output on stdout**:
   - JSON output from all commands → stdout (already correct)
   - Table data → stdout (already correct)

3. **Ensure quiet mode suppresses non-essential output**:
   - When format is "quiet", skip headers, decorations, and informational messages
   - Only emit essential data (error messages, final result line)

4. **Tests** (FR-011, FR-012, SC-003, SC-005):
   - `wave status --json | jq .` succeeds (no non-JSON on stdout)
   - Progress messages not on stdout
   - Quiet mode suppresses all non-essential stderr

### Phase 6: shouldLaunchTUI Updates and Edge Cases

**Files**: `cmd/wave/main.go` (MODIFY)

1. **Update `shouldLaunchTUI()`**:
   - Add check: if `--json` → return false
   - Add check: if `--quiet` → return false
   - Existing checks: `--no-tui`, `WAVE_FORCE_TTY`, `TERM=dumb`, TTY detection

2. **Edge cases**:
   - `TERM=dumb`: Already handled for TUI. Color already handled via `DetectANSISupport()`. No additional changes needed.
   - Empty output in JSON mode: Commands must output `[]` or `{}`, never empty string (already handled by most commands)
   - `clean --quiet` coexistence: Root `--quiet` sets output format; `clean --quiet` boolean is a separate flag. Both paths lead to suppressed output. No conflict.

3. **Tests** (edge cases from spec):
   - `--json` prevents TUI launch
   - `--quiet` prevents TUI launch
   - `TERM=dumb` prevents TUI and color

## Complexity Tracking

_No constitution violations. No complexity tracking entries needed._
