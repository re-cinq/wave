# CLI Reference

The Muzzle CLI provides commands for managing projects and running pipelines.

## Global Flags

- `--help, -h`: Show help
- `--version, -v`: Show version

## Commands

### init

Initialize a new Muzzle project.

```bash
muzzle init [flags]
```

**Flags**:
- `--adapter string`      Default adapter to use (default "claude")
- `--persona string`      Initial persona to create (default "craftsman")
- `--workspace string`    Workspace root directory (default "/tmp/muzzle")

**Creates**:
- `muzzle.yaml` - Manifest file
- `.muzzle/personas/` - Persona system prompts
- `.muzzle/pipelines/` - Example pipelines

**Examples**:
```bash
# Initialize with defaults
muzzle init

# Initialize with specific adapter
muzzle init --adapter claude

# Initialize for fullstack development
muzzle init --persona fullstack
```

### validate

Validate Muzzle configuration files.

```bash
muzzle validate [flags]
```

**Flags**:
- `--manifest string`     Path to manifest (default "muzzle.yaml")
- `--verbose`            Show detailed validation errors

**Validates**:
- Manifest YAML syntax
- All persona system prompt files exist
- All adapter binaries are on PATH
- All hook scripts exist
- Pipeline YAML files have valid DAGs

**Exit Codes**:
- 0: Validation passed
- 1: Validation failed
- 2: Manifest not found

### run

Execute a pipeline.

```bash
muzzle run [flags]
```

**Required Flags**:
- `--pipeline string`     Path to pipeline YAML file

**Optional Flags**:
- `--input string`       Input prompt for the pipeline
- `--dry-run`           Walk pipeline without invoking adapters
- `--from-step string`    Resume from specific step

**Examples**:
```bash
# Run a named pipeline
muzzle run --pipeline .muzzle/pipelines/speckit-flow.yaml --input "add user auth"

# Dry run to validate structure
muzzle run --pipeline .muzzle/pipelines/speckit-flow.yaml --dry-run

# Resume from step
muzzle run --pipeline .muzzle/pipelines/speckit-flow.yaml --from-step "implement"
```

### do

Execute an ad-hoc task with a generated 2-step pipeline.

```bash
muzzle do "task description" [flags]
```

**Optional Flags**:
- `--persona string`      Persona to use for execution (default "craftsman")
- `--save string`         Save generated pipeline to file

**Examples**:
```bash
# Quick fix
muzzle do "fix typo in README"

# With specific persona
muzzle do "check security" --persona auditor

# Save generated pipeline
muzzle do "add feature" --save my-pipeline.yaml
```

### resume

Resume an interrupted pipeline.

```bash
muzzle resume [flags]
```

**Required Flags**:
- `--pipeline-id string`  UUID of the pipeline to resume

**Optional Flags**:
- `--from-step string`    Resume from specific step

### clean

Clean up ephemeral workspaces.

```bash
muzzle clean [flags]
```

**Optional Flags**:
- `--pipeline-id string`  Clean only specific pipeline's workspace
- `--all`                Clean all workspaces

## Output Format

All commands emit structured events to stdout:

```json
{
  "timestamp": "2026-02-01T10:00:00Z",
  "pipeline_id": "123",
  "step_id": "navigate",
  "state": "running",
  "duration_ms": 0,
  "message": "Starting step navigate"
}
```

**States**:
- `pending`: Step queued to run
- `running`: Step currently executing
- `completed`: Step finished successfully
- `failed`: Step failed after retries
- `retrying`: Step retrying after failure

## Exit Codes

- `0`: Success
- `1`: General error
- `2`: Usage error (invalid flags, missing args)
- `3`: Pipeline failed (step exceeded max retries)
- `4`: Validation error
- `130`: Interrupted (Ctrl+C)

## Environment Variables

- `MUZZLE_DEBUG`: Enable debug logging
- `MUZZLE_WORKSPACE_ROOT`: Override default workspace root
- `MUZZLE_LOG_FORMAT`: Output format (`json` or `text`)