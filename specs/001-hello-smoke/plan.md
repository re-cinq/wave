# Implementation Plan: hello.go smoke

## Objective
Create `hello.go` at repo root defining `package main` with `hello()` returning `"hello, world"`. Validates impl-issue pipeline end-to-end.

## Approach
Write single file verbatim from issue body. No dependencies, no refactor, no extra scaffolding.

## File Mapping

| Path | Action | Purpose |
|------|--------|---------|
| `hello.go` | create | Contains `package main` + `hello()` function |

## Architecture Decisions

- **Exact snippet**: Copy code block from issue verbatim — no deviations.
- **Package `main`**: Matches issue spec even though function unexported. No `main()` entry required by issue.
- **Repo root placement**: Issue explicitly demands root, not subdir.
- **No test file**: Issue acceptance does not require unit test; PR body will document manual verification. Optional lightweight test added only if CI/contract demands compilation check.

## Risks

| Risk | Mitigation |
|------|------------|
| Package `main` with no `main()` may fail `go build ./...` | Keep as-is per issue; if build fails, add minimal `func main() {}` — flag in PR |
| File already exists | Overwrite only if content differs; otherwise no-op |
| Formatting drift | Run `gofmt -w hello.go` after write |

## Testing Strategy

- **Compile check**: `go vet ./...` and `go build ./...` must succeed.
- **Manual verification**: Inspect file contents match snippet exactly.
- **No unit test required** by acceptance criteria; skip to keep PR minimal.
