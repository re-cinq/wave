# Implementation Plan: Fix Auto-Close Keywords

## Objective

Replace GitHub closing keywords (`Closes #N`, `Fixes #N`, `Resolves #N`) with non-closing references (`Related to #N`) in all Wave pipeline PR creation prompts, so that closing a PR without merging does not falsely close linked issues.

## Approach

Use **non-closing references** (`Related to #N`) in all PR body templates. This is the simplest, most reliable strategy because:
- It prevents false-positive issue closures at the source
- No external tooling (GitHub Actions, webhooks) needed
- Works identically across all forge platforms (GitHub, GitLab, Gitea, Bitbucket)
- Issues can still be manually closed or closed by a merge commit with closing keywords added at merge time by the human reviewer

The format validator must also be updated to accept `Related to #N` references instead of only `Closes/Fixes/Resolves`.

## File Mapping

### Files to Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/defaults/prompts/implement/create-pr.md` | modify | Replace `Closes #N` with `Related to #N` in all forge templates and constraints |
| `internal/contract/format_validator.go` | modify | Update regex and violation message to accept `Related to #N` alongside closing keywords |
| `internal/contract/format_validator_test.go` | modify | Update test case to use `Related to #N` |
| `.wave/prompts/gh-implement/create-pr.md` | modify | Replace `Closes #N` with `Related to #N` |
| `.wave/prompts/gl-implement/create-mr.md` | modify | Replace `Closes #N` with `Related to #N` |
| `.wave/prompts/gt-implement/create-pr.md` | modify | Replace `Closes #N` with `Related to #N` |
| `.wave/prompts/bb-implement/create-pr.md` | modify | Replace `Closes #N` with `Related to #N` |
| `.wave/prompts/gh-implement-epic/report.md` | modify | Update search pattern to also find `Related to #N` PRs |
| `.wave/prompts/gl-implement-epic/report.md` | modify | Update search pattern to also find `Related to #N` MRs |
| `.wave/prompts/gt-implement-epic/report.md` | modify | Update search pattern to also find `Related to #N` PRs |
| `.wave/prompts/bb-implement-epic/report.md` | modify | Update search pattern to also find `Related to #N` PRs |
| `internal/defaults/pipelines/wave-audit.yaml` | modify | Add `Related to #N` to search patterns (alongside existing closing keywords) |
| `.wave/pipelines/wave-audit.yaml` | modify | Same as above (local override copy) |

### Files NOT Changed (by design)

| File | Reason |
|------|--------|
| `internal/defaults/prompts/speckit-flow/create-pr.md` | Already uses safe pattern (no closing keywords) |
| `.wave/prompts/speckit-flow/create-pr.md` | Already uses safe pattern (no closing keywords) |

## Architecture Decisions

1. **`Related to #N` over `For #N`**: "Related to" is more widely recognized across platforms and conveys the relationship clearly without triggering auto-close behavior on any platform.

2. **Keep validator accepting both patterns**: The format validator should accept BOTH `Related to #N` AND closing keywords (`Closes/Fixes/Resolves #N`). This is forward-compatible — if someone manually uses closing keywords in a custom PR, the validator won't reject it.

3. **Epic reports use both patterns**: The epic report search commands must search for BOTH old closing keywords AND new non-closing references, since historical PRs use the old pattern and new PRs will use the new pattern.

4. **Audit pipeline backward compatibility**: The wave-audit pipeline reads historical data, so it must continue searching for `Fixes #N`, `Closes #N` in addition to the new `Related to #N`.

## Risks

| Risk | Mitigation |
|------|-----------|
| Issues no longer auto-close on merge | Humans must manually close issues after merge, or add closing keywords to merge commit. This is the desired behavior — deliberate closure. |
| Historical PRs searched differently | Epic report and audit patterns updated to search both old and new patterns |
| Other closing keyword locations missed | Comprehensive grep identified all 12 files; speckit-flow already safe |

## Testing Strategy

1. **Unit tests**: Update `format_validator_test.go` to verify both `Related to #N` and `Closes #N` are accepted
2. **Manual validation**: Run `go test -race ./...` to ensure no regressions
3. **Grep audit**: Post-change grep for `Closes #|Fixes #|Resolves #` should only find:
   - Read-only search patterns in audit/epic report contexts
   - Format validator regex (which accepts both patterns)
   - Test fixtures using both patterns
