# refactor: write ADR for layered architecture separation

**Issue**: [#364](https://github.com/re-cinq/wave/issues/364)
**Author**: nextlevelshit
**State**: OPEN
**Labels**: none

## Summary

Write an Architecture Decision Record (ADR) documenting the plan to enforce layered architecture boundaries in Wave. This ADR should build on the architecture audit from #298 and complement the existing ADR-002 (Extract StepExecutor).

## Background

Wave's `internal/` directory contains 25 packages. While the presentation/backend separation is already **reasonably clean** (see analysis below), some areas have accumulated coupling during rapid prototyping (v0.15.0 → v0.84.1). The primary concern is within the pipeline executor, not between presentation and backend layers.

### Current Layer Separation (Audit)

**Presentation layer** (`display/`, `tui/`, `webui/`):
- `display/` imports only `event/`, `pathfmt/`, `deliverable/` — no backend imports
- `tui/` reads from `state/` and `pipeline/` as data sources (read-only) — correct Observer pattern
- `pipeline/` does NOT import `display/` or `tui/` — no reverse dependency

**Backend layer** (`pipeline/`, `adapter/`, `contract/`, `state/`, `workspace/`, `worktree/`):
- `adapter/claude.go` (855 lines) — clean subprocess execution, no display imports
- `pipeline/executor.go` (2,848 lines, 37 methods) — **primary god-object concern**, owns 11+ responsibilities including DAG traversal, workspace creation, artifact injection, adapter invocation, contract validation, state persistence, relay monitoring, error recovery, and resume logic
- Event system (`event/`) properly decouples presentation from backend

**Cross-cutting** (`manifest/`, `security/`, `audit/`, `defaults/`):
- These are appropriately shared across layers

### What ADR-002 Already Covers

ADR-002 (`docs/adr/002-extract-step-executor.md`, status: Proposed) addresses the `executor.go` god-object by extracting a `StepExecutor` component. This ADR should complement — not duplicate — that work.

## Proposed ADR Scope

The new ADR (likely ADR-003) should cover:

1. **Define formal layer boundaries** — which packages belong to presentation, domain/orchestration, infrastructure, and cross-cutting layers
2. **Establish dependency rules** — e.g., presentation may depend on domain but not vice versa; infrastructure implements domain interfaces
3. **Document current violations** — any packages that cross layer boundaries inappropriately
4. **Migration strategy** — concrete steps to enforce boundaries (Go build constraints, linting rules, or CI checks)
5. **Agent/LLM impact** — how clean layer separation helps personas operate within bounded contexts (fresh memory per step, artifact-based communication)

## Relationship to Other Issues

- **#298** — parent issue for architecture audit and layered architecture transition. This issue delivers one of #298's remaining tasks
- **ADR-002** — complements this by addressing the god-object within the pipeline package

## Acceptance Criteria

- [ ] ADR written in `docs/adr/003-layered-architecture.md` following the template in `docs/adr/000-template.md`
- [ ] ADR documents current package-to-layer mapping
- [ ] ADR defines dependency rules between layers
- [ ] ADR includes Options Considered with pros/cons
- [ ] ADR addresses how layer separation benefits multi-agent pipeline execution
- [ ] References ADR-002 and issue #298 for continuity
