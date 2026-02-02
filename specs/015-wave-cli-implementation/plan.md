# Implementation Plan: Wave CLI Implementation

**Branch**: `015-wave-cli-implementation` | **Date**: 2026-02-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/015-wave-cli-implementation/spec.md`

## Summary

Harden and complete the existing Wave CLI implementation from spec 014. The
codebase already has scaffolding for all major subsystems (cmd/wave commands,
internal packages for manifest, pipeline, adapter, workspace, contract, relay,
state, event, audit). This spec focuses on: fixing bugs, adding comprehensive
tests, improving error handling, and ensuring production readiness. No rewrite
— refine what exists.

## Technical Context

**Language/Version**: Go 1.22+ (single static binary, goroutines for concurrency)
**Primary Dependencies**:
- `gopkg.in/yaml.v3` (YAML parsing)
- `github.com/santhosh-tekuri/jsonschema/v6` (JSON schema validation)
- `modernc.org/sqlite` (pure Go SQLite for state persistence)
- `github.com/spf13/cobra` (CLI framework)
**Storage**: SQLite for pipeline state persistence; filesystem for workspaces and artifacts
**Testing**: `go test` with table-driven tests; integration tests using mock adapter binary
**Target Platform**: Linux (primary), macOS (secondary). Single static binary.
**Project Type**: Single CLI binary
**Performance Goals**: Pipeline step startup overhead <500ms; support 10 concurrent matrix workers without resource contention
**Constraints**: Single binary with no runtime dependencies; credentials never touch disk; ephemeral workspaces in configurable temp directory
**Scale/Scope**: Manages pipelines with up to 20 steps, up to 10 concurrent matrix workers, repos up to 1M LOC

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| # | Principle | Status | Notes |
|---|-----------|--------|-------|
| 1 | Single Binary, Zero Dependencies | ✅ PASS | Existing code uses `modernc.org/sqlite` (pure Go). No CGo. |
| 2 | Manifest as Single Source of Truth | ✅ PASS | `wave.yaml` is the only config file. Already implemented in `internal/manifest`. |
| 3 | Persona-Scoped Execution Boundaries | ✅ PASS | Every step binds exactly one persona via `internal/adapter`. Permissions enforced. |
| 4 | Fresh Memory at Every Step Boundary | ✅ PASS | Each step spawns a new subprocess in `internal/pipeline/executor.go`. |
| 5 | Navigator-First Architecture | ✅ PASS | Pipeline DAG validation in place. Ad-hoc execution starts with navigator. |
| 6 | Contracts at Every Handover | ✅ PASS | Contract validation implemented in `internal/contract`. |
| 7 | Relay via Dedicated Summarizer | ✅ PASS | Relay spawns separate summarizer in `internal/relay`. |
| 8 | Ephemeral Workspaces for Safety | ✅ PASS | Workspace management in `internal/workspace`. |
| 9 | Credentials Never Touch Disk | ✅ PASS | Env vars inherited by subprocess. Audit log scrubbing implemented. |
| 10 | Observable Progress, Auditable Operations | ✅ PASS | Structured JSON events in `internal/event`. Audit logging in `internal/audit`. |
| 11 | Bounded Recursion and Resource Limits | ✅ PASS | Meta-pipeline depth tracking in `internal/pipeline/meta.go`. |
| 12 | Minimal Step State Machine | ✅ PASS | 5 states implemented: Pending, Running, Completed, Failed, Retrying. |

No violations. No complexity tracking entries needed.

## Project Structure

### Documentation (this feature)

```
specs/015-wave-cli-implementation/
├── plan.md              # This file
├── research.md          # Phase 0: hardening focus areas
├── data-model.md        # Phase 1: (reuse 014, no changes needed)
├── quickstart.md        # Phase 1: testing guide for hardened code
├── contracts/           # Phase 1: (reuse 014)
└── tasks.md             # Phase 2: hardening task list
```

### Source Code (existing - to be hardened)

```
cmd/
└── wave/
    ├── main.go              # CLI entry point
    └── commands/
        ├── init.go          # wave init - NEEDS: tests, error handling
        ├── validate.go      # wave validate - NEEDS: tests, verbose output
        ├── run.go           # wave run - NEEDS: tests, dry-run mode
        ├── do.go            # wave do - NEEDS: tests, save flag
        ├── list.go          # wave list - NEEDS: tests
        ├── resume.go        # wave resume - NEEDS: tests, listing mode
        └── clean.go         # wave clean - NEEDS: tests, keep-last flag

internal/
├── manifest/
│   ├── types.go             # ✅ Has tests
│   ├── parser.go            # ✅ Has tests
│   └── parser_test.go       # NEEDS: edge case coverage
├── pipeline/
│   ├── types.go             # ✅ Defined
│   ├── dag.go               # ✅ Has tests
│   ├── dag_test.go          # NEEDS: cycle detection edge cases
│   ├── executor.go          # ✅ Implemented - NEEDS: tests
│   ├── matrix.go            # ✅ Has tests
│   ├── matrix_test.go       # NEEDS: partial failure tests
│   ├── meta.go              # ✅ Has tests
│   ├── meta_test.go         # NEEDS: recursion limit tests
│   ├── router.go            # ✅ Has tests
│   ├── router_test.go       # NEEDS: routing edge cases
│   └── adhoc.go             # ✅ Implemented - NEEDS: tests
├── adapter/
│   ├── adapter.go           # ✅ Has tests
│   ├── claude.go            # ✅ Implemented - NEEDS: error handling tests
│   ├── opencode.go          # ✅ Implemented
│   ├── mock.go              # ✅ Mock for testing
│   └── adapter_test.go      # NEEDS: subprocess error tests
├── workspace/
│   ├── workspace.go         # ✅ Has tests
│   └── workspace_test.go    # NEEDS: mount failure tests
├── contract/
│   ├── contract.go          # ✅ Has tests
│   ├── jsonschema.go        # ✅ Implemented
│   ├── typescript.go        # ✅ Implemented - NEEDS: compiler absence tests
│   ├── testsuite.go         # ✅ Implemented
│   └── contract_test.go     # NEEDS: validation failure tests
├── relay/
│   ├── relay.go             # ✅ Has tests
│   ├── checkpoint.go        # ✅ Implemented - NEEDS: parsing tests
│   └── relay_test.go        # NEEDS: threshold edge case tests
├── state/
│   ├── store.go             # ✅ Implemented - NEEDS: tests
│   └── schema.sql           # ✅ Defined
├── event/
│   ├── emitter.go           # ✅ Has tests
│   └── emitter_test.go      # NEEDS: concurrent emission tests
└── audit/
    ├── logger.go            # ✅ Has tests
    └── logger_test.go       # NEEDS: credential scrubbing tests

go.mod
go.sum
```

**Structure Decision**: Existing structure from spec 014 is correct and follows Go conventions. No structural changes needed. Focus is on adding tests and improving implementations within the existing packages.

## Hardening Focus Areas

Based on spec 015 requirements and the comprehensive checklist gaps:

### Area 1: CLI Command Tests (High Priority)
- No test files exist for `cmd/wave/commands/`
- Need integration tests for all 7 commands
- Need error handling tests for invalid inputs

### Area 2: State Persistence Tests (High Priority)
- `internal/state/store.go` has no tests
- Need SQLite CRUD tests
- Need concurrent access tests for matrix workers
- Need corruption recovery tests

### Area 3: Error Handling Hardening (Medium Priority)
- Improve error messages with context (file paths, line numbers)
- Add graceful degradation for missing compilation tools
- Add timeout handling for subprocess hangs

### Area 4: Edge Case Coverage (Medium Priority)
- Empty pipeline handling
- Zero tasks in matrix strategy
- Concurrent pipeline execution
- Disk full scenarios

### Area 5: Security Hardening (Medium Priority)
- Credential scrubbing in audit logs
- Workspace isolation verification
- Permission bypass attempt handling

## Complexity Tracking

_No violations to justify._
