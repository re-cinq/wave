# Tasks: Comprehensive Test Coverage and Documentation for Skill Management System

**Feature**: `specs/387-skill-test-docs`
**Branch**: `387-skill-test-docs`
**Generated**: 2026-03-14

## Phase 1: Setup

- [X] T001 [P1] [Setup] Measure current test coverage baseline with `go test -coverprofile=baseline.out ./internal/skill/...` and record numbers for gap comparison — `internal/skill/`

## Phase 2: Foundational — Store & Parse Test Gaps (US1, FR-001, FR-002)

- [X] T002 [P1] [US1] Add `TestDirectoryStoreConcurrency` — launch 10 goroutines doing Write+Delete on distinct skill names concurrently, verify no race with `-race` flag (US1-5) — `internal/skill/store_test.go`
- [X] T003 [P1] [US1] Add `TestParseCRLF` — parse a full SKILL.md with `\r\n` line endings, verify all fields (name, description, license, metadata, allowed-tools, body) extracted correctly (US1-1 + edge case) — `internal/skill/store_test.go`
- [X] T004 [P1] [US1] Add `TestSerializeCRLFRoundTrip` — create a Skill with CRLF body content, serialize, parse back, verify data preserved with LF normalization — `internal/skill/store_test.go`

## Phase 3: CLI Adapter Test Gaps (US2, FR-003)

- [X] T005 [P1] [US2] Add `TestBMADAdapterTimeout` — mirror existing `TestTesslAdapterTimeout` pattern for BMAD adapter with `lookPath` returning a path and expired context (US2-3) — `internal/skill/source_cli_test.go`
- [X] T006 [P] [P1] [US2] Add `TestOpenSpecAdapterTimeout` — mirror existing `TestTesslAdapterTimeout` pattern for OpenSpec adapter (US2-3) — `internal/skill/source_cli_test.go`
- [X] T007 [P] [P1] [US2] Add `TestSpecKitAdapterTimeout` — mirror existing `TestTesslAdapterTimeout` pattern for SpecKit adapter (US2-3) — `internal/skill/source_cli_test.go`
- [X] T008 [P1] [US2] Add `TestCLIAdapterStderrCapture` — verify that when a CLI adapter command fails, the error message includes "stderr:" substring with the captured stderr content (US2-4) — `internal/skill/source_cli_test.go`

## Phase 4: ProvisionFromStore Test Gaps (US4, FR-005)

- [X] T009 [P2] [US4] Add `TestProvisionFromStore_AllResources` — create a skill with files in all 3 resource dirs (scripts/, references/, assets/), provision to workspace, verify all resource files exist at correct paths under `.wave/skills/<name>/` (US4-1) — `internal/skill/provision_test.go`
- [X] T010 [P2] [US4] Add `TestProvisionFromStore_ContentMatch` — verify SKILL.md Body written to workspace matches original skill Body content exactly (US6-2) — `internal/skill/provision_test.go`
- [X] T011 [P2] [US4] Add `TestProvisionFromStore_IsolatedDirs` — provision 3 skills to same workspace, verify each occupies its own `.wave/skills/<name>/` subdirectory with no cross-contamination of files (US4-4) — `internal/skill/provision_test.go`

## Phase 5: Hierarchical Merge Traceability (US3, FR-004)

- [X] T012 [P1] [US3] Add SC-005 traceability comments to existing `TestResolveSkills` test cases mapping each sub-test to its acceptance criterion (US3-1 through US3-5) — no new test functions needed, resolve_test.go already at 100% coverage — `internal/skill/resolve_test.go`

## Phase 6: CLI Command Test Gaps (US5, FR-006, FR-012)

- [X] T013 [P2] [US5] Add `TestSkillsHelpOutput` — create the skills command, execute with `--help` flag, verify output includes descriptions for all 5 subcommands (list, install, remove, search, sync) (SC-007) — `cmd/wave/commands/skills_test.go`
- [X] T014 [P] [P2] [US5] Add `TestParseTesslSearchOutput` — unit test `parseTesslSearchOutput` with sample tab/space-separated output including 0, 1, and 3+ field lines, verify parsed `SkillSearchResult` slices — `cmd/wave/commands/skills_test.go`
- [X] T015 [P] [P2] [US5] Add `TestParseTesslSyncOutput` — unit test `parseTesslSyncOutput` with sample output including "installed golang", "updated spec-kit", "warning: foo", empty lines, verify synced and warnings slices — `cmd/wave/commands/skills_test.go`
- [X] T016 [P2] [US5] Add `TestClassifySkillError` — verify all error code mappings: `DependencyError` → `CodeSkillDependencyMissing`, `ErrNotFound` → `CodeSkillNotFound`, unknown prefix string → `CodeSkillSourceError` — `cmd/wave/commands/skills_test.go`

## Phase 7: Integration Tests (US6, FR-007)

- [X] T017 [P2] [US6] Add `TestSkillLifecycle_FileAdapter` — end-to-end test: create temp dir with valid SKILL.md, install via `FileAdapter` into real `DirectoryStore`, List to verify appears, `ProvisionFromStore` to workspace, verify SKILL.md body content, Delete, verify gone (SC-003, US6-1, US6-2) — `internal/skill/integration_test.go`
- [X] T018 [P2] [US6] Add `TestSkillLifecycle_MultiSource` — install two skills from different `file://` paths into a 2-source `DirectoryStore`, List, verify both appear with correct metadata and source paths (US6-3) — `internal/skill/integration_test.go`

## Phase 8: Documentation — Skill Authoring Guide (US7, FR-009)

- [X] T019 [P3] [US7] Create `docs/guide/skills.md` — skill authoring guide covering: SKILL.md format with all frontmatter fields (name, description, license, compatibility, metadata, allowed-tools) with examples, resource directories (scripts/, references/, assets/), naming conventions (`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, max 64 chars), and best practices — `docs/guide/skills.md`

## Phase 9: Documentation — Configuration Guide (US8, FR-010)

- [X] T020 [P3] [US8] Create `docs/guide/skill-configuration.md` — wave.yaml configuration guide covering: global skills declaration, persona-scoped skills, pipeline-scoped skills, precedence rules (pipeline > persona > global) with worked examples, `commands_glob` pattern, and deduplication behavior — `docs/guide/skill-configuration.md`

## Phase 10: Documentation — Ecosystem Guide (US8, FR-011)

- [X] T021 [P3] [US8] Create `docs/guide/skill-ecosystems.md` — ecosystem integration guide covering: Tessl (prerequisites: `npm i -g @tessl/cli`, install/search/sync commands), BMAD (prerequisites: npx, install), OpenSpec (prerequisites: `npm i -g @openspec/cli`, install), SpecKit (prerequisites: `npm i -g @speckit/cli`, install), GitHub (format `github:owner/repo`, install), File (format `file:./path`, install), URL (format `https://...`, install) — `docs/guide/skill-ecosystems.md`

## Phase 11: Verification & Validation

- [X] T022 [P1] [Verify] Run `go test -race ./internal/skill/... ./cmd/wave/commands/...` — verify zero failures (SC-001, FR-008) — `internal/skill/`, `cmd/wave/commands/`
- [X] T023 [P1] [Verify] Run `go test -coverprofile=cov.out ./internal/skill/...` — verify ≥80% line coverage (SC-002) — `internal/skill/` — **Result: 80.5%**
- [X] T024 [P1] [Verify] Grep for `t.Skip()` in all new test files — verify zero occurrences without a linked GitHub issue (SC-006) — `internal/skill/*_test.go`, `cmd/wave/commands/skills_test.go`
- [X] T025 [P1] [Verify] Verify SC-005 traceability: every acceptance criterion from spec has a corresponding named test function per the traceability table in plan.md — all test files

## Acceptance Criteria Traceability (SC-005)

| Acceptance Criterion | Task | Test Function | File |
|---------------------|------|---------------|------|
| US1-1: valid frontmatter parsed | existing | TestParse/"valid with all fields" | store_test.go |
| US1-2: missing required → ParseError | existing | TestParse/"missing name","missing description" | store_test.go |
| US1-3: empty body accepted | existing | TestParse/"valid with empty body" | store_test.go |
| US1-4: multi-source precedence | existing | TestMultiSourceResolution/"higher precedence shadows" | store_test.go |
| US1-5: concurrent access race-free | T002 | TestDirectoryStoreConcurrency | store_test.go |
| US2-1: missing CLI → DependencyError | existing | TestTesslAdapterMissingDependency et al. | source_cli_test.go |
| US2-2: mocked CLI → skills written | existing | TestParseAndWriteSkills/"valid skills" | source_cli_test.go |
| US2-3: timeout → error | T005-T007 | TestBMADAdapterTimeout, TestOpenSpecAdapterTimeout, TestSpecKitAdapterTimeout | source_cli_test.go |
| US2-4: stderr in diagnostics | T008 | TestCLIAdapterStderrCapture | source_cli_test.go |
| US3-1: global-only resolved | T012 | TestResolveSkills/"global only sorted" | resolve_test.go |
| US3-2: global+persona deduplicated | T012 | TestResolveSkills/"duplicate across scopes" | resolve_test.go |
| US3-3: all scopes merged sorted | T012 | TestResolveSkills/"all three scopes merged" | resolve_test.go |
| US3-4: all nil → nil | T012 | TestResolveSkills/"all empty returns nil" | resolve_test.go |
| US3-5: all empty → nil | T012 | TestResolveSkills/"empty slices not nil" | resolve_test.go |
| US4-1: resources provisioned | T009 | TestProvisionFromStore_AllResources | provision_test.go |
| US4-2: missing skill skipped | existing | TestProvisionFromStore_MissingSkillSkipsWithWarning | provision_test.go |
| US4-3: path traversal blocked | existing | TestProvisionFromStore_PathTraversal | provision_test.go |
| US4-4: multi-skill isolation | T011 | TestProvisionFromStore_IsolatedDirs | provision_test.go |
| US5-1: list --format json | existing | TestSkillsListJSON | skills_test.go |
| US5-2: install file source | existing | TestSkillsInstallFileSource | skills_test.go |
| US5-3: remove nonexistent → CodeSkillNotFound | existing | TestSkillsRemoveNonexistent | skills_test.go |
| US5-4: unknown prefix → CodeSkillSourceError | existing | TestSkillsInstallUnknownPrefix | skills_test.go |
| US6-1: full lifecycle | T017 | TestSkillLifecycle_FileAdapter | integration_test.go |
| US6-2: content match after provision | T010 | TestProvisionFromStore_ContentMatch | provision_test.go |
| US6-3: two skills listed | T018 | TestSkillLifecycle_MultiSource | integration_test.go |
| US7-1: authoring guide → valid SKILL.md | T019 | Manual: follow guide | docs/guide/skills.md |
| US7-2: fields match source code | T019 | Manual: cross-check | docs/guide/skills.md |
| US8-1: config guide → correct resolution | T020 | Manual: follow examples | docs/guide/skill-configuration.md |
| US8-2: ecosystem guide → understand setup | T021 | Manual: follow steps | docs/guide/skill-ecosystems.md |

## Parallel Execution Opportunities

Tasks marked with [P] can be executed in parallel within their phase:
- T005, T006, T007 (timeout tests for BMAD, OpenSpec, SpecKit — identical pattern)
- T014, T015 (parseTesslSearchOutput and parseTesslSyncOutput — independent parsers)
- T019, T020, T021 (all documentation guides — no code dependencies between them)

**Total parallelizable tasks**: 8
