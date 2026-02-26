# Add GitLab and Gitea Platform Support Pipelines

**Issue**: [#168](https://github.com/re-cinq/wave/issues/168)
**Labels**: enhancement, pipeline, priority: high
**Author**: nextlevelshit
**State**: OPEN

## Overview

Create multi-platform pipeline variants for GitLab and Gitea to extend Wave's orchestration capabilities beyond GitHub.

## Context

The current `gh-*` pipelines are ready for release. This work establishes pipeline equivalents for other popular Git platforms to provide consistent Wave functionality across different infrastructure environments.

## Scope

### Platforms to Support

- **GitLab** (SaaS and self-hosted)
  - CLI tool: `glab` (official GitLab CLI)
  - Version support: v14.0+
- **Gitea**
  - CLI tool: `tea` CLI
  - Version support: v1.20+

### Existing gh-* Pipelines to Mirror

1. **gh-implement** — Implement a GitHub issue end-to-end: fetch, assess, plan, implement, create PR (4 steps)
2. **gh-research** — Research a GitHub issue and post findings as a comment (5 steps)
3. **gh-refresh** — Refresh a stale issue by comparing against recent codebase changes (3 steps)
4. **gh-rewrite** — Analyze and rewrite poorly documented issues (4 steps)

### Initial Deliverables

1. `gl-*` pipeline variants mirroring `gh-*` pipeline structure (gl-implement, gl-research, gl-refresh, gl-rewrite)
2. `gt-*` pipeline variants mirroring `gh-*` pipeline structure (gt-implement, gt-research, gt-refresh, gt-rewrite)
3. Platform-specific personas (gitlab-analyst, gitlab-enhancer, gitlab-commenter, gitea-analyst, gitea-enhancer, gitea-commenter)
4. Platform-specific contracts (reuse existing schemas where possible)
5. Platform-specific prompts adapted for each CLI tool's commands
6. Documentation for platform-specific configuration differences
7. Persona and pipeline entries in `wave.yaml`

## Acceptance Criteria

- [ ] All new pipelines pass test suite with equivalent functionality to `gh-*` pipelines
- [ ] Configuration examples provided for GitLab SaaS and self-hosted setups
- [ ] Configuration examples provided for Gitea v1.20+
- [ ] CLI tool dependencies documented in manifest
- [ ] Platform authentication patterns documented
- [ ] Pipeline YAML files validate against Wave's manifest parser
- [ ] Personas defined in `wave.yaml` with appropriate permissions
- [ ] Prompts adapted for `glab` and `tea` CLI syntax
- [ ] Contracts reused or extended as needed
- [ ] Existing `gh-*` pipelines remain unaffected (no regressions)

## Technical Requirements

- Maintain compatibility with Wave's ephemeral workspace architecture
- Enforce same security model (permission enforcement, credential scrubbing)
- Validate API compatibility for each platform version
- Follow the same pipeline step structure as `gh-*` variants
- Use platform-native CLI tools (`glab`, `tea`) in prompts instead of `gh`

## Platform-Specific Differences

### GitLab (`glab` CLI)
- Uses "merge requests" instead of "pull requests"
- Issue/MR commands: `glab issue view`, `glab mr create`, `glab issue comment`
- Auth: `GITLAB_TOKEN` or `glab auth login`
- Self-hosted: Requires `GITLAB_HOST` configuration
- API: REST v4 + GraphQL

### Gitea (`tea` CLI)
- Uses "pull requests" (same terminology as GitHub)
- Issue commands: `tea issues view`, `tea pulls create`, `tea issues comment`
- Auth: `tea login add` with token
- Self-hosted only (no SaaS default)
- API: REST API compatible with Swagger spec

## Risks & Questions

- Platform prioritization: Recommend GitLab first (larger user base), then Gitea
- Release strategy: Phased — GitLab pipelines first, Gitea second
- Feature parity gaps between platforms may need workarounds (e.g., GitLab MR approvals vs GitHub PR reviews)
- The `internal/github/` Go package is GitHub-specific; platform adapters at the Go level are out of scope for this issue (pipelines use CLI tools via subprocess, not the Go API client)
