# Implementation Plan

## Objective

Add comprehensive test coverage for three under-tested areas: deliverable tracker (IMP-003), state store performance/progress/artifact methods (IMP-005), and executor transitive skip + batch cancellation edge cases (IMP-009). Step controller (IMP-008) is already well-tested.

## Approach

Write table-driven tests following existing patterns in each package. All tests must pass with `-race` flag. Focus on the specific untested methods identified in the audit.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/deliverable/tracker_test.go` | create | Tests for all Tracker methods: NewTracker, SetPipelineID, Add (dedup), GetAll, GetByStep, Count, FormatSummary, FormatByStep, GetLatestForStep, AddWorkspaceFiles, AddOutcomeWarning, OutcomeWarnings, concurrent access |
| `internal/state/store_test.go` | modify | Add tests for RecordPerformanceMetric, GetPerformanceMetrics, GetStepPerformanceStats, GetRecentPerformanceHistory, GetProgressSnapshots, SaveArtifactMetadata, GetArtifactMetadata |
| `internal/pipeline/executor_test.go` | modify | Add edge-case tests for transitive skip (diamond deps, multi-branch) and concurrent batch cancellation |

## Architecture Decisions

1. **New file for deliverable tracker tests** — `types_test.go` already exists testing type constructors and some tracker convenience methods. A dedicated `tracker_test.go` is cleaner for the tracker's core methods.
2. **Extend existing state store_test.go** — follows the pattern already established with setupTestStore helper.
3. **Extend executor_test.go** — uses existing test infrastructure (stepAwareAdapter, testEventCollector, createTestManifest).

## Risks

| Risk | Mitigation |
|------|-----------|
| State store progress methods require `CreateRun` + `SaveStepState` setup | Use existing `setupTestStore` helper, create proper run+step first |
| Concurrent tests may be flaky without proper synchronization | Use `sync.WaitGroup`, controlled goroutine counts, deterministic assertions |
| Executor concurrent batch cancellation requires careful adapter mocking | Use slow/blocking adapters with context cancellation |

## Testing Strategy

- **Deliverable tracker**: Table-driven tests for each method, plus dedicated concurrent goroutine tests with `-race`
- **State store**: Table-driven CRUD tests for each method family, round-trip verification (write then read)
- **Executor**: Integration-style tests using mock adapters with step-aware routing, event collector verification
