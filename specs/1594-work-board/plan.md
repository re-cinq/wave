# Implementation Plan — #1594 /work board + detail

## 1. Objective

Add functional `/work` board and `/work/{forge}/{owner}/{repo}/{number}`
detail pages in `internal/webui/`, backed by `worksource.Service` from
PR #1591. The visual design follows `templates/preview/work.html` and
`templates/preview/work_item.html`, but uses Tailwind CDN per issue spec
and renders real binding data instead of fixtures.

## 2. Approach

1. Wire a `worksource.Service` instance into `serverRuntime` so handlers
   can query it. The service is constructed via
   `worksource.NewService(s.runtime.rwStore)` (the `state.StateStore`
   embeds `state.WorksourceStore`).
2. Add `handlers_work.go` with two handler methods on `*Server`:
   - `handleWorkBoard(w, r)` — `GET /work`. Calls
     `s.runtime.worksource.ListBindings(ctx, BindingFilter{})` and renders
     `templates/work/board.html`. The board lists every binding with
     active/inactive state, forge, repo pattern, pipeline name, trigger,
     and label filter. A second section lists "recent matches" — a
     placeholder backed by recent runs whose pipeline matches one of the
     binding pipeline names (best-effort until #2.4 wires real
     work_item_ref → run links).
   - `handleWorkItemDetail(w, r)` — `GET /work/{forge}/{owner}/{repo}/{number}`.
     Parses path values, builds a `worksource.WorkItemRef`, calls
     `MatchBindings`, fetches the live work item from `forge.Client` (if
     configured) for title/labels/state, then renders
     `templates/work/detail.html` with: work item header, matched
     bindings table, recent runs whose pipeline name matches a matched
     binding, and a "Run on this issue" button (disabled placeholder
     until #2.4).
3. Register both routes in `routes.go`. Use `mux.HandleFunc("GET /work", ...)`
   and `mux.HandleFunc("GET /work/{forge}/{owner}/{repo}/{number}", ...)`.
4. Add the two new templates under `templates/work/` and register them
   in `pageTemplates` in `embed.go` so the embedded FS picks them up.
   Tailwind CDN: include `<script src="https://cdn.tailwindcss.com"></script>`
   in the templates. The pages do not extend `layout.html` — they render
   a self-contained Tailwind page so the CDN classes don't collide with
   the existing site stylesheet. A minimal nav bar links to /work,
   /runs, /pipelines for parity with the rest of the dashboard.
5. Add tests in `handlers_work_test.go` following the
   `httptest.NewRecorder` + `strings.Contains(body, ...)` pattern used
   by `handlers_runs_test.go`. Cover:
   - `/work` empty-state, populated-state.
   - `/work/{forge}/{owner}/{repo}/{number}` with no matching bindings,
     with one matching binding, malformed path (bad number).

## 3. File mapping

Create:

- `internal/webui/handlers_work.go` — two handlers + small `WorkBindingRow`
  / `WorkItemDetailData` view-models that flatten `worksource.BindingRecord`
  for template consumption (avoid leaking domain types into HTML where
  small label maps add value, e.g. trigger labels).
- `internal/webui/handlers_work_test.go` — round-trip tests.
- `internal/webui/templates/work/board.html` — board view (Tailwind).
- `internal/webui/templates/work/detail.html` — detail view (Tailwind).
- `specs/1594-work-board/spec.md`, `plan.md`, `tasks.md` — planning docs.

Modify:

- `internal/webui/server.go` — extend `serverRuntime` with
  `worksource worksource.Service`; populate it in `NewServer` after the
  rwStore is opened.
- `internal/webui/routes.go` — register `GET /work` and
  `GET /work/{forge}/{owner}/{repo}/{number}` (placed alongside other
  dashboard routes, before the feature-registry route block).
- `internal/webui/embed.go` — append `"templates/work/board.html"` and
  `"templates/work/detail.html"` to `pageTemplates`.
- `internal/webui/handlers_test.go` — extend `testServer` template stub
  registration if needed so tests can render the work templates without
  a full embed.

Delete: none.

## 4. Architecture decisions

- **Service injection via `serverRuntime`**: matches the existing
  injection pattern (store, scheduler, forgeClient). Keeps handler
  signatures unchanged.
- **Standalone Tailwind pages, no layout.html extension**: the issue
  explicitly mandates Tailwind CDN. Mixing Tailwind utility classes
  with the existing `style.css` (which defines its own `.btn`, `.list`,
  `.badge` rules) would cause specificity collisions. Self-contained
  pages with a minimal in-template nav are the cleanest path. When the
  /work page becomes default landing (#1579), the rest of the
  dashboard can migrate gradually if desired.
- **Run history on detail page**: until #2.4 wires real
  `work_item_ref` → run linkage, "run history" is best-effort: filter
  recent runs by pipeline name from the matched bindings (limit 20,
  newest first, no pagination). This is documented in template copy
  ("recent runs of pipelines that match this work item").
- **Forge data fetch on detail page is best-effort**: if `forgeClient`
  is nil or the fetch fails, render the page with the URL-derived
  fields (forge/repo/number) and an "info unavailable" notice rather
  than 500. The `MatchBindings` call only needs the path-supplied
  fields to function.
- **Path parameters via `http.ServeMux` 1.22+ patterns**: the codebase
  already uses `mux.HandleFunc("GET /pattern/{var}", ...)` (see
  routes.go), so `r.PathValue("forge")` etc. is the idiomatic access.
- **No emojis**: nav and buttons use SVG icons identical to those in
  `templates/preview/work.html`, or plain text.
- **No dispatch wiring**: the "Run on this issue" button is a disabled
  `<button>` with a tooltip pointing at #2.4. Avoid creating a stub
  POST route that does nothing.

## 5. Risks & mitigations

- **Tailwind CDN means external network dependency at render time** —
  mitigated: this is dev/local UI; no offline guarantee in scope. If
  problematic later, switch to bundled Tailwind output.
- **`worksource.Service` constructor expects `state.WorksourceStore`** —
  the existing `state.StateStore` embeds it, so passing `s.runtime.rwStore`
  works. Verified via `internal/state/store.go:77`.
- **Template parsing in `parseTemplates`** — the new pages do NOT
  extend `templates/layout.html`, so they should not be cloned from
  the layout-bearing base. Add a separate parsing path: parse each
  work template into its own root template so block names don't
  collide with the layout-extending pages. Either:
  (a) Skip cloning for these two templates and parse them via
  `template.New("work-board").Funcs(funcMap).Parse(...)`, or
  (b) Keep them in `pageTemplates` but render with `Execute` (not
  `ExecuteTemplate("layout")`).
  Decision: (a) — clearer intent. Add a `standalonePageTemplates`
  slice in `embed.go`.
- **Test isolation** — `handlers_test.go` `testTemplates` builds a
  minimal stub set; tests for /work need their own minimal stubs that
  match the standalone-template approach. Add `testWorkTemplates()`
  helper that registers two trivial templates with placeholder content
  exposing the data fields the assertions check.
- **Path collision with feature registry routes** — verify no
  feature already claims `/work` or `/work/...`. Quick grep before
  registration.
- **Empty state** — board with zero bindings should render a clear
  empty state pointing at the bindings CRUD UI (or note that #2.x
  will add it). No hidden 404.

## 6. Testing strategy

Unit/round-trip tests in `handlers_work_test.go`:

1. `TestHandleWorkBoard_Empty` — no bindings created → 200, body
   contains the empty-state message and zero list rows.
2. `TestHandleWorkBoard_WithBindings` — create 2 bindings via the
   service → 200, body contains both pipeline names and trigger
   labels.
3. `TestHandleWorkItemDetail_NoMatch` — call /work/github/foo/bar/1
   when no binding matches → 200, body contains "no bindings match".
4. `TestHandleWorkItemDetail_OneMatch` — create a binding whose
   `RepoPattern` matches "foo/bar" → 200, body contains the binding
   pipeline name and "Run on this issue" button (disabled state
   present in markup).
5. `TestHandleWorkItemDetail_MalformedPath` — already handled by
   `http.ServeMux` 404; document that no test is needed for the path
   parser since the framework rejects mismatched paths.

Run with `go test ./internal/webui/... -race`.

CI/lint:

- `golangci-lint run ./internal/webui/...` (existing project lint).
- `go vet ./...`.

Acceptance verification (manual):

- Build: `go build -o ~/.local/bin/wave ./cmd/wave` outside the
  sandbox (per repo memory constraints).
- Run dashboard: `wave server` and visit `/work` and
  `/work/github/re-cinq/wave/1594` in a browser; confirm bindings
  render and detail page loads even when no forge client is
  configured.
