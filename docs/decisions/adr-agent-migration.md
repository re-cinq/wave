# ADR: Persona-to-Agent Migration Path

## Status

Accepted (PoC implemented)

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

Provide a **gradual migration path** rather than a hard cutover:

1. **`PersonaToAgentMarkdown()`** compiler in `internal/adapter/claude.go`
   converts `manifest.Persona` to a Claude Code agent `.md` file
2. **`UseAgentFlag`** field on `AdapterRunConfig` opt-in enables the new path
3. **`wave agent list/inspect/export`** CLI commands let users preview and
   export agent files
4. **No forced migration** — the existing CLAUDE.md + settings.json path
   remains the default

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
- Agent file is self-contained — no separate settings.json needed
- Migration is opt-in per pipeline or per run via `UseAgentFlag`
- Future Claude Code features (agent orchestration, tool permissions) can be
  adopted incrementally

## Implementation

- `PersonaToAgentMarkdown()` in `internal/adapter/claude.go`
- `UseAgentFlag` on `AdapterRunConfig` in `internal/adapter/adapter.go`
- `wave agent` CLI in `cmd/wave/commands/agent.go`
- Test coverage in `cmd/wave/commands/agent_test.go`
