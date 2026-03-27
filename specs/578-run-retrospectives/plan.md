# Implementation Plan: Run Retrospectives (#578)

## 1. Objective

Add automatic run retrospectives that combine quantitative metrics (duration, retries, tokens, files) with an optional LLM-narrated analysis (smoothness rating, friction points, learnings, open items) after every pipeline run. Provide CLI commands for viewing, listing, and aggregating retrospectives, and surface retro data in the web UI.

## 2. Approach

### Two-Phase Architecture

**Phase 1 — Quantitative (synchronous, no LLM)**: After each pipeline run completes, the executor calls `retro.Generate()` which collects metrics from the existing state store (step states, performance metrics, step attempts, events) and writes a structured JSON retro to `.wave/retros/<run-id>.json` and a SQLite `retrospective` table.

**Phase 2 — Narrative (async, LLM-powered, optional)**: If `runtime.retro.narrate: true`, a cheap adapter invocation (e.g., haiku) is triggered to generate the narrative section. This runs after the quantitative retro is saved to avoid blocking. The narrative is appended to the existing retro file and SQLite record.

### Integration Strategy

- **Hook point**: `internal/pipeline/executor.go` after line 645 (pipeline completion event), before `cleanupCompletedPipeline()`. The executor already has access to `e.store`, `e.emitter`, and the full `execution` state.
- **Data sources**: `state.GetPerformanceMetrics()`, `state.GetStepAttempts()`, `state.GetStepStates()`, `state.GetEvents()` — all existing infrastructure.
- **Adapter reuse**: Use the existing `AdapterRunner` interface with `Model: "haiku"` for narrative generation. No new adapter code needed.
- **Filesystem + SQLite dual storage**: JSON files under `.wave/retros/` for human-readable access; SQLite `retrospective` table for querying and aggregation.

## 3. File Mapping

### New Files (Create)

| Path | Purpose |
|------|---------|
| `internal/retro/retro.go` | Core types: `Retrospective`, `QuantitativeData`, `StepMetrics`, `NarrativeData`, `FrictionPoint`, `Learning`, `OpenItem` |
| `internal/retro/collector.go` | `Collector` — gathers quantitative data from state store |
| `internal/retro/narrator.go` | `Narrator` — generates LLM narrative via adapter |
| `internal/retro/store.go` | `Store` interface — filesystem + SQLite persistence |
| `internal/retro/retro_test.go` | Unit tests for core types and JSON marshaling |
| `internal/retro/collector_test.go` | Unit tests for quantitative collector |
| `internal/retro/narrator_test.go` | Unit tests for narrator (mock adapter) |
| `internal/retro/store_test.go` | Unit tests for file + SQLite store |
| `cmd/wave/commands/retro.go` | CLI: `wave retro <run-id>`, `wave retro list`, `wave retro stats` |
| `cmd/wave/commands/retro_test.go` | CLI command tests |

### Modified Files

| Path | Change |
|------|--------|
| `internal/manifest/types.go` | Add `RetroConfig` to `Runtime` struct |
| `internal/pipeline/executor.go` | Add retro generation hook after pipeline completion |
| `internal/pipeline/executor_options.go` | Add `WithRetroStore()` option (if exists, else in executor.go) |
| `internal/state/migration_definitions.go` | Add migration 12: `retrospective` table |
| `internal/state/store.go` | Add `SaveRetrospective()`, `GetRetrospective()`, `ListRetrospectives()` to `StateStore` interface |
| `internal/state/types.go` | Add `RetrospectiveRecord` type (if exists, else in store.go) |
| `cmd/wave/main.go` | Register `NewRetroCmd()` |
| `cmd/wave/commands/run.go` | Add `--no-retro` flag |
| `internal/webui/routes.go` | Add retro API routes |
| `internal/webui/server.go` | Add retro handler methods |

## 4. Architecture Decisions

### AD-1: Filesystem + SQLite dual storage (not either/or)
- **Decision**: Store full JSON retros on filesystem (`.wave/retros/<run-id>.json`) and index/summary data in SQLite
- **Rationale**: JSON files are human-readable and diff-friendly. SQLite enables efficient querying/aggregation for `wave retro list` and `wave retro stats`. This follows the same pattern as audit traces (`.wave/traces/`)
- **Trade-off**: Slight duplication, but the concerns are different (browsing vs querying)

### AD-2: Quantitative first, narrative async
- **Decision**: Generate quantitative retro synchronously in the executor's post-run hook. Kick off narrative generation asynchronously (goroutine) to avoid blocking run completion
- **Rationale**: Quantitative data is cheap to collect (just state store queries). Narrative requires an LLM call that could take 10-30 seconds. The run should not be held open for this
- **Trade-off**: Narrative may not be available immediately, but `--narrate` flag can regenerate it

### AD-3: Narrative via existing adapter infrastructure
- **Decision**: Reuse `AdapterRunner` with a minimal `AdapterRunConfig` (model: haiku, prompt: structured summary request)
- **Rationale**: No new adapter code needed. The adapter registry already handles model resolution and execution. Using a cheap model (haiku) keeps costs low
- **Alternative rejected**: Dedicated API call — would bypass sandbox/permissions infrastructure

### AD-4: RetroConfig on Runtime struct (singular `retro`, not `retros`)
- **Decision**: Use `runtime.retro` in YAML config, matching the singular naming of other Runtime fields (`audit`, `relay`, `sandbox`)
- **Rationale**: Consistency with existing manifest schema naming conventions

### AD-5: New `internal/retro/` package (not extending `internal/state/`)
- **Decision**: Self-contained package with its own store interface that delegates to state store for SQLite and handles filesystem writes directly
- **Rationale**: Separation of concerns — retro logic (collection, narration, aggregation) is distinct from state persistence. Keeps the state package focused

## 5. Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Narrative LLM call fails | No narrative in retro | Graceful degradation — quantitative retro still persisted; error logged; user can retry with `wave retro <id> --narrate` |
| Executor cleanup races with retro | Data unavailable | Generate retro BEFORE `cleanupCompletedPipeline()` — data is still in memory |
| Large state store queries slow on big DBs | Slow retro generation | Use indexed queries; limit event history to current run only; quantitative collector is bounded |
| Migration 12 on existing DBs | DB upgrade needed | Auto-migration system handles this; `CREATE TABLE IF NOT EXISTS` is safe |
| Disk space from retro files | `.wave/retros/` grows | Future: add cleanup/retention policy (not in scope for initial impl) |
| Concurrent adapter call for narrative | API rate limits | Use cheapest model (haiku); single call per run; async so doesn't compete with pipeline steps |

## 6. Testing Strategy

### Unit Tests
- `internal/retro/retro_test.go`: JSON marshaling/unmarshaling of all retro types, smoothness enum validation
- `internal/retro/collector_test.go`: Collector with mock state store — verifies correct aggregation of step metrics, retry counts, duration calculations
- `internal/retro/narrator_test.go`: Narrator with mock adapter — verifies prompt construction, result parsing, graceful failure handling
- `internal/retro/store_test.go`: File store write/read roundtrip; SQLite save/get/list with real temp DB

### Integration Tests
- `internal/pipeline/executor_test.go`: Verify retro is generated after pipeline completion, verify `--no-retro` skips generation
- `cmd/wave/commands/retro_test.go`: CLI output formatting for view, list, stats subcommands

### Edge Cases
- Pipeline with 0 steps (empty retro)
- Pipeline where all steps failed
- Pipeline with retries on every step
- Narrative generation timeout/failure
- Missing/corrupt retro file on disk
- `wave retro` for a run that has no retro yet
