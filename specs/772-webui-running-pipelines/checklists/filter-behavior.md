# Filter Interaction Requirements Quality Checklist

**Feature**: `772-webui-running-pipelines`  
**Checklist Type**: Filter/Search Interaction Completeness  
**Generated**: 2026-04-11

Tests the quality of requirements for the pipeline-name filter's effect on the
running-pipelines section (FR-008, SC-006).

---

## Completeness

- [ ] CHK-F001 — Does FR-008 specify what the running-pipelines section must display when the active filter matches zero running runs — does it show the standard empty-state CTA (FR-005) or simply render an empty card list? [Completeness]
- [ ] CHK-F002 — Does FR-009 (section header count badge) specify whether the displayed count reflects the filtered count or the total running count when a pipeline-name filter is active? [Completeness]

---

## Clarity

- [ ] CHK-F003 — Does SC-006 define "reduces the running-pipelines section to show only matching active runs" precisely enough to be testable? Does "reduces" mean the section can reach zero cards (triggering empty-state) or only that non-matching cards are hidden? [Clarity]
- [ ] CHK-F004 — Does the spec clarify whether the pipeline-name filter is applied server-side (new request) or client-side (DOM filter)? FR-008 says the section "must respect" the filter but does not define the mechanism, which affects how the filter requirement is tested. [Clarity]
