# Wave: Multi-Agent Pipeline Orchestrator

Wave coordinates multiple AI personas through structured pipelines for AI-assisted development workflows, enforcing permissions, contracts, and workspace isolation at every step.

**Note**: Wave requires AI adapter binaries to be installed and available in PATH. Currently supports `claude`, `opencode`, and `dummy` adapters.

## Quick Start

### 1. Build and Install

```bash
# Build the CLI
go build -o wave ./cmd/wave

# (Optional) Install to PATH
sudo mv wave /usr/local/bin/
```

### 2. Set Up Adapter (Required)

**Option A: Use Real AI Adapter**
```bash
# Install Claude CLI or OpenCode CLI to PATH
# Then use:
wave init --adapter claude
```

**Option B: Create Dummy Adapter for Testing**
```bash
# Create dummy adapter script
echo '#!/bin/bash
echo "Dummy adapter response for: $*"' > /usr/local/bin/dummy
chmod +x /usr/local/bin/dummy

# Initialize with dummy adapter
wave init --adapter dummy
```

### 3. Initialize Project

```bash
cd your-project
wave init --adapter <claude|opencode|dummy>
```

This creates:
- `wave.yaml` - Project manifest with adapters and personas
- `.wave/personas/` - AI persona system prompts
- `.wave/pipelines/` - Example pipeline definitions

### 4. Configure Personas

The dummy adapter initialization creates a minimal manifest. You need to add required personas:

```yaml
# wave.yaml
apiVersion: v1
kind: Wave
metadata:
  name: my-project
  description: "My Wave project"

adapters:
  dummy:
    binary: dummy
    mode: agent
    outputFormat: markdown

personas:
  navigator:
    adapter: dummy
    systemPromptFile: .wave/personas/navigator.md
    temperature: 0.1
    description: "Codebase navigator and analyzer"
  
  craftsman:
    adapter: dummy
    systemPromptFile: .wave/personas/craftsman.md
    temperature: 0.7
    description: "Implementation specialist"

runtime:
  workspaceRoot: ./workspace
  defaultTimeoutMin: 10
  maxConcurrentWorkers: 4
```

Create the persona files:
```bash
mkdir -p .wave/personas
echo "You are a code navigator. Analyze and understand codebases." > .wave/personas/navigator.md
echo "You are a craftsman developer. Implement high-quality code." > .wave/personas/craftsman.md
```

### 5. Validate Configuration

```bash
wave validate
```

### 6. Run Your First Pipeline

```bash
# Quick task without creating a pipeline file
wave do "fix the typo in README.md line 42"

# Or run a full pipeline (requires pipeline file)
wave run --pipeline speckit-flow --input "add user authentication"
```

## Command Reference

### `wave init`

Initialize a new Wave project with default configuration.

```bash
wave init [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--adapter` | `claude` | Default adapter to use (`claude`, `opencode`, `dummy`) |
| `--force` | `false` | Overwrite existing files |
| `--workspace` | `.wave/workspaces` | Workspace directory path |
| `--output` | `wave.yaml` | Output path for manifest file |

**Examples:**
```bash
wave init                           # Initialize with Claude adapter
wave init --adapter opencode        # Use OpenCode adapter
wave init --force                   # Overwrite existing configuration
```

**Creates:**
- `wave.yaml` — Project manifest with adapters and personas
- `.wave/personas/` — AI persona system prompts (navigator, philosopher, craftsman, auditor, summarizer)
- `.wave/pipelines/` — Example pipelines (speckit-flow, hotfix)
- `.wave/contracts/` — JSON schema contracts for validation
- `.wave/workspaces/` — Ephemeral workspace root
- `.wave/traces/` — Audit log directory

---

### `wave do`

Execute an ad-hoc task with a minimal navigate→execute pipeline.

```bash
wave do <task description> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--persona` | `craftsman` | Override execute persona |
| `--save` | — | Save generated pipeline YAML to path |
| `--manifest` | `wave.yaml` | Path to manifest file |
| `--mock` | `false` | Use mock adapter (for testing) |
| `--dry-run` | `false` | Show execution plan without running |

**Examples:**
```bash
wave do "fix the typo in README.md"
wave do "audit auth middleware for SQL injection" --persona auditor
wave do "add dark mode toggle" --save dark-mode.yaml
wave do "refactor user service" --dry-run
```

---

### `wave run`

Execute a defined pipeline.

```bash
wave run --pipeline <name> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--pipeline` | — | Pipeline name to run (required) |
| `--input` | — | Input data for the pipeline |
| `--dry-run` | `false` | Show execution plan without running |
| `--from-step` | — | Start execution from specific step |
| `--timeout` | manifest default | Timeout in minutes |
| `--manifest` | `wave.yaml` | Path to manifest file |
| `--mock` | `false` | Use mock adapter (for testing) |

**Examples:**
```bash
wave run --pipeline speckit-flow --input "add user authentication"
wave run --pipeline hotfix --input "fix memory leak in cache.go"
wave run --pipeline speckit-flow --dry-run
wave run --pipeline speckit-flow --from-step implement
wave run --pipeline speckit-flow --timeout 60
```

---

### `wave list`

List available pipelines, personas, and adapters.

```bash
wave list [pipelines|personas|adapters] [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--manifest` | `wave.yaml` | Path to manifest file |
| `--format` | `table` | Output format (`table`, `json`) |

**Examples:**
```bash
wave list                   # List everything
wave list pipelines         # List available pipelines
wave list personas          # List configured personas
wave list adapters          # List configured adapters
```

---

### `wave resume`

Resume a paused or failed pipeline execution.

```bash
wave resume --pipeline <id> [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--pipeline` | — | Pipeline ID to resume (required) |
| `--from-step` | — | Resume from specific step |
| `--manifest` | `wave.yaml` | Path to manifest file |

**Examples:**
```bash
wave resume --pipeline abc123
wave resume --pipeline abc123 --from-step implement
```

---

### `wave clean`

Remove generated workspaces, state files, and cached artifacts.

```bash
wave clean [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--all` | `false` | Clean all workspaces and state |
| `--pipeline` | — | Clean specific pipeline workspace |
| `--force` | `false` | Skip confirmation |

**Examples:**
```bash
wave clean --all                    # Remove all workspaces and state
wave clean --pipeline speckit-flow  # Clean specific pipeline
wave clean --all --force            # Clean without confirmation
```

**Removes:**
- `.wave/state.db` — Pipeline state database
- `.wave/traces/` — Audit logs
- `.wave/workspaces/` — All ephemeral workspaces

---

### `wave validate`

Validate Wave configuration and project structure.

```bash
wave validate [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--manifest` | `wave.yaml` | Path to manifest file |
| `--pipeline` | — | Specific pipeline to validate |
| `-v, --verbose` | `false` | Verbose output |

**Examples:**
```bash
wave validate                              # Validate everything
wave validate --verbose                    # Show detailed validation steps
wave validate --pipeline speckit-flow      # Validate specific pipeline
```

**Checks:**
- Manifest syntax and structure
- Required fields (apiVersion, kind, metadata.name)
- Adapter binary availability in PATH
- Persona system prompt file existence
- Pipeline step dependencies and persona references

## Configuration

### Manifest (wave.yaml)

```yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: my-project
  description: "Example project using Wave"

adapters:
  claude:
    binary: claude
    mode: headless
    output_format: json

personas:
  navigator:
    adapter: claude
    system_prompt_file: .wave/personas/navigator.md
    temperature: 0.1
    permissions:
      allowed_tools: ["Read", "Bash(git *)"]
      deny: ["Write(*)"]
  
  craftsman:
    adapter: claude
    system_prompt_file: .wave/personas/craftsman.md
    temperature: 0.7
    permissions:
      allowed: ["*"]

runtime:
  workspace_root: ./workspace
  max_concurrent_workers: 3
  default_timeout_minutes: 30
```

### Pipeline Example (.wave/pipelines/speckit-flow.yaml)

```yaml
kind: WavePipeline
metadata:
  name: speckit-flow
  description: "Specification-driven development workflow"

steps:
  - id: navigate
    persona: navigator
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: "Analyze the codebase for: {{ input }}"
  
  - id: specify
    persona: philosopher
    dependencies: [navigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: navigate
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: "Create detailed specifications for: {{ input }} using context from navigation analysis"
  
  - id: implement
    persona: craftsman
    dependencies: [specify]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: specify
          artifact: specification
          as: spec
    exec:
      type: prompt
      source: "Implement the feature according to the specification"
```

## Output and Events

Wave emits structured NDJSON events during pipeline execution:

```json
{"timestamp":"2026-02-01T10:00:00Z","pipeline_id":"123","step_id":"navigate","state":"running","duration_ms":0}
{"timestamp":"2026-02-01T10:01:30Z","pipeline_id":"123","step_id":"navigate","state":"completed","duration_ms":90000}
{"timestamp":"2026-02-01T10:01:31Z","pipeline_id":"123","step_id":"specify","state":"running","duration_ms":0}
```

Pipe events to `jq` for filtering:
```bash
wave run --pipeline flow.yaml --input "task" | jq 'select(.state == "completed")'
```

## Workspace Structure

Each pipeline step gets an isolated workspace:

```
./workspace/<pipeline-id>/<step-id>/
├── artifacts/           # Step outputs
│   ├── analysis.json   # Navigation results
│   ├── spec.md         # Generated specifications
│   └── implementation/ # Code changes
├── workspace/          # Working directory for the step
└── state.db           # SQLite state for resumption
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General Error |
| 2 | Usage Error |
| 3 | Pipeline Failed |
| 4 | Validation Error |
| 5 | Timeout |
| 130 | Interrupted (Ctrl+C) - state persisted for resumption |

## Advanced Features

### Shell Completion

```bash
# Bash
wave completion bash > /etc/bash_completion.d/wave

# Zsh  
wave completion zsh > "${fpath[1]}/_wave"

# Fish
wave completion fish > ~/.config/fish/completions/wave.fish
```

### Pipeline Features

- **DAG-based execution** with dependency resolution
- **Workspace isolation** for each step
- **State persistence** for resumption
- **Memory strategies** (fresh, cumulative, checkpoint)
- **Artifact injection** between steps
- **Permission enforcement** per persona
- **Contract validation** for structured outputs

### Development Commands

```bash
# Build locally
go build -o wave ./cmd/wave

# Run tests
go test ./...

# Lint (if configured)
golangci-lint run
```

## Examples

### Quick Bug Fix
```bash
wave do "fix the memory leak in user_service.go"
```

### Feature Development
```bash
wave run --pipeline speckit-flow --input "add OAuth2 authentication with Google provider"
```

### Code Review
```bash
wave do "review the recent authentication changes for security issues" --persona auditor
```

### Cleanup
```bash
# Clean old workspaces
wave clean --older-than 7d

# Clean specific failed pipeline
wave clean --pipeline-id a1b2c3d4
```

## Architecture

Wave implements a **pipeline orchestrator** pattern:

1. **Manifests** define available adapters and personas
2. **Pipelines** define multi-step workflows as DAGs  
3. **Steps** execute single persona tasks with isolated workspaces
4. **Events** provide real-time progress tracking
5. **State** enables resumption and debugging

The system enforces **least privilege** through persona-specific permissions and **auditability** through structured logging and artifact preservation.

For detailed documentation, see the `/docs` directory or run `wave <command> --help`.