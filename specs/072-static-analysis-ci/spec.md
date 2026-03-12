# feat(ci): add static analysis for unused/redundant Go code

**Issue**: [#72](https://github.com/re-cinq/wave/issues/72)
**Labels**: enhancement, ci, code-quality
**Author**: nextlevelshit
**Complexity**: medium

## Summary

Add lightweight static Go code analysis to identify unused, redundant, and dead code. This should run as part of GitHub Actions CI to catch code rot early.

## Motivation

As Wave grows, unused code accumulates — dead functions, unreferenced types, ineffectual assignments, unused parameters. A static analysis step in CI keeps the codebase lean and maintainable without manual auditing.

## Proposed Approach

Integrate **golangci-lint** (v2, current release) with a curated set of linters focused on unused/redundant code detection.

### Critical: v2 Format Required

The original issue references v1 syntax and the removed `deadcode` linter. Per research comment (2026-02-12), golangci-lint v2.9.0 requires v2-format configuration. The `deadcode` linter was fully removed in v2 along with 12 other deprecated linters. The v2 binary cannot parse v1 configuration files at all.

### v2 Configuration Approach

- Use `version: "2"` with `linters.default: standard` preset (includes: copyloopvar, errcheck, govet, ineffassign, staticcheck, unused)
- Additional linters: `unparam`, `wastedassign`, `gocritic`, `nolintlint`
- Use `nolintlint` with `require-explanation: true` and `require-specific: true`
- Use `only-new-issues` in CI for incremental adoption

### GitHub Actions Workflow

- Use `golangci/golangci-lint-action@v9` with `version: v2.9`
- Trigger on push to `main` and `pull_request`
- Separate `lint.yml` workflow

## Implementation Plan

1. Add `.golangci.yml` to repo root with v2-format linter config
2. Add GitHub Actions workflow (`.github/workflows/lint.yml`)
3. Run on PRs targeting `main` and on pushes to `main`
4. Use `only-new-issues` for incremental adoption (avoid fixing all existing violations at once)
5. Document in CLAUDE.md under testing section

## Acceptance Criteria

- [ ] `.golangci.yml` committed with v2-format linter selection
- [ ] GitHub Actions workflow runs golangci-lint on PRs and main pushes
- [ ] `only-new-issues` enabled for incremental adoption
- [ ] CI is green on main after merge
- [ ] CLAUDE.md updated with lint command (`golangci-lint run ./...`)
