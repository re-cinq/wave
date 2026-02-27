You are implementing the planned changes for the Bitbucket issue.

Input: {{ input }}

## Working Directory

You are running in an **isolated git worktree**. Your working directory IS the
project root. The feature branch is already checked out.

Read the issue assessment and implementation plan artifacts.

## Instructions

### Step 1: Load Context

From the artifacts, understand:
- Issue requirements
- Implementation plan (files to modify/create, test strategy)
- Branch name

### Step 2: Implement Changes

Follow the implementation plan:
1. Make code changes as specified
2. Add/modify tests
3. Update documentation if needed
4. Run tests to verify changes

### Step 3: Commit Changes

Create atomic commits following conventional commit format:
```bash
git add <files>
git commit -m "feat: <description>

<detailed explanation>

Refs: #<ISSUE_NUMBER>"
```

### Step 4: Verify

Run the test suite to ensure everything passes:
```bash
go test ./...
```

If tests fail, fix the issues and commit the fixes.

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT push to remote — that happens in the create-pr step
- Do NOT create the PR yet — that's the next step
- Follow the project's conventional commit format
- Do NOT include Co-Authored-By or AI attribution in commits

## Output

The test suite contract will validate that all tests pass.
