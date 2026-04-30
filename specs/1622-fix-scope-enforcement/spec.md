# fix(scope): 3 token-scope enforcement gaps — silent skips on error/unsupported

## Issue Metadata

- **Number**: #1622
- **URL**: https://github.com/re-cinq/wave/issues/1622
- **Labels**: enhancement, security
- **State**: OPEN
- **Author**: nextlevelshit

## Issue Body

### F1 — Token-scope audit follow-up (from Epic #1565)

Three gaps where token-scope validation silently skips enforcement instead of reporting violations.

### Finding 1 (HIGH) — `internal/scope/validator.go:151-153`

When `TokenIntrospector.Introspect()` returns `TokenInfo.Error != nil` (e.g. network failure, API error), validator appends a **warning** and `continue`s — scope check is silently skipped. If a persona declares `token_scopes: [repo]` and introspection fails, the persona runs unvalidated.

**Fix:** When scope is `required` (not `optional`), emit a `ScopeViolation` instead of warning. The persona explicitly asked for this scope — failing to validate it should block, not warn.

### Finding 2 (MED) — `internal/scope/resolver.go:28-31`

Bitbucket and unknown forge types return `nil, error` from `Resolve()` — but the error is informational ("not yet supported"), not a violation. The validator treats this as a non-blocking skip.

**Fix:** Return a violation with hint pointing to forge support status, not just an error that gets swallowed.

### Finding 3 (MED) — `internal/scope/introspect.go:102-105`

Fine-grained GitHub PATs lack `X-OAuth-Scopes` header. Introspector sets `TokenInfo.Error` with a message, which feeds into Finding 1's silent-skip path. Fine-grained PATs *do* have permissions — they're just not readable via the headers API.

**Fix:** Suggest token recreation as classic PAT (scopes readable) or add a `--skip-scope-check` flag for fine-grained PAT users. Violation hint: "fine-grained PATs cannot be introspected; recreate as classic PAT or use --skip-scope-check".

### LOW findings (document only)

- `internal/pipeline/executor_dispatch.go:538-553` — step-level `Permissions.AllowedTools` decoupled from persona `token_scopes`.
- Token-scope validation is persona-level only; step-level overrides bypass it.

## Acceptance Criteria

1. **Finding 1**: When `TokenInfo.Error != nil` and a persona declares `token_scopes`, a `ScopeViolation` is emitted (not just a warning).
2. **Finding 2**: Bitbucket/unknown forge types produce a `ScopeViolation` with a hint about forge support status.
3. **Finding 3**: Fine-grained PAT detection includes a remediation hint suggesting classic PAT recreation or `--skip-scope-check`.
4. All existing tests continue to pass (or are updated to reflect new behavior).
5. New tests cover each of the three findings.
