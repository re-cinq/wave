# Wave Ops Commands Quickstart

This guide provides a quick introduction to Wave's operational commands for managing, monitoring, and maintaining your pipelines.

## Overview

Wave Ops Commands give you visibility and control over pipeline execution:

| Command | Purpose |
|---------|---------|
| `wave status` | View running and recent pipelines |
| `wave logs` | View pipeline output and debug issues |
| `wave clean` | Free disk space by removing old workspaces |
| `wave list` | Discover available pipelines, personas, and adapters |
| `wave cancel` | Stop a running pipeline |
| `wave artifacts` | Inspect and export pipeline outputs |

## Quick Examples

### wave status - View Running Pipelines

Check what pipelines are currently running or have recently completed:

```bash
# Show currently running pipelines
wave status

# Show all recent pipelines (running, completed, failed)
wave status --all

# Get detailed info for a specific run
wave status abc123
```

Output includes pipeline name, current step, elapsed time, and token usage.

### wave logs - View Pipeline Output

Access execution logs for debugging and review:

```bash
# View logs from the most recent pipeline run
wave logs

# Follow logs in real-time for a running pipeline
wave logs --follow

# View only a specific step's output
wave logs --step investigate

# Show only errors and failed validations
wave logs --errors

# View logs for a specific run ID
wave logs abc123
```

### wave clean - Cleanup Workspaces

Remove old workspaces to free disk space:

```bash
# Preview what would be deleted (recommended first step)
wave clean --dry-run

# Remove all workspaces (prompts for confirmation)
wave clean --all

# Keep only the 5 most recent workspaces per pipeline
wave clean --keep-last 5

# Clean only a specific pipeline's workspaces
wave clean --pipeline debug

# Skip confirmation prompt
wave clean --all --force
```

### wave list - Discover Pipelines and Personas

Explore what's available in your Wave configuration:

```bash
# List all available pipelines
wave list pipelines

# List configured personas with their tools
wave list personas

# List available adapters
wave list adapters

# Output as JSON for scripting
wave list pipelines --format json
```

### wave cancel - Stop Running Pipelines

Stop a pipeline that's taking too long or was started incorrectly:

```bash
# Gracefully cancel (waits for current step to complete)
wave cancel

# Force immediate stop
wave cancel --force

# Cancel a specific run
wave cancel abc123
```

### wave artifacts - Inspect Outputs

View and export artifacts produced by pipeline steps:

```bash
# List all artifacts from the most recent run
wave artifacts

# List artifacts from a specific step
wave artifacts --step execute

# Export all artifacts to a directory
wave artifacts --export ./output

# Export artifacts from a specific run
wave artifacts abc123 --export ./output
```

## Common Workflows

### Debugging a Failed Pipeline

When a pipeline fails, use these steps to diagnose the issue:

```bash
# 1. Check the status to see which step failed
wave status

# 2. View the error messages
wave logs --errors

# 3. Look at the full output from the failed step
wave logs --step <failed-step-name>

# 4. Inspect any partial artifacts that were created
wave artifacts

# 5. After fixing the issue, resume from the failed step
wave run --pipeline <name> --from-step <failed-step>
```

### Freeing Up Disk Space

Workspaces accumulate over time. Here's how to manage them:

```bash
# 1. First, see what would be cleaned up
wave clean --dry-run

# 2. Option A: Keep only recent runs
wave clean --keep-last 3

# 3. Option B: Clean a specific pipeline that generated large artifacts
wave clean --pipeline data-processing

# 4. Option C: Clean everything and start fresh
wave clean --all --force
```

### Monitoring a Long-Running Pipeline

For pipelines that take a while, monitor progress:

```bash
# 1. Start the pipeline
wave run --pipeline long-running-task --input "..."

# 2. In another terminal, check status
wave status

# 3. Follow logs in real-time
wave logs --follow

# 4. If something goes wrong, cancel gracefully
wave cancel

# 5. Or force stop if unresponsive
wave cancel --force
```

## Tips and Gotchas

### General Tips

- **Use `--dry-run` first**: Both `wave clean` and `wave run` support `--dry-run`. Use it to preview actions before executing.

- **JSON output for scripting**: Most commands support `--format json` for integration with scripts and other tools.

- **Tab completion**: Wave supports shell completion. Run `wave completion --help` for setup instructions.

### Common Gotchas

1. **Logs disappear after cleanup**: Running `wave clean` removes workspace directories, including logs. Export or save important logs before cleaning.

2. **Cancel vs Force Cancel**: A regular `wave cancel` waits for the current step to complete. Use `--force` only if the step is truly stuck, as it may leave artifacts in an inconsistent state.

3. **Status shows "running" but nothing is happening**: The pipeline process may have crashed. Use `wave cancel --force` to reset the state, then check logs for the cause.

4. **Artifacts not found**: Artifacts are stored in workspace directories. If you've run `wave clean`, older artifacts will be gone. Consider using `--keep-last N` to retain recent runs.

5. **Permission errors during cleanup**: Wave respects workspace isolation. Ensure you have write permissions to the `.wave/` directory.

### Performance Considerations

- `wave status` reads from the local SQLite database and should return instantly (<100ms)
- `wave logs --follow` streams output with minimal latency (~500ms)
- `wave clean` processes workspaces efficiently, even with 1000+ directories

## Next Steps

- Run `wave <command> --help` for detailed usage information
- Check the full specification at `specs/016-wave-ops-commands/spec.md`
- See `wave.yaml` for pipeline and persona configuration examples
