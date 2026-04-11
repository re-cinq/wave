# Running Pipelines

Wave pipelines can be launched from three surfaces: CLI, TUI, and WebUI. All surfaces expose the same set of run options, organized into four tiers.

## Tiered Options Model

Options are grouped by usage frequency so common flags are easy to find:

| Tier | Category | Flags |
|------|----------|-------|
| 1 | **Essential** | `--pipeline`, `--input`, `--model`, `--adapter` |
| 2 | **Execution** | `--from-step`, `--force`, `--dry-run`, `--timeout`, `--steps`, `--exclude`, `--on-failure`, `--detach` |
| 3 | **Continuous** | `--continuous`, `--source`, `--max-iterations`, `--delay` |
| 4 | **Dev/Debug** | `--mock`, `--preserve-workspace`, `--auto-approve`, `--no-retro`, `--force-model`, `--run`, `--manifest` |

In the TUI and WebUI, tiers 3 and 4 are collapsed by default to reduce visual noise.

## CLI

The most direct way to run a pipeline:

```bash
# Quick positional syntax
wave run impl-issue "fix the login bug"

# Explicit flags
wave run --pipeline impl-issue --input "fix the login bug" --model opus

# Selective steps
wave run impl-speckit --steps specify,plan --dry-run

# Continuous mode (process work items from a source)
wave run impl-issue --continuous --source "https://github.com/org/repo/issues" --delay 5m
```

Run `wave run --help` to see all flags grouped by tier.

## TUI

Launch the TUI with `wave` (no subcommand). Navigate to a pipeline, press Enter to open the run form. The form shows Tier 1 and Tier 2 options by default. Expand the "Advanced" section to access Tier 3 and Tier 4 options.

Key bindings in the run form:

| Key | Action |
|-----|--------|
| Tab | Move between fields |
| Enter | Submit the form |
| Esc | Cancel and return to pipeline list |

## WebUI

Start the dashboard with `wave serve` or `wave webui`. Navigate to a pipeline detail page to see the inline run form. The form groups options into collapsible tiers matching the CLI layout.

The WebUI also supports starting pipelines from PR pages via the integrated start button, which calls the `POST /api/prs/start` endpoint.

## Common Scenarios

**Resume a failed pipeline from a specific step:**

```bash
wave run impl-speckit --from-step implement --force
```

**Run with a different adapter and model:**

```bash
wave run impl-issue --adapter gemini --model gemini-2.0-pro
```

**Background execution that survives shell exit:**

```bash
wave run --detach impl-issue -- "https://github.com/org/repo/issues/42"
```

**Skip specific steps:**

```bash
wave run impl-speckit -x validate,publish
```

**Dry run to preview execution plan:**

```bash
wave run impl-speckit --dry-run -- "add user authentication"
```

See [CLI Reference](/reference/cli) for the full flag documentation.
