# Tasks

## Phase 1: Schema Extension

- [X] Task 1.1: Add `Thread` and `Fidelity` fields to `Step` struct in `internal/pipeline/types.go`
- [X] Task 1.2: Add fidelity level constants (`FidelityFull`, `FidelityCompact`, `FidelitySummary`, `FidelityFresh`) in `internal/pipeline/types.go`
- [X] Task 1.3: Add `ThreadTranscripts` field to `PipelineExecution` struct in `internal/pipeline/executor.go`

## Phase 2: Core Implementation

- [X] Task 2.1: Create `internal/pipeline/thread.go` — `ThreadManager` with `AppendTranscript()`, `GetTranscript()`, `FormatPreamble()` methods
- [X] Task 2.2: Implement fidelity-based formatting in `ThreadManager`: `full` returns raw transcript, `compact` returns structured summary with step goals/outcomes, `fresh` returns empty string
- [X] Task 2.3: Implement transcript size cap with oldest-first truncation (default 100K chars)
- [X] Task 2.4: Implement `summary` fidelity using `relay.CompactionAdapter` interface for LLM-generated summary

## Phase 3: Executor Integration

- [X] Task 3.1: Initialize `ThreadTranscripts` map in `PipelineExecution` creation (`Execute()` method)
- [X] Task 3.2: In `runStepExecution()`, before adapter call: check `step.Thread`, retrieve transcript via `ThreadManager.GetTranscript()`, prepend to prompt
- [X] Task 3.3: In `runStepExecution()`, after adapter call: if `step.Thread` is set, capture stdout and append to thread transcript via `ThreadManager.AppendTranscript()`
- [X] Task 3.4: Thread transcript preamble format: `## Prior Conversation Context (thread: <name>)` section with step-attributed entries

## Phase 4: Validation

- [X] Task 4.1: Add validation rule: steps in the same thread group must have a dependency chain (cannot be concurrent) [P]
- [X] Task 4.2: Add validation warning: steps in the same thread group with different personas [P]
- [X] Task 4.3: Validate `fidelity` field values (must be one of: full, compact, summary, fresh) [P]

## Phase 5: Testing

- [X] Task 5.1: Create `internal/pipeline/thread_test.go` — unit tests for ThreadManager (append, retrieve, fidelity formatting, size cap, isolation)
- [X] Task 5.2: Add thread field coverage to `dryrun_test.go` and `template_test.go` [P] — covered in dag_test.go and executor_test.go
- [X] Task 5.3: Add executor integration test: threaded steps with mock adapter, verify transcript in prompt [P]
- [X] Task 5.4: Add validation tests: concurrent thread steps rejected, fidelity field validation [P]

## Phase 6: Polish

- [X] Task 6.1: Run `go test ./...` and `go test -race ./...` — fix any failures
- [X] Task 6.2: Run `golangci-lint run ./...` — fix any lint issues
