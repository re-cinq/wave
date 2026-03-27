# Tasks

## Phase 1: Domain Types and Configuration

- [ ] Task 1.1: Create `internal/retro/types.go` with domain types (`Retrospective`, `QuantitativeData`, `StepMetrics`, `Narrative`, `FrictionPoint`, `Learning`, `OpenItem`, smoothness constants)
- [ ] Task 1.2: Add `RetrosConfig` struct and `Retros` field to `Runtime` in `internal/manifest/types.go`
- [ ] Task 1.3: Add retros config validation in `internal/manifest/parser.go`
- [ ] Task 1.4: Add `RetrospectiveRecord` type to `internal/state/types.go`
- [ ] Task 1.5: Add migration 13 for `retrospective` table in `internal/state/migration_definitions.go`
- [ ] Task 1.6: Add retro methods to `StateStore` interface and implement in `internal/state/store.go`

## Phase 2: Core Retro Package

- [ ] Task 2.1: Implement `Collector` in `internal/retro/collector.go` — queries state store for run/step/attempt data, builds `QuantitativeData` [P]
- [ ] Task 2.2: Implement `Storage` in `internal/retro/storage.go` — JSON file read/write at `.wave/retros/`, SQLite index operations [P]
- [ ] Task 2.3: Implement `Narrator` in `internal/retro/narrator.go` — builds LLM prompt from quantitative data, parses structured JSON response, populates `Narrative` [P]
- [ ] Task 2.4: Implement `Generator` in `internal/retro/generator.go` — orchestrates collector → storage → async narrator → storage update

## Phase 3: Pipeline Integration

- [ ] Task 3.1: Hook retro generation into pipeline executor at completion point (`internal/pipeline/executor.go`) — call `Generator.Generate()` after `runTerminalHooks`, before `cleanupCompletedPipeline`
- [ ] Task 3.2: Add `--no-retro` flag to `wave run` command in `cmd/wave/commands/run.go`, pass through to executor
- [ ] Task 3.3: Thread retros config through executor construction (manifest → executor → generator)

## Phase 4: CLI Commands

- [ ] Task 4.1: Create `cmd/wave/commands/retro.go` with `wave retro <run-id>` (view single retro) and `wave retro <run-id> --narrate` (regenerate narrative)
- [ ] Task 4.2: Add `wave retro list` subcommand with `--pipeline` and `--since` filters
- [ ] Task 4.3: Add `wave retro stats` subcommand for aggregate statistics (smoothness distribution, most common friction points, pipeline comparison)
- [ ] Task 4.4: Register `NewRetroCmd()` in `cmd/wave/main.go`

## Phase 5: Web UI

- [ ] Task 5.1: Add retro API routes (`GET /api/retros`, `GET /api/retros/{id}`, `POST /api/retros/{id}/narrate`) in `internal/webui/routes.go`
- [ ] Task 5.2: Implement retro handlers in `internal/webui/handlers_retros.go` — list, detail, trigger narration
- [ ] Task 5.3: Wire retro dependencies (storage, generator) into `Server` struct in `internal/webui/server.go`
- [ ] Task 5.4: Add retro view to run detail page — show quantitative summary and narrative (if available) inline

## Phase 6: Testing

- [ ] Task 6.1: Write `internal/retro/collector_test.go` — mock state store, verify quantitative assembly for success/failure/retry scenarios [P]
- [ ] Task 6.2: Write `internal/retro/storage_test.go` — temp dir JSON read/write, list/filter, SQLite index CRUD [P]
- [ ] Task 6.3: Write `internal/retro/narrator_test.go` — mock adapter, verify prompt construction and JSON response parsing [P]
- [ ] Task 6.4: Write `internal/retro/generator_test.go` — end-to-end with mocks: collector → storage → narrator
- [ ] Task 6.5: Write `cmd/wave/commands/retro_test.go` — CLI flag parsing, subcommand routing, output formatting [P]
- [ ] Task 6.6: Write `internal/webui/handlers_retros_test.go` — HTTP handler request/response testing [P]
- [ ] Task 6.7: Integration test: pipeline executor generates retro after mock run, verify JSON file exists and has correct structure

## Phase 7: Polish

- [ ] Task 7.1: Ensure `wave clean` prunes old retro files and SQLite index entries
- [ ] Task 7.2: Add `.wave/retros/` to `.gitignore` if not already excluded
- [ ] Task 7.3: Run `go test -race ./...` and `golangci-lint run ./...` — fix any issues
