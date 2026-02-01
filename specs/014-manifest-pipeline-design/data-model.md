# Data Model: Manifest & Pipeline Design

**Branch**: `014-manifest-pipeline-design`
**Date**: 2026-02-01

## Entity: Manifest

The top-level configuration file (`wave.yaml`).

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| apiVersion | string | yes | Schema version (e.g., "v1") |
| kind | string | yes | Must be "WaveManifest" |
| metadata.name | string | yes | Project name |
| metadata.description | string | no | Project description |
| metadata.repo | string | no | Repository URL |
| adapters | map[string]Adapter | yes | Named adapter configurations |
| personas | map[string]Persona | yes | Named persona configurations |
| runtime | Runtime | yes | Global runtime settings |
| skill_mounts | []SkillMount | no | Skill discovery paths |

**Validation rules**:
- Every persona must reference a defined adapter.
- Every persona's `system_prompt_file` must exist on disk.
- Every hook command script must exist on disk.
- Adapter binary must be resolvable on PATH (warning, not error).

## Entity: Adapter

Wraps a specific LLM CLI.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| binary | string | yes | CLI binary name (must be on PATH) |
| mode | string | yes | "headless" (always subprocess, never interactive) |
| output_format | string | no | Default "json" |
| project_files | []string | no | Files projected into workspace |
| default_permissions | Permissions | no | Default tool permissions |
| hooks_template | string | no | Directory for hook script templates |

## Entity: Persona

Agent configuration binding an adapter to a role.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| adapter | string | yes | References a key in `adapters` |
| description | string | no | Human-readable purpose |
| system_prompt_file | string | yes | Path to persona's system prompt markdown |
| temperature | float | no | LLM temperature (0.0-1.0) |
| permissions | Permissions | no | Overrides adapter defaults |
| hooks | HookConfig | no | PreToolUse/PostToolUse hook definitions |

## Entity: Permissions

Tool access control for a persona.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| allowed_tools | []string | no | Glob patterns for allowed tools |
| deny | []string | no | Glob patterns for denied tools (takes precedence) |

**Evaluation order**: Deny patterns are checked first. If any deny
pattern matches, the tool call is blocked regardless of allowed_tools.

## Entity: HookConfig

Pre/post tool use hooks.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| PreToolUse | []HookRule | no | Hooks that fire before a tool call |
| PostToolUse | []HookRule | no | Hooks that fire after a tool call |

## Entity: HookRule

A single hook binding.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| matcher | string | yes | Glob pattern matching tool calls |
| command | string | yes | Shell command to execute |

**Behavior**: If command exits non-zero on PreToolUse, the tool call
is blocked. PostToolUse hooks are informational (exit code logged but
does not block).

## Entity: Runtime

Global runtime configuration.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| workspace_root | string | no | Default "/tmp/wave" |
| max_concurrent_workers | int | no | Default 5 |
| default_timeout_minutes | int | no | Default 30 |
| relay.token_threshold_percent | int | no | Default 80 |
| relay.strategy | string | no | Default "summarize_to_checkpoint" |
| audit.log_dir | string | no | Default ".wave/traces/" |
| audit.log_all_tool_calls | bool | no | Default false |
| audit.log_all_file_operations | bool | no | Default false |
| meta_pipeline.max_depth | int | no | Default 2 |
| meta_pipeline.max_total_steps | int | no | Default 20 |
| meta_pipeline.max_total_tokens | int | no | Default 500000 |
| meta_pipeline.timeout_minutes | int | no | Default 60 |

## Entity: Pipeline

A DAG of steps defined in a YAML file.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| kind | string | yes | Must be "WavePipeline" |
| metadata.name | string | yes | Pipeline name |
| metadata.description | string | no | Pipeline description |
| input | InputConfig | no | Work item source configuration |
| steps | []Step | yes | Ordered list of pipeline steps |

## Entity: Step

A single unit of work in a pipeline.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | Unique step identifier |
| persona | string | yes | References a persona in the manifest |
| dependencies | []string | no | Step IDs that must complete first |
| memory.strategy | string | yes | "fresh" (always) |
| memory.inject_artifacts | []ArtifactRef | no | Artifacts to inject from prior steps |
| workspace.root | string | yes | Workspace directory path template |
| workspace.mount | []Mount | no | Filesystem mounts |
| exec.type | string | yes | "prompt" or "command" |
| exec.source | string | yes | Prompt template or shell command |
| output_artifacts | []ArtifactDef | no | Expected output files/directories |
| handover | HandoverConfig | no | Contract and compaction settings |
| strategy | MatrixStrategy | no | Fan-out configuration |
| validation | []ValidationRule | no | Pre-execution validation checks |

**State transitions**:
```
Pending → Running → Completed
                  → Failed (max retries exceeded)
                  → Retrying (contract fail, crash, or timeout)
                      → Running (retry attempt)
```

## Entity: HandoverConfig

Contract and compaction settings at step boundary.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| contract.type | string | no | "json_schema", "typescript_interface", "test_suite" |
| contract.schema | string | no | Inline schema or file path |
| contract.source | string | no | File to validate |
| contract.validate | bool | no | Whether to compile-check |
| contract.command | string | no | Test command (for test_suite type) |
| contract.must_pass | bool | no | Block on failure |
| contract.on_failure | string | no | "retry" or "halt" |
| contract.max_retries | int | no | Default 2 |
| compaction.trigger | string | no | "token_limit_80%" |
| compaction.persona | string | no | Summarizer persona name |

## Entity: PipelineState (persisted in SQLite)

Runtime state for pipeline resumption.

| Field | Type | Description |
|-------|------|-------------|
| pipeline_id | string (PK) | UUID for this pipeline instance |
| pipeline_name | string | Name from pipeline YAML |
| status | string | "queued", "running", "completed", "failed" |
| created_at | timestamp | When the pipeline was started |
| updated_at | timestamp | Last state change |
| input | text | The input that triggered this pipeline |

## Entity: StepState (persisted in SQLite)

Per-step runtime state.

| Field | Type | Description |
|-------|------|-------------|
| step_id | string (PK) | Step ID from pipeline YAML |
| pipeline_id | string (FK) | Parent pipeline |
| state | string | "pending", "running", "completed", "failed", "retrying" |
| retry_count | int | Current retry attempt |
| started_at | timestamp | When step began executing |
| completed_at | timestamp | When step finished (success or failure) |
| workspace_path | string | Ephemeral workspace directory |
| error_message | text | Last error if failed/retrying |

## Relationships

```
Manifest 1──* Adapter
Manifest 1──* Persona
Persona *──1 Adapter
Pipeline 1──* Step
Step *──1 Persona (via manifest lookup)
Step *──* Step (dependencies, DAG edges)
PipelineState 1──* StepState
```
