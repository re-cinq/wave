# Personas Guide

Personas define how agents behave in Wave. Each persona binds an adapter to a specific role with its own permissions, system prompt, and behavior settings.

## Defining a Persona

Personas live in the `personas` section of `wave.yaml`:

```yaml
personas:
  navigator:
    adapter: claude
    description: "Read-only codebase exploration"
    system_prompt_file: .wave/personas/navigator.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Glob", "Grep", "Bash(git *)"]
      deny: ["Write(*)", "Edit(*)"]
```

| Field | Required | Description |
|-------|----------|-------------|
| `adapter` | yes | References a key in `adapters` |
| `system_prompt_file` | yes | Path to system prompt markdown |
| `description` | no | Human-readable purpose |
| `temperature` | no | LLM temperature (0.0-1.0) |
| `permissions` | no | Tool access control |
| `hooks` | no | Pre/post tool hooks |

## Temperature Settings

| Temperature | Use Case |
|-------------|----------|
| `0.0-0.2` | Deterministic: summarization, auditing |
| `0.3-0.5` | Balanced: specification, planning |
| `0.6-0.8` | Creative: implementation |

## Permissions System

Permissions use two lists: `allowed_tools` (permitted) and `deny` (blocked, always takes precedence).

```yaml
permissions:
  allowed_tools:
    - "Read"              # All Read operations
    - "Write(src/*)"      # Write only in src/
    - "Bash(git *)"       # Git commands only
  deny:
    - "Bash(rm -rf *)"    # Block destructive commands
```

**Evaluation order**: Check `deny` first (block if match), then `allowed_tools` (permit if match), otherwise block.

**Inheritance**: Persona `deny` patterns are additive with adapter denies. Persona `allowed_tools` replace adapter defaults.

## Hooks

Hooks execute shell commands triggered by tool call patterns.

```yaml
hooks:
  PreToolUse:    # Runs before tool call; non-zero exit blocks it
    - matcher: "Bash(git commit*)"
      command: ".wave/hooks/pre-commit-lint.sh"
  PostToolUse:   # Runs after tool call; informational only
    - matcher: "Write(src/**)"
      command: "npm test --silent"
```

## Built-in Archetypes

### Navigator (Read-only analysis)

```yaml
navigator:
  adapter: claude
  system_prompt_file: .wave/personas/navigator.md
  temperature: 0.1
  permissions:
    allowed_tools: ["Read", "Glob", "Grep", "Bash(git log*)"]
    deny: ["Write(*)", "Edit(*)"]
```

### Craftsman (Implementation)

```yaml
craftsman:
  adapter: claude
  system_prompt_file: .wave/personas/craftsman.md
  temperature: 0.7
  permissions:
    allowed_tools: ["Read", "Write", "Edit", "Bash"]
    deny: ["Bash(rm -rf /*)"]
```

### Auditor (Security review)

```yaml
auditor:
  adapter: claude
  system_prompt_file: .wave/personas/auditor.md
  temperature: 0.1
  permissions:
    allowed_tools: ["Read", "Grep", "Bash(npm audit*)"]
    deny: ["Write(*)", "Edit(*)"]
```

### Summarizer (Relay checkpoints)

```yaml
summarizer:
  adapter: claude
  system_prompt_file: .wave/personas/summarizer.md
  temperature: 0.0
  permissions:
    allowed_tools: ["Read"]
    deny: ["Write(*)", "Bash(*)"]
```

## System Prompt Files

Store prompts in `.wave/personas/` as markdown:

```markdown
# Navigator Persona

You are a codebase navigator. Explore and analyze code without making changes.

## Responsibilities
- Map file structure and dependencies
- Identify relevant code for tasks
- Report architectural patterns

## Output Format
Provide structured analysis with file paths and code snippets.
```

## Testing Personas

```bash
wave list personas              # List all personas
wave validate --verbose         # Validate configuration
wave do "analyze this" --persona navigator  # Test with task
```

## Related Topics

- [Manifest Schema Reference](/reference/manifest-schema) - Full field reference
- [Pipelines Guide](/guide/pipelines) - Using personas in steps
- [Contracts Guide](/guide/contracts) - Validating persona output
