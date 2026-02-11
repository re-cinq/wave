You are implementing a GitHub issue according to the plan and task breakdown.

Input: {{ input }}

## IMPORTANT: Working Directory

Your current working directory is a Wave workspace, NOT the project root.
Before running any commands, navigate to the project root:

```bash
cd "$(git rev-parse --show-toplevel)"
```

Run this FIRST before any other bash commands.

The issue assessment is available at `artifacts/issue_assessment`.
The implementation plan is available at `artifacts/plan`.

## Instructions

### Step 1: Load Context

1. Read `artifacts/issue_assessment` for the issue details and branch name
2. Read `artifacts/plan` for the task breakdown, file changes, and feature directory
3. Check out the feature branch:
   ```bash
   git checkout <BRANCH_NAME>
   ```

### Step 2: Read Plan Files

Navigate to the feature directory and read:
- `spec.md` — the full specification
- `plan.md` — the implementation plan
- `tasks.md` — the phased task breakdown

### Step 3: Execute Implementation

Follow the task breakdown phase by phase:

**Setup first**: Initialize project structure, dependencies, configuration

**Tests before code (TDD)**:
- Write tests that define expected behavior
- Run tests to confirm they fail for the right reason
- Implement the code to make tests pass

**Core development**: Implement the changes specified in the plan

**Integration**: Wire components together, update imports, middleware

**Polish**: Edge cases, error handling, documentation updates

### Step 4: Validate Between Phases

After each phase, run:
```bash
go test -race ./...
```

If tests fail, fix the issue before proceeding to the next phase.

### Step 5: Mark Completed Tasks

As you complete each task, mark it as `[X]` in `tasks.md`.

### Step 6: Final Validation

After all tasks are complete:
1. Run `go test -race ./...` one final time
2. Verify all tasks in `tasks.md` are marked complete
3. Review changes with `git diff` to ensure nothing was missed

## Agent Usage — USE UP TO 6 AGENTS

Maximize parallelism with up to 6 Task agents for independent work:
- Agents 1-2: Setup and foundational tasks (Phase 1-2)
- Agents 3-4: Core implementation tasks (parallelizable [P] tasks)
- Agent 5: Test writing and validation
- Agent 6: Integration and polish tasks

Coordinate agents to respect task dependencies:
- Sequential tasks (no [P] marker) must complete before dependents start
- Parallel tasks [P] affecting different files can run simultaneously
- Run test validation between phases

## Error Handling

- If a task fails, halt dependent tasks but continue independent ones
- Provide clear error context for debugging
- If tests fail, fix the issue before proceeding to the next phase
