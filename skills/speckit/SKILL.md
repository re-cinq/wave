---
name: speckit
description: Specification-driven development workflow tools for creating, analyzing, and implementing feature specifications with automated planning and validation
---

# SpecKit - Specification-Driven Development Tools

SpecKit provides a comprehensive workflow for specification-driven development, from initial feature description to implementation planning. It includes tools for specification creation, clarification, planning, validation, and implementation tracking.

## Overview

SpecKit follows a structured workflow:
1. **Specify** - Create feature specifications from natural language
2. **Clarify** - Identify and resolve underspecified areas
3. **Plan** - Generate implementation plans and design artifacts
4. **Validate** - Analyze specifications for consistency and quality
5. **Implement** - Execute implementation plans with task tracking

## Available Commands

### Core Workflow Commands

| Command | Purpose | Usage |
|---------|---------|--------|
| `speckit.specify` | Create or update feature specification | `/speckit.specify "feature description"` |
| `speckit.clarify` | Identify underspecified areas and ask clarification questions | `/speckit.clarify` |
| `speckit.plan` | Generate implementation plans and design artifacts | `/speckit.plan` |
| `speckit.implement` | Execute implementation plan with task tracking | `/speckit.implement` |

### Analysis and Validation Commands

| Command | Purpose | Usage |
|---------|---------|--------|
| `speckit.analyze` | Cross-artifact consistency and quality analysis | `/speckit.analyze` |
| `speckit.tasks` | Generate actionable, dependency-ordered task list | `/speckit.tasks` |
| `speckit.checklist` | Generate custom validation checklist | `/speckit.checklist` |

### Project Setup Commands

| Command | Purpose | Usage |
|---------|---------|--------|
| `speckit.constitution` | Create or update project constitution and principles | `/speckit.constitution` |

## Workflow Patterns

### Standard Feature Development
```bash
# 1. Create specification
/speckit.specify "Add user authentication system"

# 2. Clarify requirements (if needed)
/speckit.clarify

# 3. Generate implementation plan
/speckit.plan

# 4. Create task breakdown
/speckit.tasks

# 5. Execute implementation
/speckit.implement
```

### Quality Assurance Pattern
```bash
# 1. Analyze specification quality
/speckit.analyze

# 2. Generate validation checklist
/speckit.checklist

# 3. Address any issues found
/speckit.clarify  # if needed
```

### Project Initialization
```bash
# 1. Set up project principles
/speckit.constitution

# 2. Create first feature specification
/speckit.specify "initial feature"
```

## File Structure

SpecKit commands work with a structured file system:

```
specs/
├── [number]-[feature-name]/
│   ├── spec.md              # Main specification
│   ├── plan.md              # Implementation plan
│   ├── tasks.md             # Task breakdown
│   └── checklists/
│       ├── requirements.md  # Requirements checklist
│       └── custom.md        # Custom validation checklist
└── constitution.md          # Project constitution
```

## Integration with Development Tools

SpecKit integrates with:
- **Git workflow** - Automatic branch creation and management
- **Issue tracking** - Task generation compatible with project management tools
- **Documentation systems** - Living documentation that stays synchronized
- **Testing frameworks** - Specification-driven test generation
- **CI/CD pipelines** - Automated validation and deployment gates

## Best Practices

### Specification Quality
- Write specifications in business language, not technical implementation details
- Include clear acceptance criteria and success metrics
- Document assumptions and constraints explicitly
- Use examples to clarify complex requirements

### Workflow Management
- Complete the specify → clarify → plan → implement cycle for each feature
- Use speckit.analyze regularly to maintain quality
- Update specifications as requirements evolve
- Maintain project constitution as the single source of truth for principles

### Team Collaboration
- Use clarification questions to align team understanding
- Review specifications before implementation planning
- Track implementation progress through task lists
- Maintain consistent specification format across the team

## Advanced Usage

### Custom Validation Rules
Create custom checklists for specific project needs:
```bash
/speckit.checklist "security review"
/speckit.checklist "performance requirements"
/speckit.checklist "accessibility compliance"
```

### Iterative Refinement
Use the analyze command to continuously improve specification quality:
```bash
/speckit.analyze  # Identify issues
# Fix issues in specifications
/speckit.analyze  # Verify improvements
```

### Cross-Feature Dependencies
Track dependencies between features in the planning phase:
```bash
/speckit.plan     # Generates plan with dependency analysis
/speckit.tasks    # Creates ordered task list respecting dependencies
```

This comprehensive SpecKit skill provides all the tools necessary for effective specification-driven development, ensuring high-quality requirements that lead to successful implementations.