# Requirements Quality Review Checklist

**Feature**: `772-webui-running-pipelines`  
**Checklist Type**: Overall Requirements Quality  
**Generated**: 2026-04-11

Tests the quality of requirements, not the implementation.
Each item is a unit-test assertion about requirement completeness, clarity, consistency, or coverage.

---

## Completeness

- [ ] CHK001 — Are all acceptance scenarios for User Story 3 (empty state) aligned with FR-005? Does FR-005 specify both the placeholder message AND the CTA element, or only one? [Completeness]
- [ ] CHK002 — Is the CTA destination URL (`/pipelines`) formally encoded in a functional requirement (FR-005), or does it exist only in implementation artifacts (plan.md, template task T010)? [Completeness]
- [ ] CHK003 — Are error states defined for a failure of the secondary `ListRuns(status=running)` query (e.g., store unavailability)? [Completeness]
- [ ] CHK004 — Are the exact data fields displayed on each running run card specified (pipeline name, status badge, progress %, duration) as a requirement, or only inferred from "same visual pattern as main runs list"? [Completeness]
- [ ] CHK005 — Does the spec define whether sub-runs (child runs with `ParentRunID != ""`) are excluded from the running-pipelines section as a stated requirement, or is this an unspecified implementation detail? [Completeness]

---

## Clarity

- [ ] CHK006 — Is "expanded by default on every page load" in FR-002 unambiguous about whether this applies to all page loads or only the first? (US2-AC3 says collapse is NOT persisted, which implies every load — are spec + story consistent in wording?) [Clarity]
- [ ] CHK007 — Is "same run card visual pattern as the main runs list" in FR-004 precise enough to be testable without ambiguity? Does it define which specific visual attributes must match? [Clarity]
- [ ] CHK008 — Does FR-009 specify the exact label text ("Running") as normative, or is "or equivalent" an intentional implementation choice left to the developer? [Clarity]
- [ ] CHK009 — Does FR-010 explicitly list both `aria-expanded` AND `aria-controls` attribute names, and their expected values at initial load? Or are "appropriate ARIA attributes" underspecified? [Clarity]
- [ ] CHK010 — Does FR-008 explicitly define the filter's behavior when no filter is active (i.e., all running runs are shown, unfiltered)? Or is the default (no filter = show all) only implied? [Clarity]

---

## Consistency

- [ ] CHK011 — Does FR-007 (completed/failed card navigation) add a distinct requirement beyond FR-006 (all run cards are navigable links), or is it redundant? If redundant, does the duplication create ambiguity about whether only running-status cards are navigable? [Consistency]
- [ ] CHK012 — Is the "expanded by default" invariant in FR-002 consistent with SC-002 (100% of page loads)? Do both use identical scope language, or is one stricter? [Consistency]
- [ ] CHK013 — Does FR-008's filter requirement for the running section match the edge-case statement that "the running-pipelines section should reflect the filter"? Is "reflect" (edge case) and "respect" (FR-008) used consistently? [Consistency]
- [ ] CHK014 — Is CL-001 (page-reload-only updates) consistently reflected in all relevant acceptance scenarios? Does US3-AC3 avoid implying real-time behavior, and do all story scenarios align with this constraint? [Consistency]
- [ ] CHK015 — Are the run card fields listed in the `RunSummary` entity (`RunID`, `PipelineName`, `Status`, `Progress`, `StepsCompleted`, `StepsTotal`, `Duration`, `FormattedStartedAt`) consistent with what "same visual pattern as main runs list" (FR-004) would render? Is there a formal mapping? [Consistency]

---

## Coverage

- [ ] CHK016 — Are all 4 user stories (US1–US4) traceable to at least one functional requirement (FR-001–FR-010) AND at least one success criterion (SC-001–SC-006)? [Coverage]
- [ ] CHK017 — Is the mobile/responsive behavior formally specified as a functional requirement, or does it only appear in the edge-cases section? If only edge-case, is it explicitly accepted as out-of-scope for v1? [Coverage]
- [ ] CHK018 — Is the filter interaction from FR-008 covered by a dedicated success criterion? SC-006 addresses filter behavior — does SC-006 define what happens when a filter is applied but zero running runs match (empty-state vs. empty section)? [Coverage]
- [ ] CHK019 — Does the spec formally require or define the behavior for running runs appearing in BOTH the running-pipelines section AND the main runs list simultaneously? The edge-case mentions it but no FR mandates the duplication policy. [Coverage]
- [ ] CHK020 — Is there a requirement or success criterion covering the section's behavior when the running count changes between the time the page is rendered and when the user reads it (stale state, per CL-001)? [Coverage]
