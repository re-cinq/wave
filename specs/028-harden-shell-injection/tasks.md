# Tasks

## Phase 1: Security Module Enhancement

- [X] Task 1.1: Add `ShellEscape()` function to `internal/security/sanitize.go` that single-quotes strings and escapes interior single quotes using the `'\''` pattern
- [X] Task 1.2: Add comprehensive tests for `ShellEscape()` in `internal/security/sanitize_test.go` covering all POSIX metacharacters, command substitution (`$()`, backticks), empty strings, and realistic malicious issue titles

## Phase 2: GitHub Persona Hardening

- [X] Task 2.1: Update `internal/defaults/personas/github-enhancer.md` to use `--body-file` pattern and heredoc for title edits
- [X] Task 2.2: Update `.wave/personas/github-enhancer.md` to mirror Task 2.1
- [X] Task 2.3: Update `internal/defaults/personas/github-commenter.md` to use `--body-file` for long content and heredoc for short content [P]
- [X] Task 2.4: Update `.wave/personas/github-commenter.md` to mirror Task 2.3 [P]
- [X] Task 2.5: Update `internal/defaults/personas/github-scoper.md` to use `--body-file` for issue creation bodies and heredoc for titles [P]
- [X] Task 2.6: Update `.wave/personas/github-scoper.md` to mirror Task 2.5 [P]

## Phase 3: GitLab Persona Hardening

- [X] Task 3.1: Update `internal/defaults/personas/gitlab-enhancer.md` to use safe arg-passing patterns [P]
- [X] Task 3.2: Update `.wave/personas/gitlab-enhancer.md` to mirror Task 3.1 [P]
- [X] Task 3.3: Update `internal/defaults/personas/gitlab-commenter.md` to use heredoc/file patterns [P]
- [X] Task 3.4: Update `.wave/personas/gitlab-commenter.md` to mirror Task 3.3 [P]
- [X] Task 3.5: Update `internal/defaults/personas/gitlab-scoper.md` to use safe patterns [P]
- [X] Task 3.6: Update `.wave/personas/gitlab-scoper.md` to mirror Task 3.5 [P]

## Phase 4: Gitea Persona Hardening

- [X] Task 4.1: Update `internal/defaults/personas/gitea-enhancer.md` to use safe arg-passing patterns [P]
- [X] Task 4.2: Update `.wave/personas/gitea-enhancer.md` to mirror Task 4.1 [P]
- [X] Task 4.3: Update `internal/defaults/personas/gitea-commenter.md` to use safe patterns [P]
- [X] Task 4.4: Update `.wave/personas/gitea-commenter.md` to mirror Task 4.3 [P]
- [X] Task 4.5: Update `internal/defaults/personas/gitea-scoper.md` to use safe patterns [P]
- [X] Task 4.6: Update `.wave/personas/gitea-scoper.md` to mirror Task 4.5 [P]

## Phase 5: Pipeline Prompt Hardening

- [X] Task 5.1: Update `internal/defaults/prompts/speckit-flow/create-pr.md` to use heredoc for `--body` [P]
- [X] Task 5.2: Update `.wave/prompts/speckit-flow/create-pr.md` to mirror Task 5.1 [P]

## Phase 6: Documentation

- [X] Task 6.1: Add "Secure CLI Patterns" section to `docs/guides/github-integration.md` with examples of safe heredoc, `--body-file`, and `gh api` patterns
- [X] Task 6.2: Add shell injection warning and safe patterns guidance to `docs/guide/personas.md`

## Phase 7: Validation

- [X] Task 7.1: Run `go test ./...` to verify no regressions
- [X] Task 7.2: Run `go test -race ./...` for race detector pass
- [X] Task 7.3: Grep all `.md` persona and prompt files to verify no remaining unsafe inline `--body "` or `--title "` patterns with user-interpolated content
