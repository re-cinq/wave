# Tasks

## Phase 1: Persona Prompt Hardening

- [X] Task 1.1: Harden `github-commenter.md` — replace all inline `--body "<content>"` examples with `--body-file` pattern using temp files. Add explicit constraint: "NEVER interpolate untrusted content into --body or --title arguments. Always write content to a temp file and use --body-file." Update both `.wave/personas/` and `internal/defaults/personas/`.
  - Files: `.wave/personas/github-commenter.md`, `internal/defaults/personas/github-commenter.md`

- [X] Task 1.2: Harden `github-enhancer.md` — replace inline `--title "new title"` example with safe pattern. Reinforce the existing `--body-file` constraint. Update both copies.
  - Files: `.wave/personas/github-enhancer.md`, `internal/defaults/personas/github-enhancer.md`

- [X] Task 1.3: Harden `github-scoper.md` — replace inline `--body "<body>"` and `--title "<title>"` in `gh issue create` example with `--body-file` and heredoc patterns. Update both copies.
  - Files: `.wave/personas/github-scoper.md`, `internal/defaults/personas/github-scoper.md`

- [X] Task 1.4: Harden `gitea-commenter.md` — replace inline string interpolation examples with temp file patterns. Update both copies. [P]
  - Files: `.wave/personas/gitea-commenter.md`, `internal/defaults/personas/gitea-commenter.md`

- [X] Task 1.5: Harden `gitea-enhancer.md` — fix inline `--title "new title"` example. Update both copies. [P]
  - Files: `.wave/personas/gitea-enhancer.md`, `internal/defaults/personas/gitea-enhancer.md`

- [X] Task 1.6: Verify `bitbucket-enhancer.md` already uses safe temp file pattern — add explicit security constraint if missing. Update both copies. [P]
  - Files: `.wave/personas/bitbucket-enhancer.md`, `internal/defaults/personas/bitbucket-enhancer.md`

## Phase 2: Pipeline Prompt Template Hardening

- [X] Task 2.1: Harden `speckit-flow/create-pr.md` — replace inline `--body` and `--title` with safe patterns (heredoc or `--body-file`). Update both `.wave/prompts/` and `internal/defaults/prompts/`.
  - Files: `.wave/prompts/speckit-flow/create-pr.md`, `internal/defaults/prompts/speckit-flow/create-pr.md`

- [X] Task 2.2: Verify `gh-implement/create-pr.md` — confirm heredoc uses single-quoted delimiter (`<<'EOF'` not `<<EOF`). Fix if needed. Update both copies. [P]
  - Files: `.wave/prompts/gh-implement/create-pr.md`, `internal/defaults/prompts/gh-implement/create-pr.md`

- [X] Task 2.3: Verify `gh-implement-epic/report.md` — confirm heredoc uses single-quoted delimiter. Fix if needed. [P]
  - Files: `.wave/prompts/gh-implement-epic/report.md`

## Phase 3: Pipeline YAML Hardening

- [X] Task 3.1: Harden `gh-scope.yaml` — replace inline `--body "<body>"` in `gh issue create` example with `--body-file` pattern. Update both `.wave/pipelines/` and `internal/defaults/pipelines/`. [P]
  - Files: `.wave/pipelines/gh-scope.yaml`, `internal/defaults/pipelines/gh-scope.yaml`

- [X] Task 3.2: Harden `gh-rewrite.yaml` — replace inline `--body "body_template"` and `--title` with safe patterns. Update both copies. [P]
  - Files: `.wave/pipelines/gh-rewrite.yaml`, `internal/defaults/pipelines/gh-rewrite.yaml`

- [X] Task 3.3: Harden `gh-pr-review.yaml` — replace inline `--body` in `gh pr comment` with heredoc or `--body-file`. Update both copies. [P]
  - Files: `.wave/pipelines/gh-pr-review.yaml`, `internal/defaults/pipelines/gh-pr-review.yaml`

- [X] Task 3.4: Harden `dead-code-review.yaml` — replace inline `--body` in `gh pr comment`. [P]
  - Files: `.wave/pipelines/dead-code-review.yaml`

- [X] Task 3.5: Harden `wave-audit.yaml` — replace inline `--body` and `--title`. [P]
  - Files: `.wave/pipelines/wave-audit.yaml`

- [X] Task 3.6: Harden `recinq.yaml` — replace inline `--body` in comment commands. Update both copies. [P]
  - Files: `.wave/pipelines/recinq.yaml`, `internal/defaults/pipelines/recinq.yaml`

- [X] Task 3.7: Harden `gh-refresh.yaml` — verify `--title` uses safe pattern, confirm `--body-file` is already safe. Update both copies. [P]
  - Files: `.wave/pipelines/gh-refresh.yaml`, `internal/defaults/pipelines/gh-refresh.yaml`

- [X] Task 3.8: Harden multi-platform equivalents — update `gt-scope.yaml`, `gl-scope.yaml`, `gt-rewrite.yaml`, `gl-rewrite.yaml`, `gt-refresh.yaml`, `gl-refresh.yaml` with same safe patterns. [P]
  - Files: `internal/defaults/pipelines/gt-scope.yaml`, `internal/defaults/pipelines/gl-scope.yaml`, `internal/defaults/pipelines/gt-rewrite.yaml`, `internal/defaults/pipelines/gl-rewrite.yaml`, `internal/defaults/pipelines/gt-refresh.yaml`, `internal/defaults/pipelines/gl-refresh.yaml`

## Phase 4: Security Tests

- [X] Task 4.1: Add injection scenario tests to `internal/security/sanitize_test.go` — test that `containsShellMetachars` correctly detects all metachar patterns found in real GitHub issue content (backticks, `$()`, pipes, semicolons, etc.)
  - Files: `internal/security/sanitize_test.go`

- [X] Task 4.2: Add a test that validates all embedded persona markdown files do not contain unsafe `--body "` patterns (inline body arguments without --body-file)
  - Files: `internal/security/persona_audit_test.go`

## Phase 5: Documentation

- [X] Task 5.1: Create `docs/guides/secure-cli-patterns.md` — document safe patterns for constructing CLI commands in persona prompts: `--body-file`, single-quoted heredocs, `gh api` with JSON payloads. Include unsafe anti-patterns with explanations.
  - Files: `docs/guides/secure-cli-patterns.md`

- [X] Task 5.2: Update `docs/guides/github-integration.md` — add security subsection referencing the secure CLI patterns guide
  - Files: `docs/guides/github-integration.md`

- [X] Task 5.3: Update `docs/guide/personas.md` — add security note about CLI command construction in the persona authoring section
  - Files: `docs/guide/personas.md`

## Phase 6: Validation

- [X] Task 6.1: Run `go test ./...` to verify all tests pass including new security tests
- [X] Task 6.2: Run `go test -race ./...` to verify no race conditions
- [X] Task 6.3: Grep for remaining unsafe patterns: `--body "` and `--title "` across all persona/pipeline files to confirm none remain
