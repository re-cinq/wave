You are creating an implementation plan for a feature specification.

Feature context: {{ input }}

## IMPORTANT: Workspace Isolation via Git Worktree

Your current working directory is a Wave workspace, NOT the project root.
Use `git worktree` to create an isolated checkout — this allows multiple pipeline runs
to work concurrently without conflicts.

```bash
REPO_ROOT="$(git rev-parse --show-toplevel)"
```

A status report from the specify step is available at `artifacts/spec_info`.
Read it to find the branch name, spec file, and feature directory.

## Instructions

Follow the `/speckit.plan` workflow:

1. Set up the repo root reference (see above)
2. Read `artifacts/spec_info` and create a worktree for the feature branch:
   ```bash
   git -C "$REPO_ROOT" worktree add "$PWD/repo" <BRANCH_NAME>
   cd repo
   ```
3. Run `.specify/scripts/bash/setup-plan.sh --json` to get FEATURE_SPEC, IMPL_PLAN,
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
7. Stop after Phase 1 — commit and report:
   ```bash
   git add specs/
   git commit -m "docs: add implementation plan and research"
   ```
8. Clean up worktree:
   ```bash
   cd "$OLDPWD"
   git -C "$REPO_ROOT" worktree remove "$PWD/repo"
   ```

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
