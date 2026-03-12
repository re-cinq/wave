# security: Harden gh CLI commands against shell injection

**Issue**: [#28](https://github.com/re-cinq/wave/issues/28)
**Labels**: bug
**Author**: nextlevelshit
**Priority**: Medium

## Context

From Copilot review on PR #26: github-enhancer persona constructs shell commands with potentially untrusted issue content.

## Risk

If issue titles/bodies contain shell metacharacters like `$()` or backticks, they could be executed.

## Proposed Mitigations

1. Use `gh api` with JSON payloads instead of `gh issue edit` with shell arguments
2. Add heredoc patterns to persona prompts for safe body handling
3. Consider Wave security module integration for shell escaping

## Acceptance Criteria

- [ ] All gh CLI commands in personas use safe argument passing (JSON payloads or heredocs)
- [ ] Shell metacharacters in issue content cannot trigger command execution
- [ ] Security tests added to verify injection resistance
- [ ] Documentation updated with secure CLI patterns for persona authors

## References

- OWASP Command Injection: https://owasp.org/www-community/attacks/Command_Injection
- PR #26 review comments

## Attack Surface Analysis

### Vulnerable Persona Files

The following persona files instruct AI agents to construct shell commands with inline-interpolated arguments, making them vulnerable if issue titles/bodies contain shell metacharacters (`$()`, backticks, `|`, `&`, `;`, etc.):

1. **github-enhancer** (`.wave/personas/` + `internal/defaults/personas/`):
   - `gh issue edit <N> --title "new title"` — title inline in shell

2. **github-commenter** (`.wave/personas/` + `internal/defaults/personas/`):
   - `gh issue comment --body "<content>"` — body inline
   - `gh pr comment --body "<content>"` — body inline
   - `gh pr review --body "<content>"` — body inline
   - `gh pr create --title "<title>" --body "<description>"` — both inline

3. **github-scoper** (`.wave/personas/` + `internal/defaults/personas/`):
   - `gh issue create --title "<title>" --body "<body>" --label "<labels>"` — all inline

4. **gitea-enhancer** (`.wave/personas/` + `internal/defaults/personas/`):
   - `tea issues edit <N> --title "new title"` — title inline

5. **gitea-commenter** (`.wave/personas/` + `internal/defaults/personas/`):
   - `tea issues comment <number> "<content>"` — content inline
   - `tea pulls create --title "<title>" --description "<description>"` — inline

6. **gitlab-enhancer** (`.wave/personas/` + `internal/defaults/personas/`):
   - `glab issue update <N> --title "new title"` — title inline

7. **gitlab-commenter** (`.wave/personas/` + `internal/defaults/personas/`):
   - `glab issue note --message "<content>"` — inline
   - `glab mr note --message "<content>"` — inline
   - `glab mr create --title "<title>" --description "<description>"` — inline

### Already Safe

- **bitbucket-enhancer** / **bitbucket-commenter**: Use JSON payloads written to temp files + `curl -d @file` — no shell interpolation
- **gh-implement/create-pr.md prompt**: Uses quoted heredoc (`<<'EOF'`) for body — prevents shell expansion
- **internal/github/client.go**: Uses `net/http` with JSON bodies — no shell involvement

### Security Module (existing)

- `internal/security/sanitize.go` already has `containsShellMetachars()` detection and risk scoring
- Currently detect-only — no shell escaping utility is exposed for persona use
