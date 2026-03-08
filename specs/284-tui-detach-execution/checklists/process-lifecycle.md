# Process Lifecycle Quality Checklist: Detach Pipeline Execution

**Feature**: `284-tui-detach-execution`
**Date**: 2026-03-08
**Scope**: Domain-specific validation for OS process management, IPC, and lifecycle semantics

---

## Process Management Requirements Quality

- [ ] CHK-PL001 - Are the failure modes of `exec.Command.Start()` enumerated (binary not found, permission denied, resource limits)? Are error handling requirements defined for each? [Completeness]
- [ ] CHK-PL002 - Is it specified whether `cmd.Process.Release()` is called before or after the PID is stored? Could a race cause a stored PID for an already-released handle? [Clarity]
- [ ] CHK-PL003 - Are requirements defined for what happens when `os.FindProcess` returns a process that belongs to a different user (permission denied on signal 0)? Is this distinguishable from "process not found"? [Completeness]
- [ ] CHK-PL004 - Does the spec address the possibility of `Setsid` failing (e.g., process is already a session leader)? Is fallback behavior defined? [Coverage]
- [ ] CHK-PL005 - Are resource limits (file descriptors, memory) for the detached subprocess specified, or does it inherit the parent's limits? [Completeness]

## IPC & SQLite Concurrency Requirements Quality

- [ ] CHK-PL006 - Are requirements defined for the scenario where the detached subprocess opens the SQLite database before the TUI has finished writing the run record? Is there a timing dependency? [Completeness]
- [ ] CHK-PL007 - Does the spec address what happens when the SQLite WAL checkpoint runs while both TUI and subprocess are active? Are there performance requirements? [Coverage]
- [ ] CHK-PL008 - Is the cancellation polling interval (5s) justified against the busy_timeout (5s)? Could a slow poll overlap with a database lock, causing the subprocess to miss a cancellation check? [Clarity]
- [ ] CHK-PL009 - Are the atomicity requirements for the PID storage step defined? Can the run record exist without a PID (between CreateRun and UpdateRunPID)? [Completeness]
- [ ] CHK-PL010 - Does the spec define whether the subprocess should open its own SQLite connection or reuse any connection pool? [Clarity]

## Signal Handling Requirements Quality

- [ ] CHK-PL011 - Are the signal handling requirements for the detached subprocess defined beyond Setsid? Should it handle SIGTERM, SIGINT, SIGUSR1 for specific behaviors? [Completeness]
- [ ] CHK-PL012 - Does the force-kill escalation (SIGKILL to `-pid`) account for grandchild processes spawned by adapters? Will they be in the same session? [Coverage]
- [ ] CHK-PL013 - Is the interaction between the adapter's `Setpgid: true` and the launcher's `Setsid: true` specified? Are adapter subprocesses in the same session but different process groups? [Clarity]
- [ ] CHK-PL014 - Are requirements defined for what signals the subprocess should mask or ignore during graceful shutdown to prevent double-handling? [Coverage]

## Platform & Environment Requirements Quality

- [ ] CHK-PL015 - Is the Windows incompatibility of `Setsid` explicitly documented as a known limitation with a clear scope boundary? [Completeness]
- [ ] CHK-PL016 - Does the spec address whether the detached subprocess inherits the parent's umask, locale, and terminal settings? [Coverage]
- [ ] CHK-PL017 - Are requirements defined for the behavior under containerized environments (Docker) where PID 1 might reap orphaned processes differently? [Coverage]
