# refactor: extract shared test utilities into internal/testutil

**Issue**: [#531](https://github.com/re-cinq/wave/issues/531)
**Author**: nextlevelshit
**State**: OPEN
**Labels**: none

## Problem Statement

Each package re-implements mocks independently. `MockStateStore` in `executor_test.go` is 80+ lines with 25 stubbed methods. Any other package needing a fake state store must duplicate this. Extract `MockStateStore`, `testEventCollector`, and common assertion helpers into `internal/testutil`.

## Current Duplication Analysis

### Event Collectors (5 independent implementations)

| File | Type | Thread-safe | Methods |
|------|------|-------------|---------|
| `internal/pipeline/executor_test.go` | `testEventCollector` | Yes (sync.Mutex) | Emit, GetEvents, GetPipelineID, HasEventWithState, GetEventsByStep, GetStepExecutionOrder |
| `internal/pipeline/matrix_test.go` | `matrixTestEventCollector` | Yes (sync.Mutex) | Emit, GetEvents |
| `internal/pipeline/composition_test.go` | `testEmitter` | No | Emit, hasState |
| `cmd/wave/commands/run_test.go` | `testEventCollector` | No | Emit, HasEvent, GetEventsByState, GetEventsByStep |
| `internal/continuous/runner_test.go` | `mockEmitter` | No | Emit |

### State Store Mocks (4 independent implementations)

| File | Type | Approach |
|------|------|----------|
| `internal/pipeline/executor_test.go` | `MockStateStore` | Full implementation with maps, 80+ lines, 25+ stubbed methods |
| `internal/pipeline/stepcontroller_test.go` | `mockStepStore` | Embed `state.StateStore` interface, implement only `LogEvent` |
| `internal/tui/pipeline_provider_test.go` | `baseStateStore` + `mockStateStore` | Base with no-op stubs, mock overrides specific methods |
| `internal/pipeline/eta_test.go` | `mockStoreForETA` | Partial implementation for ETA-specific methods |

### Other Duplicated Mocks

- `mockSkillStore` in both `internal/pipeline/dag_test.go` and `internal/manifest/parser_test.go` (different interfaces)
- `createTestManifest` helper in `internal/pipeline/executor_test.go` used across pipeline tests
- `configCapturingAdapter` in `internal/pipeline/executor_test.go`

## Acceptance Criteria

1. New `internal/testutil` package exists with exported test utilities
2. `MockStateStore` is shared — fully implements `state.StateStore` with configurable behavior
3. `EventCollector` is shared — thread-safe, implements `event.EventEmitter`, with query methods
4. Common test manifest helpers are shared
5. All existing tests continue to pass after migration
6. No new test dependencies introduced (only stdlib + testify)
