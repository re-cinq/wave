# Feature Specification: Wave Task Classifier

**Feature Branch**: `772-task-classifier`  
**Created**: 2026-04-12  
**Status**: Draft  
**Input**: User description: "Implement Wave Task Classifier (issue #772 WS1.1-1.3). Create a new internal/classify/ package with TaskProfile type, input analyzer, pipeline selector, and unit tests."

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Classify Free-Text Task Input (Priority: P1)

A Wave user provides a free-text task description (e.g., "fix the login bug in auth middleware") and the system analyzes it to determine the task's complexity, domain, blast radius, and required verification depth — producing a structured TaskProfile that downstream pipeline selection can consume.

**Why this priority**: Classification is the foundational capability. Without accurate input analysis, pipeline selection cannot function. This is the core value proposition of the classifier.

**Independent Test**: Can be fully tested by passing various text inputs to ClassifyInput and asserting correct TaskProfile field values. Delivers value as a standalone analysis utility.

**Acceptance Scenarios**:

1. **Given** a free-text input "fix typo in README", **When** Classify is called, **Then** the returned TaskProfile has complexity=simple, domain=docs, blast_radius<0.2, verification_depth=structural_only
2. **Given** a free-text input "redesign the pipeline routing architecture to support plugin-based step execution", **When** Classify is called, **Then** the returned TaskProfile has complexity=architectural, domain=feature, blast_radius>0.7, verification_depth=full_semantic
3. **Given** a free-text input "fix SQL injection vulnerability in user endpoint", **When** Classify is called, **Then** the returned TaskProfile has domain=security, blast_radius>0.5

---

### User Story 2 - Classify Issue/PR URL Input (Priority: P1)

A Wave user provides a GitHub issue URL or PR URL, along with the fetched issue body text. The system detects the URL type (issue vs PR) and combines that signal with the issue body content to produce an accurate TaskProfile.

**Why this priority**: URL-based input is the most common entry point for Wave task routing. Reusing the existing InputType detection from internal/suggest/input.go ensures consistency.

**Independent Test**: Can be tested by passing URL strings with accompanying issue body text to ClassifyInput and verifying correct profiles. Verifiable independently of pipeline selection.

**Acceptance Scenarios**:

1. **Given** input "https://github.com/org/repo/issues/42" with issue body "login button doesn't work on mobile", **When** Classify is called, **Then** the returned TaskProfile has domain=bug, complexity=simple
2. **Given** input "https://github.com/org/repo/pull/99" with empty issue body, **When** Classify is called, **Then** the returned TaskProfile has domain=feature (default for PRs) and complexity=medium
3. **Given** input "https://github.com/org/repo/issues/100" with body containing "refactor the entire persistence layer to use repository pattern across 12 packages", **When** Classify is called, **Then** the returned TaskProfile has complexity=architectural, domain=refactor

---

### User Story 3 - Select Pipeline from TaskProfile (Priority: P1)

Given a classified TaskProfile, the system selects the appropriate Wave pipeline name and configuration. The mapping follows the routing table defined in AGENTS.md.

**Why this priority**: Pipeline selection is the direct consumer of classification output. Without it, classification has no actionable effect in the Wave pipeline orchestrator.

**Independent Test**: Can be tested by constructing TaskProfile values directly and asserting correct PipelineConfig output. No dependency on input analysis.

**Acceptance Scenarios**:

1. **Given** a TaskProfile with complexity=simple and domain=bug, **When** SelectPipeline is called, **Then** the returned PipelineConfig names the "impl-issue" pipeline
2. **Given** a TaskProfile with complexity=complex and domain=feature, **When** SelectPipeline is called, **Then** the returned PipelineConfig names the "impl-speckit" pipeline
3. **Given** a TaskProfile with domain=security (any complexity), **When** SelectPipeline is called, **Then** the returned PipelineConfig names the "audit-security" pipeline
4. **Given** a TaskProfile with domain=docs, **When** SelectPipeline is called, **Then** the returned PipelineConfig names the "doc-fix" pipeline

---

### User Story 4 - End-to-End Classification and Routing (Priority: P2)

A Wave user provides raw input (text or URL) and the system performs classification followed by pipeline selection in a single logical flow, returning both the TaskProfile and the selected PipelineConfig.

**Why this priority**: Combines Stories 1-3 into the integration path that Wave's orchestrator will actually invoke. Lower priority because the individual components are independently testable and valuable.

**Independent Test**: Can be tested by providing raw input strings and verifying that the final pipeline selection matches expectations for that input type.

**Acceptance Scenarios**:

1. **Given** input "fix null pointer in logger", **When** the full classify-and-select flow runs, **Then** pipeline "impl-issue" is selected with complexity=simple, domain=bug
2. **Given** input "implement new GraphQL API layer with auth, rate limiting, and caching", **When** the full flow runs, **Then** pipeline "impl-speckit" is selected with complexity=complex or architectural

---

### Edge Cases

- What happens when input is empty or whitespace-only? System MUST return a sensible default profile (complexity=simple, domain=feature, blast_radius=0.1, verification_depth=structural_only) rather than error.
- What happens when input contains mixed signals (e.g., "fix security bug in docs")? System MUST use domain priority ordering: security > performance > bug > refactor > feature > docs.
- What happens when the issue body contradicts the URL type (e.g., PR URL but body describes a bug fix)? The body content MUST take precedence over URL type for domain classification, but URL type informs the default when body is ambiguous.
- What happens when input contains no recognizable keywords? System MUST fall back to complexity=medium, domain=feature as the safe default.
- What happens when blast_radius cannot be determined? System MUST default to 0.5 (moderate risk) to avoid under-classifying risky changes.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST define a TaskProfile type with fields: blast_radius (float64, range 0.0-1.0), complexity (string enum: simple/medium/complex/architectural), domain (string enum: security/performance/docs/feature/bug/refactor/research), verification_depth (string enum: structural_only/behavioral/full_semantic), input_type (reused InputType from internal/suggest: issue_url/pr_url/repo_ref/free_text)
- **FR-002**: System MUST implement Classify(input string, issueBody string) TaskProfile that analyzes both the direct input string and optional issue body text
- **FR-003**: System MUST reuse URL and repository reference detection by calling suggest.ClassifyInput(input) from internal/suggest/input.go to identify input types before keyword analysis. The new package's analyzer function MUST be named Classify (not ClassifyInput) to avoid confusion with the existing suggest.ClassifyInput
- **FR-004**: System MUST perform keyword-based analysis on input text to determine complexity, domain, blast_radius, and verification_depth
- **FR-005**: System MUST implement SelectPipeline(profile TaskProfile) PipelineConfig that maps task profiles to pipeline names
- **FR-006**: System MUST map profiles to pipelines according to the AGENTS.md routing table. Selection uses input_type first, then domain, then complexity: (a) PR URLs always→ops-pr-review regardless of domain/complexity, (b) domain=security (any complexity)→audit-security, (c) domain=research→impl-research, (d) domain=docs→doc-fix, (e) domain=refactor with complexity=architectural→impl-speckit, (f) complexity=simple or medium→impl-issue, (g) complexity=complex or architectural→impl-speckit. This covers: simple/medium bugs→impl-issue, complex features→impl-speckit, medium features→impl-issue
- **FR-007**: System MUST provide table-driven unit tests for all three components (profile type validation, input analysis, pipeline selection)
- **FR-008**: System MUST export all public types and functions from the internal/classify package for use by other internal packages
- **FR-009**: blast_radius MUST be derived from complexity and domain signals: security and architectural changes score higher (>0.6), docs and simple fixes score lower (<0.3)
- **FR-010**: verification_depth MUST be derived from complexity: simple→structural_only, medium→behavioral, complex/architectural→full_semantic
- **FR-011**: System MUST apply domain priority ordering when multiple domains are detected: security > performance > bug > refactor > feature > docs
- **FR-012**: System MUST handle the "research" domain signal for inputs requesting investigation, analysis, or comparison — routing to impl-research pipeline

### Key Entities

- **TaskProfile**: The central classification output. Represents a structured assessment of a task's characteristics along four dimensions. Consumed by pipeline selection and potentially by step-level complexity routing.
- **PipelineConfig**: The output of pipeline selection. A struct with two fields: Name (string, the pipeline name e.g. "impl-issue") and Reason (string, a short human-readable explanation of why this pipeline was selected, e.g. "simple bug fix routed to impl-issue").
- **InputType**: Reused from internal/suggest/input.go. Represents the detected format of user input (issue URL, PR URL, repo reference, or free text).

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Classify correctly classifies at least 90% of test cases across all domain categories (security, performance, docs, feature, bug, refactor) as validated by table-driven unit tests
- **SC-002**: SelectPipeline returns the correct pipeline for 100% of the defined routing table mappings in AGENTS.md
- **SC-003**: All three package files (profile.go, analyzer.go, selector.go) have corresponding _test.go files with at least 10 table-driven test cases each
- **SC-004**: Empty, whitespace, and ambiguous inputs produce valid TaskProfile values (no panics, no zero-value structs) with sensible defaults
- **SC-005**: The classify package introduces no new external dependencies beyond the Go standard library and existing internal packages
- **SC-006**: Classification of a single input completes in under 1 millisecond (no network calls, no disk I/O during classification)

## Clarifications

### CLR-001: Function name collision with suggest.ClassifyInput
**Question**: The spec originally named the analyzer function `ClassifyInput`, which collides with `suggest.ClassifyInput` in `internal/suggest/input.go`. Should the new function have a different name?
**Resolution**: Renamed to `Classify(input string, issueBody string) TaskProfile`. The `classify` package name provides sufficient context (`classify.Classify`). This avoids confusion with `suggest.ClassifyInput` which returns `InputType`, not `TaskProfile`.
**Rationale**: Go convention — the package name qualifies the function. `classify.Classify` is idiomatic; `classify.ClassifyInput` would stutter.

### CLR-002: "research" domain missing from FR-001 enum
**Question**: FR-012 references a "research" domain for routing to impl-research, but FR-001's domain enum only listed 6 values (security/performance/docs/feature/bug/refactor). Should research be a domain?
**Resolution**: Added `research` to the domain enum in FR-001. Research is a distinct routing target (impl-research pipeline) and needs its own domain value.
**Rationale**: AGENTS.md explicitly lists "Research then implement → impl-research" as a routing path. Without the domain value, FR-012 would be unimplementable.

### CLR-003: Incomplete routing matrix in FR-006
**Question**: FR-006 only specified a few example mappings (simple bugs→impl-issue, complex features→impl-speckit). What about medium bugs, medium features, architectural refactors, etc.?
**Resolution**: Expanded FR-006 with a complete priority-ordered selection algorithm: input_type first (PR URLs always→ops-pr-review), then domain overrides (security, research, docs), then complexity-based routing (simple/medium→impl-issue, complex/architectural→impl-speckit).
**Rationale**: Matches AGENTS.md routing table where simple+medium both route to impl-issue and complex routes to impl-speckit, as confirmed by the impl-smart-route.yaml branch step.

### CLR-004: PipelineConfig structure underspecified
**Question**: PipelineConfig was described as "at minimum the pipeline name... May include additional metadata." What concrete fields?
**Resolution**: Defined PipelineConfig as a struct with Name (string) and Reason (string). Name is the pipeline identifier, Reason explains the routing decision.
**Rationale**: Keeping it minimal (two fields) matches the current codebase pattern where pipeline selection only needs a name. The Reason field aids debugging and is consistent with the impl-smart-route.yaml assessment step which produces a reason field.

### CLR-005: InputType-based vs domain-based routing interaction
**Question**: FR-006 says "PRs→ops-pr-review" but PR isn't a domain — it's an InputType. How do InputType and domain interact in routing?
**Resolution**: Added `input_type` field to TaskProfile (FR-001). SelectPipeline checks input_type first: PR URLs short-circuit to ops-pr-review regardless of domain/complexity. For all other input types, domain and complexity drive selection.
**Rationale**: PR review is fundamentally about the input format (a PR URL), not the content domain. This matches `suggest.SuggestPipelineForInput` which routes PR URLs directly to ops-pr-review without content analysis.
