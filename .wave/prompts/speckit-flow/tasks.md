You are generating an actionable, dependency-ordered task breakdown for implementation.

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

Follow the `/speckit.tasks` workflow:

1. Set up the repo root reference (see above)
2. Read `artifacts/spec_info` and create a worktree for the feature branch:
   ```bash
   git -C "$REPO_ROOT" worktree add "$PWD/repo" <BRANCH_NAME>
   cd repo
   ```
3. Run `.specify/scripts/bash/check-prerequisites.sh --json` to get FEATURE_DIR
   and AVAILABLE_DOCS
4. Load from FEATURE_DIR:
   - **Required**: plan.md (tech stack, structure), spec.md (user stories, priorities)
   - **Optional**: data-model.md, contracts/, research.md, quickstart.md
5. Execute task generation:
   - Extract user stories with priorities (P1, P2, P3) from spec.md
   - Map entities and endpoints to user stories
   - Generate tasks organized by user story

6. Write `tasks.md` following the strict checklist format:
   ```
   - [ ] [TaskID] [P?] [Story?] Description with file path
   ```

7. Organize into phases:
   - Phase 1: Setup (project initialization)
   - Phase 2: Foundational (blocking prerequisites)
   - Phase 3+: One phase per user story (priority order)
   - Final: Polish & cross-cutting concerns

8. Commit task breakdown:
   ```bash
   git add specs/
   git commit -m "docs: add task breakdown"
   ```

9. Clean up worktree:
   ```bash
   cd "$OLDPWD"
   git -C "$REPO_ROOT" worktree remove "$PWD/repo"
   ```

## CONSTRAINTS

- **Maximum 20 tasks total** — scope aggressively. A single LLM implement step must complete all tasks within a 10-minute budget. If the feature requires more than 20 tasks, split into coarser units (e.g., "implement all handlers" instead of one task per handler). Prefer fewer, larger tasks over many granular ones.
- Do NOT spawn Task subagents — work directly in the main context
- Do NOT use WebSearch — all information is in the spec artifacts
- Keep the scope tight: generate tasks from existing artifacts only

## Quality Requirements

- Every task must have a unique ID (T001, T002...), description, and file path
- Mark parallelizable tasks with [P]
- Each user story phase must be independently testable
- Tasks must be specific enough for an LLM to complete without additional context

## Output

Write a JSON status report to output/tasks-status.json with:
```json
{
  "total_tasks": 15,
  "tasks_per_story": {"US1": 5, "US2": 4, "US3": 3},
  "parallel_opportunities": 6,
  "feature_dir": "path to feature directory",
  "summary": "brief description of task breakdown"
}
```
