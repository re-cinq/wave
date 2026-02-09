# Research: Add --verbose Flag to Wave CLI

**Feature Branch**: `024-add-verbose-flag`
**Date**: 2026-02-06

## Phase 0 — Unknowns & Research Findings

### RES-001: Flag Registration Pattern (Global Persistent Flag)

**Decision**: Use `rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")` on the root command, mirroring the existing `--debug/-d` pattern.

**Rationale**: The codebase already registers `--debug` as a persistent flag in `cmd/wave/main.go:31` using `BoolP("debug", "d", false, "Enable debug mode")`. Following the identical pattern ensures consistency and leverages Cobra's built-in persistent flag propagation to all subcommands.

**Alternatives Rejected**:
- *Formal VerbosityLevel enum*: Adds unnecessary abstraction for what is two independent booleans with a simple precedence rule. The spec (CLR-002) explicitly chose bool-threading over an enum.
- *Per-command local flags*: Would require registering the flag on every subcommand individually, violating DRY and the spec requirement (FR-001) that `--verbose` be a global persistent flag.

**Key Code Locations**:
- Root command definition: `cmd/wave/main.go:26-34`
- Debug flag registration: `cmd/wave/main.go:31`
- Debug flag read in run command: `cmd/wave/commands/run.go:69`

---

### RES-002: Flag Propagation Through Executor Options

**Decision**: Add `verbose bool` field to `DefaultPipelineExecutor` struct and a `WithVerbose(bool) ExecutorOption` function, following the existing `WithDebug` pattern exactly.

**Rationale**: The executor already uses a functional options pattern (`ExecutorOption func(*DefaultPipelineExecutor)`). The `WithDebug` option at `internal/pipeline/executor.go:74-76` provides the exact template. Adding `WithVerbose` alongside it maintains consistency.

**Propagation Chain**:
1. Root flag → `cmd.Flags().GetBool("verbose")` in command handler
2. Command handler → pass `verbose` parameter to `runRun()`
3. `runRun()` → `pipeline.WithVerbose(verbose)` in executor options
4. Executor → `e.verbose` field used at event emission points

**Alternatives Rejected**:
- *Context-based propagation*: More complex, unnecessary given the existing bool-parameter pattern works well.
- *Environment variable*: Would bypass CLI flag semantics and break composability with `--debug`.

**Key Code Locations**:
- Executor struct: `internal/pipeline/executor.go:41-58` (debug field at line 50)
- WithDebug option: `internal/pipeline/executor.go:74-76`
- Executor options construction: `cmd/wave/commands/run.go:219-231`

---

### RES-003: Verbose Output Routing via Event System

**Decision**: Extend existing event emission points in the executor with additional detail when `e.verbose` is true. Add optional verbose fields to the `Event` struct rather than creating a separate verbose output path.

**Rationale**: The codebase uses a dual-stream event architecture (FR-008):
- `NDJSONEmitter` writes structured JSON to stdout
- `ProgressEmitter` renders human-readable progress to stderr
- Both consume the same `Event` struct

Adding verbose data as optional fields on `Event` ensures it flows through both streams automatically. The spec (CLR-005) explicitly chose this approach over a parallel output path.

**Fields to Add to Event struct**:
- `WorkspacePath string` — step workspace directory
- `InjectedArtifacts []string` — artifact names injected into step
- `ContractResult string` — pass/fail detail for contract validation
- `VerboseDetail string` — general-purpose verbose message field

**Alternatives Rejected**:
- *Direct fmt.Printf*: Bypasses the event system, breaks `--no-logs` interaction, and doesn't appear in NDJSON output.
- *Separate verbose emitter*: Adds complexity without benefit since the existing emitter architecture already supports optional fields.

**Key Code Locations**:
- Event struct: `internal/event/emitter.go:11-30`
- Human-readable formatting: `internal/event/emitter.go:133-193`
- Event emission in executor: `internal/pipeline/executor.go:381-389, 438-467, 522-574`

---

### RES-004: `-v` Shorthand Conflict Resolution

**Decision**: Keep both `-v` registrations (global persistent and validate local). Cobra resolves this correctly — local flags shadow persistent flags on the specific subcommand where they're defined.

**Rationale**: The spec (CLR-001, FR-007) explicitly documents this Cobra behavior. Since both the global `--verbose` and validate's local `--verbose` activate the same behavior, users get consistent results regardless of flag placement:
- `wave validate -v` → local flag resolves, verbose on
- `wave -v validate` → global flag resolves, verbose on

**Verification**: The validate command already registers `-v` at `cmd/wave/commands/validate.go:35`. Cobra's `PersistentFlags` vs `Flags` resolution is well-documented and tested behavior.

**Key Code Locations**:
- Validate local flag: `cmd/wave/commands/validate.go:35`
- Validate verbose usage: `cmd/wave/commands/validate.go:42-43, 59, 86, 98, 109, 111-117`

---

### RES-005: Verbose Output Content per Command

**Decision**: Implement verbose output for exactly 4 commands (per FR-004 scope):

| Command | Verbose Content | Output Method |
|---------|----------------|---------------|
| `run` (P1) | Workspace paths, injected artifacts, persona names, contract results | Event system (verbose fields on existing events) |
| `validate` (P2) | Already implemented — validator details, per-section results | Existing `if opts.Verbose` pattern (already works) |
| `status` (P2) | Database path, state transition timestamps, workspace locations | Direct `fmt.Fprintf(os.Stderr, ...)` in command handler |
| `clean` (P2) | Each workspace listed with size before deletion | Direct `fmt.Fprintf(os.Stderr, ...)` in command handler |

**Rationale**: The `run` command uses the event system because it runs pipelines through the executor. Non-pipeline commands (`validate`, `status`, `clean`) operate at the command handler level and emit verbose info directly, matching `validate`'s existing pattern.

**Key Code Locations**:
- Validate verbose (existing): `cmd/wave/commands/validate.go:42-117`
- Status display: `cmd/wave/commands/status.go:122-172`
- Clean workspace listing: `cmd/wave/commands/clean.go:254-260`

---

### RES-006: DisplayConfig.VerboseOutput Pre-existing Field

**Decision**: Wire the existing `DisplayConfig.VerboseOutput` field (already defined at `internal/display/types.go:101`) to the global verbose flag so that progress display renderers can conditionally show verbose detail.

**Rationale**: The `VerboseOutput` field already exists in the display config but is not currently set. This is infrastructure that was anticipated for this feature. Connecting it to the global flag requires minimal changes.

**Key Code Locations**:
- DisplayConfig.VerboseOutput: `internal/display/types.go:101`
- Display setup in run: `cmd/wave/commands/run.go:139-181`

---

### RES-007: Debug Supersedes Verbose (Precedence)

**Decision**: Resolve effective verbosity at point of use with simple precedence: `if debug { debug_output } else if verbose { verbose_output } else { normal_output }`.

**Rationale**: Per FR-005, when both `--debug` and `--verbose` are active, the system uses the higher detail level (debug). This is a simple conditional that doesn't require a formal verbosity level type. Debug already outputs via `fmt.Printf("[DEBUG] ...")` at specific points in the executor — these remain unchanged. Verbose adds a new middle tier of output.

**Key Code Locations**:
- Debug prints in executor: `internal/pipeline/executor.go:489-491, 837-839`

---

### RES-008: Testing Strategy

**Decision**: Follow the existing test patterns in the codebase:
- **Unit tests** for flag registration and propagation (table-driven)
- **Unit tests** for verbose output in each command (capture stdout/stderr with `os.Pipe()`)
- **Integration-style tests** for event system verbose fields

**Existing Test Patterns**:
- Validate verbose tests: `cmd/wave/commands/validate_test.go:225-268` — uses `cmd.SetArgs([]string{"--verbose"})` and captures stdout
- Status tests: `cmd/wave/commands/status_test.go:126-148` — uses `executeStatusCmd()` helper
- Clean tests: `cmd/wave/commands/clean_test.go:180-205` — uses `executeCleanCmd()` helper
- Helpers tests: `cmd/wave/commands/helpers_test.go` — table-driven tests

**Key Files for Test Reference**:
- `cmd/wave/commands/validate_test.go` (888 lines)
- `cmd/wave/commands/status_test.go` (482 lines)
- `cmd/wave/commands/clean_test.go` (861 lines)
