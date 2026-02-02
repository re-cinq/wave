---
description: Create a new OpenSpec change proposal
---

## User Input

```text
$ARGUMENTS
```

## Setup

Run the setup script to initialize the change:

```bash
bash .specify/scripts/bash/opsx-new.sh --json "$ARGUMENTS"
```

## Instructions

1. Parse the JSON output to get CHANGE_DIR and PROPOSAL_FILE paths
2. Read the generated proposal template at PROPOSAL_FILE
3. Guide the user through completing the proposal:
   - Ask clarifying questions about the change scope
   - Help define clear requirements
   - Establish measurable success criteria
4. Update the proposal.md with the refined content
5. Summarize what was created and next steps (/opsx.ff)
