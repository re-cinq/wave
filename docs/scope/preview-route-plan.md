# Preview Route Plan — `/preview/*` in webui

**Status:** Plan only. No implementation yet.
**Date:** 2026-04-27
**Pairs with:** [`onboarding-as-session-plan.md`](onboarding-as-session-plan.md), [`mockups/`](mockups/)

---

## 1. Goal

Move the static dummies in `docs/scope/mockups/*.html` into the real Wave webui as a **`/preview/*` namespace**, so we can:

- Iterate on layout / IA / interaction inside the actual webui chrome (nav, theme, embedded fonts).
- Wire fixtures, then stub services, then real services — without changing the URL surface.
- Show the team / stakeholders something runnable behind a build flag, never shipped to default builds.

**Non-goals:**
- No new domain logic.
- No state DB changes.
- No replacement of existing routes (`/runs`, `/pipelines`, etc.) — preview lives alongside, doesn't compete.

---

## 2. Build gating

Two options.

### Option A — Build tag `webui_preview`
- New file: `//go:build webui_preview` on every preview handler/template-loader file.
- CI builds add `-tags webui_preview` for preview-enabled binaries.
- Default `go build` excludes preview entirely (zero binary cost when off).

### Option B — Runtime flag `WAVE_PREVIEW=1`
- Routes register conditionally based on env var or `manifest.Runtime.Preview` flag.
- Always compiled in (small binary cost), enable per-instance at runtime.

**Recommend A** — matches existing `//go:build webui` pattern (per ADR-003, `internal/webui` already gated). Zero footprint when off. Mirrors how `ontology` was historically gated.

If reviewers prefer B for ergonomics, fall back to B with `internal/webui/preview` package and a `if !cfg.PreviewEnabled { return }` guard in `registerRoutes`.

---

## 3. Route surface

| Path | Mockup | Purpose |
|---|---|---|
| `GET /preview/` | `index.html` | Preview index (links to all preview pages, version banner) |
| `GET /preview/onboard` | `01-onboarding-session.html` | Onboarding session (claude-code stream + form prompts) |
| `GET /preview/work` | `02-work-board.html` | Default landing — work-board (issues / PRs / scheduled) |
| `GET /preview/work-item` | `03-work-item.html` | Work-item detail + run-pipeline picker |
| `GET /preview/proposal` | `04-evolution-proposal.html` | Pipeline-evolution diff + signals + approve gate |

All `GET`-only. No mutation in preview phase A. Forms render but POST is no-op (or returns 200 with stub body).

A persistent **"PREVIEW" banner** at top-of-page makes it impossible to confuse with prod views.

---

## 4. Template structure

Reuse existing `internal/webui/templates/layout.html` (per arch audit: `embed.go:17–21`).

```
internal/webui/
  templates/
    layout.html               (existing, unchanged)
    partials/                 (existing)
    preview/                  (NEW)
      _nav_banner.html        (PREVIEW banner partial)
      index.html
      onboard.html
      work.html
      work-item.html
      proposal.html
      _onboard_msg.html       (chat message partial)
      _work_row.html          (list row partial)
      _diff_block.html        (diff renderer)
      _signal_card.html       (signal pill)
```

Each preview page extends `layout.html` and pulls in shared partials. Cloning per ADR-009 / current `internal/webui/embed.go:21` pattern.

Preview-specific styles either:
- (a) Inline `<style>` in each preview page during phase A (ports `mockups/style.css` token-by-token).
- (b) New `internal/webui/static/preview.css` served alongside existing CSS, loaded only on `/preview/*`.

**Recommend (b)** — keeps preview tokens isolated, easy to roll back, easy to diff against existing `static/style.css`.

---

## 5. Handler structure

```
internal/webui/
  preview.go              (build-tagged: //go:build webui_preview)
                          // registers /preview/* handlers + loads templates
  preview_fixtures.go     (build-tagged)
                          // exports fixtures: WorkItems(), Bindings(), Proposals()
  preview_test.go         (build-tagged)
                          // smoke: 200 OK on every preview route
```

`registerPreviewRoutes(mux *http.ServeMux, fixtures *PreviewFixtures)` is called from `registerRoutes` only when build tag is set.

No coupling to `state.StateStore`, `pipeline.Executor`, `adapter.Registry`, or any Domain service. Preview reads its own in-memory fixtures only.

---

## 6. Fixtures

Three layers, switchable per-route via query param `?source=fixture|stub|real`:

| Layer | Source | When |
|---|---|---|
| **fixture** (default) | Hard-coded Go structs in `preview_fixtures.go`. Same data as mockup pages. | Phase A. No runtime dependencies. |
| **stub** | `internal/service/*.Stub` implementations returning canned but pluggable data. | Phase B (after PRE-1 service layer lands). |
| **real** | Real `internal/service/*` against actual project state. | Phase C (after full service layer wired). |

Fixture file shape (sketch, not implementation):

```go
// preview_fixtures.go (build-tagged)
type PreviewFixtures struct {
    WorkItems   []WorkItem
    Bindings    map[string][]Binding
    Proposals   []Proposal
    OnboardLog  []OnboardEvent
    Signals     map[string]EvalSummary
}

func DefaultFixtures() *PreviewFixtures { ... }
```

Copy verbatim from the static mockups: `code-crispies` repo, issue #142, proposal #7, etc.

---

## 7. Migration phases

### Phase A — Static port (no dependencies)
- New build tag, package layout, templates copied from `docs/scope/mockups/`.
- All handlers return rendered HTML with hard-coded fixtures.
- Banner: "PREVIEW · phase A · static fixtures".
- **Reviewable in browser, identical to `mockups/*.html` but inside real chrome.**
- No dependency on PRE-1..6.

### Phase B — Stubbed services
- Lands after PRE-1 (`PipelineService`) and PRE-5 (StateStore extensions) — see main plan.
- Preview handlers call `service.PipelineService.Stub()` etc. that return the same fixture data.
- Banner: "PREVIEW · phase B · stub services".
- Validates that the service-layer interface shapes match what the UI actually needs. If they don't, refactor service before touching real data.

### Phase C — Real data, behind preview flag
- Preview routes call real services with real DB.
- Banner: "PREVIEW · phase C · live data — beta".
- Once stable, promote: drop the `webui_preview` tag, move templates from `templates/preview/` to `templates/`, replace existing `/runs`-rooted IA with the work-board IA.

### Phase D — Default
- Preview namespace deleted.
- New routes are the routes.

---

## 8. Risks

| Risk | Mitigation |
|---|---|
| Preview templates drift from production templates | Phase A is short. Promote to phase B as soon as PRE-1 lands. Don't let preview rot. |
| CSS tokens diverge from `static/style.css` | When phase D promotes, port tokens once; keep dark-only initially. |
| Build tag forgotten — preview leaks to prod | CI matrix builds with both `-tags webui_preview` and without. Smoke-test default build has zero `/preview/*` routes. |
| Fixtures become source of truth | Document explicitly: fixtures are throwaway. Phase B replaces them with stub-service shape. Don't grow fixtures into a fake DB. |
| Stakeholders see preview, ask for features | Banner copy is direct: "PREVIEW · not for production · features may be removed". |

---

## 9. Decisions captured

- **Build tag (Option A)** chosen over runtime flag.
- **Routes under `/preview/*`** not `/v2/*` or `/beta/*` — name signals throwaway.
- **Layout reuse** — same `layout.html`, new partials only.
- **Fixture-first** — phase A has zero dependencies on Domain refactor; can ship in parallel.
- **No JS yet.** Forms are visual only in phase A; real interactivity arrives in phase B alongside stub services.

---

## 10. Open questions for user

1. **Build tag name.** `webui_preview` or shorter `preview`? (Existing `webui` tag suggests namespacing.)
2. **Banner copy.** "PREVIEW · phase A · static fixtures" or terser ("preview · static")?
3. **CSS strategy.** Phase A: keep tokens in `mockups/style.css` or port to `internal/webui/static/preview.css` immediately? Recommend the latter (one source of truth) but defer if friction.
4. **Promote criterion.** What triggers phase C → D promotion? Suggest: when work-board has full feature parity with current `/runs` page + onboarding session works end-to-end on `code-crispies`.
5. **Mobile / small screen.** Phase A skips. Phase D requires. Worth a P3 note in webui-ux-audit?
6. **Preview index visible from prod nav?** No (default). Yes only if dev-mode env var set?

---

## 11. Estimated effort (when go-ahead given)

| Phase | Work | Notes |
|---|---|---|
| A | 2-3 days | Mostly mechanical port. Bulk is template clone + CSS port. |
| B | After PRE-1 (~1 sprint dependency); preview migration ~2 days | Stub service shapes are the design work. |
| C | Per-route, 1-2 days each, gated by underlying domain service readiness | Highest risk: real-data shapes don't match fixture shapes. |
| D | 1 day | Tag drop + IA promotion + delete preview package. |
