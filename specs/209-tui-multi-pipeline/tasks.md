# Tasks

## Phase 1: Foundation Types

- [X] Task 1.1: Create `internal/tui/proposal.go` with `Proposal`, `ProposalDecision`, and `ProposalResult` types
  - `Proposal` struct: Pipeline, Reason, Dependencies, ParallelGroup, Input, Priority
  - `ProposalDecision` enum: Accept, Modify, Skip
  - `ProposalResult` struct: ordered list of accepted pipelines with their inputs and flags
  - Validation: reject proposals with empty pipeline names, detect circular dependencies
- [X] Task 1.2: Create `internal/tui/proposal_test.go` with unit tests for proposal types and validation

## Phase 2: DAG Text Renderer

- [X] Task 2.1: Create `internal/tui/dag_preview.go` with text-based DAG rendering [P]
  - Implement Kahn's algorithm for layer assignment (adapted from `internal/webui/dag.go` logic)
  - Render layers top-to-bottom using box-drawing characters (│ ├ └ ─)
  - Show parallel groups with visual bracketing
  - Respect terminal width (lipgloss for styling, truncate labels if needed)
  - Function signature: `RenderDAGPreview(proposals []Proposal, termWidth int) string`
- [X] Task 2.2: Create `internal/tui/dag_preview_test.go` with DAG rendering tests [P]
  - Linear chain (A → B → C)
  - Diamond pattern (A → B, A → C, B → D, C → D)
  - Parallel group visualization
  - Single node
  - Empty input
  - Wide graph label truncation

## Phase 3: Proposal Selector TUI

- [X] Task 3.1: Create `internal/tui/proposal_selector.go` with `RunProposalSelector` function
  - Display Wave logo, then list proposals with accept/modify/skip per item
  - Group parallel-eligible proposals visually
  - "Modify" opens a sub-form for input and flags (reuse `buildFlagOptions` from `run_selector.go`)
  - After all decisions, render inline DAG preview of accepted pipelines
  - Final `huh.Confirm` to commit the sequence
  - Return `ProposalResult` with ordered pipeline sequence
  - Handle edge cases: no proposals (error), single proposal (simplified flow), all skipped (abort)
- [X] Task 3.2: Create `internal/tui/proposal_selector_test.go` with unit tests
  - Test grouping logic for parallel proposals
  - Test proposal-to-command composition
  - Test edge case handling (empty, single, all-skipped)

## Phase 4: CLI Integration

- [X] Task 4.1: Modify `cmd/wave/commands/run.go` to dispatch to `RunProposalSelector`
  - When proposals are available (from future #208 integration), call `RunProposalSelector`
  - Map `ProposalResult` back to execution options
  - Preserve existing single-pipeline `RunPipelineSelector` path unchanged
  - Add `--propose` flag stub that will activate proposal mode (gated on #208)
- [X] Task 4.2: Create `internal/tui/mock_proposals.go` for development/testing
  - `MockProposalProvider` that returns sample proposals for manual testing
  - Used when `--mock` is combined with `--propose`

## Phase 5: Testing & Polish

- [X] Task 5.1: Write integration tests verifying selector dispatch logic
  - Verify `run.go` calls `RunProposalSelector` when proposals present
  - Verify `run.go` falls back to `RunPipelineSelector` when no proposals
- [ ] Task 5.2: Manual testing and terminal compatibility validation
  - Test at 80, 120, and 200+ column widths
  - Verify keyboard navigation (tab, space, enter, escape)
  - Verify DAG preview readability for various pipeline topologies
- [X] Task 5.3: Run full test suite and fix any regressions
  - `go test ./...`
  - `go test -race ./...`
