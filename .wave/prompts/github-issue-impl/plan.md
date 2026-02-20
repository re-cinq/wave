You are creating an implementation plan for a GitHub issue.

Input: {{ input }}

## Working Directory

You are running in an **isolated git worktree** checked out at `main` (detached HEAD).
Your working directory IS the project root. All git operations here are isolated
from the main working tree and will not affect it.

Use `create-new-feature.sh` to create the feature branch from this clean starting point.

The issue assessment is available at `.wave/artifacts/issue_assessment`.
Read it to get the issue details, branch name, complexity, and assessment.

## Instructions

### Step 1: Read Assessment

Read `.wave/artifacts/issue_assessment` to extract:
- Issue number, title, body, and repository
- Branch name from the assessment
- Complexity estimate
- Which speckit steps were skipped

### Step 2: Create Feature Branch

Use the `create-new-feature.sh` script to create a properly numbered branch:

```bash
.specify/scripts/bash/create-new-feature.sh --json --number <ISSUE_NUMBER> --short-name "<SHORT_NAME>" "<ISSUE_TITLE>"
```

If the branch already exists (e.g. from a resume), check it out instead:
```bash
git checkout <BRANCH_NAME>
```

### Step 3: Write Spec from Issue

In the feature directory (e.g. `specs/<BRANCH_NAME>/`), create `spec.md` with:
- Issue title as heading
- Full issue body
- Labels and metadata
- Any acceptance criteria extracted from the issue
- Link back to the original issue URL

### Step 4: Create Implementation Plan

Write `plan.md` in the feature directory with:

1. **Objective**: What the issue asks for (1-2 sentences)
2. **Approach**: High-level strategy
3. **File Mapping**: Which files need to be created/modified/deleted
4. **Architecture Decisions**: Any design choices made
5. **Risks**: Potential issues and mitigations
6. **Testing Strategy**: What tests are needed

### Step 5: Create Task Breakdown

Write `tasks.md` in the feature directory with a phased breakdown:

```markdown
# Tasks

## Phase 1: Setup
- [ ] Task 1.1: Description
- [ ] Task 1.2: Description

## Phase 2: Core Implementation
- [ ] Task 2.1: Description [P] (parallelizable)
- [ ] Task 2.2: Description [P]

## Phase 3: Testing
- [ ] Task 3.1: Write unit tests
- [ ] Task 3.2: Write integration tests

## Phase 4: Polish
- [ ] Task 4.1: Documentation updates
- [ ] Task 4.2: Final validation
```

Mark parallelizable tasks with `[P]`.

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT start implementation — only planning in this step
- Do NOT use WebSearch — all information is in the issue and codebase

## Output

Write a JSON status report to `.wave/output/impl-plan.json`:

```json
{
  "issue_number": 42,
  "branch_name": "042-short-name",
  "feature_dir": "specs/042-short-name",
  "spec_file": "specs/042-short-name/spec.md",
  "plan_file": "specs/042-short-name/plan.md",
  "tasks_file": "specs/042-short-name/tasks.md",
  "tasks": [
    {
      "id": "1.1",
      "title": "Task title",
      "description": "What needs to be done",
      "file_changes": [
        {"path": "internal/foo/bar.go", "action": "modify"}
      ]
    }
  ],
  "summary": "Brief description of the plan"
}
```
