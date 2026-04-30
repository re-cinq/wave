# Building Wave

This page covers the build steps that go beyond a plain `go build`. Most
contributors only need `make build`, since the front-end assets are
committed to the repo.

## Quick reference

| Target              | Purpose                                                       |
|---------------------|---------------------------------------------------------------|
| `make build`        | Compile the `wave` binary. No Node, no Tailwind binary needed.|
| `make test`         | `go test -race ./...`                                         |
| `make lint`         | `golangci-lint run ./...`                                     |
| `make tailwind`     | Regenerate `internal/webui/static/tailwind.css` (front-end).  |
| `make tailwind-check` | Regenerate + fail if the committed CSS drifts.              |

## Front-end: vendored Tailwind

The WebUI work board (`/work`, `/work/<issue>`) renders standalone pages
that use Tailwind utility classes. To keep the binary self-contained and
offline-friendly, the compiled stylesheet lives at
`internal/webui/static/tailwind.css` and is embedded via `//go:embed`.

The file is **committed** to the repo so that:

- `go install` and plain `go build` work without Node, npm, or the
  Tailwind CLI.
- CI does not need a front-end toolchain to verify Go builds.
- Offline contributors can still build.

### Regenerating `tailwind.css`

Run the regen whenever you edit a template under
`internal/webui/templates/**/*.html` and add or change Tailwind utility
classes:

```bash
make tailwind
git add internal/webui/static/tailwind.css
```

`make tailwind` downloads the pinned standalone Tailwind CLI binary into
`tools/` (gitignored) on first use, then runs it with the project's
`internal/webui/tailwind.config.js` and `internal/webui/tailwind.input.css`
to scan the templates and emit a minified stylesheet.

### Drift check (CI)

```bash
make tailwind-check
```

Regenerates the CSS and runs `git diff --exit-code` against the committed
file. The target fails the build if a contributor edited a template
without rerunning `make tailwind`. Wire this into CI alongside `make lint`.

### Pinned version

The Tailwind CLI version is pinned in the `Makefile` via
`TAILWIND_VERSION` (currently `v3.4.17`). Bumping the pin requires
regenerating the CSS in the same commit so `tailwind-check` stays green.

### Fallback: `npx`

If the pinned standalone binary is unreachable (release URL changes,
restricted network), you can run the same compile via npm:

```bash
cd internal/webui
npx -y tailwindcss@3.4.17 \
  --config tailwind.config.js \
  --input  tailwind.input.css \
  --output static/tailwind.css \
  --minify
```

This is a contingency path; `make tailwind` is the supported workflow.

## Runtime: no CDN dependency

The vendored stylesheet replaces the previous
`<script src="https://cdn.tailwindcss.com">` runtime dependency. Wave's
WebUI now makes zero third-party network requests for styling — verify
in DevTools' Network tab while loading `/work`.
