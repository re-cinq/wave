You are creating a pull request for the implemented GitHub issue.

Input: {{ input }}

## IMPORTANT: Workspace Isolation via Git Worktree

Your current working directory is a Wave workspace, NOT the project root.
Use `git worktree` to create an isolated checkout — this allows multiple pipeline runs to work concurrently without conflicts.

```bash
REPO_ROOT="$(git rev-parse --show-toplevel)"
```

The issue assessment is available at `artifacts/issue_assessment`.
Read it to find the issue number, repository, branch name, and issue URL.

## Instructions

### Step 1: Load Context

Read `artifacts/issue_assessment` to extract:
- Issue number and title
- Repository (`owner/repo`)
- Branch name
- Issue URL

### Step 2: Create Worktree and Verify

Create an isolated worktree for the feature branch:

```bash
git -C "$REPO_ROOT" worktree add "$PWD/repo" <BRANCH_NAME>
cd repo
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

After the PR is created:
```bash
gh pr edit --add-reviewer "copilot"
```

### Step 7: Clean Up Worktree

Remove the worktree reference:

```bash
cd "$OLDPWD"
git -C "$REPO_ROOT" worktree remove "$PWD/repo"
```

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
