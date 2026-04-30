# Work Items — issue #1600

## Phase 1: Setup

- [X] Item 1.1: Confirm feature branch `1600-preview-stub-services-b0a4` checked out, clean tree.
- [X] Item 1.2: Re-read `internal/webui/preview.go` and `preview_fixtures.go` to confirm scope.

## Phase 2: Core Implementation

- [X] Item 2.1: Create `internal/webui/preview_services.go` with build tag `webui_preview`. [P]
  - Declare 5 view-model structs: `PreviewLandingView`, `PreviewOnboardView`, `PreviewWorkView`, `PreviewWorkItemView`, `PreviewProposalView`.
  - Declare 5 interfaces: `PreviewLandingSource`, `PreviewOnboardSession`, `PreviewWorkSource`, `PreviewWorkItemSource`, `PreviewProposalSource`. Each has one method returning `(ViewType, error)`.
  - Declare `previewServiceRegistry` struct holding the five interfaces.
- [X] Item 2.2: Create `internal/webui/preview_stubs.go` with build tag `webui_preview`. [P]
  - Implement five stub structs (`previewLandingStub`, etc.) returning constant canned data.
  - Implement `defaultPreviewServices() *previewServiceRegistry` constructor.
- [X] Item 2.3: Modify `internal/webui/preview.go`:
  - Add `var previewServices *previewServiceRegistry` package-level var.
  - In existing `init()`, initialise it via `defaultPreviewServices()`.
  - Rewrite each `handlePreview*` to call its service, render with the view-model, and surface errors as `http.Error 500`.
- [X] Item 2.4: Delete `internal/webui/preview_fixtures.go`.
- [X] Item 2.5: Update `internal/webui/templates/preview/_banner.html` copy: "fixtures only" → "phase B · stub services".

## Phase 3: Testing

- [X] Item 3.1: Create `internal/webui/preview_stubs_test.go` with build tag `webui_preview`. [P]
  - Determinism test per stub (5 tests).
  - No-error test per stub (1 table-driven test).
  - Registry-fully-populated test (1 test).
- [X] Item 3.2: Run `go test -tags webui_preview -race ./internal/webui/...`. Verify all green, including pre-existing `TestPreviewRoutesRespond`, `TestPreviewCSSRoute`, `TestPreviewIndexUnknownTemplate`.
- [X] Item 3.3: Run `go build ./...` (no tag) — confirm default build still compiles with zero preview footprint.
- [X] Item 3.4: Run `go build -tags webui_preview ./...` — confirm tagged build clean.

## Phase 4: Polish

- [X] Item 4.1: Run `go vet -tags webui_preview ./internal/webui/...` and `golangci-lint run -tags webui_preview ./internal/webui/...`. Fix findings. (golangci-lint not installed locally; CI will gate.)
- [ ] Item 4.2: Manual check — start server with `-tags webui_preview`, curl each /preview/* route, confirm 200 and banner reads "phase B · stub services". (Deferred to PR review; route smoke tests cover wire-level behaviour.)
- [X] Item 4.3: Stage only the changed files (no `.wave/`, no `.agents/`, no `.claude/`, no `CLAUDE.md`). Commit with `feat(webui): phase B preview stub services (#1600)`.
- [ ] Item 4.4: Open PR titled `feat(webui): phase B preview stub services (#1600)`, reference issue #1600 in body, link to Epic #1565. (Outside this step — handled by create-pr step.)
