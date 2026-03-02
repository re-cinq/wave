# feat(tui): extend interactive UI for multi-pipeline sequence selection and DAG preview

**Issue**: [#209](https://github.com/re-cinq/wave/issues/209)
**Parent**: #184
**Labels**: enhancement, ux, pipeline, priority: high
**Author**: nextlevelshit
**State**: OPEN

## Summary

Extend the existing `internal/tui/run_selector.go` (built on `charmbracelet/huh`) to support the interactive pipeline orchestration mode. The current TUI handles single pipeline selection with flag input and confirmation. This issue adds multi-pipeline sequence proposals where users can accept, modify, or skip recommended pipelines, preview the execution DAG before committing, and compose parallel pipeline groups. The UI should feel like Claude Code's interactive proposal flow — options are sufficient, no typing needed.

## Acceptance Criteria

- [ ] TUI displays pipeline proposals from the proposal engine (#208) as selectable items
- [ ] Users can accept, modify, or skip individual pipeline proposals without typing commands
- [ ] Multi-select support for choosing multiple pipelines to run in a sequence
- [ ] Parallel-eligible pipelines are visually grouped and can be selected as a batch
- [ ] DAG preview shows execution order and dependencies before the user confirms
- [ ] DAG preview logic may reuse concepts from `internal/webui/dag.go`
- [ ] Extends existing `internal/tui/run_selector.go` rather than creating a parallel TUI system
- [ ] Uses existing `charmbracelet/huh` foundation (select, multi-select, input, confirmation)
- [ ] TUI gracefully handles edge cases: no proposals, single pipeline, all pipelines skipped
- [ ] Keyboard navigation and accessibility follow existing TUI patterns

## Dependencies

- #208 — Pipeline proposal engine (provides the proposals to display)

## Scope Notes

- **In scope**: Extending the existing TUI with multi-pipeline selection, DAG preview, accept/modify/skip flow, parallel group visualization
- **Out of scope**: Complete TUI redesign (#144 covers broader TUI polish), web UI changes (#91 covers web dashboard), real-time execution monitoring in the TUI during pipeline runs
- **Overlap note**: #144 covers TUI redesign (logo dedup, clear screen, model visibility); this issue is specifically about the multi-pipeline proposal interaction pattern. The two issues should be coordinated but are independently implementable
- **Design note**: The DAG preview should be a text-based representation suitable for terminal display, not a graphical rendering
