# Phase 3.3: Evolution trigger heuristics (every-N + drift + retry-rate)

**Issue:** [#1612](https://github.com/re-cinq/wave/issues/1612)
**Repository:** re-cinq/wave
**Labels:** enhancement, pipeline
**State:** OPEN
**Author:** nextlevelshit

## Body

Part of Epic #1565 Phase 3 (evolution loop).

## Goal

Implement `EvolutionService.ShouldEvolve(pipelineName)` deciding when to fire `pipeline-evolve` based on accumulated `pipeline_eval` rows.

## Acceptance criteria

- [ ] `internal/evolution/triggers.go` — `ShouldEvolve(name) (bool, reason)`
- [ ] Heuristics (any-of):
  - **every-N**: ≥10 evals since last evolution AND median judge_score has dropped ≥0.1 vs prior window
  - **drift**: contract_pass rate dropped >15% over last 20 evals
  - **retry-rate**: avg retry_count >2.0 over last 10 evals
- [ ] Threshold values configurable via wave.yaml `evolution:` block (defaults baked)
- [ ] Test: synthetic eval rows → each heuristic fires + composes correctly
- [ ] Wired into post-run hook (#1606) — emit advisory event `evolution_proposed` when trigger fires

## Dependencies

- #1606 EvalSignal hook MERGED
- #1607 pipeline-evolve meta-pipeline MERGED

## Non-goals

- Auto-rollback (deferred F3)
- UI surface for proposals (Phase 3.4)

## Acceptance Criteria (extracted)

1. New file `internal/evolution/triggers.go` exposes `ShouldEvolve(name) (bool, reason)`.
2. Three heuristics evaluated as OR composition.
3. Thresholds configurable via wave.yaml `evolution:` block; defaults compiled-in.
4. Unit tests exercise each heuristic and combinations against synthetic eval rows.
5. Post-run hook emits an advisory event with state `evolution_proposed` when trigger fires.

## Open clarifications (from assessment, decided in plan)

- **wave.yaml schema**: top-level `evolution:` block, fields named `every_n_window`, `every_n_judge_drop`, `drift_window`, `drift_pass_drop`, `retry_window`, `retry_avg_threshold`, `enabled`.
- **Event payload**: reuse `event.Event` with `State: "evolution_proposed"`, `PipelineID: runID`, `Message: "<reason>"`, plus pipeline name carried in Message.
