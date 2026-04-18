# Wave — AI-as-Code

**Orchestration for agent factories. Just the right amount of guardrails.**

Wave is the open-source orchestration layer for AI agent factories — define multi-agent pipelines in YAML, scope each persona's permissions precisely, and run repeatable workflows with contract-validated handoffs and full audit trails.

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## Why Wave?

Agent factories need boundaries — not to hobble agents, but to make them trustworthy enough to run unsupervised.

Most teams end up at one of two extremes: agents wrapped in approval loops that accomplish nothing, or unconstrained agents one misread prompt away from a production incident. Wave finds the middle path — **just the right amount of guardrails**.

Scoping is declarative, per-persona, enforced at runtime, and versioned in git. Agents do more. Accidents don't happen.

> Wave is the orchestration layer described in [Building Agent Factories](https://re-cinq.com/blog/building-agent-factories) — the open-source runtime for specification-driven AI workflows at scale.

---

## Forge Compatibility

Wave works with every major git hosting platform. Auto-detection from git remotes — zero configuration needed.

<table>
<tr>
<td align="center"><img src="https://cdn.simpleicons.org/github/white" width="40" height="40" alt="GitHub"><br><strong>GitHub</strong><br><code>gh</code> CLI</td>
<td align="center"><img src="https://cdn.simpleicons.org/gitlab" width="40" height="40" alt="GitLab"><br><strong>GitLab</strong><br><code>glab</code> CLI</td>
<td align="center"><img src="https://cdn.simpleicons.org/gitea" width="40" height="40" alt="Gitea"><br><strong>Gitea</strong><br><code>tea</code> CLI</td>
<td align="center"><img src="https://cdn.simpleicons.org/forgejo" width="40" height="40" alt="Forgejo"><br><strong>Forgejo</strong><br><code>tea</code> CLI</td>
<td align="center"><img src="https://cdn.simpleicons.org/codeberg" width="40" height="40" alt="Codeberg"><br><strong>Codeberg</strong><br><code>tea</code> CLI</td>
<td align="center"><img src="https://cdn.simpleicons.org/bitbucket" width="40" height="40" alt="Bitbucket"><br><strong>Bitbucket</strong><br><code>bb</code> CLI</td>
</tr>
</table>

- **Auto-detection**: Hostname matching + API endpoint probing for self-hosted instances
- **Forge-aware personas**: Each forge gets specialized personas (e.g., `gitea-commenter`, `github-analyst`)
- **Template variables**: `{{ forge.cli_tool }}`, `{{ forge.pr_command }}` resolve per-forge
- **Local mode**: Full operation without any forge (`forge: local`)
- **[Setup guide](docs/guides/forge-setup.md)**

---

## Installation

### Install Script (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh -s -- 0.3.0
```

### GitHub Releases

Download pre-built binaries from [GitHub Releases](https://github.com/re-cinq/wave/releases). Archives are available for Linux (x86_64, ARM64) and macOS (Intel, Apple Silicon).

### Build from Source

```bash
git clone https://github.com/re-cinq/wave.git
cd wave
make build
# Binary is at ./wave — move it to your PATH:
make install   # installs to ~/.local/bin by default
```

### Nix Dev Shell (Optional)

```bash
nix develop
```

See [Installation Guide](docs/guide/installation.md) for more options including `.deb` packages and custom install directories.

---

## Prerequisites

Before using Wave, ensure you have the following installed:

- **Go 1.25+** — required for building from source (optional if using pre-built binaries)
- **Claude Code CLI** (`claude`) — or any supported adapter binary (required for pipeline execution)
- **Git 2.x+** — for version control and worktree isolation
- **SQLite** — bundled with Wave, no external dependency needed
- **Optional:**
  - `gh` CLI — for GitHub issue and PR automation pipelines
  - `glab` CLI — for GitLab integration
  - Docker — for container-based sandbox isolation
  - Nix — for bubblewrap sandbox environment

Run `wave doctor` after installation to verify your environment is correctly configured.

---

## Quick Start

```bash
# Initialize Wave in your project
cd /path/to/your/project
wave init

# Run your first pipeline
wave run ops-hello-world

# Run a feature development pipeline
wave run impl-speckit "add user authentication"

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
  agent       Persona-to-agent compiler utilities
  artifacts   List and export pipeline artifacts
  bench       Run and analyze SWE-bench benchmarks
  cancel      Cancel a running pipeline
  chat        Open interactive analysis of a pipeline run
  clean       Clean up project artifacts
  completion  Generate the autocompletion script for the specified shell
  compose     Validate and execute a pipeline sequence
  do          Execute an ad-hoc task
  doctor      Check project health and environment setup
  help        Help about any command
  init        Initialize a new Wave project
  list        List Wave configuration and resources
  logs        Show pipeline logs
  meta        Generate and run a custom pipeline dynamically
  migrate     Database migration management
  postmortem  Analyse a failed pipeline run and suggest recovery steps
  resume      Resume a failed pipeline run
  run         Run a pipeline
  serve       Start the web dashboard server
  skills      Skill lifecycle management
  status      Show pipeline status
  suggest     Propose pipeline runs based on codebase state
  validate    Validate Wave configuration

Flags:
  -d, --debug             Enable debug mode
  -h, --help              help for wave
  -m, --manifest string   Path to manifest file (default "wave.yaml")
  -o, --output string     Output format: auto, json, text, quiet (default "auto")
  -v, --verbose           Include real-time tool activity
      --no-tui            Disable TUI and print help text
      --version           version for wave

Use "wave [command] --help" for more information about a command.
```

---

## Commands

### Pipeline Execution

| Command | Description |
|---------|-------------|
| `wave init` | Initialize project with personas and pipelines |
| `wave run <pipeline>` | Execute a pipeline |
| `wave resume <run-id>` | Resume a failed pipeline run |
| `wave fork <run-id>` | Fork a run from a checkpoint |
| `wave rewind <run-id>` | Rewind a run to an earlier checkpoint |
| `wave do "<task>"` | Quick ad-hoc task (auto-generates 2-step pipeline) |
| `wave meta "<task>"` | Generate and run a custom pipeline dynamically |
| `wave compose <file>` | Validate and execute a pipeline sequence |
| `wave cancel [run-id]` | Cancel running pipeline (graceful or `--force`) |

### Monitoring & Inspection

| Command | Description |
|---------|-------------|
| `wave status [run-id]` | Show pipeline status (running, recent, details) |
| `wave logs [run-id]` | View event logs (`--follow`, `--tail`, `--errors`) |
| `wave artifacts [run-id]` | List and export pipeline artifacts |
| `wave list [resource]` | List Wave configuration and resources |
| `wave decisions [run-id]` | Show structured decision log for a run |
| `wave retro [run-id]` | View and manage run retrospectives |
| `wave chat [run-id]` | Open interactive analysis of a pipeline run |
| `wave postmortem [run-id]` | Analyse a failed pipeline run and suggest recovery steps |
| `wave analyze [run-id]` | Deep analysis of pipeline execution |

### Forge & Collaboration

| Command | Description |
|---------|-------------|
| `wave merge <PR>` | Merge a pull request using forge CLI with API fallback |

### Benchmarking

| Command | Description |
|---------|-------------|
| `wave bench run` | Run a pipeline against SWE-bench tasks |
| `wave bench compare` | Compare two benchmark result files |
| `wave bench report` | Generate a summary from results |
| `wave bench list` | List available benchmark datasets |

### Maintenance

| Command | Description |
|---------|-------------|
| `wave validate` | Validate Wave configuration |
| `wave clean` | Clean up project artifacts (`--older-than`, `--status`) |
| `wave cleanup` | Remove orphaned worktrees from `.agents/workspaces/` |
| `wave doctor` | Check project health and environment setup |
| `wave suggest` | Propose pipeline runs based on codebase state |
| `wave serve` | Start the web dashboard server |
| `wave migrate` | Database migration management |
| `wave skills` | Skill lifecycle management (legacy) |
| `wave skill` | Manage skill templates and install from remote sources |
| `wave agent` | Persona-to-agent compiler utilities |

### Configuration Management

| Command | Description |
|---------|-------------|
| `wave persona` | Create and manage personas (`create`, `list`) |
| `wave pipeline` | Create and manage pipelines (`create`, `list`) |

---

## Command Examples

### Running Pipelines

```bash
# Feature development
wave run impl-speckit "add OAuth2 with Google"

# Fix production bug
wave run impl-hotfix "500 errors on /api/users"

# Review PR
wave run ops-pr-review "review auth module changes"

# Generate tests
wave run test-gen "improve coverage for internal/cache"

# Quick fixes
wave do "fix typo in README"
wave do "add input validation to signup form"
wave do "review for SQL injection" --persona auditor
```

### Project Analysis

```bash
# Analyze project ontology and generate context skills
wave analyze

# AI-assisted deep analysis (extracts invariants, domain vocabulary)
wave analyze --deep

# Show orchestration decision provenance table
wave analyze --decisions

# Propose ontology updates from pipeline run history
wave analyze --evolve
```

### Benchmarking

```bash
# Run Wave pipeline against SWE-bench tasks
wave bench run --dataset swe-bench-lite.jsonl --pipeline bench-solve --limit 10

# Run standalone Claude as baseline
wave bench run --dataset swe-bench-lite.jsonl --mode claude --label baseline

# Compare results
wave bench compare --base baseline.json --compare wave-run.json
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
    #temperature: 0.3
    permissions:
      allowed_tools: [Read, Glob, Grep, "Bash(git log*)", "Bash(git status*)"]
      deny: ["Write(*)", "Edit(*)", "Bash(git commit*)", "Bash(git push*)"]
```

**31 built-in personas** (plus `base-protocol.md` shared preamble) including `navigator`, `craftsman`, `auditor`, `philosopher`, `debugger`, and more.

> Explore all personas in [`.agents/personas/`](.agents/personas/)

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

Steps without mutual dependencies run concurrently. Fan-out from a shared step, or start independent parallel tracks:

```yaml
steps:
  - id: analyze
    persona: navigator
  - id: security
    persona: auditor
    dependencies: [analyze]
  - id: quality
    persona: auditor
    dependencies: [analyze]
  - id: summary
    persona: summarizer
    dependencies: [security, quality]  # runs after both complete
```

**82 built-in pipelines** for development, debugging, documentation, and GitHub automation.

> Explore all pipelines in [`.agents/pipelines/`](.agents/pipelines/)

### Browser Automation

The built-in browser adapter provides headless Chrome automation via CDP for personas that need to scrape web pages, take screenshots, or interact with web UIs. Domain filtering enforces network restrictions at the browser level.

### Docker Sandbox

Alongside the existing Nix + bubblewrap sandbox, Wave supports a Docker-based sandbox backend for isolating adapter subprocesses in containers. Configured via `runtime.sandbox.backend: docker` in the manifest, with automatic Docker daemon detection during preflight.

### Web UI Dashboard

`wave serve` launches an embedded web operations dashboard for real-time pipeline monitoring, execution control, DAG visualization, and artifact browsing.

### Guided TUI Orchestrator

A Bubble Tea terminal UI provides interactive pipeline orchestration with an animated header, step progress tracking, and real-time agent activity display. Activate with default output mode or disable with `--no-tui`.

### Contracts — Validated Handoffs

Every step boundary validates output against JSON Schema, TypeScript interfaces, or test suites. Malformed artifacts trigger retries or halt the pipeline.

---

## Pipelines

A selection of the 82 built-in pipelines:

### Development

| Pipeline | Description |
|----------|-------------|
| `impl-speckit` | Specification-driven feature development |
| `impl-feature` | Feature planning and implementation |
| `impl-hotfix` | Quick investigation and fix for production issues |
| `impl-refactor` | Safe refactoring with comprehensive test coverage |
| `impl-prototype` | Prototype-driven development (spec → docs → dummy → implement → pr) |

### Quality & Debugging

| Pipeline | Description |
|----------|-------------|
| `ops-pr-review` | Comprehensive code review for pull requests |
| `test-gen` | Generate comprehensive test coverage |
| `ops-debug` | Systematic debugging with hypothesis testing |

### Planning & Documentation

| Pipeline | Description |
|----------|-------------|
| `plan-task` | Break down a feature into actionable tasks |
| `doc-fix` | Generate or update documentation |
| `audit-doc` | Documentation consistency gate |

### Issue Automation

| Pipeline | Description |
|----------|-------------|
| `impl-issue` | Implement a GitHub issue end-to-end |
| `plan-scope` | Decompose epics into child issues |
| `plan-research` | Research and report on issues |

### Benchmarking

| Pipeline | Description |
|----------|-------------|
| `bench-solve` | Solve a single SWE-bench task with minimal code changes |

> **More pipelines:** `ops-hello-world`, `test-smoke`, `doc-explain`, `doc-onboard`, `impl-improve`, `audit-dead-code`, `audit-security`, `doc-changelog`, `plan-adr`, `wave-land`, `impl-recinq`, `ops-supervise`, plus Wave self-evolution (wave-\*) pipelines
>
> Explore all in [`.agents/pipelines/`](.agents/pipelines/)

---

## Personas

A selection of the 30 built-in personas:

| Persona | Purpose | Key Permissions |
|---------|---------|-----------------|
| `navigator` | Codebase exploration | Read, Glob, Grep, git log/status |
| `philosopher` | Architecture & specs | Read, Write, Edit, Bash, Glob, Grep |
| `planner` | Task breakdown | Read, Write, Edit, Bash, Glob, Grep |
| `craftsman` | Implementation | Read, Write, Edit, Bash |
| `debugger` | Root cause analysis | Read, Grep, Glob, go test, git log/diff/bisect |
| `auditor` | Security review | Read, Grep, go vet, npm audit |
| `summarizer` | Context compaction | Read, Write, Edit, Bash, Glob, Grep |

> **More personas:** `implementer`, `researcher`, `reviewer`, `supervisor`, `validator`, `synthesizer`, `provocateur`, plus platform-specific personas for GitHub, GitLab, Gitea, and Bitbucket
>
> Explore all in [`.agents/personas/`](.agents/personas/)

---

## Project Structure

```
wave.yaml                    # Project manifest
.agents/
├── personas/                # System prompts
│   ├── navigator.md
│   ├── craftsman.md
│   └── ...
├── pipelines/               # Pipeline definitions
│   ├── impl-speckit.yaml
│   ├── impl-hotfix.yaml
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
      allowed_tools: [Read, Glob, Grep, "Bash(git log*)", "Bash(git status*)"]
      deny: ["Write(*)", "Edit(*)", "Bash(git commit*)", "Bash(git push*)"]
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

| Tool | Purpose | Required | Install |
|------|---------|----------|---------|
| `wave` | Pipeline orchestrator | Yes | [Installation Guide](docs/guide/installation.md) |
| `claude` | LLM adapter (Claude Code) | Yes* | `npm install -g @anthropic-ai/claude-code` |
| `gh` | GitHub CLI for issue/PR pipelines | Optional | [cli.github.com](https://cli.github.com) |
| `git` | Version control, worktree isolation | Yes | [git-scm.com](https://git-scm.com) |
| Go 1.25+ | Building from source | Optional | [go.dev](https://go.dev/dl/) |
| [Nix](https://nixos.org/download.html) | Sandboxed development shell | Optional | [nixos.org](https://nixos.org/download.html) |

\* At least one LLM CLI adapter is required. `claude` (Claude Code) is the default. Other adapters (`opencode`, `gemini`, `codex`) can be configured in `wave.yaml`.

Run `wave doctor` after installation to verify your environment is correctly configured. See the [Installation Guide](docs/guide/installation.md) for detailed setup instructions.

---

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on building, testing, commit conventions, and the PR workflow.

---

## License

[MIT](LICENSE)
