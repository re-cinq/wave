# Tasks

## Phase 1: Remove Dead Files
- [X] Task 1.1: Delete 5 stale prototype test files (DC-001..005) [P]
- [X] Task 1.2: Delete 2 orphaned schema fixture files (DC-007, DC-008) [P]
- [X] Task 1.3: Grep for orphaned helpers (`loadTestPrototypePipeline`, `findPipelineStep`) and remove if unused
- [X] Task 1.4: Run `go test ./internal/pipeline/...` to verify no breakage

## Phase 2: Replace Stdlib Reimplementation
- [X] Task 2.1: Find custom indexOf function in executor.go, replace call site with `strings.Index`, delete function (DC-006)
- [X] Task 2.2: Run `go test ./internal/pipeline/...` to verify

## Phase 3: Consolidate Duplicates
- [X] Task 3.1: Create `internal/fileutil/copy.go` with shared `copyFile` supporting both files and directories (DC-009)
- [X] Task 3.2: Update `internal/pipeline/subpipeline.go` to use `fileutil.CopyFile` [P]
- [X] Task 3.3: Update `internal/skill/skill.go` to use `fileutil.CopyFile` [P]
- [X] Task 3.4: Update `internal/workspace/workspace.go` to use `fileutil.CopyFile` [P]
- [X] Task 3.5: Extract `applyPipelineDefaults(*Pipeline)` in dag.go and call from meta.go `parsePipelineFromBytes` (DC-010)
- [X] Task 3.6: Create `internal/pipeline/emitter.go` with `emitterMixin` struct (DC-027)
- [X] Task 3.7: Embed `emitterMixin` in 5 executor types with direct emitter field, remove individual `emit()` methods [P]
- [X] Task 3.8: Run `go test ./...` to verify all consolidation

## Phase 4: Unexport Symbols & Remove Re-exports
- [X] Task 4.1: Unexport 7 re-exported state constants in types.go (DC-013)
- [X] Task 4.2: Unexport `DefaultNavigatorPersona` in adhoc.go (DC-011) [P]
- [SKIP] Task 4.3: ChatMode symbols used externally in cmd/wave/commands/chat.go -- cannot unexport (DC-012)
- [X] Task 4.4: Unexport 5 constants in meta.go: `DefaultMaxDepth`, `DefaultMaxTotalSteps`, `DefaultMaxTotalTokens`, `PhilosopherPersona`, `NavigatorPersona` (DC-014, DC-015) [P]
- [X] Task 4.5: Unexport `WithSkillStore` and `WithHookRunner` in executor.go (DC-016, DC-017) [P]
- [X] Task 4.6: Unexport 3 functions in dag.go: `ValidatePipelineSkills`, `DetectSubPipelineCycles`, `IsGraphPipeline` (DC-018) [P]
- [X] Task 4.7: Unexport `ReQueueError` in executor.go (DC-019) [P]
- [X] Task 4.8: Unexport `GateAbortError` in errors.go (DC-020) [P]
- [X] Task 4.9: Unexport `EmptyArrayError` in outcomes.go (DC-021) [P]
- [X] Task 4.10: Unexport `StackedBaseBranchFromContext` in matrix.go (DC-022) [P]
- [X] Task 4.11: Unexport `SubPipelineDefaultTimeout` in subpipeline.go (DC-025) [P]
- [X] Task 4.12: Unexport `ErrParallelStagePartialFailure` in sequence.go (DC-026) [P]
- [X] Task 4.13: Run `go build ./...` to verify no external references broken

## Phase 5: Fix go.mod
- [X] Task 5.1: Run `go mod tidy` to fix indirect dep classifications (DC-028)

## Phase 6: Final Validation
- [X] Task 6.1: Run `go test ./...`
- [ ] Task 6.2: Run `go test -race ./...`
- [ ] Task 6.3: Run `golangci-lint run ./...`
