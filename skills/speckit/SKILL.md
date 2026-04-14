---
name: speckit
description: Specification-driven development workflow tools for creating, analyzing, and implementing feature specifications with automated planning and validation
---

# SpecKit - Specification-Driven Development Tools

SpecKit provides a comprehensive workflow for specification-driven development, from initial feature description to implementation planning.

## Overview

SpecKit follows a structured workflow:
1. **Specify** - Create feature specifications from natural language
2. **Clarify** - Identify and resolve underspecified areas
3. **Plan** - Generate implementation plans and design artifacts
4. **Validate** - Analyze specifications for consistency and quality
5. **Implement** - Execute implementation plans with task tracking

## Available Commands

### Core Workflow
| Command | Purpose | Usage |
|---------|---------|--------|
| `speckit.specify` | Create or update feature specification | `/speckit.specify "feature description"` |
| `speckit.clarify` | Identify underspecified areas and ask clarification questions | `/speckit.clarify` |
| `speckit.plan` | Generate implementation plans and design artifacts | `/speckit.plan` |
| `speckit.implement` | Execute implementation plan with task tracking | `/speckit.implement` |

### Analysis & Validation
| Command | Purpose | Usage |
|---------|---------|--------|
| `speckit.analyze` | Cross-artifact consistency and quality analysis | `/speckit.analyze` |
| `speckit.tasks` | Generate actionable, dependency-ordered task list | `/speckit.tasks` |
| `speckit.checklist` | Generate custom validation checklist | `/speckit.checklist` |

### Project Setup
| Command | Purpose | Usage |
|---------|---------|--------|
| `speckit.constitution` | Create or update project constitution and principles | `/speckit.constitution` |

## Standard Feature Development

```bash
/speckit.specify "Add user authentication system"
/speckit.clarify     # if needed
/speckit.plan
/speckit.tasks
/speckit.implement
```

## File Structure

```
specs/
├── [number]-[feature-name]/
│   ├── spec.md              # Main specification
│   ├── plan.md              # Implementation plan
│   ├── tasks.md             # Task breakdown
│   └── checklists/
│       └── custom.md        # Custom validation checklist
└── constitution.md          # Project constitution
```

## Best Practices

- Write specifications in business language, not technical implementation details
- Include clear acceptance criteria and success metrics
- Complete the specify → clarify → plan → implement cycle for each feature
- Use `speckit.analyze` regularly to maintain quality
- Maintain project constitution as the single source of truth for principles

## Complete Reference

For exhaustive patterns, examples, and advanced usage see:

**[`references/full-reference.md`](references/full-reference.md)**
