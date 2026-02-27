You are creating an implementation plan for the Bitbucket issue.

Input: {{ input }}

## Working Directory

You are running in an **isolated git worktree**. Your working directory IS the
project root. The feature branch was created in the previous step and is
already checked out.

Read the issue assessment artifact to understand the issue requirements.

## Instructions

### Step 1: Load Context

From the issue assessment artifact, extract:
- Issue number, title, and body
- Repository information
- Branch name
- Skip steps (if any)
- Complexity assessment

### Step 2: Analyze Codebase

Read relevant files to understand:
- Current implementation patterns
- Testing approaches
- Documentation structure
- Code style and conventions

### Step 3: Design Implementation

Create a detailed implementation plan that includes:

1. **Files to modify**: List each file with specific changes
2. **Files to create**: New files needed with purpose
3. **Test strategy**: What tests to add/modify
4. **Edge cases**: Known scenarios to handle
5. **Rollback plan**: How to revert if needed

### Step 4: Identify Risks

List potential issues:
- Breaking changes
- Performance impacts
- Security considerations
- Backward compatibility

## CONSTRAINTS

- Do NOT spawn Task subagents — work directly in the main context
- Do NOT make code changes yet — this is planning only
- Do NOT modify the issue or create comments

## Output

Produce a JSON implementation plan matching the injected output schema.
