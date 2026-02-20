You are generating an actionable, dependency-ordered task breakdown for implementation.

Feature context: {{ input }}

## Working Directory

You are running in an **isolated git worktree** shared with previous pipeline steps.
Your working directory IS the project root. The feature branch was created by a
previous step and is already checked out.

A status report from the specify step is available at `.wave/artifacts/spec_info`.
Read it to find the branch name, spec file, and feature directory.

## Instructions

Follow the `/speckit.tasks` workflow:

1. Read `.wave/artifacts/spec_info` to find the feature directory and spec file path
2. Run `.specify/scripts/bash/check-prerequisites.sh --json` to get FEATURE_DIR
   and AVAILABLE_DOCS
3. Load from FEATURE_DIR:
   - **Required**: plan.md (tech stack, structure), spec.md (user stories, priorities)
   - **Optional**: data-model.md, contracts/, research.md, quickstart.md
4. Execute task generation:
   - Extract user stories with priorities (P1, P2, P3) from spec.md
   - Map entities and endpoints to user stories
   - Generate tasks organized by user story

5. Write `tasks.md` following the strict checklist format:
   ```
   - [ ] [TaskID] [P?] [Story?] Description with file path
   ```

6. Organize into phases:
   - Phase 1: Setup (project initialization)
   - Phase 2: Foundational (blocking prerequisites)
   - Phase 3+: One phase per user story (priority order)
   - Final: Polish & cross-cutting concerns

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT use WebSearch — all information is in the spec artifacts
- Keep the scope tight: generate tasks from existing artifacts only

## Quality Requirements

- Every task must have a unique ID (T001, T002...), description, and file path
- Mark parallelizable tasks with [P]
- Each user story phase must be independently testable
- Tasks must be specific enough for an LLM to complete without additional context

## Output

Write a JSON status report to .wave/output/tasks-status.json with:
```json
{
  "total_tasks": 15,
  "tasks_per_story": {"US1": 5, "US2": 4, "US3": 3},
  "parallel_opportunities": 6,
  "feature_dir": "path to feature directory",
  "summary": "brief description of task breakdown"
}
```
