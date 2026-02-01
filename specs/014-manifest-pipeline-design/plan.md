# Implementation Plan: Manifest & Pipeline Design

**Branch**: `014-manifest-pipeline-design` | **Date**: 2026-02-01 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/014-manifest-pipeline-design/spec.md`

## Summary

Build the Wave CLI binary in Go: a multi-agent orchestrator that
wraps Claude Code (and other LLM CLIs) via subprocess. The core
deliverables are manifest parsing/validation (`wave.yaml`), pipeline
DAG execution with handover contracts, persona-scoped agent invocation,
relay/compaction, ad-hoc execution, and meta-pipeline support. This is
a greenfield Go project — no existing source code.

## Technical Context

**Language/Version**: Go 1.22+ (single static binary, goroutines for
concurrency)
**Primary Dependencies**: `gopkg.in/yaml.v3` (YAML parsing),
`github.com/santhosh-tekuri/jsonschema` (JSON schema validation),
`github.com/mattn/go-sqlite3` (state persistence),
`github.com/spf13/cobra` (CLI framework)
**Storage**: SQLite for pipeline state persistence; filesystem for
workspaces and artifacts
**Testing**: `go test` with table-driven tests; integration tests using
a mock adapter binary
**Target Platform**: Linux (primary), macOS (secondary). Single static
binary.
**Project Type**: Single CLI binary
**Performance Goals**: Pipeline step startup overhead <500ms; support 5
concurrent matrix workers without resource contention
**Constraints**: Single binary with no runtime dependencies; credentials
never touch disk; ephemeral workspaces in configurable temp directory
**Scale/Scope**: Manages pipelines with up to 20 steps, up to 10
concurrent matrix workers, repos up to 1M LOC

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| # | Principle | Status | Notes |
|---|-----------|--------|-------|
| 1 | Single Binary, Zero Dependencies | ✅ PASS | Go produces static binary. CGo needed for SQLite — use `modernc.org/sqlite` (pure Go) to avoid CGo. |
| 2 | Manifest as Single Source of Truth | ✅ PASS | `wave.yaml` is the only config file. Pipelines are referenced from it or discovered by convention. |
| 3 | Persona-Scoped Execution Boundaries | ✅ PASS | Every step binds exactly one persona. Permissions enforced before subprocess invocation. |
| 4 | Fresh Memory at Every Step Boundary | ✅ PASS | Each step spawns a new subprocess. No shared state except explicit artifact injection. |
| 5 | Navigator-First Architecture | ✅ PASS | Pipeline schema validation enforces step[0] must use navigator persona (or be overridden explicitly). |
| 6 | Contracts at Every Handover | ✅ PASS | Contract validation is a required field in pipeline step definitions. |
| 7 | Relay via Dedicated Summarizer | ✅ PASS | Relay spawns a separate summarizer subprocess. Never self-summarizes. |
| 8 | Ephemeral Workspaces for Safety | ✅ PASS | Each step gets `/tmp/wave/<pipeline_id>/<step_id>/`. Main repo mounted readonly by default. |
| 9 | Credentials Never Touch Disk | ✅ PASS | Env vars inherited by subprocess. Audit log scrubbing for known env var patterns. |
| 10 | Observable Progress, Auditable Operations | ✅ PASS | Structured JSON events to stdout. Trace files in `.wave/traces/`. |
| 11 | Bounded Recursion and Resource Limits | ✅ PASS | Runtime tracks recursion depth via `--parent-pipeline` flag. Hard caps in manifest. |
| 12 | Minimal Step State Machine | ✅ PASS | 5 states only: Pending, Running, Completed, Failed, Retrying. |

No violations. No complexity tracking entries needed.

## Project Structure

### Documentation (this feature)

```
specs/014-manifest-pipeline-design/
├── plan.md              # This file
├── research.md          # Phase 0: technology decisions
├── data-model.md        # Phase 1: entity definitions
├── quickstart.md        # Phase 1: getting started guide
├── contracts/           # Phase 1: Go interface contracts
└── tasks.md             # Phase 2: task list (/speckit.tasks)
```

### Source Code (repository root)

```
cmd/
└── wave/
    └── main.go              # CLI entry point (cobra root command)

internal/
├── manifest/
│   ├── types.go             # Manifest, Adapter, Persona, Runtime structs
│   ├── parser.go            # YAML parsing and validation
│   └── parser_test.go
├── pipeline/
│   ├── types.go             # Pipeline, Step, Handover, Contract structs
│   ├── dag.go               # DAG resolution, cycle detection, topological sort
│   ├── dag_test.go
│   ├── executor.go          # Step execution loop, retry, state transitions
│   ├── executor_test.go
│   ├── matrix.go            # Matrix strategy: fan-out parallel workers
│   └── matrix_test.go
├── adapter/
│   ├── adapter.go           # Adapter interface + subprocess invocation
│   ├── claude.go            # Claude Code adapter (claude -p)
│   ├── adapter_test.go
│   └── mock.go              # Mock adapter for testing
├── workspace/
│   ├── workspace.go         # Ephemeral workspace creation, mounting, cleanup
│   └── workspace_test.go
├── contract/
│   ├── contract.go          # Contract interface
│   ├── jsonschema.go        # JSON schema validator
│   ├── typescript.go        # TypeScript compilation validator
│   ├── testsuite.go         # Test suite runner validator
│   └── contract_test.go
├── relay/
│   ├── relay.go             # Token threshold monitor, compaction trigger
│   ├── checkpoint.go        # Checkpoint parsing and injection
│   └── relay_test.go
├── state/
│   ├── store.go             # Pipeline state persistence (SQLite)
│   ├── store_test.go
│   └── schema.sql           # SQLite schema
├── event/
│   ├── emitter.go           # Structured event stream to stdout
│   └── emitter_test.go
└── audit/
    ├── logger.go            # Tool call and file operation logging
    └── logger_test.go

go.mod
go.sum
```

**Structure Decision**: Single project layout following standard Go
conventions. `cmd/` for the binary entry point, `internal/` for all
packages (not importable by external consumers). No `pkg/` directory
since this is a standalone CLI, not a library.

## Complexity Tracking

_No violations to justify._
