# Feature Specification: Init Cold-Start Fix, Flavour Auto-Detection, and Smart Init

**Feature Branch**: `403-init-cold-start-flavour`
**Created**: 2026-03-16
**Status**: Clarified
**Input**: User description: "https://github.com/re-cinq/wave/issues/403"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Cold-Start Init in Empty Directory (Priority: P1)

A developer creates a brand-new project directory with no git history and runs `wave init`. Today this crashes because the worktree manager expects `.git`, a commit history, and a remote. The system should gracefully handle all three missing prerequisites and produce a working Wave setup.

**Why this priority**: Without this fix, Wave is completely unusable for greenfield projects. This is a blocking bug that prevents adoption.

**Independent Test**: Run `wave init` in an empty directory and verify it completes without errors, producing a valid `wave.yaml`, `.wave/` structure, and a git repository with at least one commit.

**Acceptance Scenarios**:

1. **Given** an empty directory with no `.git`, **When** `wave init` is run, **Then** the system initializes a git repository (`git init`), writes Wave configuration files, creates an initial commit, and completes successfully.
2. **Given** a directory with `.git` but no commits, **When** `wave init` is run, **Then** the system creates an initial commit with the Wave files and completes successfully.
3. **Given** a directory with `.git` and commits but no remote, **When** `wave init` is run, **Then** the system skips forge detection gracefully, logs a warning, and completes successfully without crashing.
4. **Given** a directory with `.git`, commits, and a remote, **When** `wave init` is run, **Then** the system detects the forge from the remote and proceeds as it does today.

---

### User Story 2 - Automatic Language and Flavour Detection (Priority: P1)

A developer runs `wave init` in a project that already has source code. The system automatically detects the project's programming language and toolchain from marker files (e.g., `go.mod`, `Cargo.toml`, `package.json`) and pre-fills the `project` section of `wave.yaml` with appropriate commands.

**Why this priority**: This is the core value of the flavour system — it eliminates manual configuration for the vast majority of projects and makes Wave immediately useful out of the box.

**Independent Test**: Place a `Cargo.toml` in a directory, run `wave init`, and verify `wave.yaml` contains `project.flavour: rust`, `project.test_command: cargo test`, etc.

**Acceptance Scenarios**:

1. **Given** a directory containing `go.mod`, **When** `wave init` is run, **Then** `wave.yaml` contains `project.flavour: go`, `project.language: go`, `project.test_command: go test ./...`, `project.lint_command: go vet ./...`, `project.build_command: go build ./...`, `project.format_command: gofmt -l .`, `project.source_glob: "*.go"`.
2. **Given** a directory containing `Cargo.toml`, **When** `wave init` is run, **Then** `wave.yaml` contains `project.flavour: rust` with the Rust-specific commands from the detection matrix.
3. **Given** a directory containing `package.json` and `bun.lock`, **When** `wave init` is run, **Then** the system detects the `bun` flavour (not generic `node`) because more specific markers are checked first.
4. **Given** a directory containing no recognized marker files, **When** `wave init` is run, **Then** the system prompts the user for project commands (interactive) or leaves them empty (non-interactive) without erroring.
5. **Given** a directory containing multiple marker files (e.g., `go.mod` and `package.json`), **When** `wave init` is run, **Then** the system uses the first match from the priority-ordered detection list.

---

### User Story 3 - New Manifest Fields for Flavour (Priority: P1)

The `wave.yaml` manifest gains new fields under `project:` to capture the detected flavour and format command. These fields are available as template variables in persona prompts and pipeline definitions.

**Why this priority**: The flavour and format_command fields are prerequisites for both auto-detection (Story 2) and smart init (Story 5). Without the data model change, no downstream feature works.

**Independent Test**: Create a `wave.yaml` with `project.flavour: rust` and `project.format_command: cargo fmt -- --check`, load it, and verify `ProjectVars()` returns the new keys.

**Acceptance Scenarios**:

1. **Given** a `wave.yaml` with `project.flavour: rust`, **When** the manifest is loaded, **Then** `project.Flavour` is populated and `ProjectVars()` includes `"project.flavour": "rust"`.
2. **Given** a `wave.yaml` with `project.format_command: cargo fmt -- --check`, **When** the manifest is loaded, **Then** `project.FormatCommand` is populated and available as template variable `project.format_command`.
3. **Given** a `wave.yaml` without `project.flavour`, **When** the manifest is loaded, **Then** the field defaults to empty string and no error is raised (backward compatible).

---

### User Story 4 - Forge-Filtered Persona and Pipeline Selection (Priority: P2)

During `wave init`, the system detects the git forge (GitHub, GitLab, Bitbucket, Gitea) from the remote URL and only includes personas and pipelines relevant to that forge.

**Why this priority**: Important for a clean setup, but the system works without it (users just see extra unused pipelines). Not blocking for MVP.

**Independent Test**: Initialize Wave in a repo with a GitHub remote and verify only forge-agnostic pipelines and any forge-matching prefixed pipelines are included in the selection.

**Acceptance Scenarios**:

1. **Given** a repository with a GitHub remote, **When** `wave init` is run, **Then** only forge-agnostic pipelines (no forge prefix) and `gh-*` prefixed pipelines (if any local overrides exist) are included in the pipeline selection. Note: after the pipeline unification (PR #375), most embedded pipelines are forge-agnostic and pass through unfiltered.
2. **Given** a repository with a GitLab remote, **When** `wave init` is run, **Then** only forge-agnostic and `gl-*` prefixed pipelines are included.
3. **Given** a repository with no remote, **When** `wave init` is run, **Then** all pipelines are included (no filtering applied).
4. **Given** a repository with a recognized forge, **When** `wave init` is run in non-interactive mode (`--yes`), **Then** forge filtering is applied automatically without prompts.

---

### User Story 5 - Project Metadata Extraction (Priority: P2)

The system extracts the project name and description from language-specific manifest files (`Cargo.toml`, `go.mod`, `package.json`, `pyproject.toml`, etc.) and populates `metadata.name` and `metadata.description` in `wave.yaml`.

**Why this priority**: Nice UX polish that reduces manual entry, but not blocking for core functionality.

**Independent Test**: Place a `package.json` with `"name": "my-app"` and `"description": "A cool app"`, run `wave init`, and verify `wave.yaml` metadata fields are populated.

**Acceptance Scenarios**:

1. **Given** a `package.json` with `name` and `description` fields, **When** `wave init` is run, **Then** `wave.yaml` metadata contains the extracted name and description.
2. **Given** a `go.mod` with module path `github.com/org/repo`, **When** `wave init` is run, **Then** the project name is extracted as `repo` from the module path.
3. **Given** a `Cargo.toml` with `[package]` section containing `name` and `description`, **When** `wave init` is run, **Then** metadata is populated from those fields.
4. **Given** no recognized manifest file, **When** `wave init` is run, **Then** the directory name is used as the project name.

---

### User Story 6 - First-Run Suggestion (Priority: P3)

After initialization completes, the system suggests an appropriate first pipeline to run based on project state: `ops-bootstrap` for empty projects, `audit-dx` for projects with existing code.

**Why this priority**: Pure UX enhancement — helpful but not required for the init flow to function.

**Independent Test**: Run `wave init` in a directory with existing source code and verify the completion message suggests `audit-dx`.

**Acceptance Scenarios**:

1. **Given** an empty project (no source files beyond Wave config), **When** `wave init` completes, **Then** the success message suggests running `wave run ops-bootstrap`.
2. **Given** a project with existing source files, **When** `wave init` completes, **Then** the success message suggests running `wave run audit-dx`.

---

### User Story 7 - Ensuring impl-issue Pipeline Inclusion (Priority: P2)

The `impl-issue` pipeline is always included in the pipeline set regardless of any filtering logic, since it was previously missing from some init configurations.

**Why this priority**: This is a bug fix — `impl-issue` is the most commonly used pipeline and must always be available.

**Independent Test**: Run `wave init` with any combination of flags and verify `impl-issue` is present in the generated pipeline set.

**Acceptance Scenarios**:

1. **Given** any project configuration, **When** `wave init` is run, **Then** the `impl-issue` pipeline is included in `.wave/pipelines/`.
2. **Given** a forge-filtered init (e.g., GitLab), **When** pipelines are filtered, **Then** `impl-issue` survives the filter because it has no forge prefix.

---

### Edge Cases

- What happens when `git init` fails (e.g., permissions issue)? The system reports a clear error message and exits without writing partial configuration.
- What happens when multiple package managers are detected (e.g., both `yarn.lock` and `pnpm-lock.yaml`)? The first match in priority order wins; the more specific marker takes precedence.
- What happens when `wave init` is run in a directory that already has `wave.yaml`? Existing behavior is preserved — prompt for overwrite or use `--force`/`--merge`.
- What happens when the detected flavour's tool (e.g., `cargo`) is not installed? The flavour is still set in the manifest; the dependency check step warns about missing tools but does not block initialization.
- What happens when `wave init --yes` is used in an empty directory? Cold-start proceeds fully non-interactively: `git init`, detect language, write config, create commit.
- What happens when the remote URL uses an enterprise GitHub domain (e.g., `github.mycompany.com`)? Forge detection correctly classifies it as GitHub via existing `classifyHost()` logic.
- What happens when `wave init --reconfigure` is used and the flavour has changed (e.g., project migrated from npm to bun)? The system re-detects the flavour and updates the project section.

## Clarifications

### Q1: Where does the cold-start git init logic live?

**Ambiguity**: The spec says `wave init` should run `git init` when `.git` is absent, but doesn't specify whether this logic belongs in `cmd/wave/commands/init.go` (the `runInit` function) or in the `internal/onboarding` package.

**Resolution**: The cold-start logic (git init, initial commit) belongs in `cmd/wave/commands/init.go` in the `runInit` function, **before** the onboarding wizard is invoked. The onboarding wizard (`internal/onboarding`) is concerned with configuration collection, not git repository setup. The `runInit` function already orchestrates the high-level flow (check existing files, invoke wizard, write assets) and is the natural place for pre-wizard git bootstrapping.

**Rationale**: Follows existing separation of concerns — `runInit` handles prerequisites and orchestration, the wizard handles interactive configuration. Cold-start is a prerequisite, not a configuration step.

### Q2: Should flavour detection extend `detectProjectType` or create a new package?

**Ambiguity**: The existing `detectProjectType()` in `internal/onboarding/steps.go` covers only 5 languages and returns `map[string]string` without a `flavour` key. The spec requires 25+ languages with a `flavour` field. Should this be an in-place extension or a new `internal/flavour` package?

**Resolution**: Create a new `internal/flavour` package with a `Detect()` function that returns a structured `DetectionResult` type (not a raw map). The existing `detectProjectType()` in `internal/onboarding/steps.go` should be replaced with a call to `flavour.Detect()`. The new package provides:
- A `DetectionResult` struct with fields: `Flavour`, `Language`, `TestCommand`, `LintCommand`, `BuildCommand`, `FormatCommand`, `SourceGlob`
- A priority-ordered slice of `DetectionRule` structs (marker files → result)
- A `DetectMetadata()` function for project name/description extraction (Story 5)

**Rationale**: The flavour detection matrix is a standalone concern with its own test surface. A dedicated package keeps the onboarding package focused on wizard flow and makes the detection logic reusable by other components (e.g., `wave doctor`, `wave suggest`).

### Q3: How does forge-filtered pipeline selection interact with the unified pipeline naming?

**Ambiguity**: Story 4 references `gh-*` prefixed pipelines, but the pipeline unification (PR #375, #380) replaced forge-specific pipelines with forge-agnostic unified pipelines (e.g., `impl-issue` instead of `gh-implement`). The `FilterPipelinesByForge` function in `internal/forge/detect.go` already exists and handles prefix-based filtering. Are there still forge-prefixed pipelines to filter?

**Resolution**: After the pipeline unification, most pipelines are forge-agnostic (no forge prefix). Forge-filtered selection during init should use the existing `forge.FilterPipelinesByForge()` function, which already correctly passes through pipelines without forge prefixes. The practical effect is: if a user has custom `gh-*` or `gl-*` local pipeline overrides, those get filtered; the default embedded pipelines are all forge-agnostic and pass through unfiltered. Story 4's acceptance scenarios should be read as applying to the **full pipeline catalog** (embedded + local overrides), not just embedded pipelines. Update acceptance scenario wording accordingly.

**Rationale**: The existing `FilterPipelinesByForge` already implements the correct logic — include pipelines matching the forge prefix OR having no forge prefix. No new filtering logic is needed; the init flow just needs to call this function during pipeline selection.

### Q4: Does `impl-issue` actually need special inclusion logic (Story 7)?

**Ambiguity**: Story 7 says `impl-issue` must "always be included regardless of filtering logic," but `impl-issue` has no forge prefix. The existing `FilterPipelinesByForge` already includes all pipelines without a forge prefix. Is Story 7 already satisfied by the existing forge filter, or is there a separate filtering path that could exclude it?

**Resolution**: Story 7 is already satisfied by the existing `forge.FilterPipelinesByForge()` behavior — pipelines without a forge prefix (like `impl-issue`) always pass through. However, there is a separate risk: the `getFilteredAssets()` function in `init.go` filters by `release: true` metadata. If `impl-issue` were ever not marked as `release: true`, it would be excluded. The implementation should add a `requiredPipelines` safeguard in the pipeline selection step that ensures `impl-issue` (and potentially other essential pipelines) are always included regardless of release filtering. This is a defense-in-depth measure.

**Rationale**: Belt-and-suspenders approach. The forge filter already handles it, but the release filter is a separate code path that could exclude essential pipelines. A small `requiredPipelines` list is cheap insurance.

### Q5: What is the initial commit message and what files does it include?

**Ambiguity**: FR-002 says "create an initial git commit after writing Wave configuration files" but doesn't specify the commit message, which files to stage, or whether the commit should include only Wave files or all files in the directory.

**Resolution**: The initial commit should:
- Stage only Wave-generated files: `wave.yaml`, `.wave/` directory contents
- Use the commit message: `chore: initialize wave project`
- NOT stage any pre-existing user files (source code, etc.) — the user should control what goes into their repository
- If the directory is truly empty (no prior files), the commit contains only Wave configuration

**Rationale**: Staging only Wave files respects user intent — they haven't asked Wave to commit their source code. The conventional commit prefix `chore:` matches the project's commit style. This also avoids accidentally committing sensitive files.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST check for `.git` directory existence and run `git init` if absent during `wave init`.
- **FR-002**: System MUST create an initial git commit after writing Wave configuration files when the repository has no prior commits.
- **FR-003**: System MUST handle missing git remote gracefully during forge detection, falling back to `ForgeUnknown` without crashing.
- **FR-004**: System MUST detect project language and toolchain from at least 25 marker file patterns as specified in the detection matrix.
- **FR-005**: System MUST check more specific markers before generic ones (e.g., `bun.lock` before `package.json`) using priority-ordered detection.
- **FR-006**: System MUST add `flavour` and `format_command` fields to the `Project` struct in the manifest type system.
- **FR-007**: System MUST expose `project.flavour` and `project.format_command` as template variables via `ProjectVars()`.
- **FR-008**: System MUST detect the git forge from remote URLs and filter pipelines to only those matching the detected forge during init.
- **FR-009**: System MUST extract project name and description from language-specific manifest files when available.
- **FR-010**: System MUST always include the `impl-issue` pipeline in the pipeline set regardless of forge filtering or other selection criteria.
- **FR-011**: System MUST preserve backward compatibility — existing `wave.yaml` files without new fields MUST load without error.
- **FR-012**: System MUST suggest an appropriate first pipeline based on project state after init completes.
- **FR-013**: System MUST support the full detection matrix: go, rust, node (npm/yarn/pnpm/bun), deno, python (modern/legacy), C#, Java (Maven/Gradle), Kotlin, Elixir, Dart/Flutter, C++ (CMake), Make, PHP, Ruby, Swift, Zig, Scala, Haskell (Cabal/Stack), TypeScript standalone.

### Key Entities

- **Flavour**: A named project type (e.g., `go`, `rust`, `bun`, `python`) derived from marker file detection. Determines default commands for test, lint, build, format, and source glob. Stored in `project.flavour`.
- **Detection Rule**: A mapping from one or more marker files to a flavour with associated default commands. Rules are ordered by specificity — more specific markers are checked before generic ones.
- **Project Metadata**: Name and description extracted from language-specific manifest files (e.g., `package.json`, `Cargo.toml`), used to pre-populate `wave.yaml` metadata.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: `wave init` in an empty directory (no `.git`, no files) completes without error and produces a valid `wave.yaml` and `.wave/` directory with a git repository containing at least one commit.
- **SC-002**: All 25+ language flavours from the detection matrix are correctly identified when their marker files are present.
- **SC-003**: The generated `wave.yaml` contains correct `project.flavour`, `project.language`, and all command fields for every detected flavour.
- **SC-004**: Forge-filtered pipeline selection excludes pipelines for non-matching forges while retaining forge-agnostic pipelines.
- **SC-005**: All existing `wave init` functionality (interactive wizard, `--merge`, `--force`, `--reconfigure`, `--all`, `--yes`) continues to work without regression.
- **SC-006**: `project.flavour` and `project.format_command` are available as template variables in persona prompts and pipeline definitions.
