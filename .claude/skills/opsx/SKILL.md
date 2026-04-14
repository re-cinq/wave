---
name: opsx
description: OpenSpec workflow management system for creating, planning, implementing, and archiving specification-driven development changes with comprehensive lifecycle management
---

# OpenSpec Workflow (OpsX)

OpenSpec Workflow (OpsX) provides a comprehensive lifecycle management system for specification-driven development changes, from initial proposal through implementation and archival.

## Overview

OpsX implements a structured lifecycle:

1. **Onboard** - Get started with OpenSpec workflow
2. **New** - Create new OpenSpec change proposals
3. **Fast-Forward** - Generate complete planning documentation
4. **Apply** - Execute implementation tasks
5. **Archive** - Complete and archive finished changes

## Available Commands

| Command | Purpose | Usage |
|---------|---------|--------|
| `opsx.onboard` | Get started with OpenSpec workflow | `/opsx.onboard` |
| `opsx.new` | Create a new OpenSpec change proposal | `/opsx.new "change description"` |
| `opsx.ff` | Fast-forward to generate all planning docs | `/opsx.ff` |
| `opsx.apply` | Execute OpenSpec implementation tasks | `/opsx.apply` |
| `opsx.archive` | Archive a completed OpenSpec change | `/opsx.archive` |

## Standard Workflow

```bash
# Full lifecycle
/opsx.onboard                                    # first time only
/opsx.new "Add user authentication system"
/opsx.ff                                         # generate planning docs
/opsx.apply
/opsx.archive

# Quick implementation (urgent/small fixes)
/opsx.new "Fix critical security vulnerability"
/opsx.apply
/opsx.archive
```

## Change Structure

```
.openspec/
├── changes/
│   └── [change-id]/
│       ├── proposal.md      # Initial change proposal
│       ├── specification.md # Detailed requirements
│       ├── plan.md         # Implementation plan
│       ├── tasks.md        # Task breakdown
│       ├── implementation/ # Implementation artifacts
│       └── archive.md      # Completion summary
└── archive/
    └── [year]/[change-id]/ # Archived change artifacts
```

## Lifecycle Phases

| Phase | Command | Key Artifacts |
|-------|---------|--------------|
| Proposal | `opsx.new` | `proposal.md` |
| Planning | `opsx.ff` | `specification.md`, `plan.md`, `tasks.md` |
| Implementation | `opsx.apply` | Code, tests, docs in `implementation/` |
| Archive | `opsx.archive` | `archive.md` with completion summary |

## Best Practices

- **Single Responsibility** — each change addresses one specific concern
- **Clear Boundaries** — well-defined scope with explicit inclusions/exclusions
- **Task Granularity** — break work into manageable, trackable units
- **Lessons Learned** — document insights in archive for future reference

## Complete Reference

For exhaustive patterns, examples, and advanced usage see:

**[`references/full-reference.md`](references/full-reference.md)**
