# chore: update make lint to run golangci-lint matching CI

**Issue**: [#523](https://github.com/re-cinq/wave/issues/523)
**Author**: nextlevelshit
**Labels**: none
**Complexity**: trivial

## Description

The local `make lint` target only runs `go vet` while the CI pipeline runs the full `golangci-lint` suite. This creates a gap between local development and CI checks.

Update the Makefile to run `golangci-lint run ./...` locally to match CI behavior and catch linting issues before commit.

## Acceptance Criteria

- `make lint` runs `golangci-lint run ./...` instead of `go vet ./...`
- Local lint behavior matches CI (`.github/workflows/lint.yml` uses `golangci-lint-action@v9` with `golangci-lint v2.10`)
