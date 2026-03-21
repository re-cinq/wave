# test: fix t.Skip without linked issue in executor_schema_test.go

**Issue**: [#540](https://github.com/re-cinq/wave/issues/540)
**Author**: nextlevelshit
**Labels**: (none)
**Complexity**: trivial

## Description

`internal/pipeline/executor_schema_test.go:578` has `t.Skip("Symlink blocking feature not yet fully implemented")` which violates the project rule: "No t.Skip() without a linked issue."

Either create a tracking issue for the symlink blocking feature and link it, or delete the test if the feature is no longer planned.

## Acceptance Criteria

- The bare `t.Skip()` at `executor_schema_test.go:578` is resolved
- The resolution either:
  - Deletes the empty test stub (if the symlink blocking feature is not actively planned), OR
  - Links the skip to a tracking issue (if the feature is planned)
- `go test ./internal/pipeline/...` passes
- No new `t.Skip()` violations introduced
