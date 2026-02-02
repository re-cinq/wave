# Specification Clarifications: Wave Ops Commands

**Document**: specs/016-wave-ops-commands/spec.md
**Created**: 2026-02-02
**Status**: Resolved

This document captures questions about underspecified areas in the Wave Ops Commands specification and proposes sensible defaults that have been incorporated into the spec.

---

## Question 1: Log Retention Policy

**Issue**: The spec does not define how long logs/traces should be retained, or whether `wave clean` should have an age-based cleanup option (e.g., `--older-than 7d`).

**Proposed Default**:
- Trace logs in `.wave/traces/` are retained indefinitely by default.
- Add `--older-than <duration>` flag to `wave clean` that accepts values like `7d`, `24h`, `30d`.
- Default retention recommendation: 30 days for most development workflows.
- `wave clean --all` removes all traces; `wave clean --keep-last N` only affects workspaces, not traces.

**Rationale**: Traces contain valuable debugging information. Users should explicitly choose to delete them. The existing `wave clean` implementation only cleans workspaces with `--keep-last`, which is sensible.

---

## Question 2: Status Output Format Specifics

**Issue**: The spec mentions `wave status` shows "pipeline name, status, current step, elapsed time, and token usage" but does not specify the exact table format, column widths, or what "token usage" means (input vs output vs total).

**Proposed Default**:
- Table format columns: `RUN_ID | PIPELINE | STATUS | STEP | ELAPSED | TOKENS`
- Token usage shows total tokens (input + output combined).
- Elapsed time format: `1m23s` for durations under an hour, `1h23m` for longer.
- Running pipelines show current step; completed/failed show final step.
- JSON format (with `--format json`) includes separate `input_tokens` and `output_tokens` fields for scripting.

**Rationale**: Total tokens is the most useful summary metric. JSON output can provide the breakdown for tooling that needs it.

---

## Question 3: Cleanup Safety (Confirm Before Delete?)

**Issue**: The spec mentions `--force` to "skip confirmation" for `wave clean --all`, but the current implementation (`clean.go`) does NOT prompt for confirmation. Should it?

**Proposed Default**:
- `wave clean --all` SHOULD prompt for confirmation: `This will remove all workspaces, state, and traces. Continue? [y/N]`
- `wave clean --pipeline <name>` prompts only if workspaces exist.
- `wave clean --dry-run` shows what would be deleted without prompting.
- `--force` suppresses all prompts (useful for CI/scripts).
- If stdin is not a TTY (piped input), default to declining unless `--force` is specified.

**Rationale**: Destructive operations should require explicit confirmation. The current implementation lacks this safety measure and should be updated.

---

## Question 4: Artifact Storage Format

**Issue**: The spec mentions `wave artifacts` lists artifacts with "step, name, and path" but does not specify:
- How artifacts are discovered (naming convention, manifest, or database)?
- Whether artifacts include the adapter's raw response or just declared output files.
- What metadata is stored (size, timestamp, checksum)?

**Proposed Default**:
- Artifacts are discovered from pipeline step `output_artifacts` declarations.
- Artifacts are the files at declared paths within the step workspace.
- Metadata includes: artifact name, file path, size in bytes, and last modified timestamp.
- Raw adapter responses are NOT artifacts; they are logged in traces if `audit.log_all_tool_calls` is enabled.
- `wave artifacts --export` copies only declared artifacts, preserving directory structure.

**Rationale**: Declared artifacts represent intentional handoff data. Raw adapter responses may contain sensitive information and are better accessed via `wave logs`.

---

## Question 5: Cancellation Behavior Edge Cases

**Issue**: The spec states "the pipeline stops after the current step completes" for graceful cancel, but does not address:
- What happens to partially-written artifacts?
- How is "cancelled" state recorded vs "failed"?
- What if multiple pipelines are running?
- Does `--force` actually kill the subprocess or just stop Wave?

**Proposed Default**:
- Graceful cancel (`wave cancel`): Sets a cancellation flag via the database. The executor checks this flag between steps and stops gracefully. Current step completes. State is recorded as `cancelled`.
- Force cancel (`wave cancel --force`): Sends SIGTERM to the adapter process group, waits 5 seconds, then SIGKILL. Partially-written artifacts remain but step state is recorded as `cancelled`.
- Multiple pipelines: `wave cancel` with no arguments cancels the most recently started running pipeline. Use `wave cancel <run-id>` to target a specific pipeline.
- No running pipelines: Exit with informative message and exit code 0 (not an error).

**Rationale**: Consistent with existing signal handling in `run.go`. Process group kill ensures child processes are terminated.

---

## Question 6: Log Level Support

**Issue**: Open question in spec asks whether `wave logs` should support log levels (debug/info/error).

**Proposed Default**:
- Support `--level` flag with values: `all` (default), `info`, `error`.
- `--level error` shows only failures, contract violations, and exceptions.
- `--level info` shows step transitions, artifact production, and warnings.
- `--level all` shows everything including debug output from adapters.
- Existing `--errors` flag is an alias for `--level error`.

**Rationale**: Log levels provide useful filtering without breaking backward compatibility.

---

## Question 7: Automatic Cleanup Mode

**Issue**: Open question in spec asks whether `wave clean` should have a scheduled/automatic mode.

**Proposed Default**:
- No automatic cleanup daemon or scheduler.
- Instead, document recommended cron/CI patterns in user docs.
- Add `--quiet` flag for clean exit when nothing to clean (for scripting).
- Example cron: `0 2 * * * cd /project && wave clean --keep-last 5 --force --quiet`

**Rationale**: Wave should remain a single-purpose CLI tool. Scheduled cleanup is best handled by existing system tools (cron, CI cleanup jobs).

---

## Question 8: Cancel Implementation (SIGTERM vs Token)

**Issue**: Open question asks whether `wave cancel` should send SIGTERM or use a cancellation token.

**Proposed Default**:
- Use a database-backed cancellation flag for graceful cancel.
- The executor polls this flag between steps (already have step transition points).
- For force cancel, send SIGTERM to process group (consistent with existing subprocess handling in `adapter.go`).
- The cancellation token approach (context cancellation) is used internally but the database flag allows external `wave cancel` commands to signal running pipelines.

**Rationale**: Database flag allows decoupled cancellation - the cancel command does not need to share process space with the running pipeline. Context cancellation is already used for Ctrl+C handling.

---

## Identified Conflicts/Ambiguities

### Conflict: Clean Command Missing Confirmation
The spec acceptance scenario states `wave clean --all` removes items "after confirmation", but the current `clean.go` implementation does not prompt. This is a **gap between spec and implementation** that should be resolved by adding confirmation prompts.

### Ambiguity: Status vs Logs Overlap
Both `wave status` and `wave logs` can show information about running pipelines. The distinction should be:
- `wave status`: Summary view, quick glance, shows metadata (elapsed time, tokens, step)
- `wave logs`: Full output, detailed view, shows actual content from adapter responses

### Ambiguity: Pipeline vs Run Terminology
The spec uses "pipeline" to refer to both the definition (YAML file) and an execution instance. Recommend:
- "Pipeline" = the definition (`.wave/pipelines/*.yaml`)
- "Run" = a specific execution instance with a run ID
- `wave status` lists runs, not pipeline definitions
- `wave list pipelines` lists pipeline definitions

---

## Summary of Decisions

| Area | Decision |
|------|----------|
| Log retention | Indefinite by default; add `--older-than` to clean |
| Status format | Table with RUN_ID, PIPELINE, STATUS, STEP, ELAPSED, TOKENS |
| Cleanup confirmation | Prompt for `--all`; `--force` skips prompt |
| Artifact discovery | From `output_artifacts` declarations in pipeline steps |
| Cancel graceful | Database flag checked between steps; state = `cancelled` |
| Cancel force | SIGTERM to process group, 5s timeout, then SIGKILL |
| Log levels | Support `--level all|info|error` |
| Automatic cleanup | No; document cron patterns instead |
