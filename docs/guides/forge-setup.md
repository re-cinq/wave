# Forge Setup Guide

Wave auto-detects your git forge (GitHub, GitLab, Gitea, Forgejo, Codeberg, Bitbucket) from the repository's remote URL and configures pipelines, personas, and CLI tools accordingly.

## Quick Start

```bash
# 1. Set your forge token as an environment variable
export GH_TOKEN="ghp_..."          # GitHub
# or
export GITLAB_TOKEN="glpat-..."    # GitLab
# or
export GITEA_TOKEN="abc123..."     # Gitea / Forgejo
# or
export CODEBERG_TOKEN="abc123..."  # Codeberg
# or
export BITBUCKET_TOKEN="ATATT3x.." # Bitbucket (App Password)

# 2. Initialize Wave in your project
cd your-project
wave init

# 3. Wave auto-detects the forge and configures everything
wave run impl-issue -- "https://github.com/org/repo/issues/42"
```

## Supported Forges

| Forge | Detection | CLI Tool | Token Env Var | PR Term |
|-------|-----------|----------|---------------|---------|
| **GitHub** | `github.com` hostname | `gh` | `GH_TOKEN` or `GITHUB_TOKEN` | Pull Request |
| **GitLab** | `gitlab.com` hostname | `glab` | `GITLAB_TOKEN` or `GL_TOKEN` | Merge Request |
| **Gitea** | API probing (`/api/v1/version`) | `tea` | `GITEA_TOKEN` | Pull Request |
| **Forgejo** | API probing (`/api/forgejo/v1/version`) | `tea` | `GITEA_TOKEN` | Pull Request |
| **Codeberg** | `codeberg.org` hostname | `tea` | `CODEBERG_TOKEN` (falls back to `GITEA_TOKEN`) | Pull Request |
| **Bitbucket** | `bitbucket.org` hostname | `bb` | `BITBUCKET_TOKEN` | Pull Request |
| **Local** | No git remote / `forge: local` in manifest | — | — | — |

## Token Setup Per Forge

### GitHub

```bash
# Option 1: Personal Access Token (recommended)
export GH_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

# Option 2: GitHub CLI auth (auto-detected)
gh auth login

# Option 3: Environment variable
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

**Required scopes**: `repo`, `read:org` (for private repos)

**Create token**: https://github.com/settings/tokens/new

### GitLab

```bash
export GITLAB_TOKEN="glpat-xxxxxxxxxxxxxxxxxxxx"
```

**Required scopes**: `api` (full API access) or `read_api` + `write_repository`

**Create token**: https://gitlab.com/-/user_settings/personal_access_tokens

### Gitea / Forgejo (Self-Hosted)

```bash
export GITEA_TOKEN="your-token-here"
```

**Required scopes**: `write:repository`, `write:issue`, `read:user`, `read:organization`

**Create token**: `https://your-instance.com/user/settings/applications` → Generate New Token

> **Note**: Wave detects self-hosted Gitea/Forgejo instances by probing API endpoints (`/api/v1/version` for Gitea, `/api/forgejo/v1/version` for Forgejo). No hostname pattern matching needed.

### Codeberg

```bash
export CODEBERG_TOKEN="your-token-here"
```

Codeberg is a Forgejo instance. Wave detects it by hostname (`codeberg.org`) and uses the same CLI (`tea`) and pipeline prefix (`gt-`) as Gitea/Forgejo.

**Required scopes**: `write:repository`, `write:issue`, `read:user`

**Create token**: https://codeberg.org/user/settings/applications → Generate New Token

> **Fallback**: If `CODEBERG_TOKEN` is not set, Wave falls back to `GITEA_TOKEN`.

### Bitbucket

```bash
export BITBUCKET_TOKEN="ATATT3x..."
```

Bitbucket uses **App Passwords** (not OAuth tokens). The token format starts with `ATATT3x`.

**Required permissions**: Repositories (Read, Write), Pull requests (Read, Write), Issues (Read, Write)

**Create token**: https://bitbucket.org/account/settings/app-passwords/ → Create app password

> **Auth method**: Bitbucket API uses HTTP Basic Auth with `username:app-password`. Wave resolves the username from the git remote URL (e.g., `re-cinq-admin` from `https://re-cinq-admin@bitbucket.org/re-cinq/repo.git`).

### Local (No Forge)

For repositories with no remote or when you want to disable forge features:

```yaml
# wave.yaml
metadata:
  forge: local    # or "none"
```

Local mode:
- Only forge-independent pipelines are available (no `gh-`, `gl-`, `gt-`, `bb-` prefixed pipelines)
- Forge-dependent steps are skipped at preflight with a clear message
- Template variables <code v-pre>{{ forge.cli_tool }}</code> resolve to empty string
- No PR creation, issue management, or forge API calls

## How Detection Works

1. **Hostname matching**: GitHub, GitLab, Codeberg, Bitbucket are detected by their known hostnames
2. **API endpoint probing**: Self-hosted instances are detected by probing well-known API endpoints (3-second timeout):
   - Forgejo: `/api/forgejo/v1/version` (checked first — Forgejo also serves Gitea's API)
   - Gitea: `/api/v1/version`
   - GitLab: `/api/v4/version`
   - Bitbucket Server: `/rest/api/1.0/application-properties`
3. **Remote preference**: When multiple remotes exist, `origin` is preferred over others
4. **Manifest override**: `metadata.forge` in `wave.yaml` overrides all detection

## Manifest Override

Force a specific forge type regardless of git remote:

```yaml
# wave.yaml
metadata:
  name: my-project
  forge: gitea       # github, gitlab, gitea, forgejo, codeberg, bitbucket, local, none
```

## Using .env Files

Store tokens in a `.env` file (make sure it's in `.gitignore`):

```bash
# .env
GH_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
GITEA_TOKEN=abc123
CODEBERG_TOKEN=def456
BITBUCKET_TOKEN=ATATT3x...
```

Load before running Wave:

```bash
source .env && export GH_TOKEN GITEA_TOKEN CODEBERG_TOKEN BITBUCKET_TOKEN
wave run impl-issue -- "https://github.com/org/repo/issues/42"
```

## Verifying Your Setup

### Check forge detection

```bash
wave doctor          # Shows detected forge type and status
wave run wave-test-forge -- "validate"  # Full smoke test
```

### Check credentials (WebUI)

```bash
wave serve --port 8080
# Open http://localhost:8080/admin → Credential Status section
```

### Check from CLI

```bash
# GitHub
gh auth status

# GitLab
glab auth status

# Gitea / Codeberg
tea login list

# Bitbucket
curl -u "username:$BITBUCKET_TOKEN" https://api.bitbucket.org/2.0/user
```

## Troubleshooting

### "persona unknown-commenter not found"

Wave generates forge-specific personas during `wave init` (e.g., `github-commenter`, `gitea-commenter`). If the wrong forge was detected:

```bash
wave init --merge    # Re-detects forge and updates personas
```

### "forge.cli_tool is empty"

The forge was detected as `local` or `unknown`. Check:
- Is there a git remote? (`git remote -v`)
- Is the remote URL correct?
- For self-hosted: is the instance reachable? (Wave probes API endpoints)

### Self-hosted instance not detected

If hostname matching and API probing both fail, use the manifest override:

```yaml
metadata:
  forge: gitea   # Force Gitea detection
```

### Pipeline created PR on wrong forge

When multiple remotes exist (e.g., `origin` → Gitea, `github` → GitHub mirror), Wave prefers `origin`. To use a different remote:

```yaml
metadata:
  forge: github  # Override to use GitHub even if origin points elsewhere
```
