# Implementation Plan: Run Details Stats Card

## 1. Objective

Replace the minimal run-summary-bar on the run details page with a richer stats card grid that surfaces all available run metadata (full input, linked URL, finish time, branch) at a glance — for both completed and in-progress runs.

## 2. Approach

The existing `RunSummary` type already carries most fields (`RunID`, `PipelineName`, `BranchName`, `Duration`, `TotalTokens`, `StartedAt`, `CompletedAt`, `InputPreview`). The main gaps are:

1. **Full input text** — `runToSummary` truncates to 80 chars. Add an `Input` field carrying the untruncated value.
2. **Linked URL** — Parse GitHub issue/PR URLs from the input string and expose as `LinkedURL` (string). A simple regex on `github.com/<owner>/<repo>/(issues|pull)/<number>` suffices.
3. **Formatted timestamps** — Add `FormattedStartedAt` and `FormattedCompletedAt` for human-readable display in the template (avoids complex template logic).
4. **Frontend card grid** — Replace `run-summary-bar` in `run_detail.html` with a CSS grid of stat cards. Each card has a label and value. Long input uses a `<details>` element for expand/collapse.
5. **Copy-to-clipboard** — Small JS snippet for the Run ID card.

No DB schema changes required. All data is already in `RunRecord`.

## 3. File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/webui/types.go` | modify | Add `Input`, `LinkedURL`, `FormattedStartedAt`, `FormattedCompletedAt` fields to `RunSummary` |
| `internal/webui/handlers_runs.go` | modify | Update `runToSummary` to populate new fields; add `parseLinkedURL` helper |
| `internal/webui/templates/run_detail.html` | modify | Replace `run-summary-bar` div with stats card grid |
| `internal/webui/static/style.css` | modify | Add `.stats-card-grid`, `.stats-card`, `.stats-card-label`, `.stats-card-value` styles |
| `internal/webui/handlers_runs_test.go` | modify | Add tests for `parseLinkedURL` and new `runToSummary` field population |

## 4. Architecture Decisions

- **`<details>` for expandable input**: Native HTML element, no JS required, accessible, works in all browsers. Collapses inputs longer than ~120 chars.
- **Regex for URL parsing**: Simple `regexp.MustCompile` at package level. Only matches `https://github.com/.../(issues|pull)/\d+` — intentionally narrow to avoid false positives.
- **No new template functions**: Formatted timestamps computed in Go and passed as strings. Keeps template logic minimal.
- **CSS Grid**: Two-column grid on desktop, single column on mobile. Consistent with existing card/grid patterns in the webui.
- **Run ID copyable**: Inline JS `navigator.clipboard.writeText()` with a small copy button next to the code element.

## 5. Risks

| Risk | Mitigation |
|------|------------|
| Long input text overflows card | `<details>` element truncates by default; full text in expandable section |
| URL regex misses edge cases (GitLab, Gitea) | Issue scope is GitHub only; future forge support can extend the regex |
| CSS grid breaks existing layout on mobile | Use `@media` queries with tested breakpoints; existing responsive patterns as guide |
| Template changes break existing tests | `TestHandleRunDetailPage_ValidRun` already checks for 200 + contains runID; update if needed |

## 6. Testing Strategy

- **Unit tests for `parseLinkedURL`**: Table-driven tests covering GitHub issue URLs, PR URLs, non-GitHub URLs, empty strings, and inputs with multiple URLs (first match wins).
- **Unit tests for `runToSummary` field population**: Verify `Input`, `LinkedURL`, `FormattedStartedAt`, `FormattedCompletedAt` are correctly populated from a `RunRecord`.
- **Existing integration tests**: `TestHandleRunDetailPage_ValidRun` and `TestHandleRunDetailPage_WithPipelineAndEvents` exercise the full HTML render path — ensure they still pass.
- **No new E2E tests**: The template changes are HTML/CSS only and are verified by the existing handler tests returning 200.
