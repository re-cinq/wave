# Feature Specification: Optimize Persona Prompts for High-Signal Context Engineering

**Feature Branch**: `113-persona-prompt-optimization`
**Created**: 2026-02-20
**Status**: Draft
**Input**: [GitHub Issue #96](https://github.com/re-cinq/wave/issues/96) — Optimize persona prompts for high-signal context engineering

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Shared Base Protocol Extraction (Priority: P1)

As a Wave maintainer, I want a shared base protocol that captures guidance common to all personas, so that individual persona files contain only role-differentiating content and shared constraints are not duplicated across 17 files.

**Why this priority**: The shared base protocol is the foundation for all other optimizations. Without it, removing shared content from individual personas would leave them incomplete. This also establishes the runtime injection pattern that all other changes depend on.

**Independent Test**: Can be tested by verifying that the base protocol file exists, contains the expected shared constraints, and is injected at runtime alongside each persona's individual prompt. A pipeline step executed with any persona should include both the base protocol and persona-specific content in the generated CLAUDE.md.

**Acceptance Scenarios**:

1. **Given** a Wave project with the default manifest, **When** a pipeline step executes with any persona, **Then** the generated CLAUDE.md contains the shared base protocol content prepended to the persona-specific content.
2. **Given** the shared base protocol file, **When** inspecting its content, **Then** it contains Wave-specific constraints (fresh memory, artifact input, workspace isolation, contract compliance, security enforcement) and no role-specific guidance.
3. **Given** a persona file after optimization, **When** comparing it to the base protocol, **Then** there is zero duplication of content between the two.

---

### User Story 2 - Persona Prompt Compaction (Priority: P1)

As a Wave pipeline operator, I want each persona prompt to contain only role-differentiating content (identity, unique responsibilities, output contract, permission model), so that context window budget is maximized for actual task content rather than consumed by generic or duplicated guidance.

**Why this priority**: This is the core deliverable of the issue — making every token earn its place. Without this, the system prompt overhead per step remains unnecessarily high, directly reducing the context available for task-relevant information.

**Independent Test**: Can be tested by measuring token counts per persona before and after optimization, and by verifying that each persona file contains only the five required elements (identity statement, role-specific responsibilities, permission model, Wave architecture constraints unique to the role, output contract) and none of the identified anti-patterns (generic domain expertise lists, copy-pasted communication style, meta-workflow process descriptions, length padding).

**Acceptance Scenarios**:

1. **Given** any optimized persona file, **When** analyzing its content, **Then** it contains an identity statement, role-specific responsibilities, and an output contract section.
2. **Given** any optimized persona file, **When** searching for generic sections, **Then** it does not contain a "Communication Style" section, a "Domain Expertise" section that merely restates responsibilities, or a process section describing the generic read-process-output workflow.
3. **Given** all 17 optimized persona files, **When** measuring their token counts, **Then** each falls within the 100–400 token range.
4. **Given** the original and optimized persona files, **When** comparing behavioral coverage, **Then** no role-differentiating instruction present in the original is missing from the combination of base protocol + optimized persona.

---

### User Story 3 - Parity Maintenance Between Embedded and Workspace Personas (Priority: P2)

As a Wave developer, I want the persona files in `internal/defaults/personas/` (embedded in the binary) and `.wave/personas/` (workspace copy) to remain identical, so that development testing with local overrides produces the same behavior as the release binary.

**Why this priority**: The maintainer comment on the issue explicitly calls out parity between `.wave` and `internal/defaults` as a requirement. Drift between these locations would cause confusing behavioral differences between local development and production.

**Independent Test**: Can be tested by running a diff between every file in `internal/defaults/personas/` and `.wave/personas/` and asserting zero differences.

**Acceptance Scenarios**:

1. **Given** the optimized persona files, **When** comparing `internal/defaults/personas/<name>.md` to `.wave/personas/<name>.md` for each of the 17 personas, **Then** the files are byte-identical.
2. **Given** a CI pipeline or test suite, **When** parity is checked, **Then** any drift between the two locations causes a test failure.

---

### User Story 4 - Language-Agnostic Persona Content (Priority: P2)

As a Wave user targeting projects in any programming language, I want persona prompts to contain no language-specific references (e.g., Go, Python, TypeScript), so that Wave personas work effectively regardless of the target project's technology stack.

**Why this priority**: The maintainer comment explicitly requires personas to be "not too specific and general for all languages." Language-specific references in persona prompts would bias agent behavior toward one technology stack.

**Independent Test**: Can be tested by searching all persona files for language-specific keywords (Go, Golang, Python, TypeScript, JavaScript, Java, Rust, etc.) and asserting zero matches.

**Acceptance Scenarios**:

1. **Given** any optimized persona file, **When** searching for programming language names, **Then** no language-specific references are found.
2. **Given** a persona that previously contained language-specific advice, **When** reviewing its optimized version, **Then** the advice is either generalized to be language-agnostic or removed entirely.

---

### Edge Cases

- What happens when a persona has no unique constraints beyond the base protocol? The persona file still contains at minimum an identity statement and role-specific responsibilities. An empty persona file is not valid.
- What happens when the base protocol file is missing at runtime? The system should fail with a clear error rather than silently executing without shared constraints, since the base protocol contains security-critical content.
- What happens when a persona file references constraints already in the base protocol? This is the duplication that this feature eliminates. The optimization process must audit each persona to remove any content that is covered by the base protocol.
- What happens when permission enforcement text is moved to the base protocol? Permission enforcement MUST remain persona-specific because each persona has different allowed/denied tools. The base protocol should only state that permissions are enforced, not list specific permissions.
- What happens when a persona's unique responsibilities overlap with another persona's? Each persona must have a clearly distinct identity statement and responsibility set. If two personas share responsibilities, the prompts should emphasize what differentiates them, not what they share.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The system MUST provide a shared base protocol file named `base-protocol.md` located in `internal/defaults/personas/` (embedded via the existing `//go:embed personas/*.md` directive) and `.wave/personas/` (workspace copy). This file contains Wave-specific constraints applicable to all personas: fresh memory at step boundaries, artifact-based inter-step communication (reading input from injected artifacts, writing output to artifact files), workspace isolation (ephemeral worktree per step), contract compliance (outputs must satisfy the step's contract before completion), and security enforcement (permissions are enforced — defer to the restriction section for specifics).
- **FR-002**: The runtime MUST inject the shared base protocol as a preamble to each persona's individual system prompt when generating the CLAUDE.md for a pipeline step. Specifically, the `prepareWorkspace` method in `internal/adapter/claude.go` must read `base-protocol.md` and prepend its content before the persona-specific prompt, separated by a markdown horizontal rule (`---`). The base protocol file MUST be loaded from `.wave/personas/base-protocol.md` (same directory as persona files). If the file is missing, `prepareWorkspace` MUST return an error rather than silently proceeding without it.
- **FR-003**: Each persona file MUST contain an identity statement that differentiates it from all other personas.
- **FR-004**: Each persona file MUST contain role-specific responsibilities that describe what this persona does that others do not.
- **FR-005**: Each persona file MUST contain an output contract section specifying the expected output format for the role.
- **FR-006**: Each persona file MUST NOT contain generic "Communication Style" sections, "Domain Expertise" sections that restate responsibilities, or process sections describing the generic read-process-output workflow.
- **FR-007**: Each persona file MUST NOT contain programming language-specific references.
- **FR-008**: Each optimized persona file MUST fall within the 100–400 token range. The lower bound is 100 tokens (not 200) because some personas (e.g., navigator, philosopher) have genuinely minimal role-differentiating content once shared constraints move to the base protocol. Artificially padding short personas would contradict the goal of high-signal content. The upper bound of 400 tokens remains firm to prevent prompt bloat.
- **FR-009**: Permission enforcement (allow/deny tool lists) MUST be preserved in each persona's manifest configuration (`wave.yaml`) and projected into `settings.json` and the CLAUDE.md restriction section at runtime — this is non-negotiable for security. Persona `.md` prompt files SHOULD still contain behavioral constraint reminders relevant to the role (e.g., "NEVER modify source code" for read-only personas) as defense-in-depth, but MUST NOT duplicate the specific tool allow/deny lists which are authoritatively managed by the manifest. The base protocol MUST include a generic statement that permissions are enforced but MUST NOT list specific tools.
- **FR-010**: All 17 personas MUST be optimized: navigator, implementer, reviewer, planner, researcher, debugger, auditor, craftsman, summarizer, github-analyst, github-commenter, github-enhancer, philosopher, provocateur, validator, synthesizer, supervisor.
- **FR-011**: Files in `internal/defaults/personas/` and `.wave/personas/` MUST be kept in parity (identical content) for all 17 personas.
- **FR-012**: The shared base protocol MUST exist as `base-protocol.md` in both `internal/defaults/personas/` (embedded via the existing `//go:embed personas/*.md` directive in `embed.go`) and `.wave/personas/` (workspace copy, same directory as persona files). These two copies MUST be byte-identical, following the same parity rule as persona files (FR-011).
- **FR-013**: Existing tests MUST continue to pass after optimization — no behavioral regressions.
- **FR-014**: The system MUST NOT add new personas beyond the current 17.
- **FR-015**: The system MUST NOT change the persona loading mechanism, permission enforcement logic, or pipeline execution logic. The only permitted change to `prepareWorkspace` in `internal/adapter/claude.go` is adding the base protocol prepend logic before the existing persona prompt write. The `GetPersonas()` function in `embed.go` will automatically include `base-protocol.md` since it already reads all `personas/*.md` files, but callers that iterate over personas (e.g., `wave init`) SHOULD exclude `base-protocol.md` from the persona list to avoid creating a persona entry for it.

### Key Entities

- **Shared Base Protocol**: A markdown file (`base-protocol.md`) containing Wave-specific constraints and operational context applicable to all personas. Stored in `internal/defaults/personas/` and `.wave/personas/`. Injected at runtime as a preamble to each persona's system prompt in the generated CLAUDE.md. Contains: fresh memory constraint (no chat history between steps), artifact communication model (inter-step data passes through artifact files — read inputs from injected artifacts, write outputs to artifact files), workspace isolation (each step runs in an ephemeral worktree), contract compliance (outputs must satisfy the step's validation contract), and security enforcement reminder (permissions are enforced by the orchestrator — do not attempt to bypass restrictions).
- **Persona Prompt**: A markdown file defining a single persona's unique behavioral identity, responsibilities, and output contract. After optimization, contains only content that differentiates this persona from others.
- **Persona**: A configured agent role within Wave, defined by a prompt file, adapter, model, temperature, and permission set. The manifest links a persona name to its prompt file and runtime configuration.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Every persona prompt token count falls within the 100–400 token range (measurable via tokenizer or word-count heuristic of ~0.75 words per token). The lower bound of 100 accommodates personas with genuinely minimal role-differentiating content.
- **SC-002**: Zero content duplication exists between the shared base protocol and any individual persona file (measurable via text diff analysis).
- **SC-003**: Zero programming language-specific keywords appear in any persona file (measurable via grep for language names).
- **SC-004**: All 17 personas are covered — no persona is left unoptimized (measurable by file count and checklist).
- **SC-005**: 100% of existing tests pass after the changes (measurable via `go test ./...`).
- **SC-006**: Byte-identical parity between `internal/defaults/personas/` and `.wave/personas/` for all 17 persona files (measurable via diff).
- **SC-007**: Each persona file contains the three mandatory elements: identity statement, role-specific responsibilities, output contract (measurable via structural analysis).
- **SC-008**: The shared base protocol is injected at runtime for every pipeline step execution (measurable via integration test or CLAUDE.md inspection).

## Clarifications _(resolved during specification refinement)_

### CLR-001: Base protocol file location and naming

**Ambiguity**: FR-001 and FR-012 specified the base protocol must exist in `internal/defaults/` and `.wave/` but did not specify the exact filename or subdirectory path. The existing embed directive in `embed.go` is `//go:embed personas/*.md` — placing the file outside `personas/` would require a new embed directive.

**Resolution**: The base protocol file is named `base-protocol.md` and lives in the `personas/` subdirectory at both locations (`internal/defaults/personas/base-protocol.md` and `.wave/personas/base-protocol.md`). This leverages the existing `//go:embed personas/*.md` directive without modification. The `GetPersonas()` function will automatically include it, so callers that iterate personas for manifest purposes should filter it out by name.

**Rationale**: Placing the file alongside personas is the path of least resistance — no new embed directives, no new directory conventions. The file is conceptually part of the persona system even though it isn't a persona itself.

---

### CLR-002: Token range lower bound adjusted from 200 to 100

**Ambiguity**: FR-008 specified a 200–400 token range, but current persona word counts range from 125–297 words (~167–396 tokens at the 0.75 words/token heuristic). After extracting shared content to the base protocol, short personas like navigator (~127 words) and philosopher (~125 words) would likely fall well below 200 tokens. The spec was ambiguous about whether short personas should be artificially padded.

**Resolution**: The lower bound is adjusted to 100 tokens. Personas that genuinely have minimal role-differentiating content should not be padded with filler text, as this directly contradicts the "every token earns its place" principle from the original issue.

**Rationale**: The issue explicitly states "all content should earn its place." Padding short personas to meet an arbitrary floor would introduce the exact anti-pattern this feature eliminates.

---

### CLR-003: Runtime injection point specified

**Ambiguity**: FR-002 stated the base protocol must be "prepended" but did not specify the exact code location. The current `prepareWorkspace` method in `internal/adapter/claude.go` builds CLAUDE.md from persona prompt + restriction section. The spec was unclear whether injection should happen in the adapter, the manifest loader, or elsewhere.

**Resolution**: Injection occurs in the `prepareWorkspace` method of `internal/adapter/claude.go`, immediately before the persona-specific prompt is written. The base protocol content is read from `.wave/personas/base-protocol.md`, written first, followed by a `---` separator, then the persona prompt. If the base protocol file cannot be read, `prepareWorkspace` returns an error (fail-secure). This is the minimal change to existing code.

**Rationale**: The adapter's `prepareWorkspace` is already responsible for assembling the CLAUDE.md content. Adding the base protocol read there keeps the change localized and avoids modifying the manifest parser, persona loader, or pipeline executor.

---

### CLR-004: "Artifact input model" definition clarified

**Ambiguity**: The spec and Key Entities section referenced "artifact input model" as one of the five base protocol content areas without defining what it means. This could refer to the JSON artifact output format, the artifact injection mechanism, or both.

**Resolution**: "Artifact input model" is renamed to "artifact communication model" throughout the spec. It refers to the inter-step data passing mechanism: each step receives input from artifacts injected by the orchestrator (previous step outputs) and writes its own output to artifact files. This is distinct from the output contract/format (which is persona-specific and stays in each persona file).

**Rationale**: The artifact communication pattern (read injected inputs, write outputs) is universal across all personas and belongs in the base protocol. The specific output format (JSON schema, markdown structure, etc.) varies by persona and belongs in the persona file's output contract section.

---

### CLR-005: Permission text in persona prompts vs manifest-managed permissions

**Ambiguity**: FR-009 stated permissions must be "preserved in each persona's configuration" and Edge Case #4 said permissions must remain persona-specific. However, tool allow/deny lists are already authoritatively managed in `wave.yaml` and projected into `settings.json` + the CLAUDE.md restriction section by the adapter. The spec was unclear whether persona `.md` files should also contain permission text.

**Resolution**: Persona `.md` files SHOULD retain behavioral constraint reminders as defense-in-depth (e.g., "NEVER modify source code" for read-only personas, "NEVER commit or push" for non-write personas) but MUST NOT duplicate specific tool allow/deny lists. The base protocol includes a generic "permissions are enforced by the orchestrator" statement. The specific tool permissions remain exclusively in `wave.yaml` and are projected at runtime.

**Rationale**: The restriction section appended to CLAUDE.md already contains the authoritative tool lists. Duplicating them in persona prompts creates maintenance burden and drift risk. Behavioral reminders (in natural language, not tool-list format) serve as defense-in-depth and remain appropriate in persona prompts.
