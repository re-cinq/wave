# CLI Reference

Complete reference for all Muzzle CLI commands, flags, and output behavior.

## Synopsis

```
muzzle <command> [flags]
```

## Global Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--help` | `-h` | — | — | Show help for any command. |
| `--version` | `-v` | — | — | Print Muzzle version and exit. |
| `--manifest` | `-m` | `string` | `muzzle.yaml` | Path to manifest file. |
| `--debug` | | `bool` | `false` | Enable debug logging to stderr. |
| `--log-format` | | `string` | `json` | Output format: `json` or `text`. |

---

## `muzzle init`

Initialize a new Muzzle project in the current directory.

```bash
muzzle init [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--adapter` | `string` | `"claude"` | Default adapter to configure. |
| `--persona` | `string` | `"craftsman"` | Initial persona to create alongside the built-in navigator. |
| `--workspace` | `string` | `"/tmp/muzzle"` | Workspace root directory. |
| `--force` | `bool` | `false` | Overwrite existing `muzzle.yaml` if present. |

### Creates

```
muzzle.yaml                          # Project manifest
.muzzle/
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
muzzle init

# Initialize with specific adapter
muzzle init --adapter opencode

# Initialize with custom workspace
muzzle init --workspace ./workspaces

# Force re-initialization
muzzle init --force
```

---

## `muzzle validate`

Validate manifest and pipeline configuration files.

```bash
muzzle validate [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--manifest` | `string` | `"muzzle.yaml"` | Path to manifest file. |
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
muzzle validate

# Validate with verbose output
muzzle validate --verbose

# Validate specific manifest
muzzle validate --manifest staging.muzzle.yaml

# Validate a single pipeline
muzzle validate --pipeline .muzzle/pipelines/deploy.yaml
```

### Output

```
$ muzzle validate --verbose
✓ Manifest syntax valid
✓ All required fields present
✓ Adapter 'claude' references valid binary
✓ Persona 'navigator' references adapter 'claude'
✓ Persona 'navigator' system prompt exists: .muzzle/personas/navigator.md
✓ Persona 'craftsman' references adapter 'claude'
✓ Persona 'craftsman' system prompt exists: .muzzle/personas/craftsman.md
✓ Pipeline 'speckit-flow' DAG is valid (5 steps, no cycles)
⚠ Adapter 'opencode' binary not found on PATH

Validation passed with 1 warning.
```

---

## `muzzle run`

Execute a pipeline.

```bash
muzzle run --pipeline <path> [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--pipeline` | `string` | **required** | Path to pipeline YAML file. |
| `--input` | `string` | `""` | Input prompt for the pipeline. Available as `{{ input }}` in step templates. |
| `--dry-run` | `bool` | `false` | Walk the pipeline DAG and print execution plan without invoking adapters. |
| `--from-step` | `string` | `""` | Start execution from a specific step (skip completed predecessors). |
| `--manifest` | `string` | `"muzzle.yaml"` | Path to manifest file. |
| `--timeout` | `int` | manifest default | Override per-step timeout (minutes). |

### Output

Emits [NDJSON events](/reference/events) to stdout on every state transition. Errors and debug info go to stderr.

### Examples

```bash
# Run a pipeline with input
muzzle run --pipeline .muzzle/pipelines/speckit-flow.yaml \
  --input "add user authentication with JWT"

# Dry run to preview execution plan
muzzle run --pipeline .muzzle/pipelines/speckit-flow.yaml --dry-run

# Resume from a specific step
muzzle run --pipeline .muzzle/pipelines/speckit-flow.yaml \
  --from-step implement

# With custom timeout
muzzle run --pipeline .muzzle/pipelines/speckit-flow.yaml \
  --input "refactor database layer" --timeout 60

# Pipe events to jq for filtering
muzzle run --pipeline flow.yaml --input "task" \
  | jq 'select(.state == "completed")'
```

### Dry Run Output

```
$ muzzle run --pipeline speckit-flow.yaml --dry-run
Pipeline: speckit-flow (5 steps)
Execution order:
  1. navigate    (navigator)     → no dependencies
  2. specify     (philosopher)   → depends on: navigate
  3. plan        (philosopher)   → depends on: specify
  4. implement   (craftsman)     → depends on: plan
  5. review      (auditor)       → depends on: implement
```

---

## `muzzle do`

Execute an ad-hoc task. Generates and runs a minimal 2-step pipeline (navigate → execute) without requiring a pipeline file.

```bash
muzzle do "<task description>" [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--persona` | `string` | `"craftsman"` | Persona for the execution step. The navigate step always uses `navigator`. |
| `--save` | `string` | `""` | Save the generated pipeline YAML to this path for inspection or reuse. |
| `--manifest` | `string` | `"muzzle.yaml"` | Path to manifest file. |
| `--timeout` | `int` | manifest default | Override per-step timeout (minutes). |

### Generated Pipeline Structure

```yaml
# Auto-generated by muzzle do
kind: MuzzlePipeline
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
muzzle do "fix the typo in README.md line 42"

# With specific persona
muzzle do "audit authentication middleware for SQL injection" --persona auditor

# Save generated pipeline for inspection
muzzle do "add dark mode toggle" --save .muzzle/pipelines/dark-mode.yaml

# Use non-default manifest
muzzle do "fix broken test" --manifest staging.muzzle.yaml
```

---

## `muzzle resume`

Resume an interrupted or failed pipeline from its last checkpoint.

```bash
muzzle resume --pipeline-id <uuid> [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--pipeline-id` | `string` | **required** | UUID of the pipeline execution to resume. Found in event output. |
| `--from-step` | `string` | `""` | Override resume point. By default, resumes from the last completed step. |
| `--manifest` | `string` | `"muzzle.yaml"` | Path to manifest file. |

### Resume Behavior

1. Loads persisted pipeline state from SQLite.
2. Identifies the last completed step.
3. Resumes execution from the next pending or failed step.
4. Completed steps are **skipped** — their artifacts remain in the workspace.
5. Failed steps are **retried** from scratch (fresh workspace, full retry budget).

### Examples

```bash
# Resume from last checkpoint
muzzle resume --pipeline-id a1b2c3d4-e5f6-7890-abcd-ef1234567890

# Resume from a specific step
muzzle resume --pipeline-id a1b2c3d4 --from-step implement
```

---

## `muzzle clean`

Clean up ephemeral workspaces. Workspaces are never auto-deleted — this is the only way to reclaim disk space.

```bash
muzzle clean [flags]
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
muzzle clean --pipeline-id a1b2c3d4

# Clean all workspaces
muzzle clean --all

# Preview cleanup
muzzle clean --all --dry-run

# Clean workspaces older than 7 days
muzzle clean --older-than 7d
```

### Output

```
$ muzzle clean --all --dry-run
Would delete:
  /tmp/muzzle/a1b2c3d4/  (speckit-flow, 2026-02-01, 145MB)
  /tmp/muzzle/e5f6a7b8/  (bug-fix, 2026-01-30, 23MB)
Total: 168MB across 2 pipelines

Run without --dry-run to delete.
```

---

## `muzzle list`

List available pipelines, personas, and adapters defined in the manifest.

```bash
muzzle list <resource> [flags]
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
| `--manifest` | `string` | `"muzzle.yaml"` | Path to manifest file. |
| `--output` | `string` | `"table"` | Output format: `table`, `json`, `yaml`. |

### Examples

```bash
# List all personas
muzzle list personas

# List pipelines as JSON
muzzle list pipelines --output json

# List recent runs
muzzle list runs
```

### Output

```
$ muzzle list personas
NAME          ADAPTER   TEMPERATURE   DESCRIPTION
navigator     claude    0.1           Read-only codebase exploration
philosopher   claude    0.3           Architecture and specification design
craftsman     claude    0.7           Implementation and testing
auditor       claude    0.1           Security and quality review
summarizer    claude    0.0           Relay checkpoint summarizer

$ muzzle list pipelines
NAME            STEPS   PATH
speckit-flow    5       .muzzle/pipelines/speckit-flow.yaml
bug-fix         4       .muzzle/pipelines/bug-fix.yaml
auto-design     2       .muzzle/pipelines/auto-design.yaml

$ muzzle list runs
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
muzzle completion bash > /etc/bash_completion.d/muzzle

# Zsh
muzzle completion zsh > "${fpath[1]}/_muzzle"

# Fish
muzzle completion fish > ~/.config/fish/completions/muzzle.fish
```
