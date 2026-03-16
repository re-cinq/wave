# Requirements Quality Review: Guided TUI Orchestrator

**Feature**: `248-guided-tui-orchestrator`
**Generated**: 2026-03-16
**Scope**: Cross-cutting quality validation of spec.md, plan.md, tasks.md

---

## Completeness

- [ ] CHK001 - Are all transitions in the GuidedFlowState machine (Health→Proposals, Proposals→Fleet, Fleet→Attached, Attached→Fleet) explicitly covered by acceptance scenarios? [Completeness]
- [ ] CHK002 - Does the spec define behavior for ALL keyboard shortcuts mentioned (Tab, Shift+Tab, Enter, Esc, Space, m, s, y, q, 1-8)? [Completeness]
- [ ] CHK003 - Is the dependency installation prompt (FR-005) covered by an acceptance scenario with Given/When/Then, or only stated as a requirement? [Completeness]
- [ ] CHK004 - Are accessibility requirements specified (screen reader, high-contrast, colorblind-safe glyphs)? [Completeness]
- [ ] CHK005 - Does the spec define what happens when the user invokes `wave` with no subcommand but passes flags (e.g., `wave --debug`)? [Completeness]
- [ ] CHK006 - Is the loading state for proposals explicitly defined (what the user sees between health completion and proposal data arriving per C5)? [Completeness]
- [ ] CHK007 - Are error states for the suggest data provider defined (network failure, doctor scan crash)? [Completeness]
- [ ] CHK008 - Does the spec address the behavior when the user navigates away from health view (via number key) and then health completes — does auto-transition still fire? [Completeness]

## Clarity

- [ ] CHK009 - Is the "1 second" auto-transition delay (FR-003, SC-001) justified, or is it an arbitrary constant that should be configurable? [Clarity]
- [ ] CHK010 - Is the distinction between "infrastructure health" (TUI checks) and "codebase health" (doctor scan) clear enough for implementers, or could they be conflated? [Clarity]
- [ ] CHK011 - Is the "priority rating" on proposals (Story 2) defined — what scale, what values, how is it determined? [Clarity]
- [ ] CHK012 - Is "visually grouped" for sequence-linked runs (FR-015) sufficiently defined to produce a deterministic rendering, or is it open to interpretation? [Clarity]
- [ ] CHK013 - Does "type badge" (FR-006) specify the exact badge labels ([single], [sequence], [parallel]) or are these left ambiguous? [Clarity]
- [ ] CHK014 - Is the "hint message" for failed health checks (FR-016) specified, or left to implementer discretion? [Clarity]

## Consistency

- [ ] CHK015 - Does the Tab behavior spec (FR-011) align with Story 5 acceptance scenarios and clarification C3 — are there any contradictions in how Tab works during the health phase? [Consistency]
- [ ] CHK016 - Does FR-004 (Tab to skip health) conflict with the guided Tab toggle (FR-011) which says Tab toggles between Suggest and Pipelines — what does Tab do during health phase? [Consistency]
- [ ] CHK017 - Is the number key mapping (1-8) consistent between the plan (1→ViewPipelines) and the existing codebase view ordering? [Consistency]
- [ ] CHK018 - Do all 5 key entities in the spec map 1:1 to data model entities, or are any missing/extra? [Consistency]
- [ ] CHK019 - Are task priorities (T001-T038) consistent with the user story priorities they reference (P1/P2/P3)? [Consistency]
- [ ] CHK020 - Does the plan's "~540 LOC" estimate align with the task decomposition scope, or are tasks over/under-scoped relative to the plan? [Consistency]

## Coverage

- [ ] CHK021 - Are there acceptance scenarios for concurrent proposal launches (what if the user launches a sequence while another is already running)? [Coverage]
- [ ] CHK022 - Is the behavior defined when the user presses `m` on a proposal that has no modifiable input field? [Coverage]
- [ ] CHK023 - Does the spec cover the interaction between guided mode and the existing stale run detector in the pipeline view? [Coverage]
- [ ] CHK024 - Is there a success criterion for the DAG preview (P2) rendering correctness, or only for P1 features? [Coverage]
- [ ] CHK025 - Are there edge cases for extremely long pipeline names in DAG rendering (truncation, wrapping)? [Coverage]
- [ ] CHK026 - Does the spec address what happens when a proposal references a pipeline that no longer exists (deleted between suggest and launch)? [Coverage]
- [ ] CHK027 - Is the compose/multi-select workflow (Space + Enter) tested for >10 simultaneous selections given SC-006 (10 concurrent runs)? [Coverage]

## Traceability

- [ ] CHK028 - Does every functional requirement (FR-001 through FR-017) have at least one corresponding task in tasks.md? [Traceability]
- [ ] CHK029 - Does every user story have at least one test task ([P] tagged) in tasks.md? [Traceability]
- [ ] CHK030 - Do all success criteria (SC-001 through SC-008) have corresponding verification tasks? [Traceability]
- [ ] CHK031 - Are all clarification decisions (C1-C5) reflected in the plan's implementation phases? [Traceability]
