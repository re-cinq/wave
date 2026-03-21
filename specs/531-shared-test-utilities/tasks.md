# Tasks

## Phase 1: Create testutil Package
- [X] Task 1.1: Create `internal/testutil/doc.go` with package documentation
- [X] Task 1.2: Create `internal/testutil/events.go` — thread-safe `EventCollector` implementing `event.EventEmitter` with `NewEventCollector()`, `Emit()`, `GetEvents()`, `GetPipelineID()`, `HasEventWithState()`, `GetEventsByStep()`, `GetStepExecutionOrder()`
- [X] Task 1.3: Create `internal/testutil/statestore.go` — `MockStateStore` with functional options pattern. Default no-op implementations for all 30+ `state.StateStore` methods. Options like `WithSavePipelineState(func)`, `WithGetPipelineState(func)`, etc. for overriding specific methods
- [X] Task 1.4: Create `internal/testutil/manifest.go` — `CreateTestManifest(workspaceRoot string) *manifest.Manifest` helper
- [X] Task 1.5: Verify `go build ./internal/testutil/...` compiles cleanly

## Phase 2: Migrate Pipeline Package Tests [P]
- [X] Task 2.1: Migrate `internal/pipeline/executor_test.go` — remove `testEventCollector`, `MockStateStore`, `createTestManifest`; replace with `testutil.*` imports [P]
- [X] Task 2.2: Migrate `internal/pipeline/matrix_test.go` — remove `matrixTestEventCollector`; use `testutil.NewEventCollector()` [P]
- [X] Task 2.3: Migrate `internal/pipeline/composition_test.go` — remove `testEmitter`; use `testutil.NewEventCollector()` [P]
- [X] Task 2.4: Migrate `internal/pipeline/concurrency_test.go` — update `createTestManifest` calls [P]
- [X] Task 2.5: Migrate `internal/pipeline/resume_test.go` — update `MockStateStore` and `createTestManifest` references [P]
- [X] Task 2.6: Migrate `internal/pipeline/sequence_test.go` — update event collector usage [P]
- [X] Task 2.7: Migrate `internal/pipeline/failure_modes_test.go` — update event collector usage [P]
- [X] Task 2.8: Migrate `internal/pipeline/executor_schema_test.go` — update event collector usage [P]
- [X] Task 2.9: Migrate `internal/pipeline/contract_integration_test.go` — update event collector usage [P]
- [X] Task 2.10: Migrate `internal/pipeline/gate_test.go` — remove `testEmitter` usage; use `testutil.NewEventCollector()` [P] (discovered during migration)

## Phase 3: Validation
- [X] Task 3.1: Run `go test ./internal/pipeline/...` to verify all pipeline tests pass
- [X] Task 3.2: Run `go test ./...` to verify no regressions across entire codebase
- [X] Task 3.3: Run `go test -race ./...` to verify thread safety
- [ ] Task 3.4: Run `golangci-lint run ./...` to verify no lint issues

## Phase 4: Polish
- [X] Task 4.1: Verify no unused imports or dead code in migrated test files
- [X] Task 4.2: Ensure `configCapturingAdapter` remains in `executor_test.go` (not extracted — pipeline-specific)
