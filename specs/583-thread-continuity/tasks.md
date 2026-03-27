# Tasks

## Phase 1: Schema & Types
- [ ] Task 1.1: Add fidelity constants to `internal/pipeline/types.go` — `FidelityFull`, `FidelityCompact`, `FidelitySummary`, `FidelityFresh`
- [ ] Task 1.2: Add `Thread` and `Fidelity` fields to the `Step` struct in `internal/pipeline/types.go`
- [ ] Task 1.3: Add validation for `Thread` and `Fidelity` fields in `internal/pipeline/validation.go` — reject unknown fidelity values, warn on fidelity without thread

## Phase 2: Core Implementation
- [ ] Task 2.1: Create `internal/pipeline/thread.go` with `ThreadManager` struct — `ThreadEntry` type, `NewThreadManager()`, `AppendTranscript()`, `GetTranscript()` [P]
- [ ] Task 2.2: Implement `formatFull()` — verbatim transcript with step attribution headers [P]
- [ ] Task 2.3: Implement `formatCompact()` — step ID + status + truncated content (first 500 chars) [P]
- [ ] Task 2.4: Implement `formatSummary()` — delegate to relay `CompactionAdapter.RunCompaction()` with fallback to compact [P]
- [ ] Task 2.5: Implement transcript size cap — trim oldest entries when total exceeds `maxTranscriptSize` (100k chars)

## Phase 3: Executor Integration
- [ ] Task 3.1: Add `ThreadTranscripts` map and `ThreadManager` field to `PipelineExecution` struct in `executor.go`
- [ ] Task 3.2: In `buildStepPrompt()`, call `ThreadManager.GetTranscript()` and prepend result to prompt when `step.Thread != ""`
- [ ] Task 3.3: After successful step execution in `runStepExecution()`, call `ThreadManager.AppendTranscript()` with step's `ResultContent`
- [ ] Task 3.4: Initialize `ThreadManager` in the `Execute()` method when creating `PipelineExecution`

## Phase 4: Testing
- [ ] Task 4.1: Write unit tests for `ThreadManager` — creation, append, get transcript for each fidelity level, cap enforcement (`internal/pipeline/thread_test.go`) [P]
- [ ] Task 4.2: Write validation tests for thread/fidelity field validation (`internal/pipeline/validation_test.go`) [P]
- [ ] Task 4.3: Write integration test — two-step thread sharing, thread isolation, no-thread=fresh behavior (`internal/pipeline/executor_test.go`)
- [ ] Task 4.4: Run `go test ./...` and fix any failures
- [ ] Task 4.5: Run `go test -race ./...` and fix any data race issues

## Phase 5: Polish
- [ ] Task 5.1: Add thread-related trace events in executor for observability (`thread_inject`, `thread_append`)
- [ ] Task 5.2: Run `golangci-lint run ./...` and fix any findings
- [ ] Task 5.3: Final validation — verify all acceptance criteria from spec are met
