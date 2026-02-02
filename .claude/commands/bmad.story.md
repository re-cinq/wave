---
description: Create detailed story specification from epic
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

You are Mary (Business Analyst) with Amelia (Developer) creating a detailed story specification. This expands a story from the epics document into a full implementation-ready spec.

### Prerequisites

- Story ID provided in $ARGUMENTS (e.g., "1.1" or "2.3")
- Epics document exists with the story
- Product slug optionally in $ARGUMENTS

### Workflow

1. **Load Story Context**:
   - Find the story by ID in epics.md
   - Load parent epic for context
   - Load architecture.md for technical details
   - Load related stories (dependencies)

2. **Expand User Story** (as Mary - BA):

   ```markdown
   ## Overview

   ### User Story
   As a **[persona]**, I want to **[action]** so that **[benefit]**.

   ### Priority
   **Level**: [P0/P1/P2]
   **Rationale**: [Why this priority]

   ### Estimation
   **Story Points**: [X]
   **Confidence**: [High/Medium/Low]
   ```

3. **Add Context**:

   ```markdown
   ## Context

   ### Background
   [Additional context about why this story exists]

   ### Related Stories
   - **Blocked by**: [Story ID] - [Status]
   - **Blocks**: [Story ID]
   - **Related**: [Story ID] - [Relationship]

   ### Out of Scope
   - [Explicitly excluded functionality]
   ```

4. **Detail Acceptance Criteria** (as Mary - BA):

   For each criterion, use Given-When-Then format:

   ```markdown
   ### AC-1: [Criterion Title]
   **Given** [initial context/state]
   **When** [action is taken]
   **Then** [expected result]

   **Verification**: [How to verify - manual test, automated test, etc.]
   ```

   Include edge cases:
   ```markdown
   ### Edge Cases

   #### EC-1: [Edge Case Title]
   **Given** [edge case context]
   **When** [action]
   **Then** [expected behavior]
   ```

5. **Technical Details** (as Amelia - Developer):

   ```markdown
   ## Technical Details

   ### Implementation Notes
   - [Technical consideration 1]
   - [Recommended approach]

   ### Files to Modify
   - `path/to/file.go` - [What changes]

   ### New Files
   - `path/to/new/file.go` - [Purpose]

   ### API Changes
   [Contract changes if applicable]

   ### Database Changes
   [Schema changes if applicable]
   ```

6. **Testing Requirements**:

   ```markdown
   ## Testing Requirements

   ### Unit Tests
   - [ ] Test [specific behavior]
   - [ ] Test [edge case]

   ### Integration Tests
   - [ ] Test [integration scenario]

   ### Manual Testing
   - [ ] Verify [user-facing behavior]
   ```

7. **Definition of Done**:

   ```markdown
   ## Definition of Done

   - [ ] All acceptance criteria met
   - [ ] Unit tests written and passing
   - [ ] Integration tests passing
   - [ ] Code reviewed and approved
   - [ ] Documentation updated
   - [ ] No regression in existing functionality
   - [ ] Feature flag configured (if applicable)
   ```

8. **Write Story File**:
   Create `.bmad/products/[slug]/epics/stories/story-[id].md`

9. **Report**:

   ```markdown
   ## Story [ID] Detailed

   **Title**: [Story title]
   **Path**: [STORY_FILE]

   ### Summary
   - **Points**: [X]
   - **Acceptance Criteria**: [Count]
   - **Edge Cases**: [Count]
   - **Files to Modify**: [Count]

   ### Dependencies
   - Blocked by: [Stories]
   - Blocks: [Stories]

   ### Next Steps
   1. Review with development team
   2. Run `/bmad.dev-story [id]` to implement
   ```

### Error Handling

- If story ID not provided: List available stories
- If story not found: Check epics document
- If architecture missing: Note technical details as TBD
