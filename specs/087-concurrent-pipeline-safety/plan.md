# Implementation Plan: Concurrent Pipeline Safety

**Branch**: `087-concurrent-pipeline-safety` | **Date**: 2026-02-13 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/087-concurrent-pipeline-safety/spec.md`

## Summary

Implement repository-scoped coordination for all git worktree operations across concurrent pipeline executions within a single Wave process. The core mechanism is a package-level `sync.Map` of channel-based semaphores keyed by canonical repository path, replacing the current per-instance `sync.Mutex` in `worktree.Manager`. This enables safe concurrent pipeline execution by serializing worktree create/remove/prune operations on the same repository while allowing parallel operations on different repositories. Additional changes include a typed `WorktreeRegistry` for cleanup tracking, context-aware lock acquisition with configurable timeout, and observability enhancements.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `sync` (stdlib), `context` (stdlib), `golang.org/x/sync/errgroup` (existing)
**Storage**: N/A (in-memory coordination only; stale recovery via `git worktree prune`)
**Testing**: `go test -race ./...`
**Target Platform**: Linux/macOS (single-process deployment)
**Project Type**: Single Go binary
**Performance Goals**: <100ms lock overhead per worktree operation under no contention (SC-006)
**Constraints**: Lock held only for duration of individual git operations, not pipeline step execution (FR-011)
**Scale/Scope**: 10+ concurrent pipelines, 5+ matrix workers per pipeline (SC-001, SC-008)

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new dependencies — uses stdlib `sync`, `context` |
| P2: Manifest as Source of Truth | PASS | No manifest changes required |
| P3: Persona-Scoped Execution | PASS | No persona changes |
| P4: Fresh Memory at Step Boundary | PASS | No context inheritance changes |
| P5: Navigator-First | PASS | No pipeline structure changes |
| P6: Contracts at Every Handover | PASS | No contract changes |
| P7: Relay via Summarizer | PASS | No relay changes |
| P8: Ephemeral Workspaces | PASS | Strengthens isolation guarantees for concurrent workspaces |
| P9: Credentials Never Touch Disk | PASS | No credential handling changes |
| P10: Observable Progress | PASS | Adds run ID to worktree operation events (FR-009) |
| P11: Bounded Recursion | PASS | No recursion changes |
| P12: Minimal Step State Machine | PASS | No new states added |
| P13: Test Ownership | PASS | All changes require `go test -race ./...` (FR-012) |

**Constitution re-check after Phase 1**: All principles remain satisfied. The implementation adds coordination infrastructure without altering pipeline semantics, workspace lifecycle, or state machine transitions.

## Project Structure

### Documentation (this feature)

```
specs/087-concurrent-pipeline-safety/
├── plan.md              # This file
├── research.md          # Phase 0: technology decisions and alternatives
├── data-model.md        # Phase 1: entity definitions and relationships
├── contracts/
│   ├── worktree-manager-api.md    # Manager API contract
│   └── worktree-registry-api.md   # Registry API contract
└── tasks.md             # Phase 2 output (/speckit.tasks — NOT created here)
```

### Source Code (repository root)

```
internal/
├── worktree/
│   ├── worktree.go          # MODIFY: Manager with repo-scoped locking, context support
│   ├── lock.go              # NEW: repoLock, repoLocks sync.Map, getRepoLock, canonicalPath
│   ├── registry.go          # NEW: WorktreeEntry, WorktreeRegistry
│   ├── worktree_test.go     # MODIFY: update for new API signatures
│   └── lock_test.go         # NEW: concurrent lock acquisition, timeout, cross-repo parallelism
├── pipeline/
│   ├── executor.go          # MODIFY: use WorktreeRegistry, pass context to worktree ops
│   ├── matrix.go            # MODIFY: pass context through matrix worker worktree creation
│   └── types.go             # MODIFY: add Worktrees field to PipelineExecution
└── event/
    └── emitter.go           # NO CHANGE (event structure already supports PipelineID)
```

**Structure Decision**: This feature modifies two existing packages (`internal/worktree`, `internal/pipeline`) and adds new files within them. No new packages are created. The `lock.go` and `registry.go` files are separated from `worktree.go` for single-responsibility clarity.

## Complexity Tracking

_No constitution violations — no entries needed._
