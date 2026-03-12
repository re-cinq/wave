# Tasks

## Phase 1: Core Flag Plumbing

- [X] Task 1.1: Add `PreserveWorkspace bool` field to `RunOptions` struct in `cmd/wave/commands/run.go`
- [X] Task 1.2: Register `--preserve-workspace` flag via Cobra with help text: `"Preserve workspace from previous run (for debugging)"`
- [X] Task 1.3: Add `preserveWorkspace bool` field to `DefaultPipelineExecutor` struct in `internal/pipeline/executor.go`
- [X] Task 1.4: Add `WithPreserveWorkspace(bool) ExecutorOption` function following existing option pattern
- [X] Task 1.5: Wire flag from `runRun()` into executor options: `pipeline.WithPreserveWorkspace(opts.PreserveWorkspace)`

## Phase 2: Core Implementation

- [X] Task 2.1: Gate the `os.RemoveAll(pipelineWsPath)` call at executor.go:~318 on `!e.preserveWorkspace`
- [X] Task 2.2: Emit warning event when `preserveWorkspace` is true: `"--preserve-workspace active: stale workspace state may cause non-reproducible results"`
- [X] Task 2.3: Copy `preserveWorkspace` field in `NewChildExecutor()` for matrix strategy consistency

## Phase 3: Testing

- [X] Task 3.1: Add unit test verifying workspace content is preserved when `WithPreserveWorkspace(true)` is set
- [X] Task 3.2: Add unit test verifying workspace content is cleaned when `WithPreserveWorkspace(false)` (default behavior unchanged)
- [X] Task 3.3: Run `go test ./...` to confirm no regressions

## Phase 4: Polish

- [X] Task 4.1: Add `--preserve-workspace` to the `Example` section in the run command's help text
- [X] Task 4.2: Run `go vet ./...` for static analysis
