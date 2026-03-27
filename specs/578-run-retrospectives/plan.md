# Implementation Plan: Run Retrospectives

## Objective

Add automatic retrospective generation after every pipeline run — quantitative metrics collected from existing state infrastructure, plus optional LLM-powered narrative analysis. Exposed via CLI commands, web UI, and stored as JSON files.

## Approach

### High-Level Strategy

1. **New `internal/retro/` package** — core domain types, quantitative collector, narrative generator, storage
2. **Hook into pipeline executor** — generate retrospective at pipeline completion (after `runTerminalHooks`, before `cleanupCompletedPipeline`)
3. **SQLite migration** — add `retrospective` table for metadata/indexing (full data in JSON files)
4. **Manifest extension** — `runtime.retros` config section in `Runtime` struct
5. **CLI command** — `wave retro` with subcommands (view, list, stats, narrate)
6. **Web UI** — API endpoints and HTML handlers for retrospective views
7. **Run flag** — `--no-retro` on `wave run`

### Key Design Decisions

**Storage**: Dual storage — JSON files at `.wave/retros/<run-id>.json` for full data (easy to inspect, export, back up) plus a SQLite `retrospective` table for indexing/querying (run_id, pipeline, timestamp, smoothness, status).

**Narrative generation**: Uses the existing adapter infrastructure. The `runtime.retros.narrate_model` field resolves to an adapter+model combo. Narrative is generated asynchronously (non-blocking) after the quantitative retro is written, so it never delays pipeline completion.

**Data sources**: All quantitative data comes from existing state store tables:
- `RunRecord` → total duration, status, tokens, pipeline name
- `PerformanceMetricRecord` → per-step duration, tokens, files modified, adapter/persona
- `StepAttemptRecord` → retry counts, failure classes, error messages
- `LogRecord` → event timeline, step transitions
- `OntologyUsageRecord` → contract validation results

## File Mapping

### New Files (create)

| Path | Purpose |
|------|---------|
| `internal/retro/types.go` | Domain types: `Retrospective`, `QuantitativeData`, `StepMetrics`, `Narrative`, `FrictionPoint`, `Learning`, `OpenItem` |
| `internal/retro/collector.go` | `Collector` — builds quantitative retrospective from state store data |
| `internal/retro/collector_test.go` | Unit tests for collector |
| `internal/retro/narrator.go` | `Narrator` — generates LLM narrative from quantitative data via adapter |
| `internal/retro/narrator_test.go` | Unit tests for narrator (mock adapter) |
| `internal/retro/storage.go` | `Storage` — JSON file read/write + SQLite index operations |
| `internal/retro/storage_test.go` | Unit tests for storage |
| `internal/retro/generator.go` | `Generator` — orchestrates collector → narrator → storage pipeline |
| `internal/retro/generator_test.go` | Unit tests for generator |
| `cmd/wave/commands/retro.go` | CLI command: `wave retro` with view/list/stats/narrate subcommands |
| `cmd/wave/commands/retro_test.go` | CLI command tests |
| `internal/webui/handlers_retros.go` | Web UI API + HTML handlers for retrospectives |
| `internal/webui/handlers_retros_test.go` | Web UI handler tests |

### Modified Files

| Path | Change |
|------|--------|
| `internal/manifest/types.go` | Add `RetrosConfig` struct, add `Retros` field to `Runtime` |
| `internal/manifest/parser.go` | Add validation for retros config |
| `internal/state/types.go` | Add `RetrospectiveRecord` type |
| `internal/state/store.go` | Add retro methods to `StateStore` interface |
| `internal/state/store_impl.go` or inline | Implement retro store methods |
| `internal/state/migration_definitions.go` | Add migration 13: `retrospective` table |
| `internal/pipeline/executor.go` | Call retro generator after pipeline completion |
| `cmd/wave/commands/run.go` | Add `--no-retro` flag |
| `cmd/wave/main.go` | Register `NewRetroCmd()` |
| `internal/webui/routes.go` | Add retro API/page routes |
| `internal/webui/server.go` | Wire retro dependencies into Server |

## Architecture Decisions

### AD1: JSON files + SQLite index (not SQLite-only)

**Decision**: Store full retrospective data as JSON files in `.wave/retros/`, with a lightweight SQLite index for querying.

**Rationale**: JSON files are human-readable, easy to export to external analytics (DuckDB), and don't bloat the SQLite database with potentially large narrative text. The SQLite index enables efficient filtering by pipeline, date range, and smoothness rating.

### AD2: Non-blocking narrative generation

**Decision**: Quantitative retro is generated synchronously (fast, no LLM). Narrative is generated asynchronously after the quantitative retro is saved.

**Rationale**: Narrative generation requires an LLM call (30-60s). Blocking pipeline completion would degrade the user experience. The quantitative retro is immediately useful; the narrative enhances it later.

### AD3: Adapter reuse for narration

**Decision**: Use the existing `adapter.AdapterRunner` interface to call a cheap model for narrative generation rather than implementing direct API calls.

**Rationale**: Leverages existing infrastructure (subprocess execution, timeout handling, token tracking). Supports all configured adapters (claude, opencode, etc.) without new dependencies.

### AD4: `runtime.retros` config (not top-level)

**Decision**: Place retro configuration under `runtime` in the manifest.

**Rationale**: Retros are a runtime behavior, not a first-class manifest entity like personas or pipelines. This keeps the manifest schema clean and groups retros with other runtime behaviors (audit, relay, sandbox).

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Narrative LLM call fails/times out | Medium | Low | Quantitative retro saved first; narrative failure logged but doesn't affect run |
| Large SQLite index growth | Low | Low | Retros are per-run (not per-step); `wave clean` can prune old retros |
| State store missing data for old runs | Medium | Low | Collector gracefully handles missing metrics; reports what's available |
| Adapter binary not available for narration | Low | Medium | Skip narrative with warning if adapter binary missing |

## Testing Strategy

### Unit Tests
- `internal/retro/collector_test.go` — mock state store, verify quantitative data assembly
- `internal/retro/narrator_test.go` — mock adapter, verify prompt construction and response parsing
- `internal/retro/storage_test.go` — temp dir JSON read/write, SQLite index CRUD
- `internal/retro/generator_test.go` — integration of collector → narrator → storage
- `cmd/wave/commands/retro_test.go` — CLI flag parsing, subcommand routing
- `internal/webui/handlers_retros_test.go` — HTTP handler request/response

### Integration Tests
- Pipeline executor generates retro after successful/failed run (using mock adapter)
- `--no-retro` flag suppresses generation
- `wave retro list` queries real SQLite database
- Web UI API returns correct JSON for retro endpoints
