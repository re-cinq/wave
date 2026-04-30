# Implementation plan — issue #1600

## 1. Objective

Replace the five empty fixture structs in `internal/webui/preview_fixtures.go` with five named **service interfaces** + stub implementations, so /preview/* handlers depend on contracts that future real services can satisfy without changing handler code.

## 2. Approach

1. Define five service interfaces in a new file `internal/webui/preview_services.go` (build-tagged `webui_preview`). Each interface has one method that returns a typed view-model and `error`.
2. Define typed view-model structs (one per route) in the same file. Structs are minimal sketches — only the fields the eventual templates will need. They can stay sparse in Phase B; Phase C grows them.
3. Provide stub implementations in `internal/webui/preview_stubs.go` (build-tagged) that return canned, deterministic data.
4. Introduce a small `previewServices` registry struct holding the five sources. Initialise it once in `init()` to stub implementations. Handlers read from this registry instead of from package-level fixture vars.
5. Delete `internal/webui/preview_fixtures.go`.
6. Update banner copy in `_banner.html` from "fixtures only" to "phase B · stub services".
7. Add unit tests in `internal/webui/preview_stubs_test.go` (build-tagged) covering deterministic return for each stub.
8. Existing `preview_test.go` (5-route smoke + CSS + unknown-template) should pass unchanged. Adjust the banner-string assertion only if it was matching exact phase-A copy.

## 3. File mapping

### Created

- `internal/webui/preview_services.go` — interfaces + view-model structs (build-tagged `webui_preview`).
- `internal/webui/preview_stubs.go` — stub implementations + `defaultPreviewServices()` constructor (build-tagged).
- `internal/webui/preview_stubs_test.go` — determinism tests (build-tagged).

### Modified

- `internal/webui/preview.go`:
  - Remove direct references to `landingFixture`, `onboardFixture`, `workFixture`, `workItemFixture`, `proposalFixture`.
  - Add `previewServices` package-level registry initialised in `init()`.
  - Each `handlePreview*` calls the corresponding service method, surfaces error as 500 with quoted message, and passes the view-model to `renderPreview`.
- `internal/webui/templates/preview/_banner.html` — change "fixtures only" → "phase B · stub services".
- `internal/webui/preview_test.go` — update the `strings.Contains(body, "PREVIEW")` check is fine (uppercase still matches), but if any test asserts "fixtures only" copy, update accordingly. (Current test only checks for `"PREVIEW"` — safe.)

### Deleted

- `internal/webui/preview_fixtures.go` (replaced by `preview_services.go` + `preview_stubs.go`).

### Untouched

- `internal/webui/preview_disabled.go` — build-tag stub for default builds; signature `registerPreview(*FeatureRegistry)` unchanged.
- All `internal/webui/templates/preview/*.html` (other than `_banner.html`) — templates remain static. Phase C wires them to view-models.
- `internal/webui/static/preview/style.css` — unchanged.

## 4. Architecture decisions

### One interface per route, not one God interface

The acceptance text uses `PreviewWorkSource` and `PreviewOnboardSession` as examples — both per-route. Five small interfaces keep blast radius low when Phase C swaps in real services (each can land independently). A single `PreviewSource` would force every Phase C PR to satisfy all five at once.

### View-models as named structs, not `any`

Even though Phase B templates don't consume the data yet, declaring the typed struct now is the whole *point* of Phase B — the interface shape is what gets validated. A stub returning `any` defers the design work to Phase C and defeats the purpose.

### Registry as package-level var, not constructor parameter

`registerPreview` already takes only `*FeatureRegistry` — no `*Server`. To preserve that signature (and the existing test seam `newPreviewMux` that passes `nil` for `*Server`), services live in a package-level var initialised at `init()`. Phase C/D may later thread services via `*Server` if dependency-injection becomes warranted; for stub-only Phase B, package-level is the smallest change.

### Error returns from stubs

Stub methods return `(ViewModel, error)` even though stubs never error today. Handlers handle the error with `http.Error 500`. This matches what real services in Phase C will need (DB lookups can fail) and keeps handlers correct ahead of time. No interface churn between B and C.

### Banner copy update

The fixture banner explicitly says "fixtures only". Acceptance criterion 4 ("routes functional with stub data") is satisfied at the wire level, but the user-visible label should reflect reality. Per `docs/scope/preview-route-plan.md` § Phase B: banner becomes "PREVIEW · phase B · stub services".

## 5. Risks

| Risk | Mitigation |
|------|------------|
| Tests in `preview_test.go` assert banner string matching phase-A copy | Current test only checks `Contains(body, "PREVIEW")` — both copies match. No test change needed. Verified by reading the test file. |
| View-model struct shapes guess wrong at what templates need in Phase C | Phase B explicitly *is* the validation step for shape. If shapes turn out wrong in Phase C, refactor the interface — that's the point. Keep structs sparse (3–5 fields each) so refactor cost is low. |
| Stubs become a hidden source of "fake data" used in non-preview code paths | All new files are build-tagged `webui_preview`. Default builds cannot reference them. Verified by `preview_disabled.go` providing only the registrar stub. |
| Determinism test brittleness if stubs use `time.Now()` or `rand` | Stubs MUST use only constant literals. Test asserts byte-equality across two calls. |
| Adding service registry breaks the `newPreviewMux` test seam (passes `nil` *Server*) | Registry is a package-level var, not on `*Server`. Test seam unaffected. |

## 6. Testing strategy

### Existing tests (must continue passing)

- `TestPreviewRoutesRespond` — five routes return 200 + HTML + banner string `PREVIEW` + stylesheet link.
- `TestPreviewCSSRoute` — CSS route returns 200 with `text/css`.
- `TestPreviewIndexUnknownTemplate` — unknown template name returns 500.

### New tests (`preview_stubs_test.go`, `//go:build webui_preview`)

- `TestPreviewLandingStubDeterministic` — call `Landing()` twice, deep-equal results.
- `TestPreviewOnboardStubDeterministic` — same pattern.
- `TestPreviewWorkStubDeterministic` — same pattern.
- `TestPreviewWorkItemStubDeterministic` — same pattern.
- `TestPreviewProposalStubDeterministic` — same pattern.
- `TestPreviewStubsReturnNoError` — each stub returns nil error.
- `TestDefaultPreviewServicesAllPopulated` — registry constructor returns non-nil interface impls for all five fields.

### Default-build smoke

Verified manually via `go build ./...` (no `-tags webui_preview`). Should produce no symbols referencing `previewServices`, `PreviewLandingSource`, etc. Existing `preview_disabled.go` is the only `webui_preview`-free file.

### Race-safety

`go test -race ./internal/webui/...` with `-tags webui_preview`. Registry init happens once at package init; reads are concurrent-safe (immutable interface impls).

## 7. Validation gate before merge

- `go build ./...` — default build clean (no preview symbols).
- `go build -tags webui_preview ./...` — preview build clean.
- `go test -tags webui_preview -race ./internal/webui/...` — all tests green.
- `go vet -tags webui_preview ./internal/webui/...` — no findings.
- `golangci-lint run -tags webui_preview ./internal/webui/...` — no new findings.
- Manual: hit each of the 5 preview routes, confirm 200 + banner says "phase B · stub services".
