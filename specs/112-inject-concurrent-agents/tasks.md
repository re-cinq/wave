# Tasks

## Phase 1: Data Model

- [X] Task 1.1: Add `MaxConcurrentAgents int` field to `Step` struct in `internal/pipeline/types.go` with YAML tag `max_concurrent_agents,omitempty`
- [X] Task 1.2: Add `MaxConcurrentAgents int` field to `AdapterRunConfig` struct in `internal/adapter/adapter.go`

## Phase 2: Core Implementation

- [X] Task 2.1: Add concurrency hint injection in `prepareWorkspace` in `internal/adapter/claude.go` — insert between contract compliance section and restriction section, only when `MaxConcurrentAgents > 1`, capping at 10
- [X] Task 2.2: Wire `step.MaxConcurrentAgents` into the `AdapterRunConfig` in `internal/pipeline/executor.go` `executeStep` method (around line 820-837 where cfg is built)

## Phase 3: Testing

- [X] Task 3.1: Add table-driven unit tests in `internal/adapter/claude_test.go` for concurrency hint in CLAUDE.md (0, 1, 3, 15 values) [P]
- [X] Task 3.2: Add integration test verifying `MaxConcurrentAgents` flows from step to `AdapterRunConfig` using `configCapturingAdapter` [P]
- [X] Task 3.3: Add YAML parsing test for `max_concurrent_agents` field on pipeline steps [P]

## Phase 4: Documentation & Schema

- [X] Task 4.1: Add `max_concurrent_agents` to `docs/reference/manifest-schema.md` in the step properties table [P]
- [X] Task 4.2: Add `max_concurrent_agents` to `.wave/schemas/wave-pipeline.schema.json` step definition [P]
- [X] Task 4.3: Run `go test ./...` to validate all changes compile and pass
