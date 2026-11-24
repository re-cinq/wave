# CLI Reference

Wave CLI commands for pipeline orchestration.

## Quick Reference

| Command | Purpose |
|---------|---------|
| `wave init` | Initialize a new project |
| `wave run` | Execute a pipeline |
| `wave status` | Check pipeline status |
| `wave logs` | View execution logs |
| `wave cancel` | Cancel running pipeline |
| `wave artifacts` | List and export artifacts |
| `wave list` | List adapters, runs, pipelines, personas, contracts |
| `wave validate` | Validate configuration |
| `wave clean` | Clean up workspaces |
| `wave cleanup` | Remove orphaned worktrees from .agents/workspaces/ |
| `wave decisions` | Show decision log for a pipeline run |
| `wave doctor` | Diagnose project configuration and health |
| `wave fork` | Fork a run from a checkpoint |
| `wave merge` | Merge a pull request using forge CLI |
| `wave persona` | Persona management (create, list) |
| `wave pipeline` | Pipeline management (create, list) |
| `wave retro` | View and manage run retrospectives |
| `wave rewind` | Rewind a run to an earlier checkpoint |
| `wave skills` | Discover, validate, install, and diagnose SKILL.md files |
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
Created .agents/personas/navigator.md
Created .agents/personas/craftsman.md
Created .agents/personas/summarizer.md
Created .agents/pipelines/default.yaml

Project initialized. Run 'wave validate' to check configuration.
```

### Options

```bash
wave init --adapter opencode    # Use different adapter
wave init --force               # Overwrite existing files
wave init --merge               # Merge into existing config
wave init --reconfigure         # Re-run onboarding wizard with current settings as defaults
wave init --all                 # Include all pipelines regardless of release status
wave init --workspace ./ws      # Custom workspace directory path
wave init --output config.yaml  # Custom output path for wave.yaml
wave init -y                    # Answer yes to all confirmation prompts
```

---

## wave run

Execute a pipeline. Arguments can be provided as positional args or flags.

```bash
# Positional arguments (recommended for quick usage)
wave run ops-pr-review "Review auth module"

# Flag-based (explicit)
wave run --pipeline ops-pr-review --input "Review auth module"

# Mixed
wave run ops-pr-review --input "Review auth module"
```

**Output:**
```
[run-abc123] Starting pipeline: ops-pr-review
[run-abc123] Step: analyze (navigator) - started
[run-abc123] Step: analyze (navigator) - completed (45s)
[run-abc123] Step: review (auditor) - started
[run-abc123] Step: review (auditor) - completed (1m12s)
[run-abc123] Pipeline completed in 1m57s
```

### Options

Flags are organized into four tiers by usage frequency.

#### Essential (Tier 1)

| Flag | Description |
|------|-------------|
| `--pipeline` | Pipeline name to run |
| `--input` | Input data for the pipeline |
| `--model` | Model override (tier name or literal) |
| `--adapter` | Override adapter (claude, gemini, opencode, codex) |

#### Execution (Tier 2)

| Flag | Description |
|------|-------------|
| `--from-step` | Start execution from specific step |
| `--force` | Skip validation checks when using --from-step |
| `--dry-run` | Show what would be executed without running |
| `--timeout` | Timeout in minutes (0 = no timeout) |
| `--steps` | Run only named steps (comma-separated) |
| `-x, --exclude` | Skip named steps (comma-separated) |
| `--on-failure` | Failure policy: halt (default) or skip |
| `--detach` | Run as detached background process |

#### Dev/Debug (Tier 3)

| Flag | Description |
|------|-------------|
| `--mock` | Use mock adapter (testing) |
| `--preserve-workspace` | Keep workspace from previous run |
| `--auto-approve` | Auto-approve approval gates |
| `--no-retro` | Skip retrospective generation |
| `--force-model` | Force model on all steps |
| `--run` | Resume from specific run ID |
| `--manifest` | Path to manifest file |

### Examples

```bash
wave run plan-task --dry-run                 # Preview without executing
wave run impl-issue --from-step implement    # Start from step (auto-recovers input)
wave run impl-issue --from-step implement --force  # Skip validation for --from-step
wave run plan-research --from-step synthesize-report --run plan-research-20260219-fa19  # Recover input from specific run
wave run migrate --timeout 60                  # Custom timeout (minutes)
wave run test --mock                           # Use mock adapter for testing
wave run build -o json                         # NDJSON output to stdout (pipe-friendly)
wave run deploy -o text                        # Plain text progress to stderr
wave run review -o text -v                     # Plain text with real-time tool activity
wave run check -o quiet                        # Only final result to stderr
wave run build --model haiku                   # Override adapter model for this run
wave run impl-issue --adapter opencode --model "zai-coding-plan/glm-5-turbo"  # Override adapter and model
wave run audit-security --preserve-workspace        # Preserve workspace from previous run (for debugging)
wave run --detach impl-issue "fix login bug"   # Detach: run in background, survive shell exit
wave run impl-issue --steps fetch,implement    # Run only specific steps
wave run impl-issue -x validate               # Skip the validate step
wave run impl-issue --on-failure skip          # Continue on step failure
```

### Detached Mode

The `--detach` flag spawns the pipeline as a background process that survives shell exit.
The command prints the run ID and returns immediately. Use `wave logs` and `wave cancel` to manage it.

```bash
wave run --detach impl-issue -- "https://github.com/org/repo/issues/42"
# → Pipeline 'impl-issue' launched (detached)
# → Run ID:  impl-issue-20260317-...
# → Logs:    wave logs impl-issue-20260317-...
# → Cancel:  wave cancel impl-issue-20260317-...
```

The subprocess runs in its own session group
(`setsid`), so killing the parent terminal has no effect on the pipeline.

For structured callers, add `--output json`. Wave writes one acknowledgement to stdout
after reserving the run ID and releasing the child process:

```json
{"run_id":"impl-issue-20260317-...","pipeline_name":"impl-issue","status":"running","detached":true}
```

This acknowledgement means the detached process was launched. Observe the durable run
state with `wave status --format json -- <run-id>`; it does not mean the pipeline finished.

---

## wave meta

Generate and execute a custom multi-step pipeline dynamically using the philosopher persona.

```bash
wave meta "implement user authentication"
```

**Output:**
```
Invoking philosopher to generate pipeline...
This may take a moment while the AI designs your pipeline.

Generated pipeline: implement-auth
Steps:
  1. navigate [navigator]
  2. specify [philosopher]
  3. implement [craftsman]
  4. review [auditor]

Meta pipeline completed (3m28s)
  Total steps: 4, Total tokens: 45k
```

### Options

```bash
wave meta "build API" --save api-pipeline.yaml  # Save generated pipeline
wave meta "refactor code" --dry-run             # Preview without executing
wave meta "add tests" --mock                    # Use mock adapter for testing
wave meta "refactor" --model opus               # Override adapter model for this run
```

The `--save` flag is particularly useful for turning dynamically generated pipelines into reusable templates:

```bash
# Generate, save, and later re-run the same pipeline
wave meta "implement OAuth2 flow" --save .agents/pipelines/oauth2.yaml
wave run oauth2 "add Google provider"
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
run-abc123      ops-pr-review   running    review      2m15s      12k
run-xyz789      plan-task        completed  -           5m23s      28k
```

### Detailed Status

```bash
wave status run-abc123
```

**Output:**
```
Run ID:     run-abc123
Pipeline:   ops-pr-review
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
[14:30:22] started    analyze (navigator) Starting analysis
[14:31:07] completed  analyze (navigator) 45.0s 2.1k tokens Found 5 relevant files
[14:31:08] started    review (auditor) Beginning review
[14:32:20] completed  review (auditor) 72.0s 1.5k tokens Checking security patterns
```

### Options

```bash
wave logs --step analyze         # Filter by step
wave logs --errors               # Show only errors
wave logs --tail 20              # Last 20 entries
wave logs --follow               # Stream in real-time
wave logs --since 10m            # Last 10 minutes
wave logs --level error          # Log level filter (all, info, error)
wave logs --format json          # Output as JSON for scripting
```

---

## Logs vs Progress Output

Wave has two distinct observability mechanisms that serve different purposes:

- **`wave logs`** reads the event history from the state database *after* events have been recorded. It works on both running and completed pipelines and is the primary tool for post-hoc debugging.
- **`--output` modes** (`text`, `json`, `quiet`) control how real-time progress is rendered to the terminal *during* execution. They determine what you see while a pipeline runs.

### Comparison

| | `wave logs` | `--output` modes |
|---|---|---|
| **Mechanism** | Reads recorded events from SQLite state DB | Renders progress events to terminal in real-time |
| **Data source** | `.agents/state.db` (persisted) | Live event stream (ephemeral) |
| **Timing** | During or after execution | Only during execution |
| **Typical use** | Post-hoc debugging, audit trail, scripting | Watching progress, CI output formatting |

### Use-Case Examples

**1. Debugging a failed step**

After a pipeline fails, use `wave logs` to inspect what happened:

```bash
wave logs impl-issue-20260320-abc123 --errors
wave logs impl-issue-20260320-abc123 --step implement --format json
```

The logs are persisted in the state database, so you can inspect them long after the run finishes.

**2. Watching a pipeline run live**

To see real-time progress with tool activity while a pipeline executes:

```bash
wave run impl-issue -o text -v -- "https://github.com/org/repo/issues/42"
```

The `-o text` flag renders plain-text progress to stderr, and `-v` adds real-time tool activity lines. This output is ephemeral — once the terminal is closed, it is gone.

**3. Scripting and CI integration**

For machine-readable output, combine both mechanisms:

```bash
# Real-time: stream structured JSON events during execution
wave run impl-issue -o json -- "https://github.com/org/repo/issues/42"

# Post-hoc: query the state DB after completion
wave logs impl-issue-20260320-abc123 --format json
```

Use `-o json` when you need to process events as they happen (e.g., updating a CI dashboard). Use `wave logs --format json` when you need to analyze a completed run (e.g., extracting step durations for metrics).

---

## wave cancel

Cancel a running pipeline.

```bash
wave cancel run-abc123
```

**Output:**
```
Cancellation requested for run-abc123 (ops-pr-review)
Pipeline will stop after current step completes.
```

### Force Cancel

```bash
wave cancel run-abc123 --force
wave cancel run-abc123 -f              # Short flag for --force
```

**Output:**
```
Force cancellation sent to run-abc123 (ops-pr-review)
Process terminated.
```

### Options

```bash
wave cancel --format json       # Output cancellation result as JSON
wave cancel -f --format text    # Force cancel with text output (default)
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
analyze   analysis.json   json    2.1 KB    .agents/workspaces/.../analysis.json
review    findings.md     md      4.5 KB    .agents/workspaces/.../findings.md
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

List Wave configuration, resources, and execution history.

```bash
wave list                  # Show all categories
wave list adapters         # List configured adapters
wave list runs             # List recent pipeline runs
wave list pipelines        # List available pipelines
wave list personas         # List configured personas
wave list contracts        # List contract schemas
```

### Adapters

```bash
wave list adapters
```

**Output:**
```
Adapters
────────────────────────────────────────────────────────────

  ✓ claude
    binary: claude • mode: headless • format: json
```

### Runs

```bash
wave list runs
```

**Output:**
```
Recent Pipeline Runs
────────────────────────────────────────────────────────────────────────────────
  RUN_ID                    PIPELINE          STATUS        STARTED             DURATION
  run-abc123                ops-pr-review       completed     2026-02-03 14:30    5m23s
  run-xyz789                plan-task            failed        2026-02-03 09:30    2m15s
```

### Pipelines

```bash
wave list pipelines
```

**Output:**
```
Pipelines
────────────────────────────────────────────────────────────

  ops-pr-review [4 steps]
    Automated code review workflow
    ○ analyze → review → report → notify

  impl-issue [5 steps]
    End-to-end GitHub issue implementation
    o fetch-assess -> plan -> implement -> fix-implement -> create-pr
```

### Personas

```bash
wave list personas
```

**Output:**
```
Personas
────────────────────────────────────────────────────────────

  navigator
    adapter: claude • temp: 0.1 • allow:3
    Read-only codebase exploration

  craftsman
    adapter: claude • temp: 0.7 • allow:5
    Implementation and testing
```

### Contracts

```bash
wave list contracts
```

**Output:**
```
Contracts
────────────────────────────────────────────────────────────

  issue-assessment [json-schema]
    used by:
      * impl-issue -> fetch-assess (implementer)

  issue-impl-plan [json-schema]
    used by:
      * impl-issue -> plan (implementer)

  validation-report [json-schema]
    (unused)
```

### Options

```bash
wave list runs --run-status failed       # Filter by status
wave list runs --limit 20                # Show more runs
wave list runs --run-pipeline plan-task     # Filter by pipeline
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
ERROR: System prompt file not found: .agents/personas/missing.md

Validation failed with 2 errors.
```

### Options

```bash
wave validate -v                     # Show all checks (global --verbose flag)
wave validate --pipeline plan-task.yaml # Validate specific pipeline
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
  .agents/workspaces/run-abc123/  (ops-pr-review, 145 MB)
  .agents/workspaces/run-xyz789/  (plan-task, 23 MB)
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
wave clean --dry-run                 # Preview what would be deleted
wave clean --quiet                   # Suppress output for scripting
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

### Validate

Verify that applied migrations match their expected checksums and are in a consistent state.

```bash
wave migrate validate
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

## wave cleanup

Remove orphaned worktrees from `.agents/workspaces/` that have no corresponding running pipeline.

```bash
wave cleanup              # Remove orphaned worktrees (with confirmation)
wave cleanup --dry-run    # Show what would be removed without deleting
wave cleanup --force      # Skip confirmation prompt
```

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `false` | Show what would be removed without deleting anything |
| `--force` | `false` | Skip confirmation prompt |

---

## wave decisions

Show the structured decision log from pipeline runs. Decisions record model routing choices, retry attempts, contract validations, budget allocations, and composition selections.

```bash
wave decisions                          # Decisions from most recent run
wave decisions impl-issue-20240315-abc  # Decisions for a specific run
wave decisions --step plan              # Filter to a specific step
wave decisions --category model_routing # Filter by category
wave decisions --format json            # JSON output
```

| Flag | Default | Description |
|------|---------|-------------|
| `--step` | | Filter by step ID |
| `--category` | | Filter by category: `model_routing`, `retry`, `contract`, `budget`, `composition` |
| `--format` | `text` | Output format: `text`, `json` |
| `--manifest` | `wave.yaml` | Path to manifest file |

---

## wave fork

Create a new independent run branching from a specific step of an existing run. The forked run starts from the selected checkpoint with a fresh execution context.

```bash
wave fork impl-issue-20240315-abc123 --from-step plan
wave fork impl-issue-20240315-abc123 --list       # List available fork points
wave fork impl-issue-20240315-abc123 --from-step plan --input "updated input"
```

| Flag | Default | Description |
|------|---------|-------------|
| `--from-step` | | Fork from after this step (required unless `--list`) |
| `--list` | `false` | List available fork points |
| `--allow-failed` | `false` | Allow forking non-completed (failed/cancelled) runs |
| `--input` | | Override input for the forked run |
| `--model` | | Override adapter model for the forked run |
| `--manifest` | `wave.yaml` | Path to manifest file |

---

## wave merge

Merge a pull request by number or URL. Detects the forge type (GitHub, GitLab, Gitea, Bitbucket) and uses the appropriate forge CLI with API fallback.

```bash
wave merge 123
wave merge https://github.com/owner/repo/pull/123
wave merge --all              # Merge all approved open PRs (oldest first)
wave merge --all --yes        # Skip confirmation
```

| Flag | Default | Description |
|------|---------|-------------|
| `--all` | `false` | Merge all approved open PRs (oldest first) |
| `--yes` | `false` | Skip confirmation prompt (use with `--all`) |

---

## wave persona

Create and manage Wave personas.

### persona create

Scaffold a new persona from a built-in template.

```bash
wave persona create --name my-reviewer --template reviewer
```

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | | Name for the new persona (required) |
| `--template` | | Built-in persona template to use (required) |

### persona list

List available persona templates.

```bash
wave persona list
```

---

## wave pipeline

Create and manage Wave pipelines.

### pipeline create

Scaffold a new pipeline YAML from a built-in template.

```bash
wave pipeline create --name my-pipeline --template impl-issue
```

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | | Name for the new pipeline (required) |
| `--template` | | Built-in pipeline template to use (required) |

### pipeline list

List available pipelines (alias for `wave list pipelines`).

```bash
wave pipeline list
```

---

## wave retro

View, list, and generate retrospectives for pipeline runs.

```bash
wave retro                          # Retrospective for most recent run
wave retro impl-issue-20240315-abc  # Retrospective for a specific run
wave retro --narrate                # Generate LLM narrative
wave retro --json                   # JSON output
```

| Flag | Default | Description |
|------|---------|-------------|
| `--narrate` | `false` | Generate or regenerate LLM narrative |
| `--json` | `false` | Output in JSON format |

### retro list

List retrospectives across runs.

```bash
wave retro list
wave retro list --pipeline impl-issue
wave retro list --since 7d
```

| Flag | Default | Description |
|------|---------|-------------|
| `--pipeline` | | Filter by pipeline name |
| `--since` | | Show retros since duration (e.g. `7d`, `24h`) |

### retro stats

Show aggregate retrospective statistics.

```bash
wave retro stats
```

---

## wave rewind

Reset a run's state to an earlier checkpoint (destructive). The run can be resumed from the rewound point.

```bash
wave rewind impl-issue-20240315-abc123 --to-step plan
wave rewind impl-issue-20240315-abc123 --to-step plan --confirm
```

| Flag | Default | Description |
|------|---------|-------------|
| `--to-step` | | Rewind to after this step (required) |
| `--confirm` | `false` | Skip confirmation prompt |
| `--manifest` | `wave.yaml` | Path to manifest file |

---

## wave skills

Discover, validate, install, and diagnose SKILL.md files across project and user-global directories. Skills are lazy-loaded by each adapter via its native skill tool — Wave provisions the source files into the workspace at step start.

### Detection paths

Wave scans, in order (first match wins per skill name):

```
project:
  .agents/skills/<name>/SKILL.md      ← primary committed team source
  .claude/skills/<name>/SKILL.md      ← Claude Code skills
  .opencode/skills/                   ← opencode-specific
  .gemini/skills/                     ← gemini-specific

user-global:
  ~/.agents/skills/
  ~/.claude/skills/
  ~/.config/opencode/skills/
  ~/.gemini/skills/
```

### skills list

List discovered skills with the pipelines that reference them.

```bash
wave skills list
wave skills list --format json
```

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `table` | Output format: `table`, `json` |

### skills check

Validate a single skill and show which pipelines/steps reference it.

```bash
wave skills check golang
wave skills check golang --format json
```

### skills add

Install a skill from a local path or `file://` URL. Defaults to `~/.agents/skills/<name>/` (user-global). Use `--project` to commit the skill to `.agents/skills/<name>/`.

```bash
wave skills add ./my-skill
wave skills add file:///abs/path/to/skill
wave skills add ./my-skill --project
```

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `table` | Output format: `table`, `json` |
| `--project` | `false` | Install to `.agents/skills/` instead of user-global |

### skills doctor

Diagnose discovery issues: duplicate names across roots, malformed frontmatter, deprecated `.agents/skills/` references.

```bash
wave skills doctor
wave skills doctor --format json
```

---

## Global Options

All commands support:

| Flag | Short | Description |
|------|-------|-------------|
| `--help` | `-h` | Show help |
| `--version` | | Show version |
| `--manifest` | `-m` | Path to manifest file (default: wave.yaml) |
| `--debug` | `-d` | Enable debug mode |
| `--output` | `-o` | Output format: auto, json, text, quiet (default: auto) |
| `--verbose` | `-v` | Include real-time tool activity |
| `--json` | | Output in JSON format (equivalent to `--output json`) |
| `--quiet` | `-q` | Suppress non-essential output (equivalent to `--output quiet`) |
| `--no-color` | | Disable colored output |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (includes pipeline failures, timeouts, validation errors) |
| 2 | Usage error (invalid arguments or configuration) |

---

## Next Steps

- [Quick Start](/guide/quick-start) - Run your first pipeline
- [Pipelines](/concepts/pipelines) - Define multi-step workflows
- [Manifest Reference](/reference/manifest-schema) - Configuration options
