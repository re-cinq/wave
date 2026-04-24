# Implementation Plan: add(a, b int) int in math.go

## Objective

Add pure `func add(a, b int) int` utility to `math.go` at repository root, creating the file if absent. First of two helpers for the smoke-test epic.

## Approach

Single-file create (or append) with one pure function. No deps. Match the repository's existing package declaration; fall back to `package main` if `math.go` is new and no conflicting package exists.

## File Mapping

| Path | Action | Notes |
|------|--------|-------|
| `math.go` | create (or modify if exists) | Contains `package <match>` + `func add(a, b int) int { return a + b }` + one-line godoc |

No other files touched. No test files (explicitly out of scope per issue).

## Architecture Decisions

- **Package selection:** Use existing package in repository root. If `math.go` is new and root has other `.go` files, adopt their package. Else `package main`.
- **No overflow handling:** Issue explicitly excludes edge cases. Plain `a + b`.
- **One-line godoc:** Issue permits "one-line godoc" — include `// add returns the sum of a and b.` for `go vet` hygiene and exported-style clarity (even though lowercase).
- **No tests in this unit:** Issue scope excludes tests. Follow-up issue owns them.

## Risks

| Risk | Mitigation |
|------|-----------|
| Existing `math.go` with conflicting `add` | Read file first; append only if symbol absent |
| Package mismatch breaks `go build` | Inspect sibling `.go` files, match declaration |
| Scope creep (adding tests/mul) | Hard stop at single function; defer to follow-up |

## Testing Strategy

Issue excludes unit tests. Validation via:
- `go build ./...` — must succeed
- `go vet ./...` — must be clean
- Manual smoke: `add(2, 3) == 5` (asserted via acceptance criteria, no test code written)

Contract validation runs `go test ./...` at pipeline level — empty test set passes.
