---
description: Fast-forward to generate all OpenSpec planning docs
---

## User Input

```text
$ARGUMENTS
```

## Setup

Run the fast-forward script:

```bash
bash .specify/scripts/bash/opsx-ff.sh --json $ARGUMENTS
```

## Instructions

1. Parse the JSON output to get file paths (PROPOSAL_FILE, DESIGN_FILE, TASKS_FILE)
2. Read the proposal.md to understand the change requirements
3. Generate comprehensive design.md content:
   - Technical approach and architecture decisions
   - Component interactions and data flow
   - Edge cases and error handling
   - Testing strategy
4. Generate tasks.md with actionable implementation tasks:
   - Break down into small, testable units
   - Define clear acceptance criteria
   - Establish task dependencies
5. Update both files with the generated content
6. Summarize the planning docs and next steps (/opsx.apply)
