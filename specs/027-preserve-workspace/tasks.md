# Tasks

## Phase 1: Core Flag Plumbing

- [X] Task 1.1: Add `preserveWorkspace bool` field to `DefaultPipelineExecutor` struct in `internal/pipeline/executor.go`
- [X] Task 1.2: Add `WithPreserveWorkspace(preserve bool) ExecutorOption` function in `internal/pipeline/executor.go`
- [X] Task 1.3: Gate the `os.RemoveAll(pipelineWsPath)` call in `Execute()` on `!e.preserveWorkspace` in `internal/pipeline/executor.go`
- [X] Task 1.4: Emit a `"warning"` event when `e.preserveWorkspace` is true, before the workspace setup block in `Execute()`

## Phase 2: CLI Integration

- [X] Task 2.1: Add `PreserveWorkspace bool` field to `RunOptions` struct in `cmd/wave/commands/run.go`
- [X] Task 2.2: Register `--preserve-workspace` Cobra bool flag with help text in `NewRunCmd()`
- [X] Task 2.3: Pass `pipeline.WithPreserveWorkspace(opts.PreserveWorkspace)` to the executor options in `runRun()`
- [X] Task 2.4: Add stderr warning in `runRun()` when `opts.PreserveWorkspace` is true (before executor creation)

## Phase 3: Testing

- [X] Task 3.1: Add test `TestExecute_PreserveWorkspace` — verify workspace directory contents survive when flag is set [P]
- [X] Task 3.2: Add test `TestExecute_CleanWorkspaceDefault` — verify workspace is cleaned when flag is not set [P]
- [X] Task 3.3: Add test `TestExecute_PreserveWorkspaceWarning` — verify warning event is emitted when flag is set [P]

## Phase 4: Validation

- [X] Task 4.1: Run `go build ./...` to verify compilation
- [X] Task 4.2: Run `go test ./...` to verify all tests pass
