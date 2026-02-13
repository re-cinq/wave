You are creating a pull request for the implemented GitHub issue.

Input: {{ input }}

The issue assessment is available at `artifacts/issue_assessment`.
Read it to find the issue number, repository, branch name, and issue URL.

## SAFETY: Do NOT Modify the Working Tree

This step MUST NOT run `git checkout`, `git stash`, or any command that changes
the current branch or working tree state. The branch already exists from the
implement step — just push it and create the PR.

## Instructions

### Step 1: Load Context

Read `artifacts/issue_assessment` to extract:
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

## Output

Write a JSON status report to `output/pr-result.json`:

```json
{
  "pr_url": "https://github.com/owner/repo/pull/123",
  "pr_number": 123,
  "issue_number": 42,
  "issue_url": "https://github.com/owner/repo/issues/42",
  "copilot_review_requested": true,
  "summary": "Brief description of the PR"
}
```
