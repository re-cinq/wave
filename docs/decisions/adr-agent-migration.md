# ADR: Persona-to-Agent Migration Path

## Status

Complete

## Context

Claude Code introduced the `--agent` flag which accepts a `.md` file with YAML
frontmatter specifying model, tools, disallowed tools, and permission mode. This
maps nearly 1:1 to Wave's persona definitions, which currently compile into a
runtime `CLAUDE.md` + `settings.json` pair.

The research in issue #395 (3 rounds) established:

1. **Claude Code `--agent` frontmatter** supports `model`, `tools`,
   `disallowedTools`, and `permissionMode` — covering Wave's persona fields
2. **deny rules are broken** in Claude Code (deny `Bash(*)` removes Write+Edit
   too) — security is enforced via bubblewrap sandbox instead (#282)
3. **`--agent` simplifies adapter code** by collapsing CLAUDE.md assembly and
   settings.json generation into a single `.md` file

## Decision

Use agent `.md` files as the **only** code path for Claude Code adapter execution:

1. **`PersonaToAgentMarkdown()`** compiler in `internal/adapter/claude.go`
   converts `manifest.Persona` to a Claude Code agent `.md` file
2. Agent mode is **unconditional** — no opt-in flag
3. **`wave agent list/inspect/export`** CLI commands let users preview and
   export agent files
4. The legacy CLAUDE.md + settings.json code path has been removed

## Agent .md Format

```markdown
---
model: sonnet
tools:
  - Read
  - Glob
  - Grep
  - Agent
disallowedTools: []
permissionMode: dontAsk
---
<base protocol preamble>

---

<persona system prompt>

---

<contract compliance section>

<restriction section>
```

## Consequences

- Personas can be used outside Wave via `claude --agent <exported>.md`
- Agent file is self-contained — model, tools, and permissions in frontmatter
- `settings.json` is only written for sandbox configuration (when enabled)
- `normalizeAllowedTools` removed — tool lists pass through verbatim
- Future Claude Code features (agent orchestration, tool permissions) can be
  adopted incrementally

## Implementation

- `PersonaToAgentMarkdown()` in `internal/adapter/claude.go`
- Agent mode is unconditional — `UseAgentFlag` removed from `AdapterRunConfig`
- `wave agent` CLI in `cmd/wave/commands/agent.go`
- Test coverage in `cmd/wave/commands/agent_test.go`
