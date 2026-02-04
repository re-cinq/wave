# Data Model: Add Missing Personas

**Feature**: 021-add-missing-personas
**Date**: 2026-02-04

## Entities

### Persona (wave.yaml)

Configuration entity defining an AI agent with specific capabilities.

**Fields**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| adapter | string | Yes | AI adapter to use (claude, opencode) |
| description | string | No | Human-readable purpose |
| permissions.allowed_tools | []string | Yes | Tools the persona can use |
| permissions.deny | []string | No | Explicitly denied tool patterns |
| system_prompt_file | string | Yes | Path to markdown system prompt |

**Validation Rules**:
- adapter must reference a defined adapter in wave.yaml
- system_prompt_file must exist at specified path
- Tool patterns support wildcards (e.g., `Bash(git *)`)

### System Prompt (Markdown File)

Markdown document providing persona instructions to the AI.

**Structure**:
```markdown
# [Persona Name]

[1-2 sentence role description]

## Responsibilities
- [Duty 1]
- [Duty 2]

## Output Format
[Structured output guidance]

## Constraints
- NEVER [hard limit 1]
- NEVER [hard limit 2]
```

**Validation Rules**:
- Must have H1 title matching persona name
- Should include Responsibilities section
- Should include Constraints section for safety

## New Persona Definitions

### Implementer Persona

```yaml
# wave.yaml addition
implementer:
  adapter: claude
  description: Code execution and artifact generation for pipeline steps
  permissions:
    allowed_tools:
      - Read
      - Write
      - Edit
      - Bash
      - Glob
      - Grep
    deny:
      - Bash(rm -rf /*)
      - Bash(sudo *)
  system_prompt_file: .wave/personas/implementer.md
```

### Reviewer Persona

```yaml
# wave.yaml addition
reviewer:
  adapter: claude
  description: Quality review, validation, and assessment
  permissions:
    allowed_tools:
      - Read
      - Glob
      - Grep
      - Write(artifact.json)
      - Write(artifacts/*)
      - Bash(go test*)
      - Bash(npm test*)
      - Bash(cargo test*)
      - Bash(pytest*)
    deny:
      - Write(*.go)
      - Write(*.ts)
      - Write(*.py)
      - Write(*.rs)
      - Edit(*)
      - Bash(rm *)
      - Bash(git push*)
      - Bash(git commit*)
  system_prompt_file: .wave/personas/reviewer.md
```

## Relationships

```
wave.yaml
├── adapters
│   └── claude (referenced by personas)
└── personas
    ├── implementer
    │   └── system_prompt_file → .wave/personas/implementer.md
    └── reviewer
        └── system_prompt_file → .wave/personas/reviewer.md

Pipeline Steps
├── persona: implementer
│   └── Resolves to personas.implementer in manifest
└── persona: reviewer
    └── Resolves to personas.reviewer in manifest
```

## State Transitions

Personas are stateless configuration - no state transitions apply.

## Artifact Output (Contract Compatibility)

Both personas must support JSON artifact output for pipeline handoff:

```json
// artifact.json structure (schema injected at runtime)
{
  // Fields defined by step's handover.contract.schema_path
  // Persona writes valid JSON matching the injected schema
}
```

**Key Constraint**: Persona system prompts must NOT embed schema details - schemas are injected by executor at runtime (executor.go:656-744).
