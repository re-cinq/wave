# Implementation Plan

## Objective

Replace raw `http.Error()` responses in `handleComparePage` with template-rendered pages that use the standard layout (navbar, styling). When no run IDs are provided, show a run-selector form. When runs are not found, show a styled error within the layout.

## Approach

1. **Modify the handler** (`handlers_compare.go`): Instead of `http.Error()` for the missing-params case, render the compare template with a new "selector mode" — pass empty `Left`/`Right` and a `Runs` list so the template can render a run-selector form. For not-found errors, render the template with an `Error` field.

2. **Extend the compare template** (`compare.html`): Add conditional blocks — when `Left`/`Right` are empty (no RunID), render a run-selector form with two dropdowns and a "Compare" button. When `Error` is set, show a styled alert. The existing comparison UI renders only when both runs are present.

3. **Update tests**: The existing `TestHandleComparePage_MissingParams` expects HTTP 400 — update it to expect 200 with the selector form. Add a test for the not-found error rendering within layout.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/webui/handlers_compare.go` | modify | Replace `http.Error()` calls with template rendering; add `Runs []RunSummary` to template data; load recent runs for selector |
| `internal/webui/templates/compare.html` | modify | Add run-selector form for empty state; add styled error alert block; wrap comparison content in conditional |
| `internal/webui/handlers_compare_test.go` | modify | Update missing-params test to expect 200+HTML; add not-found-within-layout test |

## Architecture Decisions

1. **No new template file** — the compare template already exists. Add conditional blocks (`{{if .Error}}`, `{{if not .Left.RunID}}`) rather than creating a separate selector page. This keeps routing simple (single `/compare` endpoint).

2. **Reuse `listSamePipelineRuns` pattern** — the handler already fetches runs for the swap dropdown. For the selector mode, use `store.ListRuns` with a reasonable limit (50) to populate two dropdowns.

3. **Template data struct extension** — add `Error string`, `Runs []RunSummary`, and `ShowSelector bool` fields to the anonymous struct passed to the template. `ShowSelector` is true when both `left` and `right` query params are empty.

4. **Error states stay HTTP 200** — when rendering errors within the layout template, use HTTP 200 (the page rendered successfully, it just contains an error message). This matches standard web UI patterns where the error is part of the page content.

## Risks

| Risk | Mitigation |
|------|-----------|
| Template nil pointer if `Left`/`Right` are zero-value `RunSummary` | Use `{{if .ShowSelector}}` guard to skip comparison blocks entirely |
| Large run list overwhelming the dropdowns | Cap at 50 most recent runs; same limit used elsewhere |
| Existing tests break from status code change | Explicitly update test expectations |

## Testing Strategy

1. **Unit test: missing params returns 200 with selector form** — verify response contains "select" elements and layout navbar
2. **Unit test: invalid run IDs render styled error within layout** — verify response contains error message and navbar, not raw text
3. **Unit test: successful comparison still works** — existing `TestHandleComparePage_Success` should pass unchanged
4. **Verify API endpoint unchanged** — `handleAPICompare` still returns JSON errors (no change needed)
