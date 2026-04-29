# SWE-bench Item: Fix the Issue

You are solving a software engineering challenge from the SWE-bench benchmark. Your goal is to produce the smallest, most targeted code change that correctly resolves the described problem while keeping all existing tests passing.

## Problem Statement

{{ input }}

## Objective

Read the problem statement above and produce a minimal, correct fix. The fix should change as few lines as possible, touch as few files as possible, and introduce no new functionality beyond what is required to resolve the issue. This is a benchmark exercise where precision and minimality are the primary evaluation criteria.

## Context

You have access to the full project codebase via your workspace. The problem statement describes a bug, missing feature, or incorrect behavior that needs to be fixed. The fix may involve modifying source code, test code, or configuration files. The project has an existing test suite that must continue to pass after your changes. In many cases, the problem statement references specific files, functions, classes, or test cases that pinpoint the location of the issue.

## Requirements

Follow these steps in order. Do not skip any step.

### 1. Understand the Problem (read before you write)

Read the problem statement carefully, multiple times if needed. Extract these key pieces of information:
- **What is the current (incorrect) behavior?** What does the code do now that it should not?
- **What is the expected (correct) behavior?** What should the code do instead?
- **Which files and functions are involved?** The problem statement often names specific modules, classes, functions, or test cases.
- **What is the reproduction path?** How would someone trigger the bug or observe the incorrect behavior?
- **Are there specific test cases mentioned?** These tests define the acceptance criteria for your fix.

Do not start coding until you can articulate all five of these points.

### 2. Explore the Codebase (map the territory)

Use the available file-search tooling to find and understand the relevant source files:
- Locate the files mentioned in the problem statement
- Read the functions or classes that are involved in the bug
- Understand the existing behavior by reading the code, not by guessing
- Find the test files that exercise this code
- Check for related code that might be affected by your change (callers of the function you are modifying, subclasses that override the method, etc.)
- If the problem involves an edge case, understand the general case first

Spend enough time exploring that you understand WHY the current code produces the wrong behavior, not just WHERE it does.

### 3. Plan the Fix (think before you act)

Before making any changes, determine:
- **What is the root cause?** Not the symptom, but the underlying reason the code is wrong.
- **What is the minimal change?** The smallest edit that addresses the root cause.
- **What are the side effects?** Could your change break any other behavior? Check callers and tests.
- **Is there a test that validates the fix?** If the problem statement mentions a failing test, your fix should make it pass. If no test exists, the existing test suite must still pass.

### 4. Implement the Fix (minimal, surgical change)

Make the change:
- Edit only the files that need to change
- Change only the lines that need to change
- Preserve existing code style, formatting, and conventions
- Do not rename variables, reformat code, or make any cosmetic changes
- Do not add comments explaining your fix (the code should speak for itself)
- Do not add new imports unless your fix requires them
- If you need to add a new function or method, keep it as small as possible

### 5. Verify the Fix (prove it works)

Run the project's test suite to confirm:
- The previously-failing test (if mentioned) now passes
- All other existing tests continue to pass
- No new warnings or errors are introduced

If tests fail after your change, re-read the problem statement, re-examine your fix, and iterate. Do not move on until all tests pass.

## Constraints and Anti-patterns

- **Minimality is paramount**: The fewer lines changed, the better. A one-line fix is preferred over a ten-line fix if both are correct.
- **Do not refactor**: Do not improve code quality, readability, or performance unless the problem statement specifically asks for it.
- **Do not add features**: Do not implement functionality beyond what the problem statement describes, even if it seems like a good idea.
- **Do not create new files** unless the fix absolutely requires it. Most SWE-bench tasks can be solved by editing existing files.
- **Do not modify test files** unless the problem statement asks you to fix or add a test.
- **Do not add documentation**: No docstrings, comments, or README changes unless the problem statement asks for them.
- **Preserve all existing tests**: If your change causes an unrelated test to fail, your fix is wrong or too broad. Narrow it.
- **Do not guess**: If you are unsure which approach is correct, re-read the problem statement and the relevant code. The answer is in the codebase.

## Quality Bar

A correct solution fixes exactly the described problem with the smallest possible diff. It does not introduce new bugs, does not change unrelated behavior, and passes the full test suite. An incorrect solution changes too many things, breaks existing tests, or fixes a different problem than the one described. The benchmark evaluates both correctness (does the fix work?) and minimality (is the diff as small as it can be?).
