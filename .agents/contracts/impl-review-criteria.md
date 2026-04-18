# Implementation Review Criteria

You are reviewing an implementation step's output. Evaluate whether the implementation:

## Correctness

1. **Not a no-op**: The implementation actually makes meaningful changes. A no-op (creating empty files, adding placeholder comments, or doing nothing) is a critical failure.

2. **Addresses the requirement**: The changes directly address the issue or task described in the input. Superficial changes that don't tackle the core requirement are a failure.

3. **No leaked files**: The implementation does not commit Wave-internal files that should not be in PRs:
   - `.claude/settings.json` or other `.claude/` files
   - `.wave/artifacts/` files
   - `.wave/output/` files
   - `CLAUDE.md` (unless specifically modifying Wave's CLAUDE.md)

## Code Quality

4. **Tests present**: If the implementation adds or modifies Go code, tests are present for the new/modified functionality. Missing tests for non-trivial code paths is a major issue.

5. **Tests pass**: The test suite passes (if a test command was run during implementation).

6. **No obvious bugs**: The implementation does not introduce obvious logic errors, nil pointer dereferences, or other correctness issues visible from the code changes.

## Scope

7. **Focused changes**: The changes are focused on the stated task. Unrelated refactoring or "improvements" beyond the task scope without justification are minor issues.

8. **Complete**: The implementation appears complete for the stated task — not obviously truncated or missing major components.

## Output Format

**CRITICAL: Your output must be ONLY the review JSON object. No preamble, no markdown fences, no wrapper text, no other output before or after the JSON.**

Return a structured verdict. Be specific in issues — mention file names and line numbers where relevant.

- Use verdict **pass** if all critical criteria pass (items 1-3) and most important criteria pass (4-6).
- Use verdict **warn** if no critical failures but notable issues exist.
- Use verdict **fail** if any critical failure (no-op, wrong approach, leaked files) or multiple major failures.

Set confidence based on how much of the implementation you could inspect.
