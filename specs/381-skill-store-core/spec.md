# Feature Specification: Skill Store Core

**Feature Branch**: `381-skill-store-core`
**Created**: 2026-03-13
**Status**: Clarified
**Input**: https://github.com/re-cinq/wave/issues/381 — Skill store core. Foundation for skill management system (#239). SKILL.md parser, store CRUD, `.wave/skills/` directory structure, coexistence with legacy system.

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Parse SKILL.md Files (Priority: P1 — Core Parser)

Wave needs to read and understand skill definitions written in the Agent Skills Specification format (YAML frontmatter + markdown body). This is the foundational capability — without it, no other skill store operation is possible.

**Why this priority**: Every other feature depends on being able to parse SKILL.md files into structured types. This is the atomic building block of the skill store.

**Independent Test**: Can be fully tested by providing SKILL.md content as bytes/strings and verifying the returned struct contains the correct name, description, and body. Delivers the ability to load any conforming SKILL.md file.

**Acceptance Scenarios**:

1. **Given** a SKILL.md file with valid YAML frontmatter containing `name` and `description`, **When** the parser reads it, **Then** a `Skill` struct is returned with the correct name, description, and markdown body
2. **Given** a SKILL.md file with optional frontmatter fields (`license`, `compatibility`, `metadata`, `allowed-tools`), **When** the parser reads it, **Then** all optional fields are correctly populated in the returned struct
3. **Given** a SKILL.md file missing the `name` field, **When** the parser reads it, **Then** a validation error is returned identifying the missing required field
4. **Given** a SKILL.md file missing the `description` field, **When** the parser reads it, **Then** a validation error is returned identifying the missing required field
5. **Given** a file with no YAML frontmatter delimiters, **When** the parser reads it, **Then** a parse error is returned indicating missing frontmatter
6. **Given** a SKILL.md file with a `name` that contains uppercase letters or invalid characters, **When** the parser reads it, **Then** a validation error is returned describing the naming constraint

---

### User Story 2 - Discover Skills from Directory (Priority: P2)

Wave must scan a directory tree to find all `SKILL.md` files and build an inventory of available skills. This enables the store to know what skills exist.

**Why this priority**: Equal to parsing — discovery + parsing together form the minimum viable read path. Neither is useful without the other.

**Independent Test**: Can be tested by creating a temp directory with several `<skill-name>/SKILL.md` entries and verifying the discovery function returns the correct list of skill names and paths.

**Acceptance Scenarios**:

1. **Given** a directory containing subdirectories each with a `SKILL.md` file, **When** discovery runs, **Then** each skill is found and its path recorded
2. **Given** a directory containing a mix of skill directories and non-skill files, **When** discovery runs, **Then** only directories containing `SKILL.md` are included
3. **Given** an empty directory, **When** discovery runs, **Then** an empty list is returned without error
4. **Given** a directory that does not exist, **When** discovery runs, **Then** an appropriate error is returned
5. **Given** a skill directory where the `name` frontmatter does not match the parent directory name, **When** discovery runs with validation, **Then** a validation warning or error is reported

---

### User Story 3 - List Skills (Priority: P3)

Wave operators need to enumerate all skills available in a skill source directory, seeing their names and descriptions at a glance — the metadata tier of progressive disclosure.

**Why this priority**: List is the first user-facing operation and the simplest CRUD read. It enables `wave list skills` and pipeline skill resolution.

**Independent Test**: Can be tested by populating a skill directory and calling the list operation, verifying the returned slice contains the expected names and descriptions.

**Acceptance Scenarios**:

1. **Given** a skill source directory with 3 valid skills, **When** list is called, **Then** all 3 skills are returned with name and description
2. **Given** a skill source directory with 1 valid skill and 1 invalid (malformed frontmatter), **When** list is called, **Then** the valid skill is returned and the invalid one is reported as an error (not silently dropped)
3. **Given** no skill source directories configured, **When** list is called, **Then** an empty list is returned without error

---

### User Story 4 - Read a Single Skill (Priority: P4)

Wave pipelines and personas need to load the full content of a specific skill by name — the instructions tier of progressive disclosure.

**Why this priority**: Read is the core accessor needed by the pipeline executor to inject skill instructions into agent context.

**Independent Test**: Can be tested by writing a SKILL.md to disk, calling read by name, and verifying the full struct including body is returned.

**Acceptance Scenarios**:

1. **Given** a skill named "speckit" exists in the store, **When** read is called with name "speckit", **Then** the full Skill struct is returned including parsed body
2. **Given** no skill named "nonexistent" exists, **When** read is called with name "nonexistent", **Then** a typed not-found error is returned
3. **Given** a skill with associated resource files in `scripts/` or `references/`, **When** read is called, **Then** the Skill struct includes the paths to those resource files
4. **Given** a read request with a name containing path traversal sequences (e.g., `../etc`), **When** read is called, **Then** the operation is rejected with a security error before any filesystem access

---

### User Story 5 - Write a Skill (Priority: P5)

Wave must be able to persist a skill definition to the filesystem — creating the directory, writing `SKILL.md` with correct frontmatter formatting, and creating optional subdirectories.

**Why this priority**: Write enables skill creation and updates. Without it, skills can only be manually authored.

**Independent Test**: Can be tested by calling write with a Skill struct and verifying the resulting directory contains a correctly formatted SKILL.md.

**Acceptance Scenarios**:

1. **Given** a valid Skill struct, **When** write is called, **Then** a directory named after the skill is created containing a SKILL.md with correct YAML frontmatter and markdown body
2. **Given** a Skill struct with a name that already exists in the target directory, **When** write is called, **Then** the existing SKILL.md is overwritten with the new content
3. **Given** a Skill struct with an invalid name (uppercase, special characters), **When** write is called, **Then** a validation error is returned and no files are written
4. **Given** a Skill struct with empty description, **When** write is called, **Then** a validation error is returned and no files are written
5. **Given** a Skill struct with a name containing path traversal sequences (e.g., `../../malicious`), **When** write is called, **Then** the operation is rejected with a security error and no files are written

---

### User Story 6 - Delete a Skill (Priority: P7)

Wave operators need to remove skills from the store, cleaning up the directory and all associated files.

**Why this priority**: Lowest priority CRUD operation. Needed for lifecycle management but not for initial functionality.

**Independent Test**: Can be tested by writing a skill, calling delete, and verifying the directory is removed.

**Acceptance Scenarios**:

1. **Given** a skill named "old-skill" exists in the store, **When** delete is called with name "old-skill", **Then** the entire skill directory is removed
2. **Given** no skill named "nonexistent" exists, **When** delete is called with name "nonexistent", **Then** a typed not-found error is returned
3. **Given** a skill directory path that would escape the store root (path traversal), **When** delete is called, **Then** the operation is rejected with a security error

---

### User Story 7 - Multi-Source Skill Resolution (Priority: P6)

Wave needs to resolve skills from multiple source directories with a defined precedence order. Project-local skills (`.wave/skills/`) override user-global or system defaults, following Wave's existing configuration layering pattern.

**Why this priority**: Multi-source resolution is essential for the skill store to be practically useful — projects must be able to override or extend the default skill set.

**Independent Test**: Can be tested by creating two directories with overlapping skill names and verifying the higher-precedence source wins.

**Acceptance Scenarios**:

1. **Given** a skill "golang" exists in both `.wave/skills/` (project) and `.claude/skills/` (user), **When** the store resolves "golang", **Then** the project-local version is returned
2. **Given** a skill "speckit" exists only in `.claude/skills/` (user), **When** the store resolves "speckit", **Then** the user-level version is returned
3. **Given** the store is configured with source directories `[".wave/skills/", ".claude/skills/"]`, **When** list is called, **Then** skills from all sources are merged with project-local taking precedence for duplicates

---

### Edge Cases

- What happens when a SKILL.md file is empty (zero bytes)? → Parse error with descriptive message
- What happens when frontmatter YAML is malformed (missing closing `---`)? → Parse error indicating unterminated frontmatter
- What happens when the markdown body is empty (frontmatter only)? → Valid parse — body is allowed to be empty
- What happens when a skill directory contains SKILL.md but also unexpected files? → Ignored; only SKILL.md and known subdirectories (`scripts/`, `references/`, `assets/`) are processed
- What happens with symlinked skill directories? → Followed and treated as normal directories
- What happens when filesystem permissions prevent reading SKILL.md? → OS-level error wrapped with skill context
- What happens when the `name` field contains path separators? → Validation rejects it (security: prevents directory traversal)
- What happens when two source directories both have a skill with the same name but different content? → Higher-precedence source wins; no merge of content across sources

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST parse SKILL.md files conforming to the Agent Skills Specification — YAML frontmatter (delimited by `---`) containing at minimum `name` (string, `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, max 64 chars) and `description` (string, max 1024 chars, non-empty)
- **FR-002**: System MUST extract the markdown body (everything after the closing `---` delimiter) as a separate field
- **FR-003**: System MUST support optional frontmatter fields: `license` (string), `compatibility` (string, max 500 chars), `metadata` (map of string to string), `allowed-tools` (space-delimited string in YAML frontmatter, parsed into `[]string` on the Go struct — see C-001)
- **FR-004**: System MUST validate that the `name` frontmatter field matches the parent directory name when loaded from disk
- **FR-005**: System MUST discover skills by scanning a directory for subdirectories containing `SKILL.md` files
- **FR-006**: System MUST provide CRUD operations: Read (by name), Write (create/update), List (all skills), Delete (by name)
- **FR-007**: System MUST support multiple source directories with defined precedence — project-local sources override user-level sources. Default source directories are `.wave/skills/` (project) and `.claude/skills/` (user)
- **FR-008**: System MUST return typed errors for: not-found, parse-failure, validation-failure, and path-security-violation
- **FR-009**: System MUST validate skill names against path traversal attacks (reject names containing `/`, `\`, `..`, or path separators) for all operations including read, write, list, and delete
- **FR-010**: System MUST coexist with the existing legacy skill provisioning path (`Provisioner` + `SkillConfig`) in the same `internal/skill/` package — the new store operates alongside without breaking existing pipeline skill provisioning or requiring changes to existing skill configuration or provisioning code (see C-002)
- **FR-011**: System MUST support the three-tier progressive disclosure model: metadata-only loading (name + description) for listing, full SKILL.md loading for skill activation, and resource file path discovery for on-demand loading
- **FR-012**: System MUST serialize skills back to valid SKILL.md format (YAML frontmatter + markdown body) when writing

### Key Entities

- **Skill**: A parsed SKILL.md representation. Contains: name (identifier), description (trigger text), body (markdown instructions), optional fields (license, compatibility, metadata, allowed-tools), source path (where loaded from), and resource paths (scripts/, references/, assets/ contents)
- **Store**: Go interface defining CRUD operations (Read, Write, List, Delete). Enables mockability and future alternative backends (see C-005)
- **DirectoryStore**: Concrete `Store` implementation backed by filesystem directories. Manages one or more source directories, handles multi-source resolution with precedence (see C-005)
- **SkillSource**: A single directory that may contain skill subdirectories. Has a root path and a precedence level. Non-existent source directories are silently skipped during read operations (see C-004)
- **ParseError**: Typed error for SKILL.md parsing and validation failures. Contains field name, constraint violated, and actual value. Named `ParseError` (not `ValidationError`) to avoid collision with existing `manifest.ValidationError` and `contract.ValidationError` types (see C-005)
- **DiscoveryError**: Aggregate error returned when List encounters per-skill parse failures. Contains the list of individual errors alongside the successfully parsed skills (see C-003)

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: All 13 existing `.claude/skills/*/SKILL.md` files in the Wave project parse successfully with the new parser
- **SC-002**: Round-trip fidelity — parsing a SKILL.md and writing it back produces semantically equivalent output (frontmatter fields preserved, body content preserved)
- **SC-003**: Invalid SKILL.md files produce typed errors that identify the specific validation failure (field name, constraint, actual value)
- **SC-004**: Multi-source resolution correctly prioritizes project-local over user-level skills, verified with overlapping skill names
- **SC-005**: All CRUD operations have comprehensive table-driven tests with edge case coverage, achieving ≥90% line coverage for the new package code
- **SC-006**: The existing legacy skill provisioning path continues to function without modification — all tests exercising legacy skill provisioning pass unchanged
- **SC-007**: Path traversal attempts in skill names are rejected before any filesystem operation occurs
- **SC-008**: The store handles a directory with 50+ skills without performance degradation (list completes in <100ms on standard hardware)

## Clarifications _(resolved)_

### C-001: `allowed-tools` YAML type — space-delimited string vs YAML list

**Ambiguity**: FR-003 specifies `allowed-tools` as a "space-delimited string", but YAML natively supports lists and Wave's internal `AllowedTools` field is `[]string`. Should the parser accept both formats?

**Resolution**: Keep as **space-delimited string** per the Agent Skills Specification format. The SKILL.md frontmatter stores `allowed-tools: "Read Write Edit Bash"` as a single string. The parser splits it into `[]string` internally on the `Skill` struct. This maintains compatibility with the upstream spec format while providing a clean Go slice type for consumers.

**Rationale**: The Agent Skills Specification defines `allowed-tools` as a space-delimited string. No existing SKILL.md files in the project use this field yet, so there's no backward-compatibility concern — but conforming to the external spec format ensures interoperability.

### C-002: Package location — same package vs sub-package

**Ambiguity**: The spec defines new types (`Skill`, `Store`, `SkillSource`, `ValidationError`) but doesn't specify whether they go in the existing `internal/skill/` package or a new sub-package like `internal/skill/store/`.

**Resolution**: All new code goes in the **existing `internal/skill/` package**. The store is a natural extension of the skill domain. The existing types (`SkillConfig`, `Provisioner`) represent the legacy provisioning path; the new types (`Skill`, `Store`) represent the SKILL.md-based path. Both belong in the same domain package.

**Rationale**: Go convention favors cohesive packages with related types over excessive sub-packaging. A sub-package would risk import cycles if the store needs to reference `SkillConfig` or vice versa. The `internal/skill/` package is currently small (3 files), so adding store code won't create a bloated package. The `ValidationError` type should be named `ParseError` to avoid collision with the existing `manifest.ValidationError` — see C-005.

### C-003: List error handling for per-skill parse failures

**Ambiguity**: User Story 3 scenario 2 says invalid skills are "reported as an error (not silently dropped)" but doesn't specify the return signature or error aggregation strategy.

**Resolution**: List returns `([]Skill, error)`. When some skills parse successfully and others fail, the function returns the valid skills **and** a non-nil error. The error is a `*DiscoveryError` type containing a slice of per-skill errors (skill name + underlying error). Callers can inspect individual failures via `errors.As` or log the aggregate.

**Rationale**: This follows the Go pattern of returning partial results alongside errors (similar to `io.ReadAll` returning data read before an error). It matches Wave's existing error patterns — `preflight.PreflightError` wraps multiple `Result` entries. Returning `([]Skill, []error)` is unusual in Go; a single aggregate error is more ergonomic.

### C-004: Source directory non-existence behavior for read operations

**Ambiguity**: User Story 2 scenario 4 specifies that discovery errors on a non-existent directory, but the multi-source resolution (Story 7) doesn't specify what happens when one or more configured source directories don't exist. A fresh project won't have `.wave/skills/`.

**Resolution**: Non-existent source directories are **silently skipped** during List and Read operations — they contribute zero skills and produce no error. Only explicit single-directory discovery (Story 2) returns an error for non-existent paths. Write operations create the target directory if needed.

**Rationale**: A fresh Wave project should be able to list skills without `.wave/skills/` existing yet. This follows the principle of least surprise and matches how Wave handles missing optional configuration — the directory-per-source approach means absence is a valid state (no skills from that source), not an error.

### C-005: Store as Go interface vs concrete struct

**Ambiguity**: The spec says "Store" as a key entity but doesn't specify whether it's a Go interface (for testability/dependency injection) or a concrete struct.

**Resolution**: Define a **`Store` interface** with CRUD methods (`Read`, `Write`, `List`, `Delete`) and a `DirectoryStore` concrete implementation. The interface enables testing with mocks and supports future alternative backends.

**Rationale**: Wave extensively uses interfaces for testability — `SkillDataProvider` in TUI, adapter interfaces in pipeline execution, etc. The store will be consumed by the pipeline executor and CLI commands, both of which benefit from mockable dependencies. The `ValidationError` type defined in the spec should be renamed to `ParseError` in implementation to avoid collision with `manifest.ValidationError` and `contract.ValidationError` already in the codebase.
