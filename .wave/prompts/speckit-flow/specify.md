You are creating a feature specification for the following request:

{{ input }}

## Working Directory

You are running in an **isolated git worktree** checked out at `main` (detached HEAD).
Your working directory IS the project root. All git operations here are isolated
from the main working tree and will not affect it.

Use `create-new-feature.sh` to create the feature branch from this clean starting point.

## Instructions

Follow the `/speckit.specify` workflow to generate a complete feature specification:

1. Generate a concise short name (2-4 words) for the feature branch
3. Check existing branches to determine the next available number:
   ```bash
   git fetch --all --prune
   git ls-remote --heads origin | grep -E 'refs/heads/[0-9]+-'
   git branch | grep -E '^[* ]*[0-9]+-'
   ```
4. Run the feature creation script:
   ```bash
   .specify/scripts/bash/create-new-feature.sh --json --number <N> --short-name "<name>" "{{ input }}"
   ```
5. Load `.specify/templates/spec-template.md` for the required structure
6. Write the specification to the SPEC_FILE returned by the script
7. Create the quality checklist at `FEATURE_DIR/checklists/requirements.md`
8. Run self-validation against the checklist (up to 3 iterations)

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
