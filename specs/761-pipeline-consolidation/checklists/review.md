# Requirements Quality Review Checklist

**Feature**: Pipeline Full Implementation Cycle Consolidation (#761)
**Generated**: 2026-04-09
**Artifacts Reviewed**: spec.md, plan.md, data-model.md, tasks.md

## Completeness

- [ ] CHK001 - Are all five audit dimensions (security, correctness, architecture, tests, coverage) fully specified with input/output definitions? [Completeness]
- [ ] CHK002 - Does FR-027/FR-028 (ops-refresh as optional pre-flight) have corresponding tasks, or is the deliberate omission (per research) documented in the spec itself? [Completeness]
- [ ] CHK003 - Are failure recovery behaviors specified for each composition primitive (iterate failure, aggregate partial input, loop exhaustion)? [Completeness]
- [ ] CHK004 - Is the behavior defined when a disabled audit dimension (via enable_audit_* config) affects the aggregate step's expected input count? [Completeness]
- [ ] CHK005 - Are artifact naming conventions specified for all five audit findings outputs so the aggregate step can discover them? [Completeness]
- [ ] CHK006 - Is the pipeline_outputs section fully specified, including how the final PR URL propagates from create-pr through review-loop to the pipeline output? [Completeness]
- [ ] CHK007 - Does the spec define what happens when wave-land's merge step is "skipped" — is there an explicit configuration or is this assumed behavior? [Completeness]
- [ ] CHK008 - Are workspace isolation requirements specified for the rework loop (does rework reuse the existing worktree or create a new one)? [Completeness]

## Clarity

- [ ] CHK009 - Is the severity mapping between existing schema (critical/high/medium/low/info) and new schema (critical/major/minor/suggestion) unambiguously defined in one canonical location? [Clarity]
- [ ] CHK010 - Is the distinction between `audit-aggregate.yaml` (separate pipeline per T009) and the `aggregate:` composition primitive (mentioned in plan Phase 3) clear — are these the same thing or different mechanisms? [Clarity]
- [ ] CHK011 - Is "rework_only: true" behavior precisely defined — what does the rework step receive as input (aggregated_feedback string? full findings array? both?)? [Clarity]
- [ ] CHK012 - Is the conditional routing mechanism explicit — does the loop primitive natively support `until: gate.decision == "pass"` or does this require custom step logic? [Clarity]
- [ ] CHK013 - Is it clear which persona and model tier each new step uses (especially rework-gate: navigator vs summarizer, balanced vs cheapest)? [Clarity]
- [ ] CHK014 - Does the spec clarify whether the review loop's "COMMENT-only" termination (FR-022/FR-026) means zero REQUEST_CHANGES findings or a specific verdict enum value? [Clarity]

## Consistency

- [ ] CHK015 - Does the shared-findings.schema.json type enum extension (adding correctness/architecture/test/coverage) match the exact strings used in the data-model entity definition? [Consistency]
- [ ] CHK016 - Does the existing shared-findings severity enum (critical/high/medium/low/info) conflict with the new data-model severity enum (critical/major/minor/suggestion), and is the resolution strategy consistent across all artifacts? [Consistency]
- [ ] CHK017 - Is the rework gate's decision logic consistent between spec (FR-014-FR-020: "critical or major findings → fail") and plan (Phase 2.2: "critical or major → fail")? [Consistency]
- [ ] CHK018 - Are max_iterations defaults consistent across spec (FR-020/FR-026: default 3), plan (Phase 4.1: 3), tasks (T014: 3), and data-model (max 10)? [Consistency]
- [ ] CHK019 - Is the audit-aggregate step consistently described — T009 says "navigator step" but plan Phase 2.1 says it uses the aggregate composition primitive? [Consistency]
- [ ] CHK020 - Does T004 (extend shared-findings type enum) acknowledge that the existing severity levels differ from the new schemas, or does it only address type enum? [Consistency]
- [ ] CHK021 - Is the review verdict enum consistent between data-model (APPROVE/REQUEST_CHANGES/COMMENT/REJECT) and spec FR-022 (APPROVE/COMMENT-only) — is REJECT a valid terminal state? [Consistency]

## Coverage

- [ ] CHK022 - Are all 8 edge cases in the spec traceable to specific handling in the plan or task definitions? [Coverage]
- [ ] CHK023 - Does the task list cover the severity mapping issue between old and new schemas, or is this deferred to prompt engineering (T010)? [Coverage]
- [ ] CHK024 - Are acceptance scenarios for US2 (five-dimension auditing) traceable to specific tasks that create the audit pipelines? [Coverage]
- [ ] CHK025 - Are acceptance scenarios for US3 (rework gating) traceable to tasks that implement gate logic and routing? [Coverage]
- [ ] CHK026 - Is there a task covering the edge case "audit step fails to produce valid output" (malformed findings handling in aggregate)? [Coverage]
- [ ] CHK027 - Is there a task or requirement addressing the edge case "merge conflicts during fix application" beyond stating "attempt rebase"? [Coverage]
- [ ] CHK028 - Are all new files listed in the plan's Project Structure section accounted for in the task list? [Coverage]
- [ ] CHK029 - Is the `audit-aggregate.md` prompt (listed in plan) covered by a specific task? [Coverage]
