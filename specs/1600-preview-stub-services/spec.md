# Phase 1.5b: /preview/* phase B stub services

**Issue:** [re-cinq/wave#1600](https://github.com/re-cinq/wave/issues/1600)
**Author:** nextlevelshit
**State:** OPEN
**Labels:** enhancement, ready-for-impl, frontend

## Issue body

Part of Epic #1565 Phase 1.5.

### Goal

Promote /preview/* from fixtures-only (Phase A — PR #1585) to stub services. Routes still build-tag-gated (`webui_preview`), but now backed by interface-shaped service stubs that let designers wire mock data flows.

### Acceptance criteria

- [ ] Fixture data structs replaced by service interfaces (e.g. PreviewWorkSource, PreviewOnboardSession) with stub impls
- [ ] Service interfaces declared so future Phase D real impls can drop in cleanly
- [ ] All 5 /preview/* routes still functional with stub data
- [ ] Build remains tag-gated; default build = zero preview footprint
- [ ] Test coverage: stub services return fixed sample data deterministically

### Dependencies

- 1.5a Phase A (PR #1585 MERGED)

## Context

Five `/preview/*` routes exist behind `webui_preview` build tag (per Phase A, PR #1585):

| Path                  | Mockup file       | Page name   |
|-----------------------|-------------------|-------------|
| `GET /preview/{$}`    | index.html        | index       |
| `GET /preview/onboard`| onboard.html      | onboard     |
| `GET /preview/work`   | work.html         | work        |
| `GET /preview/work-item` | work_item.html | work_item   |
| `GET /preview/proposal`  | proposal.html  | proposal    |

Plus `GET /preview/static/style.css` for shared CSS.

Current fixture state (`internal/webui/preview_fixtures.go`): five empty typed structs (`previewLandingFixture`, `previewOnboardFixture`, `previewWorkFixture`, `previewWorkItemFixture`, `previewProposalFixture`). Templates render hardcoded HTML; structs are placeholders only.

Phase B task: replace empty struct fixtures with **interface-shaped service stubs** that return typed view-models. Goal is to validate the service-layer interface shape before Phase C wires real data — if the interfaces are wrong, refactor service before touching real services.

## Acceptance criteria (mapped to verifiable checks)

1. `internal/webui/preview_fixtures.go` deleted; replaced by per-route service interfaces in `internal/webui/preview_services.go` and stub impls in `internal/webui/preview_stubs.go`.
2. Five service interfaces declared (one per route): `PreviewLandingSource`, `PreviewOnboardSession`, `PreviewWorkSource`, `PreviewWorkItemSource`, `PreviewProposalSource`. Each returns a typed view-model struct.
3. Handlers call services rather than reading global fixture vars. Service registry struct (`previewServices`) holds the five sources, default-initialised to stub implementations at `init()`.
4. All five routes still return HTTP 200 with banner + stylesheet link (existing tests in `preview_test.go` still pass).
5. Build remains tag-gated: `preview_disabled.go` unchanged, default `go build ./...` produces zero preview symbols.
6. Test coverage: `preview_stubs_test.go` asserts each stub returns deterministic sample data across repeated calls (same input → same output, no randomness).
7. Banner copy updated: "PREVIEW · phase B · stub services".

## Out of scope (deferred to Phase C)

- Wiring templates to actually render the typed view-models (templates remain static; Phase C work).
- Real service implementations (Phase C).
- Removal of build tag (Phase D).
- Mutation endpoints / form POSTs.
