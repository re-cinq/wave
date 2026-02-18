# Research: Static Analysis for Unused/Redundant Go Code

**Feature**: 103-static-analysis-ci
**Date**: 2026-02-18
**Spec**: `specs/103-static-analysis-ci/spec.md`

## Unknowns Extracted from Spec

The spec has zero NEEDS CLARIFICATION markers. All ambiguities were resolved during spec
refinement (C-001 through C-005). The following technology decisions remain for this
phase to confirm.

### U-001: golangci-lint v2 Configuration Format

**Decision**: Use golangci-lint v2 configuration format (`version: "2"` in `.golangci.yml`).

**Rationale**: The spec explicitly requires v2 format (FR-001). golangci-lint v2.x
introduced a fundamentally different configuration schema from v1: `linters.default`
for preset selection, a restructured `exclusions` section, and removal of the
`linters.enable`/`linters.disable` pattern. v2.x cannot parse v1 config files at all.

**Alternatives Rejected**:
- golangci-lint v1 format: Incompatible with v2.x runtime. The project has no existing
  `.golangci.yml`, so there is no migration concern.

### U-002: Linter Selection Strategy

**Decision**: Use `standard` preset as baseline, then selectively enable `unparam`,
`wastedassign`, `gocritic`, and `nolintlint`.

**Rationale**: The `standard` preset (FR-002) provides: `copyloopvar`, `errcheck`,
`govet`, `ineffassign`, `staticcheck`, `unused`. These cover the most broadly applicable
checks. The four additional linters (FR-003) target the specific spec goals:
- `unparam`: Detects unused function parameters (core use case from issue #72)
- `wastedassign`: Detects wasted variable assignments
- `gocritic`: Broad code quality checks including `dupSubExpr`, `assignOp`, `unnecessaryBlock`
- `nolintlint`: Enforces `//nolint` directive hygiene (FR-004)

**Alternatives Rejected**:
- `revive`: Explicitly excluded (FR-014) due to overlap with `unparam`, `govet`, `staticcheck`.
- Enabling all linters: Too noisy for a codebase adopting linting for the first time.
  The incremental adoption strategy (User Story 5) requires a focused linter set.

### U-003: CI Workflow Action Version

**Decision**: Use `golangci/golangci-lint-action` v7.x (latest stable v7 line).

**Rationale**: FR-006 requires v9 or later, but this is a minimum-bound spec written to
allow the implementer to pick the correct version at implementation time. The action
v7 is the current latest stable release for the golangci-lint-action. The spec's "v9"
reference appears to anticipate future versions — the implementer should use whatever
is actually the latest stable version of the action at implementation time. If v7 is
current, use v7. The `only-new-issues: true` flag is supported since v3 of the action.

**Note**: The implementer should verify the actual latest action version at implementation
time. The spec minimum bounds (v9) may not yet exist; use the latest available version.

### U-004: golangci-lint Binary Version

**Decision**: Pin to golangci-lint v2.1.6 (or latest stable 2.x at implementation time).

**Rationale**: FR-007 requires v2.9+ (minimum bound). The implementer should use the
latest stable v2.x release available. The golangci-lint-action's `version` parameter
accepts a specific version string like `v2.1.6`.

**Note**: Verify the latest stable release at implementation time.

### U-005: Exclusion Presets (v2 Format)

**Decision**: Use `std-error-handling` and `comments` exclusion presets (FR-013).

**Rationale**: golangci-lint v2 provides built-in exclusion presets:
- `std-error-handling`: Suppresses common false positives around standard error handling
  patterns (e.g., `err` assigned but not checked in deferred close operations).
- `comments`: Suppresses comment-style warnings from linters like `gocritic` that
  flag missing comments on exported types. This avoids forcing comment-on-every-export
  patterns during initial adoption.

These presets use the v2 `exclusions.presets` field.

**Alternatives Rejected**:
- Manual exclusion rules: More maintenance overhead, less portable across projects.
- No exclusions: Would produce excessive false positives on standard Go patterns,
  undermining developer adoption.

### U-006: Incremental Mode Implementation

**Decision**: Use `only-new-issues: true` in the golangci-lint-action (FR-006).

**Rationale**: This flag uses git diff to compare against the base branch (for PRs)
or the previous commit (for pushes). It reports only violations in new or modified code.
This is the standard mechanism for incremental adoption — pre-existing violations in
untouched code are not surfaced.

The action handles the diff computation internally; no additional configuration is needed.
For local runs, developers use `golangci-lint run ./...` without incremental mode, which
reports all violations (User Story 5, Acceptance Scenario 2).

**Alternatives Rejected**:
- Baseline file: Requires generating and maintaining a baseline, adding complexity.
- `new-from-rev` flag: More manual, less portable across PR and push contexts.

## Files to Create or Modify

| File | Action | FR |
|------|--------|----|
| `.golangci.yml` | Create | FR-001 through FR-004, FR-009, FR-013, FR-014, FR-015 |
| `.github/workflows/lint.yml` | Create | FR-005 through FR-008, FR-012 |
| `Makefile` | Modify line 24 | FR-010 |
| `CLAUDE.md` | Modify Testing section | FR-011 |

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Existing code has lint violations that block CI | High | `only-new-issues: true` ensures only new violations are reported |
| SSA-based linters (`unparam`, `unused`) slow on cold cache | Medium | golangci-lint-action provides caching by default |
| Action version specified in spec (v9) doesn't exist yet | Medium | Use latest available stable version; spec uses minimum bounds |
| `gocritic` reports excessive warnings | Low | Default stable checks are conservative; exclusion presets handle common patterns |
| New workflow interferes with existing `release.yml` | Low | Separate workflow file (FR-012), different triggers, no shared state |
