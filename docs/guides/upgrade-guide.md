# Upgrade Guide

This guide walks through the recommended workflow for upgrading Wave to a new version. The process preserves all your customizations (personas, pipelines, contracts, manifest settings) while adding new defaults introduced in the upgrade.

## Prerequisites

- An existing Wave project (previously initialized with `wave init`)
- Access to the new Wave binary (downloaded or built from source)
- Familiarity with your project's customizations in `wave.yaml` and `.agents/`

## Upgrade Workflow Overview

The upgrade process has four steps:

1. **Update the Wave binary** to the latest version
2. **Run `wave init --merge`** to sync scaffolding with new defaults
3. **Run `wave migrate up`** to apply pending database migrations
4. **Run `wave validate`** to verify configuration integrity

Each step is described in detail below with expected output examples.

## Step 1: Update the Wave Binary

Download and install the latest Wave release.

### Using the install script

```bash
curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
```

### Using Go

```bash
go install github.com/recinq/wave/cmd/wave@latest
```

### Verify the new version

```bash
wave --version
```

Expected output:

```
wave version 0.5.0
```

## Step 2: Run `wave init --merge`

The `--merge` flag tells Wave to add missing defaults without overwriting your existing files. Before modifying anything, it displays a change summary showing exactly what will happen.

```bash
wave init --merge
```

### Expected output

The command prints a categorized change summary to stderr, then prompts for confirmation:

```
  Change Summary:

  Personas:
    = up to date   .agents/personas/craftsman.md
    ~ preserved    .agents/personas/navigator.md
    + new          .agents/personas/auditor.md

  Pipelines:
    = up to date   .agents/pipelines/hello-world.yaml
    ~ preserved    .agents/pipelines/plan.yaml
    + new          .agents/pipelines/security-audit.yaml

  Contracts:
    + new          .agents/contracts/audit-report.schema.json

  Manifest (wave.yaml):
    + added        runtime.relay
    ~ preserved    runtime.max_concurrent_workers
    ~ preserved    adapters.ollama

  Apply these changes? [y/N]:
```

The categories mean:

| Symbol | Status | Meaning |
|--------|--------|---------|
| `+` | **new** | File does not exist in your project and will be created |
| `~` | **preserved** | File exists and differs from the default; your version is kept |
| `=` | **up to date** | File exists and matches the current default byte-for-byte |

### Review what will change

- **New files** (`+`) are defaults introduced in the new version. They will be created in your `.agents/` directory.
- **Preserved files** (`~`) are files you have customized. They will not be modified.
- **Up-to-date files** (`=`) already match the latest defaults. No action needed.
- **Manifest changes** show which `wave.yaml` keys will be added (new defaults) and which will be preserved (your existing values).

### Confirm or abort

Type `y` and press Enter to apply the changes, or `N` (or just press Enter) to abort without modifying any files. If you abort, you can re-run the command at any time.

### Already up to date

If your project already has all the latest defaults, the command exits immediately:

```
  Already up to date — no changes needed.
```

### Merge success output

After confirmation, Wave applies the changes and prints a success message:

```
  ╦ ╦╔═╗╦  ╦╔═╗
  ║║║╠═╣╚╗╔╝║╣
  ╚╩╝╩ ╩ ╚╝ ╚═╝
  Multi-Agent Pipeline Orchestrator

  Configuration merged successfully!

  Updated:
    wave.yaml       Preserved your settings
    Added missing default adapters and personas
    Created missing .agents/ directories and files

  Next steps:
    1. Run 'wave migrate up' to apply pending migrations
    2. Run 'wave validate' to check configuration
```

## Step 3: Run `wave migrate up`

After merging configuration, apply any pending database schema migrations:

```bash
wave migrate up
```

Expected output:

```
Migrations applied successfully
```

### Check migration status

To see detailed migration information before or after applying:

```bash
wave migrate status
```

Expected output:

```
Current schema version: 6

Migration Status:
================
[x] 1: Initial schema (applied 2025-11-15 10:30:00)
[x] 2: Add step retry tracking (applied 2025-11-15 10:30:00)
[x] 3: Add workspace paths (applied 2025-12-01 14:22:10)
[x] 4: Add artifact metadata (applied 2025-12-01 14:22:10)
[x] 5: Add pipeline input column (applied 2026-03-04 09:15:30)
[x] 6: Add recovery hints (applied 2026-03-04 09:15:30)

Database is up to date
```

### Validate migration integrity

To verify that applied migrations match their expected checksums:

```bash
wave migrate validate
```

Expected output:

```
Migration integrity check passed
```

## Step 4: Run `wave validate`

Verify that your configuration is structurally sound and all references resolve correctly:

```bash
wave validate
```

Expected output:

```
✓ Validation successful
```

For detailed output, use the `--verbose` flag:

```bash
wave validate --verbose
```

Expected output:

```
Validating manifest: wave.yaml
✓ Manifest syntax is valid
✓ Manifest structure is valid
✓ System references are valid
✓ Adapter configuration checked

Summary:
  Adapters:  2 defined
  Personas:  5 defined

✓ Validation successful
```

## Step 5: Verify Setup

Run a simple pipeline to confirm everything works end-to-end:

```bash
wave run ops-hello-world "upgrade test"
```

If this completes successfully, your upgrade is finished.

## Flag Reference

The `wave init` command supports several flags that control its behavior. The table below describes each flag and its effect.

### Individual flags

| Flag | Behavior |
|------|----------|
| `wave init` | On an existing project, prompts for confirmation before overwriting all files with defaults. |
| `wave init --merge` | Preserves existing files, adds missing defaults, prompts for confirmation. |
| `wave init --force` | Overwrites all files with defaults without prompting. |
| `wave init --yes` | Same as `--force` when used without `--merge`. Skips confirmation. |

### Combined flags

| Flags | Behavior |
|-------|----------|
| `wave init --merge --force` | Applies merge logic (preserves user files, adds missing defaults) without prompting. The change summary is still printed to stderr. |
| `wave init --merge --yes` | Identical to `--merge --force`. Skips the confirmation prompt while retaining merge-safe behavior. |

### Key distinction

The `--merge` flag is the primary mode selector. When `--merge` is present, it constrains the operation to merge behavior regardless of whether `--force` or `--yes` is also specified. This means `--merge --force` will never overwrite your files -- it only skips the confirmation prompt.

Without `--merge`, the `--force` flag causes a full overwrite of all files with defaults.

## CI/CD Usage

In non-interactive environments (CI runners, Docker containers, cron jobs), Wave requires `--yes` or `--force` to skip the confirmation prompt. Without one of these flags, `wave init --merge` will abort with an error:

```
Error: non-interactive terminal detected: use --yes or --force to proceed without confirmation
```

### Recommended CI upgrade command

```bash
wave init --merge --yes && wave migrate up && wave validate
```

This single command chain performs the full upgrade workflow non-interactively:

1. Merges defaults without prompting (prints summary to stderr for logging)
2. Applies pending database migrations
3. Validates the final configuration

### GitHub Actions example

```yaml
- name: Upgrade Wave
  run: |
    curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
    wave init --merge --yes
    wave migrate up
    wave validate
```

### GitLab CI example

```yaml
upgrade-wave:
  script:
    - curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
    - wave init --merge --yes
    - wave migrate up
    - wave validate
```

## Troubleshooting

### Malformed YAML in wave.yaml

If your `wave.yaml` contains invalid YAML syntax, the merge will abort without modifying any files:

```
Error: failed to parse existing manifest /path/to/wave.yaml: yaml: line 15: did not find expected key
```

**Fix**: Open `wave.yaml` and correct the syntax error at the reported line number. Common causes include incorrect indentation, missing colons after keys, and unquoted special characters. After fixing the syntax, re-run `wave init --merge`.

### Permission errors on .agents/ directory

If the `.agents/` directory or its contents have restrictive permissions, Wave will report the specific file it cannot write:

```
Error: failed to create directory /path/to/.agents/personas: permission denied
```

**Fix**: Ensure the current user has write permissions on the `.agents/` directory and its contents:

```bash
chmod -R u+w .agents/
```

Then re-run `wave init --merge`.

### Non-interactive terminal without --yes

Running `wave init --merge` in a terminal that is not interactive (no TTY attached) without `--yes` or `--force` results in:

```
Error: non-interactive terminal detected: use --yes or --force to proceed without confirmation
```

**Fix**: Add `--yes` or `--force` to the command:

```bash
wave init --merge --yes
```

### Database file does not exist

If `.agents/state.db` does not exist when running `wave migrate up`, the migration runner creates a fresh database and applies all migrations from scratch. This is normal for projects that have not yet run any pipelines.

### Migration integrity check failure

If `wave migrate validate` reports a checksum mismatch, it means an applied migration was modified after it was run. This typically indicates manual database edits or a corrupted state file.

**Fix**: Back up your current `.agents/state.db`, then recreate it:

```bash
mv .agents/state.db .agents/state.db.bak
wave migrate up
```

This creates a fresh database with all migrations applied cleanly. Note that pipeline execution history will be lost.

### Validation errors after merge

If `wave validate` fails after a merge, common causes include:

- **Missing persona prompt file**: A persona references a `system_prompt_file` that does not exist. Check that all persona `.md` files are present in `.agents/personas/`.
- **Unknown adapter reference**: A persona references an adapter not defined in `wave.yaml`. Add the missing adapter definition or update the persona.
- **Missing pipeline file**: A pipeline YAML file referenced in the project is not in `.agents/pipelines/`. Re-run `wave init --merge` to add missing defaults.

Run `wave validate --verbose` for detailed diagnostics.

## Next Steps

- [Pipeline Configuration](/guides/pipeline-configuration) - Configure and customize pipelines
- [State & Resumption](/guides/state-resumption) - Understand pipeline state persistence
- [CI/CD Integration](/guides/ci-cd) - Automate Wave in your CI workflows
