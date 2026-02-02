# Wave

**Multi-agent pipelines for AI-assisted development.**

Wave orchestrates LLM agents through structured pipelines — each step runs a specialized persona with scoped permissions, validated contracts, and isolated workspaces.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## Why Wave?

Running AI agents without guardrails is risky. A single prompt can read secrets, delete files, or push broken code.

Wave solves this with **persona-scoped safety**:

- **Navigator** can explore but never modify
- **Craftsman** can implement but not push to remote
- **Auditor** can review but not fix

Each persona runs in isolation with explicit permissions. Deny patterns always win.

---

## Quick Start

```bash
# Install
go install github.com/recinq/wave/cmd/wave@latest

# Initialize project
wave init

# Run your first pipeline
wave run --pipeline speckit-flow --input "add user authentication"

# Or run ad-hoc tasks
wave do "fix the failing test in auth_test.go"
```

---

## Core Concepts

### Personas — Scoped Agent Roles

```yaml
personas:
  navigator:
    adapter: claude
    temperature: 0.1
    permissions:
      allowed_tools: [Read, Glob, Grep]
      deny: [Write(*), Edit(*)]
```

7 built-in personas: `navigator`, `philosopher`, `planner`, `craftsman`, `debugger`, `auditor`, `summarizer`

### Pipelines — DAG Workflows

```yaml
steps:
  - id: navigate
    persona: navigator
  - id: implement
    persona: craftsman
    dependencies: [navigate]
  - id: review
    persona: auditor
    dependencies: [implement]
```

9 built-in pipelines: `speckit-flow`, `hotfix`, `code-review`, `refactor`, `debug`, `test-gen`, `docs`, `plan`, `migrate`

### Contracts — Validated Handoffs

Every step boundary validates output against JSON Schema, TypeScript interfaces, or test suites. Malformed artifacts trigger retries or halt the pipeline.

---

## Commands

| Command | Description |
|---------|-------------|
| `wave init` | Initialize project with personas and pipelines |
| `wave run` | Execute a pipeline |
| `wave do` | Quick ad-hoc task (auto-generates 2-step pipeline) |
| `wave list` | List pipelines, personas, adapters |
| `wave validate` | Check configuration |
| `wave resume` | Resume interrupted pipeline |
| `wave clean` | Remove workspaces and state |

---

## Pipelines

### Development

| Pipeline | Flow | Use Case |
|----------|------|----------|
| `speckit-flow` | navigate → specify → plan → implement → review | Feature development |
| `hotfix` | investigate → fix → verify | Production bugs |
| `refactor` | analyze → test-baseline → refactor → verify | Safe refactoring |

### Quality

| Pipeline | Flow | Use Case |
|----------|------|----------|
| `code-review` | diff → security + quality → summary | PR reviews |
| `test-gen` | analyze-coverage → generate → verify | Test coverage |
| `debug` | reproduce → hypothesize → investigate → fix | Root cause analysis |

### Planning & Docs

| Pipeline | Flow | Use Case |
|----------|------|----------|
| `plan` | explore → breakdown → review | Task planning |
| `docs` | discover → generate → review | Documentation |
| `migrate` | impact → plan → implement → review | Migrations |

---

## Personas

| Persona | Temp | Purpose | Key Permissions |
|---------|------|---------|-----------------|
| `navigator` | 0.1 | Codebase exploration | Read-only |
| `philosopher` | 0.3 | Architecture & specs | Read + write specs |
| `planner` | 0.3 | Task breakdown | Read-only |
| `craftsman` | 0.7 | Implementation | Full access |
| `debugger` | 0.2 | Root cause analysis | Read + git bisect |
| `auditor` | 0.1 | Security review | Read + audit tools |
| `summarizer` | 0.0 | Context compaction | Read-only |

---

## Examples

```bash
# Feature development
wave run --pipeline speckit-flow --input "add OAuth2 with Google"

# Fix production bug
wave run --pipeline hotfix --input "500 errors on /api/users"

# Review PR
wave run --pipeline code-review --input "review auth module changes"

# Generate tests
wave run --pipeline test-gen --input "improve coverage for internal/cache"

# Debug issue
wave run --pipeline debug --input "memory leak after 1000 requests"

# Plan feature
wave run --pipeline plan --input "implement real-time notifications"

# Quick fixes
wave do "fix typo in README"
wave do "add input validation to signup form"
wave do "review for SQL injection" --persona auditor
```

---

## Project Structure

```
wave.yaml                    # Project manifest
.wave/
├── personas/                # System prompts
│   ├── navigator.md
│   ├── craftsman.md
│   └── ...
├── pipelines/               # Pipeline definitions
│   ├── speckit-flow.yaml
│   ├── hotfix.yaml
│   └── ...
├── contracts/               # JSON schemas
├── workspaces/              # Ephemeral step workspaces
└── traces/                  # Audit logs
```

---

## Documentation

- [Quick Start Guide](docs/guide/quick-start.md)
- [Personas Guide](docs/guide/personas.md)
- [Pipelines Guide](docs/guide/pipelines.md)
- [CLI Reference](docs/reference/cli.md)
- [Manifest Schema](docs/reference/manifest-schema.md)

---

## Requirements

- Go 1.22+
- An LLM CLI adapter (`claude`, `opencode`, or custom)

---

## License

MIT
