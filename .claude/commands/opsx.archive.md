---
description: Archive a completed OpenSpec change
---

## User Input

```text
$ARGUMENTS
```

## Setup

Run the archive script:

```bash
bash .specify/scripts/bash/opsx-archive.sh --json --force $ARGUMENTS
```

## Instructions

1. Parse the JSON output to confirm archival
2. If no --change was specified, list available changes and ask which to archive
3. Verify all tasks in tasks.md are marked complete before archiving
4. Report the archive location
5. Suggest creating a git commit to record the completed change
