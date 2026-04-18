# Personas Guide

Personas define how agents behave in Wave. Each persona binds an adapter to a specific role with its own permissions, system prompt, and behavior settings.

## Built-in Personas

Wave ships with 30 built-in personas:

| Persona | Purpose | Permissions |
|---------|---------|-------------|
| `navigator` | Codebase exploration | Read, Glob, Grep, Bash(git log\*), Bash(git status\*) |
| `philosopher` | Architecture & specs | Read, Write, Edit, Bash, Glob, Grep (full access) |
| `planner` | Task breakdown | Read, Write, Edit, Bash, Glob, Grep (full access) |
| `craftsman` | Implementation | Read, Write, Edit, Bash |
| `implementer` | Code implementation | Read, Write, Edit, Bash, Glob, Grep |
| `debugger` | Issue diagnosis | Read, Grep, Glob, Bash(go test\*), Bash(git log\*), Bash(git diff\*), Bash(git bisect\*) |
| `auditor` | Security review | Read, Grep, Bash(go vet\*), Bash(npm audit\*) |
| `reviewer` | Code review | Read, Glob, Grep, Bash(go test\*), Bash(npm test\*) |
| `researcher` | Research & analysis | Read, Write, Edit, Bash, Glob, Grep, WebSearch, WebFetch |
| `summarizer` | Context compaction | Read, Write, Edit, Bash, Glob, Grep (full access) |
| `supervisor` | Work quality review | Read, Glob, Grep, Bash(git \*), Bash(go test\*) |
| `validator` | Skeptical verification against source | Read, Glob, Grep, Bash(wc \*), Bash(git log\*) |
| `synthesizer` | Structured synthesis into JSON proposals | Read, Write, Edit, Bash, Glob, Grep (full access) |
| `provocateur` | Divergent thinking & complexity hunting | Read, Glob, Grep, Bash(wc \*), Bash(git log\*) |
| `github-analyst` | GitHub issue analysis | Read, Bash(gh issue\*), Bash(gh pr\*), Bash(git log\*) |
| `github-enhancer` | GitHub issue enhancement | Read, Bash(gh issue edit\*) |
| `github-commenter` | GitHub commenting | Read, Bash(gh issue comment\*), Bash(gh pr\*), Bash(git push\*) |
| `github-scoper` | GitHub epic scoping | Read, Bash(gh issue create\*), Bash(gh issue view\*) |
| `gitlab-analyst` | GitLab issue analysis | Read, Bash(glab issue\*), Bash(glab mr\*) |
| `gitlab-enhancer` | GitLab issue enhancement | Read, Bash(glab issue edit\*) |
| `gitlab-commenter` | GitLab commenting | Read, Bash(glab issue note\*), Bash(glab mr\*) |
| `gitlab-scoper` | GitLab epic scoping | Read, Bash(glab issue create\*) |
| `gitea-analyst` | Gitea issue analysis | Read, Bash(tea issues\*), Bash(tea pulls\*) |
| `gitea-enhancer` | Gitea issue enhancement | Read, Bash(tea issues edit\*) |
| `gitea-commenter` | Gitea commenting | Read, Bash(tea issues comment\*), Bash(tea pulls\*) |
| `gitea-scoper` | Gitea epic scoping | Read, Bash(tea issues create\*) |
| `bitbucket-analyst` | Bitbucket issue analysis | Read, Bash(curl -s\*), Bash(jq \*), Bash(git log\*) |
| `bitbucket-enhancer` | Bitbucket issue enhancement | Read, Bash(curl \*), Bash(jq \*) |
| `bitbucket-commenter` | Bitbucket commenting | Read, Bash(curl \*), Bash(git push\*) |
| `bitbucket-scoper` | Bitbucket epic scoping | Read, Bash(curl \*), Bash(jq \*) |

## Persona Definitions

### Navigator

Read-only codebase exploration. Finds files, analyzes patterns, maps architecture.

```yaml
navigator:
  adapter: claude
  description: "Read-only codebase exploration and analysis"
  system_prompt_file: .agents/personas/navigator.md
  #temperature: 0.3
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
  system_prompt_file: .agents/personas/philosopher.md
  #temperature: 0.7
  permissions:
    allowed_tools:
      - Read
      - Write
      - Edit
      - Bash
      - Glob
      - Grep
    deny: []
```

### Planner

Task breakdown and project planning. Decomposes features into actionable steps.

```yaml
planner:
  adapter: claude
  description: "Task breakdown and project planning"
  system_prompt_file: .agents/personas/planner.md
  #temperature: 0.3
  permissions:
    allowed_tools:
      - Read
      - Write
      - Edit
      - Bash
      - Glob
      - Grep
    deny: []
```

### Craftsman

Full implementation capability. Reads, writes, edits, and runs commands.

```yaml
craftsman:
  adapter: claude
  description: "Code implementation and testing"
  system_prompt_file: .agents/personas/craftsman.md
  #temperature: 0.3
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
  system_prompt_file: .agents/personas/debugger.md
  #temperature: 0.2
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

### Auditor

Security review and quality assurance. Read-only with analysis tools.

```yaml
auditor:
  adapter: claude
  description: "Security review and quality assurance"
  system_prompt_file: .agents/personas/auditor.md
  #temperature: 0.1
  permissions:
    allowed_tools:
      - Read
      - Grep
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
  system_prompt_file: .agents/personas/summarizer.md
  #temperature: 0.2
  permissions:
    allowed_tools:
      - Read
      - Write
      - Edit
      - Bash
      - Glob
      - Grep
    deny: []
```

### Implementer

Code implementation with full read/write access. Similar to craftsman but focused on spec-driven implementation.

### Reviewer

Code review specialist. Read-only analysis with detailed feedback on quality, patterns, and improvements.

### Researcher

Research and analysis agent. Explores codebases and external sources to gather information for decision-making.

### GitHub Analyst

Analyzes GitHub issues for completeness, categorization, and priority assessment.

### GitHub Enhancer

Enhances poorly documented GitHub issues with structured details, reproduction steps, and acceptance criteria.

### GitHub Commenter

Posts analysis results and recommendations as comments on GitHub issues.

### GitHub Scoper

Analyzes epic/umbrella issues and decomposes them into well-scoped child issues for implementation.

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

See [Secure CLI Patterns](/guides/secure-cli-patterns) for detailed examples
and platform-specific guidance for GitHub, GitLab, Gitea, and Bitbucket.

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
