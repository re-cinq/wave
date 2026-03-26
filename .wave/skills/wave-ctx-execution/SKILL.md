---
name: wave-ctx-execution
description: Domain context for Wave's pipeline execution bounded context
---

# Execution Context

Pipeline execution, adapter subprocess management, workspace lifecycle, and worktree isolation.

## Invariants

- Fresh memory at every step boundary -- no chat history inheritance between steps (memory.strategy defaults to "fresh" in DAG loader)
- Contract validation must complete before a step is marked successful; hard failures block, soft failures log warnings
- Steps execute in topological order; the DAG validator rejects cycles and missing dependencies before any step runs
- Each step runs in an isolated workspace at `.wave/workspaces/<pipeline>/<step>/`; worktree workspaces get a real `.git` file, mount-based workspaces get `git init -q` to anchor path resolution
- Adapter subprocesses run in their own process group (`Setpgid: true`); cancellation sends SIGTERM then SIGKILL after 3 seconds
- Artifacts from prior steps are injected into `.wave/artifacts/` before the persona prompt fires; existence, optionality, and schema validation are enforced at injection time
- A runtime CLAUDE.md is assembled per step from four layers: base protocol, persona system prompt, contract compliance section, restriction section
- `rework_only` steps are excluded from normal DAG scheduling and only execute when triggered by a rework policy
- Concurrent steps share a mutex-protected `PipelineContext` for artifact path registration

## Key Decisions

- Single adapter interface (`AdapterRunner.Run`) supports Claude, OpenCode, browser, interactive, and mock backends -- the executor is adapter-agnostic
- Pipeline context uses double-brace template resolution (`{{ forge.cli_tool }}`, `{{ project.test_command }}`, `{{ artifacts.<name> }}`) with a final regex sweep to strip unresolved `{{ project.* }}` and `{{ ontology.* }}` placeholders
- Resume creates a subpipeline with dependencies on prior steps stripped to avoid DAG validation failure; artifact paths resolve from the most recent prior run's workspace
- `maxStdoutTailChars = 2000` caps retry/rework context injection to prevent prompt bloat
- Composition primitives (sub-pipeline, iterate, branch, gate, loop, aggregate) are declared on the Step struct and detected via `IsCompositionStep()`
- Matrix strategy with `stacked: true` chains worktrees sequentially, passing the prior branch as the next step's base

## Domain Vocabulary

| Term | Meaning |
|------|---------|
| Pipeline | A topologically-sorted DAG of Steps, loaded from `.wave/pipelines/<name>.yaml` |
| Step | A single unit of work: one persona, one workspace, one prompt, zero or more output artifacts |
| Persona | A named AI agent configuration (adapter, system prompt, permissions, model, temperature) |
| Adapter | A subprocess wrapper (Claude Code, OpenCode, browser, mock) that implements `AdapterRunner` |
| Workspace | An ephemeral directory where a step executes; supports mount, worktree, and basic modes |
| Worktree | A git worktree created under `.wave/workspaces/` for full git context isolation |
| Artifact | A file produced by a step (`output_artifacts`) and optionally consumed by downstream steps (`inject_artifacts`) |
| Handover | The boundary between steps: contract validation, compaction trigger, and artifact registration |
| Contract | A validation rule applied to step output (JSON Schema, test suite, TypeScript interface, markdown spec, format, non-empty file) |
| Relay | Context compaction triggered when token usage exceeds a threshold; a summarizer persona compresses prior context |
| Gate | A blocking step that waits for an external signal (approval, PR merge, CI pass, timer) |
| Run ID | A unique identifier for a pipeline execution: `<name>-<timestamp>-<4-char-hash>` |
| Rework | When a step fails with `on_failure: rework`, the executor schedules a designated `rework_only` step instead of retrying inline |

## Neighboring Contexts

- **Validation** (`internal/contract/`, `internal/preflight/`) -- the executor delegates output validation to contract validators and pre-run dependency checks to preflight
- **Security** (`internal/security/`) -- the executor instantiates `PathValidator`, `InputSanitizer`, and `SecurityLogger` from `SecurityConfig`
- **Configuration** (`internal/manifest/`) -- the executor reads personas, adapters, runtime config, and project vars from the manifest
- **State** (`internal/state/`) -- step lifecycle transitions (pending, running, completed, failed, retrying, skipped, reworking) are persisted to SQLite
- **Display** (`internal/display/`, `internal/event/`) -- the executor emits structured progress events consumed by TUI, text, and JSON renderers

## Key Files

- `internal/pipeline/executor.go` -- main execution loop, step orchestration, artifact injection, CLAUDE.md assembly
- `internal/pipeline/types.go` -- `Pipeline`, `Step`, `RetryConfig`, `WorkspaceConfig`, `ArtifactDef`, `HandoverConfig` structs
- `internal/pipeline/dag.go` -- `DAGValidator.ValidateDAG()`, `TopologicalSort()`, cycle detection, rework target validation
- `internal/pipeline/context.go` -- `PipelineContext`, template resolution, forge/project/ontology variable injection
- `internal/pipeline/resume.go` -- `ResumeManager`, subpipeline creation, stale artifact detection
- `internal/pipeline/concurrency.go` -- concurrent step scheduling within DAG ordering
- `internal/pipeline/composition.go` -- sub-pipeline, iterate, branch, gate, loop, aggregate execution
- `internal/adapter/adapter.go` -- `AdapterRunner` interface, `ProcessGroupRunner`, process group lifecycle
- `internal/adapter/claude.go` -- Claude Code adapter with NDJSON stream parsing and `--print` / `--output-format stream-json` flags
- `internal/workspace/workspace.go` -- `WorkspaceManager` interface, mount creation, artifact injection
- `internal/worktree/worktree.go` -- `Manager.Create()`, `Manager.Remove()`, git worktree lifecycle
- `internal/pipeline/template.go` -- project and ontology variable injection from manifest
