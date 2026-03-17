# feat(webui): improve run detail view with step progress and duration display

> Issue: [#461](https://github.com/re-cinq/wave/issues/461) | Parent: #455 | Labels: enhancement, ux, frontend

## Summary

Improve the run detail and step progress visualization to show clearer step-by-step progress, duration display, and log readability. The target UX is GitHub Actions' job detail view with collapsible step sections, duration per step, and clear pass/fail indicators.

## Acceptance Criteria

- [ ] Step progress visualization shows a clear pipeline DAG or sequential step list with status per step
- [ ] Each step displays: name, status badge, duration (elapsed for running, total for completed), start time
- [ ] Running steps show an animated progress indicator
- [ ] Failed steps are visually prominent with expandable error details
- [ ] Step sections are collapsible/expandable (like GitHub Actions job groups)
- [ ] Overall run header shows: pipeline name, total duration, trigger info, final status
- [ ] Duration display uses human-friendly formatting (e.g., "2m 34s" not "154000ms")
- [ ] Log output within steps is readable with proper monospace font, line numbers, and syntax highlighting for key patterns (errors, warnings)

## Dependencies

- #459 (UX audit) — audit findings inform specific improvements needed

## Scope Notes

- **In scope**: Run detail page and step progress visualization improvements
- **Out of scope**: Log streaming improvements (covered by #462 — log streaming UX)
- **Out of scope**: New API endpoints beyond what exists
