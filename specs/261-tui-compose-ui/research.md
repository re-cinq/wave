# Research: Pipeline Composition UI (#261)

**Branch**: `261-tui-compose-ui` | **Date**: 2026-03-07

## R1: Modal State Pattern in Existing TUI

**Decision**: Compose mode is a modal state within the Pipelines view (not a new ViewType).

**Rationale**: The codebase already has a well-established modal pattern:
- `DetailPaneState` enum (`stateConfiguring`, `stateRunningLive`, etc.) controls right pane rendering
- `PipelineDetailModel.Update()` intercepts all messages when form is active (`stateConfiguring`)
- `ContentModel` gates `Tab` key forwarding — blocks view cycling when form is active
- `FormActiveMsg` / `LiveOutputActiveMsg` control status bar hint switching
- `FocusChangedMsg` manages left/right pane focus transitions

**Approach**: Add `stateComposing` to `DetailPaneState` for the right pane artifact flow, plus a `composing` boolean on `ContentModel` to gate left pane replacement (sequence list replaces pipeline list).

**Alternatives Rejected**:
- New `ViewType` — would make compose mode permanently Tab-accessible, wrong for a transient action
- Separate full-screen model — would break the existing layout system

## R2: Sequence List (Left Pane in Compose Mode)

**Decision**: Create `ComposeListModel` as a standalone Bubble Tea model replacing the left pane during compose mode.

**Rationale**: The existing `PipelineListModel` has section headers, filter, collapse, and data provider integration — none of which apply to the sequence builder. A clean model with:
- Ordered list of `SequenceEntry` (pipeline name + index)
- Cursor navigation (↑/↓)
- Reorder via shift+↑/shift+↓
- Add via `a` key (opens inline picker using `huh.Select`)
- Remove via `x` key
- Start via `Enter` (with validation gate)
- Cancel via `Esc`

**Alternatives Rejected**:
- Reusing `PipelineListModel` with a "compose mode" flag — too complex, would bloat an already substantial model

## R3: Artifact Flow Visualization (Right Pane in Compose Mode)

**Decision**: Render artifact flow as a vertically-scrolling viewport using box-drawing characters for connections.

**Rationale**: The right pane already uses `viewport.Model` for scrollable content in all detail states. The artifact flow can be rendered as styled text:
```
┌─────────────────────────┐
│ speckit-flow             │
│  outputs: spec-status    │
└──────────┬──────────────┘
           │ spec-status → spec_info ✓
┌──────────┴──────────────┐
│ wave-evolve              │
│  inputs: spec_info       │
│  outputs: evolve-result  │
└──────────┬──────────────┘
           │ evolve-result → ✗ (no match)
┌──────────┴──────────────┐
│ wave-review              │
│  inputs: review_input    │
└─────────────────────────┘
```

Below 120 columns, degrade to text-only summary:
```
speckit-flow → wave-evolve
  ✓ spec-status → spec_info (match)
wave-evolve → wave-review
  ✗ review_input (missing — no matching output)
```

**Alternatives Rejected**:
- ASCII-art DAG with horizontal layout — too wide for terminal constraints
- External rendering library — violates single-binary principle

## R4: Cross-Pipeline Artifact Matching

**Decision**: Match `output_artifacts[].name` of the **last step** in pipeline N against `inject_artifacts[].artifact` of the **first step** in pipeline N+1. Name-only matching.

**Rationale**: Per spec clarification C3, intra-pipeline injection already uses name-based matching via `ArtifactRef.Artifact`. Type fields are optional and often omitted. The "last step outputs → first step inputs" heuristic covers the standard pipeline boundary pattern.

**Key data types** (from `internal/pipeline/types.go`):
- `Step.OutputArtifacts []ArtifactDef` — has `.Name`, `.Path`, `.Type`
- `Step.Memory.InjectArtifacts []ArtifactRef` — has `.Step`, `.Artifact`, `.As`, `.Optional`

**Edge cases**:
- Pipeline with no steps → skip (shouldn't exist, validate will catch)
- Step with no output_artifacts → show "No artifacts" warning
- Optional inject artifacts (`ArtifactRef.Optional == true`) → no warning when unmatched

## R5: CLI `wave compose` Command

**Decision**: New `wave compose` subcommand with variadic pipeline names and `--validate-only` flag.

**Rationale**: Per spec clarification C4, modifying `wave run` would break the existing `[pipeline] [input]` contract. The existing CLI structure uses `cobra.Command` with subcommands in `cmd/wave/commands/`. The compose command follows the same pattern.

**Approach**: Create `cmd/wave/commands/compose.go`:
- `Use: "compose [pipelines...]"`
- `Args: cobra.MinimumNArgs(2)` (at least 2 pipelines for a sequence)
- `--validate-only` flag for dry-run compatibility checking
- Reuses the same `Sequence`, `ArtifactFlow`, and `CompatibilityResult` types from the TUI

## R6: Grouped Running Display for Sequences

**Decision**: Design data structures for grouped sequence display but stub the execution since #249 (cross-pipeline artifact handoff) is not implemented.

**Rationale**: Per FR-010 and User Story 4, running sequences should appear as a single grouped entry showing per-pipeline progress. Since #249 is not available, the `Enter` action will show an informational message. The grouped display data structures should be designed now to enable future implementation.

**Alternatives Rejected**:
- Flat list with prefix markers — doesn't visually convey the relationship
- Tree view — over-engineered for a simple sequence
