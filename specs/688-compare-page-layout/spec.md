# fix(webui): /compare page renders raw error without layout template

**Issue**: [re-cinq/wave#688](https://github.com/re-cinq/wave/issues/688)
**Parent**: #687 (item 1)
**Author**: nextlevelshit
**State**: OPEN
**Labels**: none

## Problem

Visiting `/compare` without query params renders `both left and right run IDs are required` as **unstyled plain text** — no navbar, no layout, no HTML structure at all. This is because `handleComparePage` calls `http.Error()` which bypasses the template system.

## Expected

The compare page should render inside the standard layout template with a form/UI to select two runs for comparison, or at minimum show a styled error within the layout.

## Files to investigate

- `internal/webui/handlers_compare.go` — the error path bypasses templates
- `internal/webui/templates/compare.html` — needs an empty-state / run-selector UI

## Acceptance Criteria

- [ ] `/compare` without params renders inside the layout with navbar
- [ ] User can select two runs to compare (dropdown or input)
- [ ] Error states render styled, not as raw text
