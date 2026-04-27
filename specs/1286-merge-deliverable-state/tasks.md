# Work Items

## Phase 1: Setup & Schema

- [ ] 1.1: Add `Description string` and `Metadata map[string]any` fields to `state.OutcomeRecord` in `internal/state/types.go`.
- [ ] 1.2: Add `OutcomeType` string-typed constants (`OutcomeTypeFile`, `OutcomeTypeURL`, `OutcomeTypePR`, `OutcomeTypeDeployment`, `OutcomeTypeLog`, `OutcomeTypeContract`, `OutcomeTypeArtifact`, `OutcomeTypeBranch`, `OutcomeTypeIssue`, `OutcomeTypeOther`) in `internal/state/types.go`.
- [ ] 1.3: Append migration #23 in `internal/state/migration_definitions.go` adding `description TEXT DEFAULT ''` and `metadata TEXT DEFAULT ''` columns to `pipeline_outcome`. Reflect in `internal/state/schema.sql`.
- [ ] 1.4: Update `RecordOutcome` signature in `internal/state/store.go` (and `Store` interface) to accept `description string, metadata map[string]any`; persist metadata via `json.Marshal`. Update `scanOutcomeRows` to populate the new fields.
- [ ] 1.5: Extend `migrations_test.go` to assert migration #23 columns exist and the down migration reverts cleanly.

## Phase 2: Port Tracker & Rendering

- [ ] 2.1: Create `internal/state/outcome_render.go` — port `hasNerdFont`, `(*OutcomeRecord).String()`, `(*OutcomeRecord).IsTemporary()` from `internal/deliverable/types.go`. [P]
- [ ] 2.2: Create `internal/state/outcome_tracker.go` — port `Tracker` as `OutcomeTracker` (mu, in-memory cache, optional `Store` for persistence-on-add, `outcomeWarnings`). Provide constructors `NewOutcomeTracker(pipelineID string, store Store)` and helpers (`AddFile`, `AddURL`, `AddPR`, `AddDeployment`, `AddLog`, `AddContract`, `AddArtifact`, `AddBranch`, `AddIssue`, `AddWorkspaceFiles`, `Add`, `GetAll`, `GetByStep`, `GetByType`, `Count`, `FormatSummary`, `FormatByStep`, `GetLatestForStep`, `UpdateMetadata`, `AddOutcomeWarning`, `OutcomeWarnings`, `SetPipelineID`).
- [ ] 2.3: Convert factory functions (`NewFileDeliverable`, etc.) into either `*OutcomeRecord` constructors or fold them into `Add*` helpers on the tracker. Prefer the latter for fewer exported names. [P]
- [ ] 2.4: Port `internal/deliverable/types_test.go` to `internal/state/outcome_render_test.go`. [P]
- [ ] 2.5: Port `internal/deliverable/tracker_test.go` to `internal/state/outcome_tracker_test.go`, including a new test asserting `Add` writes through to a real `Store` (use `NewMemoryStore` or in-process SQLite).

## Phase 3: Migrate Callers

- [ ] 3.1: `internal/pipeline/executor.go` — swap `deliverableTracker *deliverable.Tracker` → `outcomeTracker *state.OutcomeTracker`. Update `NewDefaultPipelineExecutor`, resume constructor, `runPipeline` lazy-init, and all `e.deliverableTracker.*` call sites (~25 lines). Rename `GetDeliverableTracker` → `GetOutcomeTracker`. Drop the separate `e.store.RecordOutcome(...)` calls now subsumed by tracker.
- [ ] 3.2: `internal/pipeline/executor_test.go` — update imports and `tracker.GetByType(deliverable.Type*)` → `tracker.GetByType(state.OutcomeType*)`.
- [ ] 3.3: `internal/display/outcome.go` — update `import`, `AllDeliverables` field type, `isOutcomeWorthy`, `filterArtifacts`, `BuildOutcome` signature.
- [ ] 3.4: `internal/display/outcome_test.go` — update imports and constructors. [P]
- [ ] 3.5: `internal/display/bubbletea_progress.go` — rename `deliverableTracker` field, `SetDeliverableTracker` → `SetOutcomeTracker`, update constructor signature. [P]
- [ ] 3.6: `cmd/wave/commands/run.go` — update `executor.GetOutcomeTracker()` and `btpd.SetOutcomeTracker(...)` call sites. [P]
- [ ] 3.7: `cmd/wave/commands/resume.go` — same renames. [P]

## Phase 4: Delete & Cleanup

- [ ] 4.1: Delete `internal/deliverable/types.go`, `types_test.go`, `tracker.go`, `tracker_test.go`. Remove the directory.
- [ ] 4.2: Strip `deliverable` from audit-duplicates pipelines: `.agents/pipelines/audit-duplicates.yaml`, `internal/defaults/pipelines/audit-duplicates.yaml`. [P]
- [ ] 4.3: Drop any `internal/deliverable` mention in `.golangci.yml`. [P]
- [ ] 4.4: Run `go mod tidy` if needed (no module changes expected).

## Phase 5: Docs

- [ ] 5.1: Update scope verdict in `docs/scope/wave-scope.md` — replace "merge" entry for `deliverable` with "merged into `state.OutcomeRecord` via #1286". [P]
- [ ] 5.2: Replace `internal/deliverable` references in `docs/reference/pipeline-schema.md`, `docs/reference/environment.md`, `docs/guide/outcomes.md`, `docs/guide/pipeline-outputs.md`, `docs/guide/chat-context.md`, `docs/architecture-audit.md`, `docs/changelog.md`. [P]
- [ ] 5.3: Update `docs/adr/006-cost-infrastructure.md` and `docs/adr/003-layered-architecture.md` to point at `state.OutcomeRecord`. [P]
- [ ] 5.4: Add changelog entry under "## Unreleased" describing the merge.

## Phase 6: Validation

- [ ] 6.1: `go build ./...` clean.
- [ ] 6.2: `go vet ./...` clean.
- [ ] 6.3: `go test -race ./...` passes.
- [ ] 6.4: `golangci-lint run` clean.
- [ ] 6.5: Build wave binary; run a small pipeline end-to-end; confirm outcome summary still rendered correctly in TUI and `wave runs view`.
- [ ] 6.6: Confirm DB migration applies cleanly against an existing state DB (`wave runs ls` against pre-existing `~/.wave/state.db`).
