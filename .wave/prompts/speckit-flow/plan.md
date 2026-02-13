You are creating an implementation plan for a feature specification.

Feature context: {{ input }}

## Working Directory

You are running in an **isolated git worktree** shared with previous pipeline steps.
Your working directory IS the project root. The feature branch was created by a
previous step and is already checked out.

A status report from the specify step is available at `artifacts/spec_info`.
Read it to find the branch name, spec file, and feature directory.

## Instructions

Follow the `/speckit.plan` workflow:

1. Read `artifacts/spec_info` to find the feature directory and spec file path
2. Run `.specify/scripts/bash/setup-plan.sh --json` to get FEATURE_SPEC, IMPL_PLAN,
   SPECS_DIR, and BRANCH paths
4. Load the feature spec and `.specify/memory/constitution.md`
5. Follow the plan template phases:

   **Phase 0 — Outline & Research**:
   - Extract unknowns from the spec (NEEDS CLARIFICATION markers, tech decisions)
   - Research best practices for each technology choice
   - Consolidate findings into `research.md` with Decision/Rationale/Alternatives

   **Phase 1 — Design & Contracts**:
   - Extract entities from spec → write `data-model.md`
   - Generate API contracts from functional requirements → `/contracts/`
   - Run `.specify/scripts/bash/update-agent-context.sh claude`

6. Evaluate constitution compliance at each phase gate
7. Stop after Phase 1 — report branch, plan path, and generated artifacts

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT use WebSearch — all information is in the spec and codebase

## Output

Write a JSON status report to output/plan-status.json with:
```json
{
  "plan_file": "path to plan.md",
  "research_file": "path to research.md",
  "data_model_file": "path to data-model.md",
  "feature_dir": "path to feature directory",
  "constitution_issues": [],
  "summary": "brief description of what was planned"
}
```
