# Creating Custom Personas

This guide walks you through creating custom personas for your Wave workflows. Custom personas allow you to define specialized AI agents with precisely scoped permissions and behaviors tailored to your team's needs.

## Overview

A custom persona consists of three key elements:

1. **System Prompt** - Instructions defining the persona's behavior and expertise
2. **Permissions** - Tool access controls using glob patterns
3. **Configuration** - Adapter settings, temperature, and other parameters

## Quick Start

Here's a minimal custom persona definition:

```yaml
# wave.yaml
personas:
  docs-writer:
    adapter: claude
    system_prompt_file: .wave/personas/docs-writer.md
    temperature: 0.5
    permissions:
      allowed_tools: ["Read", "Glob", "Write(docs/*)"]
      deny: ["Bash(*)"]
```

Create the system prompt file:

```markdown
<!-- .wave/personas/docs-writer.md -->
# Documentation Writer

You are a technical documentation expert. Your role is to:

- Write clear, concise documentation
- Follow the project's documentation style guide
- Create examples that are easy to understand
- Maintain consistency with existing documentation

Always:
- Use active voice
- Include code examples for technical concepts
- Add cross-references to related documentation
```

## Step-by-Step Guide

### 1. Define the Persona's Purpose

Before writing configuration, clearly define:

- **What the persona does** - Its primary responsibility
- **What it should NOT do** - Actions outside its scope
- **What it needs access to** - Required tools and files
- **How it behaves** - Tone, style, and approach

**Example:** A database migration specialist persona:

| Aspect | Definition |
|--------|------------|
| Purpose | Create and validate database migrations |
| Scope limit | Cannot modify application code |
| Access needed | Read all, write migrations only, execute tests |
| Behavior | Conservative, thorough, includes rollback logic |

### 2. Create the System Prompt

System prompts define how the persona thinks and responds. Write them in markdown format:

```markdown
<!-- .wave/personas/db-migrator.md -->
# Database Migration Specialist

You are an expert database migration engineer. Your primary responsibilities:

## Core Responsibilities
- Design safe, reversible database migrations
- Ensure data integrity during schema changes
- Optimize migration performance for large datasets
- Include comprehensive rollback logic

## Guidelines
1. **Safety First**: Always verify migrations won't cause data loss
2. **Idempotent Operations**: Migrations should be safe to run multiple times
3. **Rollback Logic**: Every UP migration needs a corresponding DOWN
4. **Testing**: Include test data scenarios in migration files
5. **Documentation**: Add clear comments explaining the purpose

## Constraints
- Never modify application code
- Never drop columns without explicit backup confirmation
- Always use transactions for multi-step migrations
- Limit batch sizes to prevent lock contention

## Output Format
Always produce migrations with:
- Timestamp prefix (YYYYMMDD_HHMMSS)
- Descriptive snake_case filename
- Both up() and down() functions
- Inline documentation
```

### 3. Configure Permissions

Permissions use the deny-first evaluation model. Design permissions based on the principle of least privilege:

```yaml
personas:
  db-migrator:
    adapter: claude
    system_prompt_file: .wave/personas/db-migrator.md
    temperature: 0.2  # Lower for precise output
    permissions:
      allowed_tools:
        # Read access
        - "Read"
        - "Glob"
        - "Grep"
        # Write access - migrations only
        - "Write(migrations/*)"
        - "Write(db/migrations/*)"
        - "Edit(migrations/*)"
        # Execution - specific commands only
        - "Bash(go run ./cmd/migrate*)"
        - "Bash(make migrate*)"
        - "Bash(psql -c \"SELECT*)"  # Read-only queries
      deny:
        # Block application code modification
        - "Write(src/*)"
        - "Write(internal/*)"
        - "Edit(src/*)"
        # Block destructive operations
        - "Bash(DROP DATABASE*)"
        - "Bash(rm -rf *)"
        - "Bash(truncate *)"
        # Block network access
        - "Bash(curl *)"
        - "Bash(wget *)"
```

### 4. Set Configuration Options

Fine-tune persona behavior with configuration options:

```yaml
personas:
  db-migrator:
    adapter: claude               # LLM adapter to use
    model: claude-sonnet-4-20250514     # Specific model (optional)
    system_prompt_file: .wave/personas/db-migrator.md
    temperature: 0.2              # 0.0-1.0, lower = more deterministic
    max_tokens: 4096              # Maximum response length
    timeout: 300                  # Seconds before timeout
    retry:
      max_attempts: 3             # Retry failed executions
      backoff: exponential        # Retry strategy
    permissions:
      allowed_tools: [...]
      deny: [...]
```

**Temperature Guidelines:**

| Temperature | Use Case | Examples |
|------------|----------|----------|
| 0.0 - 0.2 | Precise, deterministic output | Code generation, migrations |
| 0.3 - 0.5 | Balanced creativity | Documentation, refactoring |
| 0.6 - 0.8 | Creative exploration | Brainstorming, design |
| 0.9 - 1.0 | Maximum creativity | Rarely used in development |

## Common Persona Patterns

### Read-Only Analyst

For personas that analyze but never modify:

```yaml
personas:
  security-scanner:
    adapter: claude
    system_prompt_file: .wave/personas/security-scanner.md
    temperature: 0.1
    permissions:
      allowed_tools:
        - "Read"
        - "Glob"
        - "Grep"
      deny:
        - "Write(*)"
        - "Edit(*)"
        - "Bash(*)"
```

### Scoped Writer

For personas that write to specific directories:

```yaml
personas:
  api-generator:
    adapter: claude
    system_prompt_file: .wave/personas/api-generator.md
    temperature: 0.3
    permissions:
      allowed_tools:
        - "Read"
        - "Glob"
        - "Grep"
        - "Write(api/**/*.go)"
        - "Edit(api/**/*.go)"
        - "Bash(go fmt ./api/...)"
        - "Bash(go vet ./api/...)"
      deny:
        - "Write(internal/*)"
        - "Bash(go run *)"
```

### Test Generator

For personas focused on testing:

```yaml
personas:
  test-writer:
    adapter: claude
    system_prompt_file: .wave/personas/test-writer.md
    temperature: 0.3
    permissions:
      allowed_tools:
        - "Read"
        - "Glob"
        - "Grep"
        - "Write(*_test.go)"
        - "Write(tests/**/*)"
        - "Write(testdata/**/*)"
        - "Edit(*_test.go)"
        - "Bash(go test *)"
        - "Bash(go test -cover *)"
      deny:
        - "Write(*.go)"  # Non-test Go files
        - "Bash(go build *)"
```

### CI/CD Operator

For personas managing deployment workflows:

```yaml
personas:
  deploy-operator:
    adapter: claude
    system_prompt_file: .wave/personas/deploy-operator.md
    temperature: 0.1
    permissions:
      allowed_tools:
        - "Read"
        - "Glob"
        - "Write(.github/workflows/*)"
        - "Write(deploy/*)"
        - "Bash(kubectl get *)"
        - "Bash(kubectl describe *)"
        - "Bash(helm template *)"
      deny:
        - "Bash(kubectl delete *)"
        - "Bash(kubectl apply *)"  # Require manual approval
        - "Write(src/*)"
```

## Permission Pattern Reference

### Glob Pattern Syntax

| Pattern | Matches | Example |
|---------|---------|---------|
| `*` | Any characters in filename | `*.go` matches `main.go` |
| `**` | Any path depth | `src/**/*.ts` matches `src/a/b/c.ts` |
| `?` | Single character | `test?.go` matches `test1.go` |
| `[abc]` | Character class | `[mt]ain.go` matches `main.go` |
| `{a,b}` | Alternatives | `{src,lib}/*.go` |

### Tool-Specific Patterns

```yaml
permissions:
  allowed_tools:
    # File operations
    - "Read"                      # All reads
    - "Read(src/**/*)"            # Reads in src/
    - "Write(output.json)"        # Specific file
    - "Write(docs/**/*.md)"       # Markdown in docs/
    - "Edit(config.yaml)"         # Specific file edit
    - "Glob"                      # All glob searches
    - "Grep"                      # All grep searches

    # Bash commands
    - "Bash(go test *)"           # Go test commands
    - "Bash(npm run build)"       # Specific npm command
    - "Bash(make lint)"           # Specific make target
    - "Bash(git status)"          # Git status only
    - "Bash(git diff *)"          # Git diff commands
```

## Using Custom Personas in Pipelines

Reference your custom personas in pipeline steps:

```yaml
pipelines:
  database-update:
    description: "Create and apply database migrations"
    steps:
      - id: analyze-schema
        persona: navigator
        exec:
          type: prompt
          source: "Analyze current database schema and models"
        outputs:
          - schema-analysis.md

      - id: create-migration
        persona: db-migrator
        dependencies: [analyze-schema]
        inputs:
          - from: analyze-schema
            artifact: schema-analysis.md
        exec:
          type: prompt
          source: "Create migration for the requested schema change"
        outputs:
          - migration.sql
        contract:
          type: sql-syntax
          dialect: postgres

      - id: test-migration
        persona: tester
        dependencies: [create-migration]
        inputs:
          - from: create-migration
            artifact: migration.sql
        exec:
          type: prompt
          source: "Test the migration in a sandboxed environment"
```

## Best Practices

### 1. Principle of Least Privilege

Grant only the minimum permissions necessary:

```yaml
# Bad - too permissive
permissions:
  allowed_tools: ["*"]

# Good - specific permissions
permissions:
  allowed_tools:
    - "Read"
    - "Write(migrations/*.sql)"
  deny:
    - "Bash(*)"
```

### 2. Explicit Deny Lists

Always explicitly deny dangerous operations:

```yaml
permissions:
  deny:
    # Destructive commands
    - "Bash(rm -rf *)"
    - "Bash(drop *)"
    - "Bash(truncate *)"
    # Sensitive files
    - "Write(*.env*)"
    - "Write(*secret*)"
    - "Read(*.pem)"
    # Network operations
    - "Bash(curl *)"
    - "Bash(wget *)"
```

### 3. Clear System Prompts

Write system prompts that are:

- **Specific** - Clear responsibilities and constraints
- **Actionable** - Concrete guidelines the AI can follow
- **Bounded** - Explicit scope limitations

### 4. Version Control Persona Definitions

Keep persona definitions in version control:

```
.wave/
  personas/
    navigator.md
    implementer.md
    db-migrator.md      # Custom
    api-generator.md    # Custom
    security-scanner.md # Custom
```

### 5. Test Personas Independently

Test each persona before using in complex pipelines:

```bash
# Test persona with simple task
wave run --persona db-migrator "Create a migration to add email field to users table"
```

## Troubleshooting

### Permission Denied Errors

If a persona can't access required resources:

1. Check the `allowed_tools` patterns match the tool call
2. Verify no `deny` pattern is matching first
3. Enable debug logging to see permission evaluation:

```bash
wave run --debug --persona my-persona "task"
```

### Unexpected Behavior

If a persona behaves unexpectedly:

1. Review the system prompt for conflicting instructions
2. Adjust temperature (lower for more predictable output)
3. Add more specific guidelines to the system prompt

### Timeout Issues

If persona execution times out:

1. Increase the `timeout` configuration
2. Break complex tasks into smaller steps
3. Consider using a faster model for simple tasks

## Next Steps

- [Personas Concept](/concepts/personas) - Understand the permission model in depth
- [Pipelines](/concepts/pipelines) - Use personas in multi-step workflows
- [Contracts](/concepts/contracts) - Validate persona outputs
- [Audit Logging](/guides/audit-logging) - Monitor persona actions
