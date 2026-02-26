You are creating a merge request for the implemented GitLab issue.

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
implement step — just push it and create the merge request.

## Instructions

### Step 1: Load Context

From the issue assessment artifact, extract:
- Issue number and title
- Repository (`owner/repo`)
- Branch name
- Issue URL

### Step 2: Push the Branch

Push the feature branch without checking it out:

```bash
git push -u origin <BRANCH_NAME>
```

### Step 3: Create Merge Request

Create the merge request using `glab mr create` with `--source-branch` to target the branch. The merge request description MUST include `Closes #<NUMBER>` to auto-close the issue on merge.

```bash
glab mr create --repo <OWNER/REPO> --source-branch <BRANCH_NAME> --target-branch main --title "<concise title>" --description "$(cat <<'EOF'
## Summary
<3-5 bullet points describing the changes>

Closes #<ISSUE_NUMBER>

## Changes
<list of key files changed and why>

## Test Plan
<how the changes were validated>
EOF
)"
```

### Step 4: Request Review (Best-Effort)

After the merge request is created, attempt to add a reviewer:
```bash
glab mr update <MR_NUMBER> --reviewer "<username>"
```

This is a best-effort command. If the reviewer isn't available in the project, the command may fail and the merge request will still be created successfully.

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT run `git checkout`, `git stash`, or any branch-switching commands
- The merge request description MUST contain `Closes #<NUMBER>` to link to the issue
- Do NOT include Co-Authored-By or AI attribution in commits

## Output

Produce a JSON status report matching the injected output schema.
