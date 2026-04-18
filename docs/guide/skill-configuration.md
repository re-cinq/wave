# Skill Configuration Guide

This guide explains how to configure skills in your `wave.yaml` manifest at global, persona, and pipeline scopes, including precedence rules and deduplication behavior.

## Configuration Scopes

Skills can be declared at three levels in `wave.yaml`. When a pipeline step runs, skills from all applicable scopes are merged, deduplicated, and sorted alphabetically.

### Global Scope

Skills declared at the top level of `wave.yaml` are available to all personas and pipelines.

```yaml
# wave.yaml
skills:
  golang:
    install: "tessl:github/golang"
  spec-kit:
    install: "tessl:github/spec-kit"
```

### Persona Scope

Skills declared within a persona definition are available to all pipeline steps using that persona.

```yaml
# wave.yaml
personas:
  craftsman:
    prompt: ".agents/personas/craftsman.md"
    skills:
      golang:
        install: "tessl:github/golang"
      testing:
        install: "file:./skills/testing"
```

### Pipeline Scope

Skills declared within a pipeline step are available only to that specific step.

```yaml
# wave.yaml
pipelines:
  implement:
    steps:
      - name: code
        persona: craftsman
        skills:
          linting:
            install: "tessl:github/linting"
```

## Precedence Rules

When the same skill name appears at multiple scopes, the resolution follows this precedence order:

**pipeline > persona > global**

Skills are deduplicated by name. If `golang` is declared at all three scopes, it appears exactly once in the resolved list.

### Worked Example

Given this configuration:

```yaml
# wave.yaml
skills:
  golang:
    install: "tessl:github/golang"
  shared:
    install: "tessl:github/shared"

personas:
  craftsman:
    prompt: ".agents/personas/craftsman.md"
    skills:
      golang:
        install: "file:./skills/golang-custom"
      testing:
        install: "tessl:github/testing"

pipelines:
  implement:
    steps:
      - name: code
        persona: craftsman
        skills:
          linting:
            install: "tessl:github/linting"
```

When the `code` step runs with the `craftsman` persona, the resolved skill list is:

```
golang, linting, shared, testing
```

- `golang` appears once (deduplicated across global, persona, and pipeline)
- `linting` comes from the pipeline scope
- `shared` comes from the global scope
- `testing` comes from the persona scope
- The list is sorted alphabetically

## SkillConfig Fields

Each skill declaration supports these fields:

| Field | Description |
|-------|-------------|
| `install` | Source string for installation (e.g., `tessl:github/golang`, `file:./path`) |
| `init` | Command to run after installation for one-time setup |
| `check` | Command to verify the skill is properly installed |
| `commands_glob` | Glob pattern for discovering skill commands (e.g., `scripts/*.sh`) |

### Example with All Fields

```yaml
skills:
  golang:
    install: "tessl:github/golang"
    init: "scripts/setup.sh"
    check: "scripts/verify.sh"
    commands_glob: "scripts/*.sh"
```

## Deduplication Behavior

The `ResolveSkills` function merges skill names from all three scopes:

1. Collects all skill names from pipeline, persona, and global scopes
2. Removes duplicates (a skill name appears at most once)
3. Sorts the result alphabetically
4. Returns `nil` if no skills are declared at any scope

This means the actual skill installation source is determined by the `install` field at the highest-precedence scope where the skill is declared.

## Common Patterns

### Shared Base Skills

Declare commonly-used skills at global scope and override per-persona as needed:

```yaml
skills:
  golang:
    install: "tessl:github/golang"

personas:
  craftsman:
    skills:
      golang:
        install: "file:./skills/golang-strict"
```

### Pipeline-Specific Skills

Add specialized skills only where needed:

```yaml
pipelines:
  security-audit:
    steps:
      - name: scan
        skills:
          security:
            install: "tessl:github/security-audit"
```
