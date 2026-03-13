# Feature Specification: Unify Platform-Specific Pipelines

**Feature Branch**: `241-unify-platform-pipelines`
**Created**: 2026-03-13
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/241 — unify platform-specific implement pipelines into single configurable pipelines with skill-based flavoring

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Run Unified Pipeline on Any Forge (Priority: P1)

As a developer using Wave on a GitHub/GitLab/Gitea/Bitbucket repository, I want to run `wave run implement -- "<issue-url>"` without needing to know or specify the platform-prefixed pipeline name, so that pipelines work automatically based on my repository's detected forge.

**Why this priority**: This is the core value proposition — eliminating the 4× duplication and making pipelines forge-agnostic. Without this, all other stories are moot.

**Independent Test**: Can be fully tested by running `wave run implement` on repositories hosted on each of the four forge platforms and verifying that the correct forge-specific behavior (CLI tool, PR/MR terminology, API commands) is automatically applied.

**Acceptance Scenarios**:

1. **Given** a repository hosted on GitHub, **When** the user runs `wave run implement -- "https://github.com/org/repo/issues/42"`, **Then** the unified `implement` pipeline executes with GitHub-specific behavior (uses `gh` CLI, creates a "Pull Request", references `gh pr create`)
2. **Given** a repository hosted on GitLab, **When** the user runs `wave run implement -- "https://gitlab.com/org/repo/-/issues/42"`, **Then** the unified `implement` pipeline executes with GitLab-specific behavior (uses `glab` CLI, creates a "Merge Request", references `glab mr create`)
3. **Given** a repository hosted on Bitbucket, **When** the user runs `wave run implement -- "https://bitbucket.org/workspace/repo/issues/42"`, **Then** the unified `implement` pipeline executes with Bitbucket-specific behavior (uses Bitbucket REST API via `curl`/`jq`, creates a "Pull Request")
4. **Given** a repository hosted on Gitea, **When** the user runs `wave run implement -- "https://gitea.example.com/org/repo/issues/42"`, **Then** the unified `implement` pipeline executes with Gitea-specific behavior (uses `tea` CLI, creates a "Pull Request")
5. **Given** a repository with an unrecognized forge, **When** the user runs `wave run implement`, **Then** the system reports a clear error indicating the forge could not be detected and suggests manual configuration

---

### User Story 2 - Forge Template Variables in Prompts (Priority: P1)

As a pipeline author, I want to use `{{ forge.type }}`, `{{ forge.cli_tool }}`, `{{ forge.pr_term }}`, and other forge-scoped template variables in pipeline definitions and prompt files, so that a single prompt file can adapt to the detected platform without duplicating content.

**Why this priority**: This is the technical enabler for unification — without template variables, prompts cannot dynamically adapt to the forge.

**Independent Test**: Can be tested by creating a pipeline with `{{ forge.cli_tool }}` in a prompt template, running it on a GitHub repo, and verifying the placeholder resolves to `gh`.

**Acceptance Scenarios**:

1. **Given** a prompt containing `{{ forge.cli_tool }}`, **When** executed on a GitHub repository, **Then** the placeholder resolves to `gh`
2. **Given** a prompt containing `{{ forge.pr_term }}`, **When** executed on a GitLab repository, **Then** the placeholder resolves to `Merge Request`
3. **Given** a prompt containing `{{ forge.host }}`, **When** executed on a Bitbucket repository, **Then** the placeholder resolves to `bitbucket.org`
4. **Given** a prompt containing `{{ forge.owner }}` and `{{ forge.repo }}`, **When** executed, **Then** both resolve to the correct repository owner and name
5. **Given** a prompt containing an unresolvable `{{ forge.nonexistent }}`, **When** executed, **Then** the placeholder is left unresolved (consistent with existing template behavior)

---

### User Story 3 - Forge-Specific Persona Resolution (Priority: P1)

As the pipeline executor, I need to resolve the correct forge-specific persona (e.g., `github-commenter` vs `gitlab-commenter`) at runtime based on the detected forge, so that unified pipelines can reference personas dynamically without hardcoding platform names.

**Why this priority**: Personas contain platform-specific CLI tool permissions and prompt instructions. Unified pipelines must resolve the right persona per forge.

**Independent Test**: Can be tested by defining a unified pipeline step with a dynamic persona reference and verifying that the correct persona (with correct tool permissions) is loaded for each forge type.

**Acceptance Scenarios**:

1. **Given** a unified pipeline step with `persona: "{{ forge.prefix }}-commenter"`, **When** executed on a GitHub repository, **Then** the `github-commenter` persona is loaded with `Bash(gh *)` permissions
2. **Given** a unified pipeline step with `persona: "{{ forge.prefix }}-analyst"`, **When** executed on a Bitbucket repository, **Then** the `bitbucket-analyst` persona is loaded with `Bash(curl *)` and `Bash(jq *)` permissions
3. **Given** a forge-specific persona that does not exist (e.g., `unknown-commenter`), **When** the pipeline attempts to load it, **Then** execution fails with a clear error message identifying the missing persona

---

### User Story 4 - Unified Prompt Files with Forge Variables (Priority: P2)

As a pipeline maintainer, I want each pipeline family to have a single set of prompt files (instead of 4 copies) that use forge template variables for platform-specific CLI commands, so that prompt maintenance is centralized and bug fixes apply to all platforms simultaneously.

**Why this priority**: This is the primary maintenance benefit — fixing a prompt bug once instead of in 4 places. Lower priority than P1 stories because it's the natural consequence of the template variable system.

**Independent Test**: Can be tested by comparing the unified prompt output for each forge against the current platform-specific prompts and verifying behavioral equivalence.

**Acceptance Scenarios**:

1. **Given** a unified `implement/create-pr.md` prompt, **When** rendered for GitHub, **Then** the output contains `gh pr create` commands and "Pull Request" terminology
2. **Given** a unified `implement/create-pr.md` prompt, **When** rendered for GitLab, **Then** the output contains `glab mr create` commands and "Merge Request" terminology
3. **Given** a unified `implement/fetch-assess.md` prompt, **When** rendered for Bitbucket, **Then** the output contains the Bitbucket REST API curl commands with `$BB_TOKEN` authentication

---

### User Story 5 - PR Review Pipeline for All Forges (Priority: P2)

As a developer using GitLab, Gitea, or Bitbucket, I want to run `wave run pr-review` on my repository, so that I get the same code review pipeline that GitHub users already have.

**Why this priority**: Currently only `gh-pr-review` exists. Extending PR review to all forges is a natural outcome of unification and fills a feature gap.

**Independent Test**: Can be tested by running `wave run pr-review` against a merge request on GitLab and verifying that all review steps (diff analysis, security scan, quality review, summary) execute and the review is published as a comment.

**Acceptance Scenarios**:

1. **Given** a GitLab repository with an open merge request, **When** `wave run pr-review -- "<MR-URL>"` is executed, **Then** the review is published as a comment on the merge request using `glab`
2. **Given** a Bitbucket repository with an open pull request, **When** `wave run pr-review -- "<PR-URL>"` is executed, **Then** the review is published as a comment using the Bitbucket REST API
3. **Given** a Gitea repository with an open pull request, **When** `wave run pr-review -- "<PR-URL>"` is executed, **Then** the review is published as a comment using `tea`

---

### User Story 6 - Backward-Compatible Pipeline Names (Priority: P2)

As an existing Wave user with scripts referencing `gh-implement` or `gl-research`, I want those names to continue working during a transition period, so that my automation does not break overnight.

**Why this priority**: Breaking existing user workflows would hinder adoption. A smooth migration path is essential.

**Independent Test**: Can be tested by running `wave run gh-implement` and verifying it either routes to the unified `implement` pipeline or produces a clear deprecation notice with migration instructions.

**Acceptance Scenarios**:

1. **Given** a user runs `wave run gh-implement -- "<issue>"`, **When** the old prefixed name is used, **Then** the system routes to the unified `implement` pipeline with a deprecation warning logged to stderr
2. **Given** a user lists available pipelines with `wave list pipelines`, **When** on a GitHub repository, **Then** the unified pipeline names appear (e.g., `implement`, `research`, `refresh`) without forge prefixes

---

### User Story 7 - Conditional Tool Requirements (Priority: P3)

As a pipeline executor, I need the `requires.tools` section of unified pipelines to dynamically resolve based on the detected forge, so that preflight checks validate the correct CLI tool (e.g., `gh` for GitHub, `glab` for GitLab).

**Why this priority**: Preflight validation ensures users get early feedback if a required CLI tool is missing. Important but not blocking for core unification.

**Independent Test**: Can be tested by removing `gh` from PATH on a GitHub repo and running `wave run implement`, verifying that the preflight check reports `gh` as missing.

**Acceptance Scenarios**:

1. **Given** a GitHub repository where `gh` is not installed, **When** `wave run implement` is executed, **Then** the preflight check fails with a message indicating `gh` is required
2. **Given** a Bitbucket repository, **When** `wave run implement` is executed, **Then** the preflight check validates that `curl` and `jq` are available (not `gh` or `glab`)

---

### Edge Cases

- What happens when forge detection returns `ForgeUnknown`? The system MUST provide a clear error with instructions to configure the forge manually in `wave.yaml`.
- What happens when a repository has multiple remotes pointing to different forges? The system MUST use the first `(fetch)` remote (current behavior of `DetectFromGitRemotes`).
- What happens when a unified prompt references a forge template variable but the forge context is not available (e.g., running in a bare directory)? Unresolved placeholders MUST be left as-is (consistent with existing `ResolvePlaceholders` behavior) and the step MUST fail with a descriptive error if the unresolved variable causes a command to fail.
- What happens when a user has customized a platform-specific pipeline in their local `.wave/` directory? Local customizations MUST continue to take precedence over embedded defaults (existing override behavior).
- What happens when the `bb-*` pipelines use `$BB_TOKEN` in prompts but the environment variable is not set? The system MUST NOT expose token values in prompt templates. Token handling MUST be delegated to the forge-specific persona's instructions, not embedded as literal values in pipeline-level prompt text.
- What happens when a new forge platform is added in the future? Adding a new forge MUST only require: (1) adding forge detection logic, (2) creating forge-specific personas, (3) extending the `forgeMetadata` function. No pipeline or prompt duplication should be needed.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST expose forge metadata as template variables in `PipelineContext`: `forge.type`, `forge.host`, `forge.owner`, `forge.repo`, `forge.cli_tool`, `forge.prefix`, `forge.pr_term` (Pull Request/Merge Request), `forge.pr_command` (pr/mr)
- **FR-002**: System MUST resolve `{{ forge.* }}` template variables in pipeline YAML fields (persona, prompt path, step configuration) and prompt file content using the existing `ResolvePlaceholders` mechanism
- **FR-003**: System MUST provide 7 unified pipeline definitions (`implement`, `implement-epic`, `scope`, `research`, `rewrite`, `refresh`, `pr-review`) that replace all 25 platform-specific pipeline files
- **FR-004**: System MUST resolve forge-specific personas at runtime using template variables in the `persona` field of pipeline step definitions (e.g., `persona: "{{ forge.prefix }}-commenter"`)
- **FR-005**: System MUST consolidate platform-variant prompt files into single unified prompt files per pipeline family that use `{{ forge.* }}` template variables for platform-specific commands
- **FR-006**: System MUST extend `pr-review` pipeline to support all four forge platforms (currently GitHub-only)
- **FR-007**: System MUST resolve `requires.tools` dynamically based on detected forge, or use a mechanism that auto-resolves to the correct CLI tool for the detected platform
- **FR-008**: System MUST update `FilterPipelinesByForge` to work with unified pipeline names (no forge prefix) while maintaining backward compatibility during transition
- **FR-009**: System MUST fix all 10 known duplication bugs documented in issue #241 comments as part of unification
- **FR-010**: System MUST update the embedded asset loading (`internal/defaults/embed.go`) to serve unified pipeline files and prompt directories instead of platform-specific variants
- **FR-011**: System MUST provide a clear error message when forge detection fails and no manual forge configuration exists in `wave.yaml`
- **FR-012**: System MUST preserve the ability for users to override embedded pipelines with local `.wave/` customizations

### Key Entities

- **ForgeInfo**: Existing entity (`internal/forge/detect.go`) describing the detected forge platform — extended with derived fields (`PRTerm`, `PRCommand`) for template variable support
- **PipelineContext**: Existing entity (`internal/pipeline/context.go`) holding runtime template variables — extended with `forge.*` namespace variables populated from `ForgeInfo`
- **Unified Pipeline**: A single pipeline YAML definition that uses `{{ forge.* }}` template variables to adapt behavior to the detected forge platform at runtime
- **Forge-Specific Persona**: Existing persona definitions (e.g., `github-commenter`, `bitbucket-analyst`) that remain as separate entities but are resolved dynamically via template variable expansion in the pipeline `persona` field

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: The number of pipeline YAML files in `internal/defaults/pipelines/` with forge prefixes (`bb-`, `gh-`, `gl-`, `gt-`) is reduced from 25 to 0, replaced by 7 unified files
- **SC-002**: The number of prompt directories in `internal/defaults/prompts/` with forge prefixes is reduced from 8+ to 0, replaced by unified directories per pipeline family
- **SC-003**: All 7 unified pipelines produce correct behavior for each of the 4 forge platforms (28 pipeline×platform combinations), validated by tests
- **SC-004**: Running `wave run implement` on a GitHub repository produces identical functional behavior to the current `gh-implement` pipeline
- **SC-005**: Running `wave run pr-review` succeeds on GitLab, Gitea, and Bitbucket repositories (new capability — currently GitHub-only)
- **SC-006**: All 10 documented duplication bugs from issue #241 are resolved and do not recur
- **SC-007**: Adding support for a new forge platform requires only forge detection logic and forge-specific personas — no pipeline or prompt file duplication
- **SC-008**: The total line count of pipeline YAML + prompt markdown files is reduced by at least 60% compared to the current duplicated set
