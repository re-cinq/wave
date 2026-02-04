# Research: YAML-First Documentation Strategy

**Feature**: 001-yaml-first-docs
**Date**: 2026-02-03
**Purpose**: Research findings to support documentation paradigm shift

## Documentation Architecture Patterns

### Decision: Static Site Generator Approach
**Rationale**: Maintain simplicity and performance while enabling modern documentation features like search, navigation, and responsive design.

**Alternatives considered**:
- **Custom documentation site**: Rejected due to maintenance overhead
- **Wiki-based approach**: Rejected due to lack of version control integration
- **PDF/GitBook**: Rejected due to poor discoverability and search

**Research findings**:
- Current docs use basic markdown structure
- VitePress or similar would enable better navigation and user experience
- Static hosting aligns with Wave's "single binary" philosophy

---

## Content Positioning Strategy

### Decision: Infrastructure-as-Code Paradigm Focus
**Rationale**: Developers already understand IaC concepts (Kubernetes, Docker Compose, Terraform). Drawing these parallels makes Wave immediately comprehensible to the target audience.

**Alternatives considered**:
- **AI-first positioning**: Rejected because it attracts AI enthusiasts rather than infrastructure engineers
- **Persona-focused marketing**: Current approach, being moved away from per user feedback
- **DevOps tool positioning**: Too generic, doesn't capture the paradigm shift

**Research findings**:
- Infrastructure engineers are the primary user persona
- YAML familiarity is extremely high among target users
- "Infrastructure as Code" is a proven, successful paradigm that resonates

---

## Content Hierarchy Strategy

### Decision: Paradigm → Workflows → Concepts → Reference
**Rationale**: Lead with the conceptual breakthrough, then show practical application, then provide supporting details.

**Alternatives considered**:
- **Tutorial-first approach**: Rejected because users need to understand "why" before "how"
- **Reference-first approach**: Current structure, poor for first-time discovery
- **Use-case-first approach**: Rejected because it's too specific and fragmented

**Research findings**:
- Users currently struggle with understanding Wave's value proposition
- Persona-focused organization creates confusion about core purpose
- Progressive disclosure from concept to implementation follows proven documentation patterns

---

## Technical Examples Strategy

### Decision: Complete YAML Files Before Component Explanation
**Rationale**: Show the end result first, then break down how it works. This matches how developers learn new frameworks.

**Alternatives considered**:
- **Component-first explanation**: Current approach, creates cognitive overload
- **CLI-command focused**: Rejected because it emphasizes usage over workflow design
- **Mixed approach**: Rejected for inconsistency

**Research findings**:
- Developers scan for patterns they recognize first
- Complete examples provide context for component explanations
- YAML structure is self-documenting when presented clearly

---

## Community Ecosystem Positioning

### Decision: Workflow Marketplace Concept
**Rationale**: Position Wave workflows as shareable, reusable infrastructure components similar to Helm charts or Docker images.

**Alternatives considered**:
- **Plugin ecosystem**: Rejected because Wave workflows aren't plugins
- **Template library**: Too generic, doesn't capture the innovation
- **Recipe collection**: Too informal for enterprise adoption

**Research findings**:
- Successful developer tools have thriving ecosystem stories
- Infrastructure engineers understand marketplace/registry concepts
- Shareability is a key differentiator vs other AI tools

---

## Success Metrics Research

### Decision: Conversion-Focused Metrics
**Rationale**: Focus on whether users understand and adopt Wave, rather than traditional documentation metrics.

**Key findings**:
- Time-to-comprehension is critical for developer tools
- Workflow creation is the primary conversion goal
- Bounce rate indicates paradigm confusion in current docs

**Measurement approach**:
- Landing page comprehension testing
- First workflow creation tracking
- Support ticket categorization analysis