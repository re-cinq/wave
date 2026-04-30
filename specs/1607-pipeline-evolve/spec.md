# Phase 3.2: pipeline-evolve meta-pipeline

**Issue:** [#1607](https://github.com/re-cinq/wave/issues/1607)
**Repository:** re-cinq/wave
**Branch:** `1607-pipeline-evolve-impl`
**Labels:** `enhancement`, `ready-for-impl`
**State:** OPEN
**Author:** nextlevelshit

## Issue Body

Part of Epic #1565 Phase 3.

### Goal

Ship `internal/defaults/embedfs/pipelines/pipeline-evolve.yaml` — a meta-pipeline that ingests `pipeline_eval` history for a target pipeline and proposes a new version (yaml + persona prompt diffs) via LLM.

### Acceptance criteria

- [ ] pipeline-evolve.yaml with steps: gather-eval (fetch from pipeline_eval), analyze (find recurring failures), propose (generate v+1 yaml + prompt diffs), record (insert into evolution_proposal as 'proposed')
- [ ] Default-agnostic per memory feedback_defaults_agnostic
- [ ] Test: run on a synthetic pipeline_eval seed → verify proposal row created with status=proposed

### Dependencies

- 3.1 EvalSignal hook (gathers data) — populates `pipeline_eval` rows at run completion
- PRE-5 evolution_proposal table (MERGED) — defined in `internal/state/migration_definitions.go:612-631`

## Acceptance Criteria (extracted)

1. New file at `internal/defaults/embedfs/pipelines/pipeline-evolve.yaml`
2. Four-step structure: `gather-eval` → `analyze` → `propose` → `record`
3. `gather-eval` queries `pipeline_eval` table for target pipeline
4. `analyze` extracts recurring failure classes / low judge scores
5. `propose` produces a v+1 YAML and persona prompt diffs via an LLM step
6. `record` inserts a row into `evolution_proposal` with `status='proposed'`
7. Pipeline is **default-agnostic** — language/framework neutral; uses `{{ project.* }}` and forge templating only
8. Backed by a test seeded with synthetic `pipeline_eval` rows that verifies proposal creation

## Schema References

- `pipeline_eval`: `internal/state/migration_definitions.go:577-593`
- `evolution_proposal`: `internal/state/migration_definitions.go:612-631`
- Go API: `internal/state/evolution.go` (`EvolutionStore` interface)
- Status enum: `proposed | approved | rejected | superseded`

## Links

- Issue: https://github.com/re-cinq/wave/issues/1607
- Epic: https://github.com/re-cinq/wave/issues/1565
