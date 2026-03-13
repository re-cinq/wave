# Implementation Plan: Harden gh CLI against shell injection

## Objective

Prevent shell injection attacks in Wave persona prompts and pipeline step templates that construct `gh` / `tea` / `glab` / `curl` CLI commands with user-supplied issue content (titles, bodies, labels). The primary attack surface is the AI agent constructing shell commands via the Bash tool where untrusted content from GitHub issues is interpolated inline.

## Approach

The Go adapter layer already uses `exec.Command` (bypasses shell), so the risk is in the **persona prompts and pipeline prompt templates** that instruct AI agents to build shell commands. The fix is a two-pronged defense-in-depth strategy:

1. **Persona prompt hardening** — Replace all inline `--body "<content>"` and `--title "<title>"` examples with safe patterns: `--body-file` with temp files, or quoted heredocs (`<<'EOF'`). The single-quoted heredoc delimiter prevents shell expansion.

2. **Security documentation** — Add a `docs/guides/secure-cli-patterns.md` guide for persona authors, documenting the safe patterns and explaining why inline interpolation is dangerous.

The existing `internal/security/sanitize.go` already has `containsShellMetachars()` for defense-in-depth detection. No changes needed at the Go level — the fix is entirely in the prompt/documentation layer.

## File Mapping

### Persona files to modify (`.wave/personas/` AND `internal/defaults/personas/`)

| File | Action | Change |
|------|--------|--------|
| `github-commenter.md` | modify | Replace inline `--body "<content>"` examples with `--body-file` pattern |
| `github-enhancer.md` | modify | Replace inline `--title "new title"` with safe quoting pattern |
| `github-scoper.md` | modify | Replace inline `--body "<body>"` in `gh issue create` with `--body-file` |
| `gitea-commenter.md` | modify | Replace inline string interpolation examples |
| `gitea-enhancer.md` | modify | Already mentions `--body-file` but title pattern needs hardening |
| `bitbucket-enhancer.md` | modify | Already uses temp file pattern — verify and reinforce |

### Pipeline prompt templates to modify (`.wave/prompts/` AND `internal/defaults/prompts/`)

| File | Action | Change |
|------|--------|--------|
| `gh-implement/create-pr.md` | modify | Already uses heredoc — verify single-quoted delimiter |
| `speckit-flow/create-pr.md` | modify | Inline `--body` and `--title` — switch to `--body-file` |
| `gh-implement-epic/report.md` | modify | Already uses quoted heredoc — verify |

### Pipeline YAML files with inline examples (`.wave/pipelines/` AND `internal/defaults/pipelines/`)

| File | Action | Change |
|------|--------|--------|
| `gh-scope.yaml` | modify | Replace `--body "<body>"` in `gh issue create` example |
| `gh-rewrite.yaml` | modify | Replace `--body "body_template"` and `--title` examples |
| `gh-refresh.yaml` | modify | Already uses `--body-file` — verify `--title` is safe |
| `gh-pr-review.yaml` | modify | Replace inline `--body` in `gh pr comment` |
| `dead-code-review.yaml` | modify | Replace inline `--body` in `gh pr comment` |
| `wave-audit.yaml` | modify | Replace inline `--body` and `--title` |
| `recinq.yaml` | modify | Replace inline `--body` in comment commands |
| `feature.yaml` | modify | Uses `$(git log ...)` in title — acceptable (controlled input) |

### Documentation

| File | Action | Change |
|------|--------|--------|
| `docs/guides/secure-cli-patterns.md` | create | New guide for persona authors |
| `docs/guides/github-integration.md` | modify | Add security section referencing secure patterns guide |
| `docs/guide/personas.md` | modify | Add security note about CLI command construction |

### Tests

| File | Action | Change |
|------|--------|--------|
| `internal/security/sanitize_test.go` | modify | Add injection scenario tests for shell metachar patterns found in issue content |

## Architecture Decisions

1. **Heredocs over `gh api`**: The issue proposed using `gh api` with JSON payloads. While more secure, this would be a significant deviation from the current `gh issue edit` / `gh pr create` patterns used across all personas and pipelines. Instead, we use `--body-file` with temp files (already partially adopted) and single-quoted heredocs (`<<'EOF'` prevents shell expansion). This is a minimal-disruption fix.

2. **Both .wave/ and internal/defaults/ must be updated**: Wave has two copies of persona/pipeline files — the live `.wave/` directory and the `internal/defaults/` embedded defaults. Both must stay in sync.

3. **No Go-level changes needed**: The adapter uses `exec.Command` which bypasses the shell entirely. The risk is specifically in the AI agent constructing shell command strings via the Bash tool. The fix is in the prompts that guide agent behavior.

4. **Title handling**: Titles are shorter and less likely to contain injection, but still need protection. For `--title`, use single quotes with proper escaping, or write to a temp file when the content comes from untrusted sources.

## Risks

| Risk | Mitigation |
|------|------------|
| Persona behavior is non-deterministic — agents may still construct unsafe commands | Add explicit "NEVER use inline --body with untrusted content" constraints to persona prompts |
| Two-copy sync (`.wave/` and `internal/defaults/`) could drift | Verify both copies are updated in each task |
| Pipeline YAML changes might break heredoc parsing in YAML multiline strings | Test YAML parsing after changes |
| Some forge CLIs (tea, glab) may not support `--body-file` | Verify CLI support; fall back to heredoc patterns where needed |

## Testing Strategy

1. **Existing tests**: `internal/security/sanitize_test.go` already tests `containsShellMetachars()` and risk scoring — extend with injection scenario tests
2. **Manual verification**: Review each modified persona/pipeline prompt for unsafe patterns
3. **YAML validation**: Run `go test ./...` to ensure embedded defaults still parse correctly
4. **Integration check**: `wave validate --verbose` to verify manifest integrity after changes
