# Implementation Plan — #1601 Tailwind vendored compile

## 1. Objective

Replace runtime Tailwind CDN script in `templates/work/*.html` with a build-time-compiled, `go:embed`-served stylesheet so the WebUI works offline and has no third-party runtime dependency.

## 2. Approach

Use the **standalone Tailwind CLI binary** (downloaded to a `tools/` cache dir, gitignored) to scan `internal/webui/templates/**/*.html` and emit `internal/webui/static/tailwind.css`. Wire it into `make build` via a new `tailwind` target. The existing `staticFS` `//go:embed static/*` already covers the new file — no Go embed change required. Templates swap the CDN `<script>` for `<link rel="stylesheet" href="/static/tailwind.css">`.

The compiled CSS **is committed to the repo** so:
- `go install`/`go build` works without Node, npm, or the Tailwind binary
- CI does not need a Tailwind toolchain to verify Go builds
- Offline contributors can still build

A pre-commit/CI guard (`make tailwind-check`) regenerates and `git diff --exit-code`s to keep the committed CSS in sync.

### Why standalone binary over `npx @tailwindcss/cli`

- Zero npm/node dep at the Go-project root (no new `package.json`)
- Single ~30 MB binary, deterministic version pin
- Matches Wave's "no Node toolchain in core" stance (Node lives only under `docs/`)

### Tailwind CLI version

Pin to `tailwindcss-linux-x64` v3.4.x (latest stable v3) — v4 still in beta and changes config format. Source URL recorded in `Makefile` for reproducibility.

## 3. File Mapping

### Created

- `internal/webui/static/tailwind.css` — generated, committed
- `internal/webui/tailwind.config.js` — content globs (templates dir), theme passthrough
- `internal/webui/tailwind.input.css` — `@tailwind base; @tailwind components; @tailwind utilities;` source
- `docs/build.md` — front-end build steps, how to install Tailwind CLI, how to regenerate CSS
- `internal/webui/static/tailwind_test.go` — sanity test: file is non-empty, contains `--tw-` custom prop

### Modified

- `Makefile` — add `tailwind` target (download binary if missing, run compile), add `tailwind-check` (regenerate + diff), wire `build` to depend on `tailwind` only when `WAVE_BUILD_CSS=1` (default off so plain `go build` still works)
- `internal/webui/templates/work/board.html` — replace `<script src="https://cdn.tailwindcss.com">` with `<link rel="stylesheet" href="/static/tailwind.css">`
- `internal/webui/templates/work/detail.html` — same swap
- `.gitignore` — ignore `tools/tailwindcss` binary cache
- `internal/webui/embed.go` — update the comment near `standalonePageTemplates` from "Tailwind CDN classes" to "Tailwind utility classes" (no behavior change)

### Deleted

None.

## 4. Architecture Decisions

- **Commit generated CSS**: Trades repo size (~30 KB gzip) for buildability without Node. Aligns with Wave's monolith-binary distribution model.
- **Standalone binary, not npm**: Avoids dragging a Node toolchain into the Go project root. `docs/` already has its own `package.json` for VitePress; we keep that boundary.
- **Optional rebuild in `make build`**: Default `make build` does not invoke Tailwind (CSS is committed). `make tailwind` regenerates explicitly. CI runs `make tailwind-check` to enforce sync.
- **No CDN fallback**: Per acceptance criteria, runtime must have zero CDN dependency. Templates only reference `/static/tailwind.css`.
- **Config scope**: Tailwind config scans `internal/webui/templates/**/*.html` only. JavaScript files are not Tailwind-source today.

## 5. Risks

| Risk | Mitigation |
|---|---|
| Standalone binary URL changes / 404 | Pin version, document fallback `npx @tailwindcss/cli@3.4` invocation |
| Generated CSS drifts from templates | `make tailwind-check` in CI; PR fails if diff |
| Existing `style.css` collides with Tailwind utilities | Standalone pages already isolated (`standalonePageTemplates`); main `style.css` unaffected |
| Contributor edits template, forgets to regenerate | `tailwind-check` CI gate catches it |
| Binary not executable in sandboxed CI | Use `npx @tailwindcss/cli` fallback path documented in `docs/build.md` |

## 6. Testing Strategy

- **Unit**: `internal/webui/static/tailwind_test.go` — assert `tailwind.css` exists in `staticFS`, is non-empty, contains a known utility class string (e.g. `bg-slate-50` used by board.html).
- **Integration**: Existing `embed_test.go` template-parse test still passes (no template structural change).
- **Manual smoke**: `make build && ./wave webui` then load `/work` and `/work/<issue>`, verify styling identical to CDN version, no network requests to `cdn.tailwindcss.com` (DevTools network tab).
- **CI gate**: `make tailwind-check` runs in lint workflow — fails on uncommitted CSS regen.

## 7. Out of Scope

- Migrating other pages off the project `style.css` to Tailwind (separate refactor).
- Tailwind v4 upgrade.
- Purging unused Tailwind classes from compiled output beyond default content-scan behavior.
