# Implementation Plan: Selective Step Execution

## Objective

Add `--steps` and `-x`/`--exclude` flags to `wave run` so users can selectively include or exclude specific pipeline steps, reducing execution cost during development.

## Approach

The implementation introduces a **step filter** concept that sits between DAG topological sorting and the execution loop. The filter is applied after sorting but before execution, so dependency validation can catch issues early. The filter is a new type `StepFilter` in the `pipeline` package, keeping it testable independently from the CLI layer.

### Architecture Overview

```
CLI (run.go)          Pipeline (executor.go)         Filter (step_filter.go)
┌──────────┐          ┌───────────────────┐          ┌──────────────────┐
│ --steps  │──parse──▶│ ExecuteWithFilter  │──apply──▶│ StepFilter       │
│ -x       │          │                   │          │ .Include []string│
│ --from-  │          │ TopologicalSort   │◀─filtered│ .Exclude []string│
│   step   │          │ + filter          │          └──────────────────┘
└──────────┘          └───────────────────┘
```

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `cmd/wave/commands/run.go` | **modify** | Add `--steps` and `-x`/`--exclude` flags to `RunOptions`, wire validation, pass to executor |
| `internal/pipeline/step_filter.go` | **create** | New `StepFilter` type with `Apply()`, `Validate()`, and `ValidateCombinations()` methods |
| `internal/pipeline/step_filter_test.go` | **create** | Unit tests for all filter logic, combinations, and error cases |
| `internal/pipeline/executor.go` | **modify** | Add `WithStepFilter()` option, integrate filter into `Execute()` and `performDryRun()` flow |
| `internal/pipeline/resume.go` | **modify** | Support `-x` exclusion in `ResumeFromStep()` to skip steps during resume |
| `cmd/wave/commands/run.go` (dry-run) | **modify** | Enhance `performDryRun()` to show step skip/include status |
| `cmd/wave/commands/run_test.go` | **modify** | Add tests for flag parsing, mutual exclusivity, and combination validation |

## Architecture Decisions

### 1. StepFilter as a separate type (not inline in executor)

The filter logic is non-trivial (validation, error messages, combination rules) and benefits from isolated testing. A `StepFilter` struct with `Include` and `Exclude` fields keeps the executor clean.

### 2. Filter applied to topologically-sorted steps

Rather than modifying the DAG itself, we filter the sorted step list. This preserves the original pipeline definition and allows clear error messages when dependencies are missing. The executor's main loop already processes `[]*Step` — filtering this list is minimally invasive.

### 3. Comma-separated string flag (not StringSlice)

Use `StringVar` with manual comma-splitting rather than Cobra's `StringSliceVar`. This avoids Cobra's quirky multi-value behavior with `--steps a --steps b` and keeps the UX consistent: `--steps clarify,plan,tasks`.

### 4. Dependency validation on filtered set

When steps are filtered out, the system must check that remaining steps have their artifact dependencies satisfied either by:
- Another step in the filtered set, OR
- An existing workspace artifact from a prior run

This reuses `ResumeManager.loadResumeState()` logic for finding prior artifacts.

### 5. `--from-step` + `-x` combination

This is implemented by first applying `--from-step` (creating the resume subpipeline) and then applying the exclusion filter to the resulting step set. The two operations compose naturally.

## Risks

| Risk | Mitigation |
|------|------------|
| Users exclude a step that produces required artifacts | Validate dependencies at filter time; fail with clear error listing missing artifacts and which step produces them |
| Interaction with matrix/composition steps | Matrix steps are treated as atomic — you include/exclude the parent step ID, not individual matrix variants |
| Interaction with `--preserve-workspace` | No conflict — `--preserve-workspace` affects workspace cleanup, not step selection |
| Stale artifacts from prior runs used when steps are skipped | Same risk as `--from-step` — document and accept; the `--force` flag already exists for this |

## Testing Strategy

### Unit Tests (`internal/pipeline/step_filter_test.go`)

1. **Include filter**: verify only named steps are returned
2. **Exclude filter**: verify named steps are removed
3. **Mutual exclusivity**: verify error when both Include and Exclude are set
4. **Invalid step names**: verify error with available steps listed
5. **Dependency validation**: verify error when a required dependency is filtered out
6. **Empty result**: verify error when filter produces zero runnable steps
7. **All steps included**: verify no-op when filter matches all steps

### Integration Tests (`cmd/wave/commands/run_test.go`)

1. **Flag parsing**: verify `--steps` and `-x`/`--exclude` are parsed correctly
2. **Combination validation**: `--steps` + `-x` errors; `--from-step` + `--steps` errors; `--from-step` + `-x` works
3. **Dry-run output**: verify skip/include status in dry-run output

### Existing Test Preservation

- All existing `--from-step` tests must continue passing unchanged
- Existing `Execute()` tests must pass (filter defaults to nil = no filtering)
