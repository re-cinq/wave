# Tasks: Publish Wave Skills as Standalone SKILL.md Artifacts

**Feature**: #559 Skills Publish
**Generated**: 2026-03-24
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Data Model**: [data-model.md](data-model.md)

---

## Phase 1: Setup

- [X] T001 [P1] [Setup] Add skill publish error code constants to `cmd/wave/commands/errors.go`
  - Add `CodeSkillPublishFailed = "skill_publish_failed"` for publish operation failures
  - Add `CodeSkillValidationFailed = "skill_validation_failed"` for pre-publish validation failures
  - Add `CodeSkillAlreadyExists = "skill_already_exists"` for registry name conflicts
  - Follow existing constant pattern (lines 10-29 of errors.go)

## Phase 2: Foundational — Core Library

- [X] T002 [P1] [US1] [P] Create `internal/skill/classify.go` — SkillClassification type and classification functions
  - Define `SkillClassification` struct: `Name string`, `Tag string` (standalone|wave-specific|both), `WaveRefCount int`, `Warnings []string`, `SourcePath string`
  - Define `waveKeywords` slice: `wave`, `.wave/`, `pipeline`, `persona`, `wave.yaml`, `manifest`, `worktree`, `wave run`, `wave init`
  - Implement `ClassifySkill(skill Skill) SkillClassification` — count keyword occurrences in `skill.Body` (case-insensitive word boundary matching), apply thresholds: 0 → `standalone`, 1-10 → `both`, >10 → `wave-specific`
  - Add compliance warnings for missing optional fields (no `Description` → warning, no `License` → warning)
  - Implement `ClassifyAll(store Store) ([]SkillClassification, error)` — call `store.List()`, then `store.Read(name)` for full body access, classify each
  - Classification tag constants: `TagStandalone = "standalone"`, `TagWaveSpecific = "wave-specific"`, `TagBoth = "both"`

- [X] T003 [P1] [US4] [P] Create `internal/skill/digest.go` — SHA-256 content digest computation
  - Implement `ComputeDigest(skill Skill) (string, error)` — content-addressed hashing
  - Read raw SKILL.md bytes from `filepath.Join(skill.SourcePath, "SKILL.md")`
  - Hash SKILL.md bytes first via `sha256.New()` + `h.Write(skillMdBytes)`
  - Sort `skill.ResourcePaths` alphabetically, then for each: write delimiter `"\n---resource:<relpath>---\n"` + resource file bytes
  - Read resource files from `filepath.Join(skill.SourcePath, relpath)`
  - Return format: `"sha256:" + hex.EncodeToString(h.Sum(nil))`
  - Error if SKILL.md cannot be read; skip unreadable resource files with warning (don't fail)

- [X] T004 [P2] [US4] [P] Create `internal/skill/lockfile.go` — Lockfile persistence with atomic writes
  - Define `PublishRecord` struct: `Name string`, `Digest string`, `Registry string`, `URL string`, `PublishedAt time.Time` (all JSON-tagged)
  - Define `Lockfile` struct: `Version int`, `Published []PublishRecord` (JSON-tagged)
  - Implement `LoadLockfile(path string) (*Lockfile, error)` — read JSON file; return `&Lockfile{Version: 1}` for `os.IsNotExist`; return error for corrupt JSON
  - Implement `(*Lockfile) Save(path string) error` — marshal to indented JSON, write to `path + ".tmp"`, then `os.Rename()` for atomicity (FR-012)
  - Implement `(*Lockfile) FindByName(name string) *PublishRecord` — linear scan, return pointer to matching record or nil
  - Implement `(*Lockfile) Upsert(record PublishRecord)` — find by name and replace, or append if not found
  - Lockfile path convention: `.wave/skills.lock` (FR-005)

- [X] T005 [P3] [US5] [P] Extend `internal/skill/validate.go` with ValidationReport and ValidateForPublish
  - Define `ValidationIssue` struct: `Field string`, `Message string`
  - Define `ValidationReport` struct: `Errors []ValidationIssue`, `Warnings []ValidationIssue`
  - Implement `(*ValidationReport) Valid() bool` — returns `len(r.Errors) == 0`
  - Implement `ValidateForPublish(skill Skill) ValidationReport`:
    - Error if `skill.Name` is empty or fails `ValidateName()` (FR-013)
    - Error if `skill.Description` is empty
    - Warning if `skill.License` is empty
    - Warning if `skill.Compatibility` is empty
    - Warning if `len(skill.AllowedTools) == 0`
  - Reuse existing `ValidateName()` from `parse.go` for name format validation

## Phase 3: US1 — Skills Audit & Classification (P1)

- [X] T006 [P1] [US1] Add `wave skills audit` subcommand to `cmd/wave/commands/skills.go`
  - Create `newSkillsAuditCmd()` returning `*cobra.Command` with `--format json|table` flag
  - Implement `runSkillsAudit(cmd, format)`:
    1. Call `newSkillStore()` to get store
    2. Call `skill.ClassifyAll(store)` to get classifications
    3. Build `SkillAuditOutput` with `[]SkillAuditItem` and `AuditSummary`
  - Define `SkillAuditItem` struct: `Name`, `Classification`, `WaveRefCount`, `Warnings`, `Source` (all JSON-tagged)
  - Define `SkillAuditOutput` struct: `Skills []SkillAuditItem`, `Summary AuditSummary`
  - Define `AuditSummary` struct: `Total`, `Standalone`, `WaveSpecific`, `Both` (int, JSON-tagged)
  - Table rendering: columns NAME | CLASSIFICATION | WAVE REFS | WARNINGS | SOURCE
  - Summary line: "X standalone, Y wave-specific, Z both (N total)"
  - JSON mode: encode `SkillAuditOutput` to stdout
  - Register `newSkillsAuditCmd()` in `NewSkillsCmd()` via `cmd.AddCommand()`
  - Update `NewSkillsCmd()` Long description to include `audit` subcommand

- [X] T007 [P1] [US1] [P] Create `internal/skill/classify_test.go` — table-driven classification tests
  - Test `ClassifySkill` with standalone skill body (0 wave refs) → Tag == "standalone"
  - Test `ClassifySkill` with wave-specific skill body (>10 wave refs) → Tag == "wave-specific"
  - Test `ClassifySkill` with "both" skill body (1-10 wave refs) → Tag == "both"
  - Test `ClassifySkill` counts are accurate (verify WaveRefCount matches expected)
  - Test `ClassifySkill` with empty body → Tag == "standalone", WaveRefCount == 0
  - Test `ClassifyAll` with mock store containing mixed skills
  - Test `ClassifyAll` with empty store → returns empty slice, no error
  - Use `Skill{Name: "test", Description: "test", Body: "..."}` for test inputs

- [X] T008 [P1] [US1] Add audit subcommand tests to `cmd/wave/commands/skills_test.go`
  - Create temp directory with test skills (SKILL.md files with known wave ref counts)
  - Test `wave skills audit` table output contains expected columns and classification tags
  - Test `wave skills audit --format json` output is valid JSON matching `SkillAuditOutput` schema
  - Test audit with empty store returns "No skills found" message
  - Use `cobra.Command` test pattern consistent with existing skills_test.go tests

## Phase 4: US2 — Publish a Single Skill (P1)

- [X] T009 [P1] [US2] Create `internal/skill/publish.go` — Publisher struct and PublishOne workflow
  - Define `PublishOpts` struct: `Force bool`, `DryRun bool`, `Registry string`
  - Define `PublishResult` struct: `Name string`, `Success bool`, `URL string`, `Digest string`, `Warnings []string`, `Error string`, `Skipped bool`, `SkipReason string`
  - Define `Publisher` struct: `store Store`, `lockfilePath string`, `registryName string`, `lookPath lookPathFunc`
  - Implement `NewPublisher(store Store, lockfilePath, registryName string, lookPath lookPathFunc) *Publisher`
  - Implement `(*Publisher) PublishOne(ctx context.Context, name string, opts PublishOpts) PublishResult`:
    1. Read skill from store → fail with `skill_not_found` if missing
    2. `ValidateForPublish(skill)` → fail with errors if `!report.Valid()`, include warnings
    3. `ClassifySkill(skill)` → if wave-specific and !Force, return warning result (not error)
    4. `ComputeDigest(skill)` → set result.Digest
    5. Load lockfile → `FindByName(name)` → if digest matches, skip (idempotent, FR-007)
    6. If DryRun: return success result with "[dry-run]" note, no tessl call
    7. Exec `tessl publish <skill.SourcePath>` via `exec.CommandContext` with 30s timeout
    8. Parse stdout for published URL
    9. Load lockfile → `Upsert(PublishRecord{...})` → `Save()` (atomic update)
    10. Return success result with URL and digest
  - Use `lookPath` for testability (inject mock tessl binary path)
  - Return structured `PublishResult` on all code paths (never panic)

- [X] T010 [P1] [US2] Add `wave skills publish <name>` subcommand to `cmd/wave/commands/skills.go`
  - Create `newSkillsPublishCmd()` returning `*cobra.Command`
  - Args: `<name>` (positional, required unless `--all`)
  - Flags: `--force` (bypass wave-specific warning), `--dry-run` (validate only), `--registry <name>` (default: "tessl"), `--format json|table`
  - Define `PublishResultItem` struct: `Name`, `Status` (published|skipped|failed), `URL`, `Digest`, `Reason`, `Warnings` (JSON-tagged)
  - Define `SkillPublishOutput` struct: `Results []PublishResultItem`, `Lockfile string`
  - Implement `runSkillsPublish(cmd, name, format, opts)`:
    1. Create `Publisher` with `newSkillStore()`, lockfile path `.wave/skills.lock`, registry from flag
    2. Call `publisher.PublishOne(ctx, name, opts)`
    3. Render result as table (status symbol + name + URL/reason) or JSON
  - Table status symbols: `✓ Published`, `⊘ Skipped`, `✗ Failed` (using display.Formatter)
  - Error handling: wrap publish errors with `CodeSkillPublishFailed` or `CodeSkillValidationFailed`
  - Register `newSkillsPublishCmd()` in `NewSkillsCmd()`

- [X] T011 [P1] [US2] [P] Create `internal/skill/publish_test.go` — PublishOne unit tests with mock tessl
  - Create helper `createMockTessl(t, tmpDir, stdout, exitCode)` that writes a shell script acting as fake `tessl`
  - Test successful publish: mock tessl returns URL, verify lockfile updated
  - Test validation failure: skill missing description → result.Success == false, result.Error contains field info
  - Test wave-specific warning: wave-specific skill without Force → result.Skipped == true
  - Test wave-specific force: wave-specific skill with Force → proceeds to publish
  - Test dry-run: DryRun == true → no tessl execution, digest computed, lockfile NOT updated
  - Test idempotent skip: lockfile has matching digest → result.Skipped == true, SkipReason == "up-to-date"
  - Test tessl not found: lookPath returns error → result includes DependencyError
  - Test tessl failure: mock tessl exits non-zero → result.Success == false with stderr in error
  - Use `t.TempDir()` for workspace isolation

- [X] T012 [P1] [US2] Add publish subcommand tests to `cmd/wave/commands/skills_test.go`
  - Test `wave skills publish golang` table output shows published status
  - Test `wave skills publish golang --format json` output matches SkillPublishOutput schema
  - Test `wave skills publish golang --dry-run` shows dry-run indicator
  - Test `wave skills publish` with no name arg → error message
  - Test `wave skills publish nonexistent` → skill_not_found error

## Phase 5: US3 — Batch Publish All Standalone Skills (P2)

- [X] T013 [P2] [US3] Add PublishAll() to `internal/skill/publish.go`
  - Implement `(*Publisher) PublishAll(ctx context.Context, opts PublishOpts) ([]PublishResult, error)`:
    1. Call `ClassifyAll(p.store)` to get all classifications
    2. Filter: keep skills where Tag == "standalone" or Tag == "both"
    3. Skip wave-specific skills → add to results as Skipped with reason "wave-specific"
    4. For each eligible skill: call `PublishOne(ctx, name, opts)`
    5. Continue on individual failure (don't abort batch)
    6. Return all results (successes + skips + failures)
  - Lockfile is updated after each successful publish (not batched) for crash safety

- [X] T014 [P2] [US3] Wire --all flag in publish subcommand and add batch tests
  - Update `runSkillsPublish` in `cmd/wave/commands/skills.go`:
    - If `--all` set: call `publisher.PublishAll(ctx, opts)` instead of `PublishOne`
    - Validate mutual exclusion: `--all` with positional name → error
    - Table output: list each skill result + summary line ("X published, Y skipped, Z failed")
  - Add `PublishAll` tests to `internal/skill/publish_test.go`:
    - Test with mixed skills: standalone published, wave-specific skipped
    - Test with one failure: other skills still publish
    - Test idempotent batch: all up-to-date → all skipped
  - Add CLI test to `cmd/wave/commands/skills_test.go`:
    - Test `wave skills publish --all` with mock store

## Phase 6: US4 — Content Integrity & Verify (P2)

- [X] T015 [P2] [US4] Add `wave skills verify` subcommand to `cmd/wave/commands/skills.go`
  - Create `newSkillsVerifyCmd()` returning `*cobra.Command` with `--format json|table`
  - Define `SkillVerifyItem` struct: `Name`, `Status` (ok|modified|missing), `ExpectedDigest`, `ActualDigest` (JSON-tagged)
  - Define `SkillVerifyOutput` struct: `Results []SkillVerifyItem`, `Summary VerifySummary`
  - Define `VerifySummary` struct: `Total`, `OK`, `Modified`, `Missing` (int, JSON-tagged)
  - Implement `runSkillsVerify(cmd, format)`:
    1. Load lockfile from `.wave/skills.lock` → if missing, print "No published skills" and return
    2. Create store via `newSkillStore()`
    3. For each `PublishRecord` in lockfile:
       a. Try `store.Read(record.Name)` → if error, mark "missing"
       b. `ComputeDigest(skill)` → compare with record.Digest
       c. If match → "ok", if mismatch → "modified"
    4. Build `SkillVerifyOutput` with results and summary
  - Table: columns NAME | STATUS | EXPECTED | ACTUAL
  - Table status coloring: ok → green, modified → yellow, missing → red (via display.Formatter)
  - Summary line: "X ok, Y modified, Z missing"
  - Register `newSkillsVerifyCmd()` in `NewSkillsCmd()`

- [X] T016 [P2] [US4] [P] Create `internal/skill/lockfile_test.go` — lockfile CRUD and atomicity tests
  - Test `LoadLockfile` with valid JSON file → correct struct populated
  - Test `LoadLockfile` with nonexistent file → returns empty Lockfile with Version 1
  - Test `LoadLockfile` with corrupt JSON → returns parse error
  - Test `Save` writes valid JSON readable by `LoadLockfile` (round-trip)
  - Test `Save` creates intermediate directories if lockfile parent dir missing
  - Test `FindByName` returns matching record or nil
  - Test `Upsert` inserts new record when name not found
  - Test `Upsert` replaces existing record when name matches
  - Test atomic write: verify `.tmp` file is cleaned up after successful rename
  - Use `t.TempDir()` for isolation

- [X] T017 [P2] [US4] [P] Create `internal/skill/digest_test.go` — content digest tests
  - Test digest of skill with SKILL.md only (no resource files) → valid sha256: prefixed hash
  - Test digest with resource files → includes resource content in hash
  - Test determinism: call ComputeDigest twice on same skill → identical digest
  - Test different content → different digest
  - Test resource file sort order: same files in different ResourcePaths order → same digest
  - Test missing SKILL.md → returns error
  - Create temp skill directories with known content for predictable digest values

- [X] T018 [P2] [US4] Add verify subcommand tests to `cmd/wave/commands/skills_test.go`
  - Test `wave skills verify` with matching digests → all "ok" status
  - Test `wave skills verify` with modified skill → "modified" status
  - Test `wave skills verify` with skill deleted from disk → "missing" status
  - Test `wave skills verify` with no lockfile → "No published skills" message
  - Test `wave skills verify --format json` → valid JSON matching SkillVerifyOutput

## Phase 7: US5 — SKILL.md Spec Compliance Validation (P3)

- [X] T019 [P3] [US5] Extend `internal/skill/validate_test.go` with publish validation tests
  - Test `ValidateForPublish` with fully valid skill → report.Valid() == true, no errors
  - Test missing name → report.Errors contains "name" field issue
  - Test missing description → report.Errors contains "description" field issue
  - Test invalid name format (uppercase, special chars) → report.Errors contains name format issue
  - Test missing license → report.Warnings contains "license" warning, report.Valid() == true
  - Test missing compatibility → report.Warnings contains "compatibility" warning
  - Test missing allowed-tools → report.Warnings contains "allowed-tools" warning
  - Test multiple issues → all errors and warnings collected (not fail-fast)
  - Use table-driven test pattern consistent with existing validate_test.go

## Phase 8: Distributable Skills Directory

- [X] T020 [P2] [US2] Populate `skills/` directory at repo root with standalone-classified skills
  - Copy from `.claude/skills/<name>/` to `skills/<name>/` for each eligible skill
  - Include SKILL.md and resource subdirectories (scripts/, references/, assets/)
  - Eligible skills (standalone + both, per research.md): `agentic-coding`, `bmad`, `cli`, `ddd`, `gh-cli`, `golang`, `opsx`, `software-architecture`, `software-design`, `spec-driven-development`, `speckit`, `tui` (12 skills)
  - Exclude: `wave` (wave-specific, 139 wave refs)
  - Verify each copied SKILL.md parses correctly
  - Directory structure: `skills/<name>/SKILL.md` (one directory per skill)

## Phase 9: Polish & Cross-cutting

- [X] T021 [P1] [US2,US4] Create `internal/skill/integration_publish_test.go` — end-to-end roundtrip test
  - Create temp directory with 3 test skills (1 standalone, 1 both, 1 wave-specific)
  - Test audit: verify correct classifications for all 3
  - Test publish with mock tessl: standalone publishes, wave-specific is skipped
  - Test lockfile: verify PublishRecord created with correct digest
  - Test idempotent re-publish: same content → skipped as "up-to-date"
  - Test modify skill content → verify produces different digest
  - Test verify: modified skill detected, unmodified skill ok
  - Use build tag `//go:build integration` or skip if mock tessl setup fails

- [X] T022 [P3] [Setup] Update `NewSkillsCmd()` Long description in `cmd/wave/commands/skills.go`
  - Add `audit`, `publish`, `verify` to the subcommand list in the Long description
  - Ensure `wave skills --help` output reflects the full subcommand set
  - Follows existing pattern: list subcommand name + short description

---

## Dependency Graph

```
T001 (error codes) ──────────────────────────┐
                                              │
T002 (classify) ─────┬───→ T006 (audit cmd) ─┤
                     │     T007 (classify test)│
                     │                         │
T003 (digest) ──────┬┤                         │
T004 (lockfile) ────┤├───→ T009 (Publisher) ──→ T010 (publish cmd) ──→ T014 (--all)
T005 (validate) ────┘│     T011 (publish test)  T012 (publish test)    T013 (PublishAll)
                     │
                     ├───→ T015 (verify cmd)
                     │     T016 (lockfile test)
                     │     T017 (digest test)
                     │     T018 (verify test)
                     │
                     └───→ T020 (skills dir)
                           T021 (integration)
```

## Summary

| Phase | Tasks | Parallel | Key Deliverable |
|-------|-------|----------|----------------|
| 1: Setup | T001 | — | Error code constants |
| 2: Foundational | T002-T005 | 4 | Core library: classify, digest, lockfile, validate |
| 3: US1 Audit (P1) | T006-T008 | 1 | `wave skills audit` command |
| 4: US2 Publish (P1) | T009-T012 | 1 | `wave skills publish <name>` command |
| 5: US3 Batch (P2) | T013-T014 | — | `--all` flag for batch publish |
| 6: US4 Verify (P2) | T015-T018 | 2 | `wave skills verify` command |
| 7: US5 Validation (P3) | T019 | — | Publish validation test coverage |
| 8: Skills Dir | T020 | — | `skills/` distributable directory |
| 9: Polish | T021-T022 | — | Integration test + help text |
