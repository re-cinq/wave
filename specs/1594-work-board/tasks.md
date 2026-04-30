# Work Items

## Phase 1: Setup

- [X] Item 1.1: Add `worksource worksource.Service` field to
  `serverRuntime` in `internal/webui/server.go` and populate it in
  `NewServer` via `worksource.NewService(rwStore)`.
- [X] Item 1.2: Confirm no existing route claims `/work` or
  `/work/...` (grep `routes.go` and feature registry files).

## Phase 2: Core Implementation

- [X] Item 2.1: Implement `handleWorkBoard` in
  `internal/webui/handlers_work.go` with `BindingFilter{}` query +
  view-model mapping. [P]
- [X] Item 2.2: Implement `handleWorkItemDetail` in the same file:
  parse path values, build `WorkItemRef`, call `MatchBindings`,
  best-effort forge fetch, run-history filter by matched pipeline
  names. [P]
- [X] Item 2.3: Create `templates/work/board.html` — Tailwind CDN,
  in-template nav, bindings list, empty state. SVG icons only.
- [X] Item 2.4: Create `templates/work/detail.html` — Tailwind CDN,
  work item header, matched bindings table, recent runs list,
  disabled "Run on this issue" button.
- [X] Item 2.5: Register both routes in `routes.go` next to other
  dashboard routes.
- [X] Item 2.6: Update `embed.go` — introduce
  `standalonePageTemplates` slice (parsed without layout cloning)
  containing both work templates; thread parsed entries into the
  returned template map so handlers resolve them by key.

## Phase 3: Testing

- [X] Item 3.1: Add `handlers_work_test.go` with round-trip tests
  for both handlers (Empty / WithBindings / NoMatch / OneMatch /
  BadNumber).
- [X] Item 3.2: Extend `testTemplates` in `handlers_test.go` with
  inline standalone-page stubs so the work tests don't require the
  full embedded FS.
- [X] Item 3.3: `go test ./internal/webui/... -race` — passes
  (72.7s, no failures).

## Phase 4: Polish

- [X] Item 4.1: `go vet ./...` clean; full `go test ./...` passes
  (golangci-lint runs in CI — not installed locally in this
  sandbox).
- [ ] Item 4.2: Build wave binary outside sandbox; manually visit
  `/work` and `/work/github/re-cinq/wave/1594` to verify rendering
  and confirm "no emojis" constraint. (deferred — needs host)
- [ ] Item 4.3: Open PR referencing issue #1594 and the merged
  dependency PRs. (next step in the pipeline)
