# CLI Reference

Wave CLI commands for pipeline orchestration.

## Quick Reference

| Command | Purpose |
|---------|---------|
| `wave init` | Initialize a new project |
| `wave run` | Execute a pipeline |
| `wave do` | Run an ad-hoc task |
| `wave status` | Check pipeline status |
| `wave logs` | View execution logs |
| `wave resume` | Resume interrupted pipeline |
| `wave cancel` | Cancel running pipeline |
| `wave artifacts` | List and export artifacts |
| `wave list` | List pipelines and personas |
| `wave validate` | Validate configuration |
| `wave clean` | Clean up workspaces |
| `wave migrate` | Database migrations |

---

## wave init

Initialize a new Wave project.

```bash
wave init
```

**Output:**
```
Created wave.yaml
Created .wave/personas/navigator.md
Created .wave/personas/craftsman.md
Created .wave/personas/summarizer.md
Created .wave/pipelines/default.yaml

Project initialized. Run 'wave validate' to check configuration.
```

### Options

```bash
wave init --adapter opencode    # Use different adapter
wave init --force               # Overwrite existing files
wave init --merge               # Merge into existing config
```

---

## wave run

Execute a pipeline.

```bash
wave run --pipeline code-review --input "Review auth module"
```

**Output:**
```
[run-abc123] Starting pipeline: code-review
[run-abc123] Step: analyze (navigator) - started
[run-abc123] Step: analyze (navigator) - completed (45s)
[run-abc123] Step: review (auditor) - started
[run-abc123] Step: review (auditor) - completed (1m12s)
[run-abc123] Pipeline completed in 1m57s
```

### Options

```bash
wave run --pipeline hotfix --dry-run           # Preview without executing
wave run --pipeline speckit-flow --from-step implement  # Start from step
wave run --pipeline migrate --timeout 60       # Custom timeout (minutes)
```

---

## wave do

Run an ad-hoc task without a pipeline file.

```bash
wave do "fix the typo in README.md"
```

**Output:**
```
[run-xyz789] Generated 2-step pipeline: navigate -> execute
[run-xyz789] Step: navigate (navigator) - started
[run-xyz789] Step: navigate (navigator) - completed (23s)
[run-xyz789] Step: execute (craftsman) - started
[run-xyz789] Step: execute (craftsman) - completed (1m05s)
[run-xyz789] Task completed in 1m28s
```

### Options

```bash
wave do "audit auth" --persona auditor         # Use specific persona
wave do "build API" --meta                     # Generate custom pipeline
wave do "refactor" --save my-pipeline.yaml     # Save generated pipeline
wave do "test" --dry-run                       # Preview only
```

---

## wave status

Check pipeline run status.

```bash
wave status
```

**Output:**
```
RUN_ID          PIPELINE      STATUS     STEP        ELAPSED    TOKENS
run-abc123      code-review   running    review      2m15s      12k
run-xyz789      hotfix        completed  -           5m23s      28k
```

### Detailed Status

```bash
wave status run-abc123
```

**Output:**
```
Run ID:     run-abc123
Pipeline:   code-review
Status:     running
Step:       review
Started:    2026-02-03 14:30:22
Elapsed:    2m15s
Input:      Review auth module

Steps:
  analyze   completed   45s
  review    running     1m30s
```

### Options

```bash
wave status --all                # Show all recent runs
wave status --format json        # JSON output for scripting
```

---

## wave logs

View execution logs.

```bash
wave logs run-abc123
```

**Output:**
```
[14:30:22] started   analyze   (navigator)              Starting analysis
[14:31:07] completed analyze   (navigator)  45s  2.1k   Found 5 relevant files
[14:31:08] started   review    (auditor)                Beginning review
[14:32:20] info      review    (auditor)        1.5k   Checking security patterns
```

### Options

```bash
wave logs --step analyze         # Filter by step
wave logs --errors               # Show only errors
wave logs --tail 20              # Last 20 entries
wave logs --follow               # Stream in real-time
wave logs --since 10m            # Last 10 minutes
```

---

## wave resume

Resume an interrupted pipeline.

```bash
wave resume run-abc123
```

**Output:**
```
[run-abc123] Resuming from step: review
[run-abc123] Skipping completed: analyze
[run-abc123] Step: review (auditor) - started
[run-abc123] Step: review (auditor) - completed (52s)
[run-abc123] Pipeline completed in 52s
```

### Options

```bash
wave resume                      # List resumable runs
wave resume run-abc123 --from-step implement  # Override resume point
```

---

## wave cancel

Cancel a running pipeline.

```bash
wave cancel run-abc123
```

**Output:**
```
Cancellation requested for run-abc123 (code-review)
Pipeline will stop after current step completes.
```

### Force Cancel

```bash
wave cancel run-abc123 --force
```

**Output:**
```
Force cancellation sent to run-abc123 (code-review)
Process terminated.
```

---

## wave artifacts

List and export artifacts.

```bash
wave artifacts run-abc123
```

**Output:**
```
STEP      ARTIFACT        TYPE    SIZE      PATH
analyze   analysis.json   json    2.1 KB    .wave/workspaces/.../analysis.json
review    findings.md     md      4.5 KB    .wave/workspaces/.../findings.md
```

### Export Artifacts

```bash
wave artifacts run-abc123 --export ./output
```

**Output:**
```
Exported 2 artifacts to ./output/
  ./output/analyze/analysis.json
  ./output/review/findings.md
```

### Options

```bash
wave artifacts --step analyze    # Filter by step
wave artifacts --format json     # JSON output
```

---

## wave list

List resources.

```bash
wave list pipelines
```

**Output:**
```
NAME          STEPS   DESCRIPTION
code-review   4       Automated code review
hotfix        3       Fast-track bug fix
speckit-flow  5       Feature development
```

```bash
wave list personas
```

**Output:**
```
NAME          ADAPTER   TEMP   DESCRIPTION
navigator     claude    0.1    Read-only codebase exploration
craftsman     claude    0.7    Implementation and testing
auditor       claude    0.1    Security and quality review
```

```bash
wave list runs
```

**Output:**
```
RUN_ID          PIPELINE      STATUS      STARTED               DURATION
run-abc123      code-review   completed   2026-02-03 14:30:22   5m23s
run-xyz789      hotfix        failed      2026-02-03 09:30:00   2m15s
```

### Options

```bash
wave list runs --status failed           # Filter by status
wave list runs --limit 20                # Show more runs
wave list --format json                  # JSON output
```

---

## wave validate

Validate configuration.

```bash
wave validate
```

**Output (success):**
```
Validating wave.yaml...
  Adapters: 1 defined
  Personas: 5 defined
  Pipelines: 3 discovered

All validation checks passed.
```

**Output (errors):**
```
Validating wave.yaml...
ERROR: Persona 'craftsman' references undefined adapter 'opencode'
ERROR: System prompt file not found: .wave/personas/missing.md

Validation failed with 2 errors.
```

### Options

```bash
wave validate --verbose              # Show all checks
wave validate --pipeline hotfix.yaml # Validate specific pipeline
```

---

## wave clean

Clean up workspaces.

```bash
wave clean --dry-run
```

**Output:**
```
Would delete:
  .wave/workspaces/run-abc123/  (code-review, 145 MB)
  .wave/workspaces/run-xyz789/  (hotfix, 23 MB)
Total: 168 MB across 2 runs

Run without --dry-run to delete.
```

### Options

```bash
wave clean --all                     # Clean all workspaces
wave clean --older-than 7d           # Clean runs older than 7 days
wave clean --status completed        # Clean only completed runs
wave clean --keep-last 5             # Keep 5 most recent
wave clean --force                   # Skip confirmation
```

---

## wave migrate

Database migration commands.

```bash
wave migrate status
```

**Output:**
```
Current version: 3
Available migrations: 5
Pending: 2

  v1  core_tables      applied   2026-01-15 10:00:00
  v2  add_artifacts    applied   2026-01-20 14:30:00
  v3  add_metrics      applied   2026-02-01 09:00:00
  v4  add_checkpoints  pending
  v5  add_relay        pending
```

### Apply Migrations

```bash
wave migrate up
```

**Output:**
```
Applying migration v4: add_checkpoints... done
Applying migration v5: add_relay... done

Migrations complete. Current version: 5
```

### Rollback

```bash
wave migrate down 3
```

**Output:**
```
Rolling back to version 3...
  Reverting v5: add_relay... done
  Reverting v4: add_checkpoints... done

Rollback complete. Current version: 3
```

---

## Global Options

All commands support:

| Flag | Short | Description |
|------|-------|-------------|
| `--help` | `-h` | Show help |
| `--version` | `-v` | Show version |
| `--manifest` | `-m` | Path to wave.yaml |
| `--debug` | `-d` | Enable debug logging |
| `--log-format` | | Output format: text, json |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Usage error |
| 3 | Pipeline failed |
| 4 | Validation error |
| 5 | Timeout |
| 130 | Interrupted (Ctrl+C) |

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

## Next Steps

- [Quickstart](/quickstart) - Run your first pipeline
- [Pipelines](/concepts/pipelines) - Define multi-step workflows
- [Manifest Reference](/reference/manifest) - Configuration options
