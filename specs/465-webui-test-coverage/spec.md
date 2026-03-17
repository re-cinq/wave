# test(webui): increase test coverage beyond 53.3% targeting handlers and SSE edge cases

**Issue**: [#465](https://github.com/re-cinq/wave/issues/465)
**Parent**: #455
**Labels**: enhancement, frontend
**Author**: nextlevelshit

## Summary

Increase webui test coverage beyond the current 53.3%, targeting handler edge cases, SSE streaming scenarios, and template rendering. This hardens the webui against regressions as UX changes land from the other sub-issues in this epic.

## Acceptance Criteria

- [ ] Test coverage for `internal/webui/` reaches at least 70%
- [ ] Handler tests cover: success paths, error responses, missing/invalid parameters, auth failures
- [ ] SSE handler tests cover: initial connection, event streaming, `Last-Event-ID` reconnection, client disconnect
- [ ] Pagination edge cases tested: empty results, single page, boundary cursors
- [ ] Template rendering tests verify key pages render without errors given valid data
- [ ] Control handlers tested: pipeline start, cancel, retry, resume with valid and invalid inputs
- [ ] No test uses `t.Skip()` without a linked issue

## Dependencies

- Should be done after or in parallel with other UX sub-issues to cover new code paths

## Scope Notes

- **In scope**: Go test coverage for webui handlers, SSE, pagination, templates
- **Out of scope**: Browser-level/E2E testing (no JS testing framework currently exists)
- **Out of scope**: Backend/non-webui test coverage improvements
