---
name: opsx
description: OpenSpec workflow management system for creating, planning, implementing, and archiving specification-driven development changes with comprehensive lifecycle management
---

# OpenSpec Workflow (OpsX)

OpenSpec Workflow (OpsX) provides a comprehensive lifecycle management system for specification-driven development changes, from initial proposal through implementation and archival.

## Overview

OpsX implements a structured approach to managing development changes through the OpenSpec methodology:

1. **Onboard** - Get started with OpenSpec workflow
2. **New** - Create new OpenSpec change proposals
3. **Fast-Forward** - Generate complete planning documentation
4. **Apply** - Execute implementation tasks
5. **Archive** - Complete and archive finished changes

## Available Commands

### Lifecycle Management Commands

| Command | Purpose | Usage |
|---------|---------|--------|
| `opsx.onboard` | Get started with OpenSpec workflow | `/opsx.onboard` |
| `opsx.new` | Create a new OpenSpec change proposal | `/opsx.new "change description"` |
| `opsx.ff` | Fast-forward to generate all planning docs | `/opsx.ff` |
| `opsx.apply` | Execute OpenSpec implementation tasks | `/opsx.apply` |
| `opsx.archive` | Archive a completed OpenSpec change | `/opsx.archive` |

## Workflow Patterns

### Standard Change Lifecycle
```bash
# 1. Initialize OpenSpec workflow (first time)
/opsx.onboard

# 2. Create new change proposal
/opsx.new "Add user authentication system"

# 3. Generate complete planning documentation
/opsx.ff

# 4. Execute implementation
/opsx.apply

# 5. Archive completed change
/opsx.archive
```

### Quick Implementation Pattern
```bash
# For urgent changes or small fixes
/opsx.new "Fix critical security vulnerability"
/opsx.apply  # Skip detailed planning for urgent changes
/opsx.archive
```

### Comprehensive Planning Pattern
```bash
# For complex changes requiring thorough planning
/opsx.new "Migrate to microservices architecture"
/opsx.ff     # Generate comprehensive planning docs
# Review and refine plans as needed
/opsx.apply  # Execute with full planning context
/opsx.archive
```

## OpenSpec Change Structure

OpsX manages changes with a standardized structure:

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
├── templates/              # OpenSpec templates
├── config/                # Workflow configuration
└── archive/               # Completed changes
    └── [year]/
        └── [change-id]/   # Archived change artifacts
```

## Change Lifecycle Phases

### 1. Proposal Phase (`opsx.new`)
- **Purpose**: Capture initial change request and basic scope
- **Artifacts**: `proposal.md` with change description, rationale, and initial scope
- **Next Steps**: Fast-forward planning or direct implementation

### 2. Planning Phase (`opsx.ff`)
- **Purpose**: Generate comprehensive planning documentation
- **Artifacts**:
  - `specification.md` - Detailed requirements and acceptance criteria
  - `plan.md` - Implementation strategy and design decisions
  - `tasks.md` - Ordered task breakdown with dependencies
- **Next Steps**: Review plans, then execute implementation

### 3. Implementation Phase (`opsx.apply`)
- **Purpose**: Execute planned tasks and track progress
- **Artifacts**: Implementation code, tests, documentation in `implementation/`
- **Next Steps**: Complete implementation and archive change

### 4. Archive Phase (`opsx.archive`)
- **Purpose**: Complete the change lifecycle and preserve artifacts
- **Artifacts**: `archive.md` with completion summary and lessons learned
- **Outcome**: Change moved to archive for future reference

## Integration Points

### Version Control Integration
- **Automatic Branching** - Creates feature branches for each change
- **Commit Automation** - Structures commits according to OpenSpec conventions
- **Pull Request Generation** - Automated PR creation with proper documentation
- **Merge Management** - Handles branch cleanup after successful completion

### Project Management Integration
- **Issue Tracking** - Links changes to project management tools
- **Sprint Planning** - Integrates task breakdown with sprint planning
- **Progress Reporting** - Automated status updates and progress tracking
- **Dependency Management** - Tracks cross-change dependencies

### Quality Assurance Integration
- **Validation Gates** - Automated quality checks at each phase
- **Documentation Standards** - Ensures consistent documentation quality
- **Review Workflows** - Structured peer review processes
- **Compliance Tracking** - Monitors adherence to organizational standards

## Configuration and Customization

### Workflow Customization
OpsX supports organization-specific customization:

```yaml
# .openspec/config/workflow.yml
phases:
  planning:
    required: true
    templates: ["specification", "plan", "tasks"]
  implementation:
    validation_gates: ["tests", "documentation", "security"]
  archive:
    retention_policy: "permanent"
    backup_location: "archive/"

integrations:
  git:
    branch_prefix: "openspec/"
    commit_conventions: "conventional"
  project_management:
    provider: "github"
    link_issues: true
```

### Template Customization
Organizations can customize templates for their specific needs:
- **Proposal templates** - Standardized change request format
- **Specification templates** - Requirements documentation format
- **Plan templates** - Implementation planning format
- **Archive templates** - Completion documentation format

## Best Practices

### Change Scoping
- **Single Responsibility** - Each change should address one specific concern
- **Clear Boundaries** - Well-defined scope with explicit inclusions/exclusions
- **Dependency Mapping** - Identify and document relationships with other changes
- **Risk Assessment** - Evaluate potential impacts and mitigation strategies

### Planning Quality
- **Stakeholder Input** - Gather requirements from all affected parties
- **Technical Validation** - Ensure feasibility of proposed solutions
- **Timeline Realism** - Accurate estimation of effort and dependencies
- **Quality Gates** - Define clear acceptance criteria and validation steps

### Implementation Management
- **Task Granularity** - Break work into manageable, trackable units
- **Progress Tracking** - Regular updates on task completion and blockers
- **Quality Assurance** - Continuous validation against acceptance criteria
- **Documentation Maintenance** - Keep artifacts current throughout implementation

### Archive Management
- **Lessons Learned** - Document insights for future reference
- **Artifact Preservation** - Maintain complete change history
- **Knowledge Transfer** - Share learnings with team and organization
- **Process Improvement** - Use archive data to refine workflow

## Error Handling and Recovery

### Common Issues and Solutions

| Issue | Symptoms | Resolution |
|-------|----------|------------|
| Incomplete planning | Missing artifacts, unclear requirements | Run `/opsx.ff` to generate missing documentation |
| Stalled implementation | Long-running tasks, blocked progress | Review task breakdown, identify blockers, replan if needed |
| Quality gate failures | Failed validation, incomplete criteria | Address specific failures, update implementation |
| Merge conflicts | Git conflicts, integration issues | Resolve conflicts, validate integration, continue |

### Recovery Procedures
- **Rollback Capability** - Ability to revert changes if issues arise
- **Checkpoint Management** - Regular save points during long implementations
- **Alternative Paths** - Fallback strategies for blocked implementations
- **Emergency Procedures** - Fast-track critical fixes when needed

## Metrics and Analytics

OpsX tracks key metrics for process improvement:

### Process Metrics
- **Cycle Time** - Time from proposal to archive
- **Planning Accuracy** - Actual vs. estimated effort
- **Quality Metrics** - Defect rates, rework frequency
- **Success Rate** - Percentage of changes successfully completed

### Organizational Metrics
- **Change Velocity** - Number of changes completed per time period
- **Resource Utilization** - Efficiency of team capacity usage
- **Knowledge Retention** - Effectiveness of documentation and archival
- **Process Adoption** - Team adherence to OpenSpec methodology

This comprehensive OpsX skill provides a complete lifecycle management system for specification-driven development that ensures quality, traceability, and continuous improvement.