# Implementation Plan

## Objective

Replace `go vet ./...` with `golangci-lint run ./...` in the Makefile's `lint` target so local linting matches the CI pipeline.

## Approach

Single-line change in the Makefile. The CI workflow (`.github/workflows/lint.yml`) uses `golangci-lint v2.10` via `golangci-lint-action@v9`. The Makefile should invoke the same tool.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `Makefile` | modify | Change `lint` target from `go vet ./...` to `golangci-lint run ./...` |

## Architecture Decisions

- Use bare `golangci-lint run ./...` without pinning a version — the developer is responsible for installing a compatible version (documented in CLAUDE.md already: `golangci-lint run ./...`)
- No golangci-lint config file exists; the tool runs with defaults, matching CI behavior

## Risks

- **golangci-lint not installed locally**: Developers who don't have `golangci-lint` installed will get a "command not found" error. This is acceptable — the tool is a standard Go development dependency and is already referenced in CLAUDE.md.

## Testing Strategy

- Run `make lint` locally to verify it invokes `golangci-lint run ./...`
- No automated tests needed — this is a build tooling change
