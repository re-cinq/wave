# AGENTS.md - Development Guidelines for Wave Repository

This file contains guidelines for agentic coding agents working in the Wave repository. Wave is a SpeckIt-enabled project management and development workflow system.

## Repository Overview

This is a **SpeckIt methodology repository** that provides:
- Feature specification templates and workflows
- Development planning and task management tools  
- Agent skill definitions for Claude and OpenCode agents
- Constitutional governance for feature development

The repository follows a structured approach where every feature is documented, planned, and implemented through standardized templates.

## Development Environment Setup

### Prerequisites
- Bash shell environment
- Git (optional, repository supports non-git workflows)
- Standard Unix tools (find, grep, etc.)

### Initial Setup
```bash
# Run prerequisite checks
./.specify/scripts/bash/check-prerequisites.sh

# Create new feature
./.specify/scripts/bash/create-new-feature.sh "feature-name"
```

## Project Structure

```
Wave/
├── .claude/                    # Claude agent skills and configurations
│   └── skills/                 # Skill definition files for Claude AI
├── .specify/                   # Core SpeckIt methodology and templates
│   ├── memory/                 # Runtime state and constitution
│   ├── scripts/                # Automation scripts
│   └── templates/              # Feature development templates
└── .opencode/                  # OpenCode agent configurations
    └── skills/                 # Skill definition files for OpenCode AI
```

## Development Commands

### Feature Management
```bash
# Create new feature with proper branch naming (###-feature-name)
./.specify/scripts/bash/create-new-feature.sh "feature-name"

# Update agent context files
./.specify/scripts/bash/update-agent-context.sh

# Setup implementation plan
./.specify/scripts/bash/setup-plan.sh
```

### Testing and Validation
```bash
# Check prerequisites and environment
./.specify/scripts/bash/check-prerequisites.sh
```

## Code Style Guidelines

### Bash Scripts
- Use `#!/usr/bin/env bash` shebang
- Follow the existing patterns in `.specify/scripts/bash/common.sh`
- Use functions for reusable logic
- Include proper error handling and validation
- Use `set -euo pipefail` where appropriate
- Follow the established naming conventions:
  - Functions: `snake_case` with descriptive names
  - Variables: `UPPER_SNAKE_CASE` for constants, `lower_snake_case` for locals
  - Private functions: prefix with `_`

### Markdown Documentation
- Use GitHub-flavored markdown
- Include frontmatter with description for skill files
- Use consistent heading hierarchy (##, ###, ####)
- Wrap lines at ~100 characters for readability
- Use code blocks with language specification
- Include HTML comments for implementation notes and TODOs

### Template Files
- Use `[PLACEHOLDER]` syntax for replaceable tokens
- Include detailed comments explaining each section
- Use consistent indentation (2 spaces for markdown, 4 for code blocks)
- Maintain backward compatibility when updating templates
- Include version information in headers

## Feature Development Workflow

### 1. Specification Phase
- Use `.specify/templates/spec-template.md` for feature specifications
- Include mandatory sections: User Scenarios, Requirements, Success Criteria
- Prioritize user stories (P1, P2, P3)
- Ensure each story is independently testable

### 2. Planning Phase  
- Use `.specify/templates/plan-template.md` for implementation plans
- Include technical context and constitution checks
- Define project structure and complexity tracking
- Reference feature specifications directly

### 3. Task Generation
- Use `.specify/templates/tasks-template.md` for task breakdowns
- Follow constitutional governance requirements
- Ensure all tasks are atomic and testable

## File Naming Conventions

### Feature Branches
- Format: `###-feature-name` (e.g., `001-user-authentication`)
- Three-digit prefix with hyphen separator
- Descriptive, lowercase feature names with hyphens

### Skill Files
- Format: `speckit.{action}.md` (e.g., `speckit.implement.md`)
- Use lowercase with single dot separator
- Align with command names

### Template Files
- Format: `{type}-template.md` (e.g., `spec-template.md`)
- Descriptive, lowercase names with hyphens

## Error Handling

### Bash Scripts
- Use explicit error codes (0 for success, 1+ for errors)
- Include descriptive error messages to stderr
- Use `||` operators for fallback handling
- Validate inputs before processing

### Templates
- Use `NEEDS CLARIFICATION` placeholders for ambiguous requirements
- Include validation sections in specifications
- Provide clear acceptance criteria

## Agent-Specific Guidelines

### For Claude Agents
- Skills located in `.claude/skills/`
- Follow frontmatter format with description
- Use constitution-driven governance checks
- Leverage template patterns for consistency

### For OpenCode Agents  
- Skills located in `.opencode/skills/`
- Mirror Claude skill structure
- Use package.json for dependency management
- Follow JavaScript/TypeScript conventions where applicable

## Constitutional Governance

All feature development must align with the constitution in `.specify/memory/constitution.md`:
- Respect versioning requirements (semantic versioning)
- Follow amendment procedures
- Conduct compliance reviews
- Maintain consistency across templates

## Testing Strategy

### Template Validation
- Verify all placeholder tokens are replaceable
- Test template generation with sample data
- Validate markdown syntax and structure
- Ensure cross-template consistency

### Script Testing  
- Test both git and non-git environments
- Validate branch naming conventions
- Test prerequisite checking functionality
- Verify file creation and permissions

## Documentation Standards

### Inline Comments
- Use HTML comments for implementation notes in markdown
- Include ACTION REQUIRED sections for manual steps
- Provide clear rationale for architectural decisions

### README Files
- Include quick start instructions
- Document prerequisite requirements
- Provide troubleshooting guidance
- Include contribution guidelines

## Version Management

- Use semantic versioning for constitution updates
- Increment MAJOR for breaking changes to governance
- Increment MINOR for new principles or sections  
- Increment PATCH for clarifications and fixes
- Document all changes in sync impact reports

## Security Considerations

- Validate all user inputs in scripts
- Use relative paths to avoid directory traversal
- Sanitize environment variables
- Follow principle of least privilege in automation

## Performance Guidelines

- Optimize script execution for large repositories
- Use efficient file operations (find, grep sparingly)
- Cache expensive operations where appropriate
- Minimize external dependencies in scripts

This AGENTS.md file should be updated whenever:
- New script commands are added
- Template structures change significantly
- Constitutional amendments are made
- New agent capabilities are introduced