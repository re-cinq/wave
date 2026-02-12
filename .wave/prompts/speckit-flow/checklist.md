You are generating quality checklists to validate requirement completeness before
implementation.

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

Follow the `/speckit.checklist` workflow:

1. Set up the repo root reference (see above)
2. Read `artifacts/spec_info` and create a worktree for the feature branch:
   ```bash
   git -C "$REPO_ROOT" worktree add "$PWD/repo" <BRANCH_NAME>
   cd repo
   ```
3. Run `.specify/scripts/bash/check-prerequisites.sh --json` to get FEATURE_DIR
4. Load feature context: spec.md, plan.md, tasks.md
5. Generate focused checklists as "unit tests for requirements":
   - Each item tests the QUALITY of requirements, not the implementation
   - Use format: `- [ ] CHK### - Question about requirement quality [Dimension]`
   - Group by quality dimensions: Completeness, Clarity, Consistency, Coverage

6. Create the following checklist files in `FEATURE_DIR/checklists/`:
   - `review.md` — overall requirements quality validation
   - Additional domain-specific checklists as warranted by the feature

7. Commit checklists:
   ```bash
   git add specs/
   git commit -m "docs: add quality checklists"
   ```

8. Clean up worktree:
   ```bash
   cd "$OLDPWD"
   git -C "$REPO_ROOT" worktree remove "$PWD/repo"
   ```

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT use WebSearch — all information is in the spec artifacts

## Checklist Anti-Patterns (AVOID)

- WRONG: "Verify the button clicks correctly" (tests implementation)
- RIGHT: "Are interaction requirements defined for all clickable elements?" (tests requirements)

## Output

Write a JSON status report to output/checklist-status.json with:
```json
{
  "checklist_files": ["checklists/review.md"],
  "total_items": 25,
  "critical_gaps": 0,
  "feature_dir": "path to feature directory",
  "summary": "brief description of checklists created"
}
```
