# Requirements Quality Review: Detach Pipeline Execution from TUI Process Lifecycle

**Feature**: `284-tui-detach-execution`
**Date**: 2026-03-08
**Scope**: Overall requirements quality validation across spec.md, plan.md, and tasks.md

---

## Completeness

- [ ] CHK001 - Are all four user stories traceable to at least one functional requirement? [Completeness]
- [ ] CHK002 - Does the spec define what happens to stdout/stderr from the detached subprocess? Are output streams redirected, discarded, or logged? [Completeness]
- [ ] CHK003 - Is the behavior defined for when the `wave` binary is not found at `os.Args[0]` (e.g., deleted, upgraded, or path-relative invocation)? [Completeness]
- [ ] CHK004 - Does the spec address how `--input` is passed when the input is large (e.g., multi-line text, file paths, or structured data exceeding shell arg limits)? [Completeness]
- [ ] CHK005 - Are requirements defined for what the TUI displays while a subprocess is being spawned (loading state between Launch() and PipelineLaunchedMsg)? [Completeness]
- [ ] CHK006 - Is there a requirement for how multiple concurrent detached pipelines interact with SQLite write contention beyond the existing WAL busy timeout? [Completeness]
- [ ] CHK007 - Does the spec define the behavior when the state database file is locked, corrupted, or inaccessible to the detached subprocess? [Completeness]
- [ ] CHK008 - Are there requirements for notifying the user about pipeline completion when the TUI is closed (e.g., desktop notification, exit code file)? [Completeness]
- [ ] CHK009 - Does the spec define log rotation or cleanup for events from completed detached runs in the SQLite event_log table? [Completeness]
- [ ] CHK010 - Is the working directory for the detached subprocess specified? Does it inherit the TUI's CWD or use a specific path? [Completeness]

## Clarity

- [ ] CHK011 - Is "reasonable grace period" in acceptance scenario 2 of Story 2 quantified? The SC-003 says 30 seconds, but the acceptance scenario is vague [Clarity]
- [ ] CHK012 - Does FR-001 clearly specify whether `os.Args[0]` or `os.Executable()` should be used for binary re-exec? The research doc and spec differ in specificity [Clarity]
- [ ] CHK013 - Is the boundary between "TUI-side monitoring state" (FR-004) and "detached pipeline state" clearly defined? What constitutes monitoring state? [Clarity]
- [ ] CHK014 - Is "within the existing refresh interval" (Story 4, acceptance 2) defined with a specific interval or is it deliberately left to the existing implementation? [Clarity]
- [ ] CHK015 - Does the spec clearly distinguish between the `--run` flag for resume (existing) and `--run` flag for pre-created run ID reuse (new)? Are there ambiguity risks? [Clarity]
- [ ] CHK016 - Is the phrase "graceful shutdown" in FR-006 defined in terms of specific actions (workspace cleanup, state persistence, adapter termination)? [Clarity]
- [ ] CHK017 - Are the error messages for stale run detection ("stale: subprocess exited unexpectedly" vs "process not found -- stale run") consistent between plan.md and research.md? [Clarity]

## Consistency

- [ ] CHK018 - Does the plan's `buildPassthroughEnv()` approach align with how the existing CLI `wave run` handles environment variables? Are there divergence risks? [Consistency]
- [ ] CHK019 - Is the `pid INTEGER DEFAULT 0` migration (data-model.md) consistent with `pid INTEGER` (research.md R7)? The DEFAULT clause differs between documents [Consistency]
- [ ] CHK020 - Does the task list (T013) claim `--run` already exists for resume, while plan D2 says "reuse it"? Is this flag actually implemented or aspirational? [Consistency]
- [ ] CHK021 - Is the cancellation polling interval (5 seconds in FR-006) consistent across all documents (spec, plan D4/D7, research R4, tasks T017)? [Consistency]
- [ ] CHK022 - Are the `PipelineLauncher` fields removed in T010 (cancelFns, buffers) consistent with the fields listed in data-model.md's refactored struct? [Consistency]
- [ ] CHK023 - Does the force-kill mechanism (T018) sending `SIGKILL` to process group (`-pid`) align with the subprocess's `Setsid: true` session isolation? Will the negative PID target the session or the process group? [Consistency]

## Coverage

- [ ] CHK024 - Are there acceptance scenarios covering the case where multiple pipelines are launched and the TUI is closed -- do all survive, not just one? [Coverage]
- [ ] CHK025 - Do the edge cases cover what happens when disk space is exhausted during subprocess spawn (not just during event writing as covered in edge case 6)? [Coverage]
- [ ] CHK026 - Are security requirements defined for preventing arbitrary command injection through the `--input` flag passed to the subprocess? [Coverage]
- [ ] CHK027 - Are there requirements for how the feature behaves under the Nix/bubblewrap sandbox? Does `Setsid` work within the sandbox's PID namespace? [Coverage]
- [ ] CHK028 - Are there requirements for what happens when the user upgrades the `wave` binary while a detached subprocess (old version) is still running? [Coverage]
- [ ] CHK029 - Do the tasks cover updating the WebUI and `wave status` CLI to display detached pipeline metadata (PID, detached flag) per SC-007? [Coverage]
- [ ] CHK030 - Is there test coverage specified for the race condition between `cmd.Start()` and TUI exit (edge case 5 / T029)? The task mentions it but no test task exists [Coverage]
- [ ] CHK031 - Are there requirements for the behavior when `env_passthrough` config is missing or empty? Does the subprocess get a bare environment? [Coverage]
- [ ] CHK032 - Does the spec address how the feature interacts with the existing `--from-step` and `--force` flags when launched from the TUI? [Coverage]
