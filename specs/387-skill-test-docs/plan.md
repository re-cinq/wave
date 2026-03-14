# Implementation Plan: Comprehensive Test Coverage and Documentation for Skill Management System

**Branch**: `387-skill-test-docs` | **Date**: 2026-03-14 | **Spec**: `specs/387-skill-test-docs/spec.md`
**Input**: Feature specification from `/specs/387-skill-test-docs/spec.md`

## Summary

Bring `internal/skill/` test coverage from 73.6% to ≥80% by filling identified gaps in store concurrency, CLI adapter mocking, ProvisionFromStore resource verification, CRLF parsing, and CLI help output. Add three documentation guides for skill authoring, configuration, and ecosystem integrations. All new tests supplement existing ~2,500 lines — no rewrites.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `gopkg.in/yaml.v3`, `github.com/spf13/cobra`, `os/exec`
**Storage**: Filesystem (DirectoryStore with SkillSource directories)
**Testing**: `go test -race ./...`, `go test -coverprofile`
**Target Platform**: Linux (primary), macOS, Windows
**Project Type**: Single binary CLI
**Performance Goals**: List 55 skills in <100ms (already tested)
**Constraints**: No external CLI dependencies for tests; mock all adapters
**Scale/Scope**: ~12 test files modified/created, 3 documentation guides

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new dependencies required |
| P2: Manifest as Source of Truth | PASS | Docs reference wave.yaml as canonical config |
| P3: Persona-Scoped Execution | N/A | Tests don't touch persona execution |
| P4: Fresh Memory | N/A | No pipeline steps added |
| P5: Navigator-First | N/A | No pipeline changes |
| P6: Contracts at Handovers | N/A | No pipeline changes |
| P7: Relay via Summarizer | N/A | No relay changes |
| P8: Ephemeral Workspaces | PASS | ProvisionFromStore tests use t.TempDir() |
| P9: Credentials Never Touch Disk | PASS | No credential handling in tests |
| P10: Observable Progress | N/A | No event changes |
| P11: Bounded Recursion | N/A | No recursion changes |
| P12: Minimal Step State Machine | N/A | No state changes |
| P13: Test Ownership | PASS | All new tests must pass with `-race`, no `t.Skip()` without linked issue |

**Post-Phase 1 re-check**: PASS — No constitution violations introduced.

## Project Structure

### Documentation (this feature)

```
specs/387-skill-test-docs/
├── plan.md              # This file
├── research.md          # Phase 0 output (coverage gap analysis)
├── data-model.md        # Phase 1 output (entity mapping)
└── tasks.md             # Phase 2 output (from /speckit.tasks)
```

### Source Code (repository root)

```
internal/skill/
├── store_test.go         # MODIFY: add concurrency tests, CRLF full parse
├── source_cli_test.go    # MODIFY: add mocked success paths, stderr capture, timeout tests
├── provision_test.go     # MODIFY: add ProvisionFromStore resource+content tests
├── skill_test.go         # MODIFY: (if needed for Provisioner edge cases)
├── source_test.go        # existing, no changes needed
├── resolve_test.go       # existing, no changes needed (100% coverage)
├── validate_test.go      # existing, no changes needed (100% coverage)
├── source_file_test.go   # existing, gap-fill only if coverage <80% after other additions
├── source_github_test.go # existing, gap-fill only
└── source_url_test.go    # existing, gap-fill only

cmd/wave/commands/
├── skills_test.go        # MODIFY: add help output test, search/sync parse tests
└── skills.go             # existing, no changes

docs/guide/
├── skills.md             # NEW: skill authoring guide (FR-009)
├── skill-configuration.md # NEW: wave.yaml config guide (FR-010)
└── skill-ecosystems.md   # NEW: ecosystem integration guide (FR-011)
```

**Structure Decision**: This is a test-and-docs feature. All new test code goes into existing `_test.go` files in their respective packages. All new documentation goes under `docs/guide/` consistent with existing guides.

## Implementation Phases

### Phase A: Store & Parse Test Gaps (FR-001, FR-002, FR-008)

**Files**: `internal/skill/store_test.go`

1. **Concurrency test** (`TestDirectoryStoreConcurrency`): Launch 10 goroutines doing Write+Delete concurrently on distinct skill names. Verify no race with `-race` flag. (Acceptance: US1-5)

2. **CRLF full parse test** (`TestParseCRLF`): Parse a full SKILL.md with `\r\n` line endings, verify all fields (name, description, license, metadata, allowed-tools, body) extracted correctly. (Acceptance: US1-1 + edge case)

3. **CRLF serialize round-trip** (`TestSerializeCRLFRoundTrip`): Normalize CRLF→LF in Parse, verify round-trip through Serialize+Parse preserves data.

### Phase B: CLI Adapter Test Gaps (FR-003)

**Files**: `internal/skill/source_cli_test.go`

1. **Mocked successful install** for each adapter: Create temp dir with valid SKILL.md files pre-populated, inject a `lookPath` returning a path. Since the actual `exec.CommandContext` will fail (no real binary), test the shared helper `parseAndWriteSkills` path with pre-created files. The dependency-check and prefix tests already cover the adapter-specific logic. Document that full Install() with real execution is an integration test requiring the actual CLI binary.

2. **Stderr capture verification**: For error path tests, verify the error message includes "stderr:" substring. (Acceptance: US2-4)

3. **Context cancellation test** for BMAD, OpenSpec, SpecKit: Mirror the existing `TestTesslAdapterTimeout` pattern. (Acceptance: US2-3)

### Phase C: ProvisionFromStore Test Gaps (FR-005)

**Files**: `internal/skill/provision_test.go` (lower section, ProvisionFromStore tests)

1. **Full resource provisioning test** (`TestProvisionFromStore_AllResources`): Create a skill with files in all 3 resource dirs (scripts/, references/, assets/), provision, verify all copied. (Acceptance: US4-1)

2. **Content match test** (`TestProvisionFromStore_ContentMatch`): Verify SKILL.md body written to workspace matches original skill body content. (Acceptance: US6-2)

3. **Multi-skill isolation test** (`TestProvisionFromStore_IsolatedDirs`): Provision 3 skills, verify each in its own `.wave/skills/<name>/` subdirectory with no cross-contamination. (Acceptance: US4-4)

### Phase D: Hierarchical Merge Test Completeness (FR-004)

**Files**: `internal/skill/resolve_test.go`

Already at 100% coverage. Verify all 5 acceptance scenarios from US3 are explicitly covered by existing tests:
- US3-1 (global-only): `"global only sorted"` test ✓
- US3-2 (global+persona dedup): `"duplicate across scopes appears once"` test ✓
- US3-3 (all three overlapping): `"all three scopes merged deduped sorted"` test ✓
- US3-4 (all nil): `"all empty returns nil"` test ✓ (nil and empty both return nil)
- US3-5 (all empty slices): `"empty slices not nil"` test ✓

**Action**: Add SC-005 traceability comments to existing tests mapping them to acceptance criteria. No new test functions needed.

### Phase E: CLI Command Test Gaps (FR-006, FR-012)

**Files**: `cmd/wave/commands/skills_test.go`

1. **Help output test** (`TestSkillsHelpOutput`): Verify `wave skills --help` includes descriptions for all 5 subcommands (list, install, remove, search, sync). (SC-007)

2. **parseTesslSearchOutput test** (`TestParseTesslSearchOutput`): Unit test the output parser with sample tessl output. Covers search command output correctness without requiring tessl binary.

3. **parseTesslSyncOutput test** (`TestParseTesslSyncOutput`): Unit test the sync output parser with sample output including "installed" and "warning:" lines.

4. **classifySkillError test** (`TestClassifySkillError`): Verify all error code mappings (DependencyError→CodeSkillDependencyMissing, ErrNotFound→CodeSkillNotFound, unknown prefix→CodeSkillSourceError).

### Phase F: Integration Tests (FR-007)

**Files**: `internal/skill/store_test.go` (or new `integration_test.go`)

1. **End-to-end lifecycle test** (`TestSkillLifecycle_FileAdapter`): Install skill via file adapter into real DirectoryStore → List → verify appears → ProvisionFromStore to workspace → verify SKILL.md → Delete → verify gone. (SC-003, US6-1, US6-2, US6-3)

2. **Two skills from different sources** (`TestSkillLifecycle_MultiSource`): Install two skills from different file:// paths, list, verify both with correct metadata. (US6-3)

### Phase G: Documentation (FR-009, FR-010, FR-011)

1. **`docs/guide/skills.md`** — Skill authoring guide:
   - SKILL.md format (all frontmatter fields with examples)
   - Resource directories (scripts/, references/, assets/)
   - Naming conventions (regex, max length)
   - Best practices

2. **`docs/guide/skill-configuration.md`** — Configuration guide:
   - Global skills in wave.yaml
   - Persona-scoped skills
   - Pipeline-scoped skills
   - Precedence rules with examples
   - `commands_glob` pattern

3. **`docs/guide/skill-ecosystems.md`** — Ecosystem integration guide:
   - Tessl: prerequisites, install, search, sync
   - BMAD: prerequisites, install
   - OpenSpec: prerequisites, install
   - SpecKit: prerequisites, install
   - GitHub: format, install
   - File: local install
   - URL: HTTPS install

### Phase H: Verification

1. Run `go test -race ./internal/skill/... ./cmd/wave/commands/...` — zero failures
2. Run `go test -coverprofile=cov.out ./internal/skill/...` — verify ≥80%
3. Grep for `t.Skip()` in new tests — zero without linked issue
4. Verify SC-005: each acceptance criterion traceable to a named test function

## Acceptance Criteria Traceability (SC-005)

| Acceptance Criterion | Test Function | File |
|---------------------|---------------|------|
| US1-1: valid frontmatter parsed | TestParse/"valid with all fields" | store_test.go |
| US1-2: missing required → ParseError | TestParse/"missing name", "missing description" | store_test.go |
| US1-3: empty body accepted | TestParse/"valid with empty body" | store_test.go |
| US1-4: multi-source precedence | TestMultiSourceResolution/"higher precedence shadows" | store_test.go |
| US1-5: concurrent access race-free | TestDirectoryStoreConcurrency | store_test.go (NEW) |
| US2-1: missing CLI → DependencyError | TestTesslAdapterMissingDependency et al. | source_cli_test.go |
| US2-2: mocked CLI → skills written | TestParseAndWriteSkills/"valid skills" | source_cli_test.go |
| US2-3: timeout → error | TestTesslAdapterTimeout + NEW adapters | source_cli_test.go |
| US2-4: stderr in diagnostics | TestCLIAdapterStderrCapture (NEW) | source_cli_test.go |
| US3-1: global-only resolved | TestResolveSkills/"global only sorted" | resolve_test.go |
| US3-2: global+persona deduplicated | TestResolveSkills/"duplicate across scopes" | resolve_test.go |
| US3-3: all scopes merged sorted | TestResolveSkills/"all three scopes merged" | resolve_test.go |
| US3-4: all nil → nil | TestResolveSkills/"all empty returns nil" | resolve_test.go |
| US3-5: all empty → nil | TestResolveSkills/"empty slices not nil" | resolve_test.go |
| US4-1: resources provisioned | TestProvisionFromStore_AllResources (NEW) | provision_test.go |
| US4-2: missing skill skipped | TestProvisionFromStore_MissingSkillSkipsWithWarning | provision_test.go |
| US4-3: path traversal blocked | TestProvisionFromStore_PathTraversal | provision_test.go |
| US4-4: multi-skill isolation | TestProvisionFromStore_IsolatedDirs (NEW) | provision_test.go |
| US5-1: list --format json | TestSkillsListJSON | skills_test.go |
| US5-2: install file source | TestSkillsInstallFileSource | skills_test.go |
| US5-3: remove nonexistent → CodeSkillNotFound | TestSkillsRemoveNonexistent | skills_test.go |
| US5-4: unknown prefix → CodeSkillSourceError | TestSkillsInstallUnknownPrefix | skills_test.go |
| US6-1: full lifecycle | TestSkillLifecycle_FileAdapter (NEW) | store_test.go |
| US6-2: content match after provision | TestProvisionFromStore_ContentMatch (NEW) | provision_test.go |
| US6-3: two skills listed | TestSkillLifecycle_MultiSource (NEW) | store_test.go |

## Complexity Tracking

No constitution violations to track. All changes are additive (tests and docs).
