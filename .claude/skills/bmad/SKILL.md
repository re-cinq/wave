---
name: bmad
description: Expert BMAD (Breakthrough Method for Agile AI-Driven Development) implementation including role-based agent specialization, structured workflows, living artifacts, and scale-adaptive processes
---

# BMAD - Breakthrough Method for Agile AI-Driven Development

BMAD provides a comprehensive methodology for AI-enhanced development workflows with role-based agent specialization and structured collaborative development processes.

## Overview

BMAD implements two distinct development paths:

### Quick Path (Bug fixes & small features)
Streamlined workflow for small work items (< 1 week):
- **Quick Specification** - Analyze codebase and create tech spec with stories
- **Development** - Implement individual stories
- **Code Review** - Validate quality and compliance

### Full Path (Products & complex features)
Comprehensive planning workflow for major initiatives (> 1 week):
- **Product Brief** → **PRD** → **Architecture** → **Epic Breakdown** → **Sprint Planning** → **Story Development** → **Code Review**

## Available Commands

### Quick Path
| Command | Purpose | Usage |
|---------|---------|--------|
| `bmad.quick-spec` | Analyze codebase and create tech spec with stories | `/bmad.quick-spec "description"` |
| `bmad.dev-story` | Implement a single story from spec | `/bmad.dev-story [story-id]` |
| `bmad.gh-pr-review` | Validate code quality and compliance | `/bmad.gh-pr-review` |

### Full Path
| Command | Purpose | Usage |
|---------|---------|--------|
| `bmad.product-brief` | Define problem, users, and MVP scope | `/bmad.product-brief "product name"` |
| `bmad.prd` | Create detailed Product Requirements Document | `/bmad.prd` |
| `bmad.architecture` | Create technical architecture document | `/bmad.architecture` |
| `bmad.epics` | Create prioritized epic and story breakdown | `/bmad.epics` |
| `bmad.sprint` | Initialize sprint tracking | `/bmad.sprint` |
| `bmad.story` | Create detailed story specification | `/bmad.story [epic-id]` |

### Utility
| Command | Purpose | Usage |
|---------|---------|--------|
| `bmad.help` | Get contextual guidance and command suggestions | `/bmad.help [question]` |
| `bmad.party` | Multi-agent collaborative session | `/bmad.party [topic]` |

## Role-Based Agent Personas

| Persona | Focus | Key Deliverables |
|---------|-------|-----------------|
| **Mary** (Business Analyst) | Requirements elicitation | User stories, acceptance criteria |
| **Winston** (Architect) | Technical design, patterns | Architecture decisions, design patterns |
| **Amelia** (Developer) | Implementation, code quality | Production code, unit tests |
| **John** (Product Manager) | Prioritization, roadmap | Product roadmaps, feature prioritization |
| **Bob** (Scrum Master) | Process facilitation | Sprint plans, retrospectives |
| **Sally** (UX Designer) | User experience, accessibility | User flows, wireframes |

## Core Workflow Patterns

### Quick Path
```bash
/bmad.quick-spec "Fix login timeout issue"
/bmad.dev-story 1
/bmad.gh-pr-review
```

### Full Path
```bash
/bmad.product-brief "Customer Support Portal"
/bmad.prd && /bmad.architecture && /bmad.epics && /bmad.sprint
/bmad.story 1 && /bmad.dev-story 1.1
/bmad.gh-pr-review
```

## Best Practices

- **Quick Path**: Bug fixes, small features, technical debt, < 1 week effort
- **Full Path**: New products, major features, complex integrations, > 1 week effort
- Use `/bmad.party` for complex decisions requiring multiple perspectives
- Run `/bmad.gh-pr-review` after every implementation

## Complete Reference

For exhaustive patterns, examples, and advanced usage see:

**[`references/full-reference.md`](references/full-reference.md)**
