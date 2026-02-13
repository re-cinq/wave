You are implementing a feature according to the specification, plan, and task breakdown.

Feature context: {{ input }}

A status report from the specify step is available at `artifacts/spec_info`.
Read it to find the branch name, spec file, and feature directory.

## Instructions

Follow the `/speckit.implement` workflow:

1. Read `artifacts/spec_info`
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

6. For each completed task, mark it as `[X]` in tasks.md
7. Run `go test -race ./...` after each phase to catch regressions early
8. Final validation: verify all tasks complete, tests pass, spec requirements met

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
