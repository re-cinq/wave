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
