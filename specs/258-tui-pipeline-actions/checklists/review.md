# Quality Review Checklist: TUI Finished Pipeline Interactions

**Purpose**: Validate requirements quality across Completeness, Clarity, Consistency, and Coverage dimensions.
**Created**: 2026-03-06
**Feature**: [spec.md](../spec.md) | [plan.md](../plan.md) | [tasks.md](../tasks.md)

## Completeness

- [ ] CHK101 - Are all three core actions (chat, checkout, diff) fully specified with trigger conditions, preconditions, execution mechanism, and result handling? [Completeness]
- [ ] CHK102 - Does every functional requirement (FR-001 through FR-018) have at least one acceptance scenario or success criterion that verifies it? [Completeness]
- [ ] CHK103 - Are data flow paths defined for all new message types (ChatSessionEndedMsg, BranchCheckoutMsg, DiffViewEndedMsg, FinishedDetailActiveMsg) from emission through routing to final handler? [Completeness]
- [ ] CHK104 - Is the workspace path derivation algorithm fully specified including sanitization rules, fallback strategy (glob), and validation? [Completeness]
- [ ] CHK105 - Are lifecycle requirements defined for transient error state — when it appears, how it's styled, and when it clears? [Completeness]
- [ ] CHK106 - Is the status bar hint priority chain fully ordered with all possible states documented, including what happens when multiple states overlap? [Completeness]
- [ ] CHK107 - Are subprocess environment requirements defined — specifically which environment variables (NO_COLOR, PATH) must be inherited by chat, checkout, and diff subprocesses? [Completeness]

## Clarity

- [ ] CHK201 - Is the distinction between "focus right pane on finished detail" (first Enter) and "launch chat session" (second Enter) unambiguous — can a developer implement the two-step Enter behavior without additional clarification? [Clarity]
- [ ] CHK202 - Is the workspace path convention (`.wave/workspaces/<RunID>/__wt_<sanitized_branch>/`) specified with enough precision to derive the exact path for any given RunID and branch? [Clarity]
- [ ] CHK203 - Are the conditions under which `[Enter]`, `[b]`, and `[d]` hints are fainted vs disabled vs hidden specified without ambiguity — is "fainted" purely visual or does it also mean non-functional? [Clarity]
- [ ] CHK204 - Is the error message content for each failure mode specified literally or by pattern — can a developer produce the exact error strings without judgment calls? [Clarity]
- [ ] CHK205 - Is the `git diff main...<branch>` triple-dot behavior explained sufficiently for someone unfamiliar with the merge-base computation? [Clarity]
- [ ] CHK206 - Are the key bindings explicitly enumerated for each TUI state, or could a developer be confused about which keys are active in which pane/state combination? [Clarity]

## Consistency

- [ ] CHK301 - Does the focus-gating pattern for `Enter`/`b`/`d` keys match the established patterns from #256 (form keys) and #257 (display flag toggles)? [Consistency]
- [ ] CHK302 - Does the `FinishedDetailActiveMsg` follow the exact same emission and handling pattern as `FormActiveMsg` and `LiveOutputActiveMsg`? [Consistency]
- [ ] CHK303 - Is the `tea.Exec()` suspend/resume pattern consistent between chat session (FR-001/003) and diff view (FR-010) — same callback structure, same resume behavior? [Consistency]
- [ ] CHK304 - Does the transient error display mechanism (FR-014) align with the existing `stateError` pattern in the detail model, or is the deviation from that pattern justified? [Consistency]
- [ ] CHK305 - Are the new `FinishedDetail` fields (`WorkspacePath`, `BranchDeleted`) consistent with the existing struct's naming conventions and population patterns? [Consistency]
- [ ] CHK306 - Does the header bar override branch behavior specified here match the existing `OverrideBranch` mechanism implemented in #253? [Consistency]
- [ ] CHK307 - Is the `git checkout` background command pattern (non-interactive `tea.Cmd`) consistent with how other TUI components handle background I/O? [Consistency]

## Coverage

- [ ] CHK401 - Are failure modes covered for all external dependencies — `claude` binary not found, `git` binary not found, workspace directory deleted, branch deleted, git index locked? [Coverage]
- [ ] CHK402 - Are concurrent action scenarios addressed — what happens if the user rapidly presses `b` then `d`, or presses `Enter` while a checkout is in progress? [Coverage]
- [ ] CHK403 - Are terminal state edge cases covered — terminal resize during subprocess, SIGTSTP/SIGCONT during subprocess, subprocess killed by external signal? [Coverage]
- [ ] CHK404 - Are the "already on branch" and "empty diff" non-error edge cases handled as graceful outcomes rather than error states? [Coverage]
- [ ] CHK405 - Are accessibility implications considered for fainted text — is the faint style sufficiently distinguishable for users with reduced contrast displays? [Coverage]
- [ ] CHK406 - Does the spec address what happens when `WorkspacePath` exists but is a non-worktree directory (e.g., corrupted or partially cleaned up workspace)? [Coverage]
- [ ] CHK407 - Are there requirements for how the TUI behaves if the user modifies or deletes the workspace during a chat session (workspace disappears between launch and return)? [Coverage]
