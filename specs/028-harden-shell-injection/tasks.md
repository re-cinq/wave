# Tasks

## Phase 1: Security Module Enhancement

- [X] Task 1.1: Add `ShellEscape()` function to `internal/security/sanitize.go` that escapes POSIX shell metacharacters in a string by wrapping in single quotes and escaping internal single quotes
- [X] Task 1.2: Add table-driven unit tests for `ShellEscape()` in `internal/security/sanitize_test.go` covering all metacharacters, empty strings, strings with single quotes, multi-byte Unicode, and known injection payloads

## Phase 2: GitHub Persona Hardening

- [X] Task 2.1: Harden `github-enhancer.md` — replace inline `--title "..."` with write-to-file + `gh api` JSON payload pattern; enforce `--body-file` for all body content [P]
- [X] Task 2.2: Harden `github-commenter.md` — replace all inline `--body "..."` patterns with `--body-file /tmp/wave-comment.md`; replace inline `--title "..."` in PR creation with write-to-file pattern [P]
- [X] Task 2.3: Harden `github-scoper.md` — replace inline `--title "..."`, `--body "..."`, `--label "..."` in `gh issue create` with `gh api` JSON payload written to temp file [P]
- [X] Task 2.4: Sync all GitHub persona changes from `.wave/personas/` to `internal/defaults/personas/` (files must be identical)

## Phase 3: GitLab and Gitea Persona Hardening

- [X] Task 3.1: Harden `gitea-enhancer.md` — replace inline `--title "..."` with write-to-file + heredoc pattern [P]
- [X] Task 3.2: Harden `gitea-commenter.md` — replace inline content args with write-to-file patterns [P]
- [X] Task 3.3: Harden `gitlab-enhancer.md` — replace inline `--title "..."` with write-to-file + heredoc pattern [P]
- [X] Task 3.4: Harden `gitlab-commenter.md` — replace inline `--message "..."` and creation args with write-to-file patterns [P]
- [X] Task 3.5: Sync all GitLab/Gitea persona changes from `.wave/personas/` to `internal/defaults/personas/`
- [X] Task 3.6: Harden `gitea-scoper.md` — replace inline `--title`, `--body`, `--labels` with write-to-file patterns (discovered during validation)
- [X] Task 3.7: Harden `gitlab-scoper.md` — replace inline `--title`, `--description`, `--label` with write-to-file patterns (discovered during validation)

## Phase 4: Prompt File Hardening

- [X] Task 4.1: Harden `.wave/prompts/speckit-flow/create-pr.md` — ensure `gh pr create` uses heredoc or `--body-file` for PR body
- [X] Task 4.2: Sync speckit-flow prompt to `internal/defaults/prompts/speckit-flow/create-pr.md`

## Phase 5: Security Tests

- [X] Task 5.1: Add injection-specific test cases to `internal/security/sanitize_test.go` — verify `containsShellMetachars` and risk scoring for payloads like `$(whoami)`, `` `id` ``, `; rm -rf /`, `| cat /etc/passwd`
- [X] Task 5.2: Add `TestShellEscape_RealWorldPayloads` covering OWASP command injection examples and GitHub issue title/body edge cases (emoji, Unicode, nested quotes)

## Phase 6: Documentation

- [X] Task 6.1: Add "Secure CLI Patterns for Persona Authors" section to `docs/guides/custom-personas.md` with examples of safe vs unsafe patterns, covering `--body-file`, heredocs, `gh api` JSON, and the `ShellEscape` utility
- [X] Task 6.2: Add security warning section to `docs/guides/github-integration.md` about shell injection risks when constructing CLI commands with untrusted input

## Phase 7: Validation

- [X] Task 7.1: Run `go test ./...` to verify all tests pass
- [X] Task 7.2: Run `go test -race ./...` to verify no race conditions
- [X] Task 7.3: Grep all persona and prompt files for remaining inline `--body "` and `--title "` patterns to confirm none use shell-interpolated untrusted content
