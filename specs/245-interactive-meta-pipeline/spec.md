# Feature Specification: Interactive Meta-Pipeline Orchestrator (`wave run wave`)

**Feature Branch**: `245-interactive-meta-pipeline`  
**Created**: 2026-03-04  
**Status**: Draft  
**Input**: https://github.com/re-cinq/wave/issues/245  
**Version Target**: 1.0.0-rc1  
**Breaking Change**: Yes — backward compatibility is explicitly not a concern

## User Scenarios & Testing _(mandatory)_

### User Story 1 — System Health Dashboard (Priority: P1)

A developer runs `wave run wave` in a Wave-configured repository and receives a comprehensive health report covering initialization status, dependency availability, codebase activity, and hosting platform detection — all gathered in parallel.

**Why this priority**: Without a functioning health check foundation, no other feature in this epic can operate. The health check is the prerequisite for intelligent pipeline proposals, platform routing, and auto-tuning. It also provides standalone value as a diagnostic tool.

**Independent Test**: Can be fully tested by running `wave run wave` in any Wave-configured repository and verifying that the health report is displayed within a reasonable time. Delivers immediate value: users see what's configured, what's missing, and how healthy their project is.

**Acceptance Scenarios**:

1. **Given** a Wave-configured repository with `wave.yaml`, **When** the user runs `wave run wave`, **Then** the system executes initialization check, dependency audit, codebase health analysis, and platform detection as parallel jobs and presents a unified health report.
2. **Given** a repository missing required CLI tools (e.g., `gh` not installed), **When** health checks complete, **Then** the report clearly identifies each missing dependency with its name, purpose, and whether auto-installation is available.
3. **Given** a repository with open GitHub issues and pending PRs, **When** codebase health analysis runs, **Then** the report summarizes recent commit activity, open issue count, PR review status (approved/changes-requested/pending), and overall project health indicators.
4. **Given** a repository hosted on GitLab (detected via remote URL), **When** platform detection runs, **Then** the system correctly identifies GitLab as the hosting platform and notes which platform-specific pipelines are available.

---

### User Story 2 — Interactive Pipeline Proposal & Selection (Priority: P1)

After health checks complete, the system analyzes the codebase state and proposes a ranked list of pipeline runs or sequences. The user selects from an interactive menu — choosing a single pipeline, multiple pipelines for parallel execution, or a pre-composed sequence.

**Why this priority**: This is the core interactive experience that transforms Wave from a manual pipeline tool into an intelligent orchestrator. Without this, `wave run wave` would be just a health checker.

**Independent Test**: Can be tested by running `wave run wave` after health checks pass and verifying that the interactive menu presents contextually relevant pipeline proposals based on codebase state (e.g., open issues suggest `gh-implement`, failing tests suggest `wave-bugfix`).

**Acceptance Scenarios**:

1. **Given** health checks have completed and identified open GitHub issues, **When** the pipeline recommendation engine runs, **Then** it proposes at least one pipeline sequence targeting issue resolution (e.g., `gh-research → gh-implement → wave-land`).
2. **Given** the interactive menu is displayed, **When** the user selects a single pipeline, **Then** that pipeline executes with the recommended input pre-filled and editable.
3. **Given** the interactive menu is displayed, **When** the user marks multiple independent pipelines for parallel execution, **Then** all selected pipelines run concurrently with independent inputs and workspaces.
4. **Given** the recommendation engine proposes a sequence (e.g., `gh-rewrite → recinq → gh-implement`), **When** the user selects that sequence, **Then** the pipelines execute in order with outputs from earlier pipelines automatically available to downstream ones.

---

### User Story 3 — Platform-Aware Pipeline Routing (Priority: P2)

The meta-pipeline detects the repository's hosting platform (GitHub, GitLab, Bitbucket, Gitea) and automatically selects the correct platform-specific pipeline variants. Users never need to manually choose between `gh-implement` and `gl-implement`.

**Why this priority**: Reduces friction for multi-platform users and prevents errors from selecting wrong platform pipelines. Important but depends on health check infrastructure (P1).

**Independent Test**: Can be tested by configuring repositories with different git remote URLs (GitHub, GitLab, Bitbucket, Gitea) and verifying the system selects the correct platform-specific pipeline family each time.

**Acceptance Scenarios**:

1. **Given** a repository with a GitHub remote URL, **When** the system proposes implementation pipelines, **Then** it presents `gh-implement`, `gh-research`, `gh-scope` (not GitLab/Bitbucket/Gitea variants).
2. **Given** a repository with a GitLab remote URL, **When** the system proposes pipelines, **Then** it presents `gl-implement`, `gl-research`, `gl-scope` variants.
3. **Given** a repository with multiple remotes pointing to different platforms, **When** platform detection runs, **Then** the system detects the primary platform (origin remote) and surfaces a notice about additional platforms.
4. **Given** a repository with no recognized platform remote (e.g., self-hosted), **When** platform detection runs, **Then** the system falls back to generic pipelines and reports that platform-specific features are unavailable.

---

### User Story 4 — Dependency Auto-Installation (Priority: P2)

When the health check identifies missing CLI tools or skills required by proposed pipelines, the system offers to auto-install them (where possible) before pipeline execution begins.

**Why this priority**: Removes a common source of pipeline failures. Builds on the dependency audit from P1 but adds the auto-remediation capability.

**Independent Test**: Can be tested by removing a known installable dependency (e.g., a skill), running `wave run wave`, and verifying the system detects the missing dependency and offers installation.

**Acceptance Scenarios**:

1. **Given** a missing skill with a configured `install` command, **When** the dependency audit reports it, **Then** the system offers to install it automatically and proceeds with installation upon user confirmation.
2. **Given** a missing CLI tool without a known install method, **When** the dependency audit reports it, **Then** the system provides the tool name and a message indicating manual installation is required.
3. **Given** auto-installation of a dependency fails, **When** the error is caught, **Then** the system reports the failure clearly and continues with a degraded proposal set (excluding pipelines that require the missing dependency).

---

### User Story 5 — Pipeline Composition & Chaining (Priority: P2)

Users can compose multi-pipeline sequences where the output artifacts of one pipeline feed as input to the next. The meta-pipeline manages artifact handoffs between independently-defined pipelines.

**Why this priority**: Enables the "endlessly self-evolving codebase" vision. Users compose sophisticated workflows from existing pipeline building blocks without writing new pipeline YAML.

**Independent Test**: Can be tested by selecting a two-pipeline sequence (e.g., `gh-research → gh-implement`) and verifying that artifacts from the research pipeline are accessible to the implementation pipeline.

**Acceptance Scenarios**:

1. **Given** a user selects a sequence of two pipelines, **When** the first pipeline completes successfully, **Then** its output artifacts are automatically injected into the second pipeline's workspace as input.
2. **Given** a pipeline sequence where the first pipeline fails, **When** the failure is detected, **Then** the system halts the sequence, reports which pipeline and step failed, and offers options: retry the failed step, skip to the next pipeline, or abort.
3. **Given** an artifact type mismatch between pipeline output and downstream input, **When** the system validates the chain before execution, **Then** it reports the incompatibility and prevents execution with a clear error message.

---

### User Story 6 — Codebase Auto-Tuning (Priority: P3)

The meta-pipeline analyzes the repository to customize personas, pipeline configurations, and contracts to the specific codebase — adapting to project size (small app vs. monorepo), language, framework, and conventions.

**Why this priority**: Enhances the quality of all pipeline executions by tailoring them to the specific project. Depends on health check and platform detection being stable first.

**Independent Test**: Can be tested by running `wave run wave` in repositories of different sizes and languages, then verifying that generated persona configurations reflect the project's characteristics (e.g., test commands, build systems, source globs).

**Acceptance Scenarios**:

1. **Given** a Go monorepo with 50+ packages, **When** auto-tuning runs, **Then** persona prompts are augmented with project-specific context (package structure, test patterns, build commands) without modifying the generic pipeline definitions.
2. **Given** a Python project with pytest, **When** auto-tuning runs, **Then** the `project.test_command`, `project.build_command`, and `project.source_glob` in the runtime context reflect Python conventions.
3. **Given** auto-tuning detects a platform (e.g., GitHub), **When** it creates platform-specific configurations, **Then** it generates new platform-specific pipeline variants rather than modifying generic pipeline definitions.

---

### Edge Cases

- What happens when `wave.yaml` is missing or invalid? The system MUST report a clear initialization error and suggest running `wave init`.
- What happens when the repository has no git remote configured? Platform detection MUST gracefully degrade to "unknown" without crashing.
- What happens when all proposed pipelines are filtered out due to missing dependencies? The system MUST display the health report with a clear message that no pipelines are currently runnable and list what needs to be installed.
- What happens when the user cancels the interactive menu (Ctrl+C / ESC)? The system MUST exit cleanly without error output, consistent with existing TUI behavior.
- What happens when `wave run wave` is invoked in a non-interactive terminal (CI/CD)? The system MUST detect non-TTY and either output the health report in machine-readable JSON format or exit with an error suggesting interactive mode.
- What happens when health checks take too long (e.g., network timeout on API calls)? Individual checks MUST have timeouts and report partial results rather than blocking the entire health report.
- What happens during parallel pipeline execution when one pipeline fails? Other running pipelines MUST continue unless they depend on the failed pipeline's output. The system reports the failure after all independent pipelines complete.

## Requirements _(mandatory)_

### Functional Requirements

#### Phase 1: System & Codebase Health Check

- **FR-001**: System MUST execute initialization check, dependency audit, codebase health analysis, and platform detection as parallel jobs when `wave run wave` is invoked.
- **FR-002**: Initialization check MUST verify the presence and validity of `wave.yaml`, report Wave version, and show the last configuration update date.
- **FR-003**: Dependency audit MUST verify all CLI tools and skills required by available pipelines, reporting each as available, missing (auto-installable), or missing (manual install required).
- **FR-004**: Codebase health analysis MUST gather recent commit history, open issue count, PR status distribution (open/merged/review-pending), and branch activity from the hosting platform's API. For Phase 1, full API-based codebase metrics (issue count, PR status) are supported for GitHub only (using the existing `internal/github/` client). For GitLab, Bitbucket, and Gitea, codebase health reports git-local data only (commit history, branch activity) and notes that platform API integration is not yet available. Platform API clients for non-GitHub platforms are deferred to a follow-up issue.
- **FR-005**: Platform detection MUST identify the hosting platform (GitHub, GitLab, Bitbucket, Gitea, or unknown) by inspecting git remote URLs.
- **FR-006**: Each health check job MUST have an independent timeout so that a single slow check does not block the entire report.
- **FR-007**: Health check results MUST be presented as a unified, structured report after all parallel jobs complete (or time out).

#### Phase 2: Interactive Pipeline Proposal

- **FR-008**: System MUST analyze health check results and generate a ranked list of pipeline proposals based on codebase state (open issues, failing tests, pending reviews, etc.).
- **FR-009**: Each proposal MUST include a pipeline name (or sequence of names), a brief rationale, and pre-filled input based on the analysis.
- **FR-010**: The interactive menu MUST allow the user to select a single proposal for immediate execution.
- **FR-011**: The interactive menu MUST allow the user to mark multiple proposals for parallel execution with independent inputs. Parallel execution spawns independent `DefaultPipelineExecutor` instances (via `NewChildExecutor()`) running concurrently in separate goroutines coordinated by `errgroup`, consistent with the existing matrix execution pattern. Each pipeline gets its own workspace and independent state tracking.
- **FR-012**: For sequence proposals, the system MUST execute pipelines in order, passing output artifacts from each pipeline to the next. Cross-pipeline artifact handoff is implemented by a `SequenceExecutor` that runs each pipeline sequentially and copies output artifacts from the completed pipeline's workspace into the next pipeline's `.wave/artifacts/` directory before execution. This reuses the existing `ArtifactPaths` tracking from `PipelineExecution` without requiring changes to the single-pipeline executor.
- **FR-013**: The system MUST only propose pipelines whose dependencies (tools, skills) are satisfied or auto-installable.
- **FR-014**: Pipeline proposals MUST use platform-specific variants when a platform is detected (e.g., propose `gh-implement` on GitHub, `gl-implement` on GitLab).

#### Phase 3: Codebase Customization

- **FR-015**: Auto-tuning MUST analyze the repository to determine language, framework, test/build commands, and project structure (single app vs. monorepo).
- **FR-016**: Auto-tuning MUST create platform-specific pipeline configurations rather than modifying generic pipeline definitions.
- **FR-017**: Auto-tuning MUST respect existing user customizations in `wave.yaml` — it augments but never overwrites user-defined settings.

#### Cross-Cutting

- **FR-018**: `wave run wave` MUST be implemented as a dedicated special-case handler within the existing `wave run` CLI command, not as a pipeline YAML file. When `wave run` receives `wave` as the pipeline argument, it invokes the interactive meta-orchestrator directly (health check → proposal → execution) rather than loading a YAML pipeline definition. This is necessary because the interactive multi-phase workflow (parallel health checks, user selection, dynamic pipeline dispatch) cannot be expressed as a static DAG of persona-driven steps.
- **FR-019**: System MUST support non-interactive mode (non-TTY) by outputting the health report in JSON format and exiting, or accepting a `--proposal` flag to auto-select a specific proposal.
- **FR-020**: The following legacy/deprecated code MUST be removed as part of this release: (a) `extractYAMLLegacy` backward-compatibility fallback in `internal/pipeline/meta.go` (old meta-pipeline output format without `--- PIPELINE ---`/`--- SCHEMAS ---` markers); (b) legacy template variable handling (spaced variants) in `internal/pipeline/context.go`; (c) legacy workspace directory lookup (exact-name dir without hash suffix) in `internal/pipeline/resume.go`. No pipeline YAML files are deprecated — all existing platform-family pipelines (gh-*, gl-*, gt-*, bb-*) are current and required.
- **FR-021**: System MUST emit structured progress events during all phases for monitoring and observability.

### Key Entities

- **HealthReport**: Aggregated results from all parallel health check jobs. Contains initialization status, dependency inventory, codebase metrics, and detected platform.
- **PipelineProposal**: A recommended pipeline run or sequence with rationale, pre-filled input, estimated complexity, and dependency status. Can represent a single pipeline or an ordered chain.
- **PlatformProfile**: Detected hosting platform identity with available API endpoints, CLI tool, and supported pipeline family (gh/gl/bb/gt).
- **CodebaseProfile**: Analysis of the repository's language, framework, structure, test infrastructure, and size classification (small/medium/large/monorepo).
- **ProposalSelection**: User's choice from the interactive menu — single pipeline, multiple parallel pipelines, or a sequence — with any modified inputs.
- **SequenceExecutor**: Orchestrator for multi-pipeline sequences that runs pipelines in order and manages cross-pipeline artifact handoff by copying output artifacts between workspaces.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: `wave run wave` completes all four health checks (init, deps, codebase, platform) within 30 seconds on a standard repository with network access.
- **SC-002**: Platform detection correctly identifies GitHub, GitLab, Bitbucket, and Gitea from standard remote URL patterns with 100% accuracy for well-formed URLs.
- **SC-003**: The interactive menu presents at least one actionable pipeline proposal for any repository that has at least one runnable pipeline.
- **SC-004**: Pipeline sequences correctly pass artifacts between stages — no artifact loss or type mismatch during handoff.
- **SC-005**: Users can go from `wave run wave` to executing a recommended pipeline in 3 or fewer interactive selections (health report → proposal selection → confirm/execute).
- **SC-006**: All existing project tests continue to pass after the breaking changes, demonstrating that the refactoring preserves core pipeline execution integrity.
- **SC-007**: Non-interactive mode (non-TTY) produces a valid JSON health report that can be consumed by CI/CD tools.
- **SC-008**: Auto-installation of dependencies with configured `install` commands succeeds without manual user intervention beyond initial confirmation.

## Clarifications

The following ambiguities were identified and resolved during spec refinement:

### C-001: Deprecated Format Inventory (FR-020)

**Ambiguity**: FR-020 originally said "exact list of deprecated formats and pipelines to remove needs to be inventoried." The codebase had no clear registry of what was deprecated.

**Resolution**: Inventoried the codebase for legacy/backward-compatibility code. Found three specific items: (a) `extractYAMLLegacy` in `internal/pipeline/meta.go` — a fallback for the old meta-pipeline output format that predates the `--- PIPELINE ---`/`--- SCHEMAS ---` marker format; (b) legacy spaced template variables in `internal/pipeline/context.go`; (c) legacy exact-name workspace directory lookup (no hash suffix) in `internal/pipeline/resume.go`. No pipeline YAML files are deprecated — all 46+ pipelines under `.wave/pipelines/` are current.

**Rationale**: These three items are the only backward-compatibility shims found via codebase grep for "legacy", "deprecated", "old format", and "backward compat" patterns. Removing them is safe because: the new meta-pipeline format has been stable since initial implementation, template variables were standardized, and hashed run IDs are now the only format.

### C-002: Cross-Pipeline Artifact Handoff Mechanism (FR-012)

**Ambiguity**: The spec described pipeline sequences passing artifacts between stages, but the existing `DefaultPipelineExecutor` only handles artifact injection within a single pipeline (via `ArtifactPaths` and `inject_artifacts`). No mechanism existed for cross-pipeline artifact handoff.

**Resolution**: A new `SequenceExecutor` component manages multi-pipeline sequences. After each pipeline completes, the `SequenceExecutor` copies output artifacts from the completed pipeline's workspace (tracked via `PipelineExecution.ArtifactPaths`) into the next pipeline's `.wave/artifacts/` directory. This is a file-copy operation that requires no changes to the single-pipeline executor — it reuses `DefaultPipelineExecutor` as-is and only operates between invocations.

**Rationale**: This approach follows the existing pattern of artifact injection via filesystem (`.wave/artifacts/`), avoids modifying the well-tested single-pipeline executor, and keeps the sequence coordination at a higher abstraction level. The alternative of generating a combined DAG was rejected because independently-defined pipelines may have conflicting step IDs, workspace configurations, and persona assumptions.

### C-003: `wave run wave` Invocation Model (FR-018)

**Ambiguity**: FR-018 originally said "`wave run wave` MUST be invocable as a regular pipeline via the existing `wave run` CLI command," but the interactive multi-phase workflow (parallel health checks, user-interactive proposal selection, dynamic pipeline dispatch) cannot be expressed as a static DAG of persona-driven steps.

**Resolution**: `wave run wave` is implemented as a special-case handler within the `wave run` CLI command. When the pipeline argument is literally `wave`, the command dispatches to the interactive meta-orchestrator instead of loading a YAML pipeline. This is analogous to how `wave meta` already has a dedicated subcommand for meta-pipeline execution. The `wave` keyword is reserved and cannot be used as a pipeline YAML filename.

**Rationale**: The Wave pipeline model (static DAG of persona steps with fresh memory at each boundary) is fundamentally designed for non-interactive, AI-driven execution. The interactive meta-orchestrator needs user input mid-flow (proposal selection), parallel Go-native operations (health checks), and dynamic pipeline dispatch — none of which fit the step-based execution model. A special-case handler is the cleanest approach and mirrors the existing `wave meta` pattern.

### C-004: Health Check Platform API Coverage (FR-004)

**Ambiguity**: FR-004 required codebase health analysis to gather "open issue count, PR status distribution" from the hosting platform's API, but only GitHub has an existing API client (`internal/github/`). GitLab, Bitbucket, and Gitea have personas and pipelines but no equivalent Go API clients.

**Resolution**: Phase 1 health checks provide full platform API metrics (issues, PRs, review status) for GitHub only, using the existing `internal/github/` client. For GitLab, Bitbucket, and Gitea, health checks report git-local data only (commit history, branch activity) and include a note that platform API integration is not yet available. Building Go API clients for three additional platforms is out of scope for the 1.0.0-rc1 release.

**Rationale**: The existing `internal/github/` client provides a proven implementation pattern. Attempting to build API clients for all four platforms simultaneously would delay the release significantly. Git-local data (commit count, branch list, recent activity) still provides meaningful health information. Platform API clients for GL/BB/GT can be added incrementally post-release.

### C-005: Parallel Multi-Pipeline Execution (FR-011)

**Ambiguity**: FR-011 said users can "mark multiple proposals for parallel execution," but the current `DefaultPipelineExecutor` supports parallel steps within a pipeline (via `executeStepBatch` and `errgroup`), not parallel execution of multiple independent pipelines.

**Resolution**: Parallel multi-pipeline execution uses `NewChildExecutor()` to spawn independent `DefaultPipelineExecutor` instances, each running in its own goroutine coordinated by `errgroup.WithContext()`. Each pipeline gets its own workspace, state tracking, and independent error handling. This follows the exact same concurrency pattern already used by the `MatrixExecutor` for child pipeline execution (`internal/pipeline/matrix.go`), which spawns child executors via `NewChildExecutor()` with `errgroup` coordination.

**Rationale**: The `NewChildExecutor()` + `errgroup` pattern is already proven in the matrix execution path. Reusing it for parallel multi-pipeline execution avoids introducing new concurrency primitives and ensures consistent behavior (shared adapter runner, independent state, proper cancellation propagation).
