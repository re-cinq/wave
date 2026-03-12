# security: Harden gh CLI commands against shell injection

**Issue**: [#28](https://github.com/re-cinq/wave/issues/28)
**Repository**: re-cinq/wave
**Labels**: bug
**Author**: nextlevelshit
**State**: OPEN

## Context

From Copilot review on PR #26: github-enhancer persona constructs shell commands with potentially untrusted issue content.

## Risk

If issue titles/bodies contain shell metacharacters like `$()` or backticks, they could be executed.

## Proposed Mitigations

1. Use `gh api` with JSON payloads instead of `gh issue edit` with shell arguments
2. Add heredoc patterns to persona prompts for safe body handling
3. Consider Wave security module integration for shell escaping

## Priority

Medium - current use is on trusted repos, but should be addressed before wider adoption.

## Acceptance Criteria

- [ ] All gh CLI commands in personas use safe argument passing (JSON payloads or heredocs)
- [ ] Shell metacharacters in issue content cannot trigger command execution
- [ ] Security tests added to verify injection resistance
- [ ] Documentation updated with secure CLI patterns for persona authors

## References

- OWASP Command Injection: https://owasp.org/www-community/attacks/Command_Injection
- PR #26 review comments

## Analysis

### Affected Persona Prompts (12 files across 4 forge platforms)

**GitHub personas** (in both `internal/defaults/personas/` and `.wave/personas/`):
- `github-enhancer.md` — instructs `gh issue edit <N> --title "new title"` with inline shell quotes
- `github-commenter.md` — instructs `--body "<content>"` pattern with inline interpolation
- `github-scoper.md` — instructs `gh issue create --title "<title>" --body "<body>"` inline

**GitLab personas**:
- `gitlab-enhancer.md` — `glab issue update <N> --title "new title"` inline
- `gitlab-commenter.md` — `--message "<content>"` and `--title` inline
- `gitlab-scoper.md` — `glab issue create --title "<title>" --description "<body>"` inline

**Gitea personas**:
- `gitea-enhancer.md` — `tea issues edit <N> --title "new title"` inline
- `gitea-commenter.md` — `tea issues comment <number> "<content>"` inline
- `gitea-scoper.md` — `tea issues create --title "<title>" --body "<body>"` inline

### Affected Pipeline Prompts

- `speckit-flow/create-pr.md` — uses inline `--body` (but with Wave-generated content, lower risk)
- `gh-implement/create-pr.md` — **already uses heredoc** (safe pattern)
- `gl-implement/create-mr.md` — **already uses heredoc** (safe pattern)
- `gt-implement/create-pr.md` — **already uses heredoc** (safe pattern)

### Existing Security Infrastructure

- `internal/security/sanitize.go` — has `containsShellMetachars()` detector and risk scoring
- `internal/github/client.go` — uses Go HTTP API with JSON payloads (already safe)
- **Missing**: No `ShellEscape()` utility for persona authors to reference
- **Missing**: No documentation on secure CLI patterns for persona prompt authors
