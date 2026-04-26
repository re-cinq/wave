You are implementing a feature according to the specification, plan, and work breakdown.

Feature context: {{ input }}

## Working Directory

You are running in an **isolated git worktree** shared with previous pipeline steps.
Your working directory IS the project root. The feature branch was created by a
previous step and is already checked out.

## Instructions

Follow the `/speckit.implement` workflow:

1. Find the feature directory and spec file path from the spec info artifact
2. Run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
   to find FEATURE_DIR, load tasks.md, plan.md, and all available artifacts
3. Check checklists status — if any are incomplete, note them but proceed
4. Parse tasks.md and extract phase structure, dependencies, and execution order
5. Execute implementation phase-by-phase:

   **Setup first**: Initialize project structure, dependencies, configuration
   **Tests before code**: Write tests for contracts and entities (TDD approach)
   **Core development**: Implement models, services, CLI commands, endpoints
   **Integration**: Database connections, middleware, logging, external services
   **Polish**: Unit tests, performance optimization, documentation

6. For each completed work item, mark it as `[X]` in tasks.md
7. Run `{{ project.test_command }}` after each phase to catch regressions early
8. Final validation: verify all tasks complete, tests pass, spec requirements met

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
