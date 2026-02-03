<!--
  Sync Impact Report
  Version change: 1.0.0 → 2.0.0 (MAJOR: backward-incompatible principle redefinition)
  Added sections: None
  Modified principles: Principle 1 (Single Binary, Zero Dependencies → Single Binary, Minimal Dependencies)
  Removed sections: None
  Rationale: Allow essential UI/UX libraries (e.g., Bubble Tea) for enhanced user experience
             while maintaining single binary deployment model
  Templates requiring updates:
    - .specify/templates/plan-template.md: ✅ compatible (Constitution Check section exists)
    - .specify/templates/spec-template.md: ✅ compatible (no changes needed)
    - .specify/templates/tasks-template.md: ✅ compatible (no changes needed)
  Follow-up TODOs: Update PR #14 to reference constitutional amendment
-->

# Wave Project Constitution

**Version**: 2.0.0
**Ratification Date**: 2026-02-01
**Last Amended**: 2026-02-03
**Project**: Wave — Multi-agent orchestrator wrapping LLM CLIs

## Preamble

Wave is a Go-based multi-agent orchestrator that wraps Claude Code
and other LLM CLIs via subprocess. It composes personas, pipelines,
contracts, and relay/compaction into a continuous development system.
This constitution defines the non-negotiable principles governing all
design, implementation, and operational decisions.

## Principle 1: Single Binary, Minimal Dependencies

Wave MUST be a single statically-linked Go binary with minimal runtime
dependencies. Installation remains `curl | sh` or a single `COPY` in a
Dockerfile. No interpreters, no package managers, no virtual
environments required on the target host.

- The binary MUST NOT require Node.js, Python, or language runtimes to
  function.
- Essential UI/UX libraries (e.g., terminal handling, progress display)
  MAY be included as Go dependencies when they significantly enhance
  user experience without compromising deployment simplicity.
- Adapter binaries (e.g., `claude`, `opencode`) are external
  prerequisites provided by the host, not bundled by Wave.
- Compilation tools for contract validation (e.g., `tsc`) are
  optional; Wave MUST degrade gracefully when absent.
- Dependencies MUST be justified by clear user value and MUST NOT
  require additional system-level package installation.

## Principle 2: Manifest as Single Source of Truth

`wave.yaml` MUST be the sole configuration file declaring all
adapters, personas, runtime settings, and skill mounts. No other
configuration file may override or supplement the manifest without
being explicitly referenced from it.

- Every adapter, persona, pipeline, and skill mount MUST trace back
  to a declaration in the manifest.
- The manifest MUST be version-controlled and human-readable.
- `wave validate` MUST catch all reference errors (missing files,
  unknown adapters, undefined personas) before any pipeline runs.

## Principle 3: Persona-Scoped Execution Boundaries

Every agent invocation MUST be scoped by exactly one persona. A
persona defines the adapter, system prompt, temperature, permissions,
and hooks. No agent runs without a persona binding.

- Permission deny patterns MUST be enforced unconditionally — a
  read-only persona MUST NOT write files under any circumstance.
- PreToolUse hooks that exit non-zero MUST block the tool call.
- PostToolUse hooks MUST execute after every matching tool call.
- Personas MUST NOT share mutable state or chat history.

## Principle 4: Fresh Memory at Every Step Boundary

Every pipeline step MUST start with a fresh context. No chat history
from prior steps is inherited. Artifacts flow between steps only via
explicitly declared `inject_artifacts` bindings.

- This prevents context pollution, hallucinated cross-step
  connections, and repeated work.
- The Navigator persona is the sole exception: it MAY accumulate
  context within a single invocation to build a coherent codebase
  map, but it is read-only so the risk is contained.
- All inter-step communication MUST be via files (artifacts), never
  via shared memory or inherited conversation.

## Principle 5: Navigator-First Architecture

The first step of every pipeline MUST be a Navigator persona
producing a codebase map (navigation context). No implementation
step may search the codebase independently.

- The navigation context defines the collaboration unit: the set of
  files any subsequent persona is allowed to reference or modify.
- Implementation personas MUST NOT touch files outside the
  collaboration unit.
- This eliminates duplicate discovery and inconsistent file
  targeting across parallel agents.

## Principle 6: Contracts at Every Handover

Every pipeline step boundary MUST include a handover contract that
validates output before the next step begins. No artifact propagates
unchecked.

- Contract types: structural (JSON schema), correctness (compilation
  check), behavioral (test suite results).
- Failed contracts MUST trigger retry (up to configured max_retries)
  before halting the pipeline.
- Contracts MUST be machine-verifiable — no human-in-the-loop
  validation at step boundaries during automated execution.

## Principle 7: Relay via Dedicated Summarizer

When an agent reaches the configured context utilization threshold
(default 80%), compaction MUST be performed by a separate Summarizer
persona, not by the running agent itself.

- The Summarizer receives the chat history as input data with fresh
  context and temperature 0.1.
- Output is a structured checkpoint containing: completed work,
  remaining work, modified files, resume instructions.
- The original persona resumes from the checkpoint with clean
  context. It MUST NOT re-do completed work.
- If the Summarizer itself exceeds its token cap, the pipeline MUST
  halt rather than recurse into infinite compaction.

## Principle 8: Ephemeral Workspaces for Safety

Every pipeline step MUST execute in an ephemeral workspace directory.
The main repository is mounted with explicit access modes
(readonly/readwrite) — never modified in place.

- Workspaces MUST persist until the user explicitly runs
  `wave clean`. No automatic deletion.
- Parallel matrix workers MUST each have isolated workspaces. No
  shared mutable filesystem state between concurrent agents.
- A hallucinating agent that runs destructive commands can only
  damage its own workspace, not the source repository.

## Principle 9: Credentials Never Touch Disk

Credentials (API keys, tokens) MUST reach adapter subprocesses
exclusively via inherited environment variables. Credentials MUST
NOT appear in:

- Manifest files (`wave.yaml`)
- Checkpoint files produced by relay
- Audit logs or trace files
- Ephemeral workspace contents
- Pipeline state persistence files

## Principle 10: Observable Progress, Auditable Operations

The system MUST emit structured progress events to stdout on every
step state transition (step name, state, duration, outcome). Events
MUST be both human-readable and machine-parseable.

- When audit logging is enabled, all tool calls and file operations
  MUST be logged to the configured trace directory.
- Audit logs MUST NOT capture credential values.
- Pipeline state MUST be persisted so interrupted executions can
  resume from the last completed step.

## Principle 11: Bounded Recursion and Resource Limits

Meta-pipelines (where a persona generates pipeline definitions at
runtime) MUST be bounded:

- Maximum recursion depth: configurable, default 2.
- Maximum total steps across nested pipelines: configurable.
- Maximum total token consumption: configurable hard cap.
- Wall-clock timeout: configurable, applies to entire pipeline tree.
- Per-step timeout: configurable, crash or hang triggers retry using
  the same max_retries as contract failures.

## Principle 12: Minimal Step State Machine

Pipeline steps transition through exactly 5 states:
**Pending → Running → Completed / Failed / Retrying**.

- Relay/compaction is a sub-state of Running.
- Only Pending and Failed steps are resumable after interruption.
- Subprocess crashes and timeouts are treated as step failures and
  use the same retry mechanism as contract failures.
- No additional states may be added without a constitution amendment.

## Governance

### Amendment Procedure

1. Propose amendment as a pull request modifying this file.
2. Amendment MUST include rationale and impact analysis.
3. All dependent templates MUST be updated in the same PR.
4. Version MUST be incremented per semantic versioning:
   - MAJOR: Principle removal or backward-incompatible redefinition.
   - MINOR: New principle added or existing principle materially
     expanded.
   - PATCH: Clarification, wording fix, non-semantic refinement.

### Compliance Review

- Every `/speckit.plan` execution MUST include a Constitution Check
  gate verifying the plan does not violate any principle.
- Violations MUST be documented in the Complexity Tracking table
  with explicit justification and rejected alternatives.
- Unjustified violations MUST block plan approval.

### Dependent Artifacts

Changes to this constitution MUST be propagated to:
- `.specify/templates/plan-template.md` (Constitution Check section)
- `.specify/templates/spec-template.md` (if scope constraints change)
- `.specify/templates/tasks-template.md` (if task categorization
  changes)
- `AGENTS.md` (if governance or workflow procedures change)
