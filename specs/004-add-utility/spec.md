# add utility add(a, b int) int in math.go

**Issue:** [nextlevelshit/wave-testing#4](https://github.com/nextlevelshit/wave-testing/issues/4)
**Parent:** #3
**State:** OPEN
**Author:** nextlevelshit
**Labels:** enhancement
**Complexity:** S (trivial)

## Summary

Add `add(a, b int) int` utility in `math.go` as first of two deterministic helpers for the smoke epic. This is a distinct unit of work: a single pure function with no dependencies, independently deliverable and testable. Establishes the `math.go` file if not present.

## Acceptance Criteria

- [ ] File `math.go` exists at repository root (create if missing) with `package main` or matching existing package.
- [ ] Function signature `func add(a, b int) int` present in `math.go`.
- [ ] Function returns the integer sum of its two arguments (e.g., `add(2, 3) == 5`).
- [ ] `go build ./...` succeeds without errors.
- [ ] `go vet ./...` reports no issues in `math.go`.

## Dependencies

None. Can be implemented in parallel with the `mul` sub-issue, but note scope risk below.

## Scope Notes

NOT included: unit tests, documentation comments beyond a one-line godoc, the `mul` function, edge-case handling (overflow), or any refactor of existing files. Tests and docs belong to separate follow-ups if desired.

## Complexity

S
