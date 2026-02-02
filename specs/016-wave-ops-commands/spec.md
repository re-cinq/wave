# Feature Specification: Wave Ops Commands

**Feature Branch**: `016-wave-ops-commands`
**Created**: 2026-02-02
**Status**: Ready
**Input**: Add operational commands to Wave CLI for pipeline management, monitoring, and maintenance
**Clarifications**: See [clarifications.md](./clarifications.md) for detailed Q&A

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Pipeline Status Monitoring (Priority: P1)

A developer runs `wave status` to see the current state of running and recent pipelines. The command shows pipeline name, status (running/completed/failed/cancelled), current step, elapsed time, and token usage. For running pipelines, it shows real-time progress.

**Why this priority**: Developers need visibility into what Wave is doing, especially for long-running pipelines.

**Independent Test**: Start a pipeline in background, run `wave status`, verify it shows the running pipeline with accurate step information.

**Output Format**:
- Table columns: `RUN_ID | PIPELINE | STATUS | STEP | ELAPSED | TOKENS`
- Elapsed time format: `1m23s` (under 1 hour) or `1h23m` (longer)
- Token count shows total tokens (input + output combined)
- JSON output (`--format json`) includes separate `input_tokens` and `output_tokens` fields

**Acceptance Scenarios**:

1. **Given** a running pipeline, **When** the developer runs `wave status`, **Then** it shows pipeline name, current step, elapsed time, and token count.
2. **Given** multiple pipelines have run, **When** the developer runs `wave status --all`, **Then** it shows a table of recent pipelines with their final status.
3. **Given** a specific run ID, **When** the developer runs `wave status <run-id>`, **Then** it shows detailed information for that specific run.

---

### User Story 2 - Pipeline Logs and Output (Priority: P1)

A developer runs `wave logs` to view the output from pipeline executions. This includes adapter responses, contract validation results, and any errors. Logs can be filtered by step, persona, or time range.

**Why this priority**: Debugging failed pipelines requires access to execution logs. This is essential for troubleshooting.

**Independent Test**: Run a pipeline, use `wave logs` to retrieve output, verify it contains adapter responses and step transitions.

**Log Levels**:
- `--level all` (default): Shows everything including debug output from adapters
- `--level info`: Shows step transitions, artifact production, and warnings
- `--level error`: Shows only failures, contract violations, and exceptions
- `--errors` flag is an alias for `--level error`

**Log Retention**:
- Trace logs in `.wave/traces/` are retained indefinitely by default
- Use `wave clean --older-than <duration>` (e.g., `7d`, `30d`) for age-based cleanup

**Acceptance Scenarios**:

1. **Given** a completed pipeline, **When** the developer runs `wave logs`, **Then** it shows chronological output from all steps.
2. **Given** the `--step investigate` flag, **When** the developer runs `wave logs --step investigate`, **Then** only output from that step is shown.
3. **Given** a failed pipeline, **When** the developer runs `wave logs --errors`, **Then** only error messages and failed contract validations are shown.
4. **Given** `--follow` flag with a running pipeline, **When** the developer runs `wave logs --follow`, **Then** output streams in real-time.
5. **Given** `--level info` flag, **When** the developer runs `wave logs --level info`, **Then** step transitions and artifact production are shown without debug noise.

---

### User Story 3 - Workspace Cleanup (Priority: P1)

A developer runs `wave clean` to remove old workspaces and free disk space. Workspaces accumulate over time and can consume significant storage. The command supports selective cleanup by pipeline, age, or status.

**Why this priority**: Without cleanup, Wave directories grow unbounded. This is a maintenance necessity.

**Independent Test**: Create several pipeline workspaces, run `wave clean --keep-last 2`, verify only the 2 most recent are retained.

**Confirmation Behavior**:
- `wave clean --all` prompts: `This will remove all workspaces, state, and traces. Continue? [y/N]`
- `wave clean --pipeline <name>` prompts if workspaces exist for that pipeline
- `--force` suppresses all prompts (required for CI/scripts)
- `--dry-run` shows what would be deleted without prompting
- If stdin is not a TTY, defaults to declining unless `--force` is specified
- `--quiet` flag provides clean exit when nothing to clean (for scripting)

**Age-Based Cleanup**:
- `--older-than <duration>` accepts values like `7d`, `24h`, `30d`
- Applies to both workspaces and traces
- Recommended retention for development: 30 days

**No Automatic Cleanup**: Wave does not include a cleanup daemon. For scheduled cleanup, use system tools:
```bash
# Example cron job (daily at 2am, keep 5 most recent)
0 2 * * * cd /project && wave clean --keep-last 5 --force --quiet
```

**Acceptance Scenarios**:

1. **Given** multiple workspace directories exist, **When** the developer runs `wave clean --all`, **Then** a confirmation prompt is shown and all workspaces/state are removed after confirming.
2. **Given** `--keep-last 5` flag, **When** the developer runs `wave clean --keep-last 5`, **Then** only the 5 most recent workspaces per pipeline are retained.
3. **Given** `--pipeline debug` flag, **When** the developer runs `wave clean --pipeline debug`, **Then** only debug pipeline workspaces are cleaned.
4. **Given** `--dry-run` flag, **When** the developer runs `wave clean --dry-run`, **Then** it shows what would be deleted without removing anything.
5. **Given** `--force` flag, **When** the developer runs `wave clean --all --force`, **Then** cleanup proceeds without confirmation prompt.
6. **Given** `--older-than 7d` flag, **When** the developer runs `wave clean --older-than 7d`, **Then** only workspaces and traces older than 7 days are removed.

---

### User Story 4 - Pipeline Listing (Priority: P2)

A developer runs `wave list` to discover available pipelines, personas, and adapters. This helps users understand what's configured without reading YAML files directly.

**Why this priority**: Discoverability improves user experience. New users can explore available options.

**Independent Test**: Run `wave list pipelines`, verify it shows all pipeline names with descriptions.

**Acceptance Scenarios**:

1. **Given** a manifest with 3 pipelines, **When** the developer runs `wave list pipelines`, **Then** all 3 are shown with name and description.
2. **Given** `wave list personas`, **When** executed, **Then** all personas are shown with their allowed tools summary.
3. **Given** `wave list adapters`, **When** executed, **Then** all adapters are shown with their type and status.
4. **Given** `--format json` flag, **When** any list command runs, **Then** output is valid JSON for scripting.

---

### User Story 5 - Pipeline Cancellation (Priority: P2)

A developer runs `wave cancel` to stop a running pipeline. This is useful when a pipeline is taking too long or was started with incorrect input. Cancellation is graceful - the current step completes but no further steps start.

**Why this priority**: Users need control to stop runaway processes without killing the terminal.

**Independent Test**: Start a long pipeline, run `wave cancel`, verify it stops after the current step.

**Cancellation Mechanism**:
- **Graceful cancel** (`wave cancel`): Sets a cancellation flag in the database. The executor checks this flag between steps and stops gracefully. Current step completes normally. State is recorded as `cancelled`.
- **Force cancel** (`wave cancel --force`): Sends SIGTERM to the adapter process group, waits 5 seconds, then sends SIGKILL if still running. Partially-written artifacts remain but step state is recorded as `cancelled`.
- **Multiple pipelines**: With no arguments, cancels the most recently started running pipeline. Use `wave cancel <run-id>` to target a specific pipeline.

**State Handling**:
- Cancelled pipelines are recorded with status `cancelled` (distinct from `failed`)
- Partially-completed steps retain their artifacts
- `wave resume` can restart from the last completed step

**Acceptance Scenarios**:

1. **Given** a running pipeline, **When** the developer runs `wave cancel`, **Then** the pipeline stops after the current step completes and state shows `cancelled`.
2. **Given** `--force` flag, **When** the developer runs `wave cancel --force`, **Then** the current step is interrupted immediately via SIGTERM/SIGKILL.
3. **Given** no running pipelines, **When** the developer runs `wave cancel`, **Then** an informative message is shown and exit code is 0 (not an error).
4. **Given** multiple running pipelines, **When** the developer runs `wave cancel <run-id>`, **Then** only the specified pipeline is cancelled.

---

### User Story 6 - Artifact Inspection (Priority: P2)

A developer runs `wave artifacts` to view or export artifacts produced by pipeline steps. This allows inspection of intermediate results and extraction of outputs for use elsewhere.

**Why this priority**: Artifacts contain valuable intermediate work. Users should be able to access them.

**Independent Test**: Run a pipeline with output artifacts, use `wave artifacts` to list and export them.

**Artifact Discovery**:
- Artifacts are discovered from pipeline step `output_artifacts` declarations
- Only declared artifacts are listed (not all files in workspace)
- Raw adapter responses are NOT artifacts; access them via `wave logs`

**Artifact Metadata**:
- Name: The artifact name from the pipeline declaration
- Path: Absolute path to the file within the step workspace
- Size: File size in bytes
- Modified: Last modification timestamp

**Export Behavior**:
- `wave artifacts --export ./output` copies artifacts preserving directory structure
- Step directories are created: `./output/<step-id>/<artifact-name>`
- Existing files are overwritten without prompting

**Acceptance Scenarios**:

1. **Given** a completed pipeline, **When** the developer runs `wave artifacts`, **Then** all declared output artifacts are listed with step, name, path, and size.
2. **Given** `--export ./output` flag, **When** the developer runs `wave artifacts --export ./output`, **Then** all artifacts are copied to the specified directory preserving structure.
3. **Given** `--step investigate` flag, **When** executed, **Then** only artifacts from that step are shown/exported.
4. **Given** `--format json` flag, **When** executed, **Then** output is valid JSON including size and modification timestamp.

---

## Non-Functional Requirements

### Performance
- `wave status` should return within 100ms for local state queries
- `wave logs` should stream with <500ms latency
- `wave clean` should handle 1000+ workspaces efficiently

### Reliability
- All commands should handle concurrent pipeline execution safely
- State queries should not block running pipelines
- Cleanup should be atomic - no partial deletions

### User Experience
- All commands support `--help` with examples
- Error messages include actionable suggestions
- JSON output available for all list/status commands

## Technical Constraints

- Must use existing SQLite state database
- Must respect workspace isolation
- Must work with both local and CI execution modes

## Resolved Decisions

These questions were addressed during specification review. See [clarifications.md](./clarifications.md) for detailed rationale.

| Question | Decision |
|----------|----------|
| Log levels for `wave logs`? | Yes - support `--level all\|info\|error` with `--errors` as alias for `--level error` |
| Automatic cleanup mode? | No daemon - document cron patterns; add `--quiet` flag for scripting |
| Cancel mechanism? | Database flag for graceful; SIGTERM/SIGKILL to process group for force cancel |
| Log retention policy? | Indefinite by default; add `--older-than <duration>` to clean |
| Cleanup confirmation? | Prompt for `--all`; `--force` skips prompt; auto-decline if not TTY |
| Artifact discovery? | From `output_artifacts` declarations only; not raw adapter responses |
| Status output format? | Table with RUN_ID, PIPELINE, STATUS, STEP, ELAPSED, TOKENS columns |

## Terminology

- **Pipeline**: A definition file (`.wave/pipelines/*.yaml`)
- **Run**: A specific execution instance with a unique run ID
- `wave status` lists runs; `wave list pipelines` lists pipeline definitions

## Dependencies

- Depends on: 015-wave-cli-implementation (core CLI structure)
- Relates to: 014-manifest-pipeline-design (state schema)
