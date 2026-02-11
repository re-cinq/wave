You are implementing a feature according to the specification, plan, and task breakdown.

Feature context: {{ input }}

## IMPORTANT: Working Directory

Your current working directory is a Wave workspace, NOT the project root.
Before running any scripts or accessing project files, navigate to the project root:

```bash
cd "$(git rev-parse --show-toplevel)"
```

Run this FIRST before any other bash commands.

A status report from the specify step is available at `artifacts/spec_info`.
Read it to find the branch name, spec file, and feature directory.

## Instructions

Follow the `/speckit.implement` workflow:

1. Navigate to the project root (see above)
2. Read `artifacts/spec_info` and check out the feature branch
3. Run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
   to find FEATURE_DIR, load tasks.md, plan.md, and all available artifacts
4. Check checklists status — if any are incomplete, note them but proceed
5. Parse tasks.md and extract phase structure, dependencies, and execution order
6. Execute implementation phase-by-phase:

   **Setup first**: Initialize project structure, dependencies, configuration
   **Tests before code**: Write tests for contracts and entities (TDD approach)
   **Core development**: Implement models, services, CLI commands, endpoints
   **Integration**: Database connections, middleware, logging, external services
   **Polish**: Unit tests, performance optimization, documentation

7. For each completed task, mark it as `[X]` in tasks.md
8. Run `go test -race ./...` after each phase to catch regressions early
9. Final validation: verify all tasks complete, tests pass, spec requirements met

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
