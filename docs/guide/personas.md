# Personas Guide

Personas define how agents behave in Wave. Each persona binds an adapter to a specific role with its own permissions, system prompt, and behavior settings.

## Built-in Personas

`wave init` installs these personas into `.agents/personas/`:

| Persona | Purpose |
|---------|---------|
| `navigator` | Codebase exploration and analysis |
| `planner` | Task breakdown |
| `philosopher` | Pipeline and spec design (used by `wave meta`) |
| `craftsman` | Implementation and testing |
| `implementer` | Code implementation |
| `reviewer` | Code review |
| `auditor` | Security and quality review |
| `researcher` | Web and codebase research |
| `summarizer` | Synthesis and context compaction |
| `github-analyst` | GitHub issue analysis |
| `github-commenter` | Posting comments and PRs on GitHub |
| `github-scoper` | Creating scoped child issues on GitHub |

Every shipped persona runs with full tool access (`allowed_tools: ["*"]`, `deny: []`). Roles are
enforced through the system prompt, not through tool permissions. Scoping is available: set
`permissions.allowed_tools` and `permissions.deny` on a persona in `wave.yaml`, see
[Permissions System](#permissions-system).

## Defining Custom Personas

Add to the `personas` section of `wave.yaml`:

```yaml
personas:
  my-persona:
    adapter: claude
    description: "What this persona does"
    system_prompt_file: .agents/personas/my-persona.md
    #temperature: 0.5  # Optional — uncomment and adjust if needed
    permissions:
      allowed_tools: [...]
      deny: [...]
```

| Field | Required | Description |
|-------|----------|-------------|
| `adapter` | yes | References a key in `adapters` |
| `system_prompt_file` | yes | Path to system prompt markdown |
| `description` | no | Human-readable purpose |
| `temperature` | no | LLM temperature (0.0-1.0). Optional — commented out by default in wave.yaml |
| `permissions` | no | Tool access control |
| `hooks` | no | Pre/post tool hooks |

## Temperature Guidelines

| Range | Use Case |
|-------|----------|
| `0.0-0.2` | Deterministic: summarization, auditing, analysis |
| `0.3-0.5` | Balanced: specification, planning |
| `0.6-0.8` | Creative: implementation, generation |

## Permissions System

Permissions use two lists:

- `allowed_tools` — permitted operations
- `deny` — blocked operations (always takes precedence)

```yaml
permissions:
  allowed_tools:
    - "Read"              # All Read operations
    - "Write(src/*)"      # Write only in src/
    - "Bash(git *)"       # Git commands only
  deny:
    - "Bash(rm -rf *)"    # Block destructive commands
```

**Evaluation order**:
1. Check `deny` → MATCH = blocked
2. Check `allowed_tools` → MATCH = permitted
3. No match → blocked (implicit deny)

## Hooks

Hooks execute shell commands at tool call boundaries:

```yaml
hooks:
  PreToolUse:    # Non-zero exit blocks the tool call
    - matcher: "Bash(git commit*)"
      command: ".agents/hooks/pre-commit-lint.sh"
  PostToolUse:   # Informational only
    - matcher: "Write(src/**)"
      command: "npm test --silent"
```

## System Prompt Files

Store prompts in `.agents/personas/` as markdown:

```markdown
# Navigator

You are a codebase navigator. Explore and analyze without modifications.

## Responsibilities
- Map file structure and dependencies
- Identify relevant code for tasks
- Report architectural patterns

## Output Format
Provide structured JSON analysis with file paths.

## Constraints
- NEVER write, edit, or delete files
- NEVER run destructive commands
```

### CLI Command Security

When writing persona prompts that instruct agents to use CLI tools (`gh`, `tea`,
`glab`, `curl`), follow these rules to prevent shell injection from untrusted
content such as issue titles, PR bodies, and user comments:

- **Always use `--body-file`** instead of inline `--body "$content"`. Write
  content to a temp file first, then pass the file path to the CLI.
- **Always use single-quoted heredoc delimiters** (`<<'EOF'`) when heredocs
  are necessary. An unquoted `<<EOF` allows shell expansion of `$()`,
  backticks, and variables inside the heredoc body.
- **Never interpolate untrusted strings** directly into `--title`, `--body`,
  or other arguments inside double quotes.

See Secure CLI Patterns for detailed examples
and platform-specific guidance for GitHub, GitLab, Gitea, and Bitbucket.

## Testing Personas

```bash
wave list personas              # List all personas
wave validate --verbose         # Validate configuration
wave run ops-hello-world "test" --dry-run  # Check the manifest wiring
```

## Related Topics

- [Manifest Schema Reference](/reference/manifest-schema)
- [Pipelines Guide](/guide/pipelines)
- [Contracts Guide](/guide/contracts)
