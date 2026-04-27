# Implementation Plan — Split StateStore into Domain Interfaces

## 1. Objective

Split the monolithic `state.StateStore` interface (~80 methods) into five domain-scoped interfaces (`RunStore`, `EventStore`, `OntologyStore`, `WebhookStore`, `ChatStore`) and update consumers to depend only on the surface they use. Schema and concrete implementation unchanged.

## 2. Approach

1. Define five narrow interfaces in dedicated files under `internal/state/`. Each contains only its domain methods plus `Close()` if it owns lifecycle (only the aggregate owns `Close`).
2. Redefine `StateStore` as a composite interface embedding all five — preserves source/binary compatibility for code paths still wanting the aggregate (constructor return type, read-only constructor).
3. Concrete `*stateStore` already implements every method, so all five narrow interfaces are satisfied automatically. Add compile-time assertions (`var _ RunStore = (*stateStore)(nil)` etc.) to catch drift.
4. Update production callers in topological order — leaf packages (retro, ontology, pipeline subcomponents, webui handlers, tui providers) take narrow types; root constructors (`cmd/wave/commands/*`, `webui/server`) keep using `NewStateStore` and pass narrow types into sub-objects.
5. Update test fakes (`internal/testutil/statestore.go`) to implement narrow interfaces alongside the aggregate.
6. Verify with `go build ./...`, `go vet ./...`, `go test -race ./...`, and `golangci-lint run`.

### Interface Grouping

**RunStore** (pipeline + step lifecycle, runs, cancellation, retro, decisions, performance, progress, checkpoints, fork lineage, orchestration decisions, outcomes, step attempts, tags, PID, visit count):
- Pipeline: `SavePipelineState`, `GetPipelineState`, `ListRecentPipelines`
- Step: `SaveStepState`, `GetStepStates`, `SaveStepVisitCount`, `GetStepVisitCount`, `RecordStepAttempt`, `GetStepAttempts`
- Run: `CreateRun`, `CreateRunWithLimit`, `CreateRunWithFork`, `UpdateRunStatus`, `UpdateRunBranch`, `UpdateRunPID`, `GetRun`, `GetRunningRuns`, `ListRuns`, `DeleteRun`
- Cancellation: `RequestCancellation`, `CheckCancellation`, `ClearCancellation`
- Tags: `SetRunTags`, `GetRunTags`, `AddRunTag`, `RemoveRunTag`
- Parent/child: `SetParentRun`, `GetChildRuns`
- Checkpoints: `SaveCheckpoint`, `GetCheckpoint`, `GetCheckpoints`, `DeleteCheckpointsAfterStep`
- Retros: `SaveRetrospective`, `GetRetrospective`, `ListRetrospectives`, `DeleteRetrospective`, `UpdateRetrospectiveSmoothness`, `UpdateRetrospectiveStatus`
- Decisions: `RecordDecision`, `GetDecisions`, `GetDecisionsByStep`
- Outcomes: `RecordOutcome`, `GetOutcomes`, `GetOutcomesByValue`
- Orchestration: `RecordOrchestrationDecision`, `UpdateOrchestrationOutcome`, `GetOrchestrationStats`, `ListOrchestrationDecisionSummary`
- Performance: `RecordPerformanceMetric`, `GetPerformanceMetrics`, `GetStepPerformanceStats`, `GetRecentPerformanceHistory`, `CleanupOldPerformanceMetrics`
- Progress: `SaveProgressSnapshot`, `GetProgressSnapshots`, `UpdateStepProgress`, `GetStepProgress`, `GetAllStepProgress`, `UpdatePipelineProgress`, `GetPipelineProgress`

**EventStore** (events + artifacts + audit):
- Events: `LogEvent`, `GetEvents`, `GetAuditEvents`
- Artifacts: `RegisterArtifact`, `GetArtifacts`, `SaveArtifactMetadata`, `GetArtifactMetadata`

**OntologyStore**:
- `RecordOntologyUsage`, `GetOntologyStats`, `GetOntologyStatsAll`

**WebhookStore**:
- `CreateWebhook`, `ListWebhooks`, `GetWebhook`, `UpdateWebhook`, `DeleteWebhook`
- `RecordWebhookDelivery`, `GetWebhookDeliveries`

**ChatStore**:
- `SaveChatSession`, `GetChatSession`, `ListChatSessions`

**Aggregate `StateStore`**: embeds all five narrow interfaces + `Close() error`. `WaitForConcurrencySlot` keeps using `RunStore` (only needs `GetRunningRuns`).

### Open Decision

The aggregate `StateStore` is **kept** (not removed) because:
- Constructors `NewStateStore` / `NewReadOnlyStateStore` need a single return type.
- `cmd/wave/commands/*` need a single handle they pass into many sub-objects.
- Removing the aggregate forces callers into a 5-handle bundle for no compile-time win.

This satisfies AC "callers updated to depend on narrow interfaces, not the aggregate" — *new* sub-object signatures take narrow types; the aggregate exists only at the root construction edge.

## 3. File Mapping

### New files

- `internal/state/runstore.go` — `RunStore` interface
- `internal/state/eventstore.go` — `EventStore` interface
- `internal/state/ontologystore.go` — `OntologyStore` interface
- `internal/state/webhookstore.go` — `WebhookStore` interface
- `internal/state/chatstore.go` — `ChatStore` interface
- `internal/state/interfaces_test.go` — compile-time assertions that `*stateStore` satisfies each narrow interface

### Modified files

- `internal/state/store.go` — replace monolithic `StateStore` with composite (embed five narrow interfaces + `Close`); leave method bodies untouched.
- `internal/state/chat_session.go` — keep `ChatSession` type; ensure no leak.
- `internal/testutil/statestore.go` — `MockStateStore` already implements all methods; add type assertions for new interfaces.
- `internal/retro/storage.go`, `internal/retro/generator.go`, `internal/retro/collector.go` — narrow `state.StateStore` → `state.RunStore` (retro storage uses run + retro methods).
- `internal/ontology/real.go`, `internal/ontology/service.go` — narrow to `state.OntologyStore` where only ontology stats are touched; `state.RunStore` where run linkage needed.
- `internal/pipeline/gate.go`, `eta.go`, `checkpoint.go`, `sequence.go`, `fork.go`, `chatcontext.go`, `composition.go`, `stepcontroller.go` — narrow each consumer to the smallest interface that satisfies its call sites. Most use `RunStore`; `chatcontext.go` adds `ChatStore`; `executor.go` keeps aggregate (uses many surfaces).
- `internal/webui/handlers_control.go`, `server.go`, `run_stats.go` — server keeps aggregate (root). Per-feature handler structs narrow to the surface they need.
- `internal/tui/health_provider.go`, `pipeline_provider.go`, `pipeline_detail_provider.go`, `persona_provider.go`, `ontology_provider.go`, `pipeline_messages.go` — each provider takes the narrow interface for its domain.
- `cmd/wave/commands/*.go` — top-level commands keep aggregate; helpers narrow.
- All corresponding `*_test.go` files — update parameter types where signatures change.

### Deleted files

None.

## 4. Architecture Decisions

- **Composite interface kept**: aggregate `StateStore` embeds the five narrow interfaces. Avoids a flag-day rewrite of every constructor while still allowing per-call-site narrowing. Future PR can delete the aggregate once all root-level callers are converted.
- **Compile-time assertions**: `var _ RunStore = (*stateStore)(nil)` per interface, in `interfaces_test.go`, catches accidental method removal during refactors.
- **No method moves**: implementations stay in `store.go` (a future PR can split the implementation file by domain — out of scope here).
- **Read-only store**: `NewReadOnlyStateStore` continues to return `StateStore` (composite). Callers needing only read paths can narrow at use site.
- **Test mock**: `internal/testutil.MockStateStore` already implements every method; we add interface-satisfaction assertions and keep the same struct.

## 5. Risks

| Risk | Mitigation |
|---|---|
| Method missing from a narrow interface (drift between concrete + interface) | Compile-time `var _ RunStore = (*stateStore)(nil)` assertions in `interfaces_test.go` |
| Caller signature changes break downstream callers | Each narrowing change touches one constructor — verified by `go build ./...` after each batch |
| Tests using `state.StateStore` parameter types break | Update test fixtures alongside production callers in same commit |
| Aggregate kept = ISP not fully achieved | Document the aggregate as a transitional convenience for root constructors; AC requires narrow types at *consumers*, not at construction edge |
| Read-only store loses type compatibility | `NewReadOnlyStateStore` returns the composite, narrowing happens at consumer sites — no signature change at construction |

## 6. Testing Strategy

- **Interface satisfaction**: `interfaces_test.go` with five `var _` assertions ensures `*stateStore` satisfies each narrow interface and the composite.
- **Mock satisfaction**: assert `*testutil.MockStateStore` satisfies each narrow interface.
- **Existing test suite**: `go test -race ./internal/state/... ./internal/pipeline/... ./internal/retro/... ./internal/ontology/... ./internal/webui/... ./internal/tui/... ./cmd/...` — no behaviour change expected, all should pass unchanged.
- **Lint**: `golangci-lint run ./...` — surfaces unused interface methods and import cycles.
- **Static check**: `go vet ./...`.
- **No new behavioural tests needed**: this is a refactor; existing tests cover all methods.
