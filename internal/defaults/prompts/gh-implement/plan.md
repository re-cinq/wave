You are creating an implementation plan for a GitHub issue.

Input: {{ input }}

## Working Directory

You are running in an **isolated git worktree** checked out at `main` (detached HEAD).
Your working directory IS the project root. All git operations here are isolated
from the main working tree and will not affect it.

Create a feature branch from this clean starting point.

## Instructions

### Step 1: Read Assessment

From the issue assessment artifact, extract:
- Issue number, title, body, and repository
- Branch name from the assessment
- Complexity estimate
- Which speckit steps were skipped

### Step 2: Create Feature Branch

Create a feature branch using the branch name from the assessment:

```bash
git checkout -b <BRANCH_NAME>
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

Produce a JSON status report matching the injected output schema.
