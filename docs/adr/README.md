# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the Wave project.

ADRs document significant architectural choices along with their context, options
considered, and consequences. They provide a trail of decisions that helps
contributors understand why the system is shaped the way it is.

## Creating an ADR

There are two paths for creating an ADR. Both produce the same format in this
directory.

### Manual (recommended for simple decisions)

1. Copy `000-template.md` to `NNN-short-title.md`, where NNN is the next
   sequential number (zero-padded to three digits).
2. Fill in each section of the template.
3. Open a pull request for review.

Use this path for straightforward decisions, process changes, or when Wave is
not available.

### Pipeline (recommended for complex decisions)

Run the ADR pipeline:

```bash
wave run plan-adr "Description of the decision"
```

The pipeline explores the codebase, analyzes options, drafts the record, and
opens a pull request for human review.

Use this path for decisions requiring deep codebase exploration, multi-option
analysis, or decisions that affect multiple subsystems.

## Naming Convention

ADR files follow the pattern `NNN-short-title.md`:

- `NNN` — zero-padded sequential number (001, 002, ...)
- `short-title` — lowercase, hyphen-separated summary of the decision

The template is `000-template.md` and is not itself a decision record.

## Status Lifecycle

- **Proposed** — under discussion, not yet accepted
- **Accepted** — approved and in effect
- **Deprecated** — no longer relevant
- **Superseded** — replaced by a later ADR (link to the replacement)

## Index

Last verified against the codebase: 2026-04-26.

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [001](001-formalize-adr-process.md) | Formalize ADR Process | Accepted | 2026-03-07 |
| [002](002-extract-step-executor.md) | Extract StepExecutor from Pipeline Executor | Proposed | 2026-03-12 |
| [003](003-layered-architecture.md) | Layered Architecture Separation | Accepted (Phase 1) | 2026-03-13 |
| [004](004-multi-adapter-architecture.md) | Multi-Adapter Architecture | Accepted (Implemented) | 2026-03-27 |
| [005](005-graph-execution-model.md) | Graph Execution Model | Accepted | 2026-03-27 |
| [006](006-cost-infrastructure.md) | Cost Infrastructure — Token Split, Pricing Matrix, Budget Enforcement | Accepted (Partial) | 2026-03-28 |
| [007](007-consolidate-database-access-through-statestore-interface.md) | Consolidate Database Access Through StateStore Interface | Proposed | 2026-04-13 |
| [008](008-add-skip-when-step-guard.md) | Add `skip_when` Step Guard for Token-Saving Short-Circuits | Proposed | 2026-04-13 |
| [009](009-ontology-bounded-context.md) | Ontology as a Bounded-Context Package | Accepted | 2026-04-18 |
| [010](010-pipeline-io-protocol.md) | Pipeline I/O Protocol | Accepted (Phase 1) | 2026-04-20 |
| [011](011-wave-lego-protocol.md) | Wave Lego Protocol | Accepted (Phase 1) | 2026-04-21 |
| [012](012-unified-in-memory-caching-layer.md) | Unified In-Memory Caching Layer | Proposed | 2026-04-20 |
| [013](013-failure-taxonomy-circuit-breaker.md) | Failure Taxonomy and Circuit Breaker | Accepted | 2026-03-27 |
| [014](014-composition-graph-boundary.md) | Composition / Graph-Execution Boundary | Accepted | 2026-04-26 |
| [015](015-persona-agent-migration.md) | Persona-to-Agent Migration Path | Accepted | 2026-04-26 |

### History

- **2026-04-26 — Consolidation pass.** ADR-013 was renumbered from a duplicate ADR-004 (collided with Multi-Adapter). ADR-014 was promoted from `docs/adrs/007-composition-graph-boundary.md`. ADR-015 was promoted from `docs/decisions/adr-agent-migration.md`. ADR-012's internal header was corrected from `ADR-011` to `ADR-012`. Statuses for 001, 003, 004, 005, 006, 010, 011, 013 were flipped to Accepted (some Phase 1 / Partial) after auditing each ADR's claims against the current codebase.

### Status Conventions

- **Accepted** — fully implemented and verified live.
- **Accepted (Phase 1 / Partial)** — core decision is in effect; later phases or follow-up work are still tracked in the ADR body.
- **Accepted (Implemented)** — landed end-to-end with no outstanding work.
- **Proposed** — decision is documented but not yet implemented.
- **Deprecated / Superseded** — see body for replacement reference.
