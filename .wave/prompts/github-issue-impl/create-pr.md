You are creating a pull request for the implemented GitHub issue.

Input: {{ input }}

## Working Directory

You are running in an **isolated git worktree** shared with previous pipeline steps.
Your working directory IS the project root. The feature branch was created by the
plan step and is already checked out. All git operations here are isolated from
the main working tree.

The issue assessment is available as an injected artifact.
Read it to find the issue number, repository, branch name, and issue URL.

## SAFETY: Do NOT Modify the Working Tree

This step MUST NOT run `git checkout`, `git stash`, or any command that changes
the current branch or working tree state. The branch already exists from the
implement step — just push it and create the PR.

## Instructions

### Step 1: Load Context

Read the injected issue_assessment artifact to extract:
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

Create the PR using `gh pr create` with `--head` to target the branch. The PR body MUST include `Closes #<NUMBER>` to auto-close the issue on merge.

```bash
gh pr create --repo <OWNER/REPO> --head <BRANCH_NAME> --title "<concise title>" --body "$(cat <<'EOF'
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

### Step 4: Request Copilot Review (Best-Effort)

After the PR is created, attempt to add Copilot as a reviewer:
```bash
gh pr edit --add-reviewer "copilot"
```

This is a best-effort command. If Copilot isn't available in the repository, the command will fail silently and the PR will still be created successfully.

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT run `git checkout`, `git stash`, or any branch-switching commands
- The PR body MUST contain `Closes #<NUMBER>` to link to the issue
- Do NOT include Co-Authored-By or AI attribution in commits
