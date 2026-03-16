# test: add coverage for deliverable tracker, state methods, and pipeline step controller

**Issue**: [#428](https://github.com/re-cinq/wave/issues/428)
**Author**: nextlevelshit
**State**: OPEN
**Labels**: none

## Coverage Gap Analysis (wave-test-hardening audit)

### IMP-003: Deliverable tracker mostly untested — 31.1% (HIGH)
- internal/deliverable/tracker.go: NewTracker, SetPipelineID, Add (deduplication), GetAll, GetByStep, Count, FormatSummary, FormatByStep, GetLatestForStep, AddWorkspaceFiles, AddOutcomeWarning, OutcomeWarnings
- Concurrent access via sync.RWMutex is not race-tested

### IMP-005: State performance/progress/artifact methods untested (HIGH)
- internal/state/store.go: RecordPerformanceMetric, GetPerformanceMetrics, GetStepPerformanceStats, GetRecentPerformanceHistory, progress snapshots (GetProgressSnapshots), artifact metadata (SaveArtifactMetadata, GetArtifactMetadata)

### IMP-008: Pipeline step controller entirely untested (MEDIUM)
- internal/pipeline/step_controller.go: entire step lifecycle management
- **NOTE**: `stepcontroller_test.go` already exists with comprehensive tests for NewStepController, ContinueStep, ExtendStep, RevertStep, ConfirmRevert, RewriteStep, findStep, findPipelineStep, countFiles. This item appears already addressed.

### IMP-009: Transitive skip and concurrent batch cancellation untested (HIGH)
- internal/pipeline/executor.go: TransitiveSkip behavior, batch cancellation on failure
- **NOTE**: `TestOptionalStep_TransitiveSkip` already exists in executor_test.go testing transitive skip propagation. Additional edge cases and concurrent batch cancellation still needed.

## Acceptance Criteria

1. Table-driven tests for each deliverable tracker method: Add deduplication, concurrent access with `-race`
2. State store: performance metric recording/retrieval, progress snapshot CRUD, artifact metadata CRUD
3. Executor: additional transitive skip edge cases (diamond deps, multi-branch), concurrent batch cancellation
4. All new tests pass with `go test -race ./...`
