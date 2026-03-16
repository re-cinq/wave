# Tasks: Stacked Worktrees for Dependent Matrix Child Pipelines

**Feature**: #220 | **Branch**: `220-stacked-matrix-worktrees` | **Date**: 2026-03-16

---

## Phase 1: Type Extensions (Setup)

- [X] T001 [P1] Add `Stacked bool` field to `MatrixStrategy` struct in `internal/pipeline/types.go:302`
  - Add `Stacked bool \`yaml:"stacked,omitempty"\`` after `InputTemplate` field
  - Backward compatible: omitting field defaults to `false`

- [X] T002 [P1] Add `OutputBranch string` field to `MatrixResult` struct in `internal/pipeline/matrix.go:21`
  - Add `OutputBranch string` field after `ItemID` field
  - Internal struct only — no YAML schema impact

- [X] T003 [P1] [P] Add `lastExecution` field and `LastExecution()` method to `DefaultPipelineExecutor` in `internal/pipeline/executor.go:52`
  - Add `lastExecution *PipelineExecution` field to `DefaultPipelineExecutor` struct
  - Add `LastExecution() *PipelineExecution` method that returns the field
  - Set `e.lastExecution = execution` in the `Execute()` method after creating the execution (near `executor.go:344`)

- [X] T004 [P1] [P] Add `TierContext` struct to `internal/pipeline/matrix.go`
  - Define `TierContext` with `OutputBranches map[string]string` (itemID → branch name)
  - Add `NewTierContext()` constructor
  - Add `SetBranch(itemID, branch string)` and `GetBranch(itemID string) (string, bool)` methods
  - Add `ResolveBranch(deps []string) ([]string, error)` that returns parent branches for a given dep list

---

## Phase 2: Output Branch Capture (FR-007)

- [X] T005 [P1] [US1] Capture output branch in `childPipelineWorker` in `internal/pipeline/matrix.go:932`
  - After `childExecutor.Execute()` returns successfully (line ~960), call `childExecutor.LastExecution()` to get the child's `PipelineExecution`
  - Iterate `WorktreePaths` map to find the output branch name
  - Set `result.OutputBranch` to the first worktree branch found
  - If no worktree branch exists, leave `OutputBranch` empty (graceful fallback per clarification Q5)

---

## Phase 3: Stacked Tier Execution — US1 Sequential Dependency Chain (FR-002, FR-003, FR-008, FR-010)

- [X] T006 [P1] [US1] Add stacked branch resolution to `tieredExecution` in `internal/pipeline/matrix.go:601`
  - When `strategy.Stacked && strategy.DependencyKey != ""`:
    - Create `TierContext` before the tier loop (after line 655)
    - After each tier completes, update `TierContext` with `result.OutputBranch` for each successful result
  - When `stacked: true` but `dependency_key` is empty, fall through unchanged (FR-008 — no-op)

- [X] T007 [P1] [US1] Inject resolved base branch into child pipeline worker in `internal/pipeline/matrix.go`
  - Before executing each item in a stacked tier, resolve the base branch from `TierContext`
  - For single-parent dependency: use `TierContext.GetBranch(depID)` directly
  - Wrap the worker function in a closure that modifies the child pipeline's first worktree step base branch
  - Pass resolved base through the existing `matrixWorkerFunc` signature by wrapping the worker

- [X] T008 [P1] [US1] Emit stacking-related progress events (FR-010) in `internal/pipeline/matrix.go`
  - When stacked mode resolves a base branch, emit event with state `matrix_stacked_branch_resolved` including the item ID and resolved branch name
  - When integration branch is created, emit event with state `matrix_integration_branch_created`
  - When stacking is active but `dependency_key` is empty, emit nothing (silent no-op per FR-008)

---

## Phase 4: Multi-Parent Integration Branches — US2 (FR-004, FR-005, FR-009)

- [X] T009 [P2] [US2] Add `createIntegrationBranch` method to `MatrixExecutor` in `internal/pipeline/matrix.go`
  - Signature: `createIntegrationBranch(repoRoot, pipelineID, itemID string, parentBranches []string) (string, error)`
  - Create branch from first parent branch using `git checkout -b integration/<pipelineID>/<itemID> <parentBranches[0]>`
  - Sequentially merge each additional parent: `git merge <parentBranch> --no-edit`
  - On merge conflict: abort merge, clean up branch, return error with conflicting branch names (FR-005)
  - On success: return the integration branch name

- [X] T010 [P2] [US2] Integrate multi-parent resolution into `tieredExecution` in `internal/pipeline/matrix.go`
  - In the stacked branch resolution (T006), when an item has multiple dependencies:
    - Collect all parent branches from `TierContext`
    - Call `createIntegrationBranch` to create the merged base
    - Use the integration branch name as the base for the child pipeline
  - Track all integration branches in a `cleanupBranches []string` slice

- [X] T011 [P2] [US2] Add `cleanupIntegrationBranches` method to `MatrixExecutor` in `internal/pipeline/matrix.go`
  - Signature: `cleanupIntegrationBranches(repoRoot string, branches []string)`
  - For each branch: `git branch -D <branch>` (local cleanup only)
  - Log warnings on cleanup failures but do not fail the pipeline
  - Call via `defer` in `tieredExecution` after integration branch tracking is initialized (FR-009)

---

## Phase 5: Backward Compatibility — US3 (FR-006)

- [X] T012 [P1] [US3] [P] Verify existing matrix tests pass without modification in `internal/pipeline/matrix_test.go`
  - Run `go test ./internal/pipeline/ -run TestMatrix` to confirm all existing tests pass
  - Existing tests do not set `Stacked: true`, so they exercise the `stacked: false` default path
  - No code changes expected — this is a validation task (SC-003)

---

## Phase 6: Testing

- [X] T013 [P1] [US1] [P] Add unit tests for `TierContext` in `internal/pipeline/matrix_test.go`
  - Test `NewTierContext`, `SetBranch`, `GetBranch`, `ResolveBranch`
  - Test empty context returns no branches
  - Test single and multiple branch tracking

- [X] T014 [P1] [US1] Add unit test for output branch capture in `childPipelineWorker` in `internal/pipeline/matrix_test.go`
  - Mock child executor with `WorktreePaths` populated
  - Verify `result.OutputBranch` is set after child pipeline completes
  - Test case where no worktree branch exists (empty `OutputBranch`)

- [X] T015 [P1] [US1] Add integration test for 2-tier linear stacked chain (SC-001) in `internal/pipeline/matrix_test.go`
  - Configure 2 items: A (tier 0) and B (tier 1, depends on A) with `Stacked: true`
  - Mock child pipeline for A returns branch `feature/a`
  - Verify B's child pipeline receives `feature/a` as base branch
  - Verify results contain both items with correct `OutputBranch` values

- [ ] T016 [P2] [US2] Add integration test for diamond dependency pattern (SC-002) in `internal/pipeline/matrix_test.go`
  - Configure 4 items: A (tier 0), B and C (tier 1, depend on A), D (tier 2, depends on B and C)
  - Mock child pipelines return distinct branches
  - Verify D receives an integration branch merging B's and C's branches
  - Verify integration branch is cleaned up after completion (SC-005)

- [ ] T017 [P2] [US2] Add test for merge conflict during integration branch creation (SC-004) in `internal/pipeline/matrix_test.go`
  - Configure diamond pattern where B and C produce conflicting changes
  - Verify D is marked as failed with error message naming the conflicting branches
  - Verify integration branch is cleaned up even on failure

- [X] T018 [P1] [US3] Add test for `stacked: true` without `dependency_key` (FR-008) in `internal/pipeline/matrix_test.go`
  - Configure matrix with `Stacked: true` but empty `DependencyKey`
  - Verify all items execute in parallel from the same base (unchanged behavior)
  - No error or warning emitted

- [X] T019 [P1] [US1] Add test for parent with no output branch (graceful fallback) in `internal/pipeline/matrix_test.go`
  - Configure stacked 2-tier chain where tier 0 completes but has no worktree branch
  - Verify tier 1 uses the pipeline's default base branch
  - Verify a warning event is emitted

- [X] T020 [P1] [US1] [P] Add test for partial failure in stacked tiers in `internal/pipeline/matrix_test.go`
  - Configure 3-tier chain: A (tier 0), B and C (tier 1), D depends only on B (tier 2)
  - C fails, B succeeds
  - Verify D still runs with B's output branch (unaffected by C's failure)

---

## Phase 7: Polish & Cross-Cutting

- [X] T021 [P] Run `go test -race ./internal/pipeline/...` to verify no race conditions
  - Stacked tier context is accessed sequentially between tiers (no cross-tier concurrent access)
  - But integration branch creation may involve filesystem operations — verify thread safety

- [ ] T022 [P] Run `golangci-lint run ./internal/pipeline/...` to verify no lint issues
  - Ensure all new exported types have doc comments
  - Ensure error returns are checked
