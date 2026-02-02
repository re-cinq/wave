---
description: Get started with OpenSpec workflow
---

## User Input

```text
$ARGUMENTS
```

## OpenSpec Workflow Overview

OpenSpec is a lightweight change management workflow for AI-assisted development. It provides structure without bureaucracy.

### Core Concepts

1. **Change Proposal** (`proposal.md`) - What you want to build and why
2. **Design Doc** (`design.md`) - How you'll build it technically
3. **Task List** (`tasks.md`) - Actionable implementation steps

### Directory Structure

```
openspec/
  changes/
    <change-slug>/
      proposal.md    # Requirements and success criteria
      design.md      # Technical design decisions
      tasks.md       # Implementation checklist
      specs/         # Additional specifications
  archive/           # Completed changes
```

### Workflow Commands

| Command | Description |
|---------|-------------|
| `/opsx.new <name>` | Create a new change proposal |
| `/opsx.ff` | Fast-forward: generate design and tasks |
| `/opsx.apply` | Execute implementation tasks |
| `/opsx.archive --change <slug>` | Archive completed change |

### Quick Start

1. **Start a new change:**
   ```
   /opsx.new Add user authentication
   ```

2. **Generate planning docs:**
   ```
   /opsx.ff
   ```

3. **Implement the change:**
   ```
   /opsx.apply
   ```

4. **Archive when complete:**
   ```
   /opsx.archive --change add-user-authentication
   ```

## Instructions

If the user provided arguments, help them with that specific question.

Otherwise, explain the OpenSpec workflow and ask what change they'd like to start with.
