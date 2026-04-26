You are creating a {{ forge.pr_term }} for the implemented issue.

Input: {{ input }}

## Working Directory

You are running in an **isolated git worktree** shared with previous pipeline steps.
Your working directory IS the project root. The feature branch was created by the
plan step and is already checked out. All git operations here are isolated from
the main working tree.

Read the issue assessment artifact to find the issue number, repository, branch name, and issue URL.

## SAFETY: Do NOT Modify the Working Tree

This step MUST NOT run `git checkout`, `git stash`, or any command that changes
the current branch or working tree state. The branch already exists from the
implement step — just push it and create the {{ forge.pr_term }}.

## Instructions

### Step 1: Load Context

From the issue assessment artifact, extract:
- Issue number and title
- Repository (`owner/repo`)
- Branch name
- Issue URL

### Step 2: Push the Branch

Push the feature branch. If SSH push fails, retry with HTTPS:

```bash
git push -u origin <BRANCH_NAME> || GIT_SSH_COMMAND="ssh -F /dev/null" git push -u origin <BRANCH_NAME>
```

### Step 3: Create {{ forge.pr_term }}

Use the appropriate CLI for your platform ({{ forge.type }}) to create the {{ forge.pr_term }}.
The description MUST include `Related to #<NUMBER>` to link the issue (without auto-closing it when the PR is closed without merge).

**For GitHub** (`gh`):
```bash
gh pr create --repo <OWNER/REPO> --head <BRANCH_NAME> --title "<concise title>" --body "$(cat <<'EOF'
## Summary
<3-5 bullet points describing the changes>

Related to #<ISSUE_NUMBER>

## Changes
<list of key files changed and why>

## Test Plan
<how the changes were validated>
EOF
)"
```

**For GitLab** (`glab`):
```bash
cat > /tmp/mr-body.md <<'EOF'
## Summary
<3-5 bullet points describing the changes>

Related to #<ISSUE_NUMBER>

## Changes
<list of key files changed and why>

## Test Plan
<how the changes were validated>
EOF
glab mr create --repo <OWNER/REPO> --source-branch <BRANCH_NAME> --target-branch main --title '<concise title>' --description "$(cat /tmp/mr-body.md)"
```

**For Bitbucket** (REST API):
```bash
cat > /tmp/bb-payload.json << 'PRBODY'
{
  "title": "PR title",
  "description": "PR description\n\nRelated to #NUMBER",
  "source": {"branch": {"name": "BRANCH_NAME"}},
  "destination": {"branch": {"name": "main"}},
  "close_source_branch": true
}
PRBODY

curl -s -X POST -H "Authorization: Bearer $BB_TOKEN" -H "Content-Type: application/json" \
  -d @/tmp/bb-payload.json \
  "https://api.bitbucket.org/2.0/repositories/WORKSPACE/REPO/pullrequests" \
  | jq '{id, url: .links.html.href}'
```

**For Gitea** (`tea`):
```bash
cat > /tmp/pr-body.md <<'EOF'
## Summary
<3-5 bullet points describing the changes>

Related to #<ISSUE_NUMBER>

## Changes
<list of key files changed and why>

## Test Plan
<how the changes were validated>
EOF
tea pulls create --repo <OWNER/REPO> --head <BRANCH_NAME> --base main --title '<concise title>' --description "$(cat /tmp/pr-body.md)"
```

### Step 4: Request Review (Best-Effort)

After the {{ forge.pr_term }} is created, attempt to add a reviewer. This is a best-effort
operation — if it fails, the {{ forge.pr_term }} is still created successfully.

**For GitHub**: `gh pr edit --add-reviewer "copilot"`
**For GitLab**: `glab mr update <MR_NUMBER> --reviewer "<username>"`
**For Bitbucket**: Update PR via REST API with reviewers
**For Gitea**: Skip (not directly supported by tea CLI)

## CONSTRAINTS

- Do NOT spawn sub-agents — work directly in the main context
- Do NOT run `git checkout`, `git stash`, or any branch-switching commands
- The {{ forge.pr_term }} description MUST contain `Related to #<NUMBER>` to link to the issue
- Do NOT include Co-Authored-By or AI attribution in commits

## Output

Produce a JSON status report matching the injected output schema.
