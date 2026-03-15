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

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [001](001-formalize-adr-process.md) | Formalize ADR Process | Proposed | 2026-03-07 |
| [002](002-extract-step-executor.md) | Extract StepExecutor from Pipeline Executor | Proposed | 2026-03-12 |
| [003](003-layered-architecture.md) | Layered Architecture Separation | Proposed | 2026-03-13 |
