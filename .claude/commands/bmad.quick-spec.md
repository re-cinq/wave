---
description: Analyze codebase and produce tech-spec with stories (BMAD Quick Path)
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

You are implementing the BMAD Quick Path - a streamlined workflow for bug fixes and small features. This command analyzes the codebase and produces a technical specification with implementation stories.

### Prerequisites

- User has provided a feature description (in $ARGUMENTS)
- Codebase is available for analysis

### Workflow

1. **Initialize Quick Spec**:
   Run `.specify/scripts/bash/bmad-quick-spec.sh --json "$ARGUMENTS"` to create the spec structure.
   Parse the JSON output to get SPEC_ID, SPEC_DIR, SPEC_FILE.

2. **Analyze Codebase** (as Winston - Architect):
   - Search for relevant files using patterns from the feature description
   - Identify existing patterns and conventions
   - Map integration points and dependencies
   - Document findings in the quick-spec.md

3. **Define Technical Approach** (as Amelia - Developer):
   - Based on codebase analysis, define required changes
   - Identify files to modify or create
   - Note any dependencies or prerequisites
   - Consider testing requirements

4. **Create Stories**:
   Break down the work into implementable stories:
   - Each story should be completable in 1-2 hours
   - Include clear acceptance criteria
   - Assign story points (1, 2, 3, 5, 8)
   - Order by dependency and priority

5. **Validate Spec** (as Bob - Scrum Master):
   Ensure the spec includes:
   - [ ] Clear feature description
   - [ ] Relevant files identified
   - [ ] Existing patterns documented
   - [ ] Technical approach defined
   - [ ] Stories with acceptance criteria
   - [ ] Dependencies identified

6. **Report**:
   Output the spec location and summary:
   - Spec ID and path
   - Number of stories created
   - Estimated total points
   - Recommended first story

### Agent Personas

Use these personas during the workflow:

- **Winston (Architect)**: For codebase analysis and pattern identification
- **Amelia (Developer)**: For technical approach and implementation details
- **Bob (Scrum Master)**: For story breakdown and validation

### Output Format

```markdown
## Quick Spec Created

**ID**: [SPEC_ID]
**Path**: [SPEC_FILE]

### Summary
- **Feature**: [Description]
- **Stories**: [Count]
- **Total Points**: [Sum]

### Stories
1. [Story 1 title] - [Points] pts
2. [Story 2 title] - [Points] pts
...

### Next Steps
Run `/bmad.dev-story 1` to implement the first story.
```

### Error Handling

- If $ARGUMENTS is empty: Prompt user for feature description
- If codebase analysis finds no relevant files: Expand search or ask for clarification
- If feature is too large for Quick Path: Suggest `/bmad.product-brief` instead
