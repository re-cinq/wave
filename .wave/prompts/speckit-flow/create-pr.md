You are creating a pull request for the implemented feature and requesting a review.

Feature context: {{ input }}

## IMPORTANT: Working Directory

Your current working directory is a Wave workspace, NOT the project root.
Before running any scripts or accessing project files, navigate to the project root:

```bash
cd "$(git rev-parse --show-toplevel)"
```

Run this FIRST before any other bash commands.

A status report from the specify step is available at `artifacts/spec_info`.
Read it to find the branch name, spec file, and feature directory.

## Instructions

1. Navigate to the project root (see above)
2. Read `artifacts/spec_info` and check out the feature branch

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
