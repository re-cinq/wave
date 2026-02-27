# Tasks

## Phase 1: GitLab Personas

- [X] Task 1.1: Create `.wave/personas/gitlab-analyst.md` — GitLab issue analyst persona adapted from `github-analyst.md` with `glab` CLI commands
- [X] Task 1.2: Create `.wave/personas/gitlab-enhancer.md` — GitLab issue enhancer persona adapted from `github-enhancer.md` with `glab` CLI commands
- [X] Task 1.3: Create `.wave/personas/gitlab-commenter.md` — GitLab comment poster persona adapted from `github-commenter.md` with `glab` CLI commands
- [X] Task 1.4: Add GitLab persona definitions to `wave.yaml` (gitlab-analyst, gitlab-enhancer, gitlab-commenter) with `Bash(glab *)` permissions

## Phase 2: GitLab Prompts

- [X] Task 2.1: Create `.wave/prompts/gl-implement/fetch-assess.md` — Adapt `gh-implement/fetch-assess.md` replacing `gh issue view` with `glab issue view` [P]
- [X] Task 2.2: Create `.wave/prompts/gl-implement/plan.md` — Adapt `gh-implement/plan.md` for GitLab context [P]
- [X] Task 2.3: Create `.wave/prompts/gl-implement/implement.md` — Adapt `gh-implement/implement.md` for GitLab context [P]
- [X] Task 2.4: Create `.wave/prompts/gl-implement/create-mr.md` — Adapt `gh-implement/create-pr.md` replacing `gh pr create` with `glab mr create` and using "merge request" terminology [P]

## Phase 3: GitLab Pipeline YAML

- [X] Task 3.1: Create `.wave/pipelines/gl-implement.yaml` — Mirror `gh-implement.yaml` using gitlab-* personas and gl-implement prompts
- [X] Task 3.2: Create `.wave/pipelines/gl-research.yaml` — Mirror `gh-research.yaml` using gitlab-* personas, replacing `gh issue` with `glab issue` commands in inline prompts
- [X] Task 3.3: Create `.wave/pipelines/gl-refresh.yaml` — Mirror `gh-refresh.yaml` using gitlab-* personas, replacing `gh` with `glab` in inline prompts [P]
- [X] Task 3.4: Create `.wave/pipelines/gl-rewrite.yaml` — Mirror `gh-rewrite.yaml` using gitlab-* personas, replacing `gh` with `glab` in inline prompts [P]

## Phase 4: Gitea Personas

- [X] Task 4.1: Create `.wave/personas/gitea-analyst.md` — Gitea issue analyst persona adapted from `github-analyst.md` with `tea` CLI commands
- [X] Task 4.2: Create `.wave/personas/gitea-enhancer.md` — Gitea issue enhancer persona adapted from `github-enhancer.md` with `tea` CLI commands
- [X] Task 4.3: Create `.wave/personas/gitea-commenter.md` — Gitea comment poster persona adapted from `github-commenter.md` with `tea` CLI commands
- [X] Task 4.4: Add Gitea persona definitions to `wave.yaml` (gitea-analyst, gitea-enhancer, gitea-commenter) with `Bash(tea *)` permissions

## Phase 5: Gitea Prompts

- [X] Task 5.1: Create `.wave/prompts/gt-implement/fetch-assess.md` — Adapt for `tea issues view` syntax [P]
- [X] Task 5.2: Create `.wave/prompts/gt-implement/plan.md` — Adapt for Gitea context [P]
- [X] Task 5.3: Create `.wave/prompts/gt-implement/implement.md` — Adapt for Gitea context [P]
- [X] Task 5.4: Create `.wave/prompts/gt-implement/create-pr.md` — Adapt for `tea pulls create` syntax [P]

## Phase 6: Gitea Pipeline YAML

- [X] Task 6.1: Create `.wave/pipelines/gt-implement.yaml` — Mirror `gh-implement.yaml` using gitea-* personas and gt-implement prompts
- [X] Task 6.2: Create `.wave/pipelines/gt-research.yaml` — Mirror `gh-research.yaml` using gitea-* personas, replacing `gh` with `tea` commands [P]
- [X] Task 6.3: Create `.wave/pipelines/gt-refresh.yaml` — Mirror `gh-refresh.yaml` using gitea-* personas, replacing `gh` with `tea` commands [P]
- [X] Task 6.4: Create `.wave/pipelines/gt-rewrite.yaml` — Mirror `gh-rewrite.yaml` using gitea-* personas, replacing `gh` with `tea` commands [P]

## Phase 7: Testing

- [X] Task 7.1: Run `go test ./...` to verify no regressions in existing pipeline parsing
- [X] Task 7.2: Write test cases for new GitLab pipeline YAML parsing
- [X] Task 7.3: Write test cases for new Gitea pipeline YAML parsing
- [X] Task 7.4: Verify persona permission enforcement for new personas

## Phase 8: Documentation & Polish

- [X] Task 8.1: Create `docs/multi-platform.md` with platform setup guides (GitLab SaaS, GitLab self-hosted, Gitea)
- [X] Task 8.2: Document authentication patterns for each platform (`GITLAB_TOKEN`, `GITLAB_HOST`, `tea login`)
- [X] Task 8.3: Document CLI tool installation instructions for `glab` and `tea`
- [X] Task 8.4: Final validation — run full test suite with `go test -race ./...`
