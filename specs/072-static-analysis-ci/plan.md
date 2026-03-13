# Implementation Plan: Static Analysis CI

## Objective

Add a golangci-lint v2 configuration and GitHub Actions workflow to enforce static analysis for unused/redundant Go code on every PR and push to main.

## Approach

1. Create a `.golangci.yml` at repo root using the v2 configuration format with the `standard` linter preset plus targeted additional linters for dead code detection
2. Add a separate `.github/workflows/lint.yml` workflow using `golangci-lint-action@v9`
3. Enable `only-new-issues` for incremental adoption — existing violations are not blockers
4. Update CLAUDE.md testing section with the lint command

## File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `.golangci.yml` | create | v2-format linter configuration |
| `.github/workflows/lint.yml` | create | GitHub Actions lint workflow |
| `CLAUDE.md` | modify | Add lint command to Testing section |

## Architecture Decisions

### AD-1: v2-format configuration
The issue's original config uses v1 syntax. golangci-lint v2.9.0 cannot parse v1 configs. Use v2 format with `version: "2"` and `linters.default: standard` preset.

### AD-2: Standard preset + targeted additions
The `standard` preset covers: copyloopvar, errcheck, govet, ineffassign, staticcheck, unused. Add `unparam` (unused params), `wastedassign` (wasted assignments), `gocritic` (code quality), and `nolintlint` (nolint hygiene).

### AD-3: Incremental adoption via `only-new-issues`
Rather than fixing all existing violations upfront, use the action's `only-new-issues` flag. This filters lint results to only new code, enabling gradual cleanup.

### AD-4: Skip `revive` to reduce overlap
Revive's `unused-parameter` overlaps with `unparam`, and `unreachable-code` overlaps with `govet`. The dedicated linters provide cleaner coverage.

### AD-5: Separate workflow file
Keep lint in its own `lint.yml` rather than adding to the existing `release.yml`. This provides clear separation of concerns and independent failure reporting.

### AD-6: Action versions
Use `actions/checkout@v5`, `actions/setup-go@v6` with `go-version-file: go.mod`, and `golangci/golangci-lint-action@v9` with `version: v2.9`.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Existing code has many violations | Medium | `only-new-issues` flag avoids blocking existing code |
| SSA-based linters (unparam, unused) are slow | Low | Action caching + no timeout override |
| Action version pinning may drift | Low | Pin to `@v9` (major version) for auto-patches |
| False positives from gocritic | Low | Default stable checks only; exclusion presets for comments and std-error-handling |

## Testing Strategy

- **Manual validation**: Run `golangci-lint run ./...` locally to verify config parses correctly
- **CI validation**: Push branch and verify workflow triggers and runs successfully
- **Incremental check**: Verify `only-new-issues` correctly filters to new code only
- **Config validation**: `golangci-lint config verify` to check config syntax
