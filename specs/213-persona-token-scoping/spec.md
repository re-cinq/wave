# Feature Specification: Persona Token Scoping

**Feature Branch**: `213-persona-token-scoping`
**Created**: 2026-03-16
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/213

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Declare Token Scopes Per Persona (Priority: P1)

As a Wave operator, I want to declare the minimum required API token scopes for each persona in `wave.yaml`, so that each persona only receives the credentials it needs and the principle of least privilege is enforced.

**Why this priority**: This is the foundational capability. Without scope declarations in the manifest, no validation or enforcement can occur. It directly addresses the core security gap where all personas share the same unrestricted token access.

**Independent Test**: Can be fully tested by editing `wave.yaml` to add `token_scopes` to a persona and verifying the manifest loads and validates correctly without runtime execution.

**Acceptance Scenarios**:

1. **Given** a `wave.yaml` with a persona declaring `token_scopes: [issues:read, pulls:read]`, **When** the manifest is loaded, **Then** the persona's token scope requirements are parsed and accessible via the Persona struct.
2. **Given** a `wave.yaml` with a persona declaring invalid scope syntax (e.g., empty string, malformed scope), **When** the manifest is loaded, **Then** validation fails with a clear error message identifying the invalid scope.
3. **Given** a `wave.yaml` with a persona that does NOT declare `token_scopes`, **When** the manifest is loaded, **Then** the persona is treated as having no scope requirements (backward compatible — no enforcement).

---

### User Story 2 - Preflight Token Scope Validation (Priority: P1)

As a Wave operator, I want `wave run` to validate that the active API token has sufficient scopes for every persona in the pipeline before execution begins, so that pipelines fail fast with a clear error rather than failing mid-execution due to permission denied responses.

**Why this priority**: Without preflight validation, declared scopes are documentation only. This story closes the enforcement loop and is the primary security boundary replacing deny lists.

**Independent Test**: Can be tested by running a pipeline with a deliberately under-scoped token and verifying the preflight check catches the mismatch before any step executes.

**Acceptance Scenarios**:

1. **Given** a pipeline where persona `github-analyst` requires `[issues:read, pulls:read]` and the active forge token has those scopes, **When** `wave run` starts, **Then** preflight validation passes and execution proceeds.
2. **Given** a pipeline where persona `github-commenter` requires `[issues:write]` but the active token only has `[issues:read]`, **When** `wave run` starts, **Then** preflight validation fails with an error message listing the missing scopes, the persona name, and guidance on how to create an appropriately scoped token.
3. **Given** a pipeline with multiple personas requiring different scopes, **When** `wave run` starts, **Then** all persona scope requirements are validated against the token and all violations are reported together (not one at a time).
4. **Given** a persona with no `token_scopes` declared, **When** preflight runs, **Then** no token validation is performed for that persona (opt-in enforcement).

---

### User Story 3 - Platform-Aware Scope Resolution (Priority: P2)

As a Wave operator running pipelines against GitHub, GitLab, or Gitea repositories, I want token scope declarations to be resolved against the detected forge platform, so that scope validation works correctly regardless of which hosting platform my repository uses.

**Why this priority**: Wave already supports multi-forge detection. Token scopes differ across platforms (GitHub uses `repo`, `issues`, `pull_requests`; GitLab uses `api`, `read_repository`; Gitea uses `read:issue`, `write:issue`). Without platform awareness, scope declarations would be GitHub-only.

**Independent Test**: Can be tested by configuring a persona with platform-specific scopes and verifying resolution against each detected forge type.

**Acceptance Scenarios**:

1. **Given** a persona with `token_scopes: [issues:read]` and a GitHub forge, **When** scope resolution runs, **Then** the scope maps to the GitHub-equivalent permission check (querying the token's actual scopes via API or CLI).
2. **Given** a persona with `token_scopes: [issues:read]` and a GitLab forge, **When** scope resolution runs, **Then** the scope maps to the GitLab-equivalent permission check.
3. **Given** an unrecognized forge type, **When** scope resolution runs, **Then** a warning is emitted but execution is not blocked (graceful degradation).

---

### User Story 4 - Scope Recommendations in `wave init` (Priority: P3)

As a new Wave user running `wave init`, I want the generated `wave.yaml` to include comments showing the minimum token scopes recommended for each persona, so that I can create appropriately scoped tokens before running pipelines.

**Why this priority**: This is a usability improvement that helps users adopt token scoping. The core security enforcement (stories 1-2) works without this, but discoverability is important for adoption.

**Independent Test**: Can be tested by running `wave init` and inspecting the generated `wave.yaml` for scope recommendation comments.

**Acceptance Scenarios**:

1. **Given** a user runs `wave init` and selects personas, **When** the manifest is generated, **Then** each persona section includes a comment showing recommended `token_scopes` for that role.
2. **Given** a forge-aware context (e.g., GitHub detected), **When** recommendations are generated, **Then** the comments reference the platform-specific token creation flow (e.g., GitHub fine-grained PAT settings URL).

---

### Edge Cases

- What happens when the token introspection API itself requires elevated permissions? The system degrades gracefully — warns but does not block execution.
- What happens when a forge platform doesn't support scope introspection (e.g., self-hosted Gitea without API)? The system skips validation with a warning.
- What happens when `runtime.sandbox.env_passthrough` doesn't include the token variable (e.g., `GH_TOKEN` not listed)? Preflight detects this and reports it as a configuration error.
- What happens when multiple forge tokens are needed in one pipeline (e.g., cross-platform mirroring)? Each scope entry can optionally specify a token variable via `<resource>:<permission>@<ENV_VAR>` syntax (e.g., `issues:read@GH_TOKEN`). When no `@<ENV_VAR>` suffix is present, the scope applies to the primary forge token detected via `forge.DetectFromGitRemotes()` (typically `GH_TOKEN` or `GITHUB_TOKEN` for GitHub).
- What happens when the token expires between preflight and step execution? Out of scope — preflight validates at pipeline start only.
- What happens when deny lists conflict with token scopes (e.g., deny `Bash(gh issue edit*)` but token has `issues:write`)? Deny lists are preserved as defense-in-depth; both constraints apply independently.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The Persona manifest schema MUST support an optional `token_scopes` field that declares a list of required API token scopes.
- **FR-002**: Each scope entry MUST follow the format `<resource>:<permission>` (or `<resource>:<permission>@<ENV_VAR>` for multi-token scenarios) where resource is one of the canonical set (`issues`, `pulls`, `repos`, `actions`, `packages` — extensible with lint warnings for unknown resources) and permission is one of `read`, `write`, `admin` (hierarchical: `admin` ⊇ `write` ⊇ `read`).
- **FR-003**: The manifest validator MUST reject invalid scope syntax during manifest loading with a descriptive error message.
- **FR-004**: The preflight checker MUST validate that the active forge token's actual scopes satisfy all persona scope requirements before any pipeline step executes.
- **FR-005**: Preflight validation errors MUST include: the persona name, the missing scopes, the token variable checked, and a human-readable remediation hint.
- **FR-006**: Preflight validation MUST aggregate all scope violations across all pipeline personas and report them together.
- **FR-007**: When forge detection returns an unknown platform, scope validation MUST emit a warning and skip enforcement (no hard failure).
- **FR-008**: The `wave init` wizard MUST include recommended `token_scopes` as YAML comments in generated persona configurations.
- **FR-009**: Existing deny lists MUST be preserved and enforced independently of token scoping (defense-in-depth).
- **FR-010**: Token scope validation MUST be opt-in per persona — personas without `token_scopes` declared skip validation entirely.
- **FR-011**: The system MUST support platform-specific scope resolution, mapping abstract scopes to forge-native scope identifiers for validation.
- **FR-012**: Scope validation MUST support GitHub (fine-grained PATs via `gh api user` header inspection), GitLab (project access tokens via `glab api` or API introspection), and Gitea (application tokens via API introspection). Bitbucket is deferred to a follow-up issue — the `ForgeBitbucket` forge type is recognized but scope validation emits a warning and skips enforcement, consistent with FR-007's unknown-platform behavior.

### Key Entities

- **TokenScope**: A declared permission requirement in the format `<resource>:<permission>`. Represents the minimum access a persona needs from the forge API token.
- **ScopeResolver**: Translates abstract TokenScope declarations to platform-specific scope identifiers using the detected ForgeInfo.
- **TokenIntrospector**: Queries the actual scopes/permissions of the active forge token (e.g., via CLI tool or API call).
- **ScopeValidator**: Compares declared scopes against actual token scopes and produces validation results with remediation hints.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: A pipeline using a read-only token with a write-requiring persona fails at preflight (within 5 seconds of `wave run`) with a clear error naming the persona and missing scopes.
- **SC-002**: All existing pipelines and personas continue to work without modification (100% backward compatibility for manifests without `token_scopes`).
- **SC-003**: Token scope validation covers at least GitHub and one additional forge platform (GitLab or Gitea) at launch.
- **SC-004**: All existing tests pass; new tests cover scope parsing, validation logic, platform resolution, and preflight integration with at least 80% branch coverage in new code.
- **SC-005**: `wave init` output includes scope recommendations for all default personas that interact with forge APIs.
- **SC-006**: No security regression — deny lists remain enforced alongside token scoping, verified by existing permission tests continuing to pass.

## Clarifications _(resolved during spec refinement)_

### C1: Bitbucket Token Scope Support
**Question**: Should Bitbucket token scope validation be included in the initial implementation?
**Resolution**: Deferred. The forge package already defines `ForgeBitbucket` with CLI tool `bb`, but Bitbucket's workspace-level permission model differs significantly from GitHub/GitLab/Gitea resource-level scopes. For the initial implementation, Bitbucket is treated like an unknown forge — a warning is emitted and enforcement is skipped, consistent with FR-007. A follow-up issue should investigate Bitbucket workspace permissions and App passwords.
**Rationale**: SC-003 only requires "GitHub and one additional forge platform." Shipping GitHub + GitLab covers the majority of Wave users without blocking on Bitbucket research.

### C2: Token Variable Binding for Multi-Token Scenarios
**Question**: How does a persona specify which environment variable holds the token for a given scope?
**Resolution**: Scope entries support an optional `@<ENV_VAR>` suffix: `issues:read@GH_TOKEN`. When omitted, the scope binds to the default forge token variable (inferred from `forge.DetectFromGitRemotes()` — `GH_TOKEN`/`GITHUB_TOKEN` for GitHub, `GITLAB_TOKEN` for GitLab, etc.). This keeps the common case simple while supporting cross-platform pipelines.
**Rationale**: Wave already uses `runtime.sandbox.env_passthrough` to control which env vars reach subprocesses. The `@` syntax mirrors common notation for "at this source" and doesn't conflict with the `<resource>:<permission>` base format.

### C3: Token Introspection Mechanism
**Question**: How does the system discover a token's actual scopes at runtime?
**Resolution**: Platform-specific introspection via CLI tools already available in Wave's preflight:
- **GitHub**: `gh api user --include` returns `X-OAuth-Scopes` header (classic PATs) or use `gh api /` to check fine-grained PAT permissions from 403 responses. For fine-grained PATs, query `gh api user` and inspect response headers.
- **GitLab**: `glab api /personal_access_tokens/self` returns token scopes directly.
- **Gitea**: `tea` CLI or direct API call to `/api/v1/user` with auth header, checking response status for access level.
If introspection fails (token lacks introspection permissions, API unreachable), the system warns but does not block — consistent with Edge Case 1 and FR-007.
**Rationale**: Using CLI tools (`gh`, `glab`, `tea`) keeps the implementation consistent with Wave's existing forge detection pattern and avoids adding HTTP client dependencies.

### C4: Canonical Resource Vocabulary
**Question**: What is the closed set of abstract resource names in `<resource>:<permission>`?
**Resolution**: The initial vocabulary covers resources matching Wave's actual forge API usage:
- `issues` — issue read/write/admin
- `pulls` — pull/merge request read/write
- `repos` — repository content read/write/admin
- `actions` — CI/CD workflow read/write (GitHub Actions, GitLab CI)
- `packages` — package registry read/write
The vocabulary is extensible — unknown resources pass through as-is during scope resolution but generate a lint warning during manifest validation. Permissions are fixed to `read`, `write`, `admin` (hierarchical: `admin` implies `write` implies `read`).
**Rationale**: A small, fixed initial set avoids scope sprawl while the extensibility mechanism prevents the system from blocking on new forge capabilities.

### C5: Preflight Integration Point
**Question**: Should token scope validation extend the existing `preflight.Checker` or be a separate component?
**Resolution**: Separate component. The existing `preflight.Checker` validates tool/skill availability (binary existence, install commands). Token scope validation has different concerns (network API calls, forge-specific logic, per-persona scoping) and should be a new `internal/scope` package with its own `Validator` type. The pipeline executor calls both preflight checks sequentially: tools/skills first, then token scopes.
**Rationale**: Single responsibility — `preflight.Checker` checks local tool availability; `scope.Validator` checks remote token permissions. Mixing them would create a package with two unrelated dependencies (exec.LookPath vs. forge APIs).
