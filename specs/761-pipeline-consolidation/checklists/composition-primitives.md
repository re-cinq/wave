# Composition Primitives Checklist

**Feature**: Pipeline Full Implementation Cycle Consolidation (#761)
**Generated**: 2026-04-09
**Focus**: Requirements quality for iterate/aggregate/loop primitive usage

This checklist validates that requirements around Wave's composition primitives
are sufficiently specified to avoid the known wiring bugs documented in project memory.

## Iterate Primitive

- [ ] CHK-CP01 - Is the iterate primitive's item list defined as a static list of pipeline names or a dynamic reference to enabled audits from config? [Clarity]
- [ ] CHK-CP02 - Are failure semantics for iterate specified (fail-fast on first audit error vs collect-all-then-report)? [Completeness]
- [ ] CHK-CP03 - Is max_concurrent: 5 a requirement or an implementation detail? If a requirement, is the rationale documented? [Clarity]
- [ ] CHK-CP04 - Does the spec define how each iterate child pipeline receives its input artifacts (worktree path, issue assessment)? [Completeness]

## Aggregate Primitive

- [ ] CHK-CP05 - Is the merge_arrays strategy requirement specifying which array field to merge (findings array within each schema output)? [Completeness]
- [ ] CHK-CP06 - Does the spec define aggregate behavior when fewer than 5 audit results are available (disabled audits or audit failures)? [Completeness]
- [ ] CHK-CP07 - Is it specified whether the aggregate step is a separate pipeline (audit-aggregate.yaml per T009) or an inline aggregate primitive (per plan Phase 3)? [Consistency]

## Loop Primitive

- [ ] CHK-CP08 - Is the rework loop's re-entry point specified (does it re-run from test-gen, from audit-iterate, or from rework only)? [Clarity]
- [ ] CHK-CP09 - Does the rework loop's `until` condition reference a specific field path in the verdict schema (e.g., `verdict.decision == "pass"`)? [Clarity]
- [ ] CHK-CP10 - Is the review loop's termination on COMMENT explicitly distinguished from termination on APPROVE (same behavior or different exit codes)? [Clarity]
- [ ] CHK-CP11 - Are loop iteration counters specified as starting from 0 or 1, and is this consistent between rework loop and review loop? [Consistency]
- [ ] CHK-CP12 - Is the pipeline's terminal state defined for loop exhaustion (max iterations reached without pass) — is this a pipeline failure or a degraded success? [Completeness]
