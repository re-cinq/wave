# Concurrency Safety Requirements Quality: Concurrent Pipeline Safety

**Feature**: 087-concurrent-pipeline-safety
**Date**: 2026-02-13
**Focus**: Quality of concurrency-related requirements, lock semantics, and race condition coverage

## Lock Semantics

- [ ] CHK034 - Are the preconditions for calling `Unlock()` fully specified — specifically what happens if `Unlock` is called without a preceding successful `LockWithContext`? [Completeness]
- [ ] CHK035 - Is the lock acquisition order defined for scenarios where a single goroutine needs locks on multiple repositories? Are deadlock-free guarantees documented? [Completeness]
- [ ] CHK036 - Is the fairness model of the channel-based semaphore specified — do waiters acquire the lock in FIFO order, or is it non-deterministic? [Clarity]
- [ ] CHK037 - Are requirements defined for lock behavior when `context.Background()` is passed (effectively infinite timeout) versus a context with a deadline? [Clarity]
- [ ] CHK038 - Is it specified whether the 30-second default timeout (SC-007) creates a *new* context or applies only when the caller's context has no deadline? [Clarity]
- [ ] CHK039 - Are there requirements for monitoring or logging lock contention metrics (wait times, queue depth)? [Coverage]

## Race Condition Coverage

- [ ] CHK040 - Are race conditions between `Create()` and `Remove()` on the same worktree path addressed in the requirements? [Completeness]
- [ ] CHK041 - Is the race between `WorktreeRegistry.Register()` from a matrix worker and `cleanupWorktrees()` from the parent pipeline specified? [Completeness]
- [ ] CHK042 - Are requirements defined for the race between process shutdown and deferred cleanup execution? [Completeness]
- [ ] CHK043 - Is the behavior specified when `git worktree add` succeeds but the subsequent `WorktreeRegistry.Register()` is not reached (e.g., goroutine panic between the two calls)? [Completeness]
- [ ] CHK044 - Is there a requirement ensuring that `cleanupWorktrees` does not start until all matrix workers have either completed or registered their worktrees? [Coverage]

## Lifecycle & Ownership

- [ ] CHK045 - Are ownership semantics for worktrees clear — is a worktree owned by the step, the pipeline execution, or the manager? [Clarity]
- [ ] CHK046 - Is the lifecycle of the `repoLocks` sync.Map entries specified — are locks ever removed from the map, or do they accumulate? [Completeness]
- [ ] CHK047 - Is it defined who is responsible for calling cleanup when a pipeline is resumed from persisted state (resume.go)? Does the resumed execution inherit worktree ownership? [Completeness]
- [ ] CHK048 - Are requirements specified for worktree lifecycle when a pipeline step is retried after failure? Is the old worktree cleaned up before creating a new one? [Completeness]

## Failure Mode Specifications

- [ ] CHK049 - Is the behavior defined when a lock holder's goroutine is leaked (neither completes nor panics) — does the timeout mechanism apply to waiters only, or is there proactive detection? [Completeness]
- [ ] CHK050 - Are partial failure states enumerated — e.g., lock acquired + prune succeeded + worktree add failed? Is the lock release guaranteed in all partial failure paths? [Completeness]
- [ ] CHK051 - Is it specified whether `Remove()` should retry on transient errors (e.g., "device busy") or fail immediately? [Clarity]
- [ ] CHK052 - Are requirements defined for the case where `git worktree prune` removes a worktree that another pipeline's `WorktreeRegistry` still references? [Completeness]
- [ ] CHK053 - Is the error propagation path defined when lock timeout occurs during cleanup — should the remaining worktrees still be cleaned up, or should cleanup abort? [Completeness]
