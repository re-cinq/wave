---
description: Validate code quality and compliance using BMAD review checklist
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

You are performing a BMAD code review, validating quality and compliance against the review checklist. This can be used for self-review or to prepare for peer review.

### Prerequisites

- Code changes exist to review
- Optionally: story ID or PR URL in $ARGUMENTS

### Workflow

1. **Initialize Review**:
   Parse $ARGUMENTS for optional parameters:
   - `--story <id>`: Story being reviewed
   - `--pr <url>`: PR URL
   - `--self`: Self-review mode (less formal)

   Run `.specify/scripts/bash/bmad-review.sh --json [options]` to create review file.

2. **Gather Context**:
   - Get list of changed files: `git diff --name-only HEAD~1` or `git diff --staged --name-only`
   - Get diff statistics
   - Load related story if specified
   - Load constitution from `.specify/memory/constitution.md`

3. **Functional Correctness Review**:

   a. **Requirements Alignment**:
      - [ ] Implementation matches acceptance criteria
      - [ ] All user scenarios addressed
      - [ ] Edge cases handled
      - [ ] Error states handled gracefully

   b. **Logic Verification**:
      - [ ] Business logic is correct
      - [ ] Data transformations accurate
      - [ ] State management proper
      - [ ] No off-by-one errors

4. **Code Quality Review**:

   a. **Readability**:
      - [ ] Code is self-documenting
      - [ ] Variable/function names clear
      - [ ] Complex logic has comments
      - [ ] No unnecessary comments

   b. **Structure**:
      - [ ] Single responsibility followed
      - [ ] Functions appropriately sized
      - [ ] Code duplication minimized
      - [ ] Abstractions appropriate

5. **Testing Review**:

   - [ ] Unit tests cover new functionality
   - [ ] Unit tests cover edge cases
   - [ ] Unit tests cover error paths
   - [ ] All existing tests pass

6. **Security Review**:

   - [ ] No sensitive data logged
   - [ ] No credentials in code
   - [ ] Input validation present
   - [ ] No injection vulnerabilities

7. **Constitution Compliance**:
   Check against Wave project constitution principles:
   - [ ] No runtime dependencies introduced
   - [ ] Manifest remains single source of truth
   - [ ] Persona-scoped execution maintained
   - [ ] Fresh memory at step boundaries
   - [ ] Credentials not touching disk

8. **Generate Review Report**:

   ```markdown
   ## Code Review Report

   **Review ID**: [ID]
   **Date**: [Date]
   **Files Changed**: [Count]
   **Lines**: +[Added] / -[Removed]

   ### Summary
   [Brief overview of changes]

   ### Checklist Results

   #### Functional Correctness
   - [x] Requirements alignment
   - [x] Logic verification

   #### Code Quality
   - [x] Readability
   - [x] Structure

   #### Testing
   - [x] Unit tests
   - [ ] Integration tests (N/A)

   #### Security
   - [x] No vulnerabilities found

   #### Constitution Compliance
   - [x] All principles followed

   ### Issues Found

   #### Blocking
   - [Issue requiring fix before merge]

   #### Suggestions
   - [Optional improvement]

   ### Verdict
   - [ ] **Approved** - Ready to merge
   - [x] **Approved with comments** - Minor issues
   - [ ] **Changes Requested** - Must revise
   ```

### Review Modes

**Self-Review** (`--self`):
- Focus on functional correctness and testing
- Less formal documentation
- Quick turnaround

**PR Review** (`--pr <url>`):
- Full checklist
- Detailed documentation
- Formal verdict

**Story Review** (`--story <id>`):
- Validate against acceptance criteria
- Check story completion
- Update story status

### Error Handling

- If no changes found: Check for uncommitted changes
- If story not found: List available stories
- If tests fail: Include in blocking issues
