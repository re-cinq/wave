# Feature Specification: YAML-First Documentation Paradigm

**Feature Branch**: `001-yaml-first-docs`
**Created**: 2026-02-03
**Updated**: 2026-02-03
**Status**: In Progress
**Input**: User description: "Refresh Wave documentation to emphasize YAML-first AI-as-code paradigm with deliverables and contracts as core value proposition, moving away from persona-focused marketing to focus on reproducible, shareable, version-controlled AI workflows that developers can treat like infrastructure code"

**Design Reference**: [ngrok documentation](https://ngrok.com/docs) - enterprise-grade developer documentation patterns

## User Scenarios & Testing _(mandatory)_

### User Story 1 - First Pipeline in 60 Seconds (Priority: P1)

A developer discovers Wave and runs their first pipeline within 60 seconds of landing on documentation.

**Why this priority**: ngrok's getting-started achieves immediate value. Wave must match this. Developers evaluate tools by how fast they can see results.

**Independent Test**: Time a new developer from landing to first successful `wave run` command.

**Acceptance Scenarios**:

1. **Given** a developer lands on Wave docs, **When** they follow the quickstart, **Then** they execute their first pipeline in under 60 seconds
2. **Given** a developer has no existing Wave config, **When** they run `wave init && wave run code-review "test"`, **Then** they see immediate results
3. **Given** a developer encounters a blocker, **When** they look for help, **Then** escape routes are visible ("Don't have X? Try this instead")

---

### User Story 2 - Task-Based Discovery (Priority: P2)

A developer with a specific task (code review, docs generation, security audit) finds the relevant guide immediately through task-oriented navigation.

**Why this priority**: ngrok organizes by scenario ("Ingress to IoT Devices"), not features. Users arrive with tasks, not to learn features.

**Independent Test**: Give 5 developers common tasks, measure time to find relevant documentation.

**Acceptance Scenarios**:

1. **Given** a developer wants to automate code reviews, **When** they scan navigation, **Then** they find "Automate Code Review" within 5 seconds
2. **Given** a developer needs security auditing, **When** they browse use-cases, **Then** they find a complete pipeline example immediately
3. **Given** a developer has a custom task, **When** they reach the concepts section, **Then** they understand how to compose their own pipeline

---

### User Story 3 - Progressive Complexity (Priority: P3)

A developer can start simple and progressively add complexity (contracts, dependencies, personas) as their needs grow.

**Why this priority**: ngrok shows simple examples first, then advanced. Documentation should not front-load complexity.

**Independent Test**: Track which documentation paths users follow - simple → complex should be the natural flow.

**Acceptance Scenarios**:

1. **Given** a developer sees their first pipeline example, **When** they read it, **Then** it contains maximum 10 lines of YAML
2. **Given** a developer needs output validation, **When** they look for contracts, **Then** examples show adding one line to existing pipeline
3. **Given** a developer needs multi-step workflows, **When** they find the guide, **Then** progression from simple → complex is explicit

---

### User Story 4 - Team Adoption Path (Priority: P4)

A team lead can follow a clear adoption path from individual use to team-wide standardization.

**Why this priority**: Enterprise adoption requires clear progression. ngrok provides this via guides section.

**Independent Test**: Team lead can articulate adoption steps after reading team-adoption guide.

**Acceptance Scenarios**:

1. **Given** a team lead evaluates Wave, **When** they find the team adoption guide, **Then** steps are clear and actionable
2. **Given** a team uses Wave individually, **When** they want to standardize, **Then** documentation shows git-based sharing patterns
3. **Given** an organization needs security controls, **When** they review enterprise patterns, **Then** permission and audit controls are documented

### Edge Cases

- What happens when developers don't have Claude CLI installed?
- How does documentation handle users without existing codebases to analyze?
- What if quickstart examples fail due to API key issues?

## Requirements _(mandatory)_

### Functional Requirements

**Landing & First Impression**
- **FR-001**: Landing page MUST have one clear CTA: "Get started in 60 seconds"
- **FR-002**: "What is Wave" MUST be explainable in one sentence with one diagram
- **FR-003**: Hero section MUST NOT mention personas - focus on "pipelines for AI"

**Content Structure (ngrok patterns)**
- **FR-004**: Documentation MUST use task-oriented navigation (not feature-oriented)
- **FR-005**: Every concept page MUST follow "1-2 sentences + code block" pattern
- **FR-006**: Examples MUST progress from simple (5-10 lines) to complex
- **FR-007**: Each page MUST have clear "Next Steps" with 2-3 pathways

**Quickstart Requirements**
- **FR-008**: Quickstart MUST work with copy-paste commands (no manual file creation)
- **FR-009**: Quickstart MUST include "escape routes" for common blockers
- **FR-010**: First successful output MUST be visible within 60 seconds

**Use-Case Documentation**
- **FR-011**: Use-cases MUST be organized by task: code-review, security-audit, docs-generation, test-generation
- **FR-012**: Each use-case MUST include complete, runnable pipeline example
- **FR-013**: Use-cases MUST show expected output

**Reference Documentation**
- **FR-014**: CLI reference MUST show command + expected output pairs
- **FR-015**: Pipeline schema MUST be scannable with clear required vs optional fields
- **FR-016**: Manifest reference MUST include copy-paste examples for each section

**Progressive Disclosure**
- **FR-017**: Simple pipelines shown before contracts
- **FR-018**: Single-step pipelines before multi-step
- **FR-019**: Default personas before custom personas
- **FR-020**: Git sharing before enterprise patterns

### Key Entities

- **Pipeline**: YAML definition of AI workflow steps - the primary user interface
- **Persona**: Pre-configured AI agent with specific permissions and system prompt
- **Contract**: Output validation (jsonschema, typescript, testsuite, markdownspec)
- **Artifact**: Output file produced by a pipeline step
- **Manifest**: Project configuration file (`wave.yaml`) defining personas and runtime settings

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: New users complete quickstart in under 60 seconds (measured via user testing)
- **SC-002**: Task-based navigation finds relevant content in under 5 seconds (user testing)
- **SC-003**: Documentation pages follow "1-2 sentences + code" pattern (100% of concept pages)
- **SC-004**: Every code example is copy-paste runnable (validation testing)
- **SC-005**: Escape routes present for all common blockers in quickstart
- **SC-006**: Progressive complexity path explicit: simple → contracts → multi-step → team → enterprise

## Documentation Structure _(reference)_

### Proposed Hierarchy (ngrok-inspired)

```
docs/
├── index.md                    # "What is Wave" - one paragraph, one diagram, quickstart link
├── quickstart.md               # 60-second first pipeline (CRITICAL)
│
├── use-cases/                  # Task-oriented (PRIMARY navigation)
│   ├── index.md               # Use-case overview with cards
│   ├── code-review.md         # Complete code review pipeline
│   ├── security-audit.md      # Security analysis pipeline
│   ├── docs-generation.md     # Documentation generation
│   └── test-generation.md     # Test generation pipeline
│
├── concepts/                   # Short explanations + examples (SECONDARY)
│   ├── index.md               # Concept overview
│   ├── pipelines.md           # What pipelines are (1-2 sentences + example)
│   ├── personas.md            # What personas are (supporting concept)
│   ├── contracts.md           # Output validation (progressive from simple)
│   ├── artifacts.md           # Output files and handovers
│   └── execution.md           # How Wave runs pipelines
│
├── reference/                  # Technical specs (TERTIARY)
│   ├── cli.md                 # All commands with examples
│   ├── manifest.md            # wave.yaml complete reference
│   ├── pipeline-schema.md     # Pipeline YAML schema
│   └── contract-types.md      # All contract type references
│
└── guides/                     # Advanced patterns (ADOPTION)
    ├── ci-cd.md               # CI/CD integration
    ├── team-adoption.md       # Team rollout patterns
    └── enterprise.md          # Enterprise patterns
```

### Key Differences from Current Structure

| Current | Proposed | Rationale |
|---------|----------|-----------|
| paradigm/ section | Removed - integrated into index.md | ngrok doesn't have separate "paradigm" section |
| workflows/ | use-cases/ | Task-oriented naming |
| migration/ | guides/ | Clearer purpose |
| Feature-organized nav | Task-organized nav | Users arrive with tasks |
| Dense concept pages | 1-2 sentences + code | ngrok pattern |
| No quickstart | Dedicated quickstart.md | 60-second first success |

## Assumptions

- Claude CLI is installed and API key configured (quickstart will provide escape routes)
- Users have a codebase to analyze (will provide sample project fallback)
- Git is available for version control examples
- Users can read YAML (reasonable for developer audience)

## Out of Scope

- Video tutorials (text-first approach)
- Interactive playground (future consideration)
- Community registry/marketplace (just git-based sharing)
- Multiple language translations
