# Feature Specification: Add Missing Implementer and Reviewer Personas

**Feature Branch**: `021-add-missing-personas`
**Created**: 2026-02-04
**Status**: Draft
**Input**: User description: "Add missing implementer and reviewer personas to Wave. The default pipelines reference these personas but they don't exist. Also fix permission conflicts where navigator has deny:Write(*) but pipelines expect artifact.json output."

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Run Default Pipelines Successfully (Priority: P1)

As a Wave user, I want to run the default pipelines (gh-poor-issues, umami, doc-loop) without encountering "persona not found" errors, so that I can use Wave's built-in workflows out of the box.

**Why this priority**: This is the core blocker preventing any multi-step pipeline from executing. Without the missing personas, Wave's flagship pipelines are unusable.

**Independent Test**: Run `wave run gh-poor-issues "scan re-cinq/wave for issues"` and verify it completes without persona resolution errors.

**Acceptance Scenarios**:

1. **Given** a fresh Wave installation with default configuration, **When** I run `wave run gh-poor-issues`, **Then** the pipeline should start executing without persona resolution errors
2. **Given** a pipeline with `persona: implementer` steps, **When** the executor loads the step, **Then** the implementer persona should be found in the manifest
3. **Given** a pipeline with `persona: reviewer` steps, **When** the executor loads the step, **Then** the reviewer persona should be found in the manifest

---

### User Story 2 - Pipeline Steps Produce Artifacts (Priority: P1)

As a Wave user, I want pipeline steps to successfully write their output artifacts (artifact.json), so that data flows correctly between pipeline steps.

**Why this priority**: Even if personas exist, pipelines fail if steps cannot write their required output artifacts. This is equally critical as the missing personas.

**Independent Test**: Run a single pipeline step that requires artifact output and verify artifact.json is created.

**Acceptance Scenarios**:

1. **Given** an implementer step with a json_schema contract, **When** the step executes, **Then** it should be able to write artifact.json to the workspace
2. **Given** a reviewer step with a json_schema contract, **When** the step executes, **Then** it should be able to write artifact.json to the workspace
3. **Given** any step with output_artifacts defined, **When** the step completes, **Then** the artifacts should exist at the specified paths

---

### User Story 3 - Contract Validation Works End-to-End (Priority: P2)

As a Wave user, I want steps with json_schema contracts to have their output validated against the schema, so that I can trust the data flowing between steps.

**Why this priority**: Contract validation ensures pipeline reliability but requires the personas and artifact writing to work first.

**Independent Test**: Run a pipeline with json_schema contracts and verify validation occurs without errors for valid output.

**Acceptance Scenarios**:

1. **Given** a step with json_schema contract, **When** the step produces valid JSON matching the schema, **Then** the contract validation should pass
2. **Given** a step with json_schema contract, **When** the step produces invalid JSON, **Then** the contract validation should fail with clear error message

---

### User Story 4 - Wave Init Includes All Personas (Priority: P3)

As a new Wave user, I want `wave init` to scaffold a project with all default personas including implementer and reviewer, so that I can use any default pipeline immediately.

**Why this priority**: Important for new user experience but existing users can manually add personas.

**Independent Test**: Run `wave init` in an empty directory and verify all personas are created.

**Acceptance Scenarios**:

1. **Given** an empty directory, **When** I run `wave init`, **Then** `.wave/personas/implementer.md` and `.wave/personas/reviewer.md` should be created

---

### Edge Cases

- What happens when a persona file exists in .wave/personas/ but is not defined in wave.yaml? (Current: persona not found error)
- How does the system handle when the implementer needs both Bash commands AND file writes? (Current: permission model supports this)
- What happens if artifact.json already exists from a previous run? (Current: workspace is cleaned before pipeline runs)
- How are permissions enforced when a step's prompt requests actions outside its allowed tools? (Current: Claude Code enforces permissions via hooks)

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST define an `implementer` persona in wave.yaml with appropriate permissions for executing code changes and writing artifacts
- **FR-002**: System MUST define a `reviewer` persona in wave.yaml with appropriate permissions for reviewing work and writing artifacts
- **FR-003**: System MUST include implementer persona system prompt file at .wave/personas/implementer.md
- **FR-004**: System MUST include reviewer persona system prompt file at .wave/personas/reviewer.md
- **FR-005**: The implementer persona MUST have permissions to: Read files, Write files (including artifact.json), Execute Bash commands, Edit files
- **FR-006**: The reviewer persona MUST have permissions to: Read files, Write files (for artifact output), optionally execute limited Bash commands for verification
- **FR-007**: Both personas MUST be compatible with json_schema contract validation (ability to output valid JSON to artifact.json)
- **FR-008**: System MUST include implementer and reviewer in internal/defaults/personas/ for `wave init` scaffolding
- **FR-009**: Persona system prompts MUST include guidance on JSON output format when used with contracts
- **FR-010**: Personas MUST NOT include embedded contract/schema details in their prompts (contracts are injected at runtime by executor)

### Key Entities

- **Persona**: An AI agent configuration with specific permissions, system prompt, and adapter settings
- **Permission Set**: The allowed and denied tools for a persona (Read, Write, Edit, Bash patterns)
- **Artifact**: JSON output from a pipeline step used for inter-step data flow (written to workspace at artifact.json or specified path)
- **Contract**: Validation rules for step outputs (json_schema type validates against JSON Schema files)
- **Workspace**: Isolated execution environment for each step where artifacts are created

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: All default pipelines (gh-poor-issues, umami, doc-loop, docs-to-impl) can be started without "persona not found" errors
- **SC-002**: Pipeline steps using implementer persona can successfully write artifact.json files
- **SC-003**: Pipeline steps using reviewer persona can successfully write artifact.json files
- **SC-004**: Existing tests pass with the new persona definitions (`go test ./...` exits 0)
- **SC-005**: The `wave init` command scaffolds projects with the new persona files included
- **SC-006**: Contract validation correctly validates output from both new personas

## Additional Gaps Identified

During analysis, the following additional gaps in the step/contract/permission communication were identified:

### Gap 1: Artifact Path Convention Inconsistency
- **Issue**: Some pipelines use `artifact.json`, others use `output/analysis.json`, and some use custom paths
- **Impact**: Confusion about where to write artifacts, especially when prompts say one thing but output_artifacts specifies another
- **Recommendation**: Document and enforce a consistent artifact path convention

### Gap 2: Permission Override Semantics Unclear
- **Issue**: Pipeline prompts say "CRITICAL: You MUST create artifact.json - this overrides the normal no-write constraint" but permissions in wave.yaml control actual behavior
- **Impact**: AI may attempt writes that are denied, causing failures
- **Recommendation**: Either remove misleading "override" language from prompts, or implement actual permission override in step config

### Gap 3: Schema Injection vs Prompt Duplication
- **Issue**: Executor injects schema (lines 656-744) but meta-generated pipelines also embed schema descriptions in prompts
- **Impact**: Redundant information, potential for drift between prompt description and actual schema
- **Recommendation**: Generated pipelines should rely solely on schema injection, not duplicate schema info in prompts

### Gap 4: Missing Default Personas in Embedded Defaults
- **Issue**: `internal/defaults/personas/` lacks implementer.md and reviewer.md even though default pipelines use them
- **Impact**: `wave init` creates incomplete project scaffolding
- **Recommendation**: Add missing personas to embedded defaults (part of this spec)

### Gap 5: Contract Type Mismatch in Pipelines
- **Issue**: Some steps define `handover.contract` without `output_artifacts`, others have artifacts without contracts
- **Impact**: Unclear whether artifact creation is validated or just collected
- **Recommendation**: Document when both are needed vs when either suffices

## Assumptions

- The implementer persona should have broad permissions similar to the existing `craftsman` persona, as it needs to execute arbitrary code changes
- The reviewer persona should have more limited permissions focused on reading and analysis, with write access only for artifacts
- Both personas should follow the existing persona file structure (markdown with sections for Responsibilities, Output Format, Constraints)
- The schema injection mechanism in executor.go (lines 656-744) already works correctly and does not need changes
- Permissions in wave.yaml take precedence over constraints stated in persona markdown files
- The workspace path for artifacts is determined by the executor, not the persona
