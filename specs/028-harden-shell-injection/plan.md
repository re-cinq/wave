# Implementation Plan: Harden Shell Injection

## 1. Objective

Prevent shell injection attacks in all forge CLI persona prompts by replacing inline string interpolation patterns with safe argument-passing techniques (heredocs, `--body-file`, `gh api` JSON payloads). Add a `ShellEscape()` utility to the security module and document secure CLI patterns for persona authors.

## 2. Approach

The attack surface is in **persona system prompts** (`.md` files), which instruct the AI model to construct shell commands. When issue titles/bodies contain shell metacharacters like `$(cmd)`, `` `cmd` ``, `"; rm -rf /`, the model may interpolate them directly into shell commands.

**Strategy — defense in depth with three layers:**

1. **Prompt-level hardening** (primary): Update all persona prompts to use safe patterns:
   - `--body-file <tmpfile>` for long content (write to file first, pass file path)
   - Heredoc (`<<'EOF'`) with **single-quoted delimiter** to prevent shell expansion
   - `gh api` with JSON payloads instead of `gh issue edit --body` where appropriate

2. **Security module utility** (secondary): Add `ShellEscape()` to `internal/security/sanitize.go` for defense-in-depth. This function can be referenced in documentation and potentially used by future programmatic shell command construction.

3. **Documentation** (tertiary): Add a "Secure CLI Patterns" section to `docs/guides/github-integration.md` and `docs/guide/personas.md` so persona authors know the safe patterns.

## 3. File Mapping

### Modify — Persona Prompts (internal/defaults + .wave mirrors)

Each persona `.md` file exists in two locations that must stay in sync:
- `internal/defaults/personas/<name>.md` (embedded in binary)
- `.wave/personas/<name>.md` (runtime copy)

| File | Change |
|------|--------|
| `github-enhancer.md` | Replace `--title "..."` with heredoc or `--body-file` pattern |
| `github-commenter.md` | Replace inline `--body "<content>"` with heredoc/`--body-file` |
| `github-scoper.md` | Replace inline `--title`/`--body` with heredoc/`--body-file` |
| `gitlab-enhancer.md` | Replace `--title "..."` with safe pattern |
| `gitlab-commenter.md` | Replace `--message "<content>"` with heredoc/file |
| `gitlab-scoper.md` | Replace inline `--title`/`--description` with safe pattern |
| `gitea-enhancer.md` | Replace `--title "..."` with safe pattern |
| `gitea-commenter.md` | Replace inline `"<content>"` with safe pattern |
| `gitea-scoper.md` | Replace inline `--title`/`--body` with safe pattern |

### Modify — Pipeline Prompts

| File | Change |
|------|--------|
| `internal/defaults/prompts/speckit-flow/create-pr.md` | Use heredoc pattern for `--body` |
| `.wave/prompts/speckit-flow/create-pr.md` | Mirror the above |

### Modify — Security Module

| File | Change |
|------|--------|
| `internal/security/sanitize.go` | Add `ShellEscape()` function |
| `internal/security/sanitize_test.go` | Add tests for `ShellEscape()` and injection vectors |

### Modify — Documentation

| File | Change |
|------|--------|
| `docs/guides/github-integration.md` | Add "Secure CLI Patterns" section |
| `docs/guide/personas.md` | Add shell injection warning and safe patterns |

## 4. Architecture Decisions

### AD-1: Heredoc with single-quoted delimiter over `gh api`

**Decision**: Use `<<'EOF'` heredocs as the primary safe pattern rather than switching all commands to `gh api` JSON payloads.

**Rationale**:
- Heredocs with single-quoted delimiters (`<<'EOF'` not `<<EOF`) prevent all shell expansion
- The `gh` / `glab` / `tea` CLIs all support `--body-file` which is even safer
- Switching to `gh api` would require restructuring all persona prompts significantly and would break the pattern of using CLI tools that personas are designed around
- Heredocs are already used successfully in `gh-implement/create-pr.md`

### AD-2: Write-to-file pattern for body content

**Decision**: For long content (issue bodies, PR descriptions), instruct personas to write content to a temp file first, then use `--body-file`.

**Rationale**:
- Completely eliminates shell interpolation risk for body content
- `gh`, `glab`, and `tea` all support file-based body input
- Already partially in use (github-enhancer.md mentions `--body-file` in constraints)

### AD-3: ShellEscape as defense-in-depth, not primary defense

**Decision**: Add `ShellEscape()` to the security module but do NOT rely on it in persona prompts.

**Rationale**:
- Persona prompts are natural language instructions to AI models; they cannot call Go functions
- The utility serves as documentation and as a building block for any future programmatic command construction in Go code
- Primary defense is the prompt pattern change

## 5. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| AI model ignores safe patterns in prompts | Medium | High | Add explicit warnings about injection in persona constraints section |
| Heredoc delimiter collision with content | Low | Medium | Use unique delimiters like `'WAVEBODY'` instead of `'EOF'` |
| Changes break existing pipeline behavior | Low | High | Test all affected pipelines after changes; run `go test ./...` |
| Forgetting to sync `internal/defaults/` and `.wave/` copies | Medium | Medium | Update both in same commit; add a note in CLAUDE.md |

## 6. Testing Strategy

### Unit Tests (in `internal/security/sanitize_test.go`)
- `TestShellEscape_Metacharacters` — verify all POSIX metacharacters are properly escaped
- `TestShellEscape_CommandSubstitution` — verify `$()` and backticks are neutralized
- `TestShellEscape_EmptyAndNormal` — verify clean strings pass through unchanged
- `TestShellEscape_RealWorldIssueTitle` — verify realistic malicious issue titles are escaped

### Integration Validation
- `go test ./...` — ensure no regressions across the entire codebase
- `go test -race ./...` — race detector pass
- Manual review: grep all persona `.md` files for remaining unsafe `--body "` or `--title "` inline patterns
