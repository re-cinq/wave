You are creating a pull request for the implemented feature and requesting a review.

Feature context: {{ input }}

## Working Directory

You are running in an **isolated git worktree** shared with previous pipeline steps.
Your working directory IS the project root. The feature branch was created by a
previous step and is already checked out.

A status report from the specify step is available at `.wave/artifacts/spec_info`.
Read it to find the branch name, spec file, and feature directory.

## Instructions

1. Read `.wave/artifacts/spec_info` to find the branch name and feature directory

2. **Verify implementation**: Run `go test -race ./...` one final time to confirm
   all tests pass. If tests fail, fix them before proceeding.

3. **Stage changes**: Review all modified and new files with `git status` and `git diff`.
   Stage relevant files — exclude any sensitive files (.env, credentials).

4. **Commit**: Create a well-structured commit (or multiple commits if logical):
   - Use conventional commit prefixes: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`
   - Write concise commit messages focused on the "why"
   - Do NOT include Co-Authored-By or AI attribution lines

5. **Push**: Push the branch to the remote repository:
   ```bash
   git push -u origin HEAD
   ```

6. **Create Pull Request**: Use `gh pr create` with a descriptive summary:
   ```bash
   gh pr create --title "<concise title>" --body "<PR body with summary and test plan>"
   ```

   The PR body should include:
   - Summary of changes (3-5 bullet points)
   - Link to the spec file in the specs/ directory
   - Test plan describing how changes were validated
   - Any known limitations or follow-up work needed

7. **Request Copilot Review**: After the PR is created, request a review from Copilot:
   ```bash
   gh pr edit --add-reviewer "copilot"
   ```

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context

## Output

Write a JSON status report to .wave/output/pr-result.json with:
```json
{
  "pr_url": "https://github.com/...",
  "pr_number": 42,
  "copilot_review_requested": true,
  "summary": "brief description of PR"
}
```
