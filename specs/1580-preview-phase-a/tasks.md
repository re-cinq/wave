# Work Items

## Phase 1: Setup

- [X] 1.1: Confirm five mockup files exist in `docs/scope/mockups/` and read each to inventory required fixture data shapes
- [X] 1.2: Add `registerPreview(r)` call to `NewFeatureRegistry` in `internal/webui/features.go`
- [X] 1.3: Add `internal/webui/preview_disabled.go` with `//go:build !webui_preview` and no-op `registerPreview(*FeatureRegistry)`

## Phase 2: Core Implementation

- [X] 2.1: Create `internal/webui/preview.go` (`//go:build webui_preview`) — embed.FS for templates+css, parse templates at init, register 5 GET routes via `r.addRoutes(...)` [P]
- [X] 2.2: Create `internal/webui/preview_fixtures.go` (`//go:build webui_preview`) — fixture structs + literal values for all 5 pages [P]
- [X] 2.3: Create `internal/webui/templates/preview/_banner.html` partial — persistent PREVIEW banner [P]
- [X] 2.4: Port `00-landing.html` → `templates/preview/index.html` with banner partial + fixture binding [P]
- [X] 2.5: Port `01-onboarding-session.html` → `templates/preview/onboard.html` [P]
- [X] 2.6: Port `02-work-board.html` → `templates/preview/work.html` [P]
- [X] 2.7: Port `03-work-item.html` → `templates/preview/work_item.html` [P]
- [X] 2.8: Port `04-evolution-proposal.html` → `templates/preview/proposal.html` [P]
- [X] 2.9: Copy `docs/scope/mockups/style.css` → `internal/webui/static/preview/style.css` and reference via `/preview/static/style.css` (tag-gated handler)
- [X] 2.10: Implement 5 handler funcs (`handlePreviewIndex`, `handlePreviewOnboard`, `handlePreviewWork`, `handlePreviewWorkItem`, `handlePreviewProposal`) in `preview.go`

## Phase 3: Testing

- [X] 3.1: Create `internal/webui/preview_test.go` (`//go:build webui_preview`) — table-driven smoke tests asserting 200 + banner string for each of 5 routes
- [X] 3.2: Run `go build ./...` (no tag) — succeeds with zero preview symbols
- [X] 3.3: Run `go build -tags=webui_preview ./...` — succeeds
- [X] 3.4: Run `go test ./internal/webui/...` (no tag) — passes, preview tests not compiled
- [X] 3.5: Run `go test -tags=webui_preview ./internal/webui/...` — preview smoke tests pass

## Phase 4: CI + Polish

- [X] 4.1: Update `.github/workflows/lint.yml` — add matrix `tags: ["", "webui_preview"]` for golangci-lint
- [X] 4.2: Add a build job (extended `lint.yml`) that runs `go build` in both tag modes
- [X] 4.3: Add a test job that runs `go test -tags=webui_preview ./internal/webui/...`
- [ ] 4.4: Manual visual smoke: `go run -tags=webui_preview ./cmd/wave webui` → load all 5 routes in browser, verify banner + layout (deferred — requires interactive browser; smoke tests cover 200 + banner)
- [X] 4.5: Run `go vet ./...` in both tag modes; clean (golangci-lint not installed in workspace; CI matrix runs it)
- [ ] 4.6: Open PR titled `feat(webui): /preview/* phase A (build tag, fixtures only)` referencing #1580 and Epic #1565 (handled by next pipeline step)
