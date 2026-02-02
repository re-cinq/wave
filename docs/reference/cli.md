# CLI Reference

Complete reference for all Wave CLI commands, flags, and output behavior.

## Synopsis

```
wave <command> [flags]
```

## Global Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--help` | `-h` | — | — | Show help for any command. |
| `--version` | `-v` | — | — | Print Wave version and exit. |
| `--manifest` | `-m` | `string` | `wave.yaml` | Path to manifest file. |
| `--debug` | | `bool` | `false` | Enable debug logging to stderr. |
| `--log-format` | | `string` | `json` | Output format: `json` or `text`. |

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
| `--persona` | `string` | `"craftsman"` | Initial persona to create alongside the built-in navigator. |
| `--workspace` | `string` | `"/tmp/wave"` | Workspace root directory. |
| `--force` | `bool` | `false` | Overwrite existing `wave.yaml` if present. |

### Creates

```
wave.yaml                          # Project manifest
.wave/
├── personas/
│   ├── navigator.md                 # Navigator system prompt
│   ├── craftsman.md                 # Craftsman system prompt
│   └── summarizer.md               # Summarizer system prompt
├── pipelines/
│   └── default.yaml                 # Example pipeline
├── hooks/                           # Hook script directory
└── contracts/                       # Contract schema directory
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

### Output

```
$ wave validate --verbose
✓ Manifest syntax valid
✓ All required fields present
✓ Adapter 'claude' references valid binary
✓ Persona 'navigator' references adapter 'claude'
✓ Persona 'navigator' system prompt exists: .wave/personas/navigator.md
✓ Persona 'craftsman' references adapter 'claude'
✓ Persona 'craftsman' system prompt exists: .wave/personas/craftsman.md
✓ Pipeline 'speckit-flow' DAG is valid (5 steps, no cycles)
⚠ Adapter 'opencode' binary not found on PATH

Validation passed with 1 warning.
```

---

## `wave run`

Execute a pipeline.

```bash
wave run --pipeline <path> [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--pipeline` | `string` | **required** | Path to pipeline YAML file. Can also be a pipeline name (e.g., `add-feature`) which will be auto-detected in the `.wave/pipelines/` directory. |
| `--input` | `string` | `""` | Input prompt for the pipeline. Available as `{{ input }}` in step templates. |
| `--dry-run` | `bool` | `false` | Walk the pipeline DAG and print execution plan without invoking adapters. |
| `--from-step` | `string` | `""` | Start execution from a specific step (skip completed predecessors). |
| `--manifest` | `string` | `"wave.yaml"` | Path to manifest file. |
| `--timeout` | `int` | manifest default | Override per-step timeout (minutes). |

### Output

Emits [NDJSON events](/reference/events) to stdout on every state transition. Errors and debug info go to stderr.

### Examples

```bash
# Run a pipeline with input
wave run --pipeline .wave/pipelines/speckit-flow.yaml \
  --input "add user authentication with JWT"

# Dry run to preview execution plan
wave run --pipeline .wave/pipelines/speckit-flow.yaml --dry-run

# Resume from a specific step
wave run --pipeline .wave/pipelines/speckit-flow.yaml \
  --from-step implement

# With custom timeout
wave run --pipeline .wave/pipelines/speckit-flow.yaml \
  --input "refactor database layer" --timeout 60

# Pipe events to jq for filtering
wave run --pipeline flow.yaml --input "task" \
  | jq 'select(.state == "completed")'
```

### Dry Run Output

```
$ wave run --pipeline speckit-flow.yaml --dry-run
Pipeline: speckit-flow (5 steps)
Execution order:
  1. navigate    (navigator)     → no dependencies
  2. specify     (philosopher)   → depends on: navigate
  3. plan        (philosopher)   → depends on: specify
  4. implement   (craftsman)     → depends on: plan
  5. review      (auditor)       → depends on: implement
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
| `--persona` | `string` | `"craftsman"` | Persona for the execution step. The navigate step always uses `navigator`. |
| `--save` | `string` | `""` | Save the generated pipeline YAML to this path for inspection or reuse. |
| `--manifest` | `string` | `"wave.yaml"` | Path to manifest file. |
| `--timeout` | `int` | manifest default | Override per-step timeout (minutes). |

### Generated Pipeline Structure

```yaml
# Auto-generated by wave do
kind: WavePipeline
metadata:
  name: ad-hoc-<timestamp>
steps:
  - id: navigate
    persona: navigator
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: "Analyze the codebase for: <task description>"
  - id: execute
    persona: <selected-persona>
    dependencies: [navigate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: navigate
          artifact: analysis
          as: navigation_report
    exec:
      type: prompt
      source: "<task description>"
```

### Examples

```bash
# Quick fix
wave do "fix the typo in README.md line 42"

# With specific persona
wave do "audit authentication middleware for SQL injection" --persona auditor

# Save generated pipeline for inspection
wave do "add dark mode toggle" --save .wave/pipelines/dark-mode.yaml

# Use non-default manifest
wave do "fix broken test" --manifest staging.wave.yaml
```

---

## `wave resume`

Resume an interrupted or failed pipeline from its last checkpoint.

```bash
wave resume --pipeline-id <uuid> [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--pipeline-id` | `string` | **required** | UUID of the pipeline execution to resume. Found in event output. |
| `--from-step` | `string` | `""` | Override resume point. By default, resumes from the last completed step. |
| `--manifest` | `string` | `"wave.yaml"` | Path to manifest file. |

### Resume Behavior

1. Loads persisted pipeline state from SQLite.
2. Identifies the last completed step.
3. Resumes execution from the next pending or failed step.
4. Completed steps are **skipped** — their artifacts remain in the workspace.
5. Failed steps are **retried** from scratch (fresh workspace, full retry budget).

### Examples

```bash
# Resume from last checkpoint
wave resume --pipeline-id a1b2c3d4-e5f6-7890-abcd-ef1234567890

# Resume from a specific step
wave resume --pipeline-id a1b2c3d4 --from-step implement
```

---

## `wave clean`

Clean up ephemeral workspaces. Workspaces are never auto-deleted — this is the only way to reclaim disk space.

```bash
wave clean [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--pipeline-id` | `string` | `""` | Clean only this pipeline's workspaces. |
| `--all` | `bool` | `false` | Clean all workspaces under the workspace root. |
| `--dry-run` | `bool` | `false` | Show what would be deleted without deleting. |
| `--older-than` | `string` | `""` | Clean workspaces older than duration (e.g., `24h`, `7d`). |

### Examples

```bash
# Clean a specific pipeline's workspace
wave clean --pipeline-id a1b2c3d4

# Clean all workspaces
wave clean --all

# Preview cleanup
wave clean --all --dry-run

# Clean workspaces older than 7 days
wave clean --older-than 7d
```

### Output

```
$ wave clean --all --dry-run
Would delete:
  /tmp/wave/a1b2c3d4/  (speckit-flow, 2026-02-01, 145MB)
  /tmp/wave/e5f6a7b8/  (bug-fix, 2026-01-30, 23MB)
Total: 168MB across 2 pipelines

Run without --dry-run to delete.
```

---

## `wave list`

List available pipelines, personas, and adapters defined in the manifest.

```bash
wave list <resource> [flags]
```

### Resources

| Resource | Description |
|----------|-------------|
| `pipelines` | List discovered pipeline files. |
| `personas` | List defined personas with their adapters. |
| `adapters` | List defined adapters with their binaries. |
| `runs` | List recent pipeline executions from state store. |

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--manifest` | `string` | `"wave.yaml"` | Path to manifest file. |
| `--output` | `string` | `"table"` | Output format: `table`, `json`, `yaml`. |

### Examples

```bash
# List all personas
wave list personas

# List pipelines as JSON
wave list pipelines --output json

# List recent runs
wave list runs
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

$ wave list pipelines
NAME            STEPS   PATH
speckit-flow    5       .wave/pipelines/speckit-flow.yaml
hotfix          3       .wave/pipelines/hotfix.yaml
code-review     4       .wave/pipelines/code-review.yaml
refactor        4       .wave/pipelines/refactor.yaml
debug           4       .wave/pipelines/debug.yaml
test-gen        3       .wave/pipelines/test-gen.yaml
docs            3       .wave/pipelines/docs.yaml
plan            3       .wave/pipelines/plan.yaml
migrate         4       .wave/pipelines/migrate.yaml

$ wave list runs
PIPELINE-ID   NAME           STATUS      STARTED              STEPS
a1b2c3d4      speckit-flow   completed   2026-02-01 10:00:00  5/5
e5f6a7b8      bug-fix        failed      2026-01-30 14:30:00  2/4
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

## Shell Completion

```bash
# Bash
wave completion bash > /etc/bash_completion.d/wave

# Zsh
wave completion zsh > "${fpath[1]}/_wave"

# Fish
wave completion fish > ~/.config/fish/completions/wave.fish
```
