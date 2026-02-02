# CLI Reference

Complete reference for all Wave CLI commands, flags, and output behavior.

## Synopsis

```
  ╦ ╦╔═╗╦  ╦╔═╗
  ║║║╠═╣╚╗╔╝║╣
  ╚╩╝╩ ╩ ╚╝ ╚═╝
  Multi-Agent Pipeline Orchestrator

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
  resume      Resume a paused pipeline
  run         Run a pipeline
  status      Show pipeline status
  validate    Validate Wave configuration
```

## Global Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--help` | `-h` | — | — | Show help for any command. |
| `--version` | `-v` | — | — | Print Wave version and exit. |
| `--manifest` | `-m` | `string` | `wave.yaml` | Path to manifest file. |
| `--debug` | `-d` | `bool` | `false` | Enable debug logging to stderr. |
| `--log-format` | | `string` | `text` | Output format: `json` or `text`. |

---

## `wave init`

Initialize a new Wave project in the current directory.

```bash
wave init [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--adapter` | `string` | `"claude"` | Default adapter to configure. |
| `--workspace` | `string` | `".wave/workspaces"` | Workspace root directory. |
| `--output` | `string` | `"wave.yaml"` | Output path for wave.yaml. |
| `--force` | `bool` | `false` | Overwrite existing files without prompting. |
| `--merge` | `bool` | `false` | Merge defaults into existing configuration. |
| `-y, --yes` | `bool` | `false` | Answer yes to all confirmation prompts. |

### Creates

```
wave.yaml                          # Project manifest
.wave/
├── personas/
│   ├── navigator.md               # Navigator system prompt
│   ├── craftsman.md               # Craftsman system prompt
│   └── summarizer.md              # Summarizer system prompt
├── pipelines/
│   └── default.yaml               # Example pipeline
├── hooks/                         # Hook script directory
└── contracts/                     # Contract schema directory
```

### Examples

```bash
# Initialize with defaults
wave init

# Initialize with specific adapter
wave init --adapter opencode

# Initialize with custom workspace
wave init --workspace ./workspaces

# Force re-initialization
wave init --force

# Merge defaults into existing config
wave init --merge
```

---

## `wave validate`

Validate manifest and pipeline configuration files.

```bash
wave validate [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--manifest` | `string` | `"wave.yaml"` | Path to manifest file. |
| `--pipeline` | `string` | `""` | Validate a specific pipeline file. If omitted, validates all discovered pipelines. |
| `--verbose` | `bool` | `false` | Show detailed validation output including passing checks. |

### Validation Checks

| Check | Severity | Description |
|-------|----------|-------------|
| YAML syntax | **error** | Manifest and pipeline files must be valid YAML. |
| Required fields | **error** | `apiVersion`, `kind`, `metadata.name`, `adapters`, `personas`, `runtime` must be present. |
| Adapter references | **error** | Every persona's `adapter` field must reference a defined adapter. |
| System prompt files | **error** | Every `system_prompt_file` path must exist on disk. |
| Hook scripts | **error** | Every hook `command` referencing a file must exist on disk. |
| DAG validity | **error** | Pipeline step dependencies must form a valid DAG (no cycles). |
| Step persona refs | **error** | Every step's `persona` must reference a defined persona. |
| Dependency refs | **error** | Every `dependencies` entry must reference a valid step ID. |
| Binary on PATH | **warning** | Adapter `binary` should be resolvable on `$PATH`. |
| Value ranges | **error** | Numeric fields must be within valid ranges. |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All validation passed. |
| `1` | Validation errors found. |
| `2` | Manifest file not found. |

### Examples

```bash
# Validate everything
wave validate

# Validate with verbose output
wave validate --verbose

# Validate specific manifest
wave validate --manifest staging.wave.yaml

# Validate a single pipeline
wave validate --pipeline .wave/pipelines/deploy.yaml
```

---

## `wave run`

Execute a pipeline.

```bash
wave run --pipeline <name> [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--pipeline` | `string` | **required** | Pipeline name to run. |
| `--input` | `string` | `""` | Input prompt for the pipeline. |
| `--dry-run` | `bool` | `false` | Show what would be executed without running. |
| `--from-step` | `string` | `""` | Start execution from a specific step. |
| `--manifest` | `string` | `"wave.yaml"` | Path to manifest file. |
| `--timeout` | `int` | manifest default | Override per-step timeout (minutes). |
| `--mock` | `bool` | `false` | Use mock adapter (for testing). |

### Output

Emits [NDJSON events](/reference/events) to stdout on every state transition. Errors and debug info go to stderr.

### Examples

```bash
# Run a pipeline with input
wave run --pipeline speckit-flow --input "add user authentication with JWT"

# Dry run to preview execution plan
wave run --pipeline speckit-flow --dry-run

# Resume from a specific step
wave run --pipeline speckit-flow --from-step implement

# With custom timeout
wave run --pipeline speckit-flow --input "refactor database layer" --timeout 60
```

---

## `wave do`

Execute an ad-hoc task. Generates and runs a minimal 2-step pipeline (navigate → execute) without requiring a pipeline file.

```bash
wave do "<task description>" [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--persona` | `string` | `"craftsman"` | Persona for the execution step. |
| `--meta` | `bool` | `false` | Generate pipeline dynamically using philosopher persona. |
| `--save` | `string` | `""` | Save the generated pipeline YAML to this path. |
| `--manifest` | `string` | `"wave.yaml"` | Path to manifest file. |
| `--dry-run` | `bool` | `false` | Show what would be executed without running. |
| `--mock` | `bool` | `false` | Use mock adapter (for testing). |

### Standard Mode (Default)

By default, `wave do` generates a fixed 2-step pipeline:
1. **navigate** — The navigator persona explores the codebase to understand context.
2. **execute** — The craftsman (or specified `--persona`) implements the task.

### Meta-Pipeline Mode (`--meta`)

When `--meta` is specified, Wave uses a dynamic pipeline generation approach:

1. The **philosopher** persona analyzes the task and designs a custom multi-step pipeline tailored to the specific requirements.
2. The generated pipeline is then executed automatically.

This mode is useful for complex tasks that benefit from a thoughtfully designed workflow rather than the fixed navigate→execute pattern.

**Requirements:**
- The `philosopher` persona must be configured in your manifest.
- Meta-pipeline configuration is read from the `runtime.meta_pipeline` section of the manifest.

### Examples

```bash
# Quick fix (standard mode)
wave do "fix the typo in README.md line 42"

# With specific persona
wave do "audit authentication middleware for SQL injection" --persona auditor

# Save generated pipeline for inspection
wave do "add dark mode toggle" --save .wave/pipelines/dark-mode.yaml

# Preview without executing
wave do "refactor user service" --dry-run

# Meta-pipeline: philosopher designs custom pipeline
wave do --meta "implement user authentication system"

# Preview meta-generated pipeline without executing
wave do --meta --dry-run "build REST API"

# Save meta-generated pipeline for inspection
wave do --meta --save my-pipeline --dry-run "refactor module"
```

---

## `wave resume`

Resume an interrupted or failed pipeline from its last checkpoint.

```bash
wave resume [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--pipeline` | `string` | `""` | Pipeline ID to resume. If omitted, lists recent pipelines. |
| `--from-step` | `string` | `""` | Override resume point. |
| `--manifest` | `string` | `"wave.yaml"` | Path to manifest file. |

### Resume Behavior

1. Loads persisted pipeline state from SQLite.
2. Identifies the last completed step.
3. Resumes execution from the next pending or failed step.
4. Completed steps are **skipped** — their artifacts remain in the workspace.
5. Failed steps are **retried** from scratch.

### Examples

```bash
# List resumable pipelines
wave resume

# Resume specific pipeline
wave resume --pipeline debug-20260202-143022

# Resume from a specific step
wave resume --pipeline debug-20260202-143022 --from-step implement
```

---

## `wave status`

Show the status of pipeline runs.

```bash
wave status [run-id] [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | `bool` | `false` | Show all recent pipelines (default 10). |
| `--format` | `string` | `"table"` | Output format: `table`, `json`. |
| `--manifest` | `string` | `"wave.yaml"` | Path to manifest file. |

### Behavior

- **No arguments**: Shows currently running pipelines.
- **With `--all`**: Shows recent pipelines (default 10).
- **With run-id**: Shows detailed status for that specific run.

### Examples

```bash
# Show running pipelines
wave status

# Show all recent pipelines
wave status --all

# Show specific run details
wave status debug-20260202-143022

# Output as JSON for scripting
wave status --format json
```

### Output

```
$ wave status
RUN_ID                     PIPELINE        STATUS       STEP            ELAPSED    TOKENS
debug-20260202-143022      debug           running      investigate     2m15s      12k

$ wave status debug-20260202-143022
Run ID:     debug-20260202-143022
Pipeline:   debug
Status:     running
Step:       investigate
Started:    2026-02-02 14:30:22
Elapsed:    2m15s
Tokens:     12k
Input:      memory leak after 1000 requests
```

---

## `wave logs`

Show logs from pipeline runs.

```bash
wave logs [run-id] [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--step` | `string` | `""` | Filter by step ID. |
| `--errors` | `bool` | `false` | Only show errors (alias for `--level error`). |
| `--level` | `string` | `"all"` | Log level: `all`, `info`, `error`. |
| `--tail` | `int` | `0` | Show last N lines. |
| `--since` | `string` | `""` | Filter by time (e.g., `10m`, `1h`, `7d`). |
| `--follow` | `bool` | `false` | Stream logs in real-time. |
| `--format` | `string` | `"text"` | Output format: `text`, `json`. |
| `--manifest` | `string` | `"wave.yaml"` | Path to manifest file. |

### Behavior

- **No arguments**: Shows logs from the most recent run.
- **With run-id**: Shows logs for that specific run.
- **With `--follow`**: Streams logs in real-time until the run completes or Ctrl+C.

### Examples

```bash
# Show logs from most recent run
wave logs

# Show logs for specific run
wave logs debug-20260202-143022

# Filter by step ID
wave logs --step investigate

# Show only errors
wave logs --errors

# Show last 20 log entries
wave logs --tail 20

# Show logs from last 10 minutes
wave logs --since 10m

# Stream logs in real-time
wave logs --follow

# Output as JSON for scripting
wave logs --format json
```

### Output

```
$ wave logs --tail 5
[14:30:22] started   navigate     (navigator)                 Starting navigation
[14:30:45] completed navigate     (navigator)   23s    2.1k   Found 3 relevant files
[14:30:46] started   investigate  (debugger)                  Beginning investigation
[14:31:02] info      investigate  (debugger)          1.5k   Analyzing memory patterns
[14:32:15] completed investigate  (debugger)   89s    8.2k   Root cause identified
```

---

## `wave cancel`

Cancel a running pipeline execution.

```bash
wave cancel [run-id] [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-f, --force` | `bool` | `false` | Interrupt immediately (send SIGTERM/SIGKILL). |
| `--format` | `string` | `"text"` | Output format: `text`, `json`. |

### Behavior

**Graceful cancellation (default):**
- Sets a cancellation flag in the database.
- The executor will stop after the current step completes.
- The pipeline status is marked as "cancelled".

**Force cancellation (`--force`):**
- Immediately sends SIGTERM to the adapter process group.
- Waits 5 seconds, then sends SIGKILL if still running.
- The current step may be incomplete.

### Examples

```bash
# Cancel most recent running pipeline
wave cancel

# Cancel specific run
wave cancel debug-20260202-143022

# Forcibly terminate immediately
wave cancel --force

# Output result as JSON
wave cancel --format json
```

### Output

```
$ wave cancel
Cancellation requested for debug-20260202-143022 (debug)
Pipeline will stop after current step completes.

$ wave cancel --force
Force cancellation sent to debug-20260202-143022 (debug)
Process terminated.
```

---

## `wave artifacts`

List and export artifacts from pipeline runs.

```bash
wave artifacts [run-id] [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--step` | `string` | `""` | Filter to specific step ID. |
| `--export` | `string` | `""` | Export artifacts to specified directory. |
| `--format` | `string` | `"table"` | Output format: `table`, `json`. |
| `--manifest` | `string` | `"wave.yaml"` | Path to manifest file. |

### Behavior

- **No arguments**: Shows artifacts from the most recent run.
- **With run-id**: Shows artifacts from that specific run.
- **With `--export`**: Copies artifacts to the specified directory, organized by step.

### Examples

```bash
# List artifacts from most recent run
wave artifacts

# List artifacts from specific run
wave artifacts debug-20260202-143022

# Filter to specific step
wave artifacts --step implement

# Export all artifacts
wave artifacts --export ./output

# Export specific step's artifacts
wave artifacts --step implement --export ./output

# Output as JSON
wave artifacts --format json
```

### Output

```
$ wave artifacts
STEP          ARTIFACT              TYPE    SIZE      PATH
navigate      analysis.md           md      2.1 KB    .wave/workspaces/debug.../navigate/analysis.md
investigate   findings.json         json    4.5 KB    .wave/workspaces/debug.../investigate/findings.json
investigate   memory_profile.txt    txt     12.3 KB   .wave/workspaces/debug.../investigate/memory_profile.txt

$ wave artifacts --export ./output
Exported 3 artifacts to ./output/
  ./output/navigate/analysis.md
  ./output/investigate/findings.json
  ./output/investigate/memory_profile.txt
```

---

## `wave clean`

Clean up ephemeral workspaces and state.

```bash
wave clean [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--pipeline` | `string` | `""` | Clean specific pipeline workspace. |
| `--all` | `bool` | `false` | Clean all workspaces and state. |
| `--dry-run` | `bool` | `false` | Show what would be deleted without deleting. |
| `--force` | `bool` | `false` | Skip confirmation prompt. |
| `--keep-last` | `int` | `-1` | Keep the N most recent workspaces (use with `--all`). |
| `--older-than` | `string` | `""` | Remove workspaces older than duration (e.g., `7d`, `24h`, `1h30m`). |
| `--status` | `string` | `""` | Only clean workspaces for pipelines with given status (`completed`, `failed`). |
| `--quiet` | `bool` | `false` | Suppress output for scripting. |

### Examples

```bash
# Clean a specific pipeline's workspace
wave clean --pipeline debug-20260202-143022

# Clean all workspaces
wave clean --all

# Preview cleanup
wave clean --all --dry-run

# Clean workspaces older than 7 days
wave clean --older-than 7d

# Clean only failed pipelines
wave clean --status failed

# Keep last 5 workspaces
wave clean --all --keep-last 5

# Quiet mode for scripting
wave clean --older-than 7d --force --quiet
```

### Output

```
$ wave clean --all --dry-run
Would delete:
  .wave/workspaces/debug-20260202-143022/  (debug, 2026-02-02, 145 MB)
  .wave/workspaces/hotfix-20260201-093000/ (hotfix, 2026-02-01, 23 MB)
Total: 168 MB across 2 pipelines

Run without --dry-run to delete.
```

---

## `wave list`

List available pipelines, personas, adapters, and runs.

```bash
wave list [resource] [flags]
```

### Resources

| Resource | Description |
|----------|-------------|
| `pipelines` | List discovered pipeline files. |
| `personas` | List defined personas with their adapters. |
| `adapters` | List defined adapters with their binaries. |
| `runs` | List recent pipeline executions. |

With no arguments, lists pipelines and personas.

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--manifest` | `string` | `"wave.yaml"` | Path to manifest file. |
| `--format` | `string` | `"table"` | Output format: `table`, `json`. |
| `--limit` | `int` | `10` | Maximum number of runs to show (for `list runs`). |
| `--run-pipeline` | `string` | `""` | Filter to specific pipeline (for `list runs`). |
| `--run-status` | `string` | `""` | Filter by status (for `list runs`). |

### Examples

```bash
# List all personas
wave list personas

# List pipelines as JSON
wave list pipelines --format json

# List recent runs
wave list runs

# List runs filtered by status
wave list runs --run-status failed

# List runs for specific pipeline
wave list runs --run-pipeline debug
```

### Output

```
$ wave list personas
NAME          ADAPTER   TEMPERATURE   DESCRIPTION
navigator     claude    0.1           Read-only codebase exploration
philosopher   claude    0.3           Architecture and specification
planner       claude    0.3           Task breakdown and planning
craftsman     claude    0.7           Implementation and testing
debugger      claude    0.2           Systematic issue diagnosis
auditor       claude    0.1           Security and quality review
summarizer    claude    0.0           Context compaction

$ wave list runs
RUN_ID                        PIPELINE     STATUS      STARTED               DURATION
debug-20260202-143022         debug        completed   2026-02-02 14:30:22   5m23s
hotfix-20260201-093000        hotfix       failed      2026-02-01 09:30:00   2m15s
speckit-20260131-160000       speckit-flow completed   2026-01-31 16:00:00   12m45s
```

---

## Exit Codes

All commands use consistent exit codes:

| Code | Name | Description |
|------|------|-------------|
| `0` | Success | Command completed successfully. |
| `1` | General Error | Unexpected error (file I/O, SQLite, etc.). |
| `2` | Usage Error | Invalid flags, missing required arguments, malformed input. |
| `3` | Pipeline Failed | One or more steps exceeded max retries. |
| `4` | Validation Error | Manifest or pipeline validation failed. |
| `5` | Timeout | Pipeline or step exceeded configured timeout. |
| `130` | Interrupted | User pressed Ctrl+C. Pipeline state is persisted for resumption. |

---

## Shell Completion

```bash
# Bash
wave completion bash > /etc/bash_completion.d/wave

# Zsh
wave completion zsh > "${fpath[1]}/_wave"

# Fish
wave completion fish > ~/.config/fish/completions/wave.fish
```
