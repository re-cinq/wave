# Tasks

## Phase 1: Core ID Generation

- [ ] Task 1.1: Create `internal/pipeline/runid.go` with `GenerateRunID(name string, hashLength int) string` function using `crypto/rand` with timestamp fallback
- [ ] Task 1.2: Create `internal/pipeline/runid_test.go` with unit tests for format, length, uniqueness, defaults, and fallback behavior [P]

## Phase 2: Manifest Configuration

- [ ] Task 2.1: Add `PipelineIDHashLength int` field with `yaml:"pipeline_id_hash_length"` tag to `Runtime` struct in `internal/manifest/types.go`
- [ ] Task 2.2: Add `WithPipelineIDHashLength(n int)` executor option to `internal/pipeline/executor.go`
- [ ] Task 2.3: Add `PipelineName string` field to `PipelineStatus` in `internal/pipeline/executor.go`

## Phase 3: Context Updates

- [ ] Task 3.1: Add `PipelineName` field to `PipelineContext` in `internal/pipeline/context.go`; update `NewPipelineContext` to accept both pipeline ID and name; update `ToTemplateVars` and `ResolvePlaceholders` to expose both fields

## Phase 4: Executor Integration — Replace `execution.Pipeline.Metadata.Name` with `execution.Status.ID`

- [ ] Task 4.1: Modify `DefaultPipelineExecutor.Execute()` in `executor.go` to call `GenerateRunID` and use the result as `pipelineID` throughout the function; set both `Status.ID` and `Status.PipelineName`
- [ ] Task 4.2: Update `executeStep()` (executor.go:264) to read pipeline ID from `execution.Status.ID` instead of `execution.Pipeline.Metadata.Name`
- [ ] Task 4.3: Update `executeMatrixStep()` (executor.go:338) to use `execution.Status.ID` [P]
- [ ] Task 4.4: Update `runStepExecution()` (executor.go:363) to use `execution.Status.ID` [P]
- [ ] Task 4.5: Update `createStepWorkspace()` (executor.go:626) to use `execution.Status.ID` [P]
- [ ] Task 4.6: Update `injectArtifacts()` (executor.go:850) to use `execution.Status.ID` [P]
- [ ] Task 4.7: Update `checkRelayCompaction()` (executor.go:981) to use `execution.Status.ID` [P]
- [ ] Task 4.8: Modify `ExecuteWithValidation()` in `executor_enhanced.go` to call `GenerateRunID` and set both `Status.ID` and `Status.PipelineName`

## Phase 5: Matrix Executor Updates

- [ ] Task 5.1: Update `MatrixExecutor.Execute()` (matrix.go:47) to use `execution.Status.ID` instead of `execution.Pipeline.Metadata.Name` [P]
- [ ] Task 5.2: Update `MatrixExecutor.executeWorker()` (matrix.go:326) to use `execution.Status.ID` [P]
- [ ] Task 5.3: Update `MatrixExecutor.createWorkerWorkspace()` (matrix.go:406) to use `execution.Status.ID` [P]

## Phase 6: Resume Updates

- [ ] Task 6.1: Update `ResumeFromStep()` in `resume.go` to accept and propagate the suffixed runtime ID
- [ ] Task 6.2: Update `loadResumeState()` to use the runtime ID for workspace path resolution
- [ ] Task 6.3: Update `executeResumedPipeline()` event emissions to use runtime ID
- [ ] Task 6.4: Update `Resume()` method in `executor.go` (line 1139) to use the stored runtime ID

## Phase 7: Test Updates

- [ ] Task 7.1: Update `TestProgressEventFields` (executor_test.go:598) — change exact `PipelineID` assertions to prefix matching [P]
- [ ] Task 7.2: Update `TestGetStatus` (executor_test.go:664) — change exact `status.ID` assertions to prefix matching [P]
- [ ] Task 7.3: Update `TestMemoryCleanupAfterCompletion` and `TestMemoryCleanupAfterFailure` — use prefix matching for pipeline IDs [P]
- [ ] Task 7.4: Update `TestRegressionProductionIssues` — adjust assertions for suffixed IDs [P]
- [ ] Task 7.5: Update `contract_integration_test.go` if it uses exact pipeline ID assertions [P]
- [ ] Task 7.6: Write new test `TestExecutor_UniqueIDsPerRun` — verify two runs of the same pipeline produce different IDs
- [ ] Task 7.7: Write new test `TestExecutor_WorkspaceIsolation` — verify separate workspace directories per run

## Phase 8: Validation

- [ ] Task 8.1: Run `go test ./internal/pipeline/...` and fix any failures
- [ ] Task 8.2: Run `go test ./...` for full regression check
- [ ] Task 8.3: Run `go test -race ./...` for race condition validation
