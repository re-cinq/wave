# Implementation Plan: Dead Code Audit (28 Findings)

## Objective

Clean up 28 dead code findings in `internal/pipeline/` and related packages: remove 5 stale prototype test files and 2 orphaned fixtures, replace a stdlib reimplementation, consolidate 3 duplicate `copyFile` implementations and 7 duplicate `emit()` methods, deduplicate pipeline defaults logic, unexport ~15 internal-only symbols, remove redundant re-exported constants, and fix `go.mod` dep classifications.

## Approach

Work in 5 phases ordered by risk: safe removals first, then consolidation (which requires careful refactoring), then symbol visibility changes (mechanical but wide-reaching), then `go mod tidy`, and finally full validation. Each phase should be a separate commit for easy bisection.

## File Mapping

### Phase 1: Remove Dead Files (DC-001..008)

| File | Action |
|------|--------|
| `internal/pipeline/prototype_spec_test.go` | delete |
| `internal/pipeline/prototype_docs_test.go` | delete |
| `internal/pipeline/prototype_dummy_test.go` | delete |
| `internal/pipeline/prototype_implement_test.go` | delete |
| `internal/pipeline/prototype_e2e_test.go` | delete |
| `internal/pipeline/.wave/contracts/mock-analysis.schema.json` | delete |
| `internal/pipeline/.wave/contracts/mock-result.schema.json` | delete |

### Phase 2: Remove Stdlib Reimplementation (DC-006)

| File | Action |
|------|--------|
| `internal/pipeline/executor.go` | modify -- find custom indexOf, replace call site with `strings.Index`, delete function |

### Phase 3: Consolidate Duplicates (DC-009, DC-010, DC-027)

| File | Action |
|------|--------|
| `internal/fileutil/copy.go` | create -- shared `copyFile` utility |
| `internal/pipeline/subpipeline.go` | modify -- use shared `copyFile` |
| `internal/skill/skill.go` | modify -- use shared `copyFile` |
| `internal/workspace/workspace.go` | modify -- use shared `copyFile` |
| `internal/pipeline/meta.go` | modify -- extract defaults to shared function |
| `internal/pipeline/dag.go` | modify -- use shared defaults function |
| `internal/pipeline/emitter.go` | create -- embedded `emitterMixin` struct |
| `internal/pipeline/composition.go` | modify -- embed `emitterMixin` |
| `internal/pipeline/concurrency.go` | modify -- embed `emitterMixin` |
| `internal/pipeline/executor.go` | modify -- embed `emitterMixin` |
| `internal/pipeline/gate.go` | modify -- embed `emitterMixin` |
| `internal/pipeline/matrix.go` | modify -- embed `emitterMixin` |
| `internal/pipeline/meta.go` | modify -- embed `emitterMixin` |
| `internal/pipeline/sequence.go` | modify -- embed `emitterMixin` |

### Phase 4: Unexport Symbols & Remove Re-exports (DC-011..026)

| File | Action |
|------|--------|
| `internal/pipeline/adhoc.go` | modify -- `DefaultNavigatorPersona` -> `defaultNavigatorPersona` |
| `internal/pipeline/chatworkspace.go` | modify -- unexport `ChatModeAnalysis`, `ChatModeManipulate` + type |
| `internal/pipeline/types.go` | modify -- remove 7 re-exported state constants |
| `internal/pipeline/meta.go` | modify -- unexport 5 constants |
| `internal/pipeline/executor.go` | modify -- unexport 3 ExecutorOption funcs + error type |
| `internal/pipeline/dag.go` | modify -- unexport 3 functions |
| `internal/pipeline/errors.go` | modify -- unexport `GateAbortError` |
| `internal/pipeline/outcomes.go` | modify -- unexport `EmptyArrayError` |
| `internal/pipeline/matrix.go` | modify -- unexport `StackedBaseBranchFromContext` |
| `internal/pipeline/subpipeline.go` | modify -- unexport `SubPipelineDefaultTimeout` |
| `internal/pipeline/sequence.go` | modify -- unexport `ErrParallelStagePartialFailure` |

### Phase 5: Fix go.mod (DC-028)

| File | Action |
|------|--------|
| `go.mod` | modify -- `go mod tidy` |
| `go.sum` | modify -- updated by `go mod tidy` |

## Architecture Decisions

1. **Shared `copyFile` location**: Create `internal/fileutil/copy.go` as a small utility package rather than picking one of the 3 existing locations. This avoids circular imports and provides a neutral home. The subpipeline.go version handles directories recursively -- the shared version should support both file and directory copy.

2. **Emitter consolidation**: Use an embedded `emitterMixin` struct with a single `emit()` method rather than an interface, since all 7 implementations are identical nil-check wrappers. Embedding avoids changing method signatures or call sites.

3. **Pipeline defaults dedup**: Extract `applyPipelineDefaults(*Pipeline)` as a package-level function in dag.go (where the canonical `Unmarshal` lives) and call it from both `Unmarshal` and `parsePipelineFromBytes` in meta.go.

4. **Unexport strategy**: For each symbol, rename with lowercase first letter and update all references within `internal/pipeline/`. Since these are all internal-only, no external consumers need updating. Must also update test files that reference these symbols.

5. **State constant removal (DC-013)**: The 7 re-exported state constants in types.go should be removed entirely. Any internal references should be updated to use `state.StatePending` etc. directly. Must grep to find all usage sites.

## Risks

| Risk | Mitigation |
|------|------------|
| Removing prototype tests might leave orphaned helpers | Grep for `loadTestPrototypePipeline` and `findPipelineStep` after deletion |
| `copyFile` consolidation could miss edge cases between the 3 implementations | Compare all 3 implementations carefully -- subpipeline handles dirs, others don't |
| Unexporting symbols could break code outside `internal/pipeline/` | Grep each symbol across the entire codebase before renaming |
| `go mod tidy` could remove needed deps | Run `go test ./...` after to catch any missing deps |
| emit() embedding could conflict with existing struct fields | Check each executor struct for field name conflicts |

## Testing Strategy

1. `go test ./...` after each phase to catch regressions immediately
2. `go test -race ./...` full run after all changes
3. `golangci-lint run ./...` for static analysis
4. No new tests needed -- this is purely removing/consolidating dead code
5. Existing tests serve as regression guards for the consolidation work
