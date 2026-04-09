You are creating a pull request for the implemented feature and requesting a review.

Feature context: {{ input }}

## Working Directory

You are running in an **isolated git worktree** shared with previous pipeline steps.
Your working directory IS the project root. The feature branch was created by a
previous step and is already checked out.

## Instructions

1. Find the branch name and feature directory from the spec info artifact

2. **Verify implementation**: Run `{{ project.test_command }}` one final time to confirm
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

6. **Create Pull Request**: Use `{{ forge.cli_tool }} {{ forge.pr_command }} create` with a descriptive summary. Write the PR body to a temp file to avoid shell injection:
   ```bash
   cat > /tmp/pr-body.md <<'EOF'
   ## Summary
   <3-5 bullet points describing the changes>

   ## Spec
   <link to spec file in specs/ directory>

   ## Test Plan
   <how the changes were validated>

   ## Known Limitations
   <any follow-up work needed>
   EOF
   {{ forge.cli_tool }} {{ forge.pr_command }} create --title '<concise title>' --body-file /tmp/pr-body.md
   ```

7. **Request Copilot Review**: After the PR is created, request a review from Copilot:
   ```bash
   {{ forge.cli_tool }} {{ forge.pr_command }} edit --add-reviewer "copilot"
   ```

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context

## Output

Produce a JSON status report matching the injected output schema.
