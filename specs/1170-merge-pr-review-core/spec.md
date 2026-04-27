# scope: merge ops-pr-review-core.yaml into ops-pr-review.yaml via a profile flag

**Issue:** [re-cinq/wave#1170](https://github.com/re-cinq/wave/issues/1170)
**Labels:** scope-audit
**State:** OPEN
**Author:** nextlevelshit

## Context

From wave-scope-audit run wave-scope-audit-20260422-075324-5b6c.

Pipeline consolidation follow-up: `ops-pr-review-core.yaml` is a near-duplicate of `ops-pr-review.yaml`. Scope action: merge via a profile flag to eliminate the fork while preserving the "core" variant behavior.

## Acceptance Criteria

- [ ] `ops-pr-review-core.yaml` merged into `ops-pr-review.yaml`
- [ ] Profile/flag selects core vs full behavior
- [ ] `ops-pr-review-core.yaml` deleted
- [ ] Both prior behaviors reproducible via the single pipeline

## Prior Attempt

Pipeline run `impl-issue-20260426-234754-eb3d` (Batch B of epic #1403) stalled at the implement step on cheapest model after 22 min. Partial work added a `SubPipelineConfig.Env` field; YAML inline merge was not attempted. Recommendation: stronger model or manual decomposition.

## Source Inventory

Two near-duplicate files exist in both `internal/defaults/pipelines/` and `.agents/pipelines/`:

- `ops-pr-review.yaml` — composition wrapper. Calls `ops-pr-review-core` then runs `publish` step that posts a PR comment via `gh pr comment`.
- `ops-pr-review-core.yaml` — the actual review machinery. 5 steps: `diff-analysis`, `security-review`, `quality-review`, `slop-review`, `summary`. Produces `verdict` artifact. No forge interaction.

`inception-bugfix.yaml` (line 60-63) embeds `ops-pr-review-core` as a sub-pipeline after its `fix` step — the only known external consumer of the core variant.

## Definition of "core" vs "full"

- **core**: 5-step review producing `review-verdict.json` artifact only. No comment posted.
- **full**: core + `publish` step that formats verdict as markdown and posts to PR via `gh pr comment`.
