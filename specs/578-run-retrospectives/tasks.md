# Tasks

## Phase 1: Core Types and Configuration

- [X] Task 1.1: Create `internal/retro/retro.go` with core types — `Retrospective`, `QuantitativeData`, `StepMetrics`, `NarrativeData`, `FrictionPoint`, `Learning`, `OpenItem`, smoothness constants
- [X] Task 1.2: Add `RetroConfig` struct to `internal/manifest/types.go` — `Enabled`, `Narrate`, `NarrateModel` fields on `Runtime`
- [X] Task 1.3: Add SQLite migration 12 in `internal/state/migration_definitions.go` — `retrospective` table with `run_id`, `pipeline_name`, `quantitative_json`, `narrative_json`, `smoothness`, `generated_at`
- [X] Task 1.4: Add `SaveRetrospective()`, `GetRetrospective()`, `ListRetrospectives()` to `StateStore` interface and implement in `internal/state/store.go`

## Phase 2: Quantitative Collector

- [X] Task 2.1: Create `internal/retro/collector.go` — `Collector` struct that queries `StateStore` for performance metrics, step states, step attempts, and events to build `QuantitativeData`
- [X] Task 2.2: Write unit tests `internal/retro/collector_test.go` — mock state store, verify aggregation logic for durations, retries, success/failure ratios

## Phase 3: Storage Layer

- [X] Task 3.1: Create `internal/retro/store.go` — `Store` interface with `Save()`, `Get()`, `List()`, `ListByPipeline()` methods; `FileStore` implementation writing to `.wave/retros/<run-id>.json`; delegates SQLite persistence to `StateStore`
- [X] Task 3.2: Write unit tests `internal/retro/store_test.go` — file write/read roundtrip, SQLite save/get/list with temp DB

## Phase 4: Narrator (LLM Narrative)

- [X] Task 4.1: Create `internal/retro/narrator.go` — `Narrator` struct that takes `AdapterRunner`, constructs a prompt from quantitative data + run context, invokes cheap model, parses structured JSON response into `NarrativeData`
- [X] Task 4.2: Write unit tests `internal/retro/narrator_test.go` — mock adapter, verify prompt construction, JSON response parsing, graceful failure handling

## Phase 5: Executor Integration

- [X] Task 5.1: Add retro generation hook in `internal/pipeline/executor.go` — after pipeline completion (line ~645), before cleanup. Call `retro.Collector.Collect()` then `retro.Store.Save()`. If narrate enabled, launch `Narrator.Narrate()` in goroutine
- [X] Task 5.2: Add `WithRetroStore()` executor option for dependency injection
- [X] Task 5.3: Add `--no-retro` flag to `cmd/wave/commands/run.go`
- [X] Task 5.4: Write integration test verifying retro is generated after pipeline completion, and `--no-retro` skips it

## Phase 6: CLI Commands

- [X] Task 6.1: Create `cmd/wave/commands/retro.go` — `wave retro <run-id>` (view), `wave retro list` (list with `--pipeline`, `--since`, `--limit` flags), `wave retro stats` (aggregate stats), `wave retro <run-id> --narrate` (regenerate narrative) [P]
- [X] Task 6.2: Register `NewRetroCmd()` in `cmd/wave/main.go` [P]
- [X] Task 6.3: Write CLI tests `cmd/wave/commands/retro_test.go` [P]

## Phase 7: Web UI Integration

- [X] Task 7.1: Add retro API routes to `internal/webui/routes.go` — `GET /api/runs/{id}/retro`, `GET /api/retros` [P]
- [X] Task 7.2: Add retro handler implementations in `internal/webui/handlers_retro.go` [P]

## Phase 8: Validation and Polish

- [X] Task 8.1: Run `go test ./...` and fix any failures
- [X] Task 8.2: Run `go test -race ./...` and fix any data races
- [X] Task 8.3: Run `golangci-lint run ./...` and fix any lint issues
- [X] Task 8.4: Verify end-to-end: run a pipeline, check retro file exists, view with CLI
