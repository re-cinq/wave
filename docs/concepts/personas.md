# Personas

Personas are the safety and specialization mechanism in Muzzle. Each persona defines a **role** — what an agent can see, do, and how it behaves. Personas enforce separation of concerns so that a read-only navigator cannot write files and a craftsman cannot install arbitrary dependencies.

## Why Personas?

Without persona scoping, every agent has identical permissions and context. This creates risk:

- A navigation step could accidentally modify files.
- An implementation step could bypass security checks.
- A review step could "fix" issues rather than report them.

Personas solve this by **binding capabilities to roles**.

## Persona Anatomy

```mermaid
graph TD
    P[Persona] --> A[Adapter Reference]
    P --> SP[System Prompt]
    P --> T[Temperature]
    P --> Pm[Permissions]
    P --> H[Hooks]
    Pm --> Allow[allowed_tools]
    Pm --> Deny[deny patterns]
    H --> Pre[PreToolUse]
    H --> Post[PostToolUse]
```

| Component | Purpose |
|-----------|---------|
| **Adapter** | Which LLM CLI to use (e.g., `claude`, `opencode`). |
| **System Prompt** | Markdown file defining the agent's role, goals, and constraints. |
| **Temperature** | Controls output determinism. Low (0.1) for analysis, high (0.7) for creative work. |
| **Permissions** | Tool access control — what the agent can and cannot do. |
| **Hooks** | Shell commands that execute before/after tool calls. |

## Built-in Archetypes

Muzzle's design encourages these persona patterns:

### Navigator

Read-only codebase exploration. Finds files, analyzes patterns, maps architecture. Never modifies anything.

```yaml
navigator:
  adapter: claude
  system_prompt_file: .muzzle/personas/navigator.md
  temperature: 0.1
  permissions:
    allowed_tools: ["Read", "Glob", "Grep", "Bash(git log*)", "Bash(git status*)"]
    deny: ["Write(*)", "Edit(*)", "Bash(git commit*)", "Bash(git push*)"]
```

### Philosopher

Design and specification. Creates specs, plans, and contracts. Limited to writing in specification directories.

```yaml
philosopher:
  adapter: claude
  system_prompt_file: .muzzle/personas/philosopher.md
  temperature: 0.3
  permissions:
    allowed_tools: ["Read", "Write(.muzzle/specs/*)"]
    deny: ["Bash(*)"]
```

### Craftsman

Full implementation capability. Reads, writes, edits, and runs commands. Protected by hooks for dangerous operations.

```yaml
craftsman:
  adapter: claude
  system_prompt_file: .muzzle/personas/craftsman.md
  temperature: 0.7
  permissions:
    allowed_tools: ["Read", "Write", "Edit", "Bash"]
    deny: ["Bash(rm -rf /*)"]
  hooks:
    PreToolUse:
      - matcher: "Bash(git commit*)"
        command: ".muzzle/hooks/pre-commit-lint.sh"
```

### Auditor

Security review and quality assurance. Read-only access plus specific analysis commands.

```yaml
auditor:
  adapter: claude
  system_prompt_file: .muzzle/personas/auditor.md
  temperature: 0.1
  permissions:
    allowed_tools: ["Read", "Grep", "Bash(npm audit*)", "Bash(go vet*)"]
    deny: ["Write(*)", "Edit(*)"]
```

### Summarizer

Context relay checkpoint generation. Minimal permissions — reads context and produces a checkpoint document.

```yaml
summarizer:
  adapter: claude
  system_prompt_file: .muzzle/personas/summarizer.md
  temperature: 0.0
  permissions:
    allowed_tools: ["Read"]
    deny: ["Write(*)", "Bash(*)"]
```

## Permission Model

Permissions use glob patterns matched against tool call signatures.

### Evaluation Order

```
1. Check deny patterns → MATCH = blocked
2. Check allowed_tools → MATCH = permitted
3. No match → blocked (implicit deny)
```

Deny **always wins**. A tool call matching both `allowed_tools` and `deny` is blocked.

### Inheritance

```
Adapter default_permissions (base)
         ↓
Persona permissions (override)
         ↓
Effective permissions (final)
```

- Persona `deny` patterns are **additive** (combined with adapter denies).
- Persona `allowed_tools` **replace** adapter allowed tools when specified.

## Hooks

Hooks inject custom logic at tool call boundaries:

| Hook | Timing | Blocking |
|------|--------|----------|
| `PreToolUse` | Before tool executes | Yes — non-zero exit blocks the call |
| `PostToolUse` | After tool completes | No — exit code logged only |

Hooks receive the tool call details via environment variables and can enforce project-specific policies (e.g., "always lint before commit", "run tests after write").

## System Prompts

System prompts are markdown files that shape agent behavior. They are injected as the `CLAUDE.md` file in the agent's workspace.

A good system prompt includes:

- **Role definition** — what the persona is and isn't.
- **Output format** — structured expectations for artifacts.
- **Constraints** — what the persona must avoid.
- **Quality criteria** — what "done" looks like.

## Further Reading

- [Manifest Schema — Persona Fields](/reference/manifest-schema#persona) — complete field reference
- [Adapters](/concepts/adapters) — how adapters and personas interact
- [Contracts](/concepts/contracts) — validating persona output
