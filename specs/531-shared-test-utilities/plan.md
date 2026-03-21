# Implementation Plan: Extract Shared Test Utilities

## Objective

Extract duplicated test mocks (`MockStateStore`, event collectors, test manifest helpers) from multiple test files into a shared `internal/testutil` package to eliminate code duplication and provide a single source of truth for test infrastructure.

## Approach

Create `internal/testutil` as a test-only utility package with three focused files:
1. **`statestore.go`** — Generic `MockStateStore` implementing `state.StateStore` with configurable method overrides
2. **`events.go`** — Thread-safe `EventCollector` implementing `event.EventEmitter` with query/assertion helpers
3. **`manifest.go`** — Shared `CreateTestManifest` helper

Use functional options pattern for `MockStateStore` to allow tests to override specific methods without stubbing all 30+ interface methods. This matches the existing `adapter.MockAdapter` pattern already used in the codebase.

## File Mapping

### New Files
| Path | Purpose |
|------|---------|
| `internal/testutil/doc.go` | Package documentation |
| `internal/testutil/statestore.go` | `MockStateStore` with functional options |
| `internal/testutil/events.go` | `EventCollector` (thread-safe event.EventEmitter mock) |
| `internal/testutil/manifest.go` | `CreateTestManifest` helper |

### Modified Files (consumers migrated to use testutil)
| Path | Change |
|------|--------|
| `internal/pipeline/executor_test.go` | Remove `testEventCollector`, `MockStateStore`, `createTestManifest`; import from testutil |
| `internal/pipeline/matrix_test.go` | Remove `matrixTestEventCollector`; use `testutil.NewEventCollector()` |
| `internal/pipeline/composition_test.go` | Remove `testEmitter`; use `testutil.NewEventCollector()` |
| `internal/pipeline/concurrency_test.go` | Update `createTestManifest` calls to `testutil.CreateTestManifest` |
| `internal/pipeline/resume_test.go` | Update `createTestManifest` and `MockStateStore` references |
| `internal/pipeline/sequence_test.go` | Update event collector references |
| `internal/pipeline/failure_modes_test.go` | Update event collector references |
| `internal/pipeline/executor_schema_test.go` | Update event collector references |
| `internal/pipeline/contract_integration_test.go` | Update event collector references |

### NOT Migrated (intentionally)
| Path | Reason |
|------|--------|
| `cmd/wave/commands/run_test.go` | Integration test (`//go:build integration`), different package, simpler collector |
| `internal/tui/pipeline_provider_test.go` | `baseStateStore` uses interface embedding — different pattern, TUI-specific subset |
| `internal/pipeline/stepcontroller_test.go` | `mockStepStore` embeds interface and only implements `LogEvent` — very focused |
| `internal/pipeline/eta_test.go` | `mockStoreForETA` is ETA-specific with custom return values |
| `internal/pipeline/dag_test.go` | `mockSkillStore` implements a different interface (`skill.Store`) |
| `internal/continuous/runner_test.go` | `mockEmitter` is 3 lines — not worth extracting |

## Architecture Decisions

1. **Functional options for MockStateStore**: Tests can override only the methods they care about. Default behavior returns zero-values/nil errors. This avoids the 80-line stub problem.
2. **Thread-safe EventCollector**: All methods protected by `sync.Mutex` since pipeline tests run concurrent steps. The non-thread-safe variants in composition_test.go and run_test.go work by accident (single-step pipelines) but should use the safe version.
3. **Exported types**: `testutil` is an internal package, but types are exported so they can be used from `internal/pipeline`, `cmd/wave/commands`, etc.
4. **No code generation**: Hand-written mocks with functional options. The interface is stable enough that maintenance burden is low.
5. **Package-level `configCapturingAdapter` stays in executor_test.go**: It wraps `MockAdapter` to capture `AdapterRunConfig` — this is pipeline-executor-specific and not broadly reusable.

## Risks

| Risk | Mitigation |
|------|------------|
| Breaking test compilation across many files | Migrate one file at a time, run `go test ./...` after each |
| Method signature drift if `StateStore` interface changes | Single point of update in testutil vs. N scattered mocks — this is actually an improvement |
| Import cycles | `internal/testutil` imports `state`, `event`, `manifest` — all leaf packages with no reverse dependencies |

## Testing Strategy

- Run `go test ./...` after creating the testutil package to ensure it compiles
- Run `go test ./...` after each file migration to catch breakage immediately
- Run `go test -race ./...` at the end to verify thread safety
- No new tests needed for testutil itself — it's validated by the existing test suite that uses it
