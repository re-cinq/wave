You are creating a pull request for the implemented Bitbucket issue.

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
implement step — just push it and create the PR.

## Instructions

### Step 1: Load Context

From the issue assessment artifact, extract:
- Issue number and title
- Repository (`workspace/repo`)
- Branch name
- Issue URL

### Step 2: Push the Branch

Push the feature branch. If SSH push fails, retry with HTTPS:

```bash
git push -u origin <BRANCH_NAME> || GIT_SSH_COMMAND="ssh -F /dev/null" git push -u origin <BRANCH_NAME>
```

### Step 3: Create Pull Request

Create the PR via the Bitbucket REST API. The PR description MUST include `Related to #<NUMBER>` to link the issue (without auto-closing it when the PR is closed without merge).

Write the PR payload to a temp file to avoid shell escaping issues:

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

### Step 4: Request Review (Best-Effort)

After the PR is created, attempt to add reviewers by updating the PR:
```bash
cat > /tmp/bb-payload.json << 'EOF'
{"reviewers": [{"username": "reviewer-username"}]}
EOF
curl -s -X PUT -H "Authorization: Bearer $BB_TOKEN" -H "Content-Type: application/json" \
  -d @/tmp/bb-payload.json \
  "https://api.bitbucket.org/2.0/repositories/WORKSPACE/REPO/pullrequests/ID"
```

This is a best-effort operation. If reviewers aren't configured, the command will fail and the PR will still be created successfully.

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT run `git checkout`, `git stash`, or any branch-switching commands
- The PR description MUST contain `Related to #<NUMBER>` to link to the issue
- Do NOT include Co-Authored-By or AI attribution in commits

## Output

Produce a JSON status report matching the injected output schema.
