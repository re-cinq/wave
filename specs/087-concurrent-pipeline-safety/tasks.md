# Tasks: Concurrent Pipeline Safety

**Branch**: `087-concurrent-pipeline-safety` | **Date**: 2026-02-13
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

## Phase 1: Setup

- [X] T001 P1 Setup: Verify existing tests pass before modifications `internal/worktree/`, `internal/pipeline/`

## Phase 2: Foundational — Repository-Scoped Locking (US2 prerequisite)

- [X] T002 P1 US2: Implement `repoLock` type with channel-based semaphore and `LockWithContext`/`Unlock` methods `internal/worktree/lock.go`
- [X] T003 P1 US2: Implement `repoLocks` package-level `sync.Map`, `getRepoLock()` and `canonicalPath()` helpers `internal/worktree/lock.go`
- [X] T004 P1 US2: Write unit tests for `repoLock` — concurrent acquisition, timeout, context cancellation, cross-repo parallelism `internal/worktree/lock_test.go`

## Phase 3: US1 & US2 — Safe Concurrent Execution via Manager Refactor

- [X] T005 P1 US2: Refactor `Manager` struct — remove per-instance `sync.Mutex`, add `lockTimeout` field, add `ManagerOption`/`WithLockTimeout` `internal/worktree/worktree.go`
- [X] T006 P1 US1/US2: Update `Manager.Create()` — accept `context.Context`, acquire repo-scoped lock via `getRepoLock`, run prune+create under lock `internal/worktree/worktree.go`
- [X] T007 P1 US1/US2: Update `Manager.Remove()` — accept `context.Context`, acquire repo-scoped lock via `getRepoLock`, run remove under lock `internal/worktree/worktree.go`
- [X] T008 P1 US1/US2: Update `NewManager()` — accept variadic `ManagerOption`, canonicalize `repoRoot`, set default lockTimeout (30s) `internal/worktree/worktree.go`
- [X] T009 P1 US1/US2: Update all existing worktree tests to pass `context.Context` to `Create`/`Remove` and use new `NewManager` signature `internal/worktree/worktree_test.go`
- [X] T010 P1 US1: Write concurrent cross-manager test — 10 `Manager` instances on same repo doing create/remove concurrently, verify zero git errors `internal/worktree/worktree_test.go`

## Phase 4: US3 — Resilient Worktree Cleanup

- [X] T011 [P] P1 US3: Implement `WorktreeEntry` and `WorktreeRegistry` types with `Register`, `Entries`, `Count` (thread-safe) `internal/worktree/registry.go`
- [X] T012 [P] P1 US3: Write unit tests for `WorktreeRegistry` — concurrent register/read, empty registry, entry isolation `internal/worktree/registry_test.go`
- [X] T013 P1 US3: Add `Worktrees *worktree.WorktreeRegistry` field to `PipelineExecution`, initialize in `Execute()` `internal/pipeline/executor.go`
- [X] T014 P1 US3: Update `createStepWorkspace()` — register worktree entries in `execution.Worktrees` instead of `__worktree_repo_root` convention `internal/pipeline/executor.go`
- [X] T015 P1 US3: Update `createStepWorkspace()` worktree path — pass `context.Context` to `mgr.Create()` `internal/pipeline/executor.go`
- [X] T016 P1 US3: Refactor `cleanupWorktrees()` — iterate `execution.Worktrees.Entries()` with typed `WorktreeEntry` instead of scanning `WorkspacePaths` for magic suffix, pass context to `mgr.Remove()` `internal/pipeline/executor.go`
- [X] T017 P1 US3: Ensure `cleanupWorktrees()` is called via defer-style pattern in `Execute()` for both success and failure paths `internal/pipeline/executor.go`

## Phase 5: US4 — Matrix Worker Safety

- [X] T018 P2 US4: Update matrix `createWorkerWorkspace()` — if using worktree workspace type, pass context and register in parent execution's `WorktreeRegistry` `internal/pipeline/matrix.go`
- [X] T019 P2 US4: Write test — 5 matrix workers creating worktrees concurrently on same repo all succeed `internal/pipeline/matrix_test.go`

## Phase 6: US5 — Workspace Path Uniqueness

- [X] T020 [P] P2 US5: Write test verifying two concurrent `GenerateRunID()` calls for same pipeline name produce different IDs `internal/pipeline/runid_test.go`
- [X] T021 [P] P2 US5: Write test verifying workspace paths for two concurrent runs of same pipeline definition are distinct `internal/pipeline/executor_test.go`

## Phase 7: US6 — Observability

- [X] T022 [P] P3 US6: Add pipeline run ID to worktree operation event emissions in `createStepWorkspace` and `cleanupWorktrees` `internal/pipeline/executor.go`
- [X] T023 [P] P3 US6: Write test verifying all worktree operation log events include pipeline run ID `internal/pipeline/executor_test.go`

## Phase 8: Polish & Cross-Cutting

- [X] T024 P1 Cross: Update `resume.go` — if `ResumeState` uses `WorkspacePaths` with `__worktree_repo_root` convention, update to use `WorktreeRegistry` `internal/pipeline/resume.go`
- [X] T025 P1 Cross: Run full test suite with `go test -race ./internal/worktree/... ./internal/pipeline/...` and fix any race conditions `internal/worktree/`, `internal/pipeline/`
- [X] T026 P1 Cross: Verify `go vet ./...` and `go build ./...` pass cleanly `internal/worktree/`, `internal/pipeline/`
