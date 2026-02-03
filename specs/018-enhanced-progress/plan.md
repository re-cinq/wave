# Implementation Plan: Enhanced Pipeline Progress Visualization

**Branch**: `018-enhanced-progress` | **Date**: 2026-02-03 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/018-enhanced-progress/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Enhance Wave's pipeline progress visualization to provide users with modern CLI/TUI-style progress indicators that show real-time step execution, overall pipeline status, and clear visual feedback that the system is actively working. This will eliminate user uncertainty about whether pipelines are running or hung, providing confidence and better user experience similar to tools like Claude Code.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.25.5
**Primary Dependencies**: github.com/spf13/cobra (CLI), golang.org/x/term (terminal), modernc.org/sqlite (state)
**Storage**: SQLite-based state persistence (pipeline runs, events, step states)
**Testing**: Go stdlib testing + github.com/stretchr/testify (table-driven tests)
**Target Platform**: Cross-platform CLI (Linux/macOS/Windows) - single static binary
**Project Type**: Single CLI application with subprocess orchestration
**Performance Goals**: Progress updates within 1 second of step state changes, <5% overhead from visualization (measured as additional CPU/memory usage relative to total pipeline execution time)
**Constraints**: No runtime dependencies, backward compatible NDJSON output, TTY graceful degradation
**Scale/Scope**: Pipelines with 1-20 steps, concurrent matrix execution, real-time progress streaming

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

### ✅ Principle 1: Single Binary, Zero Dependencies
**Status**: COMPLIANT
- Progress visualization will use existing `golang.org/x/term` and ANSI escape sequences
- No new runtime dependencies (TUI libraries like Bubble Tea explicitly avoided)
- Enhanced output remains within existing CLI framework

### ✅ Principle 2: Manifest as Single Source of Truth
**Status**: COMPLIANT
- No configuration changes to `wave.yaml` required
- Progress preferences could optionally be added to manifest but not required
- Feature operates on existing pipeline definitions

### ✅ Principle 3: Persona-Scoped Execution Boundaries
**Status**: COMPLIANT
- Progress visualization is observer-only, no persona execution changes
- No modifications to permission systems or hooks
- Enhanced display doesn't affect persona isolation

### ✅ Principle 4: Fresh Memory at Every Step Boundary
**Status**: COMPLIANT
- Progress tracking uses existing event system (no memory inheritance)
- Visualization consumes events but doesn't modify step execution
- No impact on artifact flow or context pollution prevention

### ✅ Principle 5: Navigator-First Architecture
**Status**: COMPLIANT
- Progress display is consumer of existing pipeline structure
- No changes to Navigator persona or collaboration units
- Enhanced visualization doesn't affect file access patterns

### ✅ Principle 6: Contracts at Every Handover
**Status**: COMPLIANT
- Progress events are observational, don't affect contract validation
- Handover contracts remain unchanged
- Visualization failures won't break pipeline execution

### ✅ Principle 7: Relay via Dedicated Summarizer
**Status**: COMPLIANT
- Progress tracking already integrated with existing relay/compaction
- Token threshold monitoring unchanged
- Enhanced display of compaction progress (not modification of process)

### ✅ Principle 8: Ephemeral Workspaces for Safety
**Status**: COMPLIANT
- Progress visualization reads from existing workspace structure
- No modifications to workspace isolation or persistence
- Display of workspace paths enhances transparency

### ✅ Principle 9: Credentials Never Touch Disk
**Status**: COMPLIANT
- Progress events already credential-safe (existing audit log patterns)
- Enhanced display doesn't introduce credential exposure risks
- Maintains existing scrubbing patterns

### ✅ Principle 10: Observable Progress, Auditable Operations
**Status**: ENHANCED COMPLIANCE
- **This feature directly supports this principle**
- Improves structured progress events (human and machine readable)
- Enhances audit trail visualization without modifying logging
- Better user observability while maintaining machine parseable output

### ✅ Principle 11: Bounded Recursion and Resource Limits
**Status**: COMPLIANT
- Progress visualization adds minimal overhead (<5% target)
- No impact on recursion depth or timeout mechanisms
- Resource consumption tracking enhanced, not increased

### ✅ Principle 12: Minimal Step State Machine
**Status**: COMPLIANT
- Uses existing 5-state model (Pending → Running → Completed/Failed/Retrying)
- May add intermediate progress events but no new states
- Enhanced display of existing state transitions

**Overall Assessment**: ✅ **FULLY COMPLIANT** - This feature enhances constitutional principle #10 while respecting all other constraints.

## Project Structure

### Documentation (this feature)

```
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```
cmd/wave/
├── commands/
│   ├── run.go             # Main execution command (progress display integration)
│   ├── logs.go            # Log streaming (enhanced with progress queries)
│   └── status.go          # Status display (enhanced visualization)

internal/
├── event/
│   ├── emitter.go         # Core progress event emission (enhancement target)
│   ├── emitter_test.go    # Event system tests (expand for new features)
│   └── types.go           # Event structure definitions (extend for progress)
├── pipeline/
│   ├── executor.go        # Pipeline execution (progress event sources)
│   └── status.go          # Pipeline status queries (enhanced display)
├── display/               # NEW: Enhanced visualization package
│   ├── progress.go        # Progress bar and TUI components
│   ├── dashboard.go       # Logo panel and project info display
│   ├── formatter.go       # Enhanced output formatting
│   └── animation.go       # Loading indicators and animations
└── state/
    ├── store.go           # SQLite persistence (progress queries)
    └── types.go           # State structures (extend for visualization)

tests/
├── integration/
│   └── progress_test.go   # End-to-end progress display tests
└── unit/
    └── display/           # Unit tests for visualization components
        ├── progress_test.go
        ├── dashboard_test.go
        └── formatter_test.go
```

**Structure Decision**: Single Go project extending existing Wave CLI architecture. New `internal/display/` package provides enhanced visualization components while preserving existing event system and state management. Progress features integrate with current CLI commands (`run`, `logs`, `status`) without breaking backward compatibility.

## Complexity Tracking

_Fill ONLY if Constitution Check has violations that must be justified_

| Violation                  | Why Needed         | Simpler Alternative Rejected Because |
| -------------------------- | ------------------ | ------------------------------------ |
| [e.g., 4th project]        | [current need]     | [why 3 projects insufficient]        |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient]  |
