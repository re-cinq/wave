# Tasks

## Phase 1: Deliverable Tracker Tests (IMP-003)
- [X] Task 1.1: Create `internal/deliverable/tracker_test.go` with tests for NewTracker and SetPipelineID
- [X] Task 1.2: Add table-driven tests for Add method deduplication (same path+stepID skipped, different path allowed, different stepID allowed)
- [X] Task 1.3: Add tests for GetAll (returns sorted copy), GetByStep (filters correctly, returns empty for unknown step), Count
- [X] Task 1.4: Add tests for FormatSummary (empty tracker, populated tracker) and FormatByStep (grouping correctness)
- [X] Task 1.5: Add tests for GetLatestForStep (returns most recent, returns nil for empty)
- [X] Task 1.6: Add tests for AddWorkspaceFiles (discovers files by pattern, skips duplicates)
- [X] Task 1.7: Add tests for AddOutcomeWarning and OutcomeWarnings (append, retrieve copy)
- [X] Task 1.8: Add concurrent access test — multiple goroutines calling Add, GetAll, SetPipelineID simultaneously (must pass with `-race`)

## Phase 2: State Store Tests (IMP-005)
- [X] Task 2.1: Add tests for RecordPerformanceMetric and GetPerformanceMetrics (round-trip, filter by stepID, nil completedAt) [P]
- [X] Task 2.2: Add tests for GetStepPerformanceStats (aggregation correctness, empty result for no data, since filter) [P]
- [X] Task 2.3: Add tests for GetRecentPerformanceHistory (limit, filter by pipeline/step/persona) [P]
- [X] Task 2.4: Add tests for GetProgressSnapshots (round-trip, limit, filter by step) [P]
- [X] Task 2.5: Add tests for SaveArtifactMetadata and GetArtifactMetadata (round-trip, not-found case) [P]

## Phase 3: Executor Transitive Skip & Batch Cancellation Tests (IMP-009)
- [X] Task 3.1: Add test for transitive skip with diamond dependency pattern (A fails, B and C depend on A, D depends on B+C — all skipped)
- [X] Task 3.2: Add test for transitive skip with mixed optional/required deps (only paths through failed dep are skipped)
- [X] Task 3.3: Add test for concurrent batch cancellation — when one step in a concurrent batch fails, verify sibling steps are properly handled

## Phase 4: Validation
- [X] Task 4.1: Run `go test -race ./internal/deliverable/... ./internal/state/... ./internal/pipeline/...` and fix any failures
- [X] Task 4.2: Run `golangci-lint run ./internal/deliverable/... ./internal/state/... ./internal/pipeline/...` and fix any warnings
