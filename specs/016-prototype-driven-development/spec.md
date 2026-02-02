# Feature Specification: Prototype-Driven Development Pipelines

**Feature Branch**: `016-prototype-driven-development`
**Created**: 2026-02-02
**Status**: Draft
**Input**: User description: "We want to create new pipelines, especially for a prototype driven development, or spec or dummy: the idea is to have first a solid specification partially done with speckit, build the docs, then build the dummy and last we would start implementing; works best for complete grass root projects"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Initialize New Greenfield Project with Spec Phase (Priority: P1)

A developer starting a brand new project wants to use Wave's prototype-driven development approach. They run a command that creates a new pipeline following the spec-docs-dummy-implement workflow. The system scaffolds the initial specification phase, integrating with speckit to capture requirements before any code is written.

**Why this priority**: This is the core entry point for the entire workflow. Without the ability to initialize and start the specification phase, no other functionality can be used.

**Independent Test**: Can be fully tested by initializing a new project and verifying the spec phase completes with a valid specification artifact.

**Acceptance Scenarios**:

1. **Given** no existing project, **When** user runs the prototype pipeline initialization command, **Then** the system creates a spec phase that invokes speckit to generate initial requirements.
2. **Given** an initialized spec phase, **When** the specification is complete, **Then** the system marks the spec phase as done and prepares handoff artifacts for the docs phase.
3. **Given** a partially completed spec, **When** user wants to refine requirements, **Then** the system allows iterative refinement of the specification before marking complete.

---

### User Story 2 - Generate Documentation from Specification (Priority: P2)

After completing the specification phase, the developer wants the system to automatically generate documentation that describes what will be built. This documentation serves as a reference for stakeholders and guides the subsequent dummy implementation.

**Why this priority**: Documentation bridges the gap between abstract requirements and concrete implementation. It's essential before building the dummy to ensure alignment.

**Independent Test**: Can be tested by providing a completed spec and verifying documentation artifacts are generated that accurately reflect the specification.

**Acceptance Scenarios**:

1. **Given** a completed specification, **When** the docs phase runs, **Then** the system generates human-readable documentation describing the planned feature.
2. **Given** generated documentation, **When** a stakeholder reviews it, **Then** they can understand what the feature will do without technical implementation details.
3. **Given** incomplete or missing specification artifacts, **When** attempting to run docs phase, **Then** the system reports the missing prerequisites and prevents execution.

---

### User Story 3 - Build Dummy Implementation (Priority: P3)

The developer wants to create a working "dummy" or prototype that demonstrates the feature's interfaces and user flows without full business logic. This allows early validation and feedback before investing in complete implementation.

**Why this priority**: The dummy phase validates the design and interfaces. It must come after documentation but before full implementation to catch design issues early.

**Independent Test**: Can be tested by running the dummy phase after docs completion and verifying a runnable prototype is produced that matches the documented interfaces.

**Acceptance Scenarios**:

1. **Given** completed documentation, **When** the dummy phase runs, **Then** the system generates a working prototype with stub implementations.
2. **Given** a generated dummy, **When** a user interacts with it, **Then** they experience the intended user flow with placeholder responses or data.
3. **Given** a dummy implementation, **When** stakeholders review it, **Then** they can provide feedback on the interface and flow before full implementation begins.

---

### User Story 4 - Transition to Full Implementation (Priority: P4)

After the dummy is validated, the developer wants to transition to full implementation. The system should carry forward all artifacts (spec, docs, dummy code) and provide a clear starting point for real implementation work.

**Why this priority**: This is the final phase that produces the actual deliverable. It depends on all prior phases being complete.

**Independent Test**: Can be tested by completing all prior phases and verifying the implementation phase can access all artifacts and provides guidance for developers.

**Acceptance Scenarios**:

1. **Given** a validated dummy, **When** the implement phase runs, **Then** the system provides all prior artifacts as context for the implementing persona.
2. **Given** implementation phase artifacts, **When** a developer begins coding, **Then** they have clear requirements, documentation, and interface contracts to follow.
3. **Given** partial implementation, **When** requirements change, **Then** the system supports re-running earlier phases to update artifacts.

---

### Edge Cases

- What happens when the user tries to skip a phase (e.g., jump from spec to dummy)?
  - System validates phase dependencies and blocks execution with a clear error message.
- How does the system handle spec changes after docs are generated?
  - System detects stale artifacts and prompts user to re-run downstream phases.
- What if the dummy phase fails due to external dependencies?
  - System provides clear error messages and allows retry without losing progress.
- How does the system handle concurrent runs of the same pipeline?
  - System uses workspace isolation to prevent conflicts.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST provide a "prototype" pipeline type that orchestrates spec, docs, dummy, and implement phases in sequence.
- **FR-002**: System MUST integrate with speckit for the specification phase to capture requirements.
- **FR-003**: System MUST validate that each phase's prerequisites are met before execution.
- **FR-004**: System MUST generate handoff artifacts between phases containing relevant context and outputs.
- **FR-005**: System MUST allow users to re-run individual phases to update artifacts.
- **FR-006**: System MUST support skipping previously completed phases when resuming a pipeline.
- **FR-007**: System MUST provide clear progress indicators showing current phase and overall pipeline status.
- **FR-008**: System MUST isolate each pipeline run's workspace to prevent artifact conflicts.
- **FR-009**: System MUST generate documentation that is human-readable and technology-agnostic.
- **FR-010**: System MUST create a dummy implementation that demonstrates interfaces without full business logic.
- **FR-011**: System MUST carry forward all artifacts to the implementation phase as context.
- **FR-012**: System MUST detect stale artifacts when upstream phases are re-run and notify users.

### Key Entities

- **Pipeline Definition**: Represents the prototype pipeline configuration including phase sequence, persona assignments, and contract definitions.
- **Phase**: A discrete stage in the pipeline (spec, docs, dummy, implement) with its own inputs, outputs, and completion criteria.
- **Artifact**: Output from a phase that serves as input to subsequent phases (specification document, documentation files, dummy code).
- **Handoff Contract**: Defines what artifacts must be present and valid for a phase transition to succeed.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Users can initialize and complete a prototype pipeline for a new project within a single session.
- **SC-002**: Each phase produces artifacts that can be independently reviewed before proceeding.
- **SC-003**: Documentation generated is understandable by non-technical stakeholders (validated by readability).
- **SC-004**: Dummy implementations are runnable and demonstrate the intended user experience.
- **SC-005**: 90% of users can complete the full spec-to-implement workflow without external documentation.
- **SC-006**: Phase failures provide actionable error messages that guide resolution.
- **SC-007**: Re-running a phase updates only the affected downstream artifacts.

## Assumptions

- Users have speckit available and configured in their environment.
- The target use case is greenfield (new) projects without existing codebase constraints.
- Documentation generation uses personas with appropriate writing capabilities.
- Dummy implementations are language-agnostic stubs that can be adapted to any target technology.
- Users understand the concept of phased development and the value of early prototyping.
