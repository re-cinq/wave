# 1.5a: /preview/* phase A (build tag, fixtures only)

**Issue:** [re-cinq/wave#1580](https://github.com/re-cinq/wave/issues/1580)
**Author:** nextlevelshit
**State:** OPEN
**Labels:** (none)

## Issue Body

Part of Epic #1565. Phase 1.5, parallel-safe (no Phase 1 deps for phase A).

Port `docs/scope/mockups/*.html` to `internal/webui/templates/preview/` behind a `webui_preview` build tag. Hard-coded fixtures only (no service wiring).

**Files:**
- New: `internal/webui/preview.go` (`//go:build webui_preview`)
- New: `internal/webui/preview_fixtures.go`
- New: `internal/webui/templates/preview/*.html` ported from `docs/scope/mockups/`

**Routes (all GET-only, fixture-backed):**
- `/preview/` (index) · `/preview/onboard` · `/preview/work` · `/preview/work-item` · `/preview/proposal`

**Acceptance:**
- [ ] Persistent PREVIEW banner visible on every preview route
- [ ] Default `go build` excludes preview entirely (zero footprint)
- [ ] CI matrix builds with both `-tags webui_preview` and without
- [ ] Smoke test: 200 OK on every preview route under build tag

**Pipeline:** `impl-issue` (`--adapter claude --model cheapest`)

## Source Mockups

`docs/scope/mockups/` contains:
- `00-landing.html` → `/preview/` (index)
- `01-onboarding-session.html` → `/preview/onboard`
- `02-work-board.html` → `/preview/work`
- `03-work-item.html` → `/preview/work-item`
- `04-evolution-proposal.html` → `/preview/proposal`
- `style.css` (shared stylesheet, served as static asset)

## Acceptance Criteria

1. Build tag `webui_preview` gates all preview code; default build has zero footprint (no symbols, no embedded templates).
2. Five GET routes registered under build tag — each returns 200 with rendered fixture HTML.
3. Persistent `PREVIEW` banner visible on every preview page.
4. CI builds binary in both modes (without tag, with `-tags webui_preview`).
5. Smoke test asserts 200 OK + banner presence on each route, gated by build tag.
6. No service-layer wiring — fixtures are package-level Go literals.
