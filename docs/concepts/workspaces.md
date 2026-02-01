# Workspaces

Every pipeline step executes in an **ephemeral workspace** — an isolated directory that contains only what the step needs. Workspaces enforce the principle that steps cannot accidentally share state.

## Workspace Structure

```
/tmp/wave/<pipeline-id>/<step-id>/
├── src/              # Mounted from repository (readonly by default)
├── artifacts/        # Injected artifacts from dependency steps
├── output/           # Step's output artifacts
├── .claude/          # Adapter configuration
└── CLAUDE.md         # Persona system prompt
```

## Lifecycle

```mermaid
graph LR
    C[Create] --> M[Mount Sources]
    M --> I[Inject Artifacts]
    I --> E[Execute Step]
    E --> P[Persist Output]
    P --> W[Wait for Cleanup]
```

1. **Create** — Wave creates the workspace directory under `runtime.workspace_root`.
2. **Mount** — Source repository is mounted with the configured access mode.
3. **Inject** — Artifacts from completed dependency steps are copied in.
4. **Execute** — The adapter subprocess runs within this workspace.
5. **Persist** — Output artifacts are stored for downstream steps.
6. **Wait** — Workspace persists until `wave clean` is run. Never auto-deleted.

## Mount Configuration

```yaml
workspace:
  mount:
    - source: ./                # Project root
      target: /src
      mode: readonly            # Step cannot modify source
    - source: ./test-fixtures
      target: /fixtures
      mode: readonly
    - source: ./output
      target: /out
      mode: readwrite           # Step can write here
```

### Access Modes

| Mode | Description |
|------|-------------|
| `readonly` | Step can read but not modify. Default for source code mounts. |
| `readwrite` | Step can read and modify. Use for output directories. |

Navigator and auditor personas typically use `readonly` mounts. Craftsman personas need `readwrite` access to implementation directories.

## Workspace Isolation Guarantees

- **No shared state** — steps cannot see each other's workspaces.
- **Fresh on retry** — when a step retries, it gets a clean workspace.
- **Artifacts are copies** — injected artifacts are copied, not linked. Modifying an artifact in one step doesn't affect the original.
- **Persona-scoped config** — each workspace gets its own `CLAUDE.md` based on the bound persona.

## Workspace Root Configuration

```yaml
# In wave.yaml
runtime:
  workspace_root: /tmp/wave          # Default

# Override with environment variable
# WAVE_WORKSPACE_ROOT=/data/wave

# Override with CLI flag
# wave run --workspace /data/wave
```

### Disk Usage

Workspaces accumulate until explicitly cleaned:

```bash
# Check disk usage
du -sh /tmp/wave/

# Clean specific pipeline
wave clean --pipeline-id a1b2c3d4

# Clean everything
wave clean --all

# Clean old workspaces
wave clean --older-than 7d
```

## Debugging with Workspaces

Since workspaces persist, they're useful for debugging failed steps:

```bash
# Find the failed step's workspace
ls /tmp/wave/<pipeline-id>/

# Inspect artifacts
cat /tmp/wave/<pipeline-id>/navigate/output/analysis.json

# Check what the agent saw
cat /tmp/wave/<pipeline-id>/implement/CLAUDE.md

# Review injected artifacts
ls /tmp/wave/<pipeline-id>/implement/artifacts/
```

## Further Reading

- [Pipeline Schema — WorkspaceConfig](/reference/pipeline-schema#workspaceconfig) — field reference
- [Pipelines](/concepts/pipelines) — how workspaces fit into the execution model
- [State & Resumption](/guides/state-resumption) — workspace persistence across resumes
