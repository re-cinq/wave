# Feature Specification: Skill Dependency Installation in Pipeline Steps

**Feature Branch**: `102-skill-deps-pipeline`
**Created**: 2026-02-14
**Status**: Draft
**Input**: [GitHub Issue #97](https://github.com/re-cinq/wave/issues/97) — Support external dependency installation (slash commands, speckit) in pipeline steps

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Declare and Auto-Install Pipeline Skills (Priority: P1)

As a pipeline author, I want to declare external skill dependencies (e.g., Speckit, BMAD, OpenSpec) in my pipeline configuration so that they are automatically installed before any step that needs them executes.

**Why this priority**: Without automatic installation, pipelines fail at runtime when external tools are missing. This is the core problem described in the issue and blocks all other functionality.

**Independent Test**: Can be fully tested by creating a pipeline that declares a skill dependency, running the pipeline on a system where the skill is not yet installed, and verifying that the skill is automatically installed and the step succeeds.

**Acceptance Scenarios**:

1. **Given** a pipeline declares `requires.skills: [speckit]` and speckit is not installed, **When** the pipeline is executed, **Then** the preflight phase installs speckit using the configured install command before any step runs, and the step succeeds.
2. **Given** a pipeline declares `requires.skills: [speckit, bmad]` and both are already installed, **When** the pipeline is executed, **Then** the preflight phase detects they are present (via check commands), skips installation, and proceeds to step execution without delay.
3. **Given** a pipeline declares `requires.skills: [speckit]` but speckit's install command fails, **When** the pipeline is executed, **Then** the pipeline fails with a clear error message identifying which skill failed to install, the install command that was attempted, and the error output.

---

### User Story 2 - Declare CLI Tool Dependencies (Priority: P1)

As a pipeline author, I want to declare required CLI tools (e.g., `git`, `go`, `node`) in my pipeline configuration so that the pipeline fails fast with a clear message if a required tool is missing from PATH.

**Why this priority**: Tool availability checks prevent confusing mid-execution failures and provide actionable error messages. This is equally critical to skill installation as it addresses the same class of dependency failures.

**Independent Test**: Can be fully tested by creating a pipeline that declares a tool dependency for a binary not on PATH, running the pipeline, and verifying it fails immediately with a message naming the missing tool.

**Acceptance Scenarios**:

1. **Given** a pipeline declares `requires.tools: [git, go]` and both are on PATH, **When** the pipeline is executed, **Then** preflight validation passes and step execution proceeds.
2. **Given** a pipeline declares `requires.tools: [nonexistent-tool]`, **When** the pipeline is executed, **Then** the pipeline fails before any step runs with an error message stating `nonexistent-tool` was not found on PATH.
3. **Given** a pipeline declares both `requires.tools` and `requires.skills`, **When** the pipeline is executed, **Then** both tool availability and skill installation are validated in the preflight phase before any step runs.

---

### User Story 3 - Per-Step Skill Provisioning (Priority: P2)

As a pipeline author, I want skill command files (slash commands like `/speckit.specify`) to be automatically provisioned into each step's workspace so that agents running in those steps can invoke the skill commands.

**Why this priority**: Installation alone is insufficient — skill commands must also be discoverable within each step's isolated workspace. This completes the dependency lifecycle.

**Independent Test**: Can be fully tested by running a pipeline step that invokes a slash command (e.g., `/speckit.specify`) and verifying the command file is present in the workspace's `.claude/commands/` directory and the agent can execute it.

**Provisioning chain**: The `commands_glob` pattern is resolved relative to the main project repository root (where `wave.yaml` lives). The skill `Provisioner` copies matching command files into a staging directory (`.wave-skill-commands/.claude/commands/`) within the step workspace. The adapter layer then copies these staged files into the adapter's own settings directory (e.g., `.claude/commands/` for Claude Code). This two-stage approach keeps the provisioner adapter-agnostic while ensuring commands are discoverable by the specific adapter.

**Acceptance Scenarios**:

1. **Given** a pipeline declares `requires.skills: [speckit]` and speckit defines `commands_glob: ".claude/commands/speckit.*.md"`, **When** a step workspace is created, **Then** all matching command files are staged in the workspace and made available to the adapter's command directory.
2. **Given** a pipeline has multiple steps and declares `requires.skills: [speckit]`, **When** each step workspace is created, **Then** skill commands are provisioned independently into each workspace (fresh copy per step, no shared state).
3. **Given** a step workspace uses `type: worktree`, **When** skill commands are provisioned, **Then** the command files are placed in the worktree's staging directory and do not pollute the main repository.

---

### User Story 4 - Skill Definition in Manifest (Priority: P2)

As a Wave administrator, I want to define external skills with install, init, and check commands in `wave.yaml` so that the pipeline system knows how to manage their lifecycle.

**Why this priority**: The manifest is the single source of truth for configuration. Skills must be declarable there for the preflight system to manage them.

**Independent Test**: Can be fully tested by adding a skill definition to `wave.yaml`, running manifest validation, and verifying the skill is recognized with its install/check/init commands.

**Acceptance Scenarios**:

1. **Given** a `wave.yaml` contains a skill definition with `check`, `install`, and `commands_glob` fields, **When** the manifest is loaded, **Then** the skill configuration is parsed and validated.
2. **Given** a `wave.yaml` contains a skill definition without a `check` command, **When** the manifest is loaded, **Then** validation fails with an error indicating the `check` field is required and suggesting how to add one.
3. **Given** a `wave.yaml` contains a skill definition with an `init` command, **When** the skill is installed for the first time, **Then** the init command runs after the install command succeeds.

---

### User Story 5 - Preflight Progress Reporting (Priority: P3)

As a pipeline operator, I want to see real-time progress events during dependency installation so that I can monitor which dependencies are being checked, installed, or have failed.

**Why this priority**: Observability is a core Wave principle but not required for basic functionality. This enhances the user experience for long-running installs.

**Independent Test**: Can be fully tested by running a pipeline with skill dependencies and verifying that progress events are emitted for each dependency check and installation action.

**Acceptance Scenarios**:

1. **Given** a pipeline with 3 skill dependencies, **When** preflight runs, **Then** a progress event is emitted for each skill showing its status (checking, installing, installed, failed).
2. **Given** a skill requires installation (check fails initially), **When** the install command runs, **Then** progress events report the install phase start, completion, and the subsequent re-check result.

---

### Edge Cases

- What happens when a skill's `check` command succeeds but the skill's command files cannot be found by the `commands_glob` pattern? The system logs a warning but does not fail — the skill binary is available even if no slash commands are defined for it.
- What happens when a skill's `install` command succeeds but the subsequent `check` command still fails? The system fails the preflight with a clear error indicating the install appeared to succeed but the check still fails, suggesting the install command or check command may be incorrect.
- What happens when two pipelines run concurrently and both try to install the same skill? The install is idempotent — if the skill is already installed (check passes), the second pipeline skips installation. No locking mechanism is required since install commands are expected to be idempotent.
- What happens when a skill is declared in `requires.skills` but not defined in the `skills` section of `wave.yaml`? The preflight phase fails with a validation error naming the undeclared skill and suggesting it be added to the manifest's `skills` section.
- What happens when the `init` command fails after a successful `install`? The preflight fails with an error distinguishing init failure from install failure, including the init command's error output.
- What happens when a required tool exists on PATH but is the wrong version? Tool checks are presence-only (PATH lookup). Version validation is out of scope — pipeline authors should use skill definitions with custom check commands if version-specific validation is needed.

## Approach Selection

Issue #97 proposes three approaches for managing external dependencies:

- **Option A: Workspace-wide installation** — Install all dependencies once at the workspace level. Rejected because it installs tools unnecessarily for steps that do not need them and couples workspace setup to pipeline-specific requirements.
- **Option B: Per-step installation with handover** — Each step installs its own dependencies. Rejected because it causes redundant installations when multiple steps share the same skill, and adds complexity to the artifact handover logic.
- **Option C: Preflight dependency phase** (selected) — A dedicated preflight phase installs all declared dependencies before any step runs. Selected because it provides clean separation of concerns, avoids redundant installations, enables fail-fast behavior (pipeline fails before any step starts if dependencies cannot be satisfied), and aligns with Wave's existing `preflight` package patterns.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST allow pipelines to declare required skills via a `requires.skills` list in the pipeline YAML file (alongside `kind`, `metadata`, `input`, and `steps`).
- **FR-002**: System MUST allow pipelines to declare required CLI tools via a `requires.tools` list in the pipeline YAML file (same `requires` block as FR-001).
- **FR-003**: System MUST validate all declared skills exist in the manifest's `skills` section before executing any pipeline step.
- **FR-004**: System MUST run a preflight phase before step execution that checks and installs all declared skill dependencies.
- **FR-005**: System MUST verify tool availability using PATH lookup during the preflight phase.
- **FR-006**: System MUST ensure declared skills are installed, initialized (if configured), and verified before step execution begins. If a skill is already available, no installation action is taken.
- **FR-007**: System MUST ensure skill commands are available in each step's workspace for agent invocation via the two-stage provisioning chain (provisioner stages into workspace, adapter copies to settings directory).
- **FR-008**: System MUST require a `check` command for every declared skill in the manifest. The `install` and `init` commands are optional — if `install` is omitted and the skill's `check` fails, the preflight reports a failure indicating no install command is configured.
- **FR-009**: System MUST fail the pipeline with a descriptive error when any preflight dependency check fails and cannot be auto-resolved.
- **FR-010**: System MUST emit structured progress events for each dependency check and installation action. Events use state `"preflight"` with per-dependency messages. A new `StatePreflight` constant should be added to the event package for consistency with other state constants (e.g., `StateStarted`, `StateCompleted`).
- **FR-011**: System MUST support at minimum three external skills: Speckit, BMAD, and OpenSpec.
- **FR-012**: System MUST ensure each step workspace has its own independent set of skill commands, isolated from other steps.

### Key Entities

- **Skill**: An external tool that can be declared as a pipeline dependency, with lifecycle management for installation verification and workspace provisioning. Declared in the manifest and referenced by pipelines.
- **Pipeline Requires**: A declaration block within a pipeline configuration that lists the skill names and CLI tool names needed for the pipeline's steps to execute.
- **Preflight Result**: The outcome of validating a single dependency, indicating whether it is satisfied and providing diagnostic information if not. Includes the dependency name and its kind (skill or tool).
- **Skill Command**: A command file associated with a skill that is provisioned into step workspaces to enable agent invocation of skill-provided operations.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: A pipeline declaring `requires.skills: [speckit]` on a system without speckit installed completes successfully after auto-installing the skill during preflight.
- **SC-002**: A pipeline declaring `requires.tools: [nonexistent]` fails within the preflight phase (before any step starts) with an error message naming the missing tool.
- **SC-003**: All three target skills (Speckit, BMAD, OpenSpec) can be declared, installed, and provisioned in pipelines without code changes to the core system.
- **SC-004**: Skill command files are discoverable by agents in the step workspace after provisioning, verifiable by file existence checks.
- **SC-005**: The preflight phase emits at least one structured progress notification per declared dependency, observable via structured progress output.
- **SC-006**: Pipeline execution with all dependencies pre-installed adds no more than 500ms overhead from preflight checks (tool PATH lookups and skill check commands).

## Clarifications

The following ambiguities were identified during specification review and resolved based on existing codebase patterns:

### C1: Skill command provisioning chain (User Story 3, FR-007)

**Ambiguity**: The original spec stated commands are "copied into the step's `.claude/commands/` directory," but the existing implementation (`internal/pipeline/executor.go:516-528`) uses a two-stage approach: commands are first staged in `.wave-skill-commands/` within the workspace, then the adapter copies them to its settings directory.

**Resolution**: Updated User Story 3 and FR-007 to document the two-stage provisioning chain. This is the correct design because it keeps the `skill.Provisioner` adapter-agnostic — it doesn't need to know where Claude Code (or any other adapter) stores its commands. The adapter layer handles the final placement.

**Rationale**: Follows the existing pattern in `internal/adapter/claude.go` where `SkillCommandsDir` is consumed by `copySkillCommands()`.

### C2: `requires` declaration location (FR-001, FR-002)

**Ambiguity**: The spec used the phrase "pipeline configuration" without specifying whether `requires` lives in the pipeline YAML file or in `wave.yaml`.

**Resolution**: Clarified that `requires` is a top-level field in the pipeline YAML file (alongside `kind`, `metadata`, `input`, and `steps`). This matches the existing `Requires` field on the `Pipeline` struct at `internal/pipeline/types.go:14` and the existing `Requires` YAML struct at `types.go:20-23`.

**Rationale**: Skills are defined in `wave.yaml` (global), but which skills a pipeline needs is a pipeline-level concern. This separation matches the existing manifest/pipeline architecture.

### C3: `install` command optionality (FR-008, User Story 4)

**Ambiguity**: FR-008 required `check` for every skill, but the spec did not explicitly state whether `install` is required or optional in `SkillConfig`.

**Resolution**: Clarified that `install` and `init` are optional. If a skill's `check` fails and no `install` command is configured, the preflight reports a clear error stating the skill is not installed and no install command is available. This matches the existing `SkillConfig` struct where `Install` has `yaml:"install,omitempty"` (`internal/manifest/types.go:144`) and the preflight logic at `internal/preflight/preflight.go:100-109`.

**Rationale**: Some skills may be pre-installed (e.g., system packages), needing only `check` to verify presence without an automatic install mechanism.

### C4: Preflight event state naming (FR-010, User Story 5)

**Ambiguity**: FR-010 required "structured progress events" but did not specify the event state name to use.

**Resolution**: Clarified that preflight events use state `"preflight"`, matching the existing executor code at `internal/pipeline/executor.go:173`. Recommended adding a `StatePreflight` constant to `internal/event/emitter.go` for consistency with other state constants (`StateStarted`, `StateCompleted`, etc.).

**Rationale**: The existing codebase uses string literals for the preflight state. Adding a named constant follows the established pattern and prevents typo-related bugs.

### C5: `commands_glob` resolution root (User Story 3)

**Ambiguity**: The spec stated `commands_glob: ".claude/commands/speckit.*.md"` without specifying the base directory for glob resolution.

**Resolution**: Clarified in the provisioning chain description that `commands_glob` resolves relative to the main project repository root (where `wave.yaml` lives). This is where skill command files are stored as part of the project's `.claude/commands/` directory. The `Provisioner` accepts a `repoRoot` parameter for this purpose (`internal/skill/skill.go:16,52`).

**Rationale**: Skill commands are project-level assets stored in the repository. Resolving relative to the repo root ensures consistent discovery regardless of which workspace directory a step uses.
