# Task-Plan Alignment Checklist: Init Merge & Upgrade Workflow

**Feature**: #230 | **Date**: 2026-03-04
**Focus**: Cross-artifact consistency between spec.md, plan.md, and tasks.md

---

## Spec-to-Plan Traceability

- [ ] CHK301 - Does the plan address all 16 functional requirements (FR-001 through FR-016)? Are any requirements missing from the implementation approach? [Coverage]
- [ ] CHK302 - Does the plan's Phase A-G structure map cleanly to the spec's User Story priorities (P1 stories implemented before P2)? [Consistency]
- [ ] CHK303 - Does the plan account for all 7 edge cases listed in the spec, or are some deferred/omitted without justification? [Coverage]
- [ ] CHK304 - Is the plan's decision to put all types in `init.go` consistent with the project's package structure conventions (single responsibility per package)? [Consistency]

## Plan-to-Tasks Traceability

- [ ] CHK305 - Does every task in tasks.md map to a specific phase and function in the plan? Are there orphan tasks with no plan backing? [Consistency]
- [ ] CHK306 - Are task dependencies (blocking prerequisites) correctly identified — can T005 (confirmMerge) be implemented before T004 (displayChangeSummary) completes? [Consistency]
- [ ] CHK307 - Do the test tasks (T012-T016) collectively cover all acceptance scenarios from all user stories? [Coverage]
- [ ] CHK308 - Is T011 (verify migrate handles missing DB) marked as verification-only [P], and is that appropriate given the edge case's criticality? [Consistency]

## Spec-to-Tasks Traceability

- [ ] CHK309 - Is every acceptance scenario from spec.md traceable to at least one implementation task and one test task? [Coverage]
- [ ] CHK310 - Are the success criteria (SC-001 through SC-007) each verifiable by the defined test tasks? [Coverage]
- [ ] CHK311 - Does the task ordering respect User Story priorities — all P1 tasks before P2 tasks before P3 tasks? [Consistency]
- [ ] CHK312 - Are there any requirements in the spec that are not addressed by any task (gap analysis)? [Coverage]

## Data Model Alignment

- [ ] CHK313 - Do the data model types (FileChangeEntry, ManifestChangeEntry, ChangeSummary) in data-model.md match the types referenced in plan.md and tasks.md? [Consistency]
- [ ] CHK314 - Are the invariants defined in data-model.md enforceable by the implementation described in the plan? [Consistency]
- [ ] CHK315 - Is the "ephemeral" lifecycle of ChangeSummary consistent with the plan's pre-mutation pattern (no persistence needed)? [Consistency]
