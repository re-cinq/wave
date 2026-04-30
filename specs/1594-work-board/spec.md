# Phase 2.3: /work board + detail templates (replaces /runs as default landing)

Source issue: https://github.com/re-cinq/wave/issues/1594
Repository: re-cinq/wave
Author: nextlevelshit
State: OPEN
Labels: enhancement, ready-for-impl, frontend

Part of Epic #1565 Phase 2 (work-source dispatch).

## Goal

Build the webui `/work` board: a unified view of all bindings + active work
items, with detail pages for each work item. Becomes default landing
(`/` redirects to `/work` once #1579 1.4 lands).

## Acceptance criteria

- [ ] `internal/webui/handlers_work.go` — handlers for:
  - `GET /work` — board view, lists bindings with status (active/inactive)
    and recent matches
  - `GET /work/{forge}/{owner}/{repo}/{number}` — detail view of a single
    work item: matched bindings, run history, "Run on this issue" button
    (#2.4 wires the action)
- [ ] `internal/webui/templates/work/{board,detail}.html` — uses Tailwind CDN
- [ ] Calls `worksource.Service.ListBindings` + `MatchBindings` (from #1591)
- [ ] No emojis (per Wave constraint)
- [ ] Test coverage: at least one round-trip test for each handler
  (htmltest pattern)

## Out of scope

- Dispatch wiring (#2.4)
- Entry-page redirect to /work (1.4 #1579)

## Dependencies

- #1591 WorkSourceService — MERGED (commit b8f4e01a)
- #1590 work_item_ref schema — MERGED (commit 7b6aadc9)
- 1.5a /preview/* phase A (PR #1585 MERGED) — visual reference
  (`internal/webui/templates/preview/work.html`,
  `internal/webui/templates/preview/work_item.html`)

## Notes

Design references (preview templates) are the visual contract for layout
and structure; this issue wires real data through `worksource.Service` so
the screens become functional rather than fixture-driven.
