# Implementation Plan: Skill Dependency Installation in Pipeline Steps

**Branch**: `102-skill-deps-pipeline` | **Date**: 2026-02-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/102-skill-deps-pipeline/spec.md`

## Summary

Enable Wave pipelines to declare external skill dependencies (Speckit, BMAD, OpenSpec) and CLI tool requirements that are automatically validated and installed during a preflight phase before any step executes. The core infrastructure already exists — this feature wires together the existing `preflight.Checker`, `skill.Provisioner`, and `event.EventEmitter` with targeted enhancements for per-dependency event emission, a `StatePreflight` constant, and a repoRoot propagation fix.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `gopkg.in/yaml.v3`, `github.com/spf13/cobra` (existing Wave dependencies)
**Storage**: SQLite for pipeline state (existing), filesystem for workspaces and artifacts (existing)
**Testing**: `go test -race ./...` (existing test infrastructure)
**Target Platform**: Linux (primary), macOS (secondary)
**Project Type**: Single Go binary (CLI tool)
**Performance Goals**: Preflight phase adds <500ms overhead when all dependencies are pre-installed (SC-006)
**Constraints**: Single static binary, no new runtime dependencies
**Scale/Scope**: 3 target skills, 12 functional requirements, 6 edge cases

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new dependencies. Uses existing `os/exec` for skill commands. |
| P2: Manifest as SSOT | PASS | Skills declared in `wave.yaml`. Pipeline `requires` in pipeline YAML. |
| P3: Persona-Scoped Execution | PASS | No changes to persona execution model. |
| P4: Fresh Memory | PASS | Preflight runs before steps; no cross-step memory. |
| P5: Navigator-First | PASS | Preflight is a pre-step phase, not a step. Does not bypass navigator. |
| P6: Contracts at Handovers | PASS | No new handover contracts needed for preflight. |
| P7: Relay via Summarizer | N/A | Preflight doesn't use relay. |
| P8: Ephemeral Workspaces | PASS | Skill commands provisioned per-step workspace. |
| P9: Credentials Never Touch Disk | PASS | No credential handling in preflight. |
| P10: Observable Progress | PASS | FR-010 adds structured preflight events. StatePreflight constant. |
| P11: Bounded Recursion | N/A | Preflight is not recursive. |
| P12: Minimal State Machine | PASS | No new step states. Preflight is pre-step, not a step state. |
| P13: Test Ownership | PASS | Existing tests must pass. New tests for new behavior. |

**Post-Phase 1 Re-check**: PASS. No constitution violations identified.

## Project Structure

### Documentation (this feature)

```
specs/102-skill-deps-pipeline/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model
├── contracts/           # Phase 1 API contracts
│   ├── preflight-checker.go
│   ├── event-states.go
│   ├── provisioner.go
│   └── manifest-skills-schema.yaml
└── tasks.md             # Phase 2 output (NOT created by plan)
```

### Source Code (repository root)

```
internal/
├── event/
│   └── emitter.go              # Add StatePreflight constant
├── preflight/
│   ├── preflight.go            # Add event emission callback to Checker
│   └── preflight_test.go       # Extend tests for event emission
├── pipeline/
│   └── executor.go             # Fix repoRoot, use StatePreflight constant
├── skill/
│   ├── skill.go                # No changes needed
│   └── skill_test.go           # No changes needed
└── manifest/
    ├── types.go                # No changes needed
    └── parser.go               # No changes needed

wave.yaml                       # Add skills section with speckit, bmad, openspec
```

**Structure Decision**: Existing Go package structure. All changes are modifications to existing files. No new packages or files needed (except contracts which are design artifacts).

## Implementation Changes

### Change 1: Add `StatePreflight` constant

**File**: `internal/event/emitter.go`
**Type**: Addition (1 line)
**Risk**: Low

Add `StatePreflight = "preflight"` to the existing event state constants block (after line 69). This replaces the string literal `"preflight"` used in executor.go:173.

### Change 2: Add event emission to preflight Checker

**File**: `internal/preflight/preflight.go`
**Type**: Modification (extend Checker struct and methods)
**Risk**: Medium

- Add `emitter func(name, kind, message string)` field to `Checker` struct
- Add `WithEmitter` functional option for `NewChecker`
- Call emitter callback at each decision point in `CheckTools()` and `CheckSkills()`
- Emit events for: checking, found, not-found, installing, installed, install-failed, init-failed, still-not-detected

Backward compatible: existing code that doesn't pass an emitter gets nil callback (no-op).

### Change 3: Fix repoRoot in executor

**File**: `internal/pipeline/executor.go`
**Type**: Bug fix (1-2 lines)
**Risk**: Low

At line 512, change `skill.NewProvisioner(execution.Manifest.Skills, "")` to pass the actual repository root. Derive from the working directory or manifest path. The simplest approach: use the current working directory (which is the project root when Wave executes).

### Change 4: Use `StatePreflight` in executor events

**File**: `internal/pipeline/executor.go`
**Type**: Refactor (replace string literal)
**Risk**: Low

Replace `State: "preflight"` at line 173 with `State: event.StatePreflight`.

### Change 5: Wire emitter callback in executor

**File**: `internal/pipeline/executor.go`
**Type**: Modification (extend preflight integration)
**Risk**: Medium

In the preflight section (lines 158-181):
- Create Checker with emitter callback that wraps the executor's event emitter
- The callback constructs `event.Event{State: event.StatePreflight, ...}` and calls `e.emit()`
- Remove the post-hoc event emission loop (lines 170-175) since events are now emitted inline

### Change 6: Add skill definitions to wave.yaml

**File**: `wave.yaml`
**Type**: Addition (configuration)
**Risk**: Low

Add `skills` section with speckit, bmad, and openspec definitions.

### Change 7: Extend preflight tests

**File**: `internal/preflight/preflight_test.go`
**Type**: Addition (new test cases)
**Risk**: Low

- Test event emission callback is called during checks
- Test event emission for install+init sequence
- Test nil emitter doesn't panic
- Test edge cases: install succeeds but re-check fails, init fails

## Complexity Tracking

_No constitution violations identified. No complexity tracking entries needed._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| (none)    |           |                                      |
