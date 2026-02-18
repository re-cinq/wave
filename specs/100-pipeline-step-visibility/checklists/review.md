# Requirements Quality Review Checklist

**Feature**: Pipeline Step Visibility in Default Run Mode
**Spec**: `specs/100-pipeline-step-visibility/spec.md`
**Date**: 2026-02-14

## Completeness

- [ ] CHK001 - Are all six step states (pending, running, completed, failed, skipped, cancelled) explicitly covered by at least one user story or functional requirement? [Completeness]
- [ ] CHK002 - Is the persona name display requirement (FR-002) specified for ALL states, including failed, skipped, and cancelled, not just pending/running/completed? [Completeness]
- [ ] CHK003 - Does the spec define what happens to timing display for failed steps (live timer stops, final duration shown), or only for completed steps (US-3 Scenario 2)? [Completeness]
- [ ] CHK004 - Is the cancelled state's timing behavior explicitly defined? The spec mentions no timing for cancelled in the research display format table, but no FR or user story codifies this. [Completeness]
- [ ] CHK005 - Does the spec address what the step list looks like BEFORE any step begins running (all steps pending at pipeline startup)? [Completeness]
- [ ] CHK006 - Is the behavior defined for steps with empty or missing persona names (e.g., a step configured without a persona field)? [Completeness]
- [ ] CHK007 - Is there a requirement for the step list header or separator from the progress bar area, or is the visual relationship between progress bar and step list left undefined? [Completeness]
- [ ] CHK008 - Does the spec define whether deliverable tree rendering (existing behavior for completed steps) is preserved, removed, or modified in the new all-step layout? [Completeness]
- [ ] CHK009 - Is the tool activity line (shown below the running step) explicitly required to be preserved in the new layout? [Completeness]

## Clarity

- [ ] CHK010 - Is the term "step list area" defined precisely enough to identify the exact region of the TUI that changes? [Clarity]
- [ ] CHK011 - Are the color assignments for each state indicator specified with sufficient precision (lipgloss color codes or named colors), or only described as "muted", "red", etc.? [Clarity]
- [ ] CHK012 - Does FR-002 unambiguously specify the format when persona is present vs. absent, or could `<indicator> <step-name> ()` result from an empty persona? [Clarity]
- [ ] CHK013 - Is the distinction between "step name" and "step ID" clear throughout the spec? The display format uses stepID as the visible name — is this intentional and documented? [Clarity]
- [ ] CHK014 - Is "real time" (FR-009) defined in terms of the existing render cycle cadence (33ms tick), or is it left ambiguous? [Clarity]
- [ ] CHK015 - Does the elapsed time format specification (US-3: "(15s)" or "equivalent human-readable duration") provide enough precision to implement consistently, or should the exact format be locked down? [Clarity]

## Consistency

- [ ] CHK016 - Is the skipped indicator consistent between the spec (`—` em dash U+2014), the research doc (`—`), and the plan (`—`)? Verify no artifact uses a hyphen-minus `-` instead. [Consistency]
- [ ] CHK017 - Does the cancelled indicator (`⊛` U+229B) match across spec, research, plan, and data-model documents? [Consistency]
- [ ] CHK018 - Is the "muted color" for pending and skipped steps specified consistently across bubbletea and dashboard rendering paths? The plan says Color("244") for bubbletea — is an equivalent defined for dashboard? [Consistency]
- [ ] CHK019 - Are the user story acceptance scenarios numerically consistent with the functional requirements (e.g., every FR is testable by at least one acceptance scenario)? [Consistency]
- [ ] CHK020 - Is the display format `<indicator> <step-name> (<persona-name>) [(<timing>)]` consistent between the spec (FR-002), the plan (Change 4 algorithm), the research (Section 4 table), and the data-model (rendering entity)? [Consistency]
- [ ] CHK021 - Does the spec's edge case for "many steps exceeding terminal height" align with the plan's decision to not implement scrolling/truncation? Verify no FR contradicts this. [Consistency]

## Coverage

- [ ] CHK022 - Are there acceptance scenarios that test transitions (pending → running → completed) end-to-end, not just static states? [Coverage]
- [ ] CHK023 - Is the non-TTY degradation behavior (edge case 6) covered by a requirement or only mentioned as an edge case? Is there enough detail to implement? [Coverage]
- [ ] CHK024 - Is terminal resize handling (edge case 5) specified beyond "handle gracefully"? Are there testable criteria? [Coverage]
- [ ] CHK025 - Does the task list (tasks.md) cover testing of the dashboard rendering path with the same rigor as the bubbletea path (T025-T026 vs T016-T023)? [Coverage]
- [ ] CHK026 - Is there a task or requirement ensuring verbose mode is NOT affected (FR-011)? Is this validated by a specific test? [Coverage]
- [ ] CHK027 - Does the test plan cover the race condition edge case (two steps with spinners simultaneously, FR-012)? [Coverage]
- [ ] CHK028 - Are error/fallback paths defined for when `StepPersonas[stepID]` returns empty string at render time? [Coverage]
