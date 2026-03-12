# Implementation Plan: Selective Step Execution

## 1. Objective

Add `--steps` and `-x`/`--exclude` flags to `wave run` so users can selectively include or exclude specific pipeline steps without running the full pipeline. This enables faster iteration during development by skipping expensive steps.

## 2. Approach

The implementation adds step filtering as a **pre-execution transformation** on the topologically-sorted step list. The filtering happens after DAG validation and topological sort but before the execution loop, ensuring that:

1. The full pipeline DAG is validated first (catches config errors early)
2. Filtering produces a subset of the sorted steps
3. Dependency validation runs on the filtered set to catch missing artifact issues
4. The executor's existing concurrent step batch mechanism works unchanged on the filtered set

This is a clean separation: CLI layer handles flag parsing and mutual exclusivity validation, while the executor layer handles step filtering and dependency checking.

## 3. File Mapping

### Modified Files

| File | Action | Purpose |
|------|--------|---------|
| `cmd/wave/commands/run.go` | modify | Add `--steps` and `-x`/`--exclude` flags to `RunOptions` and `NewRunCmd()`. Pass filter options to executor. Validate mutual exclusivity. |
| `internal/pipeline/executor.go` | modify | Add `StepFilter` field to executor options. Apply filter in `Execute()` after topological sort. Handle artifact availability for skipped steps. |
| `internal/pipeline/resume.go` | modify | Apply exclusion filter in `executeResumedPipeline()` when `-x` is combined with `--from-step`. |
| `cmd/wave/commands/run.go` (`performDryRun`) | modify | Enhance dry-run output to show skip/include status per step when filters are active. |

### New Files

| File | Action | Purpose |
|------|--------|---------|
| `internal/pipeline/filter.go` | create | `StepFilter` type with `FilterSteps()` method. Validation for step names, dependency checking for filtered sets. |
| `internal/pipeline/filter_test.go` | create | Unit tests for all filter combinations, edge cases, and error scenarios. |
| `cmd/wave/commands/run_filter_test.go` | create | Integration tests for CLI flag parsing, mutual exclusivity, and end-to-end execution with filters. |

## 4. Architecture Decisions

### Decision 1: Filter as pre-execution transformation (not DAG modification)

**Choice**: Filter the sorted step list, don't modify the pipeline DAG itself.

**Rationale**: The DAG should remain the source of truth. Modifying it would require re-validating and could mask dependency issues. Filtering the sorted output is simpler, reversible, and doesn't affect state management.

### Decision 2: Comma-separated StringSliceVar for step names

**Choice**: Use Cobra's `StringSliceVar` with comma separation.

**Rationale**: Matches the issue specification. Step names are short identifiers, so comma-separated lists are ergonomic. This also matches `--from-step` which takes a single step name.

### Decision 3: Dependency validation on filtered set

**Choice**: After filtering, validate that all remaining steps have their dependencies satisfied (either in the filtered set or with existing workspace artifacts).

**Rationale**: Blindly running a step without its dependency artifacts would produce confusing failures. Early validation with clear error messages (listing missing artifacts and which skipped step produced them) gives users actionable guidance.

### Decision 4: Reuse ResumeManager artifact loading for skipped-step artifacts

**Choice**: When a step is excluded but downstream steps need its artifacts, use the same workspace scanning logic from `ResumeManager.loadResumeState()`.

**Rationale**: The resume flow already handles loading artifacts from prior runs. Reusing this avoids duplicating complex workspace scanning logic.

### Decision 5: StepFilter as a standalone type in filter.go

**Choice**: Create a dedicated `StepFilter` type rather than adding methods directly to the executor.

**Rationale**: Single responsibility — the filter logic is testable in isolation. The executor stays focused on execution. The filter can be composed with the resume manager's subpipeline creation.

## 5. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Artifact dependency issues when steps are skipped | Medium | High | Validate filtered set dependencies before execution; clear error messages listing missing artifacts |
| Flag interaction bugs with `--from-step` | Medium | Medium | Comprehensive test matrix covering all flag combinations |
| Breaking existing `--from-step` behavior | Low | High | Existing tests preserved; new logic is additive (filter applied after resume subpipeline creation) |
| Concurrent step batch logic affected by filtering | Low | Medium | Filter happens before batch selection; `findReadySteps` operates on already-filtered list |

## 6. Testing Strategy

### Unit Tests (`internal/pipeline/filter_test.go`)

- **Include filter**: `--steps a,b` with pipeline `[a, b, c, d]` → executes `[a, b]`
- **Exclude filter**: `-x c,d` with pipeline `[a, b, c, d]` → executes `[a, b]`
- **Invalid step names**: `--steps nonexistent` → error listing available steps
- **Mutual exclusivity**: `--steps a -x b` → error
- **From-step + exclude**: `--from-step b -x d` with pipeline `[a, b, c, d]` → executes `[b, c]`
- **From-step + steps**: `--from-step b --steps c` → error
- **Dependency validation**: `--steps c` where c depends on b → error (missing artifacts from b)
- **Empty result**: `-x a,b,c,d` (all steps excluded) → error
- **Single step**: `--steps a` → executes only `[a]`
- **Artifact availability**: `--steps c` where c depends on b and b has workspace artifacts → succeeds

### Integration Tests (`cmd/wave/commands/run_filter_test.go`)

- Flag registration and parsing via Cobra
- Dry-run output with filters active
- End-to-end execution with mock adapter and step filters

### Existing Test Preservation

- All existing `TestRunFromStep*` tests must continue to pass unchanged
- `TestNewRunCmdFlags` extended to verify new flags exist
