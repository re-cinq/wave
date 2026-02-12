You are creating a pull request for the implemented feature and requesting a review.

Feature context: {{ input }}

## IMPORTANT: Workspace Isolation via Git Worktree

Your current working directory is a Wave workspace, NOT the project root.
Use `git worktree` to create an isolated checkout — this allows multiple pipeline runs
to work concurrently without conflicts.

```bash
REPO_ROOT="$(git rev-parse --show-toplevel)"
```

A status report from the specify step is available at `artifacts/spec_info`.
Read it to find the branch name, spec file, and feature directory.

## Instructions

1. Set up the repo root reference (see above)
2. Read `artifacts/spec_info` and create a worktree for the feature branch:
   ```bash
   git -C "$REPO_ROOT" worktree add "$PWD/repo" <BRANCH_NAME>
   cd repo
   ```

3. **Verify implementation**: Run `go test -race ./...` one final time to confirm
   all tests pass. If tests fail, fix them before proceeding.

4. **Stage changes**: Review all modified and new files with `git status` and `git diff`.
   Stage relevant files — exclude any sensitive files (.env, credentials).

5. **Commit**: Create a well-structured commit (or multiple commits if logical):
   - Use conventional commit prefixes: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`
   - Write concise commit messages focused on the "why"
   - Do NOT include Co-Authored-By or AI attribution lines

6. **Push**: Push the branch to the remote repository:
   ```bash
   git push -u origin HEAD
   ```

7. **Create Pull Request**: Use `gh pr create` with a descriptive summary:
   ```bash
   gh pr create --title "<concise title>" --body "<PR body with summary and test plan>"
   ```

   The PR body should include:
   - Summary of changes (3-5 bullet points)
   - Link to the spec file in the specs/ directory
   - Test plan describing how changes were validated
   - Any known limitations or follow-up work needed

8. **Request Copilot Review**: After the PR is created, request a review from Copilot:
   ```bash
   gh pr edit --add-reviewer "copilot"
   ```

9. **Clean up worktree**:
   ```bash
   cd "$OLDPWD"
   git -C "$REPO_ROOT" worktree remove "$PWD/repo"
   ```

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context

## Output

Write a JSON status report to output/pr-result.json with:
```json
{
  "pr_url": "https://github.com/...",
  "pr_number": 42,
  "copilot_review_requested": true,
  "summary": "brief description of PR"
}
```
