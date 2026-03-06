# Subprocess Lifecycle Checklist: TUI Finished Pipeline Interactions

**Purpose**: Validate requirements quality for subprocess management (chat session, diff pager, branch checkout).
**Created**: 2026-03-06
**Feature**: [spec.md](../spec.md)

## Subprocess Initiation

- [ ] CHK501 - Are precondition checks defined for every subprocess launch — workspace existence before chat, branch existence before checkout/diff? [Completeness]
- [ ] CHK502 - Is the working directory requirement specified for each subprocess type — chat uses workspace dir, checkout uses CWD, diff uses CWD? [Clarity]
- [ ] CHK503 - Are the exact command invocations specified for each action — `claude` (chat), `git checkout <branch>` (checkout), `git diff main...<branch>` (diff)? [Clarity]
- [ ] CHK504 - Is it clear which actions use `tea.Exec()` (suspend TUI) vs `tea.Cmd` (background) — and is the rationale for each choice documented? [Clarity]

## Terminal State Management

- [ ] CHK505 - Are terminal state transitions defined for `tea.Exec()` — alternate screen exit before subprocess, raw mode teardown, and restoration after? [Completeness]
- [ ] CHK506 - Is the stdin/stdout/stderr inheritance requirement explicit for the chat subprocess — does the spec confirm full interactive terminal control? [Completeness]
- [ ] CHK507 - Is the pager behavior for `git diff` specified — does the spec confirm that git's `core.pager` config is respected? [Clarity]

## Subprocess Completion

- [ ] CHK508 - Is the callback/message type defined for each subprocess exit — ChatSessionEndedMsg, BranchCheckoutMsg, DiffViewEndedMsg? [Completeness]
- [ ] CHK509 - Are post-completion side effects specified for each action — chat triggers data+git refresh, checkout triggers git refresh, diff is no-op? [Completeness]
- [ ] CHK510 - Is error propagation defined for each subprocess — how is a non-zero exit code from `claude`, `git checkout`, or `git diff` handled? [Completeness]
- [ ] CHK511 - Is the distinction between "subprocess failed to start" (binary not found) and "subprocess exited with error" (non-zero exit) addressed? [Coverage]

## Concurrency and Ordering

- [ ] CHK512 - Is it specified whether multiple actions can be in-flight simultaneously, or are they serialized (e.g., can a checkout be pending while the user opens a diff)? [Coverage]
- [ ] CHK513 - Is the behavior defined when a `BranchCheckoutMsg` arrives after the user has already navigated away from the finished detail view? [Coverage]
- [ ] CHK514 - Are there timing requirements for the data refresh after chat session exit — must the refresh complete before the user can interact, or is it asynchronous? [Clarity]
