# feat: adopt epic #589 features into shipped pipelines, TUI, and WebUI

**Issue**: [re-cinq/wave#627](https://github.com/re-cinq/wave/issues/627)
**Labels**: enhancement
**Author**: nextlevelshit
**State**: OPEN

## Context

Epic #589 delivered 24,000 lines of engine code (failure taxonomy, graph loops, gates, hooks, retro, multi-adapter, etc.) but **zero existing pipelines, TUI views, or WebUI pages were updated**. The user-facing product is unchanged. See #609 post-session audit for the full gap analysis.

The `delivery` ontology invariant now states: *"A feature is not done until shipped pipelines use it."*

## Scope

### Phase 1: Pipeline Adoption (CRITICAL)
- [ ] `impl-issue`: implement→test→fix graph loop, thread continuity, retry policies, haiku routing
- [ ] `ops-pr-review`: llm_judge contract for structured scoring
- [ ] ALL pipelines: replace raw `max_attempts` with named retry policies
- [ ] ALL navigator/analysis steps: route to haiku
- [ ] Copy `plan-approve-implement.yaml` to `internal/defaults/pipelines/`
- [ ] Sync all pipeline changes to `internal/defaults/pipelines/`

### Phase 2: TUI + WebUI
- [ ] Health checks for new capabilities (adapters, hooks, retro store)
- [ ] Gate rendering in pipeline graph (different node shape for gate steps)
- [ ] Loop edge visualization (backward arrows, visit counts)
- [ ] Retro viewer in run details
- [ ] Command/conditional step type rendering

### Phase 3: Documentation
- [ ] Guide: graph loops and conditional routing
- [ ] Guide: human approval gates
- [ ] Guide: retry policies
- [ ] Guide: multi-adapter model routing

## Validation Criteria

A user who runs `wave init && wave list pipelines` must see pipelines using the new features. Running `impl-issue` must use a graph loop. The WebUI must render gates and loops differently from regular steps. The TUI health check must verify new capabilities.

## Acceptance Criteria (Derived)

1. `impl-issue` pipeline uses a graph loop (implement→test→fix cycle with `edges` and `max_visits`)
2. `ops-pr-review` quality-review step has `llm_judge` contract like security-review
3. No pipeline step uses raw `max_attempts` without a named `policy`
4. All navigator/reviewer/summarizer/analyst steps specify `model: claude-haiku-4-5`
5. `.wave/pipelines/` and `internal/defaults/pipelines/` are in sync
6. TUI shows step type indicators (gate, command, conditional, pipeline) in run details
7. TUI has retro viewer accessible from run detail view
8. Doctor health checks validate hooks config and retro store
9. Four new guides exist in `docs/guides/`
10. `go test ./...` passes with no regressions
