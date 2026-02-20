# Requirements Quality Review Checklist

**Feature**: Pipeline Output UX — Surface Key Outcomes
**Spec**: `specs/120-pipeline-output-ux/spec.md`
**Date**: 2026-02-20

---

## Completeness

- [ ] CHK001 - Are all output modes (auto, text, json, quiet) explicitly addressed for the outcomes section behavior? [Completeness]
- [ ] CHK002 - Is the behavior of outcome rendering for partially-completed pipelines (some steps succeeded, some failed) fully specified? [Completeness]
- [ ] CHK003 - Are requirements defined for how outcomes behave when a pipeline is resumed from a failed step (via --from-step)? [Completeness]
- [ ] CHK004 - Is the visual hierarchy between the outcomes section and the existing success/duration/token line explicitly defined (ordering, separation)? [Completeness]
- [ ] CHK005 - Are requirements specified for how TypeBranch deliverables handle branch reuse across shared worktree steps (deduplication)? [Completeness]
- [ ] CHK006 - Is the behavior defined for when multiple PRs or multiple branches are created by a single pipeline run? [Completeness]
- [ ] CHK007 - Are the exact conditions for suppressing next steps defined beyond quiet mode (e.g., JSON mode, non-interactive terminals)? [Completeness]
- [ ] CHK008 - Is there a requirement for how the outcomes section handles pipelines with zero steps completed (immediate failure)? [Completeness]

## Clarity

- [ ] CHK009 - Is the distinction between "outcome-worthy" and "detail-level" deliverable types clearly enumerated with no ambiguity about which types fall in each category? [Clarity]
- [ ] CHK010 - Is the rendering format for each outcome category (branch, PR, issue, deployment, report) specified with enough precision to implement without guesswork? [Clarity]
- [ ] CHK011 - Is the exact format of the artifact summary line defined (e.g., "5 artifacts produced" vs "5 artifacts" vs "Artifacts: 5")? [Clarity]
- [ ] CHK012 - Is the contract summary line format defined for mixed pass/fail scenarios (e.g., "2/3 contracts passed" with the failure shown separately)? [Clarity]
- [ ] CHK013 - Is "prominently" in FR-001 and FR-004 defined with measurable criteria (line position, formatting style, colors)? [Clarity]
- [ ] CHK014 - Is the verbose mode rendering clearly specified — does it replace the summary or append detail below it? [Clarity]
- [ ] CHK015 - Is the relationship between OutcomeSummary rendering and the existing FormatSummary() output defined (replacement vs augmentation)? [Clarity]

## Consistency

- [ ] CHK016 - Are the deliverable type priority orderings consistent between the spec (Edge Case 3), clarification C-002, and the plan (D-006)? [Consistency]
- [ ] CHK017 - Is the "top 5" truncation rule (C-002) consistently applied in both the plan's BuildOutcome logic and the RenderOutcomeSummary logic? [Consistency]
- [ ] CHK018 - Are the JSON field names in OutcomesJSON consistent with the acceptance scenario field names in US3 (e.g., `outcomes.branch` vs `outcomes.Branch`)? [Consistency]
- [ ] CHK019 - Is FR-012's quiet mode behavior consistent across US4-AC3 and the plan's Phase 5 (T019)? [Consistency]
- [ ] CHK020 - Are the Metadata key names for branch deliverables ("pushed", "remote_ref", "push_error") used consistently between data-model.md, C-001, and the plan's Layer 2? [Consistency]
- [ ] CHK021 - Does the plan's BuildOutcome() signature in T010 match the inputs described in Layer 5 of plan.md? [Consistency]

## Coverage

- [ ] CHK022 - Do acceptance scenarios cover the case where a pipeline produces only deployments (no branch, no PR, no issues)? [Coverage]
- [ ] CHK023 - Is there a scenario testing the output when all contracts fail (not just partial failure)? [Coverage]
- [ ] CHK024 - Is there an acceptance scenario for the transition from progress display (TUI/basic) to the outcomes section (timing, clearing)? [Coverage]
- [ ] CHK025 - Are edge cases for the JSON output format covered (e.g., empty arrays vs null for missing outcome categories per US3-AC3)? [Coverage]
- [ ] CHK026 - Is there coverage for pipelines that create issues but no PRs (to ensure issue detection is independently testable)? [Coverage]
- [ ] CHK027 - Is there an edge case for concurrent pipeline runs and how outcomes are isolated per run? [Coverage]
