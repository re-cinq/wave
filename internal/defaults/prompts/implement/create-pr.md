# Create {{ forge.pr_term }}

You are in an **isolated git worktree** with the feature branch already checked out.

## Context

Read the issue assessment artifact at `.wave/artifacts/issue_assessment` to get:
- OWNER/REPO (from repository.owner and repository.name)
- BRANCH_NAME (from branch_name)
- ISSUE_NUMBER (from issue_number)
- TITLE (from title)

You are working on a **{{ forge.type }}** repository.

## Step 1: Verify Branch

Confirm you are on the correct feature branch:
```
git branch --show-current
```

The branch should already exist from the implement step. Do NOT run `git checkout` or `git stash`.

## Step 2: Push Branch

Push the feature branch to the remote:
```
git push -u origin <BRANCH_NAME>
```

If that fails due to SSH issues, try:
```
GIT_SSH_COMMAND="ssh -F /dev/null" git push -u origin <BRANCH_NAME>
```

## Step 3: Write Title File

Write the title to `/tmp/pr-title.txt` so it can be read safely by CLI commands:
```
echo 'The PR/MR title here' > /tmp/pr-title.txt
```

## Step 4: Create {{ forge.pr_term }}

### For GitHub ({{ forge.type }} = github):
```
gh pr create --repo <OWNER/REPO> --head <BRANCH_NAME> \
  --title "$(cat /tmp/pr-title.txt)" \
  --body-file /tmp/pr-body.md
```

### For GitLab ({{ forge.type }} = gitlab):
```
glab mr create --repo <OWNER/REPO> --source-branch <BRANCH_NAME> \
  --target-branch main \
  --title "$(cat /tmp/pr-title.txt)" \
  --description "$(cat /tmp/pr-body.md)"
```

### For Bitbucket ({{ forge.type }} = bitbucket):
Write the JSON payload to a file and POST to the Bitbucket API.
Use Write tool to create `/tmp/bb-payload.json` with the PR details:
```json
{
  "title": "The PR title here",
  "description": "Body content here\n\nCloses #NUMBER",
  "source": {"branch": {"name": "BRANCH_NAME"}},
  "destination": {"branch": {"name": "main"}},
  "close_source_branch": true
}
```
Then POST it:
```
curl -s -X POST \
  -H "Authorization: Bearer $BB_TOKEN" \
  -H "Content-Type: application/json" \
  -d @/tmp/bb-payload.json \
  "https://api.bitbucket.org/2.0/repositories/OWNER/REPO/pullrequests"
```

### For Gitea ({{ forge.type }} = gitea):
```
tea pulls create --repo <OWNER/REPO> --head <BRANCH_NAME> --base main \
  --title "$(cat /tmp/pr-title.txt)" \
  --body "$(cat /tmp/pr-body.md)"
```

## {{ forge.pr_term }} Body

Before creating the {{ forge.pr_term }}, write the body to `/tmp/pr-body.md`:

```
## Summary
<One-paragraph summary of changes>

## Changes
<Bullet list of key changes>

## Test Plan
<How to verify the changes work>

Closes #<ISSUE_NUMBER>
```

## Step 5: Request Review (Best Effort)

Attempt to request a reviewer. This step is best-effort -- do not fail if it doesn't work.

### For GitHub: `gh pr edit <PR_NUMBER> --add-reviewer ...`
### For GitLab: `glab mr update <MR_NUMBER> --reviewer "<username>"`
### For Bitbucket: Use the PR participants API endpoint
### For Gitea: Skip (not supported via tea CLI)

## Output

Write the result to `.wave/output/pr-result.json` matching the contract schema:
- pr_url: The URL of the created {{ forge.pr_term }}
- pr_number: The number/ID
- branch_name: The feature branch name
- status: "created" or "failed"
- title: The {{ forge.pr_term }} title

## Safety Constraints

- Do NOT run `git checkout`, `git stash`, or any branch-switching commands
- Do NOT modify the working tree
- The worktree already has the feature branch with all committed changes
