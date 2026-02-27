# Tasks: Remove Backwards-Compatibility Shims

**Feature**: 115-remove-compat-shims
**Generated**: 2026-02-20
**Spec**: `specs/115-remove-compat-shims/spec.md`
**Plan**: `specs/115-remove-compat-shims/plan.md`

---

## Phase 1 — Setup

- [X] T001 [P1] Pre-flight baseline: run `go test -race ./...` and `go vet ./...` to confirm green before changes — `./...`

---

## Phase 2 — Foundational: StrictMode Field Removal (US1)

> **User Story 1**: Remove Deprecated Contract Field (P1)
> All StrictMode references must be eliminated before any other cleanup, since downstream tasks (US2) also touch `contract/` files.

- [X] T002 [P1] [US1] Remove `StrictMode` field from `ContractConfig` struct — `internal/contract/contract.go:18`
- [X] T003 [P1] [US1] Remove `StrictMode` assignment in `buildContractConfig` — `internal/pipeline/executor.go:675`
- [X] T004 [P1] [US1] Replace `contractCfg.StrictMode` with `contractCfg.MustPass` in soft-failure check — `internal/pipeline/executor.go:700`
- [X] T005 [P1] [US1] Remove the `if !cfg.MustPass && cfg.StrictMode` fallback block — `internal/contract/jsonschema.go:266-267`
- [X] T006 [P1] [US1] Update `StrictMode` comment to reference `MustPass` — `internal/contract/jsonschema.go:333`
- [X] T007 [P1] [US1] Replace `if !cfg.StrictMode` with `if !cfg.MustPass` — `internal/contract/jsonschema.go:346`
- [X] T008 [P1] [US1] Replace `if cfg.StrictMode` with `if cfg.MustPass` — `internal/contract/typescript.go:25`
- [X] T009 [P1] [US1] Update all `StrictMode: true/false` test fixtures to `MustPass` — `internal/contract/contract_test.go:214,246,260,269`
- [X] T010 [P1] [US1] Update all `StrictMode: true/false` test fixtures to `MustPass` — `internal/contract/typescript_test.go:29,38,117,130,248`
- [X] T011 [P1] [US1] Run `go test ./internal/contract/... ./internal/pipeline/...` to verify StrictMode removal — `internal/contract/`, `internal/pipeline/`

---

## Phase 3 — Foundational: IsTypeScriptAvailable & Legacy Extraction Removal (US2)

> **User Story 2**: Collapse Legacy JSON/YAML Extraction Fallbacks (P1)
> Also includes FR-002 (IsTypeScriptAvailable wrapper removal) since it touches the same package and has no separate story.

- [X] T012 [P1] [US2] Remove `IsTypeScriptAvailable()` function (lines 95-100) — `internal/contract/typescript.go`
- [X] T013 [P1] [US2] [P] Update caller at test to use `CheckTypeScriptAvailability()` — `internal/contract/contract_test.go:252,273`
- [X] T014 [P1] [US2] [P] Update callers at test to use `CheckTypeScriptAvailability()` — `internal/contract/typescript_test.go:46,100,188`
- [X] T015 [P1] [US2] Delete `TestIsTypeScriptAvailable` test function — `internal/contract/typescript_test.go:317-327`
- [X] T016 [P1] [US2] Remove `extractJSONFromTextLegacy()` method and update `ExtractJSONFromText()` to return recovery parser error directly — `internal/contract/json_cleaner.go:80-148`
- [X] T017 [P1] [US2] Remove `extractYAMLLegacy()` function and update fallback to return error requiring `--- PIPELINE ---` marker — `internal/pipeline/meta.go:577-630`
- [X] T018 [P1] [US2] Run `go test ./internal/contract/... ./internal/pipeline/...` to verify extraction removal — `internal/contract/`, `internal/pipeline/`

---

## Phase 4 — Migration & State Cleanup (US3, US4)

> **User Story 3**: Remove Migration Down Paths (P2)
> **User Story 4**: Remove Legacy State Store Fallback (P2)
> These are grouped because both modify the `state` package and can be developed together.

- [X] T019 [P2] [US3] [P] Set all 6 `Down:` SQL values to empty string `""` — `internal/state/migration_definitions.go`
- [X] T020 [P2] [US3] [P] Update `wave migrate down` help text to state rollback is unsupported and remove confirmation prompt — `cmd/wave/commands/migrate.go:72-114`
- [X] T021 [P2] [US4] Remove `go:embed schema.sql` directive and `schemaFS` variable — `internal/state/store.go:15-17`
- [X] T022 [P2] [US4] Remove `"embed"` import — `internal/state/store.go:6`
- [X] T023 [P2] [US4] Replace dual-path initialization with migrations-only; error when `WAVE_MIGRATION_ENABLED=false` — `internal/state/store.go:150-165`
- [X] T024 [P2] [US4] Delete the legacy schema file — `internal/state/schema.sql`
- [X] T025 [P2] [US3+US4] Run `go test ./internal/state/... ./cmd/wave/...` to verify migration and state changes — `internal/state/`, `cmd/wave/`

---

## Phase 5 — Legacy Workspace Lookup Removal (US6)

> **User Story 6**: Collapse Legacy Workspace Directory Lookup (P3)

- [X] T026 [P3] [US6] Remove the exact-name directory fallback (`os.Stat` + prepend block) — `internal/pipeline/resume.go:187-189`
- [X] T027 [P3] [US6] Run `go test ./internal/pipeline/...` to verify resume still works — `internal/pipeline/`

---

## Phase 6 — Comment Cleanup (US5)

> **User Story 5**: Clean Up Legacy Comments and Labels (P3)
> All tasks in this phase are parallelizable since they touch different files.

- [X] T028 [P3] [US5] [P] Update `WorkspaceConfig.Type` comment from `empty for legacy directory` to `empty for default directory workspace` — `internal/pipeline/types.go:77`
- [X] T029 [P3] [US5] [P] Update worktree comment from `legacy behavior` to `default` — `internal/worktree/worktree.go:92`
- [X] T030 [P3] [US5] [P] Update comment from `Handle legacy template variables` to `Short-form template variables (primary format used by pipeline YAML)` — `internal/pipeline/context.go:75`
- [X] T031 [P3] [US5] [P] Update comment from `// Not tracked in legacy state store` to `// Not available from pipeline_state record` — `internal/pipeline/executor.go:1454`
- [X] T032 [P3] [US5] Scan for any remaining stale "legacy", "backward compat", "deprecated" comments referencing removed functionality in Go source — `internal/`

---

## Phase 7 — Verification & Polish

- [X] T033 [P1] Run `go vet ./...` and confirm zero issues (FR-014) — `./...`
- [X] T034 [P1] Run `go test -race ./...` and confirm zero failures (FR-013) — `./...`
- [X] T035 [P1] Verify net negative LOC by reviewing `git diff --stat` (SC-004, SC-006) — `./...`
- [X] T036 [P1] Verify all pipeline YAML configs in `.wave/pipelines/` and `internal/defaults/pipelines/` still function without modification (SC-007) — `.wave/pipelines/`, `internal/defaults/pipelines/`
