# Implementation Plan: Static Analysis for Unused/Redundant Go Code

**Branch**: `103-static-analysis-ci` | **Date**: 2026-02-18 | **Spec**: `specs/103-static-analysis-ci/spec.md`
**Input**: Feature specification from `/specs/103-static-analysis-ci/spec.md`

## Summary

Add golangci-lint v2 static analysis to the Wave CI pipeline and local development
workflow. The implementation creates a `.golangci.yml` configuration using the `standard`
linter preset plus `unparam`, `wastedassign`, `gocritic`, and `nolintlint`; adds a
GitHub Actions lint workflow with incremental mode (`only-new-issues: true`); updates the
Makefile `lint` target from `go vet` to `golangci-lint run`; and documents the new lint
commands in CLAUDE.md. No Go code changes are required — this is purely a
configuration-and-CI feature.

## Technical Context

**Language/Version**: Go 1.25.5 (from `go.mod`)
**Primary Dependencies**: golangci-lint v2.x (external binary, not a Go module dependency)
**Storage**: N/A
**Testing**: `go test -race ./...` (existing), `golangci-lint run ./...` (new lint target)
**Target Platform**: GitHub Actions (ubuntu-latest), local development (any OS with golangci-lint)
**Project Type**: Single Go project
**Performance Goals**: CI lint step completes within action default timeout; caching mitigates cold-cache SSA analysis
**Constraints**: No new Go module dependencies; golangci-lint is an external tool prerequisite
**Scale/Scope**: 4 files created/modified; ~50 lines of configuration total

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | golangci-lint is an external dev/CI tool, not a Wave runtime dependency |
| P2: Manifest as Single Source of Truth | PASS | No changes to `wave.yaml` or Wave config system |
| P3: Persona-Scoped Execution | PASS | No persona changes |
| P4: Fresh Memory at Step Boundary | PASS | No pipeline changes |
| P5: Navigator-First Architecture | PASS | No pipeline changes |
| P6: Contracts at Every Handover | PASS | No pipeline changes |
| P7: Relay via Dedicated Summarizer | PASS | No relay changes |
| P8: Ephemeral Workspaces | PASS | No workspace changes |
| P9: Credentials Never Touch Disk | PASS | No credentials involved |
| P10: Observable Progress | PASS | GitHub Actions provides CI observability natively |
| P11: Bounded Recursion | PASS | No recursion involved |
| P12: Minimal Step State Machine | PASS | No state machine changes |
| P13: Test Ownership | PASS | `only-new-issues: true` prevents existing code from blocking CI; `make lint` behavioral change is documented |

**Result**: All 13 principles pass. No violations to justify.

## Project Structure

### Documentation (this feature)

```
specs/103-static-analysis-ci/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0: technology decisions and rationale
├── data-model.md        # Phase 1: entity descriptions and relationships
├── contracts/           # Phase 1: validation contracts
│   ├── golangci-config.yaml   # .golangci.yml structure validation
│   ├── lint-workflow.yaml     # lint.yml workflow validation
│   ├── makefile-lint.yaml     # Makefile target validation
│   └── claude-docs.yaml       # CLAUDE.md documentation validation
└── checklists/          # Spec quality checklists
```

### Source Code (repository root)

```
.golangci.yml                    # CREATE: golangci-lint v2 configuration (FR-001–FR-004, FR-009, FR-013–FR-015)
.github/workflows/lint.yml       # CREATE: CI lint workflow (FR-005–FR-008, FR-012)
Makefile                         # MODIFY: lint target go vet → golangci-lint (FR-010)
CLAUDE.md                        # MODIFY: Testing + Code Style sections (FR-011)
```

**Structure Decision**: This feature adds configuration files at the repository root
and in `.github/workflows/`. No new Go packages, source directories, or test files
are needed. The existing project structure is unchanged.

## Complexity Tracking

_No constitution violations to justify._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| (none)    | —          | —                                    |
