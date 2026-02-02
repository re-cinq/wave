# Feature Specification: Wave CLI Implementation

**Feature Branch**: `015-wave-cli-implementation`
**Created**: 2026-02-02
**Status**: Draft
**Input**: User description: "Real-world implementation of WAVE - the multi-agent pipeline orchestrator for AI-assisted development"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Project Initialization (Priority: P1)

A developer runs `wave init` in their project root to bootstrap Wave configuration. The command creates a `wave.yaml` manifest with default adapters, the 7 built-in personas, and runtime settings. It also scaffolds the `.wave/` directory structure with persona system prompts, pipeline definitions, and contract schemas.

**Why this priority**: Without initialization, users cannot use Wave. This is the entry point for all adoption.

**Independent Test**: Can be fully tested by running `wave init` in an empty directory and verifying all scaffolded files exist, parse correctly, and pass `wave validate`.

**Acceptance Scenarios**:

1. **Given** an empty project directory, **When** the developer runs `wave init`, **Then** a `wave.yaml` is created with the claude adapter configured, all 7 built-in personas defined, and default runtime settings.
2. **Given** a project directory with an existing `wave.yaml`, **When** the developer runs `wave init`, **Then** the system prompts for confirmation before overwriting, or merges new defaults non-destructively with `--merge` flag.
3. **Given** `wave init` completes successfully, **When** the developer runs `wave validate`, **Then** no errors are reported and all persona system prompt files are found.

---

### User Story 2 - Manifest Validation (Priority: P1)

A developer runs `wave validate` to check their configuration before running pipelines. Validation catches missing files, invalid references, malformed YAML, and schema violations. Clear error messages pinpoint the problem location.

**Why this priority**: Validation prevents runtime failures. Users need confidence their configuration is correct before executing expensive LLM calls.

**Independent Test**: Can be tested by creating manifests with deliberate errors (missing persona, invalid adapter reference, non-existent system prompt file) and confirming validation reports each error with file path and line number.

**Acceptance Scenarios**:

1. **Given** a `wave.yaml` with a persona referencing a non-existent adapter, **When** the developer runs `wave validate`, **Then** the error message includes the persona name, the referenced adapter name, and states "adapter not defined in manifest".
2. **Given** a `wave.yaml` with a persona whose `system_prompt_file` does not exist, **When** the developer runs `wave validate`, **Then** the error message includes the full path and suggests creating the file.
3. **Given** a fully valid configuration, **When** the developer runs `wave validate --verbose`, **Then** a summary shows counts of adapters, personas, and pipelines found, plus adapter binary availability.

---

### User Story 3 - Ad-Hoc Task Execution (Priority: P1)

A developer runs `wave do "fix the auth bug"` to quickly accomplish a single task without defining a pipeline. Wave generates an in-memory two-step pipeline (navigate → execute), runs it with the configured personas, and produces results. This provides Wave's safety model (permissions, ephemeral workspaces) with minimal overhead.

**Why this priority**: This is the fastest path to value for new users. They can experience Wave's benefits before learning full pipeline syntax.

**Independent Test**: Can be tested by running `wave do "add a comment to main.go"` and confirming the navigator analyzes first, then the craftsman makes the change, all in ephemeral workspaces.

**Acceptance Scenarios**:

1. **Given** a developer runs `wave do` with a task description, **When** execution completes, **Then** a navigator analyzes the codebase first, producing context for the execute step.
2. **Given** `wave do` with `--persona debugger` flag, **When** execution completes, **Then** the execute step uses the debugger persona instead of the default craftsman.
3. **Given** `wave do` with `--dry-run` flag, **When** the command runs, **Then** it prints the generated pipeline YAML without executing.
4. **Given** `wave do` with `--save pipeline.yaml` flag, **When** execution completes, **Then** the generated pipeline is written to the specified path for reuse.

---

### User Story 4 - Pipeline Execution (Priority: P1)

A developer runs `wave run --pipeline speckit-flow --input "add user authentication"` to execute a full DAG workflow. Each step runs its bound persona in an isolated workspace, artifacts flow between steps, contracts validate outputs, and progress events stream to stdout.

**Why this priority**: Pipeline execution is the core value proposition of Wave. Without it, the tool is incomplete.

**Independent Test**: Can be tested by running the hotfix pipeline with a known bug description and observing all three steps (investigate → fix → verify) complete in sequence with artifacts passing between them.

**Acceptance Scenarios**:

1. **Given** a pipeline with steps A → B → C, **When** the developer runs the pipeline, **Then** steps execute in topological order, with B waiting for A's completion.
2. **Given** parallel steps (e.g., test and review both depend on implement), **When** the pipeline executes, **Then** both steps run concurrently.
3. **Given** a step fails its handover contract, **When** validation fails, **Then** the step retries up to `max_retries` before the pipeline halts with a clear error.
4. **Given** the `--dry-run` flag, **When** the command runs, **Then** the execution plan is printed (step order, personas, dependencies) without invoking any adapters.
5. **Given** the `--from-step implement` flag, **When** the pipeline runs, **Then** execution skips to the specified step, assuming prior steps completed (useful for resumption).

---

### User Story 5 - Persona Permission Enforcement (Priority: P1)

Each pipeline step executes within the permission boundaries of its persona. The navigator can read but not write. The craftsman can write but not push. Deny patterns always take precedence. When a tool call violates permissions, it is blocked and the agent receives a clear denial message.

**Why this priority**: Permission enforcement is the core safety mechanism. Without it, Wave provides no guardrails over raw LLM CLI invocation.

**Independent Test**: Can be tested by configuring a navigator with deny patterns for Write, invoking the navigator persona, and confirming any write attempts are blocked with an appropriate message.

**Acceptance Scenarios**:

1. **Given** a navigator persona with `deny: ["Write(*)"]`, **When** the agent attempts to write a file, **Then** the operation is blocked and the agent sees "Permission denied: Write is not allowed for navigator persona".
2. **Given** a craftsman persona with `allowed_tools: ["Write"]`, **When** the agent writes to a file, **Then** the operation succeeds.
3. **Given** a persona with both `allowed_tools: ["Bash(*)"]` and `deny: ["Bash(rm -rf *)"]`, **When** the agent runs `rm -rf /`, **Then** the specific destructive command is blocked while other bash commands succeed.

---

### User Story 6 - Pipeline State Persistence and Resume (Priority: P2)

If a pipeline is interrupted (process killed, CI timeout, system crash), the state is persisted so execution can resume from the last completed step rather than restarting from scratch. The developer runs `wave resume <pipeline-id>` to continue.

**Why this priority**: Long-running pipelines are expensive. Resume capability prevents wasted compute and ensures reliability.

**Independent Test**: Can be tested by starting a multi-step pipeline, killing the process mid-execution, and running `wave resume` to confirm it picks up from the correct step.

**Acceptance Scenarios**:

1. **Given** a pipeline interrupted after step 2 of 4 completes, **When** the developer runs `wave resume`, **Then** execution continues from step 3.
2. **Given** a failed step with `state: retrying`, **When** resume is invoked, **Then** the step re-executes up to remaining retries.
3. **Given** no interrupted pipelines, **When** `wave resume` is run without arguments, **Then** it lists recent pipeline executions with their states.

---

### User Story 7 - Context Relay and Compaction (Priority: P2)

When an agent approaches its context token limit (configurable threshold, default 80%), the relay mechanism triggers automatically. A summarizer persona compacts the chat history into a checkpoint, and a fresh instance resumes from that checkpoint.

**Why this priority**: Long-running tasks degrade without relay. This ensures consistent quality across extended executions.

**Independent Test**: Can be tested by configuring a low token threshold, running a task that exceeds it, and verifying relay fires, checkpoint is produced, and resumed instance continues without repeating work.

**Acceptance Scenarios**:

1. **Given** an agent at 80% context utilization, **When** relay triggers, **Then** the summarizer produces a checkpoint containing completed actions, remaining work, and modified files.
2. **Given** a checkpoint from compaction, **When** the original persona resumes, **Then** it reads the checkpoint first and continues from remaining work.
3. **Given** relay trigger fires, **When** compaction completes, **Then** only one additional LLM call occurs (the summarizer) before the original persona resumes.

---

### User Story 8 - Handover Contract Validation (Priority: P2)

Every pipeline step boundary includes a handover contract that validates output before the next step begins. Contracts can check JSON schema compliance, TypeScript compilation, or test suite passage. Failed contracts trigger retries.

**Why this priority**: Contracts prevent bad artifacts from propagating through pipelines, catching errors early.

**Independent Test**: Can be tested by defining a step with a JSON schema contract, producing output that violates the schema, and confirming the step retries.

**Acceptance Scenarios**:

1. **Given** a step with a JSON schema contract, **When** output is missing a required field, **Then** validation fails and the step retries.
2. **Given** a step with a TypeScript interface contract, **When** the output file fails compilation, **Then** the step retries.
3. **Given** a step with a test suite contract, **When** tests fail, **Then** the pipeline does not proceed to the next step.
4. **Given** `max_retries: 2` and three consecutive failures, **When** the third failure occurs, **Then** the pipeline halts with a detailed error.

---

### User Story 9 - Matrix Strategy Parallel Execution (Priority: P2)

A pipeline step can use matrix strategy to spawn parallel agent instances from a task list. The plan step produces tasks, and the execute step fans out to N parallel craftsman instances, each receiving one task.

**Why this priority**: Matrix execution scales Wave to handle multiple parallel workstreams within a single pipeline run.

**Independent Test**: Can be tested by running a pipeline where the plan step produces 5 tasks and the execute step spawns 5 parallel workers, confirming all complete independently.

**Acceptance Scenarios**:

1. **Given** a plan step produces 5 tasks, **When** the matrix step executes, **Then** 5 parallel agent instances launch.
2. **Given** `max_concurrency: 2`, **When** 5 tasks exist, **Then** only 2 workers run simultaneously, with others queued.
3. **Given** one worker fails, **When** the failure occurs, **Then** other workers continue and the pipeline reports partial success with the failure details.

---

### User Story 10 - CLI List and Information Commands (Priority: P2)

A developer runs `wave list` to see available pipelines, personas, and adapters. The command provides quick reference without reading YAML files.

**Why this priority**: Discoverability helps users understand what's available in their configuration.

**Independent Test**: Can be tested by running `wave list pipelines` and confirming all defined pipelines appear with their descriptions.

**Acceptance Scenarios**:

1. **Given** `wave list pipelines`, **When** the command runs, **Then** all pipelines are listed with name, description, and step count.
2. **Given** `wave list personas`, **When** the command runs, **Then** all personas are listed with name, temperature, and permission summary.
3. **Given** `wave list adapters`, **When** the command runs, **Then** all adapters are listed with binary path and availability status.

---

### User Story 11 - Workspace Cleanup (Priority: P3)

A developer runs `wave clean` to remove ephemeral workspaces and pipeline state. Workspaces persist until explicitly cleaned, giving users control over when to reclaim disk space.

**Why this priority**: Workspace cleanup is necessary for disk hygiene but not essential for core functionality.

**Independent Test**: Can be tested by running pipelines, confirming workspaces exist, running `wave clean`, and verifying they are removed.

**Acceptance Scenarios**:

1. **Given** multiple pipeline runs have created workspaces, **When** `wave clean` runs, **Then** all workspaces under the configured root are removed.
2. **Given** `wave clean --keep-last 3`, **When** the command runs, **Then** only the 3 most recent pipeline workspaces are preserved.
3. **Given** `wave clean --dry-run`, **When** the command runs, **Then** it lists what would be deleted without deleting.

---

### User Story 12 - Meta-Pipeline Self-Design (Priority: P3)

For novel problems that don't fit existing templates, the meta-pipeline has the philosopher persona design a custom pipeline at runtime. The runtime validates the generated pipeline, then executes it. Recursion depth is capped.

**Why this priority**: Meta-pipelines handle edge cases but are expensive and complex. Most work should use standard templates.

**Independent Test**: Can be tested by routing a novel task to the meta pipeline and confirming a valid pipeline is generated and executed.

**Acceptance Scenarios**:

1. **Given** a novel task routed to meta pipeline, **When** the philosopher designs a pipeline, **Then** the generated definition passes schema and semantic validation.
2. **Given** recursion depth limit of 2, **When** a meta pipeline attempts to spawn another meta pipeline at depth 2, **Then** it is blocked with a depth limit error.
3. **Given** the generated pipeline fails execution, **When** failure occurs, **Then** the trace is logged and the generated definition is preserved.

---

### Edge Cases

- What happens when the adapter binary is not found on PATH? The step fails immediately with a clear error naming the missing binary.
- What happens when two matrix workers modify the same file? The merge step detects conflicts and reports them rather than silently overwriting.
- What happens when the relay summarizer itself hits its token limit? The relay has a hard cap; if the summarizer cannot complete, the pipeline halts.
- What happens when a pipeline has circular dependencies? The DAG parser detects cycles at load time and rejects the pipeline.
- What happens when YAML syntax is invalid? Clear error with line/column number from the YAML parser.
- What happens when credentials are not set? Clear error message at adapter invocation time, not buried in subprocess output.

## Requirements _(mandatory)_

### Functional Requirements

**Core CLI Commands**:
- **FR-001**: System MUST implement `wave init` to scaffold project configuration with adapters, personas, pipelines, and runtime settings.
- **FR-002**: System MUST implement `wave validate` to check manifest and pipeline files for errors before execution.
- **FR-003**: System MUST implement `wave run --pipeline <name> --input <text>` to execute named pipelines.
- **FR-004**: System MUST implement `wave do <task>` for ad-hoc task execution with auto-generated two-step pipeline.
- **FR-005**: System MUST implement `wave list [pipelines|personas|adapters]` to display configuration.
- **FR-006**: System MUST implement `wave resume [pipeline-id]` to continue interrupted pipelines.
- **FR-007**: System MUST implement `wave clean` to remove ephemeral workspaces and state.

**Manifest and Configuration**:
- **FR-008**: System MUST parse `wave.yaml` manifests declaring adapters, personas, runtime settings, and skill mounts.
- **FR-009**: System MUST validate all persona references resolve to defined adapters.
- **FR-010**: System MUST validate all system prompt file paths exist on disk.
- **FR-011**: System MUST validate all hook command scripts exist on disk.
- **FR-012**: System MUST warn (not error) when adapter binaries are not found on PATH.

**Pipeline Execution**:
- **FR-013**: System MUST execute pipeline DAGs respecting step dependencies and topological order.
- **FR-014**: System MUST execute independent steps in parallel when no dependency exists between them.
- **FR-015**: System MUST bind each step to exactly one persona with that persona's permissions, hooks, and temperature.
- **FR-016**: System MUST create ephemeral workspaces per step with configurable mount modes (readonly/readwrite).
- **FR-017**: System MUST inject artifacts from completed steps into dependent steps.
- **FR-018**: System MUST validate handover contracts at step boundaries before proceeding.
- **FR-019**: System MUST retry failed steps up to configured `max_retries` before halting.
- **FR-020**: System MUST enforce per-step timeouts, killing processes that exceed the limit.

**Persona and Permission Enforcement**:
- **FR-021**: System MUST enforce persona permission boundaries, blocking tool calls matching deny patterns.
- **FR-022**: System MUST evaluate deny patterns before allowed patterns (deny always wins).
- **FR-023**: System MUST execute PreToolUse hooks before tool calls, blocking on non-zero exit.
- **FR-024**: System MUST execute PostToolUse hooks after tool calls (informational, non-blocking).

**State Management**:
- **FR-025**: System MUST persist pipeline execution state to enable resume after interruption.
- **FR-026**: System MUST persist step state transitions (pending, running, completed, failed, retrying).
- **FR-027**: System MUST preserve ephemeral workspaces until explicit cleanup.

**Relay and Compaction**:
- **FR-028**: System MUST monitor agent context utilization during execution.
- **FR-029**: System MUST trigger relay when configured token threshold is reached.
- **FR-030**: System MUST invoke a dedicated summarizer persona for compaction (never self-summarize).
- **FR-031**: System MUST inject checkpoint into fresh persona instance for resumption.

**Matrix Execution**:
- **FR-032**: System MUST support matrix strategy for parallel task execution.
- **FR-033**: System MUST respect `max_concurrency` limits for parallel workers.
- **FR-034**: System MUST handle partial failures in matrix execution gracefully.

**Observability**:
- **FR-035**: System MUST emit structured progress events on step state transitions.
- **FR-036**: System MUST support audit logging of tool calls when enabled.
- **FR-037**: System MUST NOT log credentials or environment variable values in audit logs.

**Adapter Integration**:
- **FR-038**: System MUST invoke adapter CLIs via subprocess with inherited environment variables.
- **FR-039**: System MUST pass credentials via environment only (never disk).
- **FR-040**: System MUST support the Claude Code adapter (claude -p) as the primary integration.

### Key Entities

- **Manifest**: Top-level configuration file (`wave.yaml`) declaring adapters, personas, runtime, and skill mounts.
- **Adapter**: LLM CLI wrapper configuration with binary path, mode, and default permissions.
- **Persona**: Agent role configuration binding an adapter to permissions, system prompt, temperature, and hooks.
- **Pipeline**: DAG of steps defined in YAML with dependencies, artifacts, and contracts.
- **Step**: Single unit of work executing one persona in one workspace.
- **Handover Contract**: Validation rule at step boundaries ensuring output quality.
- **Workspace**: Ephemeral directory for step execution with mounted sources and artifacts.
- **Checkpoint**: Compacted context document for relay resumption.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: A developer can go from zero to first pipeline execution in under 10 minutes using only `wave init` and `wave do`.
- **SC-002**: Manifest validation catches 100% of configuration errors before any LLM calls are made.
- **SC-003**: Pipeline execution completes multi-step workflows without manual intervention for standard feature development.
- **SC-004**: Contract validation catches malformed artifacts at step boundaries 100% of the time.
- **SC-005**: Permission enforcement prevents 100% of tool calls matching deny patterns.
- **SC-006**: Interrupted pipelines resume from the last completed step rather than restarting.
- **SC-007**: Relay compaction preserves enough context that resumed agents continue without repeating work.
- **SC-008**: Ad-hoc execution provides the same safety guarantees (permissions, ephemeral workspaces) as full pipeline execution.
- **SC-009**: The CLI is a single static binary with no runtime dependencies beyond the adapter binaries.
- **SC-010**: Matrix execution scales to 10 parallel workers without resource contention.
- **SC-011**: Progress events are both human-readable in terminal and machine-parseable by CI systems.

## Clarifications

### Session 2026-02-02

- Q: What is the implementation strategy given existing code from spec 014? → A: Refine & harden existing code (fix bugs, add tests, improve error handling)

## Assumptions

- This implementation builds upon and refines the existing Go codebase from spec 014 (`cmd/wave/`, `internal/`), focusing on hardening, testing, and bug fixes rather than rewriting from scratch.
- The underlying adapter CLIs (Claude Code, etc.) maintain stable interfaces for subprocess invocation.
- The host environment provides adapter binaries on PATH; Wave does not install them.
- Compilation tools for contract validation are available; if not, compilation contracts degrade to syntax checks.
- Pipeline state uses local SQLite storage; distributed state is out of scope.
- The project will be implemented in Go 1.22+ as specified in the 014 plan.

## Scope Boundaries

**In scope**:
- CLI binary implementation (`wave init`, `validate`, `run`, `do`, `list`, `resume`, `clean`)
- Manifest parsing and validation
- Pipeline DAG execution engine
- Persona binding and permission enforcement
- Handover contract validation (JSON schema, TypeScript, test suites)
- Ephemeral workspace management
- State persistence for pipeline resumption
- Relay/compaction mechanism
- Matrix strategy execution
- Audit logging
- Claude Code adapter integration

**Out of scope**:
- GUI or web dashboard
- Multi-machine distributed execution
- Billing or token cost tracking
- Custom adapter protocol development (adapters wrap existing CLIs)
- LLM fine-tuning or model training
- User authentication or multi-tenancy
- Documentation site (covered by spec 014)
