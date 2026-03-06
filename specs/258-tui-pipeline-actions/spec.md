# Feature Specification: TUI Finished Pipeline Interactions

**Feature Branch**: `258-tui-pipeline-actions`  
**Created**: 2026-03-06  
**Status**: Draft  
**Issue**: [#258](https://github.com/re-cinq/wave/issues/258) (part 7 of 10, parent: [#251](https://github.com/re-cinq/wave/issues/251))  
**Input**: Implement interactive actions for finished pipelines in the TUI. When a finished pipeline is selected and the user presses `Enter`, Wave opens an interactive chat session (Claude Code subprocess) in the pipeline's workspace context, suspending the TUI and restoring it on exit. `b` checks out the pipeline's worktree branch. `d` opens a diff view of the pipeline's changes. The header bar's branch display updates dynamically to show the selected finished pipeline's branch.

## Clarifications

The following ambiguities were identified and resolved during specification refinement:

### C1: Chat session subprocess — adapter reuse vs direct Claude Code invocation

**Ambiguity**: The issue says "opens an interactive chat session (Claude Code subprocess)" but doesn't specify whether this should reuse Wave's adapter infrastructure (`internal/adapter/claude.go`) or directly spawn the `claude` binary. The adapter is designed for non-interactive pipeline execution with structured output, not interactive terminal sessions.

**Resolution**: Spawn the `claude` binary directly as an interactive subprocess, bypassing Wave's adapter infrastructure entirely. The adapter is designed for pipeline step execution (structured prompts, contract validation, artifact I/O), none of which apply to an interactive chat session. The chat session is a raw Claude Code invocation in a specific working directory. Use `tea.Exec()` from Bubble Tea to suspend the TUI, run the subprocess, and restore the TUI on exit. The subprocess inherits stdin/stdout/stderr for full interactive control. The working directory is set to the pipeline's workspace path (the worktree directory where the pipeline ran).

### C2: Workspace path resolution for chat sessions

**Ambiguity**: The issue says "chat session inherits workspace directory and artifacts from completed pipeline run" but `RunRecord` only stores `BranchName`, not the workspace path. The workspace path is stored per-step in `StepStateRecord.WorkspacePath`, and a pipeline may have multiple steps with different workspace paths. Which workspace path should the chat session use?

**Resolution**: Derive the workspace path deterministically from `RunRecord.RunID` and `RunRecord.BranchName` using the executor's known path convention: `.wave/workspaces/<runID>/__wt_<sanitized_branch>/`. The `sanitizeBranchName()` function in `internal/pipeline/context.go` defines the sanitization rules (replace non-alphanumeric with `-`, collapse consecutive dashes, trim, limit to 50 chars). **Note**: The `StepStateRecord.WorkspacePath` column exists in the database schema but `SaveStepState()` always inserts `NULL` — the workspace path is only tracked in-memory during execution via `execution.WorkspacePaths`. Therefore, the path must be reconstructed rather than queried. `FetchFinishedDetail()` will compute the workspace path using this convention, then validate that the directory exists on disk before populating `FinishedDetail.WorkspacePath`. If the directory doesn't exist, `WorkspacePath` is left empty and the chat action shows an error message. This requires extending `DetailDataProvider.FetchFinishedDetail()` to also return the workspace path.

### C3: TUI suspend/resume mechanism with Bubble Tea

**Ambiguity**: The issue says "TUI suspends cleanly during chat session" and "TUI restores fully after chat session exits" but doesn't specify the Bubble Tea mechanism. Bubble Tea provides `tea.Exec()` for running external processes while the program is suspended, and `tea.Suspend` for manual suspension.

**Resolution**: Use `tea.Exec(exec.Command("claude"), func(err error) tea.Msg { return ChatSessionEndedMsg{Err: err} })`. This is Bubble Tea's standard mechanism for suspending the program, running an external command with full terminal control, and resuming when the command exits. The `ChatSessionEndedMsg` handler triggers a data refresh (the user may have made changes during the chat session) and returns focus to the left pane. No manual terminal state management is needed — Bubble Tea handles alternate screen exit/enter, raw mode teardown/setup, and signal handling.

### C4: Branch checkout — git checkout vs worktree checkout

**Ambiguity**: The issue says "`b` checks out the finished pipeline's worktree branch" but doesn't specify the mechanism. Wave pipelines run in worktrees (separate working directories with their own HEAD). "Checkout" could mean: (a) `git checkout <branch>` in the main repository, switching the user's working tree to the pipeline's branch, or (b) navigating to the worktree directory.

**Resolution**: Perform `git checkout <branch>` in the main repository working directory. This is the standard git workflow — the user wants to switch their main working tree to the pipeline's branch to review, test, or extend the pipeline's output. The operation runs as a background command (not `tea.Exec()`, since it's non-interactive) and the result is reported via a message. If the checkout fails (uncommitted changes, branch doesn't exist), the error is displayed in the detail pane as a transient error message. The header bar's branch display updates to reflect the new branch via a git state refresh after the checkout completes.

### C5: Diff view — inline viewport vs external pager

**Ambiguity**: The issue says "`d` opens a diff view of the pipeline's changes" but doesn't specify whether the diff is rendered inline in the TUI right pane or opened in an external pager (like `less` or `delta`). Inline rendering would require parsing and colorizing diff output. An external pager leverages existing tooling and user preferences.

**Resolution**: Use `tea.Exec()` to run `git diff main...<branch>` in an external pager, suspending the TUI. This approach leverages the user's configured `core.pager` (which may be `delta`, `diff-so-fancy`, or `less`), respects user preferences, handles large diffs naturally via scrolling, and avoids reimplementing diff rendering in the TUI. The TUI suspends and resumes exactly as with the chat session (C3). If the branch doesn't exist or has been deleted, the diff action shows an error message instead. The triple-dot syntax (`...`) in `git diff main...<branch>` automatically computes the merge-base, showing only changes introduced on the pipeline's branch since it diverged from `main`.

### C6: Header bar dynamic branch display — selection-driven updates

**Ambiguity**: The issue says "header bar branch display updates when a finished pipeline is selected" and "reverts to current branch when no finished pipeline is selected." The header already has `OverrideBranch` logic (set via `PipelineSelectedMsg.BranchName`), implemented in #253. The issue confirms this but the acceptance criteria also mention it, suggesting it may need refinement.

**Resolution**: The existing header bar `OverrideBranch` mechanism (from #253) already handles this. When a finished pipeline is selected in the left pane, `PipelineSelectedMsg` carries `BranchName`, which the header uses as `OverrideBranch`. When a non-finished item is selected (available, running, or section header), `BranchName` is empty and the header reverts to the current git branch. No new work is needed for the header bar itself — the existing implementation satisfies the acceptance criteria. However, after a branch checkout (`b` key), a git state refresh must be triggered so the header shows the updated current branch.

### C7: Key binding activation — focus-gated vs always-active

**Ambiguity**: The issue lists `Enter`, `b`, and `d` as actions on finished pipelines but doesn't specify whether they require the right pane to be focused (consistent with how form/live-output work in #256/#257) or work from the left pane.

**Resolution**: The `Enter` key transitions focus from left pane to right pane (opening the chat session). The `b` and `d` keys are active when the right pane is focused and showing `stateFinishedDetail`. This follows the established pattern from #256 (form keys active in right pane) and #257 (display flag toggles active in right pane). The status bar updates to show `"[Enter] Chat  [b] Branch  [d] Diff  [Esc] Back"` when the right pane is focused on a finished detail view. This requires a new message type (e.g., `FinishedDetailActiveMsg{Active bool}`) to signal the status bar, following the same pattern as `FormActiveMsg` and `LiveOutputActiveMsg`.

### C8: Error feedback for failed actions

**Ambiguity**: The issue says "appropriate feedback shown if branch checkout or diff fails" but doesn't specify where or how errors are displayed.

**Resolution**: Action errors are displayed as a transient message in the right pane, replacing the action hints section at the bottom of the finished detail view. The error message is styled in red and includes the error detail (e.g., "Branch checkout failed: you have uncommitted changes"). The error clears when the user presses any key or navigates away. This follows the existing `stateError` pattern in the detail model but is scoped to the action hints area rather than replacing the entire detail view, so the user can still see the pipeline summary while understanding why the action failed.

### C9: Chat session exit — key binding behavior

**Ambiguity**: The issue says "`Esc` or `q` from chat returns to the TUI" but in an interactive Claude Code session, `Esc` and `q` are handled by Claude Code itself (e.g., `Esc` dismisses autocomplete, `q` is a regular character). The user exits Claude Code using its own exit mechanism (`/exit`, Ctrl-C, Ctrl-D).

**Resolution**: The TUI does not intercept keys during the chat session — the subprocess has full terminal control via `tea.Exec()`. The user exits Claude Code using its native exit mechanism (`/exit` command, or Ctrl-C/Ctrl-D). When the Claude Code process exits (regardless of how), `tea.Exec()` calls its completion callback, which sends `ChatSessionEndedMsg` to the TUI, triggering resume. The issue's mention of "Esc or q from chat" refers to the conceptual user experience, not literal key bindings — the user exits the chat and returns to the TUI.

### C10: Workspace existence validation before actions

**Ambiguity**: After a pipeline completes, the worktree may have been cleaned up (by Wave's cleanup process or manual deletion). The chat and diff actions depend on the workspace/branch still existing.

**Resolution**: Before launching a chat session, validate that the workspace directory exists. Before running branch checkout or diff, validate that the branch exists (via `git rev-parse --verify <branch>`). If validation fails, display a descriptive error: "Workspace directory no longer exists — the worktree may have been cleaned up" or "Branch '<name>' no longer exists." The `BranchDeleted` flag already exists on `PipelineSelectedMsg` and affects the `[b]` hint rendering (faint style when deleted) — extend this to also disable the `b` and `d` key handlers when the branch is known to be deleted.

### C11: `BranchDeleted` flag population — detection timing and mechanism

**Ambiguity**: The `BranchDeleted` field exists on `PipelineSelectedMsg` (from #253) and is consumed by the header and detail models, but the pipeline list's `emitSelectionMsg()` never sets it to `true` — it is only populated in test code. When and how should branch existence be checked?

**Resolution**: Check branch existence during the `FetchFinishedDetail()` call rather than on every cursor movement in the pipeline list. The `FetchFinishedDetail()` method already queries the state store and computes workspace path — it should also run `git rev-parse --verify <branch>` to determine if the branch still exists, and populate a `BranchDeleted` field on `FinishedDetail`. The detail model then uses this to gate key handlers and render faint hints. Checking on detail fetch (which happens once per selection) avoids the performance cost of spawning `git` processes on every cursor navigation in the list. The `PipelineSelectedMsg.BranchDeleted` field in the list remains unused for now — the detail model determines branch status from its own data.

### C12: Diff base branch — `main` hardcoded vs dynamic detection

**Ambiguity**: The spec uses `git diff main...<branch>` which hardcodes `main` as the base branch. Not all repositories use `main` — some use `master` or other default branch names. The `RunRecord` does not store the branch the pipeline was started from.

**Resolution**: Use `main` as the default base branch. This is the standard default branch name in modern Git and is used consistently across the Wave project. If `git diff main...<branch>` fails because `main` doesn't exist, the error propagates naturally through the `tea.Exec()` callback and is displayed as a transient error. A future enhancement could detect the default branch via `git symbolic-ref refs/remotes/origin/HEAD`, but this is out of scope for #258 and unnecessary given Wave's convention.

### C13: Status bar hint priority — `FinishedDetailActiveMsg` ordering

**Ambiguity**: The status bar (`statusbar.go`) has an existing priority chain: `formActive` → `liveOutputActive` → generic right-pane → left-pane. Adding `finishedDetailActive` needs a defined position in this chain.

**Resolution**: `finishedDetailActive` slots after `liveOutputActive` but before the generic right-pane fallback. The priority chain becomes: `formActive` → `liveOutputActive` → `finishedDetailActive` → generic right-pane → left-pane. This means if a form is active (which can't happen simultaneously with finished detail), the form hints take precedence. The finished detail hints replace the generic "scroll, back, quit" hints when the right pane shows a finished pipeline. The `FinishedDetailActiveMsg{Active: false}` is sent when focus returns to the left pane or when the detail state changes away from `stateFinishedDetail`.

### C14: `git checkout` working directory — project root determination

**Ambiguity**: FR-007 says "execute `git checkout <branch>` in the project root directory" but the TUI doesn't explicitly track the project root. The TUI process's CWD is wherever `wave tui` was launched from. Git commands need to run in the git repository root, not an arbitrary CWD.

**Resolution**: Use the process's current working directory (CWD) for git commands, which is the directory from which `wave tui` was invoked. Wave is designed to be run from the project root (where `wave.yaml` lives), and git commands naturally find the repository by walking up from CWD. The `exec.Command("git", "checkout", branch)` call does not need an explicit `Dir` set — the default CWD inheritance is correct. This matches how the header bar's `DefaultMetadataProvider` already runs `git` commands (e.g., `git rev-parse --abbrev-ref HEAD`) without specifying a directory.

### C15: `[d]` diff hint not fainted when branch is deleted

**Ambiguity**: The existing `renderFinishedDetail()` code (from prior PRs) faints the `[b]` checkout branch hint when `branchDeleted` is true, but does NOT faint the `[d]` view diff hint. The diff action also depends on the branch existing, so both should be fainted.

**Resolution**: FR-013 already states both `b` and `d` hints MUST be fainted when disabled. The implementation must update `renderFinishedDetail()` to also faint `[d]` when `branchDeleted` is true or `BranchName` is empty, making the rendering consistent with the key handler gating. The `[Enter]` chat hint is NOT fainted by branch deletion — chat depends on workspace path, not branch existence (a workspace may still exist even if the branch was force-deleted).

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Open Chat in Finished Pipeline Workspace (Priority: P1)

A developer has run a `speckit-flow` pipeline that completed successfully. They select the finished pipeline in the left pane and press Enter to focus the right pane, which shows the finished detail view with step results, artifacts, and action hints. The developer presses Enter again to open an interactive chat session. The TUI suspends, and Claude Code launches in the pipeline's workspace directory with all artifacts available. The developer iterates on the pipeline's output (e.g., refining a spec, fixing a failed test). When they exit Claude Code (via `/exit` or Ctrl-D), the TUI resumes fully — the same finished pipeline detail is displayed, and all TUI state is preserved.

**Why this priority**: Chat entry is the primary action for finished pipelines and the core feature of this issue. Without it, developers must manually navigate to the workspace directory and start Claude Code, breaking the TUI workflow.

**Independent Test**: Can be tested by selecting a finished pipeline with a valid workspace path, pressing Enter twice (focus right pane, then open chat), verifying the TUI suspends with `tea.Exec()` launching `claude` in the correct workspace directory, and verifying the TUI resumes after the process exits.

**Acceptance Scenarios**:

1. **Given** a finished pipeline is selected in the left pane, **When** the user presses Enter, **Then** focus moves to the right pane showing the finished detail view with action hints.
2. **Given** the right pane is focused on a finished detail view, **When** the user presses Enter, **Then** the TUI suspends and Claude Code launches as an interactive subprocess in the pipeline's workspace directory.
3. **Given** a chat session is active, **When** the user exits Claude Code, **Then** the TUI resumes with the finished detail view displayed and all state preserved.
4. **Given** the finished pipeline's workspace directory has been deleted, **When** the user presses Enter to open chat, **Then** an error message is displayed: the workspace no longer exists.

---

### User Story 2 - Checkout Pipeline Branch (Priority: P1)

A developer views a finished pipeline's detail and wants to switch their main working tree to the pipeline's branch to run additional tests or review changes. They press `b` while the finished detail view is focused. Wave runs `git checkout <branch>` in the project root. The header bar updates to show the new branch. The developer can now work on the pipeline's branch in their normal editor/terminal workflow.

**Why this priority**: Branch checkout is the most common follow-up action after reviewing a pipeline's results. It bridges the gap between TUI-based pipeline monitoring and the developer's standard git workflow.

**Independent Test**: Can be tested by selecting a finished pipeline with a valid branch, pressing `b`, verifying `git checkout <branch>` is executed, and verifying the header bar updates to show the new branch.

**Acceptance Scenarios**:

1. **Given** the right pane is focused on a finished detail view with a valid branch, **When** the user presses `b`, **Then** `git checkout <branch>` is executed in the project root.
2. **Given** a successful branch checkout, **When** the checkout completes, **Then** the header bar refreshes to show the newly checked-out branch.
3. **Given** the branch has been deleted, **When** the user presses `b`, **Then** an error message is shown: "Branch '<name>' no longer exists."
4. **Given** the working tree has uncommitted changes, **When** the user presses `b`, **Then** an error message is shown with the git error (e.g., "Your local changes would be overwritten by checkout").

---

### User Story 3 - View Pipeline Diff (Priority: P2)

A developer wants to see what changes a finished pipeline made before deciding to check out or merge. They press `d` while viewing a finished pipeline's detail. The TUI suspends and opens a diff view showing the pipeline's changes compared to the base branch, rendered through the user's configured git pager. After reviewing the diff, the developer closes the pager and the TUI resumes.

**Why this priority**: Diff viewing is important for code review but is secondary to chat entry (which provides a richer interaction) and branch checkout (which enables immediate action). Some developers may prefer to review diffs in their IDE instead.

**Independent Test**: Can be tested by selecting a finished pipeline with changes on its branch, pressing `d`, verifying the TUI suspends and runs `git diff` with the appropriate branch range, and verifying the TUI resumes after the pager exits.

**Acceptance Scenarios**:

1. **Given** the right pane is focused on a finished detail view with a valid branch, **When** the user presses `d`, **Then** the TUI suspends and opens `git diff main...<branch>` in the user's configured pager.
2. **Given** the diff pager is open, **When** the user closes the pager (e.g., `q` in less), **Then** the TUI resumes with the finished detail view.
3. **Given** the branch has been deleted, **When** the user presses `d`, **Then** an error message is shown: "Branch '<name>' no longer exists."
4. **Given** the branch has no changes relative to the base, **When** the user presses `d`, **Then** the pager opens with empty output (no diff).

---

### User Story 4 - Dynamic Header Branch Display (Priority: P2)

A developer navigates through the left pane, selecting different pipelines. When they select a finished pipeline, the header bar's branch display updates to show that pipeline's branch name. When they move the cursor to an available or running pipeline, the header reverts to showing the current git branch. This provides immediate context about which pipeline's branch they're looking at without having to read the detail pane.

**Why this priority**: Dynamic branch display enhances situational awareness but is not strictly required for the core interactions (chat, checkout, diff). The existing header already supports this via `OverrideBranch` from #253.

**Independent Test**: Can be tested by navigating between finished and non-finished pipelines and verifying the header bar's branch display updates accordingly.

**Acceptance Scenarios**:

1. **Given** the user selects a finished pipeline in the left pane, **When** the header renders, **Then** the branch display shows the finished pipeline's branch name.
2. **Given** a finished pipeline is selected and the user moves to an available pipeline, **When** the header renders, **Then** the branch display reverts to the current git branch.
3. **Given** the finished pipeline's branch has been deleted, **When** the header renders, **Then** the branch display shows the branch name with a "(deleted)" annotation.

---

### User Story 5 - Status Bar Context Hints for Finished Detail (Priority: P3)

When the right pane is focused on a finished pipeline detail view, the status bar at the bottom updates to show the available key bindings: `[Enter] Chat  [b] Branch  [d] Diff  [Esc] Back`. This helps users discover the available actions without memorization.

**Why this priority**: Status bar hints improve discoverability but are not essential for functionality — the action hints are already rendered within the finished detail view itself.

**Independent Test**: Can be tested by focusing the right pane on a finished detail view and verifying the status bar displays the correct key bindings.

**Acceptance Scenarios**:

1. **Given** the right pane is focused on a finished detail view, **When** the status bar renders, **Then** it shows `"[Enter] Chat  [b] Branch  [d] Diff  [Esc] Back"`.
2. **Given** the user presses Esc to return to the left pane, **When** the status bar renders, **Then** it reverts to the default pipeline navigation hints.

---

### Edge Cases

- What happens when the user presses `b` or `d` while the left pane is focused? The keys are ignored — action keys only work when the right pane is focused on `stateFinishedDetail`, following the established focus-gating pattern.
- What happens when the Claude Code binary is not found in PATH? The `tea.Exec()` callback receives an error, which is displayed as an error message: "Claude Code not found — is `claude` installed?"
- What happens when the user resizes the terminal during a suspended TUI (chat or diff)? Bubble Tea handles terminal resize on resume — the TUI re-renders at the new dimensions.
- What happens when a pipeline has no branch name (older runs or runs without worktrees)? The `[b]` and `[d]` hints are rendered in faint style and the key handlers are disabled. The `[Enter]` chat action checks for workspace path independently of branch name.
- What happens when the user opens chat, makes changes, and returns — does the detail view refresh? Yes. `ChatSessionEndedMsg` triggers a data refresh (re-fetching the finished detail from the state store) to reflect any changes the user made during the chat session.
- What happens when the user presses `b` while already on the pipeline's branch? The checkout is a no-op (git reports "Already on '<branch>'"). No error is shown.
- What happens when the user presses Enter on a failed pipeline? The chat session still opens in the workspace directory — the user may want to debug the failure interactively. All pipeline artifacts up to the failure point are available.
- What happens when `git diff` output is extremely large? The external pager handles this naturally — `less` (or the user's configured pager) provides scrolling without TUI memory concerns.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: When the right pane is focused on `stateFinishedDetail` and the user presses Enter, the system MUST suspend the TUI and launch an interactive Claude Code subprocess (`claude`) in the finished pipeline's workspace directory using `tea.Exec()`.
- **FR-002**: The Claude Code subprocess MUST inherit stdin, stdout, and stderr for full interactive terminal control. The subprocess MUST have its working directory set to the pipeline's workspace path.
- **FR-003**: When the Claude Code subprocess exits (by any means), the TUI MUST resume fully — restoring the alternate screen, the finished detail view, and all component state.
- **FR-004**: After chat session exit, the system MUST trigger a data refresh (re-fetch finished detail, git state) to reflect any changes made during the chat session.
- **FR-005**: Before launching a chat session, the system MUST validate that the workspace directory exists. If it does not, the system MUST display an error message instead of launching.
- **FR-006**: The `FinishedDetail` data projection MUST be extended to include the workspace path. The path MUST be derived deterministically from `RunRecord.RunID` and `RunRecord.BranchName` using the convention `.wave/workspaces/<runID>/__wt_<sanitized_branch>/`, then validated for existence on disk. If the directory does not exist, `WorkspacePath` is empty.
- **FR-007**: When the right pane is focused on `stateFinishedDetail` and the user presses `b`, the system MUST execute `git checkout <branch>` using the process's current working directory (inherited from `wave tui` invocation).
- **FR-008**: Before executing branch checkout, the system MUST validate that the branch exists (via `git rev-parse --verify <branch>`). If it does not, the system MUST display an error message.
- **FR-009**: After a successful branch checkout, the system MUST trigger a header bar git state refresh so the branch display updates to show the new current branch.
- **FR-010**: When the right pane is focused on `stateFinishedDetail` and the user presses `d`, the system MUST suspend the TUI and run `git diff main...<branch>` through the user's configured git pager using `tea.Exec()`. The triple-dot syntax automatically computes the merge-base.
- **FR-011**: Before executing diff, the system MUST validate that the branch exists. If it does not, the system MUST display an error message.
- **FR-012**: The diff MUST be computed against `main` as the base branch, using `git diff main...<branch>` (triple-dot). This shows only the pipeline's changes since it diverged from `main`.
- **FR-013**: The `b` and `d` key handlers MUST be disabled when `FinishedDetail.BranchDeleted` is true or when `FinishedDetail.BranchName` is empty. The `[b]` and `[d]` action hints in the detail view MUST both be rendered in faint style when disabled. The `[Enter]` chat hint is NOT fainted by branch deletion — it depends on workspace path, not branch existence.
- **FR-014**: Action errors (checkout failure, diff failure, missing workspace) MUST be displayed as transient messages in the right pane's action hints area, styled in red, clearing on next key press or navigation.
- **FR-015**: The `Enter`, `b`, and `d` key handlers MUST only be active when the right pane is focused and showing `stateFinishedDetail`. They MUST be ignored in all other contexts (left pane focused, other detail states).
- **FR-016**: The header bar MUST continue to update its branch display dynamically when a finished pipeline is selected (showing the pipeline's branch) and revert to the current git branch when a non-finished item is selected. This uses the existing `OverrideBranch` mechanism from #253.
- **FR-017**: The status bar MUST display context-appropriate hints when the right pane is focused on `stateFinishedDetail`: action key bindings for chat, branch, diff, and back. A `FinishedDetailActiveMsg{Active bool}` MUST signal the status bar to switch hint text. Priority order: `formActive` → `liveOutputActive` → `finishedDetailActive` → generic right-pane → left-pane.
- **FR-018**: All action commands (`claude`, `git checkout`, `git diff`) MUST respect `NO_COLOR` by passing the environment variable through to subprocesses.

### Key Entities

- **ChatSessionEndedMsg**: Message sent when the Claude Code subprocess exits. Carries an optional error. Triggers data refresh and state restoration in the detail model.
- **BranchCheckoutMsg**: Message carrying the result of a branch checkout attempt. Includes success/failure status and any error message. Triggers header refresh on success or error display on failure.
- **DiffViewEndedMsg**: Message sent when the diff pager subprocess exits. Triggers TUI resume and state restoration.
- **FinishedDetailActiveMsg**: Message signaling the status bar that the right pane is showing a finished detail view. Carries `Active bool` to switch hint text.
- **FinishedDetail.WorkspacePath**: New field on the existing `FinishedDetail` struct. Contains the filesystem path to the pipeline's workspace directory, derived from `RunID` + `BranchName` using the executor's path convention and validated for existence.
- **FinishedDetail.BranchDeleted**: New field on the existing `FinishedDetail` struct. Set by `FetchFinishedDetail()` via `git rev-parse --verify <branch>` — `true` if the branch no longer exists. Used to gate `b`/`d` key handlers and faint their hints.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Pressing Enter on a focused finished detail view launches Claude Code in the correct workspace directory — verified by unit tests with mock `tea.Exec()` capturing the command and working directory.
- **SC-002**: TUI suspends cleanly and resumes fully after chat session exit — verified by integration tests asserting alternate screen state transitions and component state preservation.
- **SC-003**: Pressing `b` executes `git checkout <branch>` and the header bar updates to reflect the new branch — verified by unit tests with mock git commands and header state assertions.
- **SC-004**: Pressing `d` launches `git diff` in an external pager and the TUI resumes after pager exit — verified by unit tests with mock `tea.Exec()` capturing the diff command.
- **SC-005**: Action keys (`Enter`, `b`, `d`) are ignored when the right pane is not focused on `stateFinishedDetail` — verified by unit tests sending key events in other states and asserting no state change.
- **SC-006**: Missing workspace directory or deleted branch produces appropriate error messages instead of crashes — verified by tests with non-existent paths and branches.
- **SC-007**: Status bar displays finished-detail-specific hints when the right pane is focused on a finished detail view — verified by unit tests asserting status bar content.
- **SC-008**: All existing TUI tests continue to pass after integration — the finished pipeline actions do not break existing pipeline list, detail, header, status bar, launch flow, or live output components.
- **SC-009**: Error messages for failed actions are transient and clear on next user interaction — verified by tests asserting error display and subsequent clearing.
- **SC-010**: Branch checkout and diff commands handle edge cases (already on branch, empty diff, uncommitted changes) without crashing — verified by tests covering each edge case.
