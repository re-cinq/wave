# Multi-Platform Pipeline Support

Wave supports pipeline orchestration across multiple Git platforms. Each platform
has its own set of pipelines, personas, and prompts that mirror the GitHub (`gh-*`)
equivalents.

## Supported Platforms

| Platform | CLI Tool | Pipeline Prefix | Version Support |
|----------|----------|-----------------|-----------------|
| GitHub   | `gh`     | `gh-*`          | Any             |
| GitLab   | `glab`   | `gl-*`          | v14.0+          |
| Gitea    | `tea`    | `gt-*`          | v1.20+          |

## Available Pipelines

### GitLab Pipelines

| Pipeline        | Description                                           |
|-----------------|-------------------------------------------------------|
| `gl-implement`  | Implement a GitLab issue end-to-end: fetch, assess, plan, implement, create MR |
| `gl-research`   | Research a GitLab issue and post findings as a comment |
| `gl-refresh`    | Refresh a stale GitLab issue against recent codebase changes |
| `gl-rewrite`    | Analyze and rewrite poorly documented GitLab issues   |

### Gitea Pipelines

| Pipeline        | Description                                           |
|-----------------|-------------------------------------------------------|
| `gt-implement`  | Implement a Gitea issue end-to-end: fetch, assess, plan, implement, create PR |
| `gt-research`   | Research a Gitea issue and post findings as a comment  |
| `gt-refresh`    | Refresh a stale Gitea issue against recent codebase changes |
| `gt-rewrite`    | Analyze and rewrite poorly documented Gitea issues    |

## CLI Tool Installation

### glab (GitLab CLI)

Install the official GitLab CLI:

```bash
# macOS
brew install glab

# Linux (Homebrew)
brew install glab

# Linux (apt)
apt install glab

# From source
go install gitlab.com/gitlab-org/cli/cmd/glab@latest
```

Verify installation:
```bash
glab --version
```

### tea (Gitea CLI)

Install the Gitea CLI:

```bash
# macOS
brew tap gitea/tap https://gitea.com/gitea/homebrew-gitea
brew install tea

# Linux (binary)
# Download from https://gitea.com/gitea/tea/releases

# From source
go install code.gitea.io/tea@latest
```

Verify installation:
```bash
tea --version
```

## Authentication

### GitLab SaaS (gitlab.com)

Set the `GITLAB_TOKEN` environment variable with a personal access token:

```bash
export GITLAB_TOKEN="glpat-xxxxxxxxxxxxxxxxxxxx"
```

Or authenticate interactively:
```bash
glab auth login
```

Required token scopes:
- `api` — full API access
- `read_repository` — read repository content
- `write_repository` — push branches and create merge requests

### GitLab Self-Hosted

For self-hosted GitLab instances, set the hostname:

```bash
export GITLAB_HOST="gitlab.example.com"
export GITLAB_TOKEN="glpat-xxxxxxxxxxxxxxxxxxxx"
```

Or authenticate interactively:
```bash
glab auth login --hostname gitlab.example.com
```

### Gitea

Gitea requires explicit login configuration since there is no default SaaS host:

```bash
tea login add \
  --name my-gitea \
  --url https://gitea.example.com \
  --token "your-api-token"
```

Generate a Gitea API token from: `https://<your-gitea-instance>/user/settings/applications`

Required token permissions:
- Repository read/write
- Issue read/write
- Pull request read/write

## Usage

### GitLab

```bash
# Implement an issue
wave run gl-implement "owner/repo 42"

# Research an issue
wave run gl-research "owner/repo 42"

# Refresh a stale issue
wave run gl-refresh "owner/repo 42 -- acceptance criteria are outdated"

# Rewrite a poorly documented issue
wave run gl-rewrite "owner/repo 42"
```

### Gitea

```bash
# Implement an issue
wave run gt-implement "owner/repo 42"

# Research an issue
wave run gt-research "owner/repo 42"

# Refresh a stale issue
wave run gt-refresh "owner/repo 42 -- needs updating after refactor"

# Rewrite a poorly documented issue
wave run gt-rewrite "owner/repo 42"
```

## Platform-Specific Differences

### Terminology

| Concept       | GitHub          | GitLab            | Gitea           |
|---------------|-----------------|-------------------|-----------------|
| Code review   | Pull Request    | Merge Request     | Pull Request    |
| PR/MR create  | `gh pr create`  | `glab mr create`  | `tea pulls create` |
| Issue view    | `gh issue view` | `glab issue view` | `tea issues view` |
| Issue comment  | `gh issue comment` | `glab issue note` | `tea issues comment` |
| Issue edit    | `gh issue edit` | `glab issue update` | `tea issues edit` |

### Contract Schema Reuse

All platform variants share the same contract schemas. The schemas validate
JSON structure (field names, types, required properties) — not platform-specific
content. This means:

- `issue-assessment.schema.json` — used by all `*-implement` pipelines
- `issue-impl-plan.schema.json` — used by all `*-implement` pipelines
- `pr-result.schema.json` — used by all `*-implement` pipelines (MR results also use this schema)
- `issue-content.schema.json` — used by all `*-research` pipelines

### Persona Permissions

Each platform's personas are scoped to their respective CLI tool:

- **GitLab personas** — `Bash(glab *)` allowed; `Bash(gh *)` and `Bash(tea *)` denied
- **Gitea personas** — `Bash(tea *)` allowed; `Bash(gh *)` and `Bash(glab *)` denied
- **GitHub personas** — `Bash(gh *)` allowed (existing behavior)

This prevents cross-platform command contamination during pipeline execution.

## Troubleshooting

### "glab: command not found"
Install the GitLab CLI. See [CLI Tool Installation](#cli-tool-installation).

### "tea: command not found"
Install the Gitea CLI. See [CLI Tool Installation](#cli-tool-installation).

### "401 Unauthorized" from GitLab
Check that `GITLAB_TOKEN` is set and has the required scopes. For self-hosted
instances, verify `GITLAB_HOST` is correctly configured.

### "Could not determine Gitea login"
Run `tea login add` to configure your Gitea instance. See [Authentication](#authentication).

### Pipeline runs but uses wrong CLI commands
Verify you're using the correct pipeline prefix: `gl-*` for GitLab, `gt-*` for Gitea.
Each platform's prompts contain platform-specific CLI commands.
