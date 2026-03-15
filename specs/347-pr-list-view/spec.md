# feat(tui): add Pull Requests list view mirroring the Issues list panel

**Issue**: [#347](https://github.com/re-cinq/wave/issues/347)
**Labels**: enhancement
**Author**: nextlevelshit
**State**: OPEN

## Summary

Add a Pull Requests list view to the Wave TUI that mirrors the existing Issues list panel, allowing users to browse, filter, and inspect open PRs directly from the terminal interface.

## Context

The TUI currently has an Issues list view. Users need equivalent functionality for Pull Requests so they can manage their full GitHub workflow without leaving the terminal.

## Acceptance Criteria

- [ ] A new "Pull Requests" tab/panel is available in the TUI navigation
- [ ] PR list displays: number, title, author, status (open/draft/review), updated date
- [ ] PR list supports the same filtering and sorting capabilities as the Issues list
- [ ] Selecting a PR shows a detail view with description, review status, and checks
- [ ] Keyboard navigation matches the Issues list UX (same keybindings where applicable)
- [ ] PR data is fetched via the GitHub API client (`ListPullRequests`, `GetPullRequest`)

## Technical Notes

- Reuse list component patterns from the existing Issues list implementation
- Follow the Bubble Tea component architecture established in the TUI package (`internal/tui/`)
- Ensure the PR model struct mirrors the Issue model where fields overlap
- The GitHub client already exposes `ListPullRequests` and `GetPullRequest` APIs plus a `PullRequest` type in `internal/github/types.go`

## Out of Scope

- PR creation or editing from the TUI
- Merge/close actions (read-only for this issue)
