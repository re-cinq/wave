# Execution

Pipeline execution transforms your YAML configuration into orchestrated AI steps. Wave handles dependency resolution, workspace isolation, and artifact passing automatically.

```bash
wave run --pipeline code-review --input "Review auth changes"
```

Use `wave run` when you need to execute a multi-step AI workflow with guaranteed output validation.

## Execution Phases

When you run a pipeline, Wave performs:

1. **Parse** - Load and validate the pipeline YAML
2. **Plan** - Build the dependency graph
3. **Execute** - Run each step with its persona
4. **Validate** - Check outputs against contracts
5. **Handover** - Pass artifacts to dependent steps

## Dependency Resolution

Wave builds a directed acyclic graph (DAG) from step dependencies:

```yaml
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze: {{ input }}"

  - id: security
    dependencies: [analyze]
    persona: auditor
    exec:
      type: prompt
      source: "Security review"

  - id: quality
    dependencies: [analyze]
    persona: auditor
    exec:
      type: prompt
      source: "Quality review"
```

Steps without mutual dependencies (`security` and `quality`) run in parallel.

## Workspace Isolation

Each step executes in an isolated workspace:

```
.wave/workspaces/<run-id>/<step-id>/
├── artifacts/    # Injected from dependencies
├── src/          # Mounted source (readonly)
└── output/       # Step outputs
```

**Guarantees:**
- Steps cannot access each other's workspaces directly
- Each execution starts with a clean slate
- Fresh memory at every step boundary

## Step States

| State | Description |
|-------|-------------|
| `pending` | Waiting for dependencies |
| `running` | Currently executing |
| `completed` | Finished successfully |
| `retrying` | Failed, attempting retry |
| `failed` | Max retries exceeded |

## Monitoring Runs

```bash
# Check run status
wave status

# View specific run
wave status run-abc123

# View step logs
wave logs run-abc123 --step analyze
```

**Output:**
```
RUN_ID          PIPELINE      STATUS     STEP        ELAPSED
run-abc123      code-review   running    security    2m15s
```

## Resuming Interrupted Runs

```bash
wave resume run-abc123
```

Wave skips completed steps and resumes from the last successful checkpoint.

## Error Handling

When a step fails:

1. Error details are logged
2. Failure policy executes (`retry` or `halt`)
3. State is persisted for debugging

```bash
# Diagnose failures
wave logs run-abc123 --errors
wave artifacts run-abc123 --step analyze
```

## Next Steps

- [Pipelines](/concepts/pipelines) - Define multi-step workflows
- [Contracts](/concepts/contracts) - Configure output validation
- [CLI Reference](/reference/cli) - Complete command documentation
