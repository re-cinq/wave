You are creating a pull request for the implemented Gitea issue.

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
- Repository (`owner/repo`)
- Branch name
- Issue URL

### Step 2: Push the Branch

Push the feature branch without checking it out:

```bash
git push -u origin <BRANCH_NAME>
```

### Step 3: Create Pull Request

Create the PR using `tea pulls create` with `--head` to target the branch. The PR description MUST include `Closes #<NUMBER>` to auto-close the issue on merge.

```bash
tea pulls create --repo <OWNER/REPO> --head <BRANCH_NAME> --base main --title "<concise title>" --description "$(cat <<'EOF'
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

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT run `git checkout`, `git stash`, or any branch-switching commands
- The PR description MUST contain `Closes #<NUMBER>` to link to the issue
- Do NOT include Co-Authored-By or AI attribution in commits

## Output

Produce a JSON status report matching the injected output schema.
