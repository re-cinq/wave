# Wave — AI-as-Code

**Infrastructure thinking for AI-native workflows.**

Wave brings Infrastructure-as-Code principles to AI. Define multi-agent pipelines in YAML, version them in git, and run them with contracts, isolation, and audit trails.

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
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
git clone https://github.com/re-cinq/wave.git
cd wave && ./install.sh

# Initialize project (in your target project)
cd /path/to/your/project
wave init

# Run your first pipeline
wave run speckit-flow "add user authentication"

# Or run ad-hoc tasks
wave do "fix the failing test in auth_test.go"

# Monitor running pipelines
wave status

# View logs
wave logs --follow
```

---

## CLI Reference

```
  ╦ ╦╔═╗╦  ╦╔═╗
  ║║║╠═╣╚╗╔╝║╣
  ╚╩╝╩ ╩ ╚╝ ╚═╝
  Multi-Agent Pipeline Orchestrator

Wave coordinates multiple AI personas through structured pipelines,
enforcing permissions, contracts, and workspace isolation at every step.

Usage:
  wave [command]

Available Commands:
  artifacts   List and export pipeline artifacts
  cancel      Cancel a running pipeline
  clean       Clean up project artifacts
  completion  Generate the autocompletion script for the specified shell
  do          Execute an ad-hoc task
  help        Help about any command
  init        Initialize a new Wave project
  list        List pipelines and personas
  logs        Show pipeline logs
  meta        Generate a custom pipeline
  migrate     Database migration commands
  resume      Resume a paused pipeline
  run         Run a pipeline
  status      Show pipeline status
  validate    Validate Wave configuration

Flags:
  -d, --debug             Enable debug mode
  -h, --help              help for wave
  -m, --manifest string   Path to manifest file (default "wave.yaml")
  -o, --output string     Output format: auto, json, text, quiet (default "auto")
  -v, --verbose           Include real-time tool activity
      --version           version for wave

Use "wave [command] --help" for more information about a command.
```

---

## Commands

### Pipeline Execution

| Command | Description |
|---------|-------------|
| `wave init` | Initialize project with personas and pipelines |
| `wave run --pipeline <name>` | Execute a pipeline |
| `wave do "<task>"` | Quick ad-hoc task (auto-generates 2-step pipeline) |
| `wave meta "<task>"` | Generate custom multi-step pipeline with schemas |
| `wave resume` | Resume interrupted pipeline |
| `wave cancel [run-id]` | Cancel running pipeline (graceful or `--force`) |

### Monitoring & Inspection

| Command | Description |
|---------|-------------|
| `wave status [run-id]` | Show pipeline status (running, recent, details) |
| `wave logs [run-id]` | View event logs (`--follow`, `--tail`, `--errors`) |
| `wave artifacts [run-id]` | List and export pipeline artifacts |
| `wave list [resource]` | List pipelines, personas, adapters, or runs |

### Maintenance

| Command | Description |
|---------|-------------|
| `wave validate` | Check manifest and pipeline configuration |
| `wave clean` | Remove workspaces and state (`--older-than`, `--status`) |

---

## Command Examples

### Running Pipelines

```bash
# Feature development
wave run speckit-flow "add OAuth2 with Google"

# Fix production bug
wave run hotfix "500 errors on /api/users"

# Review PR
wave run code-review "review auth module changes"

# Generate tests
wave run test-gen "improve coverage for internal/cache"

# Quick fixes
wave do "fix typo in README"
wave do "add input validation to signup form"
wave do "review for SQL injection" --persona auditor
```

### Meta Pipeline Generation

Generate custom pipelines dynamically using `wave meta`:

```bash
# Generate a custom pipeline for your task
wave meta "implement user authentication"
wave meta "add database migrations"
wave meta "create REST API endpoints"

# Save generated pipeline for reuse
wave meta "implement caching layer" --save my-cache-pipeline.yaml

# Dry run to see what would be generated
wave meta "add monitoring dashboard" --dry-run
```

**What happens with `wave meta`:**
1. **AI Architect** analyzes your task and designs a custom pipeline
2. **Auto-generates schemas** for data validation between steps
3. **Creates optimized workflow** with appropriate personas and dependencies
4. **Executes immediately** or saves for later use

**Generated pipelines include:**
- ✅ Navigator step for codebase analysis
- ✅ Specification and planning steps
- ✅ Implementation with proper personas
- ✅ Review and validation steps
- ✅ JSON schemas for contract validation
- ✅ Proper output artifact handling

### Monitoring Pipelines

```bash
# Show currently running pipelines
wave status

# Show all recent pipelines
wave status --all

# Show specific run details
wave status debug-20260202-143022

# Stream logs in real-time
wave logs --follow

# Show only errors from last run
wave logs --errors

# Show last 50 log entries
wave logs --tail 50

# Filter logs by step
wave logs --step investigate

# JSON output for scripting
wave status --format json
wave logs --format json
```

### Managing Pipelines

```bash
# Cancel most recent running pipeline
wave cancel

# Force cancel (SIGTERM/SIGKILL)
wave cancel --force

# Cancel specific run
wave cancel abc123-def456

# List recent runs
wave list runs

# List runs filtered by status
wave list runs --run-status failed

# Export artifacts from a run
wave artifacts --export ./output

# Export specific step's artifacts
wave artifacts --step implement --export ./output
```

### Cleanup

```bash
# Preview what would be deleted
wave clean --dry-run --all

# Clean all workspaces
wave clean --all

# Clean workspaces older than 7 days
wave clean --older-than 7d

# Clean only failed pipelines
wave clean --status failed

# Keep last 5 workspaces
wave clean --all --keep-last 5
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

**14 built-in personas** including `navigator`, `craftsman`, `auditor`, `philosopher`, `debugger`, and more.

> Explore all personas in [`.wave/personas/`](.wave/personas/)

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

**23 built-in pipelines** for development, debugging, documentation, and GitHub automation.

> Explore all pipelines in [`.wave/pipelines/`](.wave/pipelines/)

### Contracts — Validated Handoffs

Every step boundary validates output against JSON Schema, TypeScript interfaces, or test suites. Malformed artifacts trigger retries or halt the pipeline.

---

## Pipelines

A selection of the 23 built-in pipelines:

### Development

| Pipeline | Description |
|----------|-------------|
| `speckit-flow` | Specification-driven feature development |
| `hotfix` | Quick investigation and fix for production issues |
| `refactor` | Safe refactoring with comprehensive test coverage |
| `prototype` | Prototype-driven development (spec → docs → dummy → implement → pr) |

### Quality & Debugging

| Pipeline | Description |
|----------|-------------|
| `code-review` | Comprehensive code review for pull requests |
| `test-gen` | Generate comprehensive test coverage |
| `debug` | Systematic debugging with hypothesis testing |

### Planning & Documentation

| Pipeline | Description |
|----------|-------------|
| `plan` | Break down a feature into actionable tasks |
| `docs` | Generate or update documentation |
| `migrate` | Database or API migration with rollback plan |

### GitHub Automation

| Pipeline | Description |
|----------|-------------|
| `github-issue-enhancer` | Analyze and enhance poorly documented issues |
| `doc-loop` | Documentation impact analysis before merge |

> **More pipelines:** `docs-to-impl`, `hello-world`, `issue-research`, `smoke-test`, `gh-poor-issues`, `umami`
>
> Explore all in [`.wave/pipelines/`](.wave/pipelines/)

---

## Personas

A selection of the 14 built-in personas:

| Persona | Temp | Purpose | Key Permissions |
|---------|------|---------|--------------------|
| `navigator` | 0.1 | Codebase exploration | Read-only |
| `philosopher` | 0.3 | Architecture & specs | Read + write specs |
| `planner` | 0.3 | Task breakdown | Read-only |
| `craftsman` | 0.7 | Implementation | Full access |
| `debugger` | 0.2 | Root cause analysis | Read + git bisect |
| `auditor` | 0.1 | Security review | Read + audit tools |
| `summarizer` | 0.0 | Context compaction | Read-only |

> **More personas:** `implementer`, `researcher`, `reviewer`, `github-analyst`, `github-commenter`, `github-enhancer`, `github-pr-creator`
>
> Explore all in [`.wave/personas/`](.wave/personas/)

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
├── pids/                    # Process ID files for cancel
├── state.db                 # SQLite state database
└── traces/                  # Audit logs
```

---

## Documentation

- [Quick Start Guide](docs/guide/quick-start.md)
- [Installation](docs/guide/installation.md)
- [Sandbox Setup](docs/guides/sandbox-setup.md)
- [Personas Guide](docs/guide/personas.md)
- [Pipelines Guide](docs/guide/pipelines.md)
- [CLI Reference](docs/reference/cli.md)
- [Manifest Schema](docs/reference/manifest-schema.md)
- [Pipeline Schema](docs/reference/pipeline-schema.md)
- [Event Reference](docs/reference/events.md)
- [Adapters Reference](docs/reference/adapters.md)

---

## Sandboxed Development

Wave provides defense-in-depth isolation for AI agent sessions.

### Nix Dev Shell (Recommended)

```bash
# Enter sandboxed shell (bubblewrap on Linux, unsandboxed on macOS)
nix develop

# Escape hatch: no sandbox
nix develop .#yolo
```

The sandbox isolates the entire session:
- **Filesystem**: `/` is read-only, only project dir + `~/.claude` + `~/go` + `/tmp` are writable
- **Home directory**: hidden via `tmpfs` — selective read-only mounts for `~/.ssh`, `~/.gitconfig`, `~/.config/gh`
- **Environment**: Nix-provided environment inherited (no `--clearenv`)
- **Process isolation**: `--unshare-all` + `--die-with-parent`

### Manifest-Driven Permissions

Persona permissions from `wave.yaml` are projected into Claude Code's `settings.json` and `CLAUDE.md`:

```yaml
personas:
  navigator:
    permissions:
      allowed_tools: [Read, Glob, Grep]
      deny: [Write(*), Edit(*)]
    sandbox:
      allowed_domains: [api.anthropic.com]

runtime:
  sandbox:
    enabled: true
    default_allowed_domains: [api.anthropic.com, github.com]
    env_passthrough: [ANTHROPIC_API_KEY, GH_TOKEN]
```

See [Sandbox Setup Guide](docs/guides/sandbox-setup.md) for details.

---

## Requirements

- Go 1.25+
- An LLM CLI adapter (`claude`, `opencode`, or custom)
- [Nix](https://nixos.org/download.html) (optional, for sandboxed development)

---

## License

MIT
