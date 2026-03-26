---
name: wave-ctx-execution
description: Domain context for Pipeline execution engine — DAG resolution and topological sorting, step orchestration with retry/rework/skip policies, workspace lifecycle (worktree, mount, basic), adapter subprocess management with process group isolation, cross-pipeline artifact flow, and resume/recovery.
---

# Execution Context

Pipeline execution engine — DAG resolution and topological sorting, step orchestration with retry/rework/skip policies, workspace lifecycle (worktree, mount, basic), adapter subprocess management with process group isolation, cross-pipeline artifact flow, and resume/recovery.

## Invariants

- DAG must be acyclic — cycle detection via DFS with recursion stack rejects circular dependencies before execution begins
- All step dependencies must reference existing step IDs within the same pipeline
- Memory strategy defaults to 'fresh' for every step — constitutional requirement ensuring no chat history crosses step boundaries
- Deadlock detection: if no steps are ready but work remains, execution fails with 'deadlock: no steps ready but N remain'
- Persona referenced by a step must exist in the manifest — hard error 'persona %q not found in manifest'
- Adapter referenced by a persona must exist in the manifest — hard error 'adapter %q not found in manifest'
- Workspace ref must reference an already-executed step — prevents forward references in workspace sharing
- Required artifacts must be found — optional artifacts silently skipped, required ones cause hard error
- Cross-pipeline artifacts must exist unless marked optional — validated at injection time
- Artifact type mismatch is a hard error — declared type must match expected type
- Schema validation on injected artifacts — inputs validated against ref.SchemaPath before step execution
- Stdout artifact size enforced — exceeding runtime.artifacts.max_stdout_size is a hard error
- Rate limit causes immediate step failure — no retry, returns 'adapter rate limited' error
- Contract validation with must_pass:true blocks pipeline on failure — soft failures only emit warning events
- Context cancellation propagated — pipeline and step-level cancellation via context.Context
- Rework target must exist, must not be self-referential, and must be marked rework_only:true
- Each rework target must be unique across all steps — prevents concurrent rework race conditions
- on_failure values constrained to enum: fail, skip, continue, rework
- Concurrency and matrix strategy are mutually exclusive on a step
- rework_step required when on_failure is 'rework', forbidden otherwise
- ArtifactRef Step and Pipeline fields are mutually exclusive
- Pipeline Kind defaults to 'WavePipeline' if empty after YAML parse
- Step filter --steps and --exclude are mutually exclusive; --from-step and --steps are also mutually exclusive
- Prompt injection in input is sanitized — detected content replaced with '[INPUT SANITIZED FOR SECURITY]'
- Process groups killed on timeout — SIGTERM to process group, 3-second grace, then SIGKILL
- Workspace symlinks must resolve within source tree — symlinks pointing outside silently skipped to prevent path traversal
- Files >10MB silently skipped during workspace copy
- Pipeline success requires no required (non-optional) step failures
- rework_only steps excluded from normal DAG scheduling — only triggered by on_failure:rework policy

## Key Decisions

- DFS-based topological sort with postorder traversal — guarantees all dependencies appear before dependents
- Four-tier timeout resolution: step-level > CLI --timeout > manifest default > configurable fallback in internal/timeouts
- Three-tier model resolution: CLI --model > per-persona model > adapter default
- Retry loop with backoff strategies (fixed, linear, exponential capped at timeouts.RetryMaxDelay) — step-level retries re-run the entire adapter, not just validation
- Rework architecture: failed step triggers separate rework_step with AttemptContext (error, stdout tail, partial artifacts) injected
- Cross-pipeline artifact flow via crossPipelineArtifacts map — enables pipeline composition without filesystem coupling
- Resume creates sub-pipeline by stripping dependencies on completed steps — avoids DAG validation failure on partial graph
- Workspace isolation via three strategies: git worktree (reuses branches), mount-based copy (with git init anchor), basic directory
- Contract prompt injected into user prompt (not system prompt) — 'SINGLE source of truth for schema injection'
- Relay compaction is best-effort — failure logged but does not fail the step
- Process group isolation (Setpgid:true) for clean subprocess termination on timeout/cancellation
- Composition primitives (sub-pipeline, iterate, branch, gate, loop, aggregate) enable pipeline-of-pipelines without executor changes
- Stream event parsing from NDJSON — real-time tool use, text, and token accounting during adapter execution
- Skip directories during workspace copy: node_modules, .git, .wave, .claude, vendor, __pycache__, .venv, dist, build, .next, .cache

## Domain Vocabulary

| Term | Meaning |
|------|--------|
| Pipeline | A DAG of Steps with metadata, input config, output aliases, and optional skill/tool requirements |
| Step | Atomic execution unit — has persona, dependencies, timeout, workspace config, exec config, output artifacts, retry policy, and optional composition primitives |
| PipelineExecution | In-flight pipeline state — tracks step states, results, artifact paths, workspace paths, worktree info, and attempt contexts |
| PipelineStatus | Observable pipeline state — ID, name, state, current step, completed/failed steps, timestamps |
| AdapterRunConfig | Complete configuration for a single adapter subprocess invocation — persona, workspace, prompt, permissions, sandbox, skills, ontology |
| AdapterResult | Subprocess outcome — exit code, stdout, token counts, artifacts, result content, failure reason, subtype |
| StreamEvent | Real-time NDJSON event from adapter — tool_use, tool_result, text, result, system with token accounting |
| AttemptContext | Failure context passed to retry/rework — attempt number, prior error, failure class, stdout tail, contract errors, partial artifacts |
| WorktreeInfo | Git worktree metadata — absolute path and repository root for cleanup |
| StepFilter | Include/exclude filter for selective step execution via --steps and --exclude CLI flags |
| ResumeState | Recovered state from prior run — step states, results, artifact paths, workspace paths, failure contexts, rework transitions |
| ResumeManager | Orchestrates pipeline resume — validates phase sequence, detects stale artifacts, acquires workspace locks, creates sub-pipeline |
| StatePending | Step has not started execution |
| StateRunning | Step is currently executing via adapter |
| StateCompleted | Step finished successfully with validated output |
| StateFailed | Step exhausted retries and on_failure policy applied |
| StateRetrying | Step failed but has remaining retry attempts |
| StateSkipped | Step skipped due to on_failure:skip or dependency failure |
| StateReworking | Step is being re-executed by its rework target |
| OnFailureFail | Default policy — step and pipeline fail when retries exhausted |
| OnFailureSkip | Step marked skipped, pipeline continues |
| OnFailureContinue | Step marked failed but pipeline continues |
| OnFailureRework | Triggers rework_step with failure context injected |
| FailureReasonTimeout | Adapter subprocess exceeded its timeout |
| FailureReasonContextExhaustion | LLM context window was exhausted (error_max_turns or prompt too long) |
| FailureReasonRateLimit | API rate limit reached — immediate failure, no retry |
| maxStdoutTailChars | 2000-character cap on stdout retained for retry/rework context |

## Neighboring Contexts

- **validation**
- **security**
- **configuration**

## Key Files

- `internal/pipeline/executor.go`
- `internal/pipeline/types.go`
- `internal/pipeline/dag.go`
- `internal/pipeline/resume.go`
- `internal/pipeline/validation.go`
- `internal/pipeline/step_filter.go`
- `internal/pipeline/errors.go`
- `internal/adapter/adapter.go`
- `internal/adapter/claude.go`
- `internal/adapter/errors.go`
- `internal/workspace/workspace.go`

