# Requirements Quality Review: Concurrent Pipeline Safety

**Feature**: 087-concurrent-pipeline-safety
**Date**: 2026-02-13
**Artifacts Reviewed**: spec.md, plan.md, tasks.md, data-model.md, research.md, contracts/

## Completeness

- [ ] CHK001 - Are all six user stories traceable to at least one functional requirement (FR-001 through FR-012)? [Completeness]
- [ ] CHK002 - Does every functional requirement have at least one acceptance scenario that would verify it? [Completeness]
- [ ] CHK003 - Are error behaviors specified for all lock acquisition failure modes (timeout, context cancellation, panic recovery)? [Completeness]
- [ ] CHK004 - Is the behavior defined for when `canonicalPath()` itself fails (e.g., symlink resolution error, non-existent directory)? [Completeness]
- [ ] CHK005 - Are success criteria defined for the cleanup registry lifecycle (creation, population, iteration, garbage collection)? [Completeness]
- [ ] CHK006 - Is the expected behavior specified when `git worktree prune` fails during a `Create()` call? Should the create proceed or abort? [Completeness]
- [ ] CHK007 - Are requirements specified for what happens when the `repoLocks` sync.Map grows unboundedly (many distinct repositories over process lifetime)? [Completeness]
- [ ] CHK008 - Is the interaction between `resume.go` worktree tracking and the new `WorktreeRegistry` fully specified, or only noted as a task (T024)? [Completeness]
- [ ] CHK009 - Are requirements defined for graceful shutdown behavior (SIGTERM/SIGINT) with respect to in-progress lock acquisitions and worktree cleanup? [Completeness]
- [ ] CHK010 - Is there an acceptance scenario for the edge case where `os.MkdirAll` races between two concurrent workspace creations? [Completeness]

## Clarity

- [ ] CHK011 - Is the distinction between "lock timeout" (SC-007: 30s for mutex acquisition) and "context timeout" (caller-provided `context.Context`) clearly defined? Which takes precedence? [Clarity]
- [ ] CHK012 - Is the term "coordination mechanism" used consistently to mean the repository-scoped `sync.Map` + channel semaphore, or does it sometimes refer to broader concepts? [Clarity]
- [ ] CHK013 - Is it clear whether `Manager.lockTimeout` is used to create a derived context, or if it only applies when the caller's context has no deadline? [Clarity]
- [ ] CHK014 - Does the spec clearly define what "serialized" means for FR-002 — strict FIFO ordering, or simply mutual exclusion? [Clarity]
- [ ] CHK015 - Is the contract for `Unlock()` on a never-locked or double-unlocked `repoLock` clearly documented as a blocking call or a panic? [Clarity]
- [ ] CHK016 - Is it clear from the spec alone (without reading research.md) that the channel-based semaphore was chosen over `sync.Mutex`, and why the plan.md summary says "channel-based semaphore" while the spec entity definition says "`sync.Mutex`"? [Clarity]
- [ ] CHK017 - Are the terms "pipeline run ID", "run ID", and "pipeline ID" used consistently, or could they be confused with each other? [Clarity]

## Consistency

- [ ] CHK018 - Does the spec's Key Entities section ("`sync.Mutex` keyed by canonical repository root path") align with the research.md decision to use a channel-based semaphore instead of `sync.Mutex`? [Consistency]
- [ ] CHK019 - Does the plan.md summary ("channel-based semaphores") match the data-model.md implementation (`chan struct{}` semaphore)? [Consistency]
- [ ] CHK020 - Is the `WorktreeRegistry` package placement consistent — data-model.md says "`internal/worktree/`" while plan.md project structure also places `registry.go` in `internal/worktree/`, but data-model.md entity header says "`internal/pipeline/` or embedded concept"? [Consistency]
- [ ] CHK021 - Does the task list (T011-T012) placing `WorktreeRegistry` in `internal/worktree/registry.go` align with all references in plan.md and data-model.md? [Consistency]
- [ ] CHK022 - Is the `Manager.Create()` signature consistent across all artifacts — spec says `(ctx, path, branch)`, contract says `(ctx, worktreePath, branch)`, data-model says `(ctx, path, branch)`? [Consistency]
- [ ] CHK023 - Are the success criteria numbering (SC-001 through SC-008) and functional requirements (FR-001 through FR-012) referenced consistently across spec.md, plan.md, and tasks.md? [Consistency]
- [ ] CHK024 - Does the clarification CLR-001 decision (in-process `sync.Mutex`) contradict the research.md Unknown 2 decision (channel-based semaphore replacing `sync.Mutex`)? [Consistency]

## Coverage

- [ ] CHK025 - Are there acceptance scenarios covering the boundary between "lock held for git operation" and "lock released before step execution" (FR-011)? [Coverage]
- [ ] CHK026 - Is there a test requirement for verifying that locks for different repositories do NOT contend with each other (FR-003 negative test)? [Coverage]
- [ ] CHK027 - Are there requirements for testing the `canonicalPath` function with edge cases (symlinks, relative paths, non-existent paths, paths with `..`)? [Coverage]
- [ ] CHK028 - Is there a test requirement for verifying that `WorktreeRegistry.Entries()` returns a copy (not a reference to internal state) as claimed in the contract? [Coverage]
- [ ] CHK029 - Are there requirements for testing the interaction between matrix workers and pipeline-level cleanup (US4 Scenario 2 + US3)? [Coverage]
- [ ] CHK030 - Is there a performance test requirement for SC-006 (<100ms overhead), or is it only stated as a success criterion without a corresponding task? [Coverage]
- [ ] CHK031 - Are there test requirements for the `WithLockTimeout` option — does changing the timeout actually affect behavior? [Coverage]
- [ ] CHK032 - Is there a test requirement for concurrent `getRepoLock` calls returning the same lock instance (atomic upsert guarantee)? [Coverage]
- [ ] CHK033 - Are there acceptance scenarios for US6 (Observability) that specify the exact format or structure of log messages containing run IDs? [Coverage]
