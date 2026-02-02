---
description: Execute OpenSpec implementation tasks
---

## User Input

```text
$ARGUMENTS
```

## Setup

Run the apply script to get context:

```bash
bash .specify/scripts/bash/opsx-apply.sh --json $ARGUMENTS
```

## Instructions

1. Parse the JSON output to get file paths
2. Read the design.md to understand the technical approach
3. Read the tasks.md to get the task list
4. Execute tasks in dependency order:
   - Mark task as in-progress before starting
   - Implement the changes following the design
   - Run tests to verify the implementation
   - Mark task as complete when done
5. Update tasks.md status as work progresses
6. After all tasks complete, summarize changes and suggest /opsx.archive
