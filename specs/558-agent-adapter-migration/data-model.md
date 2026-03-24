# Data Model: Adapter-to-Agent Migration

## Entities

### Agent File (`.claude/wave-agent.md`)

A self-contained Claude Code agent definition written to each pipeline step's workspace.

**Structure**:
```
---                              # YAML frontmatter start
model: <string>                  # Claude model identifier (e.g., "opus", "sonnet")
tools:                           # List of allowed tool names (passthrough, no normalization)
  - <string>
disallowedTools:                 # List of denied tool patterns
  - <string>
permissionMode: dontAsk          # Always "dontAsk" — Wave handles permissions
---                              # YAML frontmatter end
<base protocol preamble>         # .wave/personas/base-protocol.md content

---

<persona system prompt>          # From .wave/personas/<name>.md or cfg.SystemPrompt

---

<contract compliance section>    # Auto-generated from step contract config

<restriction section>            # Denied/allowed tools and network domains
```

**Lifecycle**: Created by `prepareWorkspace()` before each step execution. Lives at `.claude/wave-agent.md` relative to workspace root. Deleted when workspace is cleaned.

**Invariants**:
- YAML frontmatter always present, even if empty (minimal: `permissionMode: dontAsk`)
- `tools:` and `disallowedTools:` omitted when respective lists are empty (not empty arrays)
- `model:` omitted when persona has no model override (inherits CLI default)
- Body preserves four-layer assembly order: base protocol → persona → contract → restrictions

### Sandbox Config (`.claude/settings.json`, conditional)

Minimal JSON file written ONLY when `cfg.SandboxEnabled == true`.

**Structure**:
```json
{
  "sandbox": {
    "enabled": true,
    "allowUnsandboxedCommands": false,
    "autoAllowBashIfSandboxed": true,
    "network": {                          // Omitted when no domains configured
      "allowedDomains": ["domain1.com"]
    }
  }
}
```

**Lifecycle**: Written alongside agent file in `prepareWorkspace()` only when sandbox is enabled. When sandbox is disabled, `.claude/settings.json` is NOT created.

**Invariants**:
- Contains ONLY the `sandbox` field — no `model`, `temperature`, `output_format`, or `permissions`
- `network` sub-object omitted when `AllowedDomains` is empty

### PersonaSpec (in-memory intermediate)

Unchanged struct used as input to `PersonaToAgentMarkdown()`.

```go
type PersonaSpec struct {
    Model        string
    AllowedTools []string
    DenyTools    []string
}
```

**Changes**: None to the struct. The change is that `PersonaToAgentMarkdown` no longer calls `normalizeAllowedTools` — it passes `AllowedTools` through directly.

### SandboxOnlySettings (new type, replaces ClaudeSettings)

```go
type SandboxOnlySettings struct {
    Sandbox *SandboxSettings `json:"sandbox,omitempty"`
}
```

**Replaces**: `ClaudeSettings` (with `Model`, `Temperature`, `OutputFormat`, `Permissions` fields removed).

### AdapterRunConfig (modified)

**Removed fields**:
- `UseAgentFlag bool` — agent mode is now unconditional

**Unchanged fields**: All other fields remain. `AllowedTools`, `DenyTools`, `SandboxEnabled`, `AllowedDomains`, etc. are still populated by the executor.

## Deleted Types

| Type | Reason |
|------|--------|
| `ClaudeSettings` | Replaced by `SandboxOnlySettings` |
| `ClaudePermissions` | No longer needed — permissions are in agent frontmatter |

## Deleted Functions

| Function | Reason |
|----------|--------|
| `normalizeAllowedTools()` | Tool lists passed through without normalization (FR-005) |

## Relationships

```
manifest.Persona
    ↓ (executor maps fields)
AdapterRunConfig
    ↓ (prepareWorkspace builds PersonaSpec)
PersonaSpec
    ↓ (PersonaToAgentMarkdown compiles)
.claude/wave-agent.md  (always)
.claude/settings.json  (only if sandbox enabled)
    ↓ (buildArgs references)
claude --agent .claude/wave-agent.md --output-format stream-json --verbose --no-session-persistence
```
