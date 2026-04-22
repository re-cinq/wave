You are implementing an issue according to the plan and work breakdown.

Input: {{ input }}

## Working Directory

You are running in an **isolated git worktree** shared with previous pipeline steps.
Your working directory IS the project root. The feature branch was created by the
plan step and is already checked out. All git operations here are isolated from
the main working tree.

## Instructions

### Step 1: Load Context

1. Get the issue details and branch name from the issue assessment artifact
2. Get the work breakdown, file changes, and feature directory from the plan artifact

### Step 2: Read Plan Files

Navigate to the feature directory and read:
- `spec.md` — the full specification
- `plan.md` — the implementation plan
- `tasks.md` — the phased work breakdown

### Step 3: Execute Implementation

Follow the work breakdown phase by phase:

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
{{ project.test_command }}
```

If tests fail, fix the issue before proceeding to the next phase.

### Step 5: Mark Completed Items

As you complete each item, mark it as `[X]` in `tasks.md`.

### Step 6: Final Validation

After all items are complete:
1. Run `{{ project.test_command }}` one final time
2. Verify all items in `tasks.md` are marked complete
3. Stage and commit all changes — YOU MUST run the git reset to exclude Wave internals:
   ```bash
   git add -A
   git reset HEAD -- .wave/artifacts/ .wave/output/ .claude/ CLAUDE.md 2>/dev/null || true
   git diff --cached --name-only | head -20  # verify no .wave/artifacts, .wave/output, .claude, or CLAUDE.md
   git commit -m "feat: implement #<ISSUE_NUMBER> — <short description>"
   ```

   CRITICAL: Never use `Closes #N`, `Fixes #N`, or `Resolves #N` in commit messages — these auto-close issues on merge. Use the issue number without closing keywords as shown above.
   CRITICAL: Never commit `.claude/settings.json`, `CLAUDE.md`, `.wave/artifacts/`, or `.wave/output/`.
   These are Wave-managed files. The `specs/` directory IS allowed.

Commit changes to the worktree branch.

## Parallelism

Maximize parallelism by working on independent items in batches:
- Batch 1-2: Setup and foundational items (Phase 1-2)
- Batch 3-4: Core implementation items (parallelizable [P] items)
- Batch 5: Test writing and validation
- Batch 6: Integration and polish items

Respect inter-item dependencies:
- Sequential items (no [P] marker) must complete before dependents start
- Parallel items [P] affecting different files can be batched together
- Run test validation between phases

## Error Handling

- If a work item fails, halt dependent items but continue independent ones
- Provide clear error context for debugging
- If tests fail, fix the issue before proceeding to the next phase
