You are performing a cross-artifact consistency and quality analysis across the
specification, plan, and tasks before implementation begins.

Feature context: {{ input }}

## Working Directory

You are running in an **isolated git worktree** shared with previous pipeline steps.
Your working directory IS the project root. The feature branch was created by a
previous step and is already checked out.

A status report from the specify step is available at `.wave/artifacts/spec_info`.
Read it to find the branch name, spec file, and feature directory.

## Instructions

Follow the `/speckit.analyze` workflow:

1. Read `.wave/artifacts/spec_info` to find the feature directory and spec file path
2. Run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
   to find FEATURE_DIR and locate spec.md, plan.md, tasks.md
3. Load all three artifacts and build semantic models:
   - Requirements inventory from spec.md
   - User story/action inventory with acceptance criteria
   - Task coverage mapping from tasks.md
   - Constitution rule set from `.specify/memory/constitution.md`

4. Run detection passes (limit to 50 findings total):
   - **Duplication**: Near-duplicate requirements across artifacts
   - **Ambiguity**: Vague adjectives, unresolved placeholders
   - **Underspecification**: Requirements missing outcomes, tasks missing file paths
   - **Constitution alignment**: Conflicts with MUST principles
   - **Coverage gaps**: Requirements with no tasks, tasks with no requirements
   - **Inconsistency**: Terminology drift, data entity mismatches, ordering contradictions

5. Assign severity: CRITICAL / HIGH / MEDIUM / LOW
6. Produce a compact analysis report (do NOT modify files — read-only analysis)

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT use WebSearch — all information is in the spec artifacts
- This is a READ-ONLY analysis — do NOT modify any files

## Output

Write a JSON status report to .wave/output/analysis-report.json with:
```json
{
  "total_requirements": 8,
  "total_tasks": 15,
  "coverage_percent": 95,
  "issues": {"critical": 0, "high": 1, "medium": 2, "low": 1},
  "can_proceed": true,
  "feature_dir": "path to feature directory",
  "summary": "brief analysis summary"
}
```

IMPORTANT: If CRITICAL issues are found, document them clearly but do NOT block
the pipeline. The implement step will handle resolution.
