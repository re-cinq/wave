# Pipeline Execution

Wave transforms declarative pipeline configurations into orchestrated AI execution. This page explains how Wave executes your pipelines, from configuration parsing to artifact delivery.

## Execution Overview

When you run a pipeline:

```bash
wave run code-review "Review authentication changes"
```

Wave performs these phases:

1. **Parse** - Load and validate the pipeline configuration
2. **Plan** - Build the execution graph from step dependencies
3. **Execute** - Run each step in the correct order
4. **Validate** - Check outputs against contracts
5. **Handover** - Pass artifacts to dependent steps

## Configuration Parsing

Wave loads the pipeline definition and validates it against the schema:

```yaml
kind: WavePipeline
metadata:
  name: code-review
  description: "Automated code review"

input:
  source: cli

steps:
  - id: analyze
    persona: navigator
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: "Analyze: {{ input }}"
    output_artifacts:
      - name: analysis
        path: output/analysis.json
        type: json
```

**Validation checks:**
- YAML syntax correctness
- Required fields present (`kind`, `metadata.name`, `steps`)
- Step IDs are unique
- Dependencies reference valid step IDs
- Personas exist in the manifest
- Contract schemas are accessible

Run validation manually:

```bash
wave validate
```

## Dependency Graph Resolution

Wave builds a directed acyclic graph (DAG) from step dependencies:

```yaml
steps:
  - id: analyze        # No dependencies

  - id: security       # Depends on analyze
    dependencies: [analyze]

  - id: quality        # Depends on analyze (parallel with security)
    dependencies: [analyze]

  - id: summary        # Waits for both
    dependencies: [security, quality]
```

This produces:

```
analyze ─┬─> security ─┬─> summary
         └─> quality  ─┘
```

Wave determines:
- **Execution order**: Which steps can run first
- **Parallelization opportunities**: `security` and `quality` run concurrently
- **Critical path**: The longest dependency chain

## Workspace Provisioning

Each step executes in an isolated workspace:

```
.wave/workspaces/<run-id>/<step-id>/
├── input/          # Injected artifacts from dependencies
├── workspace/      # Working directory for the persona
└── output/         # Step outputs for contract validation
```

**Workspace guarantees:**
- **Isolation**: Steps cannot access each other's workspaces directly
- **Clean slate**: Each execution starts fresh
- **Mount configuration**: Source files mounted per step config

```yaml
workspace:
  mount:
    - source: ./src
      target: /code
      mode: readonly
```

## Memory Strategy

The `memory.strategy` configuration controls what context a step receives:

### Fresh Memory (Default)

```yaml
memory:
  strategy: fresh
```

The step starts with no conversation history. This ensures:
- **Reproducibility**: Same inputs produce same outputs
- **Isolation**: Previous step failures don't contaminate
- **Security**: No unintended context leakage

### Artifact Injection

```yaml
memory:
  strategy: fresh
  inject_artifacts:
    - step: analyze
      artifact: analysis
      as: context
```

Artifacts from previous steps are explicitly injected:
- Wave retrieves the artifact from the dependency step
- Artifact content is included in the step's context
- Referenced via template: `{{ artifacts.context }}`

## Step Execution

Wave executes each step through the configured adapter:

```yaml
steps:
  - id: review
    persona: auditor
    exec:
      type: prompt
      source: |
        Review the code changes:

        Context: {{ artifacts.context }}

        Provide structured feedback.
```

**Execution process:**

1. **Workspace setup**: Create isolated directory, mount files
2. **Artifact injection**: Load declared artifacts into context
3. **Persona initialization**: Configure adapter with persona settings
4. **Prompt execution**: Run the AI with the configured prompt
5. **Output collection**: Gather files from output directory

The adapter (e.g., Claude) runs with:
- System prompt from persona configuration
- Permissions from persona configuration
- Tools allowed by persona permissions

## Contract Validation

After step completion, Wave validates outputs:

```yaml
handover:
  contract:
    type: jsonschema
    schema_path: .wave/contracts/review.schema.json
    on_failure: retry
    max_retries: 2
```

**Validation flow:**

1. Locate output file (from `output_artifacts` or contract `source`)
2. Apply contract validation
3. **Pass**: Mark step complete, make artifacts available
4. **Fail**: Execute failure policy

**Failure policies:**

- `on_failure: retry` - Re-execute step with fresh workspace
- `on_failure: halt` - Stop pipeline execution

## State Management

Wave persists execution state in SQLite:

```sql
-- Pipeline runs
pipeline_runs (id, pipeline_name, status, started_at, completed_at)

-- Step executions
step_executions (run_id, step_id, status, workspace_path, started_at)

-- Artifacts
artifacts (step_execution_id, name, path, checksum)
```

This enables:

**Status queries:**
```bash
wave status
wave list
```

**Pipeline resumption:**
```bash
wave resume <run-id>
```

**Artifact inspection:**
```bash
wave artifacts <run-id>
```

## Execution Flow Example

A complete code review execution:

```
1. wave run code-review "Review auth changes"

2. Parse configuration
   ✓ Pipeline: code-review
   ✓ Steps: analyze → security, quality → summary

3. Create run: run-abc123

4. Execute step: analyze
   → Workspace: .wave/workspaces/run-abc123/analyze/
   → Persona: navigator
   → Mount: ./src → /code (readonly)
   → Execute prompt
   → Output: output/analysis.json
   → Contract: jsonschema validation ✓
   → Artifacts available: [analysis]

5. Execute steps: security, quality (parallel)
   → Both inject: [analysis from analyze]
   → Both execute with fresh memory
   → Contract validation on each

6. Execute step: summary
   → Inject: [security-report, quality-report]
   → Generate final summary
   → Contract validation ✓

7. Pipeline complete: run-abc123
```

## CLI Commands

### Running Pipelines

```bash
# Run a pipeline
wave run code-review "Review the authentication module"

# Run with specific input
wave run code-review --input "path/to/diff.patch"
```

### Monitoring Execution

```bash
# View all runs
wave list

# Check specific run status
wave status <run-id>

# View step logs
wave logs <run-id>
wave logs <run-id> --step analyze
```

### Pipeline Control

```bash
# Resume interrupted pipeline
wave resume <run-id>

# Cancel running pipeline
wave cancel <run-id>
```

### Artifacts and Cleanup

```bash
# View artifacts from a run
wave artifacts <run-id>

# Clean up completed runs
wave clean
wave clean --all
```

## Ad-hoc Execution

For single-step tasks without a full pipeline:

```bash
wave do navigator "Analyze the authentication flow in src/auth"
```

This runs a single persona with fresh memory, useful for:
- Quick exploration tasks
- One-off analysis
- Testing persona configurations

## Observability

Wave provides visibility into execution through:

**Structured logging:**
```
[2025-01-15 10:30:00] STEP_START: analyze (persona=navigator)
[2025-01-15 10:32:15] CONTRACT_PASS: analysis.json validated
[2025-01-15 10:32:15] STEP_COMPLETE: analyze (2m15s)
```

**Audit trails:**
```bash
# Logs stored in configured audit directory
.wave/traces/<run-id>/
├── step-analyze.log
├── step-security.log
└── step-summary.log
```

**Progress events:**
- Step started/completed/failed
- Contract validation results
- Artifact creation
- Pipeline completion

## Error Handling

### Step Failures

When a step fails:

1. Error details logged
2. Failure policy applied (retry or halt)
3. State persisted for debugging

```bash
# Check what failed
wave status <run-id>

# View step logs
wave logs <run-id> --step <failed-step>
```

### Contract Failures

When contract validation fails:

1. Specific validation errors logged
2. If `on_failure: retry`, step re-executes with fresh workspace
3. After max retries, step marked failed

### Pipeline Resumption

If a pipeline is interrupted:

```bash
# See current state
wave status <run-id>

# Resume from last successful step
wave resume <run-id>
```

Wave skips completed steps and resumes execution.

## Next Steps

- [Pipelines](/concepts/pipelines) - The paradigm behind pipeline design
- [Contracts](/concepts/contracts) - Output validation
- [Personas](/concepts/personas) - Configuring AI agents
- [Creating Pipelines](/guide/pipelines) - Build your first pipeline
