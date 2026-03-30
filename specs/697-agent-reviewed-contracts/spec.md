# Feature Specification: Agent-Reviewed Contracts

**Feature Branch**: `697-agent-reviewed-contracts`
**Created**: 2026-03-30
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/697

## User Scenarios & Testing _(mandatory)_

### User Story 1 — Pipeline author adds agent review to a step (Priority: P1)

A pipeline author wants to add a quality gate where a separate agent reviews the output of a step before the pipeline proceeds. They add an `agent_review` contract to their step's handover block, specifying which persona should review, what criteria to evaluate against, and what context the reviewer needs.

**Why this priority**: This is the core capability — without it, nothing else in the epic functions. It delivers the fundamental value proposition: catching substance issues that mechanical checks miss.

**Independent Test**: Can be fully tested by configuring a pipeline step with an `agent_review` contract, running the pipeline, and verifying that a separate agent is spawned to evaluate the output against the specified criteria, returning a structured verdict.

**Acceptance Scenarios**:

1. **Given** a pipeline step with an `agent_review` contract specifying a reviewer persona, criteria path, and context sources, **When** the step completes execution, **Then** a separate agent (using the reviewer persona) evaluates the step output and returns a structured verdict (pass/fail with issues and suggestions).
2. **Given** an `agent_review` contract where the reviewer persona is the same as the step persona, **When** the pipeline is loaded, **Then** validation fails with a clear error indicating self-review is not allowed.
3. **Given** an `agent_review` contract with a model override (e.g., `claude-haiku`), **When** the review agent is spawned, **Then** it uses the specified model regardless of the step's model configuration.
4. **Given** an `agent_review` contract with a token budget, **When** the review exceeds the budget, **Then** the review is terminated and treated as a failure with an appropriate error message.

---

### User Story 2 — Review failure triggers rework with feedback (Priority: P1)

When an agent review fails, the pipeline author wants the review feedback (issues found, suggestions for improvement) to be fed into a rework step so the implementing agent can fix the problems without starting from scratch.

**Why this priority**: Rework-with-feedback is what makes agent review actionable rather than just a gate. Without it, a failed review simply blocks the pipeline with no path to resolution.

**Independent Test**: Can be tested by configuring an `agent_review` with `on_failure: rework` and a `rework_step`, intentionally producing flawed output, and verifying the rework step receives structured feedback as an injectable artifact.

**Acceptance Scenarios**:

1. **Given** an `agent_review` contract with `on_failure: rework` and a configured `rework_step`, **When** the review returns a "fail" verdict, **Then** the review feedback is written as a structured artifact and injected into the rework step's context.
2. **Given** a rework step that receives review feedback, **When** the rework step executes, **Then** its prompt includes the specific issues and suggestions from the review, enabling targeted fixes.
3. **Given** a rework step that successfully addresses the review feedback, **When** the rework step completes, **Then** the agent review runs again on the rework output to verify the fixes. The review→rework→re-review cycle is bounded by the contract's `max_retries` field (default 1 cycle); if the re-review fails again and retries are exhausted, the contract's `on_failure` policy applies.

---

### User Story 3 — Multiple contracts run in sequence on a single step (Priority: P2)

A pipeline author wants to chain multiple contracts on a single step — mechanical checks first (fast, cheap), then agent review (slower, more expensive). If an early contract fails, later contracts are skipped to save cost.

**Why this priority**: This enables the "cheap first, expensive second" principle. Without it, authors must choose between mechanical validation and agent review rather than composing them.

**Independent Test**: Can be tested by configuring a step with multiple contracts (e.g., `test_suite` then `agent_review`), and verifying execution order, early termination on failure, and independent `on_failure` policies per contract.

**Acceptance Scenarios**:

1. **Given** a step with an ordered list of contracts `[test_suite, agent_review]`, **When** the step completes, **Then** contracts execute in the specified order.
2. **Given** a step with contracts `[test_suite, agent_review]` where the test suite fails, **When** contract validation runs, **Then** the agent review is skipped entirely (no tokens spent).
3. **Given** a step with multiple contracts each having independent `on_failure` policies, **When** a contract fails, **Then** the failure is handled according to that specific contract's policy, not a global policy.
4. **Given** a step using the existing singular `contract` field (not a list), **When** the pipeline is loaded, **Then** it continues to work exactly as before with no behavior change.

---

### User Story 4 — Review context includes git diff and artifacts (Priority: P2)

The reviewing agent needs relevant context to make informed judgments. The pipeline author specifies which artifacts and sources (such as the git diff of changes made during the step) are provided to the reviewer.

**Why this priority**: Review quality depends entirely on context quality. Without configurable context, the reviewer either sees too much (wasting tokens) or too little (missing issues).

**Independent Test**: Can be tested by configuring an `agent_review` with `context` entries referencing artifacts and `git_diff`, and verifying the reviewer receives the specified context in its prompt.

**Acceptance Scenarios**:

1. **Given** an `agent_review` with `context: [{source: git_diff}]`, **When** the review agent is spawned, **Then** it receives the uncommitted diff from the step's workspace (truncated at a configurable limit).
2. **Given** an `agent_review` with `context: [{artifact: assessment}, {artifact: impl-plan}]`, **When** the review agent is spawned, **Then** it receives the content of those artifacts from prior steps.
3. **Given** a git diff that exceeds the configured size limit (default 50KB), **When** the diff is prepared for the reviewer, **Then** it is truncated with a clear indication that truncation occurred.

---

### User Story 5 — Review feedback carries structured data (Priority: P2)

The contract result from an agent review carries structured feedback — not just pass/fail, but specific issues, suggestions, and a confidence score — so that rework steps, dashboards, and retrospectives can consume it programmatically.

**Why this priority**: Structured feedback enables the downstream consumers (rework, observability) to function. Without structure, feedback is opaque text that cannot be parsed or acted on.

**Independent Test**: Can be tested by running an agent review and verifying the returned result contains structured fields (verdict, issues list, suggestions list, confidence score) that can be serialized and deserialized.

**Acceptance Scenarios**:

1. **Given** an agent review that identifies issues, **When** the review completes, **Then** the contract result includes a `ReviewFeedback` structure with: verdict (pass/fail/warn), a list of issues (each with severity and description), a list of suggestions, and a confidence score (0.0–1.0).
2. **Given** an agent review that passes, **When** the review completes, **Then** the `ReviewFeedback` has verdict "pass", an empty issues list, and a confidence score above the configured threshold.

---

### User Story 6 — Review verdicts visible in dashboard and retros (Priority: P3)

Operations teams want to see agent review verdicts in the web dashboard's run detail page and track review-triggered rework as a friction type in retrospectives.

**Why this priority**: Observability validates the system end-to-end and enables continuous improvement of review criteria. Lower priority because the core system works without it.

**Independent Test**: Can be tested by running a pipeline with agent review, then checking the web dashboard for review verdict display and the retrospective output for review-related friction tracking.

**Acceptance Scenarios**:

1. **Given** a completed pipeline run with agent review steps, **When** a user views the run detail page, **Then** review verdicts (pass/fail, issue count, reviewer persona, token spend) are displayed per step.
2. **Given** a pipeline run where agent review triggered rework, **When** a retrospective is generated, **Then** it includes `review_rework` as a friction type with the associated review feedback.
3. **Given** multiple pipeline runs over time, **When** a user views analytics, **Then** they can see aggregate review token spend and review pass/fail rates.

---

### User Story 7 — Wave's own pipelines use agent review (Priority: P3)

Wave's built-in pipelines (`impl-issue`, `impl-speckit`, `ops-pr-review`) are upgraded with `agent_review` contracts to validate that the system works in production and to catch quality issues in Wave's own development workflow.

**Why this priority**: This is the integration test for the entire system. Lower priority because it depends on all other stories being complete.

**Independent Test**: Can be tested by running the upgraded Wave pipelines on real issues and verifying that agent reviews execute, provide meaningful feedback, and achieve a false-positive rate below 20%.

**Acceptance Scenarios**:

1. **Given** the `impl-issue` pipeline with an `agent_review` contract on its implementation step, **When** a real issue is implemented, **Then** the navigator persona reviews the implementation and provides substantive feedback.
2. **Given** Wave's upgraded pipelines running in production for a representative sample of issues, **When** review results are analyzed, **Then** the false-positive rate (reviews that flag correct implementations) is below 20%.

---

### Edge Cases

- What happens when the reviewer agent crashes or times out mid-review? The review is treated as a failure, and the contract's `on_failure` policy governs the next action (retry, rework, fail, skip).
- What happens when referenced context artifacts do not exist? The review proceeds with available context, but emits a warning event noting the missing artifact.
- What happens when the reviewer produces output that cannot be parsed into structured feedback? The review is treated as a failure with a parse error. The raw output is preserved for debugging.
- What happens when the review criteria file does not exist at the specified path? Pipeline validation fails at load time with a clear error message.
- What happens when multiple contracts are configured and one uses `on_failure: rework` while a later one uses `on_failure: fail`? Each contract's `on_failure` is independent. If a contract triggers rework, ALL contracts in the list re-run from the beginning on the rework output (see Clarification C4). Each full re-run counts as one retry attempt. If a later contract fails with `on_failure: fail`, the pipeline fails regardless of earlier successes.
- What happens when the token budget is set to zero or negative? Validation fails at pipeline load time with a clear error.
- What happens when the git diff is empty (no changes made)? The reviewer receives an indication that no changes were made, which itself may be a review finding (e.g., no-op implementation).

## Clarifications

### C1: Relationship between `agent_review` and `llm_judge` contract types

**Question**: The codebase already has an `llm_judge` contract type (`internal/contract/llm_judge.go`) that performs LLM-based evaluation with criteria, model pinning, threshold scoring, and structured `JudgeResponse` output via a single API call. How does the new `agent_review` contract type relate to `llm_judge`?

**Resolution**: They are **complementary, not competing** contract types for different use cases:
- **`llm_judge`** — lightweight, stateless evaluation. Single API call or CLI invocation. No tools, no workspace access, no persona. Best for: schema compliance, style checks, output quality scoring against simple criteria. Cost: minimal tokens.
- **`agent_review`** — heavyweight, persona-driven evaluation. Spawns a full agent via the adapter runner with tools, workspace access, and a persona system prompt. Can read files, navigate code, run commands. Best for: substantive implementation review, catching wrong-approach or no-op PRs. Cost: higher token spend.

Pipeline authors choose based on review depth needed. A step can chain both via the plural `contracts` list (e.g., `llm_judge` first for cheap checks, then `agent_review` for deep review).

**Rationale**: Follows the existing pattern where contract types are specialized validators (`json_schema`, `test_suite`, `format`, `llm_judge`) each targeting a specific validation need. `agent_review` fills the gap for reviews requiring tool use and codebase navigation.

### C2: Reviewer agent execution environment

**Question**: FR-016 says "pass the adapter runner into the contract validation path" but the spec doesn't specify: What workspace does the reviewer agent get? What tools and permissions does it have? How is its prompt structured?

**Resolution**: The reviewer agent runs in the **step's workspace in read-only mode**:
- **Workspace**: The reviewer receives the same workspace path as the step that produced the output. The workspace is not copied — the reviewer operates on the same directory but the persona's permissions should restrict write operations (deny `Edit`, `Write` tools unless explicitly allowed in the reviewer persona).
- **Adapter execution**: The `agent_review` validator calls `adapter.Run()` with an `AdapterRunConfig` built from the reviewer persona's configuration (system prompt, permissions, model override from the contract).
- **Prompt structure**: The reviewer's user prompt is assembled from: (1) the review criteria (loaded from `criteria_path`), (2) the assembled context sources (artifacts, git_diff), and (3) the `ReviewFeedback` JSON schema as the required output format. This follows the same pattern as `buildContractPrompt()` in `executor.go`.
- **System prompt**: Standard persona system prompt from the reviewer persona definition, assembled via the same CLAUDE.md layering as any other step (base protocol + persona + restrictions).

**Rationale**: Matches the existing adapter execution model. Read-only workspace access prevents the reviewer from modifying the implementation, maintaining separation of concerns.

### C3: Review→rework→re-review loop bounds

**Question**: User Story 2, Acceptance Scenario 3 states "the agent review runs again on the rework output to verify the fixes." What bounds the review→rework→re-review loop? Can it cycle indefinitely?

**Resolution**: The review→rework→re-review loop is bounded by the contract's existing `max_retries` field:
- Each rework-then-re-review cycle counts as one retry attempt.
- **Default**: `max_retries: 1` — one rework attempt, one re-review. If the re-review fails again, the contract's `on_failure` policy applies (typically `fail`).
- Pipeline authors can increase `max_retries` for more cycles (e.g., `max_retries: 3` allows up to 3 rework→re-review iterations).
- This reuses the existing `RetryConfig.MaxAttempts` semantics from `internal/pipeline/types.go:133` rather than introducing a new field.

**Rationale**: Reuses the existing retry infrastructure. Unbounded loops risk runaway token spend; the `max_retries` field already has validation and is well-understood by pipeline authors.

### C4: Contract list execution after rework

**Question**: Edge case 5 describes rework triggered by one contract in a list, then says "subsequent contracts run on the rework output." But after rework modifies the output, do all contracts re-run from the beginning, or only the remaining ones?

**Resolution**: After a rework step completes, **all contracts in the list re-run from the beginning**:
- The rework may have changed the output in ways that invalidate earlier contract results (e.g., fixing a review issue might break a schema contract).
- This matches CI pipeline semantics where a retry re-runs the full validation suite.
- Each full re-run of the contract list counts as one retry attempt against the triggering contract's `max_retries`.
- If a contract that previously passed now fails on the rework output, its own `on_failure` policy governs.

**Rationale**: Re-running all contracts is the safe default that prevents silent regressions. The cost is acceptable because mechanical contracts (schema, test_suite) are fast and cheap.

### C5: ReviewFeedback extraction from agent output

**Question**: FR-008 defines the `ReviewFeedback` structure but doesn't specify how the reviewing agent's freeform output is parsed into this structure. How does the system ensure the reviewer produces valid structured feedback?

**Resolution**: Follow the established `llm_judge` pattern (`internal/contract/llm_judge.go:106-124`):
- The `ReviewFeedback` JSON schema is **injected into the reviewer's user prompt** as a required output format, alongside the review criteria and context.
- After the reviewer agent completes, the `agent_review` validator **parses stdout** for JSON matching the `ReviewFeedback` schema, using the same `extractJSON()` helper that strips markdown fences.
- If parsing fails, the review is treated as a failure with a parse error (matching edge case 3: "raw output is preserved for debugging").
- The `ReviewFeedback` is serialized as **JSON** — consistent with all other contract artifacts and the `JudgeResponse` format used by `llm_judge`.

**Rationale**: Proven pattern from `llm_judge`. Injecting the schema into the prompt gives the LLM clear structure to follow. The `extractJSON` helper already handles common LLM output quirks (markdown fences, whitespace).

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST support an `agent_review` contract type that spawns a separate agent (via the adapter runner) to evaluate step output against configurable review criteria. This is distinct from `llm_judge` — see Clarification C1.
- **FR-002**: System MUST enforce that the reviewer persona differs from the step's executing persona (self-review prevention).
- **FR-003**: System MUST support model pinning for the reviewer agent (e.g., `model: claude-haiku`) independent of the step's model configuration.
- **FR-004**: System MUST enforce a configurable token budget for agent reviews, terminating the review if the budget is exceeded.
- **FR-005**: System MUST support `criteria_path` pointing to a markdown file containing the review criteria, validated at pipeline load time.
- **FR-006**: System MUST support configurable context sources for the reviewer, including named artifacts from prior steps and `git_diff` as an automatic source.
- **FR-007**: System MUST provide `git_diff` as a context source that captures the uncommitted diff from the step's workspace, truncated at a configurable size limit (default 50KB).
- **FR-008**: System MUST extend the contract result to carry structured `ReviewFeedback` containing: verdict (pass/fail/warn), issues (each with severity and description), suggestions, and confidence score (0.0–1.0). The reviewer's output is parsed from JSON in stdout following the `llm_judge` extraction pattern — see Clarification C5.
- **FR-009**: System MUST support `on_failure: rework` for agent review contracts, writing `ReviewFeedback` as a structured JSON artifact and injecting it into the designated rework step. The review→rework→re-review loop is bounded by `max_retries` — see Clarification C3.
- **FR-010**: System MUST support an ordered list of contracts per step (`contracts` field, plural), executing them in sequence with early termination when a contract fails (unless its `on_failure` policy allows continuation). After rework, all contracts re-run from the beginning — see Clarification C4.
- **FR-011**: System MUST maintain backward compatibility with the existing singular `contract` field — pipelines using the singular form continue to work without modification.
- **FR-012**: Each contract in a list MUST support its own independent `on_failure` policy.
- **FR-013**: System MUST emit progress events for agent review lifecycle: review started, review completed (with verdict), review failed.
- **FR-014**: System MUST display review verdicts (verdict, issue count, reviewer, token spend) in the web dashboard's run detail page.
- **FR-015**: System MUST track `review_rework` as a friction type in pipeline retrospectives when agent review triggers rework.
- **FR-016**: System MUST pass the adapter runner into the contract validation path so that `agent_review` contracts can spawn agents. The reviewer runs in the step's workspace with read-only tool permissions — see Clarification C2.

### Key Entities

- **ReviewFeedback**: The structured JSON output of an agent review — contains verdict (pass/fail/warn), issues (each with severity and description), suggestions, and confidence score (0.0–1.0). Extracted from the reviewer agent's stdout using the `extractJSON` pattern established by `llm_judge`. First-class artifact that can be serialized, injected into rework steps, displayed in dashboards, and tracked in retros.
- **AgentReviewContract**: A contract configuration that specifies: reviewer persona, model, criteria path, context sources, token budget, and timeout. Extends the existing `ContractConfig` type in `internal/pipeline/types.go`. Distinct from `llm_judge` in that it spawns a full agent with tools and workspace access rather than making a single API call.
- **ReviewContext**: The assembled context provided to the reviewer agent — composed from specified artifacts and sources (e.g., git diff). Built at review time from the contract's `context` configuration and injected into the reviewer's user prompt alongside the criteria and output schema.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Agent review contracts catch at least one substantive issue (wrong implementation, missing requirement, no-op PR) that mechanical contracts (schema, tests) would not catch, demonstrated across a representative sample of pipeline runs.
- **SC-002**: Agent review adds less than $0.02 per pipeline step in token cost when using a cost-efficient model (e.g., Haiku) on medium-sized diffs (5–20KB).
- **SC-003**: All existing pipelines using the singular `contract` field continue to work without any configuration changes after the feature is deployed.
- **SC-004**: The false-positive rate (reviews that flag correct implementations as failures) is below 20% when using Wave's own review criteria on Wave's own pipelines.
- **SC-005**: Review feedback in rework steps leads to successful fixes (rework step passes subsequent review) at least 60% of the time without human intervention.
- **SC-006**: Pipeline authors can add agent review to an existing step by adding fewer than 10 lines of YAML configuration.
- **SC-007**: Agent reviews complete within 60 seconds for diffs under 20KB using a cost-efficient model.
