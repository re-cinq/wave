# Audit Current Architecture and Plan Layered Architecture Transition

**Issue**: [#298](https://github.com/re-cinq/wave/issues/298)
**Labels**: documentation, enhancement
**Author**: nextlevelshit

## Summary

Introduce Architecture Decision Records (ADRs) as a standard practice for documenting significant architectural decisions in the Wave project. As the first application of this practice, conduct a full audit of the current architecture and create a transition plan toward a layered architecture.

## Background

ADRs provide a lightweight, versioned record of architectural decisions and their rationale. They help current and future contributors understand why the codebase is structured the way it is.

## Progress

ADR infrastructure has been established via PR #280 (`docs: ADR — formalize architectural decision records`), which introduced:

- `docs/adr/000-template.md` — standard ADR template
- `docs/adr/001-formalize-adr-process.md` — ADR-001 documenting the hybrid manual/pipeline approach
- `docs/adr/README.md` — guide for manual and pipeline creation paths

Additionally, ADR-002 (Extract StepExecutor) and ADR-003 (Layered Architecture Separation) already exist as proposed ADRs.

## Remaining Tasks

- [ ] Audit the current Wave architecture — document the existing package structure, dependency flow, and key design patterns
- [ ] Identify areas that would benefit from layered architecture principles (e.g., separating domain logic from infrastructure concerns)
- [ ] Write an ADR proposing the layered architecture transition with rationale, trade-offs, and migration strategy
- [ ] Define clear layer boundaries and dependency rules

## Acceptance Criteria

- ~~ADR template and directory structure are established~~ (done)
- ~~At least one ADR documents the decision to adopt ADRs~~ (done)
- Architecture audit document captures the current state
- A transition plan ADR outlines the path to layered architecture with concrete steps

## Current State Assessment

ADR-003 already proposes the layered architecture with four layers (Presentation, Domain/Orchestration, Infrastructure, Cross-cutting), dependency rules, and CI enforcement via depguard. However, a standalone architecture audit document that comprehensively captures the current state is still missing. The audit should be thorough enough to serve as the baseline reference for ADR-003's transition plan.

## References

- [Michael Nygard's ADR article](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
- [adr-tools](https://github.com/npryce/adr-tools)
- PR #280 — ADR infrastructure
- ADR-002 — Extract StepExecutor from Pipeline Executor
- ADR-003 — Layered Architecture Separation
