# Tasks

## Phase 1: State DB — Checkpoint Enrichment

- [X] Task 1.1: Add `CheckpointRecord` type to `internal/state/types.go`
- [X] Task 1.2: Add migration version 12 for `checkpoint` table in `internal/state/migration_definitions.go`
- [X] Task 1.3: Update `schema.sql` with checkpoint table definition
- [X] Task 1.4: Add checkpoint CRUD methods to `StateStore` interface (`SaveCheckpoint`, `GetCheckpoint`, `GetCheckpoints`, `DeleteCheckpointsAfterStep`)
- [X] Task 1.5: Implement checkpoint CRUD methods on `stateStore` struct
- [X] Task 1.6: Add `forked_from` column to `pipeline_run` (migration + schema + types)

## Phase 2: Checkpoint Recording in Executor

- [X] Task 2.1: Create `internal/pipeline/checkpoint.go` with `CheckpointRecorder` that builds cumulative artifact snapshots and captures workspace commit SHAs
- [X] Task 2.2: Integrate `CheckpointRecorder` into `DefaultPipelineExecutor.executeStep()` — call after step completion
- [X] Task 2.3: Write unit tests for checkpoint recording logic in `internal/pipeline/checkpoint_test.go`

## Phase 3: Fork Command

- [X] Task 3.1: Create `internal/pipeline/fork.go` with `ForkManager` — validates source run, loads checkpoint, copies artifacts, creates new workspace, delegates to resume [P]
- [X] Task 3.2: Create `cmd/wave/commands/fork.go` with `wave fork` CLI command (`--from-step`, `--list`, `--input`, `--model`, `--json` flags) [P]
- [X] Task 3.3: Register `NewForkCmd()` in `cmd/wave/main.go`
- [X] Task 3.4: Write unit tests for `ForkManager` in `internal/pipeline/fork_test.go`
- [X] Task 3.5: Write CLI tests for fork command in `cmd/wave/commands/fork_test.go`

## Phase 4: Rewind Command

- [X] Task 4.1: Create `cmd/wave/commands/rewind.go` with `wave rewind` CLI command (`--to-step`, `--confirm`, `--json` flags) [P]
- [X] Task 4.2: Implement rewind logic: delete state DB records for steps after rewind point, update run status [P]
- [X] Task 4.3: Register `NewRewindCmd()` in `cmd/wave/main.go`
- [X] Task 4.4: Write CLI tests for rewind command in `cmd/wave/commands/rewind_test.go`

## Phase 5: Testing & Validation

- [X] Task 5.1: Run `go test ./...` — fix any failures
- [X] Task 5.2: Run `go test -race ./...` — fix any race conditions
- [X] Task 5.3: Run `golangci-lint run ./...` — fix lint issues
- [X] Task 5.4: Verify existing `wave resume` behavior is not broken
