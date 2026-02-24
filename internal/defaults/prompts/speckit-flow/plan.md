You are creating an implementation plan for a feature specification.

Feature context: {{ input }}

## Working Directory

You are running in an **isolated git worktree** shared with previous pipeline steps.
Your working directory IS the project root. The feature branch was created by a
previous step and is already checked out.

## Instructions

Follow the `/speckit.plan` workflow:

1. Find the feature directory and spec file path from the spec info artifact
2. Run `.specify/scripts/bash/setup-plan.sh --json` to get FEATURE_SPEC, IMPL_PLAN,
   SPECS_DIR, and BRANCH paths
3. Load the feature spec and `.specify/memory/constitution.md`
4. Follow the plan template phases:

   **Phase 0 — Outline & Research**:
   - Extract unknowns from the spec (NEEDS CLARIFICATION markers, tech decisions)
   - Research best practices for each technology choice
   - Consolidate findings into `research.md` with Decision/Rationale/Alternatives

   **Phase 1 — Design & Contracts**:
   - Extract entities from spec → write `data-model.md`
   - Generate API contracts from functional requirements → `/contracts/`
   - Run `.specify/scripts/bash/update-agent-context.sh claude`

5. Evaluate constitution compliance at each phase gate
6. Stop after Phase 1 — report branch, plan path, and generated artifacts

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT use WebSearch — all information is in the spec and codebase

## Output

Produce a JSON status report matching the injected output schema.
