# Implementation Plan: Multi-Platform Pipeline Support

## Objective

Create GitLab (`gl-*`) and Gitea (`gt-*`) pipeline variants that mirror the existing `gh-*` pipelines, providing equivalent issue management, research, refresh, and rewrite workflows for each platform.

## Approach

### Strategy: Clone-and-Adapt

The most reliable approach is to clone each `gh-*` pipeline and systematically adapt it for the target platform's CLI tool and terminology. This preserves the proven pipeline structure while only changing platform-specific interactions.

**Phase 1 — GitLab pipelines** (higher priority, larger user base)
**Phase 2 — Gitea pipelines** (follows same pattern established in Phase 1)

### Layered Changes

Each platform variant requires changes at four layers:

1. **Pipeline YAML** (`.wave/pipelines/gl-*.yaml`, `.wave/pipelines/gt-*.yaml`) — Step definitions with platform-specific prompts
2. **Personas** (`.wave/personas/gitlab-*.md`, `.wave/personas/gitea-*.md`) — CLI tool instructions
3. **Prompts** (`.wave/prompts/gl-implement/*.md`, `.wave/prompts/gt-implement/*.md`) — Platform-adapted step prompts
4. **Manifest** (`wave.yaml`) — Persona definitions and pipeline registration

### Contract Reuse

Existing contract schemas (e.g., `issue-assessment.schema.json`, `issue-impl-plan.schema.json`, `pr-result.schema.json`) are **platform-agnostic** — they validate JSON structure, not platform-specific content. These can be reused directly. Only the `pr-result` schema may need a `mr-result` alias for GitLab semantics, but the schema itself is identical.

## File Mapping

### New Files — GitLab (Phase 1)

| Path | Action | Description |
|------|--------|-------------|
| `.wave/pipelines/gl-implement.yaml` | create | GitLab implementation pipeline |
| `.wave/pipelines/gl-research.yaml` | create | GitLab research pipeline |
| `.wave/pipelines/gl-refresh.yaml` | create | GitLab issue refresh pipeline |
| `.wave/pipelines/gl-rewrite.yaml` | create | GitLab issue rewrite pipeline |
| `.wave/personas/gitlab-analyst.md` | create | GitLab issue analyst persona |
| `.wave/personas/gitlab-enhancer.md` | create | GitLab issue enhancer persona |
| `.wave/personas/gitlab-commenter.md` | create | GitLab comment poster persona |
| `.wave/prompts/gl-implement/fetch-assess.md` | create | GitLab fetch-assess prompt |
| `.wave/prompts/gl-implement/plan.md` | create | GitLab plan prompt |
| `.wave/prompts/gl-implement/implement.md` | create | GitLab implement prompt |
| `.wave/prompts/gl-implement/create-mr.md` | create | GitLab MR creation prompt |

### New Files — Gitea (Phase 2)

| Path | Action | Description |
|------|--------|-------------|
| `.wave/pipelines/gt-implement.yaml` | create | Gitea implementation pipeline |
| `.wave/pipelines/gt-research.yaml` | create | Gitea research pipeline |
| `.wave/pipelines/gt-refresh.yaml` | create | Gitea issue refresh pipeline |
| `.wave/pipelines/gt-rewrite.yaml` | create | Gitea issue rewrite pipeline |
| `.wave/personas/gitea-analyst.md` | create | Gitea issue analyst persona |
| `.wave/personas/gitea-enhancer.md` | create | Gitea issue enhancer persona |
| `.wave/personas/gitea-commenter.md` | create | Gitea comment poster persona |
| `.wave/prompts/gt-implement/fetch-assess.md` | create | Gitea fetch-assess prompt |
| `.wave/prompts/gt-implement/plan.md` | create | Gitea plan prompt |
| `.wave/prompts/gt-implement/implement.md` | create | Gitea implement prompt |
| `.wave/prompts/gt-implement/create-pr.md` | create | Gitea PR creation prompt |

### Modified Files

| Path | Action | Description |
|------|--------|-------------|
| `wave.yaml` | modify | Add GitLab and Gitea persona definitions |

### Documentation Files

| Path | Action | Description |
|------|--------|-------------|
| `docs/multi-platform.md` | create | Platform configuration guide |

## Architecture Decisions

### AD-1: Pipeline-level adaptation, not Go-level

Platform differences are handled entirely in pipeline YAML, prompts, and personas. The `internal/github/` Go package and `GitHubAdapter` are NOT extended. Rationale: the `gh-*` pipelines already use subprocess execution (Claude Code runs `gh` CLI commands), so the same pattern works for `glab` and `tea` commands without Go code changes.

### AD-2: Shared contract schemas

Contract schemas are platform-neutral JSON structures. Rather than creating `gitlab-issue-assessment.schema.json`, reuse `issue-assessment.schema.json`. The persona and prompt instruct the agent what data to fill in; the schema validates structure only.

### AD-3: Platform prefix convention

- `gh-*` → GitHub (`gh` CLI)
- `gl-*` → GitLab (`glab` CLI)
- `gt-*` → Gitea (`tea` CLI)

### AD-4: Persona naming convention

- `github-analyst` → `gitlab-analyst`, `gitea-analyst`
- `github-enhancer` → `gitlab-enhancer`, `gitea-enhancer`
- `github-commenter` → `gitlab-commenter`, `gitea-commenter`

### AD-5: CLI tool permission scoping

GitLab personas get `Bash(glab *)` permissions; Gitea personas get `Bash(tea *)` permissions. Each platform's personas deny the other platforms' CLI tools to prevent cross-contamination.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| `glab` CLI syntax differs significantly from `gh` | High | Test each command mapping; document differences |
| `tea` CLI has missing features vs `gh` | Medium | Identify gaps early; use HTTP API fallback where needed |
| Self-hosted GitLab API differences | Medium | Test against GitLab v14.0+ minimum; document `GITLAB_HOST` setup |
| Manifest parser rejects new persona/pipeline format | Low | Existing parser is generic; add tests to confirm |
| Prompt injection via platform-specific issue content | Medium | Same security model as `gh-*`; credential scrubbing applies |

## Testing Strategy

### Unit Tests
- Validate all new pipeline YAML files parse correctly via `manifest.Parse()`
- Validate persona definitions in `wave.yaml` are well-formed
- Verify contract schemas validate expected output structures

### Integration Tests
- End-to-end pipeline execution tests for each `gl-*` and `gt-*` pipeline (mocked CLI output)
- Verify prompt templates render correctly with platform-specific variables
- Verify permission enforcement (e.g., `gitlab-commenter` cannot edit issues)

### Manual Validation
- Run `gl-implement` against a real GitLab repository (with `glab` installed)
- Run `gt-implement` against a real Gitea instance (with `tea` installed)
- Verify pipeline outcomes match `gh-*` equivalents
