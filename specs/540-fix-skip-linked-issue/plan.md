# Implementation Plan

## Objective

Remove the empty test stub `TestContractPrompt_SymlinkBlocking` that contains a bare `t.Skip()` without a linked issue, violating the project's testing policy.

## Approach

Delete the test function entirely. The symlink blocking feature has no implementation anywhere in the codebase — the test is a dead placeholder with no corresponding code. Creating a tracking issue for a feature with zero implementation artifacts is unnecessary overhead.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/executor_schema_test.go` | modify | Remove `TestContractPrompt_SymlinkBlocking` function (lines 577-580) |

## Architecture Decisions

- **Delete over link**: The assessment and codebase search confirm no symlink blocking implementation exists. Deleting the empty stub is cleaner than creating a tracking issue for an unplanned feature.
- **Minimal change**: Only the 4-line function (comment + signature + skip + closing brace) is removed. No other modifications.

## Risks

- **None significant**: The test has no body — it only calls `t.Skip()`. Removing it cannot break anything.

## Testing Strategy

- Run `go test ./internal/pipeline/...` to confirm no regressions
- Verify no remaining bare `t.Skip()` calls without linked issues via grep
