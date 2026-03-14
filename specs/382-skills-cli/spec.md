# Feature Specification: Wave Skills CLI — list/search/install/remove/sync

**Feature Branch**: `382-skills-cli`
**Created**: 2026-03-14
**Status**: Draft
**Input**: [Issue #382](https://github.com/re-cinq/wave/issues/382) — wave skills CLI command with subcommands for skill lifecycle management: list, search, install, remove, sync. Supports `--format json` output and source prefix routing for install.
**Parent**: [Issue #239](https://github.com/re-cinq/wave/issues/239) — Skill management system

## User Scenarios & Testing _(mandatory)_

### User Story 1 — List Installed Skills (Priority: P1)

A Wave user wants to see all skills currently installed in their project by running `wave skills list`, so they can understand what capabilities are available, where each skill came from, and which pipelines or personas use them.

**Why this priority**: Listing is the most fundamental operation — users need to inspect what they have before they can install, remove, or troubleshoot. It also validates that the entire skill store infrastructure is accessible via the CLI.

**Independent Test**: Can be tested by populating `.wave/skills/` and `.claude/skills/` with known SKILL.md files, running `wave skills list`, and verifying the output matches the installed skills with correct metadata columns.

**Acceptance Scenarios**:

1. **Given** a project with skills installed in `.wave/skills/` and `.claude/skills/`, **When** the user runs `wave skills list`, **Then** the output displays a table with columns: name, description, source path, and pipeline/persona usage.
2. **Given** a project with no skills installed, **When** the user runs `wave skills list`, **Then** the output displays an informative message indicating no skills are installed, with a hint to use `wave skills install`.
3. **Given** the user runs `wave skills list --format json`, **When** skills are installed, **Then** the output is a valid JSON array of skill objects with name, description, source, and usage fields.
4. **Given** skills exist at multiple precedence levels (project `.wave/skills/` and user `.claude/skills/`), **When** the user runs `wave skills list`, **Then** skills from both sources are shown, with project-level skills indicated as taking precedence over user-level duplicates.

---

### User Story 2 — Install a Skill from Any Source (Priority: P1)

A Wave user wants to install a skill by running `wave skills install <source>`, where `<source>` includes a prefix indicating the ecosystem (e.g., `tessl:github/spec-kit`, `github:owner/repo`, `file:./local/skill`). The CLI parses the prefix, dispatches to the appropriate source adapter, and reports the result.

**Why this priority**: Installation is the primary write operation for the skill lifecycle. Without it, users cannot add new capabilities. It directly exercises the source prefix routing from #383.

**Independent Test**: Can be tested by running `wave skills install file:./test-skill` with a local skill directory containing a valid SKILL.md, then verifying the skill appears in `wave skills list` output.

**Acceptance Scenarios**:

1. **Given** a valid source string `tessl:github/spec-kit`, **When** the user runs `wave skills install tessl:github/spec-kit`, **Then** the CLI dispatches to the Tessl adapter, installs the skill, and displays a success message with the installed skill name.
2. **Given** a valid source string `github:re-cinq/wave-skills/golang`, **When** the user runs `wave skills install github:re-cinq/wave-skills/golang`, **Then** the skill is cloned, validated, and installed, with a success message showing the skill name and source.
3. **Given** a source string with an unrecognized prefix `unknown:something`, **When** the user runs `wave skills install unknown:something`, **Then** the CLI displays an error listing all recognized prefixes (`tessl:`, `bmad:`, `openspec:`, `speckit:`, `github:`, `file:`, `https://`).
4. **Given** a source string `file:./my-skill`, **When** the directory does not exist, **Then** the CLI displays an error with the resolved absolute path and a suggestion to check the path.
5. **Given** the user runs `wave skills install github:owner/repo --format json`, **When** the installation succeeds, **Then** the output is a valid JSON object containing `installed_skills`, `warnings`, and `source` fields.
6. **Given** a source adapter's required CLI tool is not installed (e.g., `tessl` for `tessl:` prefix), **When** the user runs the install command, **Then** the CLI displays a clear error naming the missing tool and providing install instructions.

---

### User Story 3 — Remove an Installed Skill (Priority: P2)

A Wave user wants to remove a skill they no longer need by running `wave skills remove <name>`, so the skill directory is deleted from the skill store and no longer available to pipelines.

**Why this priority**: Removal completes the CRUD lifecycle for skills. Users need to clean up unused skills to keep their project tidy and avoid confusion.

**Independent Test**: Can be tested by installing a skill, running `wave skills remove <name>`, and verifying it no longer appears in `wave skills list`.

**Acceptance Scenarios**:

1. **Given** a skill named `golang` is installed in `.wave/skills/`, **When** the user runs `wave skills remove golang`, **Then** the CLI prompts for confirmation, and upon confirmation, removes the skill directory and displays a success message.
2. **Given** a skill named `golang` is installed, **When** the user runs `wave skills remove golang --yes`, **Then** the skill is removed without a confirmation prompt (non-interactive mode).
3. **Given** the user provides a skill name that does not exist, **When** the user runs `wave skills remove nonexistent`, **Then** the CLI displays an error indicating the skill was not found, listing installed skills as suggestions.
4. **Given** the user runs `wave skills remove golang --format json`, **When** the removal succeeds, **Then** the output is a valid JSON object with `removed` skill name and `source` path fields.

---

### User Story 4 — Search the Tessl Registry (Priority: P3)

A Wave user wants to discover available skills by running `wave skills search <query>`, which searches the Tessl registry and displays matching results with name, rating, and description.

**Why this priority**: Discovery enhances the skill ecosystem but is not essential for basic skill management. Users can install skills by direct reference without searching first.

**Independent Test**: Can be tested by running `wave skills search "golang"` and verifying the output contains formatted search results from the Tessl registry (or mock results in tests).

**Acceptance Scenarios**:

1. **Given** the Tessl CLI is installed and the registry is reachable, **When** the user runs `wave skills search golang`, **Then** the CLI displays matching skills in a table with name, rating, and description columns.
2. **Given** the search returns no results, **When** the user runs `wave skills search nonexistent-skill-name`, **Then** the CLI displays a "no results found" message.
3. **Given** the Tessl CLI is not installed, **When** the user runs `wave skills search golang`, **Then** the CLI displays an error explaining that the `tessl` CLI is required for registry search, with install instructions.
4. **Given** the user runs `wave skills search golang --format json`, **When** results are found, **Then** the output is a valid JSON array of search result objects.

---

### User Story 5 — Sync Project Dependencies (Priority: P3)

A Wave user wants to synchronize all skill dependencies declared in their project with the Tessl registry by running `wave skills sync`, which wraps `tessl install --project-dependencies` to ensure all declared skills are installed.

**Why this priority**: Sync is a convenience operation that automates bulk installation of declared dependencies. It becomes valuable once teams adopt skill-based workflows but is not needed for individual skill management.

**Independent Test**: Can be tested by declaring skill dependencies in the manifest and running `wave skills sync`, verifying that all declared skills are present in the skill store afterward.

**Acceptance Scenarios**:

1. **Given** the project manifest declares skill dependencies, **When** the user runs `wave skills sync`, **Then** the CLI delegates to `tessl install --project-dependencies` and reports which skills were installed or updated.
2. **Given** the Tessl CLI is not installed, **When** the user runs `wave skills sync`, **Then** the CLI displays an error explaining that `tessl` is required, with install instructions.
3. **Given** the user runs `wave skills sync --format json`, **When** the sync completes, **Then** the output is a valid JSON object with `synced_skills` and `warnings` fields.
4. **Given** all declared dependencies are already installed, **When** the user runs `wave skills sync`, **Then** the CLI reports "all skills up to date" with no changes.

---

### Edge Cases

- What happens when the user runs `wave skills` with no subcommand? The CLI displays the help text listing all available subcommands with usage examples.
- What happens when `wave skills list` encounters a malformed SKILL.md? The skill is listed with a warning indicator, and the error details are reported in verbose mode. The command does not fail entirely.
- What happens when `wave skills install` is given no arguments? The CLI displays an error with usage instructions showing the expected source format and prefix examples.
- What happens when `wave skills remove` targets a skill that exists at both project and user level? The CLI removes from the highest-precedence source (project-level `.wave/skills/`) and informs the user that a lower-precedence copy still exists.
- What happens when `wave skills install` is run concurrently for the same skill? Last write wins — consistent with `DirectoryStore.Write` behavior.
- What happens when the skill store directory (`.wave/skills/`) does not exist? The CLI creates it on first write operation (install/sync).
- What happens when `wave skills list --format json` encounters discovery errors for some skills? The JSON output includes a `warnings` array alongside the skills array.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST provide a `wave skills` top-level command with subcommands: `list`, `search`, `install`, `remove`, and `sync`.
- **FR-002**: All subcommands MUST support a local `--format` flag (values: `table`, `json`) defaulting to `table`. The global `--json`, `--quiet`, and `--output` flags take precedence over the local flag via the existing `ResolveFormat(cmd, localFormat)` function in `cmd/wave/commands/output.go`.
- **FR-003**: The `list` subcommand MUST read from the `DirectoryStore` (multi-source with precedence) and display: skill name, description, and source path. Pipeline usage is determined by scanning pipeline YAML files for `requires.skills` entries matching installed skill names, reusing the existing `collectSkillPipelineUsage()` pattern from `cmd/wave/commands/list.go`. Note: this shows pipeline usage by name-matching only; skills discovered via `DirectoryStore` are a different dataset from manifest-declared `SkillConfig` entries in `wave list skills`.
- **FR-004**: The `install` subcommand MUST accept a source string argument and dispatch to the appropriate `SourceAdapter` via the `SourceRouter` based on prefix parsing.
- **FR-005**: The `install` subcommand MUST support all 7 recognized source prefixes: `tessl:`, `bmad:`, `openspec:`, `speckit:`, `github:`, `file:`, `https://`.
- **FR-006**: The `remove` subcommand MUST delete the named skill from the skill store via `DirectoryStore.Delete`, with an interactive confirmation prompt by default.
- **FR-007**: The `remove` subcommand MUST support a `--yes` flag to skip the confirmation prompt for scripted/non-interactive use.
- **FR-008**: The `search` subcommand MUST delegate to the Tessl CLI for registry search and format the results for display.
- **FR-009**: The `sync` subcommand MUST delegate to `tessl install --project-dependencies` to synchronize declared project dependencies.
- **FR-010**: All subcommands MUST return structured `CLIError` responses on failure, following the existing error handling pattern in `cmd/wave/commands/errors.go`. New error codes MUST be defined: `skill_not_found` (removal of non-existent skill), `skill_source_error` (unrecognized source prefix or adapter failure), `skill_dependency_missing` (required CLI tool not installed).
- **FR-011**: The `wave skills` command (no subcommand) MUST display help text listing available subcommands and usage examples.
- **FR-012**: All subcommands MUST follow the existing CLI command pattern: `NewXxxCmd()` returning `*cobra.Command`, options struct, separate implementation function.
- **FR-013**: Unrecognized source prefixes in `install` MUST produce an error listing all recognized prefixes.
- **FR-014**: Missing CLI dependencies (`tessl`, `npx`, `git`, etc.) MUST produce actionable error messages with install instructions.
- **FR-015**: The command MUST be registered in the root command (in `cmd/wave/main.go`) alongside existing commands.

### Key Entities

- **SkillsCommand**: Top-level `wave skills` command that groups all skill lifecycle subcommands. Registered as a Cobra command with `AddCommand()` for each subcommand.
- **SkillListOutput**: Structured output for `list` — contains an array of installed skills with name, description, source path, precedence level, and pipeline/persona usage.
- **SkillInstallOutput**: Structured output for `install` — contains installed skill names, source string, warnings, and success/failure status.
- **SkillRemoveOutput**: Structured output for `remove` — contains the removed skill name, source path, and confirmation status.
- **SkillSearchResult**: Structured output for `search` — contains matched skill name, rating, description, and source registry.
- **SkillSyncOutput**: Structured output for `sync` — contains synced skill names, any warnings, and overall status.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Running `wave skills list` in a project with N installed skills displays all N skills with correct metadata — verified by integration test with a known skill store.
- **SC-002**: Running `wave skills install <source>` for each of the 7 supported prefixes dispatches to the correct adapter — verified by unit tests with mock adapters.
- **SC-003**: Running `wave skills remove <name>` deletes the skill directory and the skill no longer appears in `wave skills list` — verified by integration test.
- **SC-004**: All subcommands produce valid, parseable JSON when `--format json` is specified — verified by unit tests parsing the output through `json.Unmarshal`.
- **SC-005**: Unrecognized source prefixes and missing CLI dependencies produce structured error messages — verified by unit tests checking error codes and suggestion text.
- **SC-006**: The command integrates with the existing output system (`OutputConfig`, `ResolveOutputConfig`) — verified by passing `--json`, `--quiet`, and default modes.
- **SC-007**: All tests pass with `go test -race ./cmd/wave/commands/...` — no data races.
- **SC-008**: `wave skills` with no subcommand displays help text listing all 5 subcommands — verified by running the command and checking output.

## Assumptions

- The `DirectoryStore` (from #381) and `SourceRouter` with all 7 adapters (from #383) are already merged and available in `internal/skill/`.
- The `tessl` CLI is the authoritative registry interface. `search` and `sync` depend on it being installed. This is documented as a soft dependency.
- The `list` subcommand provides a complementary view to the existing `wave list skills` subcommand. `wave skills list` shows SKILL.md-based skills discovered via `DirectoryStore` (from `.wave/skills/` and `.claude/skills/`), while `wave list skills` continues to show manifest-declared `SkillConfig` entries from pipeline `requires.skills` blocks. The existing `wave list skills` is NOT deprecated by this feature.
- The confirmation prompt for `remove` MUST use the injectable `promptConfirm(in io.Reader, out io.Writer, prompt string)` pattern established in `cmd/wave/commands/doctor.go`, accepting `io.Reader` and `io.Writer` parameters for testability rather than hardcoding `os.Stdin`/`os.Stderr`.

## Clarifications

### C1: JSON output flag resolution mechanism
**Ambiguity**: FR-002 mentioned `--format json` alongside global `--json`/`--output json` without specifying precedence.
**Resolution**: Follow the established pattern — each subcommand has a local `--format` flag defaulting to `table`. The global flags (`--json`, `--quiet`, `--output`) override via `ResolveFormat()`. This is the same pattern used by `wave list` (list.go:141).
**Rationale**: Consistency with existing CLI commands; avoids inventing a new flag resolution mechanism.

### C2: Relationship to existing `wave list skills`
**Ambiguity**: The spec said `wave skills list` "replaces" `wave list skills`, but they show fundamentally different data: DirectoryStore SKILL.md files vs manifest-declared SkillConfig pipeline dependencies.
**Resolution**: They are complementary views. `wave skills list` shows the SKILL.md skill store (what's installed). `wave list skills` shows pipeline `requires.skills` declarations (what's declared). No deprecation.
**Rationale**: Removing `wave list skills` would lose the pipeline-dependency view. Users benefit from both: "what do I have installed?" vs "what do my pipelines require?"

### C3: Pipeline usage data source for `list`
**Ambiguity**: FR-003 required "pipeline/persona usage" but the `Skill` struct has no such field and the `DirectoryStore` doesn't track usage.
**Resolution**: Pipeline usage is gathered by scanning pipeline YAML files for `requires.skills` entries, reusing the pattern from `collectSkillPipelineUsage()` in list.go. Skills are matched by name only.
**Rationale**: This is the only reliable source of pipeline-skill relationships in the codebase. It provides a best-effort match between installed SKILL.md skills and pipeline requirements.

### C4: Confirmation prompt testability
**Ambiguity**: The spec hardcoded `os.Stdin`/`os.Stderr` for the remove confirmation prompt, making it untestable.
**Resolution**: Use the injectable `promptConfirm(in io.Reader, out io.Writer, prompt string)` function pattern from doctor.go, which accepts reader/writer interfaces.
**Rationale**: doctor.go already established this testable pattern. Using io.Reader/io.Writer enables unit tests to simulate user input without requiring actual stdin.

### C5: Skill-specific error codes
**Ambiguity**: FR-010 required structured `CLIError` responses but no skill-specific error codes were defined, while the existing errors.go has domain-specific codes for pipelines, manifests, etc.
**Resolution**: Three new error codes: `skill_not_found` (remove non-existent skill), `skill_source_error` (unrecognized prefix or adapter failure), `skill_dependency_missing` (required CLI tool not installed).
**Rationale**: Follows the established pattern of domain-specific error codes (e.g., `pipeline_not_found`, `adapter_not_found`). Machine-parseable error codes are essential for `--format json` error output.

## Out of Scope

- Actual ecosystem CLI delegation logic (handled by #383 source adapters — already merged).
- Hierarchical configuration merging (handled by #385 — already merged).
- Skill update/upgrade command (future enhancement).
- Skill version pinning or lock files.
- Tessl registry authentication or token management.
- Skill dependency resolution (one skill depending on another).
