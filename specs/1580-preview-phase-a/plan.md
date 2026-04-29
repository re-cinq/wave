# Implementation Plan: /preview/* phase A

## 1. Objective

Add a build-tag-gated `/preview/*` route group to `internal/webui` that renders five fixture-backed HTML pages ported from `docs/scope/mockups/`. Default builds must remain unaffected (zero footprint).

## 2. Approach

Mirror the existing `features_<name>.go` / `features_<name>_disabled.go` pattern already used for analytics/metrics/webhooks/retros. Add `preview.go` (build tag `webui_preview`) and `preview_disabled.go` (negation) that hook a route registrar into `FeatureRegistry.routeFns` via `addRoutes`. Templates live under `internal/webui/templates/preview/` and are also gated by the build tag — they are embedded only when the tag is set, so default `go build` ships zero preview bytes.

Templates are ported as **standalone** HTML pages (not the layout/partial system used by the production webui) because the mockups are self-contained design artifacts. Each preview page injects a fixed `PREVIEW` banner via a shared `_banner.html` partial included from each page.

Fixtures are pure Go literals in `preview_fixtures.go` — no DB, no services, no state.

## 3. File Mapping

### New files

| Path | Tag | Purpose |
|------|-----|---------|
| `internal/webui/preview.go` | `//go:build webui_preview` | Embed preview templates, register `/preview/*` routes, render handlers |
| `internal/webui/preview_disabled.go` | `//go:build !webui_preview` | No-op stub `registerPreview(r *FeatureRegistry)` |
| `internal/webui/preview_fixtures.go` | `//go:build webui_preview` | Fixture structs + values for each page |
| `internal/webui/preview_test.go` | `//go:build webui_preview` | Smoke tests: 200 OK + banner check on each route |
| `internal/webui/templates/preview/_banner.html` | (embed-gated) | Shared persistent PREVIEW banner partial |
| `internal/webui/templates/preview/index.html` | (embed-gated) | Port of `00-landing.html` |
| `internal/webui/templates/preview/onboard.html` | (embed-gated) | Port of `01-onboarding-session.html` |
| `internal/webui/templates/preview/work.html` | (embed-gated) | Port of `02-work-board.html` |
| `internal/webui/templates/preview/work_item.html` | (embed-gated) | Port of `03-work-item.html` |
| `internal/webui/templates/preview/proposal.html` | (embed-gated) | Port of `04-evolution-proposal.html` |
| `internal/webui/static/preview/style.css` | (always embedded, tiny) | Copied from `docs/scope/mockups/style.css` for self-contained rendering |
| `specs/1580-preview-phase-a/{spec,plan,tasks}.md` | — | Planning artifacts |

> **Style.css note**: copying the css into `internal/webui/static/preview/` keeps the css under the tag-gated handler (handler only registered with tag); since `staticFS` is always embedded, the css bytes ship in default builds (~few KB). If we need true zero footprint, gate the css behind a separate `//go:embed` block in `preview.go` and serve via `http.FileServer` from a tag-only `embed.FS`. **Decision: gate the css too** (see Architecture Decisions §4.2).

### Modified files

| Path | Change |
|------|--------|
| `internal/webui/features.go` | Add `registerPreview(r)` call inside `NewFeatureRegistry()` |
| `.github/workflows/lint.yml` | Add matrix dimension: `tags: ["", "webui_preview"]` so golangci-lint runs in both modes |
| `.github/workflows/lint.yml` (or new `build.yml`) | Add a build job that runs `go build -tags=""` and `go build -tags=webui_preview` |

> **CI choice**: extending `lint.yml` keeps a single workflow; if matrix complicates the existing single-job structure too much, add a new `build.yml` workflow with the matrix. Lean toward extending the existing job.

### Deleted files

None.

## 4. Architecture Decisions

### 4.1 Reuse `FeatureRegistry`, not a parallel mux

The existing `FeatureRegistry` pattern (`features.go:30-37`) already supports tag-gated route registration via `routeFns`. Adding `registerPreview` slots in cleanly without touching the main router. Confirmed: `routes.go:92` iterates `s.assets.features.routeFns` after all production routes — preview routes register last, ensuring no path conflicts (none of `/preview/*` collides with existing routes).

### 4.2 Embed templates and css under build tag

Default `go build` must produce zero preview footprint. To achieve this:
- Both `//go:embed templates/preview/*.html` and `//go:embed static/preview/*.css` directives live inside `preview.go` (which is itself tag-gated). Without the tag, the directives are not compiled and the bytes are not embedded.
- Render handlers parse templates from a tag-local `embed.FS`, independent of the existing `templatesFS` / `staticFS` in `embed.go`.

### 4.3 Standalone templates (no layout extension)

Mockups are self-contained pages with inline `<style>` blocks. Porting them as **standalone** templates (each begins with `<!doctype html>`) preserves design fidelity and avoids cross-cutting refactors of the production layout. The shared `PREVIEW` banner is a small `_banner.html` partial rendered via `{{ template "_banner" . }}` from each page.

### 4.4 Routing pattern

```go
mux.HandleFunc("GET /preview/{$}", s.handlePreviewIndex)
mux.HandleFunc("GET /preview/onboard", s.handlePreviewOnboard)
mux.HandleFunc("GET /preview/work", s.handlePreviewWork)
mux.HandleFunc("GET /preview/work-item", s.handlePreviewWorkItem)
mux.HandleFunc("GET /preview/proposal", s.handlePreviewProposal)
```

`{$}` ensures `/preview/` matches the index exactly without swallowing subpaths (Go 1.22+ pattern). Static assets resolve via existing `staticHandler` since css is in `static/preview/`.

### 4.5 Fixtures as typed structs

Each page receives a typed struct (e.g. `LandingFixture`, `WorkBoardFixture`) defined in `preview_fixtures.go`. This makes the template binding contract explicit and lets future Phase B replace fixture literals with real service calls with minimal template churn.

## 5. Risks

| Risk | Mitigation |
|------|------------|
| Build tag drift (forgetting `_disabled.go` stub) breaks default build | Mirror exact pattern of `features_analytics_disabled.go`; add features_test.go-style invariant test |
| Mockup CSS uses CSS variables defined globally — pages may render unstyled | Inline a minimal `:root { --bg: ...; }` block at top of each page or include `style.css` link. Use `style.css` link via `/static/preview/style.css` |
| Template parse errors only surface at request time | Parse all templates at server startup (in `registerPreview`); panic if parse fails |
| Smoke test runs in default build mode (no tag) → tests skipped silently | Tag the test file `//go:build webui_preview` so it compiles only with the tag; CI matrix ensures it runs |
| `lint.yml` matrix change breaks existing single-job ergonomics | If matrix adds noise, split into a dedicated `build-matrix.yml` |
| Route prefix `/preview/` conflicts with future production route | Confirmed no current collision via `grep "/preview" internal/webui/*.go`; document the namespace as reserved in spec |

## 6. Testing Strategy

### Unit / smoke (`preview_test.go`, tag-gated)

For each of the 5 routes:
1. Start a test server with `NewFeatureRegistry()` (preview registered under tag).
2. `GET <route>` → assert status 200.
3. Assert response body contains the literal string `PREVIEW` (banner check).
4. Assert response `Content-Type` starts with `text/html`.

### Build matrix (CI)

- `go build ./...` (no tags) must succeed and contain no `preview` symbols. Verify via `go tool nm ./bin/wave | grep -i preview` exits non-zero (optional sanity check, not blocking).
- `go build -tags=webui_preview ./...` must succeed.
- `go test -tags=webui_preview ./internal/webui/...` runs the smoke tests.
- `go test ./internal/webui/...` (no tag) skips preview tests via tag exclusion.

### Lint matrix

Run golangci-lint in both modes to catch tag-only files that violate lint rules.

## 7. Open Questions Resolved

- **5 mockup files confirmed**: `00-landing.html`, `01-onboarding-session.html`, `02-work-board.html`, `03-work-item.html`, `04-evolution-proposal.html` (matches 5 routes in issue).
- **Router decision**: register on the main router via the existing `FeatureRegistry.routeFns` hook. No separate mux. (Resolves assessment open question §missing_info.)
