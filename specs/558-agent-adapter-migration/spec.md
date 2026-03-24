# Feature Specification: Migrate Adapter to Agent-Based Execution

**Feature Branch**: `558-agent-adapter-migration`
**Created**: 2026-03-23
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/558

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Pipeline Steps Use Agent Files Instead of settings.json (Priority: P1)

As a Wave pipeline operator, I want each pipeline step to execute via a self-contained agent `.md` file so that tool permissions, model selection, and system prompts are unified in a single artifact instead of scattered across `settings.json`, CLI flags, and CLAUDE.md.

**Why this priority**: This is the core value proposition — collapsing three permission enforcement surfaces into one eliminates the primary source of adapter fragility.

**Independent Test**: Run any existing pipeline (e.g., `impl-issue`) end-to-end and verify that no `settings.json` is written to the workspace, an agent `.md` file is generated instead, and the pipeline produces identical results.

**Acceptance Scenarios**:

1. **Given** a pipeline step with a persona that has `allowed_tools: [Read, Write, Edit, Bash]` and `deny: ["Bash(rm -rf /*)"]`, **When** the adapter prepares the workspace, **Then** it writes a `.claude/wave-agent.md` file with YAML frontmatter containing `tools: [Read, Write, Edit, Bash]` and `disallowedTools: ["Bash(rm -rf /*)"]` and does NOT write `.claude/settings.json`.
2. **Given** a pipeline step with sandbox enabled and allowed domains configured, **When** the adapter prepares the workspace, **Then** it writes a minimal `.claude/settings.json` containing ONLY sandbox configuration (no model, temperature, or permissions fields) alongside the agent file.
3. **Given** a pipeline step using the agent path, **When** the adapter builds CLI arguments, **Then** it passes `--agent .claude/wave-agent.md` instead of `--allowedTools`, `--disallowedTools`, and `--dangerously-skip-permissions` flags.

---

### User Story 2 - normalizeAllowedTools Removed (Priority: P1)

As a Wave maintainer, I want the `normalizeAllowedTools` heuristic removed so that tool permission lists are passed through directly without fragile subsumption logic.

**Why this priority**: `normalizeAllowedTools` is the single most problematic function in the adapter — its edge cases (bare Write subsuming scoped Write, Bash removal taking Edit/Write) are the root cause of permission enforcement bugs.

**Independent Test**: Run the full test suite after removing `normalizeAllowedTools` and verify all tests pass. The agent frontmatter `tools:` list should contain exactly the tools specified in the persona manifest without deduplication or subsumption.

**Acceptance Scenarios**:

1. **Given** a persona with `allowed_tools: ["Read", "Write(.wave/output/*)", "Write", "Edit"]`, **When** the agent file is generated, **Then** the `tools:` frontmatter contains exactly `["Read", "Write(.wave/output/*)", "Write", "Edit"]` — no subsumption applied.
2. **Given** the agent code path is active, **When** `PersonaToAgentMarkdown()` is called, **Then** it does NOT call `normalizeAllowedTools`.
3. **Given** the codebase after migration, **When** searching for `normalizeAllowedTools` references, **Then** the function and its tests are deleted entirely.

---

### User Story 3 - settings.json Generation Eliminated (Priority: P2)

As a Wave developer, I want the adapter to stop generating `settings.json` for tool permissions so that there is a single source of truth for persona configuration.

**Why this priority**: Removing `settings.json` generation eliminates the redundancy between settings.json permissions and agent frontmatter tools, reducing the adapter's surface area.

**Independent Test**: Run the test suite and verify that `ClaudeSettings`, `ClaudePermissions`, and related JSON marshalling types are removed or simplified. Confirm no `.claude/settings.json` is created during pipeline execution (except the sandbox-only case).

**Acceptance Scenarios**:

1. **Given** the adapter after migration, **When** any pipeline step executes without sandbox enabled, **Then** no `.claude/settings.json` file exists in the workspace.
2. **Given** the adapter after migration, **When** a pipeline step executes with sandbox enabled, **Then** a minimal `.claude/settings.json` file is written containing ONLY the `sandbox` field — no `model`, `temperature`, `permissions`, or `output_format` fields.
3. **Given** the adapter after migration, **When** searching for `ClaudeSettings` struct usage, **Then** the struct is reduced to contain only `Sandbox *SandboxSettings` or is replaced by a dedicated `SandboxConfig` type.

---

### User Story 4 - CLI Flag Simplification (Priority: P2)

As a Wave maintainer, I want the adapter to use `--agent` instead of `--allowedTools`/`--disallowedTools`/`--dangerously-skip-permissions` flags so that CLI argument assembly is simpler and less version-dependent.

**Why this priority**: Reduces coupling to Claude Code CLI flag behavior and eliminates the CSV string formatting required by `--allowedTools`.

**Independent Test**: Inspect the `buildArgs` method output and verify it contains `--agent .claude/wave-agent.md` and does NOT contain `--allowedTools`, `--disallowedTools`, or `--dangerously-skip-permissions`.

**Acceptance Scenarios**:

1. **Given** a pipeline step, **When** `buildArgs` constructs the CLI command, **Then** the arguments include `["--agent", ".claude/wave-agent.md"]` and exclude `--allowedTools`, `--disallowedTools`, and `--dangerously-skip-permissions`.
2. **Given** a pipeline step with `TodoWrite` not in the persona's deny list, **When** the agent file is generated, **Then** `TodoWrite` is injected into the `disallowedTools:` frontmatter automatically.
3. **Given** a persona with `deny: ["TodoWrite", "Bash(sudo *)"]`, **When** the agent file is generated, **Then** `TodoWrite` appears only once in `disallowedTools:` (no duplication).

---

### User Story 5 - UseAgentFlag Removed (Priority: P3)

As a Wave developer, I want `UseAgentFlag` removed from `AdapterRunConfig` so that there is no bifurcation of adapter logic to maintain.

**Why this priority**: While the PoC introduced `UseAgentFlag` as opt-in, maintaining two code paths doubles test surface and creates drift risk. Making agent mode the only path enables deletion of legacy code.

**Independent Test**: Search the codebase for `UseAgentFlag` references and verify the field is removed from `AdapterRunConfig`. All executor code that previously set this flag should be removed.

**Acceptance Scenarios**:

1. **Given** the adapter after migration, **When** searching for `UseAgentFlag` in the codebase, **Then** zero references exist — the field is removed and agent mode is unconditional.
2. **Given** the pipeline executor, **When** it constructs `AdapterRunConfig`, **Then** it does NOT set any flag to opt into agent mode — agent mode is the default behavior.
3. **Given** the `wave agent export` CLI command, **When** it exports a persona, **Then** it produces output identical to what the adapter generates at runtime (same function, `PersonaToAgentMarkdown`).

---

### Edge Cases

- **Empty tool lists**: A persona with no `allowed_tools` and no `deny` — the agent frontmatter should omit `tools:` and `disallowedTools:` fields entirely, letting Claude Code use its defaults.
- **Sandbox without permissions**: When sandbox is enabled but no `allowed_domains` are configured, the minimal settings.json should contain `sandbox.enabled: true` without a `network` field.
- **Agent file path with spaces**: If the workspace path contains spaces, the `--agent` flag value must be properly quoted/escaped in the CLI arguments.
- **Concurrent pipeline steps**: Multiple steps running in parallel in different worktrees must each get their own agent file — no shared state between `.claude/wave-agent.md` files across workspaces.
- **Non-Claude adapters**: The `BrowserAdapter`, `OpenCodeAdapter`, and `GitHubAdapter` are unaffected — this migration applies only to `ClaudeAdapter`.
- **Temperature field**: The persona `temperature` field has no equivalent in agent frontmatter — it is intentionally dropped. Document this as an accepted behavior change.
- **Existing `wave agent` CLI**: The `wave agent list/inspect/export` commands must continue to work and produce agent files matching the new runtime format.

## Clarifications

### C-001: `--model` CLI flag redundancy with agent frontmatter

**Ambiguity**: The agent frontmatter includes `model:` and `buildArgs` currently always passes `--model <model>`. After migration, should `--model` remain as a CLI flag alongside the agent frontmatter `model:` field?

**Resolution**: Remove `--model` from `buildArgs`. The agent frontmatter `model:` field is the single source of truth for model selection. Passing both creates ambiguity about which takes precedence and defeats the purpose of self-contained agent files. The `buildArgs` method should only contain `--agent`, `--output-format`, `--verbose`, and `--no-session-persistence`.

**Rationale**: Claude Code's `--agent` flag is designed to be self-contained. Duplicating model in both frontmatter and CLI flags reintroduces the multi-surface problem this migration eliminates.

### C-002: Retained CLI flags (`--output-format`, `--verbose`, `--no-session-persistence`)

**Ambiguity**: The spec says `buildArgs` should use `--agent` instead of permission-related flags, but does not mention whether `--output-format stream-json`, `--verbose`, and `--no-session-persistence` should be retained.

**Resolution**: These three flags MUST be retained in `buildArgs`. They control runtime transport behavior (NDJSON streaming, verbosity for token tracking, session lifecycle) that has no equivalent in agent frontmatter. The agent frontmatter covers model, tools, and permissions only.

**Rationale**: Agent frontmatter schema (per research spec #557) does not include output format or session persistence fields. These are orthogonal operational concerns, not persona configuration.

### C-003: `chatworkspace.go` settings.json is out of scope

**Ambiguity**: `internal/pipeline/chatworkspace.go` also writes `.claude/settings.json` with full permissions for interactive `wave chat` sessions. Should this also be migrated to agent files?

**Resolution**: `chatworkspace.go` is OUT OF SCOPE for this migration. It serves a different use case (interactive debugging sessions launched via `wave chat`) and does not use `ClaudeAdapter.prepareWorkspace()`. The `chatSettings` and `chatPermissions` types in that file are distinct from `ClaudeSettings`/`ClaudePermissions` and should not be touched. A separate issue should be filed if chat workspace migration is desired.

**Rationale**: The spec's edge case "Non-Claude adapters" principle applies — this migration targets `ClaudeAdapter` pipeline execution only. Chat workspace uses `LaunchInteractive` which has its own argument assembly. Mixing scopes increases risk.

### C-004: TodoWrite injection site

**Ambiguity**: FR-006 requires automatic `TodoWrite` injection into `disallowedTools`, but doesn't specify whether this happens in `prepareWorkspace` (before calling `PersonaToAgentMarkdown`) or inside `PersonaToAgentMarkdown` itself.

**Resolution**: TodoWrite injection MUST happen in `prepareWorkspace`, before constructing the `PersonaSpec` passed to `PersonaToAgentMarkdown`. Specifically, the adapter should append `"TodoWrite"` to `cfg.DenyTools` (if not already present) before building the `PersonaSpec`. `PersonaToAgentMarkdown` remains a pure compiler with no business logic — it writes whatever is in the spec.

**Rationale**: Keeping `PersonaToAgentMarkdown` as a pure mapping function preserves testability and ensures `wave agent export` produces output that exactly matches runtime behavior (since the CLI export path can also inject TodoWrite before calling the same function). Business rules belong in the orchestration layer, not the serialization layer.

### C-005: `interactive.go` `--dangerously-skip-permissions` is out of scope

**Ambiguity**: `internal/adapter/interactive.go:95` also passes `--dangerously-skip-permissions` in `buildInteractiveArgs`. Should this be migrated to use `--agent` as well?

**Resolution**: `interactive.go` is OUT OF SCOPE. It serves interactive `wave chat` sessions where the user directly controls Claude Code via stdin/stdout. Interactive mode uses `--allowedTools` and `--dangerously-skip-permissions` by design because there is no agent file involved — the session is user-driven, not pipeline-driven. A future migration may address this, but it requires different UX considerations.

**Rationale**: Interactive sessions fundamentally differ from pipeline steps — they don't have personas, contracts, or workspace isolation. The `--dangerously-skip-permissions` flag in interactive mode enables the user to interact without being prompted, which is the expected behavior for `wave chat`.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The adapter MUST generate a `.claude/wave-agent.md` file for every pipeline step using the `ClaudeAdapter`, containing YAML frontmatter with `model`, `tools`, `disallowedTools`, and `permissionMode` fields.
- **FR-002**: The adapter MUST NOT write `.claude/settings.json` when sandbox is disabled.
- **FR-003**: The adapter MUST write a minimal `.claude/settings.json` containing ONLY sandbox configuration when sandbox is enabled.
- **FR-004**: The adapter MUST pass `--agent .claude/wave-agent.md` in CLI arguments instead of `--allowedTools`, `--disallowedTools`, `--dangerously-skip-permissions`, and `--model`. The retained CLI flags are: `--output-format stream-json`, `--verbose`, and `--no-session-persistence` (see C-001, C-002).
- **FR-005**: The adapter MUST NOT call `normalizeAllowedTools` — tool lists from persona manifests are passed through to agent frontmatter without modification.
- **FR-006**: The adapter MUST automatically inject `TodoWrite` into `disallowedTools` if not already present in the persona's deny list. Injection happens in `prepareWorkspace` before constructing `PersonaSpec`, not inside `PersonaToAgentMarkdown` (see C-004).
- **FR-007**: The `PersonaToAgentMarkdown` function MUST pass tools through directly without normalization.
- **FR-008**: The `UseAgentFlag` field MUST be removed from `AdapterRunConfig` — agent mode is unconditional.
- **FR-009**: The `ClaudeSettings` and `ClaudePermissions` types MUST be simplified or removed — only sandbox-related types retained.
- **FR-010**: The `wave agent export` CLI command MUST produce output identical to the runtime agent file format.
- **FR-011**: The agent file body MUST preserve the four-layer CLAUDE.md assembly: base protocol, persona prompt, contract compliance, and restrictions.
- **FR-012**: All existing pipeline execution tests MUST pass after the migration.

### Key Entities

- **Agent File** (`.claude/wave-agent.md`): A self-contained Claude Code agent definition with YAML frontmatter (model, tools, disallowedTools, permissionMode) and markdown body (base protocol + persona + contract + restrictions). Replaces the settings.json + CLAUDE.md + CLI flags triple.
- **Sandbox Config** (`.claude/settings.json`, optional): A minimal JSON file containing only sandbox settings (enabled, network domains). Written only when sandbox is enabled. Replaces the full `ClaudeSettings` struct.
- **PersonaSpec**: Intermediate representation passed to `PersonaToAgentMarkdown()`. Contains model, allowed tools, deny tools extracted from the manifest persona definition.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Zero `settings.json` files written per pipeline run when sandbox is disabled — verified by checking workspace contents after execution.
- **SC-002**: `normalizeAllowedTools` function and its 8 test cases are deleted from the codebase — verified by grep.
- **SC-003**: All existing tests in `internal/adapter/` pass with the new code path — verified by `go test ./internal/adapter/...`.
- **SC-004**: All existing tests across the project pass — verified by `go test ./...`.
- **SC-005**: The `buildArgs` method produces CLI arguments containing `--agent` and NOT containing `--allowedTools`, `--disallowedTools`, `--dangerously-skip-permissions`, or `--model` — verified by unit test. It MUST still contain `--output-format`, `--verbose`, and `--no-session-persistence`.
- **SC-006**: Net reduction in adapter code — the migration should remove more lines than it adds, measured by `git diff --stat`.
- **SC-007**: `UseAgentFlag` field has zero references in the codebase after migration — verified by grep.
- **SC-008**: Pipeline execution reliability is equal or better compared to pre-migration baseline — verified by running a representative pipeline end-to-end.
