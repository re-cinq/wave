# Research: Skill Test Coverage & Documentation (Issue #387)

**Date**: 2026-03-14
**Feature Branch**: `387-skill-test-docs`

## Current State Analysis

### Existing Test Coverage Baseline

| File | Lines | Coverage | Key Functions Tested |
|------|-------|----------|---------------------|
| `store_test.go` | 1127 | ~85% (store CRUD) | Read, Write, List, Delete, multi-source, parse, serialize, error types |
| `resolve_test.go` | 129 | 100% (ResolveSkills) | All scope combos, dedup, nil, empty, deterministic, large input |
| `provision_test.go` | 192 | ~60% (Provisioner only) | Provisioner command copying; NOT ProvisionFromStore |
| `source_cli_test.go` | 331 | ~40% (CLI adapters) | Prefix, missing dep, checkDependency, discoverSkillFiles, parseAndWriteSkills |
| `source_test.go` | 310 | ~90% (Router) | Parse, Install, Prefixes, DefaultRouter, all prefix routing |
| `source_file_test.go` | 154 | ~79% (File adapter) | Install from local dir |
| `source_github_test.go` | 317 | ~60% (GitHub adapter) | parseGitHubRef, ref validation |
| `source_url_test.go` | 469 | ~70% (URL adapter) | validateURL, extractTarGz, extractZip |
| `validate_test.go` | 170 | 100% (ValidateSkillRefs) | Validation with/without store, manifest validation |
| `skill_test.go` | 207 | ~80% (Provisioner) | Provision, ProvisionAll, DiscoverCommands, FormatSkillCommandPrompt |
| `skills_test.go` (CLI) | 486 | ~60% (CLI commands) | list, install, remove, search, sync |

**Aggregate**: `internal/skill/` at 73.6%, `cmd/wave/commands/` at 59.8%.

### Coverage Gap Analysis

#### Gap 1: ProvisionFromStore Tests (provision.go)
- **What exists**: `TestProvisionFromStore_Success`, `_MissingSkillSkipsWithWarning`, `_EmptyList`, `_PathTraversal`, `_MultipleSkills` — 5 test functions
- **Gaps**:
  - No test for resource files across multiple resource directories (scripts+references+assets)
  - No test verifying SKILL.md body content matches original after provisioning
  - No test for read-only workspace directory scenario (edge case from spec)
  - No test with a skill that has ALL resource subdirectories populated

#### Gap 2: CLI Adapter Mocked Execution (source_cli.go)
- **What exists**: Each of the 4 CLI adapters tested for Prefix() and missing dependency. Tessl timeout test exists. discoverSkillFiles tested.
- **Gaps**:
  - No test for successful mocked CLI execution producing SKILL.md files (only missing-dep path tested)
  - No test for stderr capture in error diagnostics
  - BMAD, OpenSpec, SpecKit have NO timeout tests
  - No test verifying installed skills have correct metadata after CLI run

#### Gap 3: CRLF Line Ending Full Parse (parse.go)
- **What exists**: `TestSplitFrontmatterEdgeCases` tests CRLF at the splitFrontmatter level
- **Gaps**:
  - No end-to-end Parse() test with CRLF SKILL.md content verifying all fields extracted correctly
  - No CRLF round-trip through Serialize+Parse

#### Gap 4: Store Concurrent Access (store.go)
- **What exists**: No concurrency tests
- **Gaps**:
  - Spec requires: "Given concurrent goroutines writing and deleting skills, When operations complete, Then no race conditions occur (verified by -race flag)"
  - Need goroutine-based test with `-race`

#### Gap 5: CLI Commands Completeness (skills.go)
- **What exists**: Tests for list, install, remove, search (missing tessl), sync (missing tessl), unknown prefix, no args
- **Gaps**:
  - No test for `wave skills remove nonexistent` returning `CodeSkillNotFound`... wait, T028 covers this. ✓
  - No test for install with `--format json` on file source... wait, T026 covers this. ✓
  - No test for `wave skills --help` output verification (SC-007)
  - parseTesslSearchOutput and parseTesslSyncOutput not tested directly

#### Gap 6: Documentation (docs/guide/)
- **What exists**: No skill documentation files exist
- **Gaps**: All 3 guides required (FR-009, FR-010, FR-011)

### Decision Matrix

| Decision | Choice | Rationale | Alternatives Rejected |
|----------|--------|-----------|----------------------|
| Test approach | Supplement existing tests, don't rewrite | ~2500 lines already exist covering most paths | Full rewrite: risky, wasteful |
| CLI adapter mocking | Inject mocked exec commands via lookPath | Pattern already established in existing tests | Real CLI calls: requires external deps |
| Integration test scope | File adapter + real DirectoryStore only | Only file adapter testable without external CLIs | Mocking all 7 adapters: excessive |
| Doc structure | 3 separate guides under `docs/guide/` | Matches existing guide pattern (configuration.md, pipelines.md, personas.md) | Single doc: too long |
| Concurrency test approach | Multiple goroutines writing/deleting with `-race` | Standard Go pattern; spec requires it | Channel-based sync: overcomplicated |
| SC-005 traceability | Test function names match acceptance criteria pattern | `Test<Component>_<ScenarioDescription>` naming | Comment-based: harder to trace |

## Key Technical Findings

1. **ProvisionFromStore writes ONLY the Body** (not the full SKILL.md with frontmatter) to the workspace — see `provision.go:46`. Tests should verify Body content, not raw frontmatter.

2. **CLI adapters share a common pattern**: `checkDependency` → `os.MkdirTemp` → `exec.CommandContext` → `discoverSkillFiles` → `parseAndWriteSkills`. Mocking `lookPath` to return a path, then the command will still fail because no real binary exists. Need to test the shared `parseAndWriteSkills` path separately (already done) and the dep-check paths (already done). Full Install() path with mocked exec requires more invasive mocking or test-specific binaries.

3. **`collectSkillPipelineUsage`** reads `.wave/pipelines/` YAML files — only relevant for list output enrichment, not core test concern.

4. **ResolveSkills already at 100% coverage** — no new tests needed per resolve_test.go analysis. All spec acceptance criteria already have corresponding tests.

5. **Store.List uses ParseMetadata** (not Parse) — metadata-only loading. Existing tests verify this correctly.
