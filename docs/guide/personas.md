# Personas Guide

Personas define how agents behave in Wave. Each persona binds an adapter to a specific role with its own permissions, system prompt, and behavior settings.

## Built-in Personas

Wave ships with 13 built-in personas:

| Persona | Purpose | Permissions |
|---------|---------|-------------|
| `navigator` | Codebase exploration | Read, Glob, Grep, git log/status |
| `philosopher` | Architecture & specs | Read, Write(.wave/specs/*) |
| `planner` | Task breakdown | Read, Glob, Grep |
| `craftsman` | Implementation | Read, Write, Edit, Bash |
| `debugger` | Issue diagnosis | Read, Grep, git bisect, go test |
| `reviewer` | Security & code review | Read, Grep, Glob, go vet, npm audit |
| `researcher` | Research & analysis | Read, Grep, Glob, WebSearch |
| `summarizer` | Context compaction | Read only |
| `github-analyst` | GitHub issue analysis | Read, Grep, Bash(gh *) |
| `github-enhancer` | GitHub issue enhancement | Read, Bash(gh *) |
| `github-commenter` | GitHub issue commenting | Read, Bash(gh *) |
| `github-pr-creator` | Pull request creation | Read, Bash(gh *), Bash(git *) |

## Persona Definitions

### Navigator

Read-only codebase exploration. Finds files, analyzes patterns, maps architecture.

```yaml
navigator:
  adapter: claude
  description: "Read-only codebase exploration"
  system_prompt_file: .wave/personas/navigator.md
  temperature: 0.1
  permissions:
    allowed_tools:
      - Read
      - Glob
      - Grep
      - "Bash(git log*)"
      - "Bash(git status*)"
    deny:
      - "Write(*)"
      - "Edit(*)"
      - "Bash(git commit*)"
      - "Bash(git push*)"
```

### Philosopher

Design and specification. Creates specs, plans, and contracts.

```yaml
philosopher:
  adapter: claude
  description: "Architecture design and specification"
  system_prompt_file: .wave/personas/philosopher.md
  temperature: 0.3
  permissions:
    allowed_tools:
      - Read
      - "Write(.wave/specs/*)"
    deny:
      - "Bash(*)"
```

### Planner

Task breakdown and project planning. Decomposes features into actionable steps.

```yaml
planner:
  adapter: claude
  description: "Task breakdown and project planning"
  system_prompt_file: .wave/personas/planner.md
  permissions:
    allowed_tools:
      - Read
      - Glob
      - Grep
    deny:
      - "Write(*)"
      - "Edit(*)"
      - "Bash(*)"
```

### Craftsman

Full implementation capability. Reads, writes, edits, and runs commands.

```yaml
craftsman:
  adapter: claude
  description: "Code implementation and testing"
  system_prompt_file: .wave/personas/craftsman.md
  temperature: 0.7
  permissions:
    allowed_tools:
      - Read
      - Write
      - Edit
      - Bash
    deny:
      - "Bash(rm -rf /*)"
```

### Debugger

Systematic issue diagnosis with hypothesis testing and root cause analysis.

```yaml
debugger:
  adapter: claude
  description: "Systematic issue diagnosis and root cause analysis"
  system_prompt_file: .wave/personas/debugger.md
  temperature: 0.2
  permissions:
    allowed_tools:
      - Read
      - Grep
      - Glob
      - "Bash(go test*)"
      - "Bash(git log*)"
      - "Bash(git diff*)"
      - "Bash(git bisect*)"
    deny:
      - "Write(*)"
      - "Edit(*)"
```

### Reviewer

Security and code review. Read-only with analysis tools.

```yaml
reviewer:
  adapter: claude
  description: "Security and code review"
  system_prompt_file: .wave/personas/reviewer.md
  temperature: 0.1
  permissions:
    allowed_tools:
      - Read
      - Grep
      - Glob
      - "Bash(go vet*)"
      - "Bash(npm audit*)"
    deny:
      - "Write(*)"
      - "Edit(*)"
```

### Summarizer

Context compaction for relay handoffs. Creates structured checkpoints.

```yaml
summarizer:
  adapter: claude
  description: "Context compaction for relay handoffs"
  system_prompt_file: .wave/personas/summarizer.md
  temperature: 0.0
  permissions:
    allowed_tools:
      - Read
    deny:
      - "Write(*)"
      - "Bash(*)"
```

### Researcher

Research and analysis agent. Explores codebases and external sources to gather information for decision-making.

### GitHub Analyst

Analyzes GitHub issues for completeness, categorization, and priority assessment.

### GitHub Enhancer

Enhances poorly documented GitHub issues with structured details, reproduction steps, and acceptance criteria.

### GitHub Commenter

Posts analysis results and recommendations as comments on GitHub issues.

### GitHub PR Creator

Creates pull requests with proper descriptions, linking related issues and summarizing changes.

## Defining Custom Personas

Add to the `personas` section of `wave.yaml`:

```yaml
personas:
  my-persona:
    adapter: claude
    description: "What this persona does"
    system_prompt_file: .wave/personas/my-persona.md
    temperature: 0.5
    permissions:
      allowed_tools: [...]
      deny: [...]
```

| Field | Required | Description |
|-------|----------|-------------|
| `adapter` | yes | References a key in `adapters` |
| `system_prompt_file` | yes | Path to system prompt markdown |
| `description` | no | Human-readable purpose |
| `temperature` | no | LLM temperature (0.0-1.0) |
| `permissions` | no | Tool access control |
| `hooks` | no | Pre/post tool hooks |

## Temperature Guidelines

| Range | Use Case |
|-------|----------|
| `0.0-0.2` | Deterministic: summarization, review, analysis |
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
      command: ".wave/hooks/pre-commit-lint.sh"
  PostToolUse:   # Informational only
    - matcher: "Write(src/**)"
      command: "npm test --silent"
```

## System Prompt Files

Store prompts in `.wave/personas/` as markdown:

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

## Testing Personas

```bash
wave list personas              # List all personas
wave validate --verbose         # Validate configuration
wave do "analyze this" --persona navigator  # Test with task
```

## Related Topics

- [Manifest Schema Reference](/reference/manifest-schema)
- [Pipelines Guide](/guide/pipelines)
- [Contracts Guide](/guide/contracts)
