# Feature Specification: Manifest & Pipeline Design

**Feature Branch**: `014-manifest-pipeline-design`
**Created**: 2026-02-01
**Status**: Draft
**Input**: User description: "Muzzle manifest (muzzle.yaml) and pipeline DAG system for multi-agent orchestration wrapping Claude Code and other LLM CLIs via subprocess"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Define Project Manifest (Priority: P1)

A developer creates a `muzzle.yaml` file in their project root to declare the adapters (LLM CLIs), personas (agent configurations), runtime settings, and skill mounts that Muzzle will use. This single file is the source of truth for all orchestration behavior in the project.

**Why this priority**: Without the manifest, nothing else works. Every pipeline, persona, and adapter references back to this file.

**Independent Test**: Can be fully tested by writing a `muzzle.yaml`, running `muzzle validate`, and confirming it parses without errors and resolves all references (adapter binaries exist, persona prompt files exist, hook scripts exist).

**Acceptance Scenarios**:

1. **Given** a project directory with no Muzzle configuration, **When** the developer runs `muzzle init`, **Then** a `muzzle.yaml` scaffold is created with default adapter, one example persona, and runtime defaults.
2. **Given** a valid `muzzle.yaml` referencing a persona with a missing system prompt file, **When** the developer runs `muzzle validate`, **Then** the system reports the missing file path and which persona references it.
3. **Given** a `muzzle.yaml` with two adapters (claude, opencode), **When** the developer lists available adapters, **Then** both are shown with their binary paths, modes, and default permission sets.

---

### User Story 2 - Run a Pipeline End-to-End (Priority: P1)

A developer triggers a named pipeline (e.g., `speckit-flow`) that executes a DAG of steps. Each step binds a persona to an ephemeral workspace, receives injected artifacts from prior steps, and produces output artifacts validated against handover contracts before the next step begins.

**Why this priority**: The pipeline DAG is the core execution model. Without it, Muzzle is just a config file with no runtime behavior.

**Independent Test**: Can be tested by running `muzzle run --pipeline speckit-flow --input "add dark mode"` and observing that each step executes in order, artifacts flow between steps, and contract validation gates progression.

**Acceptance Scenarios**:

1. **Given** a pipeline with steps navigate → specify → plan → implement → review, **When** the developer runs the pipeline, **Then** each step executes only after its dependencies complete successfully.
2. **Given** a step that produces an artifact failing its handover contract (e.g., invalid JSON schema), **When** validation runs, **Then** the step retries up to the configured max_retries before the pipeline halts with a clear error.
3. **Given** a pipeline with a matrix strategy step (e.g., one craftsman per task), **When** the plan step produces 5 tasks, **Then** 5 parallel agent instances launch, each receiving only its assigned task and shared navigation context.

---

### User Story 3 - Persona-Scoped Agent Execution (Priority: P1)

Each pipeline step runs an agent configured by a persona. The persona determines: which adapter (CLI) to use, what system prompt is injected, what tools/permissions are allowed, what hooks fire on tool use, and the temperature setting. Personas enforce separation of concerns — a read-only navigator cannot write files; a craftsman cannot install dependencies.

**Why this priority**: Personas are the safety and specialization mechanism. Without persona scoping, all agents have the same permissions and context, defeating the purpose of multi-agent orchestration.

**Independent Test**: Can be tested by running a navigator persona and confirming it can read files and run git log but is blocked from writing files or running destructive commands.

**Acceptance Scenarios**:

1. **Given** a persona configured with deny patterns for write operations, **When** the agent attempts to write a file, **Then** the operation is blocked and the agent receives a permission denial message.
2. **Given** a persona with a PreToolUse hook on commit operations, **When** the agent attempts to commit, **Then** the hook script executes first; if it exits non-zero, the commit is blocked.
3. **Given** a persona with a low temperature and a system prompt file, **When** the agent starts, **Then** the system prompt is loaded from that file and the temperature is applied for all LLM calls.

---

### User Story 4 - Context Relay and Compaction (Priority: P2)

When an agent approaches its context token limit (configurable threshold, default 80%), the relay mechanism fires automatically. A summarizer persona receives the agent's chat history, produces a checkpoint, and a fresh instance of the original persona resumes from that checkpoint with clean context.

**Why this priority**: Without relay, long-running tasks silently degrade as context fills. Relay ensures quality is maintained across extended executions.

**Independent Test**: Can be tested by running an agent with a deliberately small context window, observing relay trigger at threshold, verifying checkpoint is produced, and confirming the resumed agent picks up where the original left off without repeating work.

**Acceptance Scenarios**:

1. **Given** an agent at the configured context utilization threshold, **When** the relay triggers, **Then** the summarizer persona produces a structured checkpoint containing completed actions, remaining work, modified files, and resume instructions.
2. **Given** a checkpoint from a compacted session, **When** a fresh instance of the original persona starts with that checkpoint injected, **Then** it reads the checkpoint first and continues from remaining work without re-doing completed work.
3. **Given** relay configured with a summarize-to-checkpoint strategy, **When** compaction occurs, **Then** only one additional LLM call is made (the summarizer) before the original persona resumes.

---

### User Story 5 - Handover Contracts Between Steps (Priority: P2)

Every pipeline step boundary includes a handover contract that validates the output of the completing step before the next step can begin. Contracts can validate structure (schema checks), correctness (compilation checks), or behavior (test suite results). Failed contracts trigger retries or pipeline halts.

**Why this priority**: Contracts prevent wasted work. Without them, a poorly-formed artifact propagates through multiple steps before being caught.

**Independent Test**: Can be tested by defining a step with a schema-based contract, having the step produce invalid output, and confirming the pipeline retries the step rather than proceeding.

**Acceptance Scenarios**:

1. **Given** a handover contract with a structural schema on the navigate step, **When** the navigator produces output missing a required field, **Then** the contract validation fails and the step retries.
2. **Given** a handover contract requiring compilation validation, **When** the philosopher produces a contract file, **Then** the runtime compiles it and only proceeds if compilation succeeds.
3. **Given** a handover contract requiring tests to pass, **When** the craftsman's implementation fails tests, **Then** the pipeline does not proceed to the review step.

---

### User Story 6 - Ad-Hoc Task Execution (Priority: P2)

A developer runs `muzzle do "fix the auth bug"` for quick, single-shot tasks that don't need the full pipeline. Muzzle generates an in-memory two-step pipeline (navigate → execute), runs it, and discards the pipeline definition. Optionally, the generated pipeline can be saved for inspection.

**Why this priority**: Not every task justifies a multi-step pipeline. Ad-hoc mode provides the Muzzle safety model (personas, permissions, ephemeral workspaces) with zero configuration overhead.

**Independent Test**: Can be tested by running `muzzle do "fix typo in README"` and confirming a navigator runs first, then a craftsman executes the fix, all within ephemeral workspaces.

**Acceptance Scenarios**:

1. **Given** a developer runs `muzzle do` with a task description, **When** the command executes, **Then** a navigator analyzes the codebase first, then a craftsman implements the fix using the navigation context.
2. **Given** `muzzle do` with a persona override flag, **When** the command executes, **Then** the execute step uses the specified persona instead of the default craftsman.
3. **Given** `muzzle do` with a save flag, **When** the command completes, **Then** the generated pipeline YAML is written to the specified path for inspection or reuse.

---

### User Story 7 - Meta-Pipeline (Self-Designing) (Priority: P3)

For novel problems that don't fit existing pipeline templates, Muzzle's meta-pipeline has the philosopher persona design a custom pipeline at runtime. The runtime validates the generated pipeline against schema and semantic rules, then executes it. Recursion depth is capped to prevent infinite loops.

**Why this priority**: Meta-pipelines handle edge cases where templates don't fit, but they're expensive and complex. Most work should use standard templates.

**Independent Test**: Can be tested by routing a novel task to the meta pipeline and confirming the philosopher designs a valid pipeline that the runtime executes successfully.

**Acceptance Scenarios**:

1. **Given** a novel task routed to the meta pipeline, **When** the philosopher designs a pipeline, **Then** the generated definition passes both schema validation and semantic checks (first step is navigator, all steps have handover contracts, all steps use fresh memory).
2. **Given** a meta pipeline at the configured maximum recursion depth, **When** the philosopher attempts to generate another meta pipeline, **Then** it is blocked and the pipeline fails with a depth limit error.
3. **Given** a meta-generated pipeline that fails execution, **When** the failure occurs, **Then** the trace is logged and the generated pipeline definition is preserved for debugging.

---

### User Story 8 - Comprehensive Documentation (Priority: P1)

Muzzle includes a VitePress documentation site that serves as the authoritative guide for users and contributors. The documentation is comprehensive, accurate, and always in sync with the codebase. It includes getting started guides, detailed references, tutorials, examples, and architectural concepts.

**Why this priority**: Documentation is essential for adoption and proper use. Users should be able to accomplish any task with only the documentation, and contributors should be able to understand and extend the system through the docs.

**Independent Test**: Can be fully tested by building the documentation site, validating all internal links resolve, and verifying all code examples against the current implementation.

**Acceptance Scenarios**:

1. **Given** a new user visits the documentation site, **When** they follow the quick-start guide, **Then** they can successfully install Muzzle, initialize a project, and run their first pipeline without external help.
2. **Given** a developer needs to understand a specific configuration option, **When** they consult the manifest reference, **Then** they find the option with type, default value, examples, and cross-references to related concepts.
3. **Given** the Muzzle codebase is updated, **When** CI runs, **Then** the documentation automatically updates (CLI reference from commands, schema docs from Go structs) and fails if examples are out of sync.
4. **Given** a user encounters an error, **When** they check the troubleshooting guide, **Then** they find their error with clear steps to resolve it.
5. **Given** a user wants to implement a complex workflow, **When** they read the tutorials and examples, **Then** they find a similar pattern they can adapt to their needs.

---

### Edge Cases

- What happens when a pipeline step's adapter binary is not found on PATH? The step fails immediately with a clear error naming the missing binary and the persona that requires it.
- What happens when two matrix workers modify the same file? The merge step must detect and report conflicts rather than silently overwriting.
- What happens when the relay summarizer itself hits its token limit? The relay has a hard token cap; if the summarizer cannot complete within it, the pipeline halts rather than entering infinite compaction.
- What happens when a pipeline is interrupted mid-execution (container crash, CI timeout)? State is persisted per-step; on resume, the pipeline restarts from the last completed step, not from scratch.
- What happens when a persona references an adapter not defined in the manifest? Validation catches this at `muzzle validate` time, before any pipeline runs.
- What happens when a pipeline has circular dependencies between steps? The DAG parser detects cycles at load time and rejects the pipeline with a clear error.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST parse and validate manifests declaring adapters, personas, runtime settings, and skill mounts.
- **FR-002**: System MUST execute pipeline DAGs defined in YAML, respecting step dependencies and sequential/parallel execution.
- **FR-003**: System MUST bind each pipeline step to exactly one persona, configuring the underlying adapter with that persona's permissions, hooks, system prompt, and temperature.
- **FR-004**: System MUST create ephemeral workspaces per step, mounting source repositories with the specified access mode (readonly/readwrite).
- **FR-005**: System MUST inject artifacts from completed steps into dependent steps as specified in the pipeline definition.
- **FR-006**: System MUST validate handover contracts at every step boundary before proceeding to dependent steps.
- **FR-007**: System MUST retry failed steps up to the configured max_retries before halting the pipeline. Subprocess crashes and timeouts count as step failures and use the same retry mechanism.
- **FR-018**: System MUST enforce a configurable per-step timeout; if an adapter subprocess exceeds it, the process is killed and the step transitions to Retrying.
- **FR-019**: System MUST persist ephemeral workspaces until the user explicitly runs a cleanup command (`muzzle clean`). Workspaces are never auto-deleted.
- **FR-020**: System MUST emit structured progress events to stdout on every step state transition, including step name, new state, duration, and outcome. Events MUST be both human-readable and machine-parseable.
- **FR-008**: System MUST support matrix strategy execution, spawning parallel agent instances from a task list with configurable concurrency limits.
- **FR-009**: System MUST monitor agent context utilization and trigger relay/compaction when the configured threshold is reached.
- **FR-010**: System MUST persist pipeline state so interrupted executions can resume from the last completed step.
- **FR-011**: System MUST enforce persona permission boundaries, blocking tool calls that match deny patterns.
- **FR-012**: System MUST execute PreToolUse and PostToolUse hooks as configured per persona, blocking operations when hooks exit non-zero.
- **FR-013**: System MUST support ad-hoc execution that generates and runs a minimal pipeline from a text prompt.
- **FR-014**: System MUST support meta-pipelines where a persona generates pipeline definitions at runtime, with recursion depth limits enforced.
- **FR-015**: System MUST log all tool calls and file operations when audit logging is enabled. Audit logs MUST NOT capture environment variable values or credential content.
- **FR-017**: System MUST pass credentials to adapter subprocesses exclusively via inherited environment variables; credentials MUST NOT be written to disk, manifest files, checkpoint files, or audit logs.
- **FR-016**: System MUST route incoming work items to the appropriate pipeline based on configurable routing rules.
- **FR-021**: System MUST include comprehensive documentation built with VitePress that covers all features, configuration options, and workflows.
- **FR-022**: Documentation MUST validate all code examples against the current implementation and fail CI if out of sync.
- **FR-023**: Documentation MUST provide copy-paste ready examples for every concept and configuration option.
- **FR-024**: Documentation MUST be automatically deployable to GitHub Pages via CI/CD pipeline.

### Key Entities

- **Manifest**: The top-level configuration file declaring all adapters, personas, runtime settings, and skill mounts for a project.
- **Adapter**: A wrapper configuration for a specific LLM CLI. Defines binary path, mode, output format, default permissions, and project files to project into workspaces.
- **Persona**: An agent configuration binding an adapter to a specific role. Includes system prompt file, temperature, permission overrides, and hook definitions.
- **Pipeline**: A DAG of steps defined in YAML. Each step binds a persona to a workspace, receives input artifacts, produces output artifacts, and validates handover contracts.
- **Step**: A single unit of work in a pipeline. Executes one persona in one ephemeral workspace. Has dependencies, memory strategy, artifact injection, and handover configuration. Transitions through states: Pending → Running → Completed / Failed / Retrying. Relay/compaction occurs as a sub-state of Running. Only Pending and Failed steps are resumable.
- **Handover Contract**: A validation rule applied at step boundaries. Ensures output artifacts meet structural, compilation, or behavioral requirements before the next step begins.
- **Relay**: The context compaction mechanism. When an agent nears its token limit, a summarizer persona compresses the chat history into a checkpoint for a fresh instance to resume from.
- **Checkpoint**: A structured document produced by relay containing completed work, remaining work, current state, and resume instructions.
- **Workspace**: An ephemeral directory where a step executes. Mounts the source repository and injected artifacts with specified access modes.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: A developer can go from project initialization to a validated manifest in under 5 minutes, with no documentation required beyond the scaffold comments.
- **SC-002**: A multi-step pipeline completes without manual intervention for standard feature work.
- **SC-003**: Contract validation catches malformed artifacts at step boundaries 100% of the time — no invalid artifacts propagate to downstream steps.
- **SC-004**: Relay compaction preserves sufficient context that resumed agents do not repeat completed work or lose track of remaining tasks.
- **SC-005**: An interrupted pipeline resumes from the last completed step, not from scratch.
- **SC-006**: Persona permission boundaries prevent 100% of denied tool calls — a read-only persona cannot write files under any circumstances.
- **SC-007**: Ad-hoc execution provides the same safety guarantees (permissions, ephemeral workspaces) as full pipeline execution.
- **SC-008**: Meta-pipelines are bounded: recursion never exceeds the configured depth, and total token consumption never exceeds the configured cap.
- **SC-009**: A new user can go from zero knowledge to running their first pipeline in under 10 minutes using only the documentation.
- **SC-010**: All documentation examples validate against the current implementation - CI fails if any example is out of date.
- **SC-011**: The documentation site builds successfully and all internal links resolve.
- **SC-012**: Troubleshooting guide covers the top 20 most common user errors with actionable solutions.

## Clarifications

### Session 2026-02-01

- Q: How do credentials (API keys) reach adapter subprocesses? → A: Environment variables only — adapter inherits from parent process, never written to disk or logs.
- Q: What lifecycle states does a pipeline step transition through? → A: Minimal 5-state: Pending → Running → Completed / Failed / Retrying. Relay/compaction is a sub-state of Running.
- Q: How are adapter subprocess crashes or hangs handled? → A: Per-step configurable timeout; crash or timeout transitions step to Retrying using the same max_retries as contract failures.
- Q: When are ephemeral workspaces cleaned up? → A: Manual only — workspaces persist until `muzzle clean` is run; user controls lifecycle.
- Q: What progress signals does the developer see during pipeline execution? → A: Structured event stream to stdout — each step transition emits a line (step, state, duration, outcome) parseable by CI and human-readable in terminal.

## Assumptions

- The underlying LLM CLI's headless mode, hooks system, and permission model remain stable across versions used during development.
- The host environment provides the adapter binaries on PATH; Muzzle does not install them.
- Compilation tools are available in the environment for contract validation; if not, compilation-based contracts degrade to syntax-only checks.
- Pipeline state persistence uses local storage; distributed state across multiple machines is out of scope.
- Ephemeral workspaces use temporary directories; custom workspace root paths are configurable but storage optimization is out of scope.

## Scope Boundaries

**In scope**:
- Manifest schema and validation
- Pipeline DAG execution engine
- Persona binding and permission enforcement
- Handover contract validation
- Relay/compaction mechanism
- Ad-hoc execution mode
- Meta-pipeline with recursion limits
- Audit logging
- Pipeline state persistence and resumption
- Comprehensive VitePress documentation site with automated validation
- Documentation generation from code (CLI commands, schemas)

**Out of scope**:
- GUI or web dashboard (documentation only)
- Multi-machine distributed execution
- Billing or token cost tracking
- Custom adapter protocol development (adapters wrap existing CLIs)
- LLM fine-tuning or model training
- User authentication or multi-tenancy
- Interactive documentation playground (static site only)
