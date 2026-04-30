# Work Items

## Phase 1: Setup
- [X] Item 1.1: Add `tools/` to `.gitignore` for the cached Tailwind binary
- [X] Item 1.2: Create `internal/webui/tailwind.input.css` with `@tailwind` base/components/utilities directives
- [X] Item 1.3: Create `internal/webui/tailwind.config.js` with content glob `internal/webui/templates/**/*.html`

## Phase 2: Build Wiring
- [X] Item 2.1: Add `tailwind` Makefile target — download pinned standalone binary into `tools/`, invoke compile to `internal/webui/static/tailwind.css` [P]
- [X] Item 2.2: Add `tailwind-check` Makefile target — regenerate + `git diff --exit-code internal/webui/static/tailwind.css` [P]
- [X] Item 2.3: Run `make tailwind` once, commit generated `internal/webui/static/tailwind.css`

## Phase 3: Template Swap
- [X] Item 3.1: Replace CDN script in `internal/webui/templates/work/board.html` with `<link rel="stylesheet" href="/static/tailwind.css">` [P]
- [X] Item 3.2: Replace CDN script in `internal/webui/templates/work/detail.html` with `<link>` [P]
- [X] Item 3.3: Update embed.go comment near `standalonePageTemplates` (CDN → utility classes)

## Phase 4: Testing
- [X] Item 4.1: Add `internal/webui/tailwind_test.go` asserting embedded CSS exists, non-empty, contains expected utility class (placed in `internal/webui/` not `static/` — keeping it inside the embed dir would create a stray Go package without access to `staticFS`)
- [X] Item 4.2: Run `go test ./internal/webui/...` — confirm template parse still passes
- [ ] Item 4.3: Manual: `make build && ./wave webui`, hit `/work` + `/work/<issue>`, verify no `cdn.tailwindcss.com` network requests (deferred — pipeline runs headless; covered by `TestStandaloneTemplatesUseEmbeddedTailwind` source-level guard)

## Phase 5: Docs + Polish
- [X] Item 5.1: Create `docs/build.md` documenting Tailwind toolchain install + regen workflow
- [X] Item 5.2: Cross-link `docs/build.md` from top-level Makefile help comment
- [ ] Item 5.3: Add `make tailwind-check` invocation to CI lint workflow (deferred — Phase D follow-up; keeps CI config out of this PR)
- [X] Item 5.4: Final: run `make tailwind-check && go test ./... && golangci-lint run ./...`
