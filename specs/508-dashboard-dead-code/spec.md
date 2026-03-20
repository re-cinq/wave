# audit: partial — dashboard.go dead code re-added (#353)

**Issue**: [#508](https://github.com/re-cinq/wave/issues/508)
**Labels**: audit
**Author**: nextlevelshit
**Detected by**: wave-audit pipeline run 2026-03-20

## Background

PR #353 (`fix(display): remove unused DisplayConfig fields and dead dashboard methods`) removed dead methods from `dashboard.go`. However, commit `64ea502` (`feat: optional pipeline steps`) later re-added content to the file. The audit found that `internal/display/dashboard.go` still exists at HEAD and `internal/display/types.go` still has DisplayConfig occurrences.

## Evidence

- `internal/display/dashboard.go` — still exists, not removed
- `git log` shows commit ae56cb9 removed dead methods from dashboard.go
- Later commit 64ea502 re-added content to dashboard.go (feat: optional pipeline steps)
- `internal/display/types.go` — still has 6 DisplayConfig occurrences
- PR claimed dashboard.go would have 159 lines removed; file still present at HEAD

## Remediation

Audit `internal/display/dashboard.go` for any methods that are again unused since commit 64ea502 re-added content. Verify which DisplayConfig fields remain genuinely unused.

## Acceptance Criteria

- [ ] All dead code in `internal/display/dashboard.go` is identified and removed
- [ ] All dead code in `internal/display/types.go` related to this audit is identified and removed
- [ ] Tests referencing removed dead code are updated or removed
- [ ] All remaining tests pass (`go test ./...`)
- [ ] No new compilation warnings introduced
