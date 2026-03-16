# Implementation Plan

## Objective

Add comprehensive test coverage for the security package's path validation (`internal/security/path.go`) and input sanitization (`internal/security/sanitize.go`), and fix a no-assertion test in `internal/workspace/workspace_test.go`.

## Approach

Create two new test files and modify one existing test:

1. **`internal/security/path_test.go`** — new file with table-driven tests for all `PathValidator` methods
2. **`internal/security/sanitize_test.go`** — extend existing file with tests for `SanitizeSchemaContent`, `ValidateInputLength`, `IsHighRisk`, `hashInput`, and strict-mode `SanitizeInput`
3. **`internal/workspace/workspace_test.go`** — fix `TestWorkspaceIsolation_NoPathTraversal` to add real assertions

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/security/path_test.go` | create | Table-driven tests for `ValidatePath`, `containsTraversal`, `validateApprovedDirectory`, `containsSymlinks`, `isWithinDirectory`, `SanitizePathForDisplay` |
| `internal/security/sanitize_test.go` | modify | Add tests for `SanitizeSchemaContent`, `ValidateInputLength`, `IsHighRisk`, strict-mode `SanitizeInput` |
| `internal/workspace/workspace_test.go` | modify | Fix `TestWorkspaceIsolation_NoPathTraversal` — add assertions that traversal path does NOT resolve to the sensitive file |

## Architecture Decisions

- Use `t.TempDir()` for filesystem-dependent tests (symlinks, path resolution) — automatic cleanup
- Follow existing test patterns: `DefaultSecurityConfig()` + `NewSecurityLogger(false)` for test setup
- Table-driven tests throughout, consistent with existing `sanitize_test.go` style
- Symlink tests use `os.Symlink` and skip on platforms that don't support them

## Risks

| Risk | Mitigation |
|------|------------|
| Symlink tests may fail on Windows/CI | Use `t.Skip()` with platform check |
| `filepath.Abs` behavior varies by CWD | Use `t.TempDir()` absolute paths for approved directories |
| `containsTraversal` checks `"./"` which is in cleaned paths | Test both raw and cleaned paths to document behavior |

## Testing Strategy

- All new tests are table-driven with descriptive subtest names
- Cover: valid paths, traversal attempts (../), encoded traversal (%2e%2e), symlinks, max length, absolute vs relative, approved directory checks
- Cover: script tag removal, event handler removal, javascript: URL removal, content size limits, strict mode rejection, input length validation, risk score thresholds
- Run with `go test -race ./internal/security/ ./internal/workspace/` to validate
