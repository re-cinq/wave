# Tasks: Pipeline Full Implementation Cycle Consolidation

**Feature Branch**: `761-pipeline-consolidation`
**Generated**: 2026-04-09
**Source**: [spec.md](./spec.md), [plan.md](./plan.md), [data-model.md](./data-model.md)
**Total Tasks**: 18

## Phase 1: Setup

- [X] T001 Setup Create prompt directory structure: `.wave/prompts/audit/` and `.wave/prompts/full-impl-cycle/`

## Phase 2: Foundation (blocking prerequisites)

- [X] T002 [P] Foundation Copy `specs/761-pipeline-consolidation/contracts/aggregated-findings.schema.json` to `.wave/contracts/aggregated-findings.schema.json`
- [X] T003 [P] Foundation Copy `specs/761-pipeline-consolidation/contracts/rework-gate-verdict.schema.json` to `.wave/contracts/rework-gate-verdict.schema.json`
- [X] T004 Foundation Extend `.wave/contracts/shared-findings.schema.json` type enum to add `"correctness"`, `"architecture"`, `"test"`, `"coverage"` values alongside existing enum entries. Note: existing severity levels are `critical/high/medium/low/info` while new schemas use `critical/major/minor/suggestion` — the rework gate prompt must handle this mapping.

## Phase 3: US2 — Five-Dimension Auditing (P2, blocks US1)

- [X] T005 [P] US2 Create `.wave/pipelines/audit-correctness.yaml` (two-step: scan + report) and `.wave/prompts/audit/correctness-scan.md` navigator prompt. Scan step reads issue assessment artifact, analyzes implementation against issue requirements, checks for logic errors and missing features. Output: `shared-findings.schema.json` with `type="correctness"`. Report step formats findings as markdown. Follow `audit-security.yaml` pattern (persona, model, workspace, contract structure).
- [X] T006 [P] US2 Create `.wave/pipelines/audit-architecture.yaml` (two-step: scan + report) and `.wave/prompts/audit/architecture-scan.md` navigator prompt. Scan step analyzes package structure, import graph, coupling issues, misplaced packages, and pattern violations. Output: `shared-findings.schema.json` with `type="architecture"`. Report step formats findings as markdown. Follow `audit-security.yaml` pattern.
- [X] T007 [P] US2 Create `.wave/pipelines/audit-tests.yaml` (two-step: scan + report) and `.wave/prompts/audit/tests-scan.md` navigator prompt. Scan step analyzes test coverage, test quality, missing tests, untested code paths, and gaps (happy-path only, no error cases). Output: `shared-findings.schema.json` with `type="test"`. Report step formats findings as markdown. Follow `audit-security.yaml` pattern.
- [X] T008 [P] US2 Create `.wave/pipelines/audit-coverage.yaml` (two-step: scan + report) and `.wave/prompts/audit/coverage-scan.md` navigator prompt. Scan step parses issue acceptance criteria and validates implementation addresses each criterion. Lists unaddressed or partially addressed requirements. Output: `shared-findings.schema.json` with `type="coverage"`. Report step formats findings as markdown. Follow `audit-security.yaml` pattern.

## Phase 4: US3 — Rework Gate & Aggregation (P3, blocks US1)

- [X] T009 US3 Create `.wave/pipelines/audit-aggregate.yaml` pipeline with a navigator step that reads findings artifacts from all five audit pipelines and merges them into a flattened array. Create `.wave/prompts/full-impl-cycle/audit-aggregate.md` prompt. Output: `.wave/contracts/aggregated-findings.schema.json`. Include `source_audits` array tracking which audits contributed.
- [X] T010 US3 Create `.wave/prompts/full-impl-cycle/rework-gate.md` navigator prompt for audit verdict synthesis. Prompt must: read aggregated findings, count findings by severity, compare against configured threshold (default: fail on critical/major), produce `rework-gate-verdict.schema.json` output with decision (pass/fail), reason, iteration, findings_summary, and aggregated_feedback (on fail). Gate logic: any critical/high finding → fail; only medium/low/info → pass.

## Phase 5: US1 + US4 — Main Composition Pipeline (P1 + P2)

- [X] T011 US1 Create `.wave/pipelines/full-impl-cycle.yaml` with metadata (name, description, category: composition, release: true), input (source: cli, example: issue URL), hooks, chat_context, skills, and core step sequence: (1) `impl` step referencing `impl-issue-core` sub-pipeline, (2) `test-gen` step referencing `test-gen` sub-pipeline with dependency on impl. Follow `impl-review-loop.yaml` for sub-pipeline reference pattern.
- [X] T012 US1 Add audit + rework gate steps to `full-impl-cycle.yaml`: (3) `audit-iterate` step using `iterate:` primitive over the 5 audit pipeline names with `mode: parallel`, `max_concurrent: 5`, dependent on test-gen; (4) `merge-findings` step using `aggregate:` primitive with `merge_arrays` strategy, dependent on audit-iterate; (5) `rework-gate` navigator step reading merged findings, using rework-gate.md prompt, producing verdict with `rework-gate-verdict.schema.json` contract; (6) `rework-loop` using `loop:` primitive with `max_iterations: 3` containing rework (craftsman, `rework_only: true`), re-test-gen, re-audit-iterate, re-aggregate, and re-gate steps, conditional on gate decision=fail.
- [X] T013 US1+US4 Add PR creation and review loop to `full-impl-cycle.yaml`: (7) `create-pr` step referencing `wave-land` sub-pipeline, conditional on gate decision=pass; (8) `review` step referencing `ops-pr-review-core`, dependent on create-pr; (9) `review-loop` using `loop:` primitive with `max_iterations: 3` and `until: verdict == 'APPROVE'`, containing fix (pipeline: `ops-pr-fix-review`) and re-review (pipeline: `ops-pr-review-core`) steps. Add `pipeline_outputs` section exposing final PR URL.

## Phase 6: Configuration

- [X] T014 Configuration Add `full_impl_cycle` params section to `wave.yaml` with: `max_rework_iterations: 3`, `max_review_iterations: 3`, `audit_severity_threshold: "major"`, `enable_audit_security: true`, `enable_audit_correctness: true`, `enable_audit_architecture: true`, `enable_audit_tests: true`, `enable_audit_coverage: true`.
- [X] T015 Configuration Wire `{{ params.full_impl_cycle.* }}` template variables into `full-impl-cycle.yaml`: max_iterations in rework loop, max_iterations in review loop, severity threshold in rework-gate prompt injection, audit enable/disable flags in iterate over list.

## Phase 7: Validation & Cross-cutting

- [X] T016 [P] Validation Validate all new pipeline YAML files for correct syntax and valid cross-references: `wave validate .wave/pipelines/full-impl-cycle.yaml`, `wave validate .wave/pipelines/audit-correctness.yaml`, `wave validate .wave/pipelines/audit-architecture.yaml`, `wave validate .wave/pipelines/audit-tests.yaml`, `wave validate .wave/pipelines/audit-coverage.yaml`, `wave validate .wave/pipelines/audit-aggregate.yaml`.
- [X] T017 [P] Validation Validate new contract schemas for JSON Schema Draft 7 compliance: parse `.wave/contracts/aggregated-findings.schema.json` and `.wave/contracts/rework-gate-verdict.schema.json` with `jq .`. Verify extended `.wave/contracts/shared-findings.schema.json` still has valid enum values.
- [X] T018 [P] Validation Run `go test ./...` to verify no regressions from shared-findings schema extension. Confirm existing audit pipelines (`audit-security`, `audit-dead-code`, etc.) are unaffected.

## Dependency Graph

```
T001 ──┬──▶ T002 ──┐
       ├──▶ T003 ──┤
       └──▶ T004 ──┤
                   │
                   ├──▶ T005 ──┐
                   ├──▶ T006 ──┤
                   ├──▶ T007 ──┤
                   ├──▶ T008 ──┤
                   │           │
                   │           ├──▶ T009 ──▶ T010 ──┐
                   │           │                    │
                   │           └────────────────────┤
                   │                                │
                   └────────────────────────────────┤
                                                    │
                                                    ├──▶ T011 ──▶ T012 ──▶ T013
                                                    │
                                                    └──▶ T014 ──▶ T015
                                                                    │
                                                    T013 + T015 ────┤
                                                                    │
                                                                    ├──▶ T016
                                                                    ├──▶ T017
                                                                    └──▶ T018
```

## Notes

- **Severity mapping**: Existing `shared-findings.schema.json` uses severity `critical/high/medium/low/info`. New contract schemas (`aggregated-findings`, `rework-gate-verdict`) use `critical/major/minor/suggestion`. The rework gate prompt (T010) must map between these: high→major, medium→minor, low/info→suggestion. This avoids breaking changes to the existing schema.
- **Two-step audit pattern**: New audits use simplified two-step (scan + report) instead of audit-security's three-step (scan + deep-dive + report). The deep-dive step is security-specific and not applicable to correctness/architecture/tests/coverage.
- **ops-refresh omitted**: Research determined `impl-issue-core`'s fetch-assess step already retrieves current issue data, making a separate refresh step redundant.
- **Coexistence**: Existing pipelines remain unchanged. The new `full-impl-cycle` pipeline composes them via sub-pipeline references.
