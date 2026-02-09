# Implementation Plan: Add --verbose Flag to Wave CLI

**Branch**: `024-add-verbose-flag` | **Date**: 2026-02-06 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/024-add-verbose-flag/spec.md`

## Summary

Add a global `--verbose/-v` persistent flag to the Wave CLI that provides a middle tier of output detail between normal and `--debug` modes. When active, verbose mode shows operational context (workspace paths, injected artifacts, persona names, contract validation results) during pipeline execution and additional command-specific details for `validate`, `status`, and `clean`. The implementation follows the existing `--debug` flag propagation pattern: register on root command, thread through executor options, and emit verbose-enriched events through the dual-stream event system.

## Technical Context

**Language/Version**: Go 1.25+ (existing Wave project)
**Primary Dependencies**: github.com/spf13/cobra (CLI framework), gopkg.in/yaml.v3
**Storage**: N/A (no database changes — verbose is runtime-only)
**Testing**: `go test ./...` with `-race` flag, table-driven tests
**Target Platform**: Linux/macOS (single static binary)
**Project Type**: Single Go binary (CLI tool)
**Performance Goals**: No measurable performance impact — verbose adds conditional event field population only
**Constraints**: Zero regression on non-verbose output (FR-006), single static binary (no new dependencies)
**Scale/Scope**: ~7 files modified, ~200-300 lines added across implementation and tests

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Assessment |
|-----------|--------|------------|
| I. Minimal Operational Overhead | ✅ Pass | No new infrastructure or dependencies. Single bool flag threaded through existing patterns. |
| II. Multi-Tenant Security & Isolation | ✅ N/A | This feature does not touch tenant data, authentication, or authorization. |
| III. Cost Efficiency & Pay-Per-Use | ✅ Pass | No cost impact — verbose is a local output formatting concern. |
| IV. Fast Time to Market | ✅ Pass | Follows existing patterns exactly (WithDebug → WithVerbose). Minimal code changes. |
| V. High Availability & Developer Experience | ✅ Pass | Improves developer experience by providing a middle output tier between normal and debug. |
| Testing & Quality Gates | ✅ Pass | All changes covered by unit tests. Existing tests must continue to pass. |

**Post-Phase 1 Re-check**: No violations identified. The design adds no new abstractions, dependencies, or architectural patterns — it extends existing infrastructure with the minimum required changes.

## Project Structure

### Documentation (this feature)

```
specs/024-add-verbose-flag/
├── plan.md              # This file
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 entity design
├── contracts/           # Phase 1 API contracts
│   └── event-schema.md  # Event struct verbose field contract
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
cmd/wave/
├── main.go                          # Add --verbose persistent flag registration
└── commands/
    ├── run.go                       # Read verbose flag, pass to runRun(), add WithVerbose to executor options
    ├── status.go                    # Add verbose output (db path, timestamps, workspace locations)
    ├── clean.go                     # Add verbose output (workspace listing with sizes)
    ├── validate.go                  # Wire global verbose to existing local verbose (may need merge logic)
    ├── run_test.go                  # Test verbose flag propagation
    ├── status_test.go               # Test verbose status output
    └── clean_test.go                # Test verbose clean output

internal/
├── pipeline/
│   └── executor.go                  # Add verbose field, WithVerbose option, enrich events when verbose
├── event/
│   ├── emitter.go                   # Add verbose fields to Event struct, render in human-readable format
│   └── emitter_test.go              # Test verbose field serialization/omission
└── display/
    └── types.go                     # Wire existing VerboseOutput field (already defined)
```

**Structure Decision**: All changes are within the existing Go project structure. No new packages or directories are created (except `specs/024-add-verbose-flag/contracts/` for design documentation). This is an enhancement to the existing CLI — not a new component.

## Design Decisions

### D-001: Bool Threading over VerbosityLevel Enum

Follow the existing `debug` bool pattern rather than introducing a formal `VerbosityLevel` type. Two booleans (`debug`, `verbose`) with precedence resolution at point of use (`if debug { ... } else if verbose { ... } else { ... }`) is the simplest approach that satisfies all requirements.

See: research.md RES-001, RES-002

### D-002: Event System Extension over Direct Printf

Verbose output during pipeline execution flows through the existing event system by adding optional fields to the `Event` struct. This preserves the dual-stream architecture (FR-008) and ensures `--no-logs` interaction works automatically.

For non-pipeline commands (`status`, `clean`), verbose output uses direct `fmt.Fprintf(os.Stderr, ...)` since these commands don't use the event system.

See: research.md RES-003, RES-005

### D-003: Validate Command — Compose Global and Local Flags

The validate command's existing local `--verbose/-v` flag already works. The global persistent flag will shadow/compose with it via Cobra's flag resolution. No merge logic is needed — both flags activate the same behavior. The validate command reads `opts.Verbose` from its local flag; on other subcommands, the global flag is read via `cmd.Flags().GetBool("verbose")`.

See: research.md RES-004

## Complexity Tracking

_No constitution violations to justify. All changes follow existing patterns with minimal additions._

## Implementation Phases

### Phase A: Core Flag Infrastructure
1. Register `--verbose/-v` persistent flag on root command
2. Add `verbose bool` field to `DefaultPipelineExecutor`
3. Add `WithVerbose(verbose bool) ExecutorOption`
4. Read verbose flag in `run` command and pass to executor

### Phase B: Event System Extension
5. Add verbose fields to `Event` struct (WorkspacePath, InjectedArtifacts, ContractResult, VerboseDetail)
6. Populate verbose fields in executor event emissions when `e.verbose` is true
7. Render verbose fields in human-readable emitter output

### Phase C: Non-Pipeline Command Verbose Output
8. Add verbose output to `status` command (db path, timestamps, workspace locations)
9. Add verbose output to `clean` command (workspace listing with sizes)
10. Verify `validate` command works with global flag (existing local flag composes)

### Phase D: Testing
11. Unit tests for flag registration and propagation
12. Unit tests for verbose event fields (present when verbose, absent when not)
13. Unit tests for status/clean verbose output
14. Run full test suite to verify zero regression (SC-004)
