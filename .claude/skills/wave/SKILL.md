# Wave — Multi-Agent Pipeline Orchestrator

Wave is a Go CLI that orchestrates multi-step AI workflows. It wraps LLM CLIs (primarily Claude Code) as subprocesses, executing pipelines where each step is performed by a specialized persona with enforced permissions, workspace isolation, and contract validation.

## Core Concepts

### Manifest (`wave.yaml`)

The manifest is the single source of truth for a Wave project. It defines adapters, personas, runtime config, and skill mounts.

```yaml
apiVersion: v1
kind: WaveManifest
metadata:
  name: my-project
  description: "Project description"
  repo: "github.com/org/repo"          # optional

adapters:
  claude:
    binary: claude                       # CLI binary name
    mode: headless                       # execution mode
    output_format: json                  # stream-json output
    project_files: [CLAUDE.md, .claude/settings.json]
    default_permissions:
      allowed_tools: [Read, Write, Edit, Bash]
      deny: []
    hooks_template: ""                   # optional hooks template

personas:
  navigator:
    adapter: claude
    description: "Read-only codebase exploration"
    system_prompt_file: .wave/personas/navigator.md
    temperature: 0.1
    model: sonnet                        # optional: opus, sonnet, haiku
    permissions:
      allowed_tools: [Read, Glob, Grep, "Bash(git log*)", "Bash(git status*)"]
      deny: ["Write(*)", "Edit(*)", "Bash(git commit*)", "Bash(git push*)"]
    hooks:                               # optional Claude Code hooks
      PreToolUse:
        - matcher: "Bash"
          command: "validate-bash.sh"
    sandbox:                             # optional per-persona sandbox
      allowed_domains: [api.github.com]

runtime:
  workspace_root: .wave/workspaces
  max_concurrent_workers: 5
  default_timeout_minutes: 30
  relay:
    token_threshold_percent: 80
    strategy: summarize_to_checkpoint    # context compaction strategy
    context_window: 200000               # optional
    summarizer_persona: summarizer       # persona for compaction
  audit:
    log_dir: .wave/traces/
    log_all_tool_calls: true
    log_all_file_operations: false
  meta_pipeline:
    max_depth: 2                         # max nested meta-pipeline depth
    max_total_steps: 20
    max_total_tokens: 500000
    timeout_minutes: 60
  routing:                               # optional pipeline routing
    default: prototype
    rules:
      - pattern: "fix*"
        pipeline: hotfix
        priority: 10
      - match_labels: {severity: critical}
        pipeline: hotfix
        priority: 20
  sandbox:
    enabled: true
    default_allowed_domains: [api.github.com]
    env_passthrough: [ANTHROPIC_API_KEY, GITHUB_TOKEN]

skill_mounts:
  - path: .wave/skills/
```

### Pipelines (`.wave/pipelines/<name>.yaml`)

Pipelines define multi-step workflows with dependency resolution, artifact chaining, and contract validation.

```yaml
kind: WavePipeline
metadata:
  name: code-review
  description: "Comprehensive code review"
  release: true       # included in `wave init` (without --all)
  disabled: false      # can be disabled

input:
  source: cli          # input source: cli
  label_filter: ""     # optional label filter
  batch_size: 1        # optional batch size

steps:
  - id: analyze
    persona: navigator
    dependencies: []                      # DAG dependencies (step IDs)
    memory:
      strategy: fresh                     # fresh = no history inheritance
      inject_artifacts:                   # artifacts from prior steps
        - step: prior-step
          artifact: artifact-name
          as: local-filename              # available at artifacts/<as>
    workspace:
      root: ./                            # workspace root override
      mount:
        - source: ./                      # host path
          target: /src                    # mount point in workspace
          mode: readonly                  # readonly | readwrite
    exec:
      type: prompt                        # execution type
      source: |                           # inline prompt (supports {{ input }})
        Analyze the code for: {{ input }}
      source_path: .wave/prompts/analyze.md  # OR external prompt file
    output_artifacts:
      - name: analysis
        path: output/analysis.json        # relative to workspace
        type: json                        # json, markdown, text, code, etc.
        required: false                   # optional artifact
    handover:
      contract:
        type: json_schema                 # json_schema or command
        schema_path: .wave/contracts/analysis.schema.json
        source: output/analysis.json      # file to validate
        must_pass: true
        on_failure: retry                 # retry | fail | skip
        max_retries: 2
        command: "go test ./..."          # for command-type contracts
        dir: project_root                 # working dir for command contracts
      compaction:
        trigger: token_threshold
        persona: summarizer
      on_review_fail: retry
      target_step: analyze                # step to retry on failure
      max_retries: 2
    strategy:                             # optional matrix execution
      type: fan_out
      items_source: artifact_json_array
      item_key: items
      max_concurrency: 3
    validation:                           # optional validation rules
      - type: file_exists
        target: output/analysis.json
        on_failure: retry
        max_retries: 1
```

### Personas

Personas are named AI agent configurations with specific roles, permissions, and system prompts. Each persona references an adapter and has permission allow/deny lists using glob patterns.

**Built-in personas:**
| Persona | Role | Key Permissions |
|---------|------|----------------|
| `navigator` | Read-only codebase exploration | Read, Glob, Grep, git log/status |
| `philosopher` | Architecture & specification | Read, Write(.wave/specs/*) |
| `craftsman` | Code implementation & testing | Read, Write, Edit, Bash |
| `auditor` | Security review & QA | Read, Grep, go vet, npm audit |
| `summarizer` | Context compaction for relay | Read only |
| `planner` | Task breakdown & planning | Read, Glob, Grep |
| `debugger` | Systematic issue diagnosis | Read, Grep, Glob, git log/bisect |
| `implementer` | Full execution specialist | Read, Write, Edit, Bash, Glob, Grep |
| `reviewer` | Quality review & validation | Read, Glob, Grep, go test |
| `researcher` | Web research & synthesis | Read, Glob, Grep, WebSearch, WebFetch |
| `github-analyst` | GitHub issue scanning | Read, Write, Bash(gh) |
| `github-enhancer` | GitHub issue improvement | Read, Write, Bash(gh) |
| `github-commenter` | Post GitHub issue comments | Read, Bash(gh issue comment) |

**Permission syntax:** Tool names with optional glob patterns:
- `Read` — allow all reads
- `Write(.wave/specs/*)` — allow writes only under .wave/specs/
- `Bash(git log*)` — allow only git log commands
- `Bash(*)` in deny — deny all bash

### Contracts (`.wave/contracts/<name>.schema.json`)

Contracts validate step output before allowing handover to the next step. They are JSON Schema files that validate `artifact.json` or other output files.

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["files", "patterns", "impact_areas"],
  "properties": {
    "files": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["path", "purpose"],
        "properties": {
          "path": {"type": "string"},
          "purpose": {"type": "string"}
        }
      },
      "minItems": 1
    }
  }
}
```

Contract types:
- **`json_schema`** — Validates output JSON against a JSON Schema file
- **`command`** — Runs a shell command (e.g., `go test ./...`) and checks exit code

### Artifacts

Artifacts are files produced by pipeline steps that can be injected into subsequent steps.

**Output artifacts** — declared per-step, written to the step's workspace:
```yaml
output_artifacts:
  - name: analysis        # referenced by downstream steps
    path: output/data.json
    type: json
```

**Inject artifacts** — pull output from prior steps into the current step's `artifacts/` directory:
```yaml
memory:
  inject_artifacts:
    - step: analyze        # source step ID
      artifact: analysis   # artifact name from that step
      as: analysis_data    # available at artifacts/analysis_data
```

### Workspaces

Each step runs in an isolated workspace at `.wave/workspaces/<pipeline>/<step>/`. Workspaces support:
- **Mounts** — bind host directories into the workspace (readonly or readwrite)
- **Root override** — `workspace.root: ./` creates an empty directory (not the project root)
- **Artifact injection** — prior step outputs are copied into `artifacts/`

### State Management

Wave uses SQLite (`.wave/state.db`) for persistence:
- Pipeline runs with status tracking (pending, running, completed, failed, cancelled)
- Step states with retry counts and error messages
- Event log for audit trail
- Artifact records with paths and sizes

### Security Model

- **Permission enforcement** — Persona allow/deny lists projected into adapter settings
- **Workspace isolation** — Each step gets its own ephemeral directory
- **Sandbox** — Optional bubblewrap sandboxing with network domain allowlisting
- **Fresh memory** — No chat history inheritance between steps
- **Env passthrough** — Only explicitly listed env vars reach subprocesses
- **Audit logging** — All tool calls and file operations logged to `.wave/traces/`
- **Path sanitization** — Input validated against path traversal attacks

### Relay / Compaction

When context windows fill up, the relay system compacts context using a summarizer persona. Configured via `runtime.relay`:
- `token_threshold_percent` — trigger compaction at this % of context window
- `strategy` — `summarize_to_checkpoint` compacts prior context into a summary
- `summarizer_persona` — which persona handles compaction

## CLI Reference

### Global Flags
```
-m, --manifest string   Path to manifest file (default "wave.yaml")
-d, --debug             Enable debug mode
-o, --output string     Output format: auto, json, text, quiet (default "auto")
-v, --verbose           Include real-time tool activity
```

### Commands

#### `wave init`
Initialize a new Wave project with default configuration.
```
--force         Overwrite existing files
--merge         Merge defaults into existing configuration
--all           Include all pipelines (not just release-gated ones)
--adapter       Default adapter name (default "claude")
--workspace     Workspace directory path (default ".wave/workspaces")
--output        Output path for wave.yaml (default "wave.yaml")
-y, --yes       Skip confirmation prompts
```

Creates: `wave.yaml`, `.wave/personas/`, `.wave/pipelines/`, `.wave/contracts/`, `.wave/prompts/`, `.wave/workspaces/`, `.wave/traces/`

#### `wave run [pipeline] [input]`
Execute a pipeline.
```
wave run code-review "Review auth module"
wave run --pipeline speckit-flow --input "add user auth"
wave run hotfix --dry-run
wave run migrate --from-step validate --force

--pipeline      Pipeline name
--input         Input data for the pipeline
--dry-run       Show execution plan without running
--from-step     Resume from a specific step
--force         Skip validation when using --from-step
--timeout       Timeout in minutes (overrides manifest)
--mock          Use mock adapter for testing
```

#### `wave do [task]`
Execute an ad-hoc task (generates a navigate-then-execute pipeline).
```
wave do "fix the login bug"
wave do "refactor database queries" --persona craftsman

--persona       Override execute persona (default: craftsman)
--dry-run       Preview the generated pipeline
--mock          Use mock adapter
```

#### `wave meta [task]`
Generate and execute a custom multi-step pipeline dynamically using the philosopher persona.
```
wave meta "implement user authentication"
wave meta "add caching layer" --dry-run
wave meta "add monitoring" --save monitoring-pipeline.yaml

--save          Save generated pipeline YAML to path
--dry-run       Show generated pipeline without executing
--mock          Use mock adapter
```

#### `wave list [category]`
List Wave configuration and resources. Categories: `adapters`, `runs`, `pipelines`, `personas`, `contracts`. No argument lists all.
```
wave list pipelines
wave list runs --limit 20 --run-status completed
wave list --format json

--format        Output format: table, json
--limit         Max runs to show (default 10, for 'list runs')
--run-pipeline  Filter runs to specific pipeline
--run-status    Filter runs by status
```

#### `wave status [run-id]`
Show pipeline status.
```
wave status                          # running pipelines
wave status --all                    # recent pipelines
wave status debug-20260202-143022    # specific run
wave status --format json

--all           Show all recent pipelines
--format        Output format: table, json
```

#### `wave logs [run-id]`
Show pipeline logs with filtering and follow mode.
```
wave logs                    # most recent run
wave logs --step investigate # filter by step
wave logs --errors           # errors only
wave logs --tail 20          # last 20 entries
wave logs --since 10m        # last 10 minutes
wave logs --follow           # stream real-time
wave logs --format json

--step      Filter by step ID
--errors    Only show errors
--follow    Stream logs in real-time
--tail N    Show last N entries
--since     Time filter (e.g., "10m", "1h")
--level     Log level: all, info, error
--format    Output format: text, json
```

#### `wave artifacts [run-id]`
List and export pipeline artifacts.
```
wave artifacts                    # most recent run
wave artifacts --step analyze     # filter by step
wave artifacts --export ./out     # export to directory
wave artifacts --format json

--step      Filter to specific step
--export    Export artifacts to directory
--format    Output format: table, json
```

#### `wave cancel [run-id]`
Cancel a running pipeline.
```
wave cancel                  # cancel most recent
wave cancel abc123           # cancel specific run
wave cancel --force          # SIGTERM then SIGKILL
wave cancel --format json

-f, --force    Interrupt immediately
--format       Output format: text, json
```

#### `wave validate`
Validate manifest and pipeline configuration.
```
wave validate
wave validate --pipeline code-review

--pipeline    Specific pipeline to validate
```

#### `wave clean`
Remove workspaces, state, and cached artifacts.
```
wave clean --all                         # remove everything
wave clean --pipeline code-review        # specific pipeline
wave clean --all --keep-last 5           # keep 5 most recent
wave clean --older-than 7d               # age-based cleanup
wave clean --status completed            # status-based cleanup
wave clean --dry-run                     # preview what would be deleted

--all           Clean all workspaces and state
--pipeline      Clean specific pipeline workspace
--force         Skip confirmation
--keep-last N   Keep N most recent workspaces
--dry-run       Preview without deleting
--older-than    Duration filter (e.g., "7d", "24h")
--status        Filter by status (completed, failed, etc.)
--quiet         Suppress output
```

#### `wave migrate [subcommand]`
Database migration management.
```
wave migrate status           # show migration status
wave migrate up [version]     # apply pending migrations
wave migrate down <version>   # rollback to version (with confirmation)
wave migrate validate         # verify migration integrity
```

## Project Structure

```
wave.yaml                    # Main manifest
.wave/
  personas/                  # System prompt markdown files
    navigator.md
    craftsman.md
    ...
  pipelines/                 # Pipeline YAML definitions
    code-review.yaml
    hello-world.yaml
    prototype.yaml
    ...
  contracts/                 # JSON Schema validation files
    navigation.schema.json
    smoke-test.schema.json
    ...
  prompts/                   # External prompt templates
  workspaces/                # Ephemeral step workspaces
    <pipeline>/<step>/
  traces/                    # Audit logs
  state.db                   # SQLite state database
  skills/                    # Mounted skills
```

## Built-in Pipelines

These pipelines are included by default with `wave init` (release-gated with `release: true`). Use `wave init --all` to include all available pipelines.

| Pipeline | Steps | Description |
|----------|-------|-------------|
| `github-issue-enhancer` | scan-issues, plan-enhancements, apply-enhancements, verify-enhancements | Analyze and enhance poorly documented GitHub issues |
| `issue-research` | fetch-issue, analyze-topics, research-topics, synthesize-report, post-comment | Research a GitHub issue and post findings as a comment |
| `doc-loop` | scan-changes, analyze-consistency, create-issue | Pre-PR documentation consistency gate — creates GitHub issue with inconsistencies |

## Key Patterns

### Template Variables in Prompts
- `{{ input }}` — the CLI input string
- `{{ timestamp }}` — current timestamp

### Parallel Step Execution
Steps with independent dependencies run concurrently (up to `max_concurrent_workers`). For example, `security-review` and `quality-review` both depend only on `diff-analysis`, so they run in parallel.

### Contract Retry
When a contract fails validation, the step can be retried (up to `max_retries`) with the validation error fed back to the persona.

### Resume / From-Step
`wave run <pipeline> --from-step <step>` creates a subpipeline starting at the given step, injecting artifacts from prior completed steps found in existing workspaces.

### Output Formats
- `auto` — BubbleTea TUI when connected to TTY, plain text when piped
- `json` — NDJSON events to stdout (for scripting)
- `text` — Plain text progress to stderr
- `quiet` — Only final result

## Development

```bash
go test ./...              # run all tests
go test -race ./...        # with race detector (required for PR)
go test -v ./internal/pipeline/...  # specific package
go run ./cmd/wave run hello-world "test"  # run from source
```

### Testing with Mock Adapter
```bash
wave run hello-world "test" --mock    # uses MockAdapter with simulated delay
```

### Environment Variables
```
WAVE_MIGRATION_ENABLED=true     # enable migration system
WAVE_AUTO_MIGRATE=true          # auto-apply on startup
WAVE_MAX_MIGRATION_VERSION=N    # limit migrations for gradual rollout
WAVE_SKIP_MIGRATION_VALIDATION=true  # skip checksums (dev only)
```
