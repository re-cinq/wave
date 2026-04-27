# Implementation Plan — Merge `internal/deliverable` into `internal/state.OutcomeRecord`

## 1. Objective

Eliminate the parallel `internal/deliverable` in-memory tracker and consolidate all pipeline outcome tracking onto `internal/state.OutcomeRecord` and a single in-memory tracker that lives inside `internal/state`. One source of truth per concern.

## 2. Approach

`internal/deliverable` ships two concerns:

1. **Data model** — `Deliverable` struct, type constants, factory constructors, `String()` rendering, `IsTemporary()`.
2. **In-memory tracker** — `Tracker` with thread-safe add/get/format and outcome warnings.

`state.OutcomeRecord` already covers the persistent slice (`ID`, `RunID`, `StepID`, `Type`, `Label`, `Value`, `CreatedAt`) and is backed by SQLite (migration 21, table `pipeline_outcome`). It does not store `Description`, `Metadata`, or expose a tracker.

Strategy:

- **Extend** `state.OutcomeRecord` with `Description string` and `Metadata map[string]any` fields. Persist `Description` as TEXT and `Metadata` as JSON via a schema migration (#23).
- **Promote** `Deliverable.Type` constants to `state` as `OutcomeType` enum values (string-typed) so the type taxonomy stays unified.
- **Move** rendering helpers (`String()`, `IsTemporary()`, nerd-font detection) into `internal/state/outcome_render.go` as methods on `*OutcomeRecord` plus package-private helpers.
- **Move** tracker into `internal/state/outcome_tracker.go` as `OutcomeTracker` (the "Tracker" name is overloaded), wrapping the existing `state.Store` for persistence-on-add while keeping the in-memory cache for fast queries during a run.
- **Migrate** callers (`internal/pipeline/executor.go`, `internal/display/*`, `cmd/wave/commands/*`, tests) to import `internal/state` and use `state.OutcomeTracker`/`state.OutcomeRecord`.
- **Delete** `internal/deliverable/` entirely (package + tests) once callers compile.
- **Persistence wiring**: where the executor currently calls `RecordOutcome` separately from `deliverableTracker.Add*`, the new `OutcomeTracker.Add` performs both the in-memory cache and the SQLite insert. This kills the double-write pattern.

## 3. File Mapping

### Created

- `internal/state/outcome_tracker.go` — new in-memory + persistent `OutcomeTracker` ported from `deliverable.Tracker`.
- `internal/state/outcome_tracker_test.go` — port of `deliverable/tracker_test.go`.
- `internal/state/outcome_render.go` — nerd-font detection + `(*OutcomeRecord).String()` + `IsTemporary()` ported from `deliverable/types.go`.
- `internal/state/outcome_render_test.go` — port of `deliverable/types_test.go`.

### Modified

- `internal/state/types.go` — extend `OutcomeRecord` with `Description string`, `Metadata map[string]any`, plus `OutcomeType` constants (`OutcomeTypeFile`, `OutcomeTypePR`, `OutcomeTypeURL`, `OutcomeTypeDeployment`, `OutcomeTypeLog`, `OutcomeTypeContract`, `OutcomeTypeArtifact`, `OutcomeTypeBranch`, `OutcomeTypeIssue`, `OutcomeTypeOther`).
- `internal/state/store.go` — update `RecordOutcome` signature to accept description + metadata; update `scanOutcomeRows` to read them; update interface in `Store`.
- `internal/state/migration_definitions.go` — append migration #23: `ALTER TABLE pipeline_outcome ADD COLUMN description TEXT DEFAULT '';` + `ADD COLUMN metadata TEXT DEFAULT '';` (JSON).
- `internal/state/schema.sql` — reflect new columns for greenfield installs.
- `internal/pipeline/executor.go` — replace `deliverableTracker *deliverable.Tracker` with `outcomeTracker *state.OutcomeTracker`; rename `GetDeliverableTracker` → `GetOutcomeTracker`; replace all `deliverable.NewTracker`, `Add*`, `GetByType`, `AddOutcomeWarning` calls with state equivalents.
- `internal/pipeline/executor_test.go` — update imports + tracker calls.
- `internal/display/outcome.go` — replace imports; rename `AllDeliverables` → `AllOutcomes` (or keep field name with new type); update `BuildOutcome` signature.
- `internal/display/outcome_test.go` — update imports + factories.
- `internal/display/bubbletea_progress.go` — replace `deliverableTracker` field/setter; update `NewBubbleTeaProgressDisplay` signature; update `FormatByStep` call.
- `cmd/wave/commands/run.go` — update calls to `executor.GetOutcomeTracker()`, `btpd.SetOutcomeTracker(...)`.
- `cmd/wave/commands/resume.go` — same as above.
- `.golangci.yml` — drop any `internal/deliverable` exclusions if present.
- `.agents/pipelines/audit-duplicates.yaml`, `internal/defaults/pipelines/audit-duplicates.yaml` — remove `deliverable` from the audit target list now that it's merged.
- `docs/scope/wave-scope.md` — change `deliverable` verdict from "merge" to "merged into state.OutcomeRecord (closed via #1286)".
- `docs/reference/pipeline-schema.md`, `docs/reference/environment.md`, `docs/guide/outcomes.md`, `docs/guide/pipeline-outputs.md`, `docs/guide/chat-context.md`, `docs/architecture-audit.md`, `docs/changelog.md` — replace `internal/deliverable` references with `internal/state`.
- `docs/adr/006-cost-infrastructure.md`, `docs/adr/003-layered-architecture.md` — same.

### Deleted

- `internal/deliverable/types.go`
- `internal/deliverable/types_test.go`
- `internal/deliverable/tracker.go`
- `internal/deliverable/tracker_test.go`

### Untouched (historical specs)

- `specs/428-test-coverage-gaps/*`, `specs/264-docs-consistency-fixes/*`, `specs/191-array-outcome-extraction/*`, `specs/186-file-uri-paths/*`, `specs/120-pipeline-output-ux/*` — frozen historical artefacts; do not rewrite.

## 4. Architecture Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| AD-1 | New tracker lives in `internal/state` not a sibling package. | Issue mandates "single outcome model **backed by** `internal/state`". A separate `internal/outcometrack` would just rename the duplication. |
| AD-2 | Extend `OutcomeRecord` rather than create `OutcomeRecordExt`. | One struct, one source of truth. Adding nullable columns is cheap and preserves existing reads. |
| AD-3 | `Metadata` persisted as JSON TEXT, not relational table. | `Metadata` is heterogeneous (`pushed`, `push_error`, `remote_ref`, etc.) and small. JSON column matches existing pattern (e.g. `webhook.events`, `webhook.headers`). |
| AD-4 | `OutcomeTracker.Add` writes to both memory cache and SQLite. | Removes the existing double-write split between `deliverableTracker.Add*` and `state.RecordOutcome`. SQLite write is best-effort; in-memory authoritative for the run. |
| AD-5 | Rendering (`String()`, nerd-font detection) stays attached to `OutcomeRecord`. | It's a presentation method on the data — tracker-agnostic. Keeps `display/` thin. |
| AD-6 | Type constants renamed `Type*` → `OutcomeType*`. | Avoids collision with other `Type*` enums in `state` package and matches the existing `OutcomeRecord` naming. |
| AD-7 | No backward-compat shim layer in `internal/deliverable` re-exporting from `state`. | Pre-1.0; per Wave versioning policy, breaking internal package paths is allowed. |
| AD-8 | `outcome_warnings` lives on the tracker, not persisted. | Warnings are transient run-time advisories. Persistence would require a new table for no current consumer. |

## 5. Risks & Mitigations

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Migration #23 fails on existing DBs | Low | `ALTER TABLE ADD COLUMN` with `DEFAULT ''` is non-breaking; covered by `migrations_test.go`. |
| Hidden caller in non-`.go` file (yaml/scripts) | Medium | Already grepped; only audit pipelines reference the package name as a string. |
| Test ordering/race on shared tracker | Low | Lock semantics ported verbatim; `go test -race ./...` is required pre-PR. |
| Display package tests assume specific `Tracker` type | High | All `outcome_test.go` cases must be updated mechanically; covered in tasks Phase 2. |
| Outcome write failure in SQLite kills pipeline | Medium | Wrap insert in `OutcomeTracker.Add` such that DB error is logged but in-memory add still succeeds. |
| Metadata JSON unmarshal errors on legacy rows | Low | Empty default → unmarshal returns `nil` map; check `len(metadata)>0` before unmarshal. |

## 6. Testing Strategy

- **Unit tests (ported)**: `outcome_tracker_test.go` and `outcome_render_test.go` mirror the existing `deliverable/*_test.go` cases (add/dedupe, GetByStep, GetByType, FormatSummary, UpdateMetadata, OutcomeWarnings, IsTemporary, String rendering with/without nerd font).
- **Migration test**: extend `migrations_test.go` to assert migration #23 adds `description` and `metadata` columns and is reversible.
- **Persistence round-trip**: new test inserts via `OutcomeTracker.Add`, queries via `Store.GetOutcomes`, asserts `Description` and `Metadata` survive.
- **Integration**: `internal/pipeline/executor_test.go` already exercises `GetByType(deliverable.TypePR/Issue/Deployment/URL)` — these convert to state types and must continue to pass.
- **Display**: `internal/display/outcome_test.go` exercises BuildOutcome flows; all 12+ subtests must pass post-rename.
- **Smoke**: build binary, run `wave run gh-implement` (or any pipeline) on a tiny issue and verify outcomes display correctly in TUI + `wave runs view`.
- **Linters**: `go vet ./...`, `golangci-lint run`, `go build ./...`, `go test -race ./...` — all must be clean.
