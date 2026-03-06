# Data Model: CLI Compliance Polish

## Entities

### OutputConfig (Modified)

**Location**: `cmd/wave/commands/output.go`

Current:
```go
type OutputConfig struct {
    Format  string
    Verbose bool
}
```

Updated:
```go
type OutputConfig struct {
    Format   string // "auto", "json", "text", "quiet"
    Verbose  bool
    NoColor  bool   // NEW — true when --no-color flag or NO_COLOR env var is set
    Debug    bool   // NEW — true when --debug flag is set (for error detail inclusion)
}
```

**Changes**: Add `NoColor` and `Debug` fields to carry resolved flag state through command execution. This avoids each command re-reading root flags independently.

### CLIError (New)

**Location**: `cmd/wave/commands/errors.go` (new file)

```go
// CLIError represents a structured error for CLI output.
// In JSON mode, this is rendered as a JSON object to stderr.
// In text mode, it renders as a human-readable error with suggestion.
type CLIError struct {
    Message    string `json:"error"`
    Code       string `json:"code"`
    Suggestion string `json:"suggestion"`
    Debug      string `json:"debug,omitempty"` // Only populated when --debug is set
}

func (e *CLIError) Error() string {
    return e.Message
}
```

**Error Codes** (machine-parseable classification):
| Code | Trigger |
|------|---------|
| `pipeline_not_found` | Pipeline name doesn't match any in `.wave/pipelines/` |
| `manifest_missing` | `wave.yaml` file not found |
| `manifest_invalid` | YAML parse error in wave.yaml |
| `contract_violation` | Step output fails contract validation |
| `adapter_not_found` | Adapter binary not on PATH |
| `flag_conflict` | Conflicting flags (e.g., `--json` + `--output text`) |
| `onboarding_required` | `wave init` not completed |
| `step_not_found` | `--from-step` references unknown step |
| `run_not_found` | `wave status <run-id>` with invalid ID |
| `preflight_failed` | Required skills/tools missing |
| `timeout` | Step or pipeline exceeds configured timeout |
| `cancelled` | Pipeline was cancelled by user |
| `internal_error` | Unexpected internal error |

### ResolvedFlags (New — Internal)

**Location**: `cmd/wave/commands/output.go` (added to existing file)

```go
// ResolvedFlags captures the full resolved state from PersistentPreRunE.
// Stored in cobra.Command context for downstream access.
type ResolvedFlags struct {
    Output  OutputConfig
    NoTUI   bool
}
```

This is computed once in `PersistentPreRunE` and stored in the command context. All subcommands read from this instead of re-parsing root flags.

## Data Flow

### Flag Resolution (PersistentPreRunE on rootCmd)

```
User Input: --json --quiet --no-color --output <val> --verbose --debug --no-tui
                |       |        |          |          |         |        |
                v       v        v          v          v         v        v
         ┌─────────────────────────────────────────────────────────────────────┐
         │                    ResolveOutputConfig()                            │
         │                                                                     │
         │  1. Check conflicts:                                                │
         │     --json + --output text → error "flag_conflict"                  │
         │     --json + --quiet (OK — orthogonal)                              │
         │     --quiet + --verbose → warn stderr, quiet wins                   │
         │                                                                     │
         │  2. Resolve Format:                                                 │
         │     if --json explicitly set → Format = "json"                      │
         │     else if --quiet explicitly set → Format = "quiet"               │
         │     else → Format = --output value (default "auto")                 │
         │                                                                     │
         │  3. Resolve NoColor:                                                │
         │     if --no-color flag → os.Setenv("NO_COLOR", "1")                │
         │     (NO_COLOR env already handled by display package)               │
         │                                                                     │
         │  4. Resolve NoTUI:                                                  │
         │     if --quiet → noTUI = true (quiet implies non-interactive)       │
         │     if --json → noTUI = true (json output excludes TUI)             │
         │                                                                     │
         └─────────────────────────────────────────────────────────────────────┘
                                      │
                                      v
                              ResolvedFlags stored
                              in cmd context
                                      │
                     ┌────────────────┼────────────────┐
                     │                │                │
                     v                v                v
              Subcommand RunE   Error Handler    TUI Decision
              reads config      renders JSON     shouldLaunchTUI()
```

### Subcommand Format Resolution

```
For subcommands with local --format flag (status, list, logs, artifacts, cancel):

  ResolveFormat(rootConfig, localFormat) → string
    │
    ├── if rootConfig.Format explicitly changed (--json, --quiet, --output)
    │   └── return rootConfig.Format mapping:
    │       "json" → "json"
    │       "quiet" → (suppress decorations, minimal output)
    │       "text" → "table"
    │
    └── else (root at default "auto")
        └── return localFormat as-is
```

### Error Rendering

```
Error occurs in any command
         │
         v
  main.go catches error
         │
         ├── Is it a *CLIError?
         │   ├── JSON mode → json.Marshal to stderr
         │   └── Text mode → "Error: <msg>\n  Suggestion: <suggestion>"
         │
         └── Is it a plain error?
             ├── JSON mode → wrap as CLIError{Message: err.Error(), Code: "internal_error"}
             └── Text mode → existing behavior (fmt.Fprintln(os.Stderr, err))
```

## Relationship to Existing Entities

| Existing Entity | Interaction |
|----------------|-------------|
| `OutputConfig` | Extended with `NoColor`, `Debug` fields |
| `SelectColorPalette()` | No change — already handles `NO_COLOR` env var |
| `CreateEmitter()` | No change — already handles all format modes |
| `shouldLaunchTUI()` | Add `--quiet` and `--json` as TUI disablers |
| `recovery.ErrorClass` | Maps to `CLIError.Code` for pipeline errors |
| `recovery.RecoveryBlock` | Recovery hints map to `CLIError.Suggestion` |
| `display.ANSICodec` | No change — already respects `colorMode` |
| `status.statusColor()` | Must check `NO_COLOR` before emitting ANSI codes |
