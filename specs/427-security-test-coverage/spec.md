# test(security): add test coverage for path validation and input sanitization

**Issue**: [#427](https://github.com/re-cinq/wave/issues/427)
**Author**: nextlevelshit
**State**: OPEN

## Coverage Gap Analysis (wave-test-hardening audit)

### IMP-001: Path validation entirely untested (HIGH)
- `internal/security/path.go`: `ValidatePath`, `containsTraversal`, `validateApprovedDirectory`, `containsSymlinks`, `isWithinDirectory`, `SanitizePathForDisplay` — all at 0% coverage
- This is a critical security package — path traversal prevention and symlink checks are unverified

### IMP-002: Sanitization gaps (HIGH)
- `internal/security/sanitize.go`: `SanitizeSchemaContent`, `ValidateInputLength`, `IsHighRisk`, `hashInput`, `calculateRiskScore`, strict-mode `SanitizeInput` — all untested
- `removeSuspiciousContent` for script tags, event handlers, javascript: URLs has zero coverage

### IMP-017: TestWorkspaceIsolation_NoPathTraversal has no assertions (HIGH)
- Test exists but has no actual assertions — false confidence

### Required tests
- Table-driven tests for `ValidatePath`: traversal sequences, symlinks, empty dirs, absolute vs relative, max length
- `SanitizeSchemaContent` with script tags, event handlers, javascript: URLs
- `SanitizeInput` strict mode with injection patterns
- Fix `TestWorkspaceIsolation_NoPathTraversal` to actually assert

## Acceptance Criteria

1. All functions in `internal/security/path.go` have table-driven test coverage
2. `SanitizeSchemaContent` tested with script tags, event handlers, javascript: URLs
3. `SanitizeInput` strict mode tested with injection pattern rejection
4. `ValidateInputLength` and `IsHighRisk` have dedicated tests
5. `TestWorkspaceIsolation_NoPathTraversal` in `internal/workspace/workspace_test.go` has real assertions
6. All tests pass with `go test -race ./internal/security/ ./internal/workspace/`
