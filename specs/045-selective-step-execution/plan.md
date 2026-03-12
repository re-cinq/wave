# Implementation Plan: Selective Step Execution

## 1. Objective

Add `--steps` and `-x`/`--exclude` flags to `wave run` so users can selectively include or exclude specific pipeline steps, reducing cost and iteration time during development. The flags must integrate cleanly with the existing `--from-step` and `--dry-run` flags.

## 2. Approach

The implementation follows a layered architecture that separates concerns:

1. **CLI Layer** (`cmd/wave/commands/run.go`): Add flag definitions and mutual-exclusivity validation
2. **Step Filter** (`internal/pipeline/stepfilter.go`): New pure-function module for step filtering logic — takes a step list and filter config, returns the filtered set with validation
3. **Executor Integration** (`internal/pipeline/executor.go`): Apply the step filter to the topologically-sorted step list before the execution loop
4. **Dry-Run Enhancement** (`cmd/wave/commands/run.go`): Update `performDryRun` to show skip/include status and artifact availability warnings
5. **Resume Integration** (`internal/pipeline/resume.go`): Support `-x` combined with `--from-step`

The step filter is a standalone module (not embedded in the executor) so it can be tested in isolation and reused by both the main execution path and the resume path.

## 3. File Mapping

| File | Action | Description |
|------|--------|-------------|
| `cmd/wave/commands/run.go` | modify | Add `--steps` and `-x`/`--exclude` flags to `RunOptions` and `NewRunCmd`; add flag combination validation; pass filter config to executor; update `performDryRun` |
| `internal/pipeline/stepfilter.go` | create | New module: `StepFilter` struct with `Apply()` method, `ValidateStepNames()`, `FilterConfig` type |
| `internal/pipeline/stepfilter_test.go` | create | Table-driven tests for all filter combinations, edge cases, error conditions |
| `internal/pipeline/executor.go` | modify | Accept `StepFilterConfig` via new `WithStepFilter` option; apply filter after topological sort in `Execute()` |
| `internal/pipeline/resume.go` | modify | Apply exclude filter in `executeResumedPipeline()` after topological sort |
| `cmd/wave/commands/run_test.go` | modify | Add flag existence tests for `--steps`, `-x`/`--exclude`; add validation tests for flag combinations |

## 4. Architecture Decisions

### 4.1 Step Filter as Pure Function Module

The step filter is a standalone `stepfilter.go` file with pure functions rather than methods on the executor. This allows:
- Isolated unit testing without executor dependencies
- Reuse in both `Execute()` and `ResumeFromStep()` paths
- Clear separation of "what to run" from "how to run"

### 4.2 Filter Applied After Topological Sort

The filter is applied to the already-sorted step list rather than modifying the DAG. This preserves the DAG validation (cycle detection, dependency checking) on the full pipeline while only changing which steps are executed. Steps excluded by the filter are marked as `StateSkipped` in the execution state.

### 4.3 Comma-Separated String Flag (not StringSlice)

Use `StringVar` with comma parsing rather than Cobra's `StringSliceVar`. This matches the UX described in the issue (`--steps clarify,plan`) and avoids Cobra's StringSlice quoting issues. The parsing splits on commas and trims whitespace.

### 4.4 Artifact Dependency Validation

When steps are filtered out, the filter validates that remaining steps don't depend on skipped steps for artifacts, unless those artifacts already exist on disk from prior runs. This catches "impossible" filter combinations early with a clear error message.

### 4.5 ExecutorOption Pattern

The filter config is passed via the existing `ExecutorOption` pattern (`WithStepFilter(config)`) to maintain API consistency. The executor stores the filter config and applies it at the right point in execution.

## 5. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Breaking `--from-step` behavior | Low | High | Existing tests cover `--from-step`; new tests verify combination; filter is applied separately from resume logic |
| Artifact injection failures for skipped steps | Medium | Medium | Validate artifact availability during filter application; clear error messages listing missing artifacts |
| Flag parsing edge cases (empty strings, trailing commas) | Low | Low | Defensive parsing with whitespace trimming and empty-string filtering |
| Concurrent step batch execution with filtered steps | Low | Medium | Filter is applied before batch formation; `findReadySteps` already skips completed steps, just needs to also skip filtered-out steps |

## 6. Testing Strategy

### Unit Tests (`internal/pipeline/stepfilter_test.go`)
- `--steps` with valid step names → only those steps returned
- `--steps` with invalid step name → error with available steps listed
- `-x` with valid step names → those steps excluded
- `-x` with invalid step name → error with available steps listed
- `--steps` + `-x` → mutual exclusivity error
- `--from-step` + `-x` → steps after from-step minus excluded ones
- `--from-step` + `--steps` → error
- Empty `--steps` or `-x` → no filtering applied
- Single step → works
- All steps excluded → error (nothing to run)
- Dependency validation: step depends on excluded step without prior artifacts → error

### Integration Tests (`cmd/wave/commands/run_test.go`)
- Flag existence on `NewRunCmd`
- Flag combination validation at CLI level
- `performDryRun` with step filter shows skip/include status

### Existing Test Preservation
- All existing `--from-step` tests must continue to pass
- All existing `Execute()` tests must continue to pass (no filter = no change)
