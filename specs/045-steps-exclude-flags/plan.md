# Implementation Plan: --steps and -x flags

## 1. Objective

Add `--steps` (include filter) and `-x`/`--exclude` (exclude filter) flags to `wave run` so users can selectively run or skip specific pipeline steps, reducing iteration cost during development.

## 2. Approach

### Step Filtering as a Pre-Execution Transform

The core idea is to introduce a **step filter** that transforms the topologically-sorted step list *before* the execution loop runs. This approach:

- Keeps the executor's main loop unchanged
- Centralizes all filtering logic in one place
- Makes filtering testable independently of execution

### Data Flow

```
CLI flags → RunOptions → StepFilter → filtered []*Step → Execute loop
```

1. **CLI layer** (`cmd/wave/commands/run.go`): Parse `--steps` and `-x`/`--exclude` as `[]string` flags. Validate mutual exclusivity with each other and with `--from-step` at the CLI layer.

2. **Filter type** (`internal/pipeline/stepfilter.go`): New `StepFilter` struct encapsulating include/exclude lists. Provides `Apply(steps []*Step, allSteps []Step) ([]*Step, error)` that returns the filtered step list and validates step names exist.

3. **Executor integration** (`internal/pipeline/executor.go`): After `TopologicalSort`, apply the filter before entering the execution loop. Pass filter via a new `ExecutorOption`.

4. **Dry-run integration** (`cmd/wave/commands/run.go`): Enhance `performDryRun` to show skip/include status per step when filters are active.

5. **Resume integration** (`internal/pipeline/resume.go`): Pass filter through to `executeResumedPipeline` so `-x` works with `--from-step`.

## 3. File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/stepfilter.go` | **create** | `StepFilter` type with `Apply()`, `ValidateStepNames()`, artifact dependency checking |
| `internal/pipeline/stepfilter_test.go` | **create** | Unit tests for all filter combinations, edge cases, error paths |
| `cmd/wave/commands/run.go` | **modify** | Add `--steps`, `-x`/`--exclude` flags to `RunOptions` and `NewRunCmd`; pass filter to executor; enhance `performDryRun` |
| `internal/pipeline/executor.go` | **modify** | Add `WithStepFilter` option; apply filter after topological sort in `Execute` |
| `internal/pipeline/resume.go` | **modify** | Apply filter in `executeResumedPipeline` for `--from-step` + `-x` support |
| `internal/pipeline/executor_test.go` | **modify** | Integration tests for filtered execution through the executor |

## 4. Architecture Decisions

### AD-1: Filter as a separate type (not inline logic)

The `StepFilter` lives in its own file to keep the executor clean and enable thorough unit testing without needing a full executor setup. The filter is stateless — it takes a step list and returns a filtered step list.

### AD-2: Filter applied after topological sort

Filtering happens *after* `TopologicalSort` produces the ordered step list but *before* the execution loop. This ensures:
- The original DAG is validated intact (catches config errors)
- The filtered list respects topological order
- Dependencies on filtered-out steps are detected early

### AD-3: Comma-separated string, not StringSlice

Cobra's `StringSliceVar` splits on commas automatically, so `--steps clarify,plan` and `--steps clarify --steps plan` both work. This matches the issue's design.

### AD-4: Artifact dependency validation

When a step is excluded but a later step depends on its artifacts, the filter checks whether workspace artifacts already exist on disk (reusing `ResumeManager`'s artifact resolution). If not, it fails with a clear error listing what's missing.

### AD-5: CLI-layer mutual exclusivity validation

`--steps` + `-x` and `--from-step` + `--steps` are rejected in `runRun()` before any executor setup, with clear error messages. This keeps error reporting fast and user-facing.

## 5. Risks

| Risk | Mitigation |
|------|-----------|
| Filter interacts badly with concurrent step execution | Filter only removes steps from the sorted list — the batch scheduler sees fewer steps but operates identically |
| Skipped steps break artifact injection for downstream steps | Explicit validation: if a required artifact is missing and the producing step is filtered out, fail early with a clear error |
| `--from-step` + `-x` has edge cases with workspace artifact resolution | Reuse proven `ResumeManager.loadResumeState()` for artifact path resolution |
| Breaking existing `--from-step` behavior | Comprehensive regression tests; filter is nil by default (no-op) |

## 6. Testing Strategy

### Unit Tests (`stepfilter_test.go`)
- Include filter: single step, multiple steps, all steps
- Exclude filter: single step, multiple steps
- Invalid step names → error with available step list
- Mutual exclusivity: `--steps` + `-x` → error
- `--from-step` + `--steps` → error
- `--from-step` + `-x` → correct filtering
- Empty filter → all steps pass through (no-op)
- Artifact dependency validation for excluded steps

### Integration Tests (`executor_test.go`)
- Execute pipeline with `--steps` filter, verify only those steps ran
- Execute pipeline with `-x` filter, verify excluded steps skipped
- Execute with filter + mock adapter, verify event sequence
- Resume with `-x`, verify combination works

### Dry-Run Tests
- Verify `performDryRun` output includes skip/include annotations when filters active
