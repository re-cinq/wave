# Research: Adapter-to-Agent Migration

## Decision 1: Agent File Generation Strategy

**Decision**: Use the existing `PersonaToAgentMarkdown()` function as the sole code path for both runtime and CLI export. Remove the CLAUDE.md assembly branch entirely.

**Rationale**: The PoC already proved the agent .md format works end-to-end. The only change needed is making agent mode unconditional rather than opt-in via `UseAgentFlag`.

**Alternatives Rejected**:
- *Dual code path with feature flag*: Maintaining both CLAUDE.md and agent.md increases test surface and creates drift risk. The PoC has been validated; no reason to keep the legacy path.
- *New function for runtime vs export*: Would duplicate the compilation logic. The spec explicitly requires `wave agent export` to produce identical output to runtime (FR-010).

## Decision 2: normalizeAllowedTools Removal

**Decision**: Delete `normalizeAllowedTools` entirely and pass tool lists through verbatim to agent frontmatter `tools:` field.

**Rationale**: The agent frontmatter's `tools:` field accepts tool names directly. Claude Code handles its own internal deduplication. The subsumption logic (bare Write subsuming scoped Write) is the root cause of multiple permission enforcement bugs — removing it is the primary value of this migration.

**Alternatives Rejected**:
- *Keep for backward compat*: Constitution explicitly says no backward compat constraint during prototype phase.
- *Move normalization to Claude Code CLI*: Out of scope — Wave should not depend on undocumented CLI behavior.

## Decision 3: settings.json Handling

**Decision**: Stop writing `settings.json` entirely when sandbox is disabled. When sandbox IS enabled, write a minimal `settings.json` containing ONLY the `sandbox` field.

**Rationale**: The agent frontmatter covers model, tools, permissions, and permissionMode. The only field without a frontmatter equivalent is `sandbox` configuration (bubblewrap/docker settings). The `temperature` field is intentionally dropped since agent frontmatter has no temperature equivalent, and the Claude CLI doesn't support temperature via settings.json in a meaningful way for pipeline execution.

**Alternatives Rejected**:
- *Always write settings.json*: Creates conflicting sources of truth — the exact problem this migration solves.
- *Embed sandbox config in agent frontmatter*: Not supported by Claude Code's `--agent` schema (confirmed in research spec #557).

## Decision 4: CLI Flag Simplification

**Decision**: `buildArgs` produces `--agent .claude/wave-agent.md` and retains only `--output-format stream-json`, `--verbose`, and `--no-session-persistence`. Remove `--model`, `--allowedTools`, `--disallowedTools`, and `--dangerously-skip-permissions`.

**Rationale**:
- `--model` is redundant with agent frontmatter `model:` field (C-001).
- `--allowedTools`/`--disallowedTools` are replaced by agent frontmatter `tools:`/`disallowedTools:`.
- `--dangerously-skip-permissions` is replaced by agent frontmatter `permissionMode: dontAsk`.
- `--output-format`, `--verbose`, `--no-session-persistence` control runtime transport behavior with no agent frontmatter equivalent (C-002).

**Alternatives Rejected**:
- *Keep --model alongside frontmatter*: Creates ambiguity about precedence; defeats self-contained agent goal.
- *Move --verbose into agent file*: Not supported by agent frontmatter schema.

## Decision 5: TodoWrite Injection Site

**Decision**: Inject `TodoWrite` into `cfg.DenyTools` in `prepareWorkspace` before constructing the `PersonaSpec`, not inside `PersonaToAgentMarkdown()` (C-004).

**Rationale**: `PersonaToAgentMarkdown` is a pure compiler — it writes whatever is in the spec. Business rules (like auto-denying TodoWrite) belong in the orchestration layer. This keeps `wave agent export` and runtime in sync by allowing both paths to inject TodoWrite before calling the same function.

## Decision 6: Type Simplification

**Decision**: Replace `ClaudeSettings` and `ClaudePermissions` with a minimal `SandboxOnlySettings` struct containing only the sandbox field. Remove `Model`, `Temperature`, `OutputFormat`, and `Permissions` fields.

**Rationale**: After migration, `settings.json` is only written for sandbox configuration. The full `ClaudeSettings` struct with model/permissions is dead code.

## Decision 7: Scope Boundaries

**Decision**: `interactive.go` (`buildInteractiveArgs` with `--dangerously-skip-permissions`) and `chatworkspace.go` are OUT OF SCOPE (C-003, C-005).

**Rationale**: Interactive sessions serve a fundamentally different use case (user-driven, no agent file, no workspace isolation). Mixing scopes increases risk. Separate issues can address these migrations.

## Decision 8: Non-Claude Adapters

**Decision**: `BrowserAdapter`, `OpenCodeAdapter`, and `GitHubAdapter` are unaffected.

**Rationale**: This migration applies only to `ClaudeAdapter`. Other adapters don't use `settings.json` or CLAUDE.md assembly.
