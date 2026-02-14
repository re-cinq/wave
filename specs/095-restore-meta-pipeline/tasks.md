# Tasks

## Phase 1: Core Bug Fix

- [X] Task 1.1: Fix `invokePhilosopherWithSchemas()` to use `ResultContent` instead of raw `Stdout`
  - File: `internal/pipeline/meta.go` (lines 277-291)
  - Replace the `buf`/`Read()` pattern with: use `result.ResultContent` first, fall back to `io.ReadAll(result.Stdout)` when `ResultContent` is empty
  - Ensure error handling is correct for the `io.ReadAll` fallback
  - Remove the 1MB buffer allocation

- [X] Task 1.2: Update `mockMetaRunner` in tests to set `ResultContent`
  - File: `internal/pipeline/meta_test.go`
  - Update `mockMetaRunner.Run()` to set `ResultContent` on the returned `AdapterResult` in addition to `Stdout`
  - This ensures tests exercise the primary code path (not just the fallback)

## Phase 2: Mock Adapter Enhancement

- [X] Task 2.1: Add `generateMetaPhilosopherOutput()` to mock adapter [P]
  - File: `internal/adapter/mock.go`
  - Add a new function that returns valid meta-pipeline output with `--- PIPELINE ---` and `--- SCHEMAS ---` sections
  - The generated pipeline must pass `ValidateGeneratedPipeline()`: first step = navigator, all steps fresh memory, all steps have handover contracts
  - Include at least 2 steps (navigator + implementer) with proper dependencies
  - Include at least 1 JSON schema definition

- [X] Task 2.2: Wire meta-philosopher detection in `generateRealisticOutput()` [P]
  - File: `internal/adapter/mock.go`
  - In `generateRealisticOutput()`, detect `meta-philosopher` in `cfg.WorkspacePath`
  - Route to `generateMetaPhilosopherOutput()` when detected
  - This follows the existing pattern for `github-issue-impl` detection

## Phase 3: Testing

- [X] Task 3.1: Add unit test for `ResultContent` preference in meta executor
  - File: `internal/pipeline/meta_test.go`
  - Test that when `ResultContent` is set, it is used instead of `Stdout`
  - Test that when `ResultContent` is empty, `Stdout` is read as fallback
  - Test that the extracted YAML is correctly parsed in both paths

- [X] Task 3.2: Add integration test for mock dry-run flow
  - File: `cmd/wave/commands/meta_test.go`
  - Test `runMeta("test task", MetaOptions{Mock: true, DryRun: true, ...})` succeeds
  - Verify it doesn't panic or return error with the mock adapter
  - May need to set up a temporary directory with manifest containing philosopher persona

- [X] Task 3.3: Run full test suite and fix any regressions
  - Run `go test ./internal/pipeline/... ./cmd/wave/commands/... ./internal/adapter/...`
  - Ensure all existing tests still pass
  - Fix any tests broken by the changes

## Phase 4: Validation

- [X] Task 4.1: Verify `extractPipelineAndSchemas()` works with mock output
  - Write a test or manually verify that the mock philosopher output is correctly parsed by `extractPipelineAndSchemas()`
  - Ensure both the pipeline YAML and schema definitions are extracted
  - Verify the extracted pipeline passes `ValidateGeneratedPipeline()`

- [X] Task 4.2: Run full project test suite
  - Run `go test -race ./...` to verify no regressions across the entire project
  - Verify with `go vet ./...` for static analysis
