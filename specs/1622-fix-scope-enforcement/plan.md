# Implementation Plan

## Objective

Fix three token-scope enforcement gaps where validation silently skips instead of reporting violations: (1) introspection errors treated as warnings, (2) unsupported forge types silently skipped, (3) fine-grained PAT detection without remediation guidance.

## Approach

Convert silent-skip patterns into explicit `ScopeViolation` entries with actionable hints. The key insight is that when a persona declares `token_scopes`, it is a **requirement** — failure to validate should block execution, not warn.

### Finding 1 (HIGH): Introspection errors → violations

In `validator.go`, the block at lines 151-153 currently appends a warning and continues. Change it to append a `ScopeViolation` when the persona has declared required scopes. The violation should include the introspection error details and a hint about checking token configuration.

### Finding 2 (MED): Unsupported forge types → violations

In `validator.go`, the resolver error path at lines 133-137 also appends a warning. For Bitbucket and unknown forges, the resolver returns an error that gets swallowed. Instead, emit a `ScopeViolation` with a hint indicating the forge type is not yet supported for scope validation.

### Finding 3 (MED): Fine-grained PAT hint

In `introspect.go`, the fine-grained PAT detection (lines 102-105) already sets an error. The fix is in the validator: when `TokenInfo.TokenType == "fine-grained"`, include a specific remediation hint in the violation: "fine-grained PATs cannot be introspected; recreate as classic PAT or use --skip-scope-check".

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/scope/validator.go` | Modify | Change warning→violation for introspection errors (F1) and resolver errors (F2). Add fine-grained PAT hint (F3). |
| `internal/scope/validator_test.go` | Modify | Update `TestValidatePersonas_IntrospectionFailure` to expect violations instead of warnings. Add new tests for F2, F3. |
| `specs/1622-fix-scope-enforcement/` | Create | Spec, plan, and task files. |

## Architecture Decisions

1. **All scope requirements are required by default**: When a persona declares `token_scopes`, those are hard requirements. There is no "optional" scope concept in the current codebase, so all introspection/resolution failures should be violations.
2. **Preserve TokenInfo.TokenType for hint generation**: The `fine-grained` token type is already tracked; use it to provide specific remediation hints.
3. **No new types or interfaces**: The existing `ScopeViolation` and `ValidationResult` types are sufficient. No API changes needed.
4. **Defer `--skip-scope-check` flag**: The issue mentions this as a possible follow-up. The core fix is to emit violations with hints; the flag can be added later.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing pipelines that relied on warnings | High | Existing pipelines with failing introspection may now be blocked. This is intentional — the previous behavior was a security gap. |
| Gitea introspection already returns errors for all cases | Medium | Gitea's introspector always returns an error (it can't read scopes). This will now become a violation. May need to treat Gitea specially or document the limitation. |
| Test changes required | Low | Update tests to reflect new violation behavior. |

## Testing Strategy

1. **Unit tests**: Update `TestValidatePersonas_IntrospectionFailure` to expect `HasViolations() == true` instead of warnings.
2. **New test for F2**: Test that Bitbucket forge type produces a `ScopeViolation` (not just a warning).
3. **New test for F3**: Test that fine-grained PAT detection includes the correct remediation hint.
4. **Regression tests**: Ensure all other existing tests pass unchanged.
