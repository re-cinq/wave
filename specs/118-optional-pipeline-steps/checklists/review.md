# Requirements Quality Review: Optional Pipeline Steps

**Feature**: 118-optional-pipeline-steps
**Date**: 2026-02-20
**Artifacts Reviewed**: spec.md, plan.md, tasks.md, data-model.md, research.md, contracts/

---

## Completeness

- [ ] CHK001 - Are all user stories traced to at least one functional requirement (FR-xxx)? [Completeness]
- [ ] CHK002 - Does every functional requirement (FR-001 through FR-012) have at least one corresponding task in tasks.md? [Completeness]
- [ ] CHK003 - Does every functional requirement have at least one behavioral contract in contracts/behavior-contract.md? [Completeness]
- [ ] CHK004 - Are all six edge cases enumerated in spec.md covered by at least one task or test in tasks.md? [Completeness]
- [ ] CHK005 - Does the spec define the expected behavior when an optional step produces partial output (adapter starts writing artifacts then fails mid-stream)? [Completeness]
- [ ] CHK006 - Are error messages for skipped steps specified with enough detail for operators to diagnose why a step was skipped (which upstream optional step failed, which artifact was unavailable)? [Completeness]
- [ ] CHK007 - Does the spec address what happens to in-flight events/progress display when an optional step is transitioning from "retrying" to "failed_optional"? [Completeness]
- [ ] CHK008 - Is the constitution amendment for P12 (adding 6th state) tracked as an explicit task or deliverable? [Completeness]

---

## Clarity

- [ ] CHK009 - Is the distinction between `dependencies` (ordering-only) and `memory.inject_artifacts` (data coupling) defined unambiguously in the spec, not just in CLR-002? [Clarity]
- [ ] CHK010 - Is the term "skipped" used consistently across all artifacts — does spec.md, plan.md, data-model.md, and tasks.md all agree on when a step enters "skipped" state vs "failed_optional" state? [Clarity]
- [ ] CHK011 - Does the spec clearly state whether "skipped" is a new StepState constant that must be added, or whether it reuses the existing `display.StateSkipped` — and if reused, is it clear this is only a display constant today? [Clarity]
- [ ] CHK012 - Are the state transition rules specified precisely enough that two independent implementers would produce the same state machine? [Clarity]
- [ ] CHK013 - Is it clear whether the `Optional` field on Event is set for "skipped" events (downstream steps skipped due to optional failure) or only for steps that are themselves marked `optional: true`? [Clarity]
- [ ] CHK014 - Does the spec define what "descriptive message" means for skipped step logging (FR-007) — is there a format or minimum content requirement? [Clarity]

---

## Consistency

- [ ] CHK015 - Does data-model.md's state transition diagram match the textual state transitions described in spec.md and research.md? [Consistency]
- [ ] CHK016 - Are the PipelineStatus struct changes in data-model.md (adding FailedOptionalSteps, SkippedSteps) consistent with how tasks.md references them in T007, T008, and T011? [Consistency]
- [ ] CHK017 - Does task T015 (dependency skipping) match the skipping logic described in CLR-002 and research Decision 2 — specifically, does the task correctly scope skipping to `inject_artifacts` references only? [Consistency]
- [ ] CHK018 - Are the event field additions in T004 (emitter.go) consistent with the Event struct changes specified in data-model.md section 3? [Consistency]
- [ ] CHK019 - Does the `completedAt` update logic in T006 match data-model.md's specification that `StateFailedOptional` gets a `completed_at` timestamp? [Consistency]
- [ ] CHK020 - Is the display color/icon for `StateFailedOptional` consistently described across T017 (capability.go), T019 (bubbletea_model.go), and T018 (bubbletea_progress.go)? [Consistency]
- [ ] CHK021 - Do tasks T022, T023, T024 (resume) correctly align with research Decision 8's approach of treating failed_optional as "completed-like" for resume purposes? [Consistency]

---

## Coverage

- [ ] CHK022 - Are there acceptance scenarios covering the interaction between optional steps and matrix strategies (Step.Strategy field)? [Coverage]
- [ ] CHK023 - Is there a requirement or edge case addressing what happens when a step has `inject_artifacts` from BOTH a failed optional step AND a successful step? [Coverage]
- [ ] CHK024 - Does the spec address observability beyond events — specifically, are audit log entries for failed optional steps defined (internal/audit package)? [Coverage]
- [ ] CHK025 - Is there a requirement for how `wave ops status` CLI output represents pipelines with optional failures? [Coverage]
- [ ] CHK026 - Does the spec address whether the web dashboard (085-web-operations-dashboard) needs updates to render the new `failed_optional` state? [Coverage]
- [ ] CHK027 - Are there requirements for how optional step failures interact with pipeline-level `on_failure` or notification hooks (if any exist)? [Coverage]
- [ ] CHK028 - Does the spec define behavior when an optional step's workspace cleanup fails after the step is marked `failed_optional`? [Coverage]
- [ ] CHK029 - Is there a task or requirement for updating pipeline YAML documentation/examples to show the new `optional` field? [Coverage]
- [ ] CHK030 - Are there requirements for how `wave run --from-step` interacts with optional steps that precede the starting step? [Coverage]
