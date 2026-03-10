# Research: CLI Compliance Polish

## R1: Current Flag Architecture — How Persistent Flags Propagate in Cobra

**Decision**: Use Cobra's `PersistentFlags()` on rootCmd for `--json`, `--quiet`, `--no-color` as shorthand aliases that set existing internal state.

**Rationale**: Cobra's `PersistentFlags` are inherited by all subcommands automatically and appear in `--help` output for every subcommand. This is exactly what clig.dev requires. The existing `--output`, `--debug`, `--verbose`, `--no-tui` flags already use this mechanism successfully.

**Alternatives Rejected**:
- Per-subcommand flags: Would require duplication across 13+ subcommands, easy to forget one.
- Middleware/PersistentPreRunE: Not needed for flag registration, but we will use `PersistentPreRunE` for conflict detection.

**Key Finding**: Cobra persistent flags ARE inherited and visible in subcommand `--help`. Verified by inspecting current `--output`, `--verbose` behavior. The `cmd.Flags().Changed("flagname")` method distinguishes explicit user flags from defaults — critical for conflict detection.

## R2: Flag Conflict Detection — `--json` + `--output text`

**Decision**: Add a `ResolveOutputConfig()` function called in `PersistentPreRunE` on the root command that detects flag conflicts before command execution.

**Rationale**: The resolution must happen early and universally, not per-command. Using `PersistentPreRunE` ensures:
1. All subcommands get conflict detection without modification
2. Errors are reported before any execution begins
3. The resolution order is explicit: conflicts → `--json` → `--quiet` → `--output` → default

**Alternatives Rejected**:
- Per-command validation: Would miss new subcommands, duplicates logic.
- Silent precedence (last-flag-wins): clig.dev recommends explicit errors for conflicts.

**Key Finding**: `cmd.Flags().Changed("flagname")` on `cmd.Root().PersistentFlags()` detects whether the user explicitly passed a flag vs the flag taking its default value. This is how we detect `--json` + `--output text` conflicts.

## R3: `--no-color` Implementation — Interaction with Lipgloss and Existing NO_COLOR

**Decision**: `--no-color` sets `os.Setenv("NO_COLOR", "1")` early in `PersistentPreRunE`, feeding into the existing `SelectColorPalette`/`DetectANSISupport` code paths. For TUI, lipgloss reads `NO_COLOR` natively.

**Rationale**: The `internal/display/capability.go` already handles `NO_COLOR` env var by returning `AsciiOnlyColorScheme`. Adding `--no-color` is a single flag that converts to the same env var. Lipgloss v1.0+ reads `NO_COLOR` automatically — structural formatting (borders, padding, alignment) remains intact.

**Alternatives Rejected**:
- Custom color stripping middleware: Over-engineered; lipgloss and ANSICodec already respect `NO_COLOR`.
- New `ColorConfig` struct: Unnecessary abstraction — the existing `colorMode string` parameter handles it.

**Key Finding**: `SelectColorPalette` with `colorMode="auto"` already checks `os.Getenv("NO_COLOR")`. Setting the env var in `PersistentPreRunE` ensures all downstream code (display, TUI, lipgloss) sees it. TUI monochrome mode preserves layout because lipgloss separates color attributes from structural formatting.

## R4: Subcommand `--format` vs Root `--json` Precedence

**Decision**: Introduce a `ResolveFormat()` helper that each subcommand calls. If root `--json`/`--quiet`/`--output` was explicitly changed (via `cmd.Root().PersistentFlags().Changed()`), the root flag wins. Otherwise, local `--format` applies.

**Rationale**: Five subcommands have local `--format` flags: `status`, `list`, `logs`, `artifacts`, `cancel`. Each reads `opts.Format` directly. The fix: each command calls `ResolveFormat(cmd, localFormat)` which checks root flag priority.

**Alternatives Rejected**:
- Removing `--format` from subcommands: Breaking change for existing scripts.
- Making subcommand `--format` override root: Violates the "global policy wins" pattern.

**Key Finding**: The five subcommands with `--format` all use the same conditional: `opts.Format == "json"`. Updating them to call a shared resolver is mechanical and low-risk.

## R5: Error Response Structure — `ErrorResponse` JSON

**Decision**: Introduce a `CLIError` type with `Error`, `Code`, `Suggestion`, and optional `Debug` fields. Render as JSON to stderr when `--json` is active.

**Rationale**: Currently, errors are rendered by Cobra's default handler or `fmt.Errorf`. For JSON mode:
1. Define `CLIError` struct with JSON tags in a new `cmd/wave/commands/errors.go`
2. In main.go's error handler, detect JSON mode → format as JSON on stderr
3. Map known error types to error codes using the existing `recovery.ErrorClass` types

**Alternatives Rejected**:
- Per-command JSON error formatting: Would miss errors from Cobra itself (flag parsing errors).
- Go error wrapping chains: Too generic; need structured codes and suggestions.

**Key Finding**: The `recovery` package already has `ErrorClass` types (`contract_validation`, `security_violation`, `preflight`, `runtime_error`) that map directly to error codes. Extending this with CLI-specific codes (`pipeline_not_found`, `manifest_missing`, `flag_conflict`) provides complete coverage.

## R6: Output Stream Discipline — stdout vs stderr Audit

**Decision**: Audit all `fmt.Printf`/`fmt.Println` calls in commands package; route progress/informational output to stderr, keep data output on stdout.

**Rationale**: 188 print calls across 12 command files. Key findings:
- `run.go`: Already correct — NDJSON on stdout, progress on stderr
- `status.go`: Table output on stdout (correct for data), hardcoded ANSI color codes need `--no-color` respect
- `list.go`: Table output on stdout (correct)
- `clean.go`: Progress messages use `fmt.Printf` to stdout — should be stderr
- `artifacts.go`: Table output on stdout (correct), informational messages should be stderr
- `logs.go`: Performance summary uses stdout — should be stderr
- `cancel.go`: Result output on stdout (correct for data)

The fix is targeted, not wholesale — move progress/informational messages to stderr.

**Key Finding**: `status.go` has hardcoded ANSI color constants (`colorReset`, `colorRed`, `colorGreen`, `colorYellow`, `colorGray`). These bypass the `SelectColorPalette`/`NO_COLOR` system and need to use the shared color infrastructure or be gated on `NO_COLOR`.

## R7: `--quiet` as Root Persistent Flag

**Decision**: Add `-q`/`--quiet` as persistent root flag equivalent to `--output quiet`. For non-streaming commands, quiet mode shows only essential data (no headers, no decorations).

**Rationale**: `clean --quiet` is command-local. The root `--quiet` maps to `--output quiet` which already has full emitter support. Combined with `--json`, `--quiet` only suppresses stderr (orthogonal concerns per spec clarification C3).

**Alternatives Rejected**:
- Keeping `--quiet` per-command: Inconsistent with clig.dev.
- Making `--quiet` suppress all output: Would be useless — equivalent to `>/dev/null`.

## R8: `TERM=dumb` and Quiet+Verbose Conflict

**Decision**: `TERM=dumb` triggers `--no-color --no-tui` equivalence. `--quiet` + `--verbose` → quiet wins, log warning.

**Rationale**: `shouldLaunchTUI()` already handles `TERM=dumb`. `DetectANSISupport()` returns false for `TERM=dumb`. Color is already disabled via this path. Gap: ensure explicit `NO_COLOR` is set for downstream code. For `--quiet` + `--verbose` conflict: spec says quiet wins with a warning.

**Key Finding**: Current code handles TERM=dumb correctly for both TUI and color (via `DetectANSISupport`). Only gap is ensuring `--quiet` + `--verbose` warning is logged.
