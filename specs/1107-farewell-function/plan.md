# Implementation Plan: Farewell Function

**Branch**: `1107-farewell-function` | **Date**: 2026-04-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/1107-farewell-function/spec.md`

## Summary

Add a single-source English farewell line printed at the end of successful
interactive Wave CLI commands. Core is a tiny, pure Go function `Farewell(name
string) string` in a new `internal/farewell` package. The CLI wires it into
the shared command post-run path (reusing the existing global `--quiet` /
non-TTY suppression already used by `internal/cli`/`cmd/wave/commands/output.go`)
and the TUI teardown. Embedders can call the function directly.

## Technical Context

**Language/Version**: Go 1.23
**Primary Dependencies**: stdlib only (`os`, `io`, `fmt`); existing
`cmd/wave/commands/output.go` for quiet/TTY decisions
**Storage**: N/A (pure function, no persistence)
**Testing**: `go test ./...` (table-driven unit tests)
**Target Platform**: Linux/macOS terminals (Wave CLI hosts)
**Project Type**: single Go module (existing layout)
**Performance Goals**: negligible; <1ms added per CLI run (SC-004 budget 50ms)
**Constraints**: no new dependencies, no new global flag, English-only,
deterministic output for fixed inputs
**Scale/Scope**: ~1 package, ~60 LOC + tests; 1–2 wiring points (CLI post-run,
TUI teardown)

## Constitution Check

Evaluated against `.specify/memory/constitution.md` v2.1.0.

| Principle                                    | Status | Notes                                                                            |
| -------------------------------------------- | ------ | -------------------------------------------------------------------------------- |
| 1. Single Binary, Minimal Dependencies       | PASS   | stdlib only.                                                                     |
| 2. Manifest as Single Source of Truth        | N/A    | No manifest surface touched.                                                     |
| 3. Persona-Scoped Execution Boundaries       | N/A    | CLI UX feature, not an agent.                                                    |
| 4. Fresh Memory at Every Step Boundary       | N/A    | Not a pipeline step.                                                             |
| 5. Navigator-First Architecture              | N/A    | Not a pipeline.                                                                  |
| 6. Contracts at Every Handover               | N/A    | No inter-step artifact.                                                          |
| 7. Relay via Dedicated Summarizer            | N/A    | No LLM I/O.                                                                      |
| 8. Ephemeral Workspaces for Safety           | N/A    | Pure function + stdout write.                                                    |
| 9. Credentials Never Touch Disk              | PASS   | No secrets handled. Uses `$USER` only.                                           |
| 10. Observable Progress, Auditable           | PASS   | stdout-only, does not interfere with structured events.                          |
| 11. Bounded Recursion / Resource Limits      | N/A    | No recursion.                                                                    |
| 12. Minimal Step State Machine               | N/A    | No state transitions.                                                            |
| 13. Test Ownership for Core Primitives       | PASS   | Adds unit tests; `go test ./...` must stay green.                                |

**Result**: No violations; Complexity Tracking empty.

## Project Structure

### Documentation (this feature)

```
specs/1107-farewell-function/
├── plan.md
├── research.md
├── data-model.md
├── contracts/
│   └── farewell.md
├── spec.md
└── checklists/
```

### Source Code (repository root)

```
internal/
└── farewell/
    ├── farewell.go        # Farewell(name string) string + WriteFarewell(w, name, suppress)
    └── farewell_test.go   # unit tests (default msg, name interpolation, determinism)

cmd/wave/commands/
└── output.go              # existing: reuse quiet/TTY logic (no flag added)
└── root.go / run.go / ... # one post-run hook calls farewell.WriteFarewell
```

**Structure Decision**: Single Go project — new `internal/farewell` package
(pure function + thin stdout writer), wired once into the existing CLI
post-command path. Suppression reuses `cmd/wave/commands/output.go` quiet/TTY
detection; no new flag, no new config surface.

## Phase 0 — Outline & Research

See [research.md](./research.md). All clarifications from the spec are
resolved; no open `NEEDS CLARIFICATION` markers remain.

## Phase 1 — Design & Contracts

- Data model: [data-model.md](./data-model.md) (one value: the farewell string).
- Contracts: [contracts/farewell.md](./contracts/farewell.md) describes the
  public `Farewell` function signature and the stdout-writing CLI helper.
- Agent context: `.specify/scripts/bash/update-agent-context.sh claude` — not
  run in this worktree (script not required for a stdlib-only change; no new
  tech stack introduced).

## Complexity Tracking

_No violations._
