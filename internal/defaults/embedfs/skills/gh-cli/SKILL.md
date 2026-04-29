---
name: gh-cli
description: GitHub CLI (gh) comprehensive reference for repositories, issues, pull requests, Actions, projects, releases, gists, codespaces, organizations, extensions, and all GitHub operations from the command line.
check_command: gh --version
---

# GitHub CLI (gh)

Work seamlessly with GitHub from the command line. For the complete command reference see [`references/full-reference.md`](references/full-reference.md).

## Authentication

```bash
# Interactive login
gh auth login

# Login with token
gh auth login --with-token < token.txt

# Check status
gh auth status

# Switch accounts
gh auth switch --hostname github.com --user username

# Setup git credential helper
gh auth setup-git

# Print active token
gh auth token
```

## Pull Requests

```bash
# Create PR (interactive)
gh pr create

# Create with title, body, base, draft, reviewer
gh pr create --title "Feature: X" --body "Details..." --base main --draft --reviewer alice

# List open PRs
gh pr list

# List with filters
gh pr list --author @me --labels bug --state all --limit 50

# View a PR
gh pr view 123
gh pr view 123 --comments --web

# Merge a PR
gh pr merge 123 --squash --delete-branch

# Check CI status
gh pr checks 123
gh pr checks 123 --watch
```

## Issues

```bash
# Create issue (interactive)
gh issue create

# Create with title and labels
gh issue create --title "Bug: X" --body "Steps..." --labels bug,high-priority --assignee @me

# List open issues
gh issue list

# List with filters
gh issue list --assignee @me --labels bug --state all --search "is:open label:bug"

# View an issue
gh issue view 123
gh issue view 123 --comments --web

# Close / reopen
gh issue close 123 --comment "Fixed in PR #456"
gh issue reopen 123
```

## Common Flags

| Flag | Description |
|---|---|
| `--repo OWNER/REPO` | Target a specific repository |
| `--json FIELDS` | Output JSON for given fields |
| `--jq EXPRESSION` | Filter JSON output with jq |
| `--web` | Open result in browser |
| `--paginate` | Follow pagination for large result sets |
| `--limit N` | Cap result count |

## API Requests

```bash
# REST GET
gh api /user --jq '.login'

# REST POST with fields
gh api --method POST /repos/owner/repo/issues \
  --field title="Issue title" --field body="Body"

# Paginate all results
gh api /user/repos --paginate

# GraphQL query
gh api graphql -f query='{ viewer { login } }'
```

## JSON Output Examples

```bash
# List PR numbers and titles
gh pr list --json number,title --jq '.[] | "\(.number): \(.title)"'

# Get issue labels
gh issue view 123 --json labels --jq '.labels[].name'

# Filter open PRs by author
gh pr list --json number,title,author \
  --jq '.[] | select(.author.login == "alice")'
```

## Complete Reference

For all commands including repos, releases, Actions, projects, gists, codespaces, search, secrets, extensions, and more, see:

**[`references/full-reference.md`](references/full-reference.md)**
