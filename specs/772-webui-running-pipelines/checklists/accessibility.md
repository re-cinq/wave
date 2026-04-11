# Accessibility Requirements Quality Checklist

**Feature**: `772-webui-running-pipelines`  
**Checklist Type**: Accessibility Requirements Completeness  
**Generated**: 2026-04-11

Tests the quality of accessibility requirements for the running-pipelines section toggle
and run card keyboard navigation.

---

## Completeness

- [ ] CHK-A001 — Does FR-010 specify that `aria-controls` must reference the exact element ID of the collapsible body (`rp-section-body`), or is the target element ID left unspecified? [Completeness]
- [ ] CHK-A002 — Does the spec define keyboard-operability for the run card links themselves (Tab focus order through cards, Enter to navigate)? FR-010 addresses the toggle control only. [Completeness]
- [ ] CHK-A003 — Does the spec define accessible label text for the chevron icon (decorative vs. labelled) and the count badge (whether a screen reader should announce the number)? [Completeness]
- [ ] CHK-A004 — Is there a requirement for focus management after the toggle action (where keyboard focus should remain after collapse/expand)? [Completeness]

---

## Clarity

- [ ] CHK-A005 — Does SC-005 ("passes automated accessibility checks") specify which tool or standard (WCAG 2.1 AA, axe-core, etc.) defines "passing"? Without this, the success criterion is not measurable by an automated test. [Clarity]
