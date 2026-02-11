You are creating a feature specification for the following request:

{{ input }}

## IMPORTANT: Workspace Isolation via Git Worktree

Your current working directory is a Wave workspace, NOT the project root.
Use `git worktree` to create an isolated checkout — this allows multiple pipeline runs
to work concurrently without conflicts.

```bash
REPO_ROOT="$(git rev-parse --show-toplevel)"
```

## Instructions

Follow the `/speckit.specify` workflow to generate a complete feature specification:

1. Set up the repo root reference (see above)
2. Generate a concise short name (2-4 words) for the feature branch
3. Check existing branches to determine the next available number:
   ```bash
   git -C "$REPO_ROOT" fetch --all --prune
   git -C "$REPO_ROOT" ls-remote --heads origin | grep -E 'refs/heads/[0-9]+-'
   git -C "$REPO_ROOT" branch | grep -E '^[* ]*[0-9]+-'
   ```
4. Create the feature branch and worktree:
   ```bash
   cd "$REPO_ROOT"
   .specify/scripts/bash/create-new-feature.sh --json --number <N> --short-name "<name>" "{{ input }}"
   cd "$OLDPWD"
   git -C "$REPO_ROOT" worktree add "$PWD/repo" <BRANCH_NAME>
   cd repo
   ```
5. Load `.specify/templates/spec-template.md` for the required structure
6. Write the specification to the SPEC_FILE returned by the script
7. Create the quality checklist at `FEATURE_DIR/checklists/requirements.md`
8. Run self-validation against the checklist (up to 3 iterations)
9. Commit planning artifacts:
   ```bash
   git add specs/
   git commit -m "docs: add feature spec for <short-name>"
   ```
10. Clean up worktree:
    ```bash
    cd "$OLDPWD"
    git -C "$REPO_ROOT" worktree remove "$PWD/repo"
    ```

## Agent Usage

Use 1-3 Task agents to parallelize independent work:
- Agent 1: Analyze the codebase to understand existing patterns and architecture
- Agent 2: Research domain-specific best practices for the feature
- Agent 3: Draft specification sections in parallel

## Quality Standards

- Focus on WHAT and WHY, not HOW (no implementation details)
- Every requirement must be testable and unambiguous
- Maximum 3 `[NEEDS CLARIFICATION]` markers — make informed guesses for the rest
- Include user stories with acceptance criteria, data model, edge cases
- Success criteria must be measurable and technology-agnostic

## Output

Write a JSON status report to output/specify-status.json with:
```json
{
  "branch_name": "the created branch name",
  "spec_file": "path to spec.md",
  "feature_dir": "path to feature directory",
  "checklist_status": "pass or fail",
  "summary": "brief description of what was created"
}
```
