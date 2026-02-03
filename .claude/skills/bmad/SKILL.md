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
- **Product Brief** - Define problem, users, and MVP scope
- **PRD** - Create detailed Product Requirements Document
- **Architecture** - Technical design and decisions
- **Epic Breakdown** - Create prioritized epic and story structure
- **Sprint Planning** - Initialize sprint tracking
- **Story Development** - Implement individual stories with detailed specs
- **Code Review** - Validate quality and compliance

## Available Commands

### Quick Path Commands

| Command | Purpose | Usage |
|---------|---------|--------|
| `bmad.quick-spec` | Analyze codebase and create tech spec with stories | `/bmad.quick-spec "description"` |
| `bmad.dev-story` | Implement a single story from spec | `/bmad.dev-story [story-id]` |
| `bmad.code-review` | Validate code quality and compliance | `/bmad.code-review` |

### Full Path Commands

| Command | Purpose | Usage |
|---------|---------|--------|
| `bmad.product-brief` | Define problem, users, and MVP scope | `/bmad.product-brief "product name"` |
| `bmad.prd` | Create detailed Product Requirements Document | `/bmad.prd` |
| `bmad.architecture` | Create technical architecture document | `/bmad.architecture` |
| `bmad.epics` | Create prioritized epic and story breakdown | `/bmad.epics` |
| `bmad.sprint` | Initialize sprint tracking | `/bmad.sprint` |
| `bmad.story` | Create detailed story specification | `/bmad.story [epic-id]` |

### Utility Commands

| Command | Purpose | Usage |
|---------|---------|--------|
| `bmad.help` | Get contextual guidance and command suggestions | `/bmad.help [question]` |
| `bmad.party` | Multi-agent collaborative session | `/bmad.party [topic]` |

## Role-Based Agent Personas

BMAD leverages specialized AI agent personas:

### Mary (Business Analyst)
- **Focus**: Requirements elicitation, stakeholder analysis
- **Expertise**: User research, business process modeling, requirement validation
- **Deliverables**: User stories, acceptance criteria, business rules

### Winston (Architect)
- **Focus**: Technical design, patterns, scalability
- **Expertise**: System architecture, technology selection, integration planning
- **Deliverables**: Architecture decisions, technical specifications, design patterns

### Amelia (Developer)
- **Focus**: Implementation, code quality, testing
- **Expertise**: Coding, refactoring, automated testing, performance optimization
- **Deliverables**: Production code, unit tests, technical documentation

### John (Product Manager)
- **Focus**: Prioritization, roadmap, metrics
- **Expertise**: Product strategy, market analysis, feature prioritization
- **Deliverables**: Product roadmaps, feature prioritization, success metrics

### Bob (Scrum Master)
- **Focus**: Process facilitation, sprint management
- **Expertise**: Agile facilitation, team coordination, impediment removal
- **Deliverables**: Sprint plans, retrospective insights, process improvements

### Sally (UX Designer)
- **Focus**: User experience, accessibility, user flows
- **Expertise**: User interface design, usability testing, accessibility compliance
- **Deliverables**: User flows, wireframes, accessibility guidelines

## Workflow Patterns

### Quick Path Example
```bash
# 1. Analyze and create specification
/bmad.quick-spec "Fix login timeout issue"

# 2. Implement story
/bmad.dev-story 1

# 3. Review and validate
/bmad.code-review
```

### Full Path Example
```bash
# 1. Define product vision
/bmad.product-brief "Customer Support Portal"

# 2. Detailed requirements
/bmad.prd

# 3. Technical design
/bmad.architecture

# 4. Break down into epics and stories
/bmad.epics

# 5. Sprint planning
/bmad.sprint

# 6. Detailed story development
/bmad.story 1

# 7. Implementation
/bmad.dev-story 1.1

# 8. Quality validation
/bmad.code-review
```

### Collaborative Sessions
```bash
# Multi-agent discussion on architecture decisions
/bmad.party "microservices vs monolith"

# Product planning session
/bmad.party "Q2 roadmap planning"

# Technical debt discussion
/bmad.party "refactoring strategy"
```

## File Structure

BMAD commands create and maintain structured artifacts:

```
.bmad/
├── products/
│   └── [slug]/
│       ├── docs/
│       │   ├── product-brief.md
│       │   ├── prd.md
│       │   └── architecture.md
│       ├── epics/
│       │   └── [id].md
│       ├── stories/
│       │   └── [epic-id].[story-id].md
│       └── sprints/
│           └── [sprint-number]/
└── specs/
    └── [id]/
        ├── quick-spec.md
        ├── stories/
        │   └── [story-id].md
        └── implementation/
```

## Living Artifacts and Governance

### Living Documentation
BMAD artifacts are designed to evolve with the project:
- **Version Control Integration** - All artifacts tracked in git
- **Cross-Reference Validation** - Automatic consistency checking between artifacts
- **Impact Analysis** - Understanding how changes affect related artifacts
- **Quality Gates** - Automated validation of artifact completeness and quality

### Quality Governance
- **Role-Based Reviews** - Each persona validates their domain expertise
- **Automated Compliance Checking** - Built-in validation rules
- **Technical Debt Tracking** - Systematic identification and management
- **Performance Metrics** - Continuous monitoring of development effectiveness

## Scale-Adaptive Processes

BMAD adapts to team size and project complexity:

### Small Teams (2-3 people)
- Lightweight processes
- Minimal documentation overhead
- Direct communication
- Continuous deployment

### Medium Teams (4-8 people)
- Structured workflows
- Standard documentation
- Regular ceremonies
- Scheduled releases

### Large Teams (8+ people)
- Comprehensive processes
- Detailed documentation
- Full ceremony implementation
- Controlled releases with sub-teams

## Best Practices

### Choosing the Right Path
- **Quick Path**: Bug fixes, small features, technical debt, < 1 week effort
- **Full Path**: New products, major features, complex integrations, > 1 week effort

### Agent Collaboration
- Leverage each persona's expertise in their domain
- Use `/bmad.party` for complex decisions requiring multiple perspectives
- Maintain clear handoffs between workflow phases
- Document decisions and rationale in artifacts

### Quality Assurance
- Run `/bmad.code-review` after every implementation
- Validate artifacts cross-reference correctly
- Maintain living documentation practices
- Use quality gates to prevent technical debt accumulation

### Process Improvement
- Regular retrospectives using agent feedback
- Adapt workflows based on team size and project complexity
- Monitor and optimize development velocity
- Maintain balance between process and productivity

## Integration Points

BMAD integrates with:
- **Version Control** - Git workflow automation
- **Project Management** - Issue tracking and sprint planning
- **CI/CD Pipelines** - Quality gate automation
- **Documentation Systems** - Living documentation maintenance
- **Monitoring Tools** - Performance and quality metrics

This comprehensive BMAD skill provides the framework for AI-enhanced development that scales with team size and project complexity while maintaining high quality and effective collaboration.