# Personas

A persona is a specialized AI agent with specific permissions and behavior. Personas control what an AI can see, do, and how it responds.

```yaml
personas:
  navigator:
    adapter: claude
    system_prompt_file: .wave/personas/navigator.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Glob", "Grep"]
      deny: ["Write(*)", "Edit(*)"]
```

Use personas to enforce separation of concerns - a read-only navigator cannot accidentally modify files, and a craftsman has full write access.

## Built-in Personas

Wave includes these personas by default:

| Persona | Purpose | Permissions |
|---------|---------|-------------|
| `navigator` | Read-only codebase exploration | Read, Glob, Grep |
| `philosopher` | Design and specification | Read, Write to specs only |
| `planner` | Task breakdown and project planning | Read, Glob, Grep |
| `craftsman` | Implementation and testing | Full Read/Write/Bash |
| `debugger` | Issue diagnosis and root cause analysis | Read, Grep, Glob, git bisect |
| `auditor` | Security and quality review | Read-only with analysis tools |
| `summarizer` | Context compaction | Read-only |
| `github-analyst` | GitHub issue analysis | Read, Bash, Write |
| `github-enhancer` | GitHub issue enhancement | Read, Write, Bash |

## Permission Model

Permissions use glob patterns to control tool access:

```yaml
permissions:
  allowed_tools:
    - "Read"                    # All Read calls
    - "Write(src/*.ts)"         # Write to TypeScript in src/
    - "Bash(npm test*)"         # Only npm test commands
  deny:
    - "Write(*.env)"            # Never write env files
    - "Bash(rm -rf *)"          # Block destructive commands
```

**Evaluation order:**
1. Check `deny` patterns - if any match, blocked
2. Check `allowed_tools` - if any match, permitted
3. No match - blocked (implicit deny)

## Custom Personas

Define custom personas in your `wave.yaml`:

```yaml
personas:
  docs-writer:
    adapter: claude
    system_prompt_file: .wave/personas/docs-writer.md
    temperature: 0.5
    permissions:
      allowed_tools: ["Read", "Write(docs/*)"]
      deny: ["Bash(*)"]
```

## Using Personas in Pipelines

Reference personas by name in pipeline steps:

```yaml
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze the codebase structure"

  - id: implement
    persona: craftsman
    dependencies: [analyze]
    exec:
      type: prompt
      source: "Implement the feature"
```

## Next Steps

- [Pipelines](/concepts/pipelines) - Use personas in multi-step workflows
- [Contracts](/concepts/contracts) - Validate persona outputs
- [Manifest Reference](/reference/manifest) - Complete persona configuration
