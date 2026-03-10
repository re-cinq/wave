# Tasks: Init Merge & Upgrade Workflow

**Feature**: #230 — Init Merge & Upgrade Workflow
**Generated**: 2026-03-04
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

---

## Phase 1: Setup

- [X] T001 [P1] Setup — Add `FileStatus`, `FileChangeEntry`, `ManifestAction`, `ManifestChangeEntry`, and `ChangeSummary` types to `cmd/wave/commands/init.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

- [X] T002 [P1] [US1] Foundational — Implement `computeChangeSummary()` in `cmd/wave/commands/init.go` that iterates all asset categories (personas, pipelines, contracts, prompts), compares each file against embedded defaults using `bytes.Equal`, and returns a `ChangeSummary` with file entries categorized as `new`, `preserved`, or `up_to_date`
- [X] T003 [P1] [US2] Foundational — Implement `computeManifestDiff()` in `cmd/wave/commands/init.go` that wraps `mergeManifests()` with change tracking, recording which keys were `added` (from defaults) vs `preserved` (user value kept), returning both the merged manifest and `[]ManifestChangeEntry`
- [X] T004 [P1] [US1] Foundational — Implement `displayChangeSummary()` in `cmd/wave/commands/init.go` that renders the `ChangeSummary` as a categorized table to stderr, grouping files by category (personas, pipelines, contracts, prompts) with status indicators, followed by manifest key changes

---

## Phase 3: User Story 1 — Pre-Merge Change Summary (P1)

- [X] T005 [P1] [US1] — Implement `confirmMerge()` in `cmd/wave/commands/init.go` that prompts for user confirmation using `bufio.Reader` from stdin, respecting `--yes` and `--force` flags to skip, and detecting non-interactive terminals via `isInitInteractive()` to require `--yes`/`--force` or abort with helpful message (FR-002, FR-014)
- [X] T006 [P1] [US1] — Implement `applyChanges()` in `cmd/wave/commands/init.go` that writes only "new" files from the `ChangeSummary` and writes the merged manifest, handling filesystem permission errors with clear messages per edge case 3
- [X] T007 [P1] [US1] — Refactor `runMerge()` in `cmd/wave/commands/init.go` to use the pre-mutation pattern: parse existing manifest (abort on parse failure per FR-013) → `computeChangeSummary()` → check `AlreadyUpToDate` (print "Already up to date" and exit per US1-AS4) → `displayChangeSummary()` → `confirmMerge()` → `applyChanges()` → `printMergeSuccess()`

---

## Phase 4: User Story 2 — Manifest Merge with Diff Preview (P1)

- [X] T008 [P1] [US2] — Integrate manifest diff display into `displayChangeSummary()` in `cmd/wave/commands/init.go` showing "Added" keys (new defaults) and "Preserved" keys (user values) with dot-path notation (e.g., `runtime.relay.token_threshold_percent`)

---

## Phase 5: User Story 3 — Consistent Force and Merge Flag Semantics (P2)

- [X] T009 [P2] [US3] — Update flag interaction logic in `runInit()` in `cmd/wave/commands/init.go` to ensure: `--merge --force` skips confirmation but prints summary to stderr (FR-007); `--merge --yes` behaves identically (FR-008); `--force` without `--merge` overwrites all (FR-009); plain `init` on existing project prompts for confirmation (FR-010)

---

## Phase 6: User Story 4 — Post-Upgrade Migration Verification (P2)

- [X] T010 [P2] [US4] — Update `printMergeSuccess()` in `cmd/wave/commands/init.go:509` to add `wave migrate up` as step 1 in the "Next steps" section, before the existing `wave validate` suggestion (per clarification C-4)
- [X] T011 [P2] [US4] [P] — Verify `wave migrate status` in `cmd/wave/commands/migrate.go:118` handles missing `.wave/state.db` gracefully (edge case 5: `NewMigrationRunner` should create a fresh database)

---

## Phase 7: User Story 5 — Integration Tests for Upgrade Path (P2)

- [X] T012 [P2] [US5] — Add `TestComputeChangeSummary` table-driven tests in `cmd/wave/commands/init_test.go` covering: all files new (fresh project), all files up-to-date, mixed states (some preserved, some new), empty persona file treated as preserved
- [X] T013 [P2] [US5] [P] — Add `TestComputeManifestDiff` table-driven tests in `cmd/wave/commands/init_test.go` covering: nested key additions, array atomic preservation, user key precedence, new subsection addition
- [X] T014 [P2] [US5] — Add `TestInitMergeUpgradeLifecycle` integration test in `cmd/wave/commands/init_test.go` covering: init → write custom persona → modify wave.yaml → init --merge → verify summary output → verify custom persona preserved → verify new files added → verify manifest merge correctness
- [X] T015 [P2] [US5] [P] — Add `TestInitMergeFlagCombinations` table-driven test in `cmd/wave/commands/init_test.go` testing all four combinations: `init` on existing (prompts), `--force` (overwrites), `--merge` (summary + prompt), `--merge --force` (summary, no prompt)
- [X] T016 [P2] [US5] [P] — Add `TestInitMergeEdgeCases` tests in `cmd/wave/commands/init_test.go` covering: malformed YAML parse error (FR-013), empty persona file preserved, read-only `.wave/` permission error, non-interactive terminal requires `--yes` (FR-014), already-up-to-date short circuit

---

## Phase 8: User Story 6 — Upgrade Workflow Documentation (P3)

- [X] T017 [P3] [US6] — Create `docs/guides/upgrade-guide.md` with step-by-step upgrade workflow: update binary → run `wave init --merge` → review change summary → confirm → run `wave migrate up` → run `wave validate` → verify setup, with expected output examples for each step (FR-016)

---

## Phase 9: Polish & Cross-Cutting Concerns

- [X] T018 [P] Polish — Run `go test ./cmd/wave/commands/ -run TestInitMerge` and `go test ./...` to verify all new and existing tests pass with zero regressions
- [X] T019 Polish — Run `go vet ./...` to verify no lint issues in changed files
