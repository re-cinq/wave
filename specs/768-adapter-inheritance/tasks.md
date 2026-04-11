# Tasks

## Phase 1: Core Fix

- [X] Task 1.1: Add `adapterOverride` propagation in `runNamedSubPipeline`
  - In `internal/pipeline/executor.go` around line 5086 (after the `modelOverride` block), add:
    ```go
    if e.adapterOverride != "" {
        childOpts = append(childOpts, WithAdapterOverride(e.adapterOverride))
    }
    ```

## Phase 2: Testing

- [X] Task 2.1: Add adapter override inheritance test in `subpipeline_test.go`
  - Test that child executor spawned by `runNamedSubPipeline` inherits parent's `adapterOverride`
  - Verify the existing resolution priority is maintained (step-level > CLI)

## Phase 3: Validation

- [X] Task 3.1: Run existing pipeline/executor tests to confirm no regressions
  - `go test ./internal/pipeline/... -run TestSubPipeline`
  - `go test ./internal/pipeline/... -run TestComposition`
- [X] Task 3.2: Run full test suite
  - `go test ./internal/pipeline/...`
