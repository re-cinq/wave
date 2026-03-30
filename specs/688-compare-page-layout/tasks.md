# Tasks

## Phase 1: Handler Changes
- [X] Task 1.1: Modify `handleComparePage` to render template for missing-params case — load recent runs from store, pass `ShowSelector: true` and `Runs` list to template instead of calling `http.Error()`
- [X] Task 1.2: Modify `handleComparePage` to render template for not-found errors — pass `Error` string to template instead of calling `http.Error()`

## Phase 2: Template Changes
- [X] Task 2.1: Add run-selector form to `compare.html` — two `<select>` dropdowns populated from `.Runs`, a "Compare" button, rendered when `.ShowSelector` is true [P]
- [X] Task 2.2: Add styled error alert block to `compare.html` — rendered when `.Error` is non-empty, styled consistently with the rest of the UI [P]
- [X] Task 2.3: Wrap existing comparison content in `{{if not .ShowSelector}}` guard to skip it when in selector mode

## Phase 3: Testing
- [X] Task 3.1: Update `TestHandleComparePage_MissingParams` — expect HTTP 200, verify response contains `<select>` elements and navbar markup
- [X] Task 3.2: Add `TestHandleComparePage_RunNotFound` — request with invalid IDs, expect HTTP 200, verify response contains error message and navbar
- [X] Task 3.3: Verify `TestHandleComparePage_Success` still passes unchanged
- [X] Task 3.4: Run `go test ./internal/webui/...` to confirm all tests pass
