# Feature Specification: Prototype-Driven Development Pipelines

**Feature Branch**: `017-prototype-driven-development`
**Created**: 2026-02-02
**Status**: Draft
**Input**: User description: "We want to create new pipelines for prototype-driven development. The cycles are: (1) Create full runnable documentation with VitePress, (2) Create a fully functional and authentic prototype/mockup that handles input and output properly without business logic, (3) Partially implement specifications (spec generation is covered by other pipelines), (4) Full hands-off PR cycle with Copilot review, Claude responses, and auto-merge."

## Clarifications

### CLI Invocation (Clarified 2026-02-02)

Users invoke the prototype pipeline using the standard Wave run command:

```bash
# Full pipeline execution
wave run --pipeline prototype --input "build a task management CLI tool"

# Resume from specific phase
wave run --pipeline prototype --from-step docs --input "build a task management CLI tool"

# Dry-run to preview execution plan
wave run --pipeline prototype --dry-run --input "build a task management CLI tool"
```

The pipeline YAML is stored at `.wave/pipelines/prototype.yaml` following the established pattern for built-in pipelines.

### Documentation Artifacts (Clarified 2026-02-02)

The docs phase produces a **full runnable documentation site** using VitePress:

| Artifact | Path | Purpose |
|----------|------|---------|
| VitePress site | `output/docs/` | Complete documentation site with dev server |
| README.md | `output/docs/index.md` | User-facing project overview, installation, quick-start |
| API Reference | `output/docs/api/` | API reference: endpoints, data structures, schemas |
| Architecture | `output/docs/architecture.md` | System architecture, component diagrams, data flow |
| User Guide | `output/docs/guide/` | Step-by-step usage guides and tutorials |

**Runnable**: `cd output/docs && npm run dev` starts the documentation site locally.
All documentation is human-readable and technology-agnostic for stakeholder review.

### Dummy Implementation Definition (Clarified 2026-02-02)

A "dummy implementation" is a **fully functional and authentic prototype** that handles input and output properly without the core business logic. It consists of:

- **Interface definitions**: Go interfaces, TypeScript types, protobuf schemas - fully typed
- **Route handlers**: HTTP/CLI handlers that accept real input, validate it, and return properly structured responses
- **UI component shells**: Fully rendered components with real navigation, form handling, and state management
- **Data layer stubs**: In-memory stores that persist during runtime, enabling realistic user flows
- **Test fixtures**: Example inputs/outputs demonstrating expected behavior with validation
- **Error handling**: Proper error responses for invalid inputs (not just happy path)

**Key distinction**: The dummy handles I/O authentically. A CLI accepts real arguments and returns formatted output. An API accepts real requests and returns valid JSON. A UI handles real user interactions. Only the core business logic is stubbed.

The dummy MUST compile/run without errors and handle real I/O - business logic returns realistic mock data.

### Re-runs and Artifact Invalidation (Clarified 2026-02-02)

Wave handles re-runs using existing infrastructure:

- **Re-run from phase**: `wave run --pipeline prototype --from-step <phase>` skips completed phases
- **Automatic regeneration**: Re-running a phase regenerates all downstream artifacts
- **Workspace isolation**: Each run uses `.wave/workspaces/prototype/<step>/` to prevent conflicts
- **State persistence**: SQLite state store tracks phase completion; `wave resume` handles interruptions
- **Stale detection**: Artifacts include timestamps; the pipeline warns if upstream was modified after downstream completed

### Persona Assignments (Clarified 2026-02-02)

| Phase | Persona | Rationale |
|-------|---------|-----------|
| spec | `philosopher` | Specification writing and architecture design |
| docs | `philosopher` | Technical documentation shares the design mindset |
| dummy | `craftsman` | Creates runnable code, even if stubs |
| implement | `craftsman` | Full implementation with tests |
| pr-create | `philosopher` | Generates PR title and summary from artifacts |
| pr-respond | `auditor` | Reviews and responds to Copilot comments |
| pr-fix | `craftsman` | Implements review-suggested changes |
| pr-merge | N/A | Uses `gh` CLI directly, no persona needed |

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Initialize New Greenfield Project with Spec Phase (Priority: P1)

A developer starting a brand new project wants to use Wave's prototype-driven development approach. They run `wave run --pipeline prototype --input "project description"` which executes a 4-phase pipeline. The spec phase uses the `philosopher` persona to generate requirements following the speckit workflow patterns.

**Why this priority**: This is the core entry point for the entire workflow. Without the ability to initialize and start the specification phase, no other functionality can be used.

**Independent Test**: Can be fully tested by running `wave run --pipeline prototype --input "build a todo app" --dry-run` and verifying the execution plan shows all 4 phases with correct persona assignments.

**Acceptance Scenarios**:

1. **Given** a Wave project with the prototype pipeline, **When** user runs `wave run --pipeline prototype --input "build a todo CLI"`, **Then** the spec phase executes with the `philosopher` persona and produces `output/spec.md`.
2. **Given** a completed spec phase, **When** the specification artifact passes contract validation, **Then** the system proceeds to the docs phase automatically.
3. **Given** a partially completed spec, **When** user wants to refine requirements, **Then** they can re-run with `--from-step spec` to regenerate the specification.

---

### User Story 2 - Generate Runnable Documentation Site (Priority: P2)

After the spec phase completes, the docs phase generates a **full VitePress documentation site** that can be run locally. The `philosopher` persona creates the complete documentation structure with index, API reference, architecture overview, and user guides.

**Why this priority**: A runnable documentation site provides immediate stakeholder value - they can browse the planned feature interactively before any code is written.

**Independent Test**: Can be tested by running `wave run --pipeline prototype --from-step docs`, then `cd output/docs && npm run dev` to start the documentation server.

**Acceptance Scenarios**:

1. **Given** a completed specification at `output/spec.md`, **When** the docs phase runs, **Then** the system generates a complete VitePress site at `output/docs/` with `package.json`, `index.md`, and structured content.
2. **Given** generated documentation, **When** user runs `npm run dev` in `output/docs/`, **Then** VitePress starts a local server at `localhost:5173` serving the documentation.
3. **Given** the documentation site, **When** a stakeholder browses it, **Then** they can navigate API reference, architecture diagrams, and user guides interactively.
4. **Given** missing `output/spec.md` artifact, **When** attempting `--from-step docs`, **Then** the contract validation fails with error: "Missing required artifact: spec:specification".

---

### User Story 3 - Build Authentic Functional Prototype (Priority: P3)

The dummy phase uses the `craftsman` persona to create a **fully functional prototype** that handles real input/output with stubbed business logic. The prototype includes: complete interface definitions, route handlers that accept real requests and validate input, UI components with real state management, and in-memory data stores for realistic user flows.

**Why this priority**: An authentic prototype validates the design with real I/O handling. Users can interact with it exactly as they would the final product - only the business logic returns mock data.

**Independent Test**: Can be tested by running `wave run --pipeline prototype --from-step dummy`, then interacting with the prototype using real inputs and verifying proper output formatting and error handling.

**Acceptance Scenarios**:

1. **Given** completed documentation, **When** the dummy phase runs, **Then** the system generates a fully runnable prototype at `output/dummy/` that compiles and starts without errors.
2. **Given** a generated dummy CLI tool, **When** user runs `./dummy create --name "test"`, **Then** the CLI validates the input, stores data in-memory, and returns a properly formatted success response.
3. **Given** a dummy REST API, **When** user POSTs `{"name": "test"}` to `/api/items`, **Then** the API validates the request, stores in-memory, and returns `201 Created` with the created item JSON.
4. **Given** invalid input to the dummy, **When** user submits malformed data, **Then** the prototype returns proper error responses (400, 422) with meaningful error messages - not crashes.
5. **Given** a dummy with in-memory state, **When** user creates items then lists them, **Then** the prototype returns the created items (state persists during the session).

---

### User Story 4 - Transition to Full Implementation (Priority: P4)

The implement phase uses the `craftsman` persona with full read/write permissions. It receives all prior artifacts via `inject_artifacts`: the spec, documentation, and dummy code. The craftsman replaces stub implementations with real business logic while preserving the interfaces. Tests must pass before the phase completes.

**Why this priority**: This is the final phase that produces the actual deliverable. It depends on all prior phases being complete.

**Independent Test**: Can be tested by running `wave run --pipeline prototype --from-step implement` and verifying the craftsman receives injected artifacts and produces working code with passing tests.

**Acceptance Scenarios**:

1. **Given** a validated dummy at `output/dummy/`, **When** the implement phase runs, **Then** the craftsman persona receives `artifacts/spec.md`, `artifacts/README.md`, `artifacts/API.md`, `artifacts/ARCHITECTURE.md`, and `artifacts/dummy/` as context.
2. **Given** the implementation phase completes, **When** the test suite runs, **Then** all tests pass via the `test_suite` handover contract.
3. **Given** requirements change after implementation starts, **When** user runs `wave run --pipeline prototype --from-step spec`, **Then** all downstream artifacts (docs, dummy, implementation) are regenerated.

---

### User Story 5 - Automated PR Cycle with Hands-Off Review (Priority: P5)

After implementation completes, the pipeline continues into a full hands-off PR cycle. The system creates a PR, adds Copilot as a reviewer, waits for review completion, responds to review comments using Claude, implements suggested changes or creates follow-up issues, and finally marks the PR for merge when approved.

**Why this priority**: This extends the pipeline beyond implementation to the full merge workflow. It's the final automation step that completes the development cycle without human intervention.

**Independent Test**: Can be tested by completing implementation, then running `wave run --pipeline prototype --from-step pr` and verifying the PR is created, reviewed, and processed through the merge cycle.

**Acceptance Scenarios**:

1. **Given** a completed implementation with passing tests, **When** the pr-create phase runs, **Then** the system creates a PR using `gh pr create` with a summary generated from the spec and implementation artifacts.
2. **Given** a created PR, **When** the pr-review phase runs, **Then** the system adds GitHub Copilot as a reviewer and polls until review is complete (max 30 minutes).
3. **Given** Copilot review comments, **When** the pr-respond phase runs, **Then** Claude analyzes each comment, responds with explanations, and either implements fixes or creates follow-up issues for larger changes.
4. **Given** all review comments are resolved, **When** the pr-merge phase runs, **Then** the system marks the PR as ready-to-merge and optionally auto-merges if configured.
5. **Given** review requests changes that require significant rework, **When** the rework threshold is exceeded, **Then** the system creates a GitHub issue linking to the PR and pauses for human review.

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
- What if Copilot review times out?
  - System creates a follow-up issue and marks PR as "pending-human-review".
- What if Claude cannot resolve a review comment?
  - System creates a follow-up issue with the unresolved comment and continues to next comment.
- What if the PR has merge conflicts?
  - System attempts automatic rebase; if conflicts persist, creates issue for manual resolution.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST provide a `prototype.yaml` pipeline at `.wave/pipelines/prototype.yaml` with 6 sequential phases: spec, docs, dummy, implement, pr-cycle, merge.
- **FR-002**: System MUST use the `philosopher` persona for spec and docs phases, `craftsman` persona for dummy and implement phases, `auditor` persona for PR review responses.
- **FR-003**: System MUST validate phase prerequisites via handover contracts before allowing downstream execution.
- **FR-004**: System MUST generate handoff artifacts: `output/spec.md` (spec), `output/README.md`, `output/API.md`, `output/ARCHITECTURE.md` (docs), `output/dummy/` (dummy).
- **FR-005**: System MUST support `--from-step <phase>` flag to re-run from any phase, regenerating downstream artifacts.
- **FR-006**: System MUST support `wave resume` using SQLite state persistence for interrupted runs.
- **FR-007**: System MUST emit NDJSON progress events via the existing event emitter infrastructure.
- **FR-008**: System MUST use workspace isolation at `.wave/workspaces/prototype/<step>/` for each phase.
- **FR-009**: System MUST generate README.md, API.md, and ARCHITECTURE.md as human-readable, technology-agnostic documentation.
- **FR-010**: System MUST create dummy implementations that compile/run and return mock data without business logic.
- **FR-011**: System MUST inject all prior artifacts via `inject_artifacts` memory configuration to the implement phase.
- **FR-012**: System MUST include artifact timestamps; warn if downstream artifacts are newer than their upstream dependencies.
- **FR-013**: System MUST create a PR via `gh pr create` with auto-generated title and body derived from spec and implementation artifacts.
- **FR-014**: System MUST add GitHub Copilot as a PR reviewer via `gh pr edit --add-reviewer`.
- **FR-015**: System MUST poll PR review status via `gh pr view` until review is complete (max 30 minute timeout).
- **FR-016**: System MUST respond to review comments using Claude via the `auditor` persona, posting replies via `gh api`.
- **FR-017**: System MUST implement review-suggested changes using the `craftsman` persona if changes are below the rework threshold.
- **FR-018**: System MUST create follow-up GitHub issues for changes exceeding the rework threshold via `gh issue create`.
- **FR-019**: System MUST mark resolved conversations and request re-review after implementing changes.
- **FR-020**: System MUST support configurable auto-merge via `gh pr merge --auto` when all checks pass.

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

- Users have Wave installed and initialized (`wave init`) with a valid `wave.yaml` manifest.
- The `philosopher` and `craftsman` personas are configured in the manifest (standard with `wave init`).
- The target use case is greenfield (new) projects without existing codebase constraints.
- Dummy implementations target the technology stack inferred from the input description (Go, TypeScript, Python, etc.).
- Users understand the concept of phased development and the value of early prototyping.

## Pipeline YAML Reference

The following shows the complete pipeline structure that will be created at `.wave/pipelines/prototype.yaml`:

```yaml
kind: WavePipeline
metadata:
  name: prototype
  description: "Prototype-driven development: spec → docs → dummy → implement"

input:
  source: cli

steps:
  - id: spec
    persona: philosopher
    # ... (see full implementation)

  - id: docs
    persona: philosopher
    dependencies: [spec]
    # ...

  - id: dummy
    persona: craftsman
    dependencies: [docs]
    # ...

  - id: implement
    persona: craftsman
    dependencies: [dummy]
    # ...
```

See `.wave/pipelines/prototype.yaml` for the complete implementation.
