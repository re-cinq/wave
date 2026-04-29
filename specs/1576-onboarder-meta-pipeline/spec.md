# 1.1: Onboarder agent + onboard-project meta-pipeline

**Issue:** [re-cinq/wave#1576](https://github.com/re-cinq/wave/issues/1576)
**Author:** nextlevelshit
**State:** OPEN
**Labels:** (none)
**Branch:** `1576-onboarder-meta-pipeline`

## Body

Part of Epic #1565. Phase 1, depends on PRE-1 (#1566 ✓), PRE-2 (#1569 ✓).

Add the onboarder persona + meta-pipeline that generates per-project `.agents/*` from project introspection. This is the first user-facing artefact of the onboarding-as-session vision.

**Files:**
- New: `internal/defaults/personas/onboarder.md`
- New: `internal/defaults/pipelines/onboard-project.yaml`
- New: `internal/defaults/contracts/detection.schema.json`
- New: prompts under `internal/defaults/prompts/onboard/`

**Pattern reference:** `internal/defaults/pipelines/ops-bootstrap.yaml` (existing greenfield pattern).

**Acceptance:**
- [ ] `wave run onboard-project` produces `.agents/personas/*`, `.agents/pipelines/*`, `.agents/prompts/*`
- [ ] `detection.schema.json` validates the JSON shape produced by the persona
- [ ] Smoke run on a throwaway Go repo + Node repo (real verification per memory)
- [ ] Sentinel `.agents/.onboarding-done` written on completion (PRE-2 helper `MarkDoneAt`)

**Pipeline:** `impl-issue` (`--adapter claude --model cheapest`)

## Acceptance Criteria

1. `wave run onboard-project` produces:
   - `.agents/personas/*` — at least one project-tailored persona
   - `.agents/pipelines/*` — at least one project-tailored pipeline
   - `.agents/prompts/*` — prompt files referenced by generated pipelines
2. `detection.schema.json` (in `internal/defaults/contracts/`) validates the JSON shape produced by the onboarder persona during the detection step.
3. Smoke run completes successfully on:
   - A throwaway Go repo (greenfield-ish)
   - A throwaway Node repo
4. Sentinel `.agents/.onboarding-done` is written on completion using the existing PRE-2 helper `onboarding.MarkDoneAt`.

## Dependencies

- PRE-1 (#1566) — service layer (merged)
- PRE-2 (#1569) — onboarding rewrite + `MarkDoneAt` sentinel helper (merged)
- Pattern: `internal/defaults/pipelines/ops-bootstrap.yaml`

## Reference

- Plan doc: `docs/scope/onboarding-as-session-plan.md` § Phase 1, row 1.1
- Sentinel constant: `internal/onboarding/service.go` `SentinelFile = ".agents/.onboarding-done"`
- Helper: `internal/onboarding/service.go` `MarkDoneAt(projectDir string) error`
