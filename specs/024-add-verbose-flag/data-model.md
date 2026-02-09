# Data Model: Add --verbose Flag to Wave CLI

**Feature Branch**: `024-add-verbose-flag`
**Date**: 2026-02-06

## Entities

### VerbosityLevel (Conceptual — Not a Formal Type)

Per CLR-002, the effective verbosity is resolved at point of use from two independent booleans rather than a formal enum type.

| Level | Condition | Output Tier |
|-------|-----------|-------------|
| Normal | `!debug && !verbose` | Standard progress events, minimal output |
| Verbose | `!debug && verbose` | Operational context: workspace paths, artifacts, personas, contract details |
| Debug | `debug` (regardless of verbose) | Full internal traces, raw adapter commands, environment dumps |

**Precedence Rule**: `debug > verbose > normal` (FR-005)

---

### Modified Entities

#### 1. Event (internal/event/emitter.go)

**Current fields** (unchanged):
- `Timestamp`, `PipelineID`, `StepID`, `State`, `DurationMs`, `Message`
- `Persona`, `Artifacts`, `TokensUsed`
- `Progress`, `CurrentAction`, `TotalSteps`, `CompletedSteps`
- `EstimatedTimeMs`, `ValidationPhase`, `CompactionStats`

**New optional fields** (added for verbose output):

| Field | Type | JSON | Purpose |
|-------|------|------|---------|
| `WorkspacePath` | `string` | `workspace_path,omitempty` | Absolute path to step's workspace directory |
| `InjectedArtifacts` | `[]string` | `injected_artifacts,omitempty` | Names of artifacts injected into step workspace |
| `ContractResult` | `string` | `contract_result,omitempty` | Detailed contract validation result text |
| `VerboseDetail` | `string` | `verbose_detail,omitempty` | General-purpose verbose context message |

These fields are populated only when `executor.verbose` is true. When absent (empty/nil), they are omitted from JSON output via `omitempty` tags, ensuring zero regression for non-verbose output (FR-006).

---

#### 2. DefaultPipelineExecutor (internal/pipeline/executor.go)

**New field**:

| Field | Type | Default | Purpose |
|-------|------|---------|---------|
| `verbose` | `bool` | `false` | Controls whether verbose fields are populated on emitted events |

**New option function**:

```
WithVerbose(verbose bool) ExecutorOption
```

Follows the existing `WithDebug` pattern at executor.go:74-76.

---

#### 3. DisplayConfig (internal/display/types.go)

**Existing field** (to be wired):

| Field | Type | Current State | Change |
|-------|------|---------------|--------|
| `VerboseOutput` | `bool` | Defined but unused | Set from global `--verbose` flag in run command setup |

No structural change needed — just needs to be connected to the flag.

---

#### 4. RunOptions (cmd/wave/commands/run.go)

No change needed to the struct. The `verbose` bool is read from `cmd.Flags().GetBool("verbose")` (global persistent flag) and passed as a parameter to `runRun()`, following the same pattern as `debug`.

---

#### 5. StatusOptions (cmd/wave/commands/status.go)

No struct change needed. The verbose flag is read from the global persistent flag via `cmd.Flags().GetBool("verbose")` within the command's RunE function and passed to the display functions.

---

#### 6. CleanOptions (cmd/wave/commands/clean.go)

No struct change needed. The verbose flag is read from the global persistent flag via `cmd.Flags().GetBool("verbose")` within the command's RunE function. The clean command already has output formatting infrastructure (dry-run mode, quiet mode) where verbose output integrates naturally.

---

## Data Flow

```
┌─────────────┐     ┌──────────────┐     ┌───────────────────────┐
│  Root Cmd    │     │  Subcommand  │     │  Executor / Handler   │
│  --verbose   │────▶│  GetBool()   │────▶│  verbose bool field   │
│  persistent  │     │              │     │                       │
└─────────────┘     └──────────────┘     └───────────┬───────────┘
                                                      │
                                          ┌───────────▼───────────┐
                                          │  Event Emission       │
                                          │  (verbose fields set  │
                                          │   when verbose=true)  │
                                          └───────────┬───────────┘
                                                      │
                                    ┌─────────────────┼─────────────────┐
                                    │                                   │
                          ┌─────────▼─────────┐             ┌──────────▼──────────┐
                          │  NDJSONEmitter     │             │  ProgressEmitter    │
                          │  (stdout - JSON)   │             │  (stderr - display) │
                          │  includes verbose  │             │  renders verbose    │
                          │  fields in JSON    │             │  details visually   │
                          └────────────────────┘             └─────────────────────┘
```

## No Database Changes

This feature does not modify the SQLite schema. The verbose flag is a runtime-only setting that affects output rendering, not data persistence.

## No New Files/Packages

All changes fit within existing packages:
- `cmd/wave/main.go` — flag registration
- `cmd/wave/commands/run.go` — flag reading, executor option
- `cmd/wave/commands/status.go` — verbose output in handler
- `cmd/wave/commands/clean.go` — verbose output in handler
- `internal/pipeline/executor.go` — verbose field, WithVerbose option, event enrichment
- `internal/event/emitter.go` — new optional fields on Event struct
- `internal/display/types.go` — wire existing VerboseOutput field
