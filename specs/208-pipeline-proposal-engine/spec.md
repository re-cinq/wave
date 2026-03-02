# feat(pipeline): pipeline proposal engine — context-aware sequencing from health artifact

**Issue**: [re-cinq/wave#208](https://github.com/re-cinq/wave/issues/208)
**Parent**: [re-cinq/wave#184](https://github.com/re-cinq/wave/issues/184)
**Labels**: enhancement, needs-design, pipeline, priority: high
**Author**: nextlevelshit
**State**: OPEN

## Summary

Implement the pipeline proposal engine that consumes the codebase health analysis artifact and the available pipeline catalog to propose optimal pipeline sequences. The engine applies contextual intelligence (nous) and practical judgment (phronesis) to recommend what pipelines to run, in what order, and which can run in parallel. This may extend or replace the existing `wave meta` / `MetaPipelineExecutor` (`internal/pipeline/meta.go`) which already supports dynamic child pipeline generation via the philosopher persona.

## Acceptance Criteria

- [ ] Engine consumes the health analysis artifact (from #207) as primary input
- [ ] Engine discovers available pipelines from the manifest/pipeline catalog
- [ ] Engine produces a structured proposal: ordered list of pipeline runs with dependency edges
- [ ] Proposals identify pipelines that can run in parallel (independent inputs)
- [ ] Proposals include rationale for each recommended pipeline (why it's relevant given codebase state)
- [ ] Engine filters pipelines by detected forge type (only propose `gh-*` for GitHub repos, etc.)
- [ ] Proposal output is a structured artifact consumable by the interactive TUI
- [ ] Engine handles edge cases: no actionable proposals, all pipelines already completed, conflicting recommendations
- [ ] Relationship with existing `wave meta` (`internal/pipeline/meta.go`) is resolved — either extends it or documents why a separate engine is needed

## Dependencies

- #207 — Codebase health analysis (provides the input artifact)
- #206 — System readiness checks (ensures forge detection is available)

## Scope Notes

- **In scope**: Proposal generation logic, pipeline catalog discovery, forge-aware filtering, dependency ordering, parallel eligibility detection, structured proposal output
- **Out of scope**: Executing the proposed pipelines (that's the orchestrator's job), interactive UI for selecting proposals (#208), creating new pipeline definitions on the fly
- **Design decision needed**: Whether to extend `MetaPipelineExecutor` or build a separate proposal engine. The existing meta executor uses a philosopher persona to generate child pipelines dynamically — the proposal engine could be a pipeline step that produces proposals as artifacts
- **Overlap note**: #95 covers stabilizing `wave meta` — this issue should coordinate with or build upon that work

## Metadata

| Field | Value |
|-------|-------|
| Complexity | Complex |
| Quality Score | 78/100 |
| Missing Info | Health artifact schema (#207 pending), forge detection API (#206 pending), proposal output schema (to be designed), MetaPipelineExecutor extend-vs-replace decision, #95 coordination strategy |
