# Feature Specification: Comprehensive Test Coverage and Documentation for Skill Management System

**Feature Branch**: `387-skill-test-docs`
**Created**: 2026-03-14
**Status**: Draft
**Input**: User description: "https://github.com/re-cinq/wave/issues/387"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Developer Validates Skill Store Reliability (Priority: P1)

A developer contributing to Wave needs confidence that the skill store (CRUD operations, SKILL.md parsing) behaves correctly across all edge cases. They run `go test -race ./internal/skill/...` and expect comprehensive unit tests that catch regressions in parsing, storage, listing, deletion, and multi-source precedence.

**Why this priority**: The skill store is the foundation of the entire skill system. Every other component (resolution, provisioning, CLI) depends on correct store behavior. Untested CRUD edge cases risk silent data corruption or skill loss.

**Independent Test**: Can be fully tested by running `go test -race ./internal/skill/...` and verifying all store CRUD operations, parse edge cases, and serialization round-trips pass.

**Acceptance Scenarios**:

1. **Given** a SKILL.md with valid frontmatter and body, **When** parsed by the store, **Then** all fields (name, description, license, compatibility, metadata, allowed-tools) are correctly extracted
2. **Given** a SKILL.md with missing required frontmatter fields, **When** parsed, **Then** a descriptive `ParseError` is returned identifying the missing field
3. **Given** a SKILL.md with empty body but valid frontmatter, **When** parsed, **Then** the skill is accepted with an empty body
4. **Given** a store with multiple sources at different precedence levels, **When** two sources contain a skill with the same name, **Then** the higher-precedence source's version is returned by `Read()`
5. **Given** concurrent goroutines writing and deleting skills, **When** operations complete, **Then** no race conditions occur (verified by `-race` flag)

---

### User Story 2 - Developer Validates Ecosystem Adapter Correctness (Priority: P1)

A developer needs assurance that each ecosystem adapter (Tessl, BMAD, OpenSpec, SpecKit) correctly delegates to its backing CLI tool, handles missing dependencies gracefully, and produces correct `DependencyError` types with helpful install instructions.

**Why this priority**: Ecosystem adapters are the primary user-facing installation pathway. Broken adapter behavior leads to confusing install failures and poor developer experience. All adapters must be tested without requiring actual CLI installations (mocked).

**Independent Test**: Can be tested by running adapter unit tests with mocked CLI lookups and verifying error types, messages, and install instructions.

**Acceptance Scenarios**:

1. **Given** the Tessl CLI is not installed, **When** `TesslAdapter.Install()` is called, **Then** a `DependencyError` is returned with the correct binary name and install instructions
2. **Given** a mocked Tessl CLI execution that produces valid SKILL.md files, **When** `TesslAdapter.Install()` completes, **Then** all discovered skills are written to the store with correct metadata
3. **Given** a CLI adapter execution that times out, **When** the context deadline expires, **Then** the adapter returns an error without leaking goroutines or zombie processes
4. **Given** a CLI command that writes to stderr, **When** the adapter captures output, **Then** stderr content is included in error diagnostics

---

### User Story 3 - Developer Validates Hierarchical Merge Correctness (Priority: P1)

A developer needs confidence that skill resolution across global, persona, and pipeline scopes follows the documented precedence rules: pipeline > persona > global. Edge cases like deduplication, empty scopes, and nil inputs must all be covered.

**Why this priority**: Incorrect merge logic causes skills to silently appear or disappear depending on configuration scope. This is a critical correctness property that is difficult to debug in production.

**Independent Test**: Can be tested by running `go test ./internal/skill/ -run TestResolve` and verifying all merge scenarios produce deterministic, correctly-ordered output.

**Acceptance Scenarios**:

1. **Given** a skill declared at global scope only, **When** resolved with empty persona and pipeline scopes, **Then** the skill appears in the resolved list
2. **Given** the same skill declared at both global and persona scope, **When** resolved, **Then** the skill appears exactly once (deduplicated)
3. **Given** skills at all three scopes with overlapping names, **When** resolved, **Then** the union is returned alphabetically sorted with no duplicates
4. **Given** all three scopes are nil, **When** resolved, **Then** nil (not empty slice) is returned
5. **Given** all three scopes are empty slices, **When** resolved, **Then** nil is returned (preserving nil-vs-empty semantics)

---

### User Story 4 - Developer Validates Worktree Provisioning Correctness (Priority: P2)

A developer needs assurance that skills are correctly provisioned into worktree workspaces: SKILL.md files are written, resource directories (scripts/, references/, assets/) are copied, path traversal attacks are rejected, and CLAUDE.md injection includes skill metadata.

**Why this priority**: Provisioning is the bridge between skill configuration and runtime execution. Incorrect provisioning causes silent skill unavailability during pipeline execution, which is difficult to diagnose.

**Independent Test**: Can be tested by running provisioning tests with a mock store and verifying file system output matches expectations.

**Acceptance Scenarios**:

1. **Given** a skill with resources in scripts/ and references/, **When** provisioned to a workspace, **Then** SKILL.md and all resource files exist at the correct paths under `.wave/skills/<name>/`
2. **Given** a skill name that does not exist in the store, **When** provisioned, **Then** it is skipped with a warning (not a fatal error) and other skills are still provisioned
3. **Given** a skill with resource paths containing `../`, **When** provisioned, **Then** the traversal is rejected and an error is returned
4. **Given** multiple skills provisioned to the same workspace, **When** provisioning completes, **Then** each skill occupies its own isolated subdirectory

---

### User Story 5 - Developer Validates CLI Command Correctness (Priority: P2)

A developer needs all `wave skills` CLI subcommands (list, install, remove, search, sync) to be thoroughly tested for argument parsing, source prefix routing, output formatting (table and JSON), and error classification.

**Why this priority**: The CLI is the user-facing surface of the skill system. Incorrect argument parsing or output formatting directly impacts usability.

**Independent Test**: Can be tested by running CLI command tests with mocked stores and routers, verifying output format and error messages.

**Acceptance Scenarios**:

1. **Given** `wave skills list --format json`, **When** skills exist in the store, **Then** valid JSON matching `SkillListOutput` schema is produced
2. **Given** `wave skills install file:./path/to/skill`, **When** the local skill is valid, **Then** the skill is installed and JSON output includes the skill name and source
3. **Given** `wave skills remove nonexistent`, **When** the skill is not in the store, **Then** a `CodeSkillNotFound` error is returned
4. **Given** `wave skills install unknown-prefix:foo`, **When** the prefix is not registered, **Then** a `CodeSkillSourceError` error with a descriptive message is returned

---

### User Story 6 - Developer Runs End-to-End Skill Lifecycle Tests (Priority: P2)

A developer needs integration tests that exercise complete skill lifecycle flows: install from a local file source, verify it appears in list, provision it to a workspace, and remove it — all using the real store implementation (not mocks) with the file adapter.

**Why this priority**: Unit tests verify individual components but miss integration failures at boundaries. The file adapter is the only adapter that can be tested without external CLI dependencies.

**Independent Test**: Can be tested by running integration tests that create a temporary store, install a test skill from a local file, and verify the full lifecycle.

**Acceptance Scenarios**:

1. **Given** a valid SKILL.md in a temporary directory, **When** installed via `file:` adapter, listed, provisioned, and removed, **Then** each operation succeeds and the store is empty after removal
2. **Given** a skill installed from a file source, **When** provisioned to a workspace and the workspace is inspected, **Then** the SKILL.md content matches the original
3. **Given** two skills installed from different file sources, **When** listed, **Then** both appear with correct metadata and source paths

---

### User Story 7 - User Reads Skill Authoring Documentation (Priority: P3)

A user or contributor wants to create a custom skill for Wave. They need a guide that explains the SKILL.md format (frontmatter fields, body content), resource directories, and best practices for skill naming and organization.

**Why this priority**: Without documentation, skill authoring is trial-and-error. Documentation enables community contribution and ecosystem growth.

**Independent Test**: Can be verified by following the guide to create a test skill that parses successfully.

**Acceptance Scenarios**:

1. **Given** a user reads the skill authoring guide, **When** they create a SKILL.md following the documented format, **Then** it parses without errors via `wave skills install file:./my-skill`
2. **Given** the guide documents all frontmatter fields, **When** the user checks the fields against the source code, **Then** all fields match (name, description, license, compatibility, metadata, allowed-tools)

---

### User Story 8 - User Reads Configuration and Ecosystem Guide (Priority: P3)

A user wants to configure skills in their wave.yaml at global, persona, and pipeline scopes, and understand how to use ecosystem integrations (Tessl, BMAD, OpenSpec, SpecKit).

**Why this priority**: Configuration documentation is essential for adoption. Users need to know how skills interact with manifests at different scopes.

**Independent Test**: Can be verified by following the configuration examples and observing correct skill resolution behavior.

**Acceptance Scenarios**:

1. **Given** the configuration guide provides wave.yaml examples for all three scopes, **When** the user applies the examples, **Then** skills resolve according to documented precedence rules
2. **Given** the ecosystem guide lists setup steps for each adapter, **When** a user follows the steps, **Then** they understand prerequisites, installation commands, and expected behavior

---

### Edge Cases

- What happens when a SKILL.md has CRLF line endings instead of LF?
- How does the system handle a SKILL.md with frontmatter but no trailing `---` delimiter?
- What happens when a skill name in wave.yaml doesn't match any installed skill?
- How does the store handle concurrent reads and writes to the same skill?
- What happens when disk space is exhausted during skill installation?
- How does provisioning handle a read-only workspace directory?
- What happens when a file adapter path contains a symlink?
- How does the system handle a SKILL.md larger than available memory?
- What happens when a GitHub adapter clone is interrupted by context cancellation?
- How does the URL adapter handle a tar.gz with more than 1000 files (extraction limit)?

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: Test suite MUST include unit tests for all `DirectoryStore` methods: `Read()`, `Write()`, `List()`, `Delete()`, covering success paths, error paths, and security edge cases (path traversal, symlink rejection)
- **FR-002**: Test suite MUST include unit tests for `Parse()` and `ParseMetadata()` covering valid documents, missing frontmatter, empty body, invalid YAML, CRLF line endings, and maximum field length boundaries
- **FR-003**: Test suite MUST include unit tests for all four ecosystem CLI adapters (Tessl, BMAD, OpenSpec, SpecKit) with mocked CLI delegation, verifying `DependencyError` types, install instruction messages, and timeout handling. Non-CLI adapters (File, GitHub, URL) are covered by their own existing test files (`source_file_test.go`, `source_github_test.go`, `source_url_test.go`) and only need gap-filling, not complete rewrite
- **FR-004**: Test suite MUST include unit tests for `ResolveSkills()` covering global-only, persona-override, pipeline-override, deduplication, empty scopes, nil scopes, and deterministic ordering
- **FR-005**: Test suite MUST include unit tests for `ProvisionFromStore()` covering file copying, resource directory handling, missing skill graceful skip, path traversal rejection, and multi-skill provisioning
- **FR-006**: Test suite MUST include unit tests for all `wave skills` CLI subcommands covering argument parsing, source prefix routing, output formatting (table and JSON), and error classification (`CodeSkillNotFound`, `CodeSkillSourceError`, `CodeSkillDependencyMissing`)
- **FR-007**: Test suite MUST include integration tests for end-to-end skill install/remove flows using the local file adapter with real `DirectoryStore` (not mocked)
- **FR-008**: All tests MUST pass with `go test -race ./...` (no data races)
- **FR-009**: Documentation MUST include a skill authoring guide at `docs/guide/skills.md` covering SKILL.md format (all frontmatter fields), resource directory structure, naming conventions, and best practices
- **FR-010**: Documentation MUST include wave.yaml skill configuration examples at `docs/guide/skill-configuration.md` covering global, persona, and pipeline scopes with clear precedence explanation
- **FR-011**: Documentation MUST include an ecosystem integration guide at `docs/guide/skill-ecosystems.md` covering Tessl, BMAD, OpenSpec, and SpecKit setup, prerequisites, and usage
- **FR-012**: `wave skills --help` and all subcommand help text MUST be clear, complete, and consistent with documentation

### Key Entities

- **Skill**: A SKILL.md document with frontmatter (name, description, license, compatibility, metadata, allowed-tools) and a markdown body. Lives in a named directory with optional resource subdirectories (scripts/, references/, assets/)
- **DirectoryStore**: Multi-source store with precedence-based resolution. The CLI configures exactly two sources: project (`.wave/skills`, precedence 2) and user (`~/.claude/skills`, precedence 1). CRUD operations with path traversal and symlink security. Tests may configure additional sources for multi-source scenario coverage
- **SourceAdapter**: Interface for installing skills from external sources. Seven implementations: Tessl, BMAD, OpenSpec, SpecKit, GitHub, File, URL
- **SkillConfig**: Wave.yaml declaration of a required skill with install, init, check, and commands_glob fields. Exists at global, persona, and pipeline scopes
- **SkillInfo**: Metadata returned after provisioning a skill into a workspace via `ProvisionFromStore()`: name, description, source path. (Note: there is no `ProvisionResult` type — `ProvisionFromStore` returns `[]SkillInfo`)

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: All tests pass with `go test -race ./internal/skill/... ./cmd/wave/commands/...` on the first run with zero failures
- **SC-002**: Unit test coverage for `internal/skill/` package reaches at least 80% line coverage (measured via `go test -coverprofile`)
- **SC-003**: At least one integration test exercises the complete skill lifecycle (install, list, provision, remove) end-to-end with no mocks for the store layer
- **SC-004**: Documentation comprises at least three guides: skill authoring, wave.yaml configuration, and ecosystem integration
- **SC-005**: Every acceptance criterion from issue #387 has a corresponding test function that can be traced by name
- **SC-006**: No `t.Skip()` calls exist in new tests without a linked GitHub issue
- **SC-007**: `wave skills --help` output includes descriptions for all subcommands (list, install, remove, search, sync) and is verified by a test

## Clarifications

Ambiguities identified and resolved during spec refinement.

### C1: Entity naming — ProvisionResult vs SkillInfo

**Ambiguity**: The Key Entities section listed "ProvisionResult (SkillInfo)" suggesting two names for the same type.

**Resolution**: The codebase only defines `SkillInfo` (in `internal/skill/provision.go:13`). There is no `ProvisionResult` type. `ProvisionFromStore()` returns `[]SkillInfo`. Updated the Key Entities section to use the correct type name.

**Rationale**: Direct code inspection confirms only `SkillInfo` exists. Using the wrong type name would confuse test authors asserting return values.

### C2: DirectoryStore source tiers — 2, not 3

**Ambiguity**: The spec said "project > user > global" (3 tiers) for DirectoryStore sources, but the actual CLI only configures 2 sources.

**Resolution**: `newSkillStore()` in `cmd/wave/commands/skills.go:92` creates exactly 2 sources: `.wave/skills` (precedence 2) and `~/.claude/skills` (precedence 1). There is no "global" directory. Updated Key Entities to reflect the 2-source reality. Tests may still create 3+ source scenarios for multi-source coverage since `DirectoryStore` supports arbitrary source counts.

**Rationale**: Documenting phantom sources would mislead implementers into looking for non-existent global skill directories.

### C3: Non-CLI adapter test scope

**Ambiguity**: FR-003 covers only 4 CLI ecosystem adapters, but 7 total adapters exist. The spec didn't clarify test expectations for File, GitHub, and URL adapters.

**Resolution**: Non-CLI adapters (File, GitHub, URL) already have dedicated test files (`source_file_test.go`, `source_github_test.go`, `source_url_test.go`). FR-003 is scoped to CLI adapters only. These existing tests should be gap-filled where coverage is low, not rewritten. Updated FR-003 to make this explicit.

**Rationale**: The issue's acceptance criteria focus on ecosystem adapter mocking (CLI-dependent adapters). File/GitHub/URL adapters have different testing needs (filesystem, HTTP mocking) and are already reasonably covered.

### C4: Existing test baseline — supplement, don't replace

**Ambiguity**: The spec doesn't acknowledge that ~2,500 lines of tests already exist across `store_test.go` (1127 lines), `resolve_test.go` (129 lines), `provision_test.go` (192 lines), `source_cli_test.go` (331 lines), `source_file_test.go` (154 lines), and `skills_test.go`.

**Resolution**: New tests MUST supplement existing tests, not replace them. The implementation should identify coverage gaps in existing test files and add new test cases to fill them. Existing test functions that already satisfy acceptance criteria should be preserved and referenced in SC-005 traceability mapping.

**Rationale**: Rewriting working tests wastes effort and risks regressions. The 80% coverage target (SC-002) is an aggregate measure — some files may already exceed it while others have gaps.

### C5: Documentation file paths

**Ambiguity**: FR-009/FR-010/FR-011 required documentation guides but didn't specify file locations. The project has multiple doc directories (`docs/guide/`, `docs/guides/`, `docs/reference/`).

**Resolution**: New skill documentation goes under `docs/guide/` to be consistent with existing user-facing guides (`docs/guide/configuration.md`, `docs/guide/pipelines.md`, `docs/guide/personas.md`). Specific paths:
- FR-009: `docs/guide/skills.md` (skill authoring)
- FR-010: `docs/guide/skill-configuration.md` (wave.yaml configuration)
- FR-011: `docs/guide/skill-ecosystems.md` (ecosystem integrations)

**Rationale**: `docs/guide/` contains the primary user-facing guides. `docs/guides/` contains more specialized operational guides. Skill documentation is user-facing conceptual content, so `docs/guide/` is the correct location.
