# fix: PRs closed without merge incorrectly auto-close linked issues

**Issue**: [#379](https://github.com/re-cinq/wave/issues/379)
**Labels**: bug, ux, pipeline, priority: high
**Author**: nextlevelshit

## Problem

When Wave pipelines produce PRs that are later closed without merging (e.g., due to merge conflicts, review failures, or superseded implementations), GitHub automatically closes any linked issues referenced via `Closes #N` or `Fixes #N` keywords in the PR description or commit messages. This creates false positives — issues appear resolved when they were never actually implemented.

### Evidence

Recent examples of closed-without-merge PRs that may have falsely closed issues:
- PR #378 (closed, not merged) — duplicate of #375
- PR #373 (closed, not merged) — conflicts with #375
- PR #361, #360, #359, #357, #355, #354 — all closed without merge

## Root Cause

GitHub closes linked issues when a PR is closed (not just merged) if the PR body or commits contain closing keywords (`Closes`, `Fixes`, `Resolves`). Wave's `implement` and `speckit-flow` pipelines include these keywords in auto-generated PR descriptions.

## Acceptance Criteria

- [ ] Identify all sources of closing keywords in Wave pipeline PR templates/personas
- [ ] Replace closing keywords (`Closes`, `Fixes`, `Resolves`) with non-closing references (`Related to #N` or `For #N`) in all PR creation prompts
- [ ] Update the format validator (`internal/contract/format_validator.go`) to accept non-closing issue references instead of requiring closing keywords
- [ ] Update the format validator test to use non-closing references
- [ ] Update epic report search patterns to also find PRs using non-closing references
- [ ] Audit pipeline search patterns (wave-audit) should continue to search for both old closing keywords AND new non-closing references (for backward compatibility with historical PRs)
- [ ] Document the chosen strategy

## Additional Context

Comment from @nextlevelshit:
> in general we should have some after impl pipelines like updating the docs, changelogs, readme or anything else useful after or right before merging, like replying to a code review and triage the issues found ...
