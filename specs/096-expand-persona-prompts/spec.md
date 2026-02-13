# Feature Specification: Expand Persona Definitions with Detailed System Prompts

**Feature Branch**: `096-expand-persona-prompts`
**Created**: 2026-02-13
**Status**: Draft
**Input**: [GitHub Issue #96](https://github.com/re-cinq/wave/issues/96) — Expand persona definitions with detailed system prompts and role specifications

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Consistent Agent Behavior Across Pipeline Steps (Priority: P1)

As a Wave pipeline operator, I want each persona to produce high-quality, consistent outputs so that pipeline steps complete reliably without requiring manual intervention or re-runs due to vague or off-target agent behavior.

**Why this priority**: Persona prompts are the **primary mechanism** for agent behavior in Wave because fresh memory at every step boundary means no chat history inheritance. Brief, underspecified personas lead to inconsistent outputs, missed requirements, and pipeline failures. Improving prompt quality directly improves pipeline reliability.

**Independent Test**: Can be tested by running any existing pipeline (e.g., `code-review`, `plan`, `refactor`) with the expanded persona prompts and verifying that outputs meet the same or better quality bar as before, without regressions.

**Acceptance Scenarios**:

1. **Given** a pipeline step using the `navigator` persona, **When** the step executes with the expanded prompt, **Then** the agent produces structured JSON output covering files, patterns, dependencies, and impact areas — consistent with the output format specification.
2. **Given** a pipeline step using the `implementer` persona, **When** the step executes against any codebase (Go, Python, TypeScript, etc.), **Then** the agent follows the implementation process without assuming language-specific patterns not present in the codebase.
3. **Given** a pipeline step using the `reviewer` persona, **When** the step executes, **Then** findings are categorized by severity (CRITICAL/HIGH/MEDIUM/LOW) with specific file path citations, matching the output format specification.

---

### User Story 2 - Language-Agnostic Persona Definitions (Priority: P1)

As a Wave user running pipelines against non-Go codebases, I want persona definitions to be language-agnostic so that agents adapt to the target project's language, frameworks, and conventions rather than assuming Go-specific patterns.

**Why this priority**: Per issue comment — personas "should not be too specific and general for all languages." Wave is a general-purpose orchestrator. Persona prompts that reference Go-specific tooling (e.g., `go test`, `go vet`) or patterns would limit Wave's applicability. Personas must describe behaviors, not language-specific commands.

**Independent Test**: Can be tested by reviewing each expanded persona definition and confirming that no language-specific commands, frameworks, or tooling are hardcoded into the prompt text. Language examples, if present, are clearly marked as illustrative and not prescriptive.

**Acceptance Scenarios**:

1. **Given** any expanded persona file, **When** the content is reviewed, **Then** it contains zero hardcoded references to a specific programming language's toolchain (e.g., no `go test`, `npm install`, `cargo build`).
2. **Given** an expanded persona for the `craftsman`, **When** it describes testing practices, **Then** it references testing in general terms (e.g., "run the project's test suite") rather than a specific test runner.
3. **Given** an expanded persona for the `debugger`, **When** it describes diagnostic techniques, **Then** it describes universal debugging strategies (hypothesis testing, bisection, logging) without prescribing language-specific debugger tools.

---

### User Story 3 - Parity Between .wave/ and internal/defaults/ (Priority: P1)

As a Wave developer, I want expanded persona files in `.wave/personas/` to be identically mirrored in `internal/defaults/personas/` so that `wave init` bootstraps new projects with the same rich persona definitions, and the embedded defaults never fall out of sync with the filesystem versions.

**Why this priority**: Per issue comment — "parity between .wave and internal/defaults." The embedded defaults (`internal/defaults/personas/`) are compiled into the Wave binary via `//go:embed` and used by `wave init` to bootstrap new projects. If only `.wave/` is updated, new installations would receive stale, brief persona definitions.

**Independent Test**: Can be tested by running a diff between every file in `.wave/personas/` and its counterpart in `internal/defaults/personas/` — all files must be byte-identical.

**Acceptance Scenarios**:

1. **Given** all 13 persona files have been expanded, **When** a diff is run between `.wave/personas/*.md` and `internal/defaults/personas/*.md`, **Then** every file is identical (zero differences).
2. **Given** a developer runs `wave init` in a new project, **When** the init process copies embedded persona defaults, **Then** the initialized `.wave/personas/` directory contains the expanded persona definitions (not the old brief versions).

---

### User Story 4 - Self-Contained Persona Identity (Priority: P2)

As a Wave pipeline author, I want each persona to include a clear "You are..." identity statement and a structured description of its domain expertise, process, and constraints so that an AI agent reading only the persona prompt has complete context to perform its role — without relying on external documentation or prior conversation history.

**Why this priority**: Fresh memory at step boundaries means the persona prompt is the **only** behavioral specification the agent receives. If the prompt is incomplete, the agent must guess, leading to inconsistent or incorrect behavior.

**Independent Test**: Can be tested by providing each expanded persona prompt to an LLM in isolation (no other context) and asking it to describe its role, process, and constraints — the LLM's self-description should match the persona's intended behavior.

**Acceptance Scenarios**:

1. **Given** any expanded persona file, **When** the content is read, **Then** it begins with a clear "You are..." identity statement within the first 3 lines.
2. **Given** any expanded persona file, **When** the content is reviewed, **Then** it includes sections for: domain expertise, responsibilities, process/methodology, output format, and constraints.
3. **Given** any expanded persona file, **When** the line count is measured, **Then** it is at least 30 lines long.

---

### User Story 5 - No Disruption to Existing Pipeline Execution (Priority: P2)

As a Wave user with existing pipelines, I want the persona expansion to be a content-only change that does not alter the persona loading mechanism, wave.yaml configuration, or Go source code so that all existing pipelines and tests continue to work without modification.

**Why this priority**: This change must be zero-risk to the runtime. The persona loading mechanism reads markdown files from disk — changing the content of those files should not break any code paths, only improve agent behavior quality.

**Independent Test**: Can be tested by running `go test ./...` against the full test suite after persona file updates — all tests must pass. Additionally, no Go source files or `wave.yaml` should be modified.

**Acceptance Scenarios**:

1. **Given** all persona files have been expanded, **When** `go test ./...` is run, **Then** all tests pass with zero failures.
2. **Given** the change set, **When** files are reviewed, **Then** only `.md` files in `.wave/personas/` and `internal/defaults/personas/` have been modified — no Go source code, no `wave.yaml`, no JSON schemas.
3. **Given** the expanded persona files, **When** loaded by the manifest parser, **Then** the parser successfully resolves all `system_prompt_file` references without errors.

---

### Edge Cases

- What happens if an expanded persona exceeds LLM context window limits when combined with the step prompt and injected artifacts? Persona prompts should remain concise enough (target: under 200 lines each) to leave ample room for task-specific context.
- What happens if a persona's output format section contradicts the contract schema injected at runtime? The persona prompt should make clear that contract schemas, when provided, take precedence over default output format guidance.
- What happens if a persona's "expected tools" list doesn't match the actual permissions in wave.yaml? The persona should document expected tools for self-awareness but explicitly note that actual permissions are enforced externally by the pipeline orchestrator.
- What happens if the `.wave/personas/` and `internal/defaults/personas/` directories contain different sets of files? This feature only updates existing files — it must not add or remove persona files from either directory.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: Each of the 13 existing persona definitions MUST be expanded with a clear "You are..." identity statement in the opening paragraph.
- **FR-002**: Each persona MUST include a "Domain Expertise" section listing the persona's specific knowledge areas relevant to its role.
- **FR-003**: Each persona MUST include a "Responsibilities" section enumerating the persona's concrete duties within a pipeline step.
- **FR-004**: Each persona MUST include a "Communication Style" or "Process" section describing the persona's methodology, tone, and approach to its work.
- **FR-005**: Each persona MUST include a "Tools and Permissions" section documenting the tools the persona expects to use, using Wave's permission syntax where applicable (e.g., `Bash(git log*)`, `Write(artifact.json)`) to give the agent accurate self-awareness. The section MUST include an explicit note that actual permissions are enforced by the pipeline orchestrator — not by the persona prompt itself. Tool references in this section MUST remain language-agnostic (e.g., "run the project's test suite" rather than `Bash(go test*)`).
- **FR-006**: Each persona MUST include an "Output Format" section specifying the default output structure, with a note that contract schemas override these defaults when provided.
- **FR-007**: Each persona MUST include a "Constraints" section listing hard behavioral boundaries.
- **FR-008**: All persona definitions MUST be language-agnostic — no hardcoded references to specific programming languages, frameworks, or toolchains. This applies to both newly written and pre-existing content. Currently 4 personas (`craftsman`, `reviewer`, `auditor`, `debugger`) contain Go-specific or language-specific references that must be generalized (e.g., "Go conventions" → "language conventions for the target codebase"; `go vet` → "static analysis tools"; `go test` → "the project's test suite").
- **FR-009**: Each expanded persona file MUST be at least 30 lines long.
- **FR-010**: The content of every file in `.wave/personas/` MUST be byte-identical to its counterpart in `internal/defaults/personas/`.
- **FR-011**: No Go source code (`.go` files), `wave.yaml` configuration, or JSON schema files MUST be modified as part of this change. Note: Markdown persona files (`.md`) in `internal/defaults/personas/` are persona content, not Go source code — updating them is explicitly in scope per FR-010.
- **FR-012**: All existing tests MUST continue to pass after persona file updates.
- **FR-013**: Each persona MUST remain under 200 lines to avoid excessive context consumption when combined with step prompts and artifacts.

### Key Entities

- **Persona Definition**: A markdown file that serves as the complete behavioral specification for an AI agent within a pipeline step. Located in `.wave/personas/` (runtime) and `internal/defaults/personas/` (embedded for `wave init`). Referenced by `system_prompt_file` field in `wave.yaml` persona configuration. Contains identity, expertise, responsibilities, process, output format, and constraints.
- **Persona Structural Template**: A consistent markdown structure that all 13 expanded personas should follow, ensuring uniformity across the persona library while allowing role-specific content within each section.

## Personas to Expand

> **Note**: Commit `6fdb3e9` applied an initial expansion to all 13 personas.
> The line counts below reflect the **current** state after that expansion.
> This feature's remaining scope is to **refine and validate** the already-expanded
> personas against FR-001 through FR-013 — particularly fixing language-specific
> references (FR-008 violations) and ensuring structural template conformance.

| # | File | Role | Current Lines | FR-008 Status |
|---|------|------|---------------|---------------|
| 1 | `navigator.md` | Codebase exploration and analysis | 57 | Pass |
| 2 | `philosopher.md` | Architecture and specification writing | 63 | Pass |
| 3 | `planner.md` | Task planning and decomposition | 65 | Pass |
| 4 | `craftsman.md` | Production-quality implementation | 60 | **FAIL** — Go-specific domain expertise |
| 5 | `implementer.md` | Task execution and artifact production | 58 | Pass |
| 6 | `reviewer.md` | Quality review and validation | 70 | **FAIL** — `go test`, `npm test` in process section |
| 7 | `auditor.md` | Security and compliance review | 60 | **FAIL** — Go-specific identity, expertise, and tools |
| 8 | `debugger.md` | Bug investigation and root cause analysis | 72 | **FAIL** — Go-specific identity, expertise, and tools |
| 9 | `researcher.md` | Web research and information synthesis | 93 | Pass |
| 10 | `summarizer.md` | Context compaction and summarization | 60 | Pass |
| 11 | `github-analyst.md` | GitHub issue analysis and scoring | 75 | Pass |
| 12 | `github-commenter.md` | GitHub issue commenting | 91 | Pass |
| 13 | `github-enhancer.md` | GitHub issue enhancement | 76 | Pass |

## Structural Template for Expanded Personas

Each persona MUST follow this consistent structure. Section names may be adapted where the persona's role calls for it (e.g., "Debugging Process" instead of "Process", "Communication Style" as a complement to "Process"), but all seven **concepts** must be present: (1) identity statement, (2) domain expertise, (3) responsibilities, (4) process/methodology, (5) tools and permissions, (6) output format, (7) constraints. A persona may include additional sections (e.g., "Best Practices", "Communication Style") beyond the required seven:

```markdown
# [Persona Name]

You are [identity statement — who you are and your primary function within Wave pipelines].

## Domain Expertise
- [Area 1]: [specific knowledge relevant to the role]
- [Area 2]: [specific knowledge relevant to the role]
- [Area 3]: [specific knowledge relevant to the role]

## Responsibilities
- [Concrete duty 1]
- [Concrete duty 2]
- [Concrete duty 3]

## Process
1. [Step 1 — methodology for approaching tasks]
2. [Step 2]
3. [Step 3]

## Tools and Permissions
- Expected tools: [list of tools this persona typically uses]
- Scope: [read-only / read-write / etc.]
- Note: Actual permissions are enforced by the pipeline orchestrator, not this prompt

## Output Format
[Default output format and quality standards]
Note: When a contract schema is provided at runtime, it takes precedence over these defaults.

## Constraints
- [Hard behavioral boundary 1]
- [Hard behavioral boundary 2]
- [What the persona must never do]
```

## Out of Scope

- Changes to the persona loading mechanism (`internal/manifest/parser.go`)
- Adding new personas — only expanding existing ones
- Changes to permission enforcement or pipeline execution logic
- Changes to `wave.yaml` persona configuration
- Changes to contract schemas or pipeline definitions
- Changes to any Go source code

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: All 13 persona files in both `.wave/personas/` and `internal/defaults/personas/` are updated with expanded definitions (26 files total).
- **SC-002**: Every expanded persona file is at least 30 lines and no more than 200 lines.
- **SC-003**: Every expanded persona file follows the structural template with all required sections present (identity statement, domain expertise, responsibilities, process, tools, output format, constraints).
- **SC-004**: Zero persona files contain hardcoded references to a specific programming language's toolchain or framework.
- **SC-005**: `diff -r .wave/personas/ internal/defaults/personas/` produces zero differences.
- **SC-006**: `go test ./...` passes with zero failures after all persona files are updated.
- **SC-007**: No Go source files, `wave.yaml`, or JSON schema files are modified in the change set.

## Clarifications _(resolved during spec refinement)_

### C-001: Language-specific references in existing expanded personas

**Question**: FR-008 requires language-agnostic personas, but 4 already-expanded personas (`craftsman`, `reviewer`, `auditor`, `debugger`) contain Go-specific references (e.g., "Go systems", "Go conventions", `go test`, `go vet`). Should these be left as-is or generalized?

**Resolution**: Generalize all language-specific references. FR-008 applies to the **final** persona content, not just newly written content. The issue comment explicitly states personas "should not be too specific and general for all languages." Specific items to fix:
- `craftsman.md`: "Go conventions including effective Go practices" → "Language conventions and idiomatic patterns for the target codebase"
- `debugger.md`: "Go systems" → "software systems"; "concurrent Go programs" → "concurrent programs"; "Go-specific debugging" → language-agnostic debugging techniques
- `auditor.md`: "Go systems" → "software systems"; "Go-specific security concerns" → language-agnostic security concerns
- `reviewer.md`: `go test`, `npm test` → "the project's test suite"

**Rationale**: Wave is a general-purpose orchestrator. Language-specific references in persona prompts bias agent behavior toward that language when running against non-Go codebases.

### C-002: Scope of FR-011 (no Go source code changes) vs. markdown files in internal/defaults/

**Question**: FR-011 says "No Go source code... MUST be modified." The `internal/defaults/personas/` directory sits under `internal/`, which is Go package space. Does updating `.md` files there violate FR-011?

**Resolution**: No. FR-011 refers specifically to `.go` files, `wave.yaml`, and `.json` schema files. Markdown persona files in `internal/defaults/personas/` are embedded content consumed via `//go:embed` — they are persona definitions, not Go source code. Updating them is explicitly required by FR-010 and SC-001.

**Rationale**: The intent of FR-011 is to ensure zero risk to runtime behavior by not modifying executable code or configuration. Persona markdown files are content, not code.

### C-003: Stale line counts in "Personas to Expand" table

**Question**: The table listed original line counts (e.g., navigator: 19) but commit `6fdb3e9` already expanded all 13 personas (navigator is now 57 lines). Is the spec describing an already-completed task?

**Resolution**: Updated the table to reflect current line counts and added a note clarifying that the initial expansion is done. The remaining scope is to **refine and validate** the expanded personas against all functional requirements — particularly fixing the 4 FR-008 violations identified above and ensuring parity between `.wave/personas/` and `internal/defaults/personas/`.

**Rationale**: Accurate line counts prevent implementers from misunderstanding the current state. The table now also includes an FR-008 status column to highlight exactly which files need attention.

### C-004: Structural template section naming flexibility

**Question**: The template specifies exact section names ("Domain Expertise", "Process", etc.) but actual personas use variations like "Communication Style", "Debugging Process", "Best Practices", "Research Process". Are these acceptable alternatives?

**Resolution**: Yes. The requirement is that all seven **concepts** are present (identity, expertise, responsibilities, methodology, tools, output format, constraints), not that section headings match exactly. A persona may use role-appropriate names (e.g., "Debugging Process" instead of "Process") and include additional sections beyond the required seven (e.g., "Communication Style", "Best Practices"). Updated the template description to state this explicitly.

**Rationale**: Forcing identical headings on all 13 personas would reduce clarity. A debugger's "Debugging Process" is more informative than a generic "Process" heading. The structural consistency comes from the concepts being present, not from header strings.

### C-005: Tools and Permissions section — abstract vs. concrete tool references

**Question**: Should the "Tools and Permissions" section list tools abstractly (e.g., "shell access for running tests") or use Wave's permission syntax (e.g., `Bash(go test*)`)?

**Resolution**: Use Wave's permission syntax for precision (e.g., `Bash(git log*)`, `Write(artifact.json)`) since it gives the agent accurate self-awareness about its expected capabilities. However, tool references that would be language-specific MUST be generalized per FR-008 (e.g., `Bash(go test*)` → "run the project's test suite via Bash"). Updated FR-005 to encode this.

**Rationale**: Wave's permission syntax is the ground truth for what tools a persona can use. Matching it in the prompt reduces confusion. But language-specific test runners violate FR-008 and must be described generically.
