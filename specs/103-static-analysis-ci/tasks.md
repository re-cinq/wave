# Tasks: Static Analysis for Unused/Redundant Go Code

**Feature**: 103-static-analysis-ci
**Generated**: 2026-02-18
**Spec**: `specs/103-static-analysis-ci/spec.md`
**Plan**: `specs/103-static-analysis-ci/plan.md`

## Phase 1: Setup — Project Initialization

_No setup tasks required. This feature adds configuration files only — no Go packages, dependencies, or scaffolding needed._

## Phase 2: Foundational — Blocking Prerequisites

These tasks must complete before any user story phase can be tested.

- [X] T001 P1 US1,US2,US3,US4,US5 Create `.golangci.yml` in repository root with golangci-lint v2 format (`version: "2"`), `linters.default: standard` preset, and enable `unparam`, `wastedassign`, `gocritic`, `nolintlint` in `linters.enable` — file: `.golangci.yml` (FR-001, FR-002, FR-003)
- [X] T002 P1 US4 Configure `nolintlint` settings in `.golangci.yml` with `require-explanation: true` and `require-specific: true` under `linters.settings.nolintlint` — file: `.golangci.yml` (FR-004)
- [X] T003 P1 US1,US2,US5 Configure exclusion presets in `.golangci.yml` with `std-error-handling` and `comments` under `exclusions.presets` — file: `.golangci.yml` (FR-013)

> **Note**: T001, T002, and T003 all modify the same file (`.golangci.yml`) and should be implemented as a single atomic write. They are listed separately for traceability to individual FRs and user stories.

## Phase 3: PR Lint Checks — US1 (P1) + US2 (P1)

US1 and US2 are combined because they share the same CI workflow file and differ only in trigger events (PR vs push). Both are P1.

- [X] T004 P1 US1,US2 Create `.github/workflows/lint.yml` with workflow name `Lint`, triggered on `pull_request` (branches: `[main]`) and `push` (branches: `[main]`), single job `lint` on `ubuntu-latest` — file: `.github/workflows/lint.yml` (FR-005, FR-012)
- [X] T005 P1 US1,US2 Add `actions/checkout@v4` step and `actions/setup-go@v5` step with `go-version-file: go.mod` to the lint job — file: `.github/workflows/lint.yml` (FR-008)
- [X] T006 P1 US1,US2,US5 Add `golangci/golangci-lint-action` step (latest stable version, currently v7) with `version` pinned to a specific v2.x release (e.g., `v2.1.6`) and `only-new-issues: true` — file: `.github/workflows/lint.yml` (FR-006, FR-007)

> **Note**: T004, T005, and T006 all modify the same file (`.github/workflows/lint.yml`) and should be implemented as a single atomic write.

## Phase 4: Local Linting — US3 (P2)

Depends on Phase 2 (`.golangci.yml` must exist for `golangci-lint run` to use the project config).

- [X] T007 [P] P2 US3 Update Makefile `lint` target from `go vet ./...` to `golangci-lint run ./...` — file: `Makefile` line 24 (FR-010)
- [X] T008 [P] P2 US3 Update CLAUDE.md Testing section to document `golangci-lint run ./...` as the lint command and `golangci-lint run --fix ./...` as the auto-fix command — file: `CLAUDE.md` (FR-011)
- [X] T009 [P] P2 US3 Update CLAUDE.md Code Style section to replace `go vet` reference with `golangci-lint` — file: `CLAUDE.md` (FR-011)

> **Parallel**: T007, T008, and T009 are independent files/sections and can be executed in parallel.

## Phase 5: Nolint Hygiene — US4 (P2)

Depends on Phase 2 (T002 specifically configures `nolintlint`). No additional tasks beyond T002 — the nolintlint configuration is the entire implementation for US4.

- [X] T010 P2 US4 Verify no existing `//nolint` directives in the codebase violate the new `require-specific` and `require-explanation` rules by scanning for bare `//nolint` patterns — validation only, file: `*.go` across repo (SC-005)

## Phase 6: Incremental Adoption — US5 (P3)

Depends on Phase 3 (T006 configures `only-new-issues: true`). No additional tasks beyond T006 — the incremental mode is the entire implementation for US5.

_All implementation for US5 is covered by T006 (`only-new-issues: true`). No additional tasks needed._

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T011 P1 Verify `.golangci.yml` is parseable by running `golangci-lint config verify` or equivalent validation command — validation task (SC-001)
- [X] T012 P1 Verify the lint workflow file (`.github/workflows/lint.yml`) does not interfere with existing `release.yml` by confirming no shared job names or conflicting triggers — validation task (SC-006)
- [X] T013 [P] P2 Verify `make lint` exits non-zero when violations are found by testing against a file with a known violation — validation task (SC-007)
- [X] T014 [P] P2 Verify CLAUDE.md contains both `golangci-lint run` and `golangci-lint run --fix` references — validation task (SC-008)

> **Parallel**: T013 and T014 are independent validation checks.

## Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| 1 | 0 | Setup (none needed) |
| 2 | 3 | Foundational: `.golangci.yml` configuration |
| 3 | 3 | US1+US2: CI lint workflow |
| 4 | 3 | US3: Local linting (Makefile + docs) |
| 5 | 1 | US4: Nolint hygiene verification |
| 6 | 0 | US5: Covered by Phase 3 |
| 7 | 4 | Validation and cross-cutting |
| **Total** | **14** | |

## Dependency Graph

```
Phase 2: .golangci.yml (T001-T003)
    ├──→ Phase 3: lint.yml (T004-T006)
    │       └──→ Phase 7: Workflow validation (T011, T012)
    ├──→ Phase 4: Makefile + CLAUDE.md (T007-T009) [parallel]
    │       └──→ Phase 7: Make/docs validation (T013, T014) [parallel]
    └──→ Phase 5: Nolint scan (T010)
```

## Task-to-FR Traceability

| FR | Tasks |
|----|-------|
| FR-001 | T001 |
| FR-002 | T001 |
| FR-003 | T001 |
| FR-004 | T002 |
| FR-005 | T004 |
| FR-006 | T006 |
| FR-007 | T006 |
| FR-008 | T005 |
| FR-009 | T001 (exclusions section available) |
| FR-010 | T007 |
| FR-011 | T008, T009 |
| FR-012 | T004 |
| FR-013 | T003 |
| FR-014 | T001 (revive not in enable list) |
| FR-015 | T001 (gocritic uses defaults) |
