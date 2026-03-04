# Feature Specification: Init Merge & Upgrade Workflow

**Feature Branch**: `230-init-merge-upgrade`  
**Created**: 2026-03-04  
**Status**: Draft  
**Input**: https://github.com/re-cinq/wave/issues/230

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Pre-Merge Change Summary (Priority: P1)

A Wave user upgrades to a new version and runs `wave init --merge` on an existing project. Before any files are modified, the system displays a clear summary of what will change: which files will be added (new defaults), which files differ from defaults (stale), and which files are unchanged. The user reviews the summary and explicitly confirms before proceeding.

**Why this priority**: Without a change summary, users risk silent data loss or unexpected modifications. This is the core safety mechanism requested in the issue and the foundation for all other upgrade workflow improvements.

**Independent Test**: Can be fully tested by running `wave init --merge` on a project with customized personas/pipelines and verifying the summary output lists the correct files categorized as new/modified/unchanged, and that no files are written until the user confirms.

**Acceptance Scenarios**:

1. **Given** an existing Wave project with a customized `navigator.md` persona and default `craftsman.md`, **When** the user runs `wave init --merge` after upgrading to a version with updated defaults, **Then** the system displays a table showing `navigator.md` as "preserved (user-modified)", `craftsman.md` as "preserved (matches default)", and any new default files as "will be created".
2. **Given** an existing Wave project, **When** the user runs `wave init --merge` and the summary is displayed, **Then** the user can abort by answering "N" and no files are modified.
3. **Given** an existing Wave project, **When** the user runs `wave init --merge --yes`, **Then** the summary is displayed to stderr but confirmation is skipped, and changes proceed automatically.
4. **Given** an existing Wave project with no differences from defaults, **When** the user runs `wave init --merge`, **Then** the system reports "Already up to date — no changes needed" and exits cleanly.

---

### User Story 2 - Manifest Merge with Diff Preview (Priority: P1)

A user runs `wave init --merge` and the system shows a human-readable summary of the manifest (wave.yaml) changes — new keys being added, existing keys being preserved, and any structural changes. The merged manifest preserves all user customizations while incorporating new defaults.

**Why this priority**: The manifest is the single source of truth for Wave configuration. Users must understand what changes to their manifest before it is rewritten. This is equally critical to the file-level summary.

**Independent Test**: Can be fully tested by creating a wave.yaml with custom adapter settings and additional personas, running `wave init --merge`, and verifying the diff preview correctly identifies additions vs preservations, and that the final manifest retains all user customizations.

**Acceptance Scenarios**:

1. **Given** an existing `wave.yaml` with a custom adapter `ollama` and custom persona `my-reviewer`, **When** `wave init --merge` is run, **Then** the manifest preview shows the custom adapter and persona as "preserved" and any new default entries as "added".
2. **Given** an existing `wave.yaml` with user-modified `runtime` settings (e.g., custom `max_concurrent_workers`), **When** `wave init --merge` is run, **Then** the user's runtime values are preserved and not overwritten by defaults.
3. **Given** an existing `wave.yaml` missing a new configuration section introduced in the upgrade (e.g., a new `runtime.relay` subsection), **When** `wave init --merge` is run, **Then** the new section is added while existing sections remain untouched.

---

### User Story 3 - Consistent Force and Merge Flag Semantics (Priority: P2)

A user needs clear, predictable behavior for the `--force` and `--merge` flags, both independently and combined. `--force` always overwrites without prompting. `--merge` always preserves user files while adding missing defaults. When combined, `--merge --force` applies merge logic but skips the confirmation prompt.

**Why this priority**: Currently `--force` is silently ignored when combined with `--merge`, creating confusing behavior. Consistent flag semantics prevent user confusion and reduce support burden.

**Independent Test**: Can be tested by running all four flag combinations (`init`, `init --force`, `init --merge`, `init --merge --force`) on an existing project and verifying each produces the documented behavior.

**Acceptance Scenarios**:

1. **Given** an existing Wave project, **When** `wave init --force` is run, **Then** all files are overwritten with defaults without any confirmation prompt.
2. **Given** an existing Wave project, **When** `wave init --merge` is run, **Then** the change summary is displayed and user must confirm before changes are applied.
3. **Given** an existing Wave project, **When** `wave init --merge --force` is run, **Then** merge logic is applied (preserving user files, adding missing defaults) without prompting for confirmation, and the change summary is still printed to stderr.

---

### User Story 4 - Post-Upgrade Migration Verification (Priority: P2)

After running `wave init --merge`, the user runs `wave migrate` to apply any pending database schema migrations. The system reports migration status and applies changes. If there are no pending migrations, it reports that the database is up to date.

**Why this priority**: The upgrade workflow is a two-step process (init merge + migrate). Users must be guided through both steps and see clear feedback about migration status.

**Independent Test**: Can be tested by creating a project with an older schema version, running `wave init --merge` followed by `wave migrate up`, and verifying all pending migrations are applied and the database is consistent.

**Acceptance Scenarios**:

1. **Given** an existing Wave project with database at migration version 4, **When** the user runs `wave migrate up` after upgrading to a version with 6 migrations, **Then** migrations 5 and 6 are applied in order and the system reports success.
2. **Given** an existing Wave project with all migrations applied, **When** the user runs `wave migrate status`, **Then** the system reports all migrations are applied and no pending migrations exist.
3. **Given** an existing Wave project, **When** the user runs `wave migrate validate`, **Then** the system verifies all applied migration checksums match and reports integrity status.

---

### User Story 5 - Integration Tests for Upgrade Path (Priority: P2)

The upgrade workflow (clean project → init → upgrade → init --merge → migrate) is covered by automated integration tests that verify no data loss, correct merge behavior, and successful migration.

**Why this priority**: Without integration tests, regressions in the upgrade path will go undetected. Tests provide confidence that the workflow works end-to-end.

**Independent Test**: Can be tested by running `go test ./...` and verifying all upgrade-path tests pass, covering the full lifecycle from initial project creation through upgrade reinitializtion.

**Acceptance Scenarios**:

1. **Given** the test suite, **When** `go test ./cmd/wave/commands/ -run TestInitMerge` is executed, **Then** tests covering the reinit-on-existing-project path pass, including preservation of user customizations, addition of new defaults, manifest merge correctness, and change summary output.
2. **Given** the test suite, **When** `go test ./cmd/wave/commands/ -run TestInitMerge` is executed, **Then** tests verify that `--force` and `--merge` flags interact correctly and produce expected behavior for all combinations.
3. **Given** the test suite, **When** integration tests run the full cycle (init → modify → init --merge → migrate), **Then** all user modifications survive and all migrations apply cleanly.

---

### User Story 6 - Upgrade Workflow Documentation (Priority: P3)

A user finds clear documentation describing the recommended upgrade workflow: how to update the Wave binary, run `wave init --merge` to sync scaffolding, run `wave migrate` to update the database, and what to expect at each step.

**Why this priority**: Documentation is essential for adoption but can be written after the implementation is stable. It depends on the final behavior of the other stories.

**Independent Test**: Can be tested by following the documented steps from scratch on a real project and verifying each step produces the described outcome.

**Acceptance Scenarios**:

1. **Given** user-facing documentation exists, **When** a user reads the upgrade guide, **Then** they find step-by-step instructions covering: updating the binary, running init --merge, reviewing the change summary, running migrate, and verifying success.
2. **Given** the documentation, **When** a user follows the guide on an existing project, **Then** the described outputs match actual command outputs and no undocumented steps are required.

---

### Edge Cases

- What happens when `wave init --merge` is run but `wave.yaml` is malformed or contains invalid YAML? The system should report a parse error with the file path and line number, and exit without modifying any files.
- What happens when `wave init --merge` encounters a persona file that exists but is empty (0 bytes)? The system should treat it as user-modified (preserved) rather than silently overwriting it.
- What happens when the filesystem has read-only permissions on `.wave/` during merge? The system should report a clear permission error for each affected file and exit with a non-zero status.
- What happens when `wave init --merge` is run in a non-interactive terminal (e.g., CI/CD)? The system should require `--yes` flag to proceed, or abort with a message explaining how to run non-interactively.
- What happens when the user runs `wave migrate up` but the database file (`.wave/state.db`) does not exist? The system should create a fresh database and apply all migrations from scratch.
- What happens when `wave init --merge` is run on a project initialized with `--all` flag but the user now wants only release pipelines? The system should preserve extra pipelines (no deletion) and only add missing release pipeline defaults.
- What happens when embedded defaults contain a renamed or removed persona/pipeline? The system should not delete user files that no longer match defaults — only add missing files.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: `wave init --merge` MUST display a categorized file summary before modifying any files, grouping files into: "new" (will be created), "preserved" (user file exists), and "up to date" (matches current default).
- **FR-002**: `wave init --merge` MUST prompt the user for confirmation after displaying the summary, unless `--yes` or `--force` is specified.
- **FR-003**: `wave init --merge` MUST preserve all user-modified persona, pipeline, contract, and prompt files — never overwriting existing files with defaults.
- **FR-004**: `wave init --merge` MUST add any new default files (personas, pipelines, contracts, prompts) that do not exist in the user's project.
- **FR-005**: `wave init --merge` MUST deep-merge the manifest (`wave.yaml`), preserving all user-defined keys while adding new default keys.
- **FR-006**: `wave init --merge` MUST display a human-readable summary of manifest changes (new keys added, existing keys preserved) before writing.
- **FR-007**: `wave init --merge --force` MUST apply merge logic without confirmation prompt but still print the change summary to stderr.
- **FR-008**: `wave init --merge --yes` MUST behave identically to `--merge --force` (skip prompt, print summary).
- **FR-009**: `wave init --force` (without merge) MUST overwrite all files with defaults without prompting.
- **FR-010**: `wave init` (without flags, existing project) MUST prompt for confirmation before overwriting, as it currently does.
- **FR-011**: `wave migrate up` MUST apply all pending database migrations in version order and report each migration applied.
- **FR-012**: `wave migrate status` MUST report current schema version and list pending migrations.
- **FR-013**: The system MUST abort `wave init --merge` with a clear error if `wave.yaml` cannot be parsed, without modifying any files.
- **FR-014**: The system MUST handle non-interactive terminals by requiring `--yes` or `--force` to skip confirmation prompts, or aborting with a helpful message.
- **FR-015**: Integration tests MUST cover the full upgrade lifecycle: init → customize → init --merge → migrate, verifying no data loss.
- **FR-016**: User-facing documentation MUST describe the recommended upgrade workflow with step-by-step instructions and expected outputs.

### Key Entities

- **Change Summary**: A structured report of file operations grouped by category (new, preserved, up-to-date). Presented to the user before any modifications. Includes both file-level (personas, pipelines, contracts, prompts) and manifest-level (key additions, preservations) information.
- **Manifest Merge Result**: The outcome of deep-merging an existing user manifest with updated defaults. User keys always take precedence. New default keys are added. No keys are removed.
- **Migration Status**: The current state of the database schema relative to the available migration definitions. Includes applied version, pending versions, and integrity check results.

## Clarifications

### C-1: Change Summary Categorization Terminology

**Ambiguity**: User Story 1 acceptance scenario 1 uses "preserved (user-modified)" and "preserved (matches default)", while FR-001 uses "new", "preserved", and "up to date". These are inconsistent terminologies for the same categories.

**Resolution**: Adopt the FR-001 terminology as canonical: **"new"** (file does not exist, will be created), **"preserved"** (file exists, differs from current default — user-modified), **"up to date"** (file exists, matches current default byte-for-byte). User Story 1 acceptance scenarios should be read with this mapping: "preserved (user-modified)" maps to "preserved", "preserved (matches default)" maps to "up to date". The three-category model from FR-001 is clearer and avoids nesting sub-categories under "preserved".

**Rationale**: FR-001 is the normative requirement; user stories are illustrative. The three distinct categories are more actionable for users and easier to implement.

### C-2: Array Merge Strategy for Manifest Deep-Merge

**Ambiguity**: FR-005 requires "deep-merge" of the manifest but does not specify how array/list values (e.g., `allowed_tools`, `deny` lists) should be handled. The existing `mergeMaps` implementation treats arrays atomically — the user's array replaces the default array entirely.

**Resolution**: Array values MUST be treated atomically — the user's existing array value takes precedence and replaces the default. No element-level union or intersection merging. This matches the existing `mergeMaps` behavior and the principle that user customizations always win.

**Rationale**: Element-level array merging is fragile (order matters for deny rules, duplicates are confusing) and violates the principle of least surprise. If a user explicitly defined `allowed_tools: [Read, Write]`, merging in additional defaults would override their intentional restriction. Atomic replacement is the safe default used by tools like Helm, Kustomize, and Docker Compose.

### C-3: `--yes` vs `--force` Semantic Distinction in Merge Context

**Ambiguity**: FR-007 and FR-008 state that `--merge --force` and `--merge --yes` behave identically (skip confirmation, print summary). However, `--force` without `--merge` has a distinct semantic (overwrite everything without prompting). Are they true aliases in merge context or do they retain distinct semantics?

**Resolution**: In merge context, `--yes` and `--force` are functionally identical — both skip the confirmation prompt while retaining merge-safe behavior (preserve user files, add missing defaults). The `--force` flag only has overwrite-all semantics when used **without** `--merge`. When `--merge` is present, it constrains the operation to merge behavior regardless of whether `--force` or `--yes` is used.

**Rationale**: `--merge` is the primary mode selector. Combining `--merge --force` should not silently switch from merge to overwrite — that would be destructive and surprising. This matches the existing code structure where `opts.Merge` takes the `runMerge` path before `opts.Force` is evaluated (init.go:248-249).

### C-4: Post-Merge Migration Guidance

**Ambiguity**: User Story 4 describes a two-step upgrade workflow (init --merge then migrate up). The spec does not state whether `wave init --merge` output should guide users to run `wave migrate up` as a next step, or whether this guidance only lives in documentation.

**Resolution**: The `wave init --merge` success output MUST include `wave migrate up` in its "Next steps" section, alongside the existing `wave validate` suggestion. This provides in-context guidance without requiring users to consult documentation.

**Rationale**: The current `printMergeSuccess` already has a "Next steps" section (init.go:524-526). Adding migration guidance there follows the existing pattern and makes the two-step workflow discoverable. Users upgrading may not realize database migrations exist if they only read CLI output.

### C-5: File Comparison Method for "Up to Date" Detection

**Ambiguity**: FR-001 requires distinguishing "preserved" (user-modified) from "up to date" (matches default). The current code (init.go:740-752) only checks `os.Stat` for existence — it does not compare file contents. The spec does not specify the comparison method.

**Resolution**: File comparison MUST use byte-for-byte content comparison between the existing file and the embedded default. If the file content is identical to the embedded default, it is categorized as "up to date". If content differs in any way (including whitespace, comments, trailing newlines), it is categorized as "preserved". No normalization or fuzzy matching.

**Rationale**: Byte-for-byte comparison is deterministic, simple to implement (just `bytes.Equal`), and avoids false positives from normalization. Since defaults are embedded via `go:embed`, the comparison source is always available. Content hashing would add unnecessary complexity for the same result.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Running `wave init --merge` on a project with 5+ customized files results in zero data loss — all user files preserved, all new defaults added.
- **SC-002**: The change summary correctly categorizes 100% of affected files (no files are modified without appearing in the summary).
- **SC-003**: The user can abort `wave init --merge` at the confirmation prompt and verify zero files were modified (byte-identical before and after).
- **SC-004**: `wave init --merge` followed by `wave migrate up` completes the full upgrade workflow with zero manual intervention (given `--yes` flag).
- **SC-005**: All flag combinations (`--force`, `--merge`, `--force --merge`, `--yes --merge`, no flags) produce documented, consistent behavior verified by tests.
- **SC-006**: Integration test suite covers the upgrade lifecycle with at least 5 test scenarios exercising different customization patterns.
- **SC-007**: Documentation upgrade guide is verified by at least one end-to-end walkthrough on a real project.
