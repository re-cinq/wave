# Implementation Plan: Optional Pipeline Steps

**Branch**: `118-optional-pipeline-steps` | **Date**: 2026-02-20 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/118-optional-pipeline-steps/spec.md`

## Summary

Add `optional: true` support to pipeline step definitions so that non-critical steps (linting, notifications, staging deploys) can fail without halting the entire pipeline. Implementation introduces a new `"failed_optional"` step state, artifact-injection-based dependency skipping with transitive propagation, distinct event/display handling, and resume compatibility. Changes span 6 packages (`pipeline`, `state`, `event`, `display`, `manifest` parsing) with no schema migration required.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `gopkg.in/yaml.v3` (config parsing), `github.com/spf13/cobra` (CLI), `modernc.org/sqlite` (state store)
**Storage**: SQLite via `internal/state/store.go` — no schema migration needed (TEXT column accepts new state string)
**Testing**: `go test ./...` with `-race` flag, table-driven tests
**Target Platform**: Linux/macOS (single static binary)
**Project Type**: Single Go project
**Performance Goals**: No new hot paths — changes are in step-boundary logic (executed once per step, not per-token)
**Constraints**: Single binary, no new dependencies
**Scale/Scope**: 6 files modified, ~2 new test files, ~300 lines of new code

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new dependencies |
| P2: Manifest as SSOT | PASS | `optional` field parsed from `wave.yaml` step definitions |
| P3: Persona-Scoped Execution | PASS | No change to persona model |
| P4: Fresh Memory at Boundaries | PASS | No change to context isolation |
| P5: Navigator-First | PASS | No change to pipeline structure requirements |
| P6: Contracts at Handover | PASS | Contract validation skipped for failed optional steps (by design — no output to validate) |
| P7: Relay via Summarizer | PASS | No change to compaction |
| P8: Ephemeral Workspaces | PASS | No change to workspace isolation |
| P9: Credentials Never Touch Disk | PASS | No credential handling changes |
| P10: Observable Progress | PASS | New `"failed_optional"` event state + `Optional` bool on events enhances observability |
| P11: Bounded Recursion | PASS | No change to recursion limits |
| **P12: Minimal Step State Machine** | **VIOLATION** | Adds `"failed_optional"` state (6th state). See Complexity Tracking. |
| P13: Test Ownership | PASS | Full test suite run required; new tests for all new behavior |

## Project Structure

### Documentation (this feature)

```
specs/118-optional-pipeline-steps/
├── plan.md              # This file
├── research.md          # Phase 0: Technology decisions and rationale
├── data-model.md        # Phase 1: Entity changes and state machine
├── contracts/           # Phase 1: Behavioral contracts
│   ├── step-type-contract.go    # Compile-time type contracts
│   └── behavior-contract.md     # Test-based behavioral contracts
└── tasks.md             # Phase 2: Implementation tasks (not yet created)
```

### Source Code (repository root)

```
internal/
├── pipeline/
│   ├── types.go          # Add Optional field to Step, add StateFailedOptional constant
│   ├── executor.go       # Main execution loop: optional failure handling, dependency skipping
│   ├── resume.go         # Resume logic: skip failed_optional steps
│   └── errors.go         # No changes needed
├── state/
│   └── store.go          # Add StateFailedOptional constant, update SaveStepState completedAt logic
├── event/
│   └── emitter.go        # Add Optional field to Event, add StateFailedOptional constant
├── display/
│   ├── types.go          # Add StateFailedOptional to ProgressState enum
│   ├── progress.go       # Handle failed_optional in display rendering
│   ├── dashboard.go      # Icon/color for failed_optional state
│   ├── capability.go     # FormatState/GetStateIcon for failed_optional
│   ├── bubbletea_progress.go  # Handle failed_optional in bubbletea model
│   └── bubbletea_model.go     # Render failed_optional state
```

**Structure Decision**: All changes are within the existing Go package structure. No new packages or directories. This is a cross-cutting feature that touches multiple packages at their boundary interfaces (state constants, event types, display rendering) with the core logic concentrated in `internal/pipeline/executor.go`.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| P12: Adding 6th step state (`failed_optional`) | Core requirement — distinguishing optional failures from pipeline-halting failures is the entire feature. Without a distinct state, consumers cannot differentiate failure types in state queries, event streams, or display. | Overloading `"failed"` with a boolean flag: rejected because all existing consumers filter on exact state strings; changing semantics of `"failed"` would silently break dashboard queries, CLI status, and event listeners. The display package already has 6 `ProgressState` values (including `skipped`, `cancelled`) establishing precedent beyond the core 5. A constitution amendment should accompany this change, which is lightweight during Rapid Prototype phase (commit modifying constitution.md). |
