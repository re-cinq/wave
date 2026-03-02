# Implementation Plan: Multi-Pipeline TUI Selection & DAG Preview

## 1. Objective

Extend the existing `internal/tui/run_selector.go` to support multi-pipeline sequence selection, an accept/modify/skip proposal flow, parallel pipeline grouping, and a text-based DAG preview — all built on the existing `charmbracelet/huh` foundation.

## 2. Approach

The implementation follows an **additive extension** strategy: new types and functions are added to the existing `internal/tui` package without restructuring the current single-pipeline selector. The new multi-pipeline flow is exposed via a new entry point (`RunProposalSelector`) that the `cmd/wave/commands/run.go` command will call when the proposal engine (#208) provides proposals.

### Key design decisions:

1. **Proposal data contract** — Define a `Proposal` struct in `internal/tui` that the proposal engine (#208) will populate. This decouples the TUI from the engine's internals. The struct captures pipeline name, reason for recommendation, dependencies between proposals, and a parallelism group identifier.

2. **Accept/Modify/Skip per proposal** — Each proposal is presented as a `huh.Select` with three options: Accept (run as-is), Modify (toggle flags/input), Skip (exclude). No typing required — pure selection.

3. **Parallel group visualization** — Proposals sharing a parallelism group are visually grouped with a bracket/indent and a "parallel" label. The `huh.MultiSelect` allows batch-selecting an entire parallel group.

4. **Text-based DAG preview** — A new `internal/tui/dag_preview.go` file implements a terminal-friendly DAG renderer using box-drawing characters (`│`, `├`, `└`, `─`). It reuses the layered topological sort algorithm from `internal/webui/dag.go` (Kahn's algorithm) but outputs text instead of SVG coordinates.

5. **Confirmation with DAG** — After proposals are accepted/modified/skipped, a DAG preview is rendered inline (using lipgloss styling), followed by a `huh.Confirm` to commit the sequence.

## 3. File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `internal/tui/proposal.go` | **create** | `Proposal`, `ProposalDecision`, `ProposalResult` types |
| `internal/tui/proposal_selector.go` | **create** | `RunProposalSelector` — multi-pipeline proposal TUI flow |
| `internal/tui/dag_preview.go` | **create** | Text-based DAG renderer for terminal display |
| `internal/tui/dag_preview_test.go` | **create** | Unit tests for DAG text rendering |
| `internal/tui/proposal_selector_test.go` | **create** | Unit tests for proposal logic (filtering, grouping, composition) |
| `internal/tui/proposal_test.go` | **create** | Unit tests for proposal types and validation |
| `internal/tui/run_selector.go` | **modify** | Extract shared helpers (e.g. `buildFlagOptions`) if needed for reuse; no behavioral changes |
| `cmd/wave/commands/run.go` | **modify** | Wire `RunProposalSelector` into the interactive flow when proposals are available |

## 4. Architecture Decisions

### 4.1 Separate entry point vs. extending `RunPipelineSelector`

**Decision**: New `RunProposalSelector` function.

**Rationale**: The existing `RunPipelineSelector` is a clean single-pipeline flow. Merging multi-pipeline logic into it would create a complex conditional structure. A separate entry point keeps both paths simple and independently testable. The `cmd/wave/commands/run.go` dispatches to the appropriate selector based on whether proposals exist.

### 4.2 DAG renderer placement

**Decision**: `internal/tui/dag_preview.go` (in the `tui` package).

**Rationale**: The DAG preview is a TUI concern — it renders text for terminal display. Placing it in `internal/tui` keeps it co-located with the selector that uses it. The algorithm (Kahn's layered sort) is reimplemented rather than imported from `internal/webui/dag.go` because (a) `webui` is behind a build tag (`//go:build webui`) and (b) the output format is fundamentally different (text vs. SVG coordinates).

### 4.3 Proposal data contract

**Decision**: Define `Proposal` in `internal/tui/proposal.go` with fields that the proposal engine (#208) must populate.

**Rationale**: The proposal engine (#208) does not yet exist. By defining the consumer-side contract now, we establish a stable interface that #208 can target. The struct includes:

```go
type Proposal struct {
    Pipeline      string   // Pipeline name
    Reason        string   // Why this pipeline is recommended
    Dependencies  []string // Other proposal pipeline names this depends on
    ParallelGroup string   // Group ID for parallel-eligible pipelines (empty = sequential)
    Input         string   // Suggested input for this pipeline
    Priority      int      // Display/execution priority
}
```

### 4.4 Modify action semantics

**Decision**: "Modify" opens a sub-form to change input and toggle flags for that specific pipeline. It does not allow reordering or changing the pipeline itself.

**Rationale**: Reordering and pipeline substitution are orchestration-level concerns that belong in the proposal engine, not the TUI. The TUI handles presentation-level modifications: input text and runtime flags.

## 5. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| #208 proposal engine not yet implemented | High | Medium | Define stable `Proposal` interface; provide a `MockProposalProvider` for testing |
| DAG preview unreadable for wide/deep pipelines | Medium | Medium | Cap display width at terminal width; truncate node labels; collapse long chains |
| `huh` library limitations for nested forms | Low | High | Prototype the accept/modify/skip pattern early; fall back to sequential forms if nested groups aren't supported |
| Breaking existing single-pipeline TUI | Low | High | New entry point; existing `RunPipelineSelector` untouched; full test coverage on both paths |

## 6. Testing Strategy

### Unit Tests
- **`dag_preview_test.go`**: Linear chain, diamond dependency, parallel groups, empty input, single node, wide graphs exceeding terminal width
- **`proposal_selector_test.go`**: Proposal grouping logic, parallel group detection, accept/modify/skip state transitions, compose-command output for multi-pipeline sequences, edge cases (no proposals, all skipped, single proposal)
- **`proposal_test.go`**: Proposal validation (missing pipeline name, circular dependencies in proposals, duplicate parallel groups)

### Integration Tests
- Verify `RunProposalSelector` returns correct `ProposalResult` with mocked `huh` forms (if feasible) or via exported helper functions that don't require TTY interaction
- Verify `cmd/wave/commands/run.go` dispatches to the correct selector based on proposal availability

### Manual Testing
- Run with `--mock` adapter and sample proposals to validate the full interactive flow
- Test keyboard navigation: tab between proposals, space to select parallel groups, enter to confirm
- Verify terminal rendering at various widths (80, 120, 200+ columns)
