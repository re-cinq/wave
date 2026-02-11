You are creating a pull request for the implemented GitHub issue.

Input: {{ input }}

## IMPORTANT: Working Directory

Your current working directory is a Wave workspace, NOT the project root.
Before running any commands, navigate to the project root:

```bash
cd "$(git rev-parse --show-toplevel)"
```

Run this FIRST before any other bash commands.

The issue assessment is available at `artifacts/issue_assessment`.
Read it to find the issue number, repository, branch name, and issue URL.

## Instructions

### Step 1: Load Context

Read `artifacts/issue_assessment` to extract:
- Issue number and title
- Repository (`owner/repo`)
- Branch name
- Issue URL

### Step 2: Check Out Branch and Verify

```bash
git checkout <BRANCH_NAME>
```

Run final test validation:
```bash
go test -race ./...
```

If tests fail, fix them before proceeding.

### Step 3: Stage and Commit

1. Review all changes with `git status` and `git diff`
2. Stage relevant files — exclude sensitive files (.env, credentials)
3. Create well-structured commits:
   - Use conventional commit prefixes: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`
   - Write concise commit messages focused on the "why"
   - Do NOT include Co-Authored-By or AI attribution lines
   - Reference the issue number in the commit message (e.g. `feat: add X for #42`)

### Step 4: Push

```bash
git push -u origin HEAD
```

### Step 5: Create Pull Request

Create the PR using `gh pr create`. The PR body MUST include `Closes #<NUMBER>` to auto-close the issue on merge.

```bash
gh pr create --repo <OWNER/REPO> --title "<concise title>" --body "$(cat <<'EOF'
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

### Step 6: Request Copilot Review

After the PR is created, optionally request a Copilot review. This is a best-effort call that fails silently if Copilot isn't available, but the PR will still be created successfully:

```bash
gh pr edit --add-reviewer "copilot"
```

> **Note**: This command may fail if GitHub Copilot reviews are not configured in the repository. The failure is expected and does not affect PR creation.

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
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
