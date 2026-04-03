# Research: Agent-Reviewed Contracts

**Feature**: #697 Agent-Reviewed Contracts
**Date**: 2026-03-30

## Decision 1: Validator Interface Extension for Adapter Access

**Context**: The current `ContractValidator` interface (`internal/contract/contract.go:70-72`) has signature `Validate(cfg ContractConfig, workspacePath string) error`. The `agent_review` contract type needs access to an `adapter.AdapterRunner` to spawn a reviewer agent. The existing `llm_judge` avoids this by calling the Anthropic API directly or shelling out to the CLI.

**Decision**: Introduce a new `AgentContractValidator` interface alongside the existing one, and a `ValidateWithRunner()` top-level function that passes the runner to validators that need it. The `agentReviewValidator` stores the runner as a field, set at construction time via a new `NewAgentValidator(cfg, runner)` factory.

**Rationale**: Adding the runner to `ContractConfig` would couple a transport-layer concern (adapter execution) to a data structure. A separate interface keeps the existing validators untouched while enabling agent-powered validators. The executor already holds the adapter runner, so passing it through is trivial.

**Alternatives Rejected**:
- *Extend ContractConfig with runner field*: Mixes configuration data with runtime dependencies. Would require all callers to populate the field even for non-agent contracts.
- *Global adapter registry*: Introduces hidden global state, harder to test, violates constructor injection pattern used elsewhere.
- *Embed adapter logic directly in executor*: Would bloat executor.go further (already 4000+ lines) and prevent reuse of the validator pattern.

## Decision 2: Plural Contracts List (`contracts` field)

**Context**: `HandoverConfig` (`types.go:417-422`) currently has a singular `Contract ContractConfig`. The spec requires an ordered list of contracts per step (FR-010) with backward compatibility for the singular form (FR-011).

**Decision**: Add `Contracts []ContractConfig` to `HandoverConfig`. The executor resolves the effective contract list as: if `Contracts` is non-empty, use it; otherwise if `Contract.Type` is set, wrap it in a single-element slice. This "sugar → canonical" normalization happens once before the validation loop.

**Rationale**: The singular form is sugar over the list form. Normalizing early means the validation loop has one code path. YAML supports both `contract:` (singular object) and `contracts:` (list) naturally.

**Alternatives Rejected**:
- *Replace singular with plural only*: Breaks all existing pipelines. The spec explicitly requires backward compatibility (FR-011, SC-003).
- *Auto-migrate YAML files*: Unnecessary churn. The normalization is trivial at runtime.

## Decision 3: ReviewFeedback Structure and Extraction

**Context**: The spec requires a `ReviewFeedback` type (FR-008) extracted from the reviewer agent's stdout using the `extractJSON` pattern from `llm_judge.go:306-319`.

**Decision**: Define `ReviewFeedback` in `internal/contract/agent_review.go` with fields: `Verdict` (string: pass/fail/warn), `Issues` ([]ReviewIssue with Severity+Description), `Suggestions` ([]string), `Confidence` (float64). Extract from agent stdout using the existing `extractJSON()` helper. Serialize as JSON artifact for rework injection.

**Rationale**: Mirrors the `JudgeResponse` pattern. JSON extraction is proven. The struct is the minimum viable feedback structure that satisfies dashboards, rework, and retros.

**Alternatives Rejected**:
- *Reuse JudgeResponse directly*: Different shape — JudgeResponse has per-criterion results, ReviewFeedback has issues/suggestions. Different consumers (threshold evaluation vs. rework injection).
- *Freeform markdown feedback*: Cannot be programmatically consumed by rework steps, dashboards, or retros. Violates FR-008 structured requirement.

## Decision 4: Context Assembly and git_diff Source

**Context**: FR-006/FR-007 require configurable context sources including artifacts and `git_diff`. The reviewer needs this context injected into its user prompt.

**Decision**: Define `ReviewContextSource` struct with `Source` (string: "git_diff", "artifact") and `Artifact` (string: artifact name). At review time, assemble context by:
1. For `git_diff`: Run `git diff HEAD` in the step's workspace, truncate at configurable limit (default 50KB), include truncation notice.
2. For artifacts: Read from `execution.ArtifactPaths` by step:artifact key.
Assembled context is concatenated into the reviewer's user prompt between criteria and output schema sections.

**Rationale**: `git diff HEAD` captures all uncommitted changes (staged and unstaged) which is exactly what a step produces. The truncation limit prevents token budget blowout. Reading artifacts from execution state is how all artifact injection already works.

**Alternatives Rejected**:
- *Pass file paths to reviewer*: Reviewer would need file-reading tools, increasing complexity and token spend. Direct injection is cheaper and more reliable.
- *Diff against branch base*: Would include changes from prior steps, not just this step's changes. `git diff HEAD` in the workspace is scoped correctly.

## Decision 5: Rework Integration with Contract-Level Failure

**Context**: The existing rework mechanism lives in `RetryConfig` (step-level retry). The spec requires contract-level `on_failure: rework` (FR-009) where the review feedback is injected as an artifact into the rework step. Additionally, after rework, ALL contracts re-run from the beginning (C4).

**Decision**: Each `ContractConfig` already has `OnFailure` and `MaxRetries` fields. For `agent_review` contracts with `on_failure: rework`:
1. Write `ReviewFeedback` as JSON to `.wave/artifacts/review_feedback.json` in the workspace.
2. Trigger the rework step (reuse `executeReworkStep` with enhanced context including the ReviewFeedback path).
3. After rework completes, re-run ALL contracts from the beginning of the list. Each full re-run counts as one retry against the triggering contract's `max_retries`.
4. Add `ReworkStep` field to `ContractConfig` (paralleling `RetryConfig.ReworkStep`).

**Rationale**: Reuses the proven rework machinery. Writing feedback as a file artifact follows the existing artifact injection model. Re-running all contracts after rework is the safe default per spec clarification C4.

**Alternatives Rejected**:
- *New rework mechanism separate from existing one*: Duplicates logic. The existing `executeReworkStep` already handles workspace copying, artifact re-registration, and state transitions.
- *Only re-run the failed contract*: Rework may invalidate earlier contract results (C4). Full re-run is the safe default.

## Decision 6: Self-Review Prevention Validation

**Context**: FR-002 requires that the reviewer persona differs from the step's executing persona.

**Decision**: Add validation in DAG validation (`internal/pipeline/dag.go` or `validation.go`) at pipeline load time. For each `agent_review` contract, check that `contract.Persona != step.Persona`. This is a hard error that blocks pipeline execution.

**Rationale**: Fail-fast at load time, before any tokens are spent. Follows the pattern of existing DAG validation checks (e.g., rework target validation).

## Decision 7: Token Budget Enforcement

**Context**: FR-004 requires configurable token budgets for reviews.

**Decision**: Add `TokenBudget` field to the `agent_review` contract config. The reviewer's `AdapterRunConfig` is configured with a timeout derived from the budget (using model-specific tokens-per-second estimates). If the adapter result's `TokensUsed` exceeds the budget, the review is treated as a failure. Validation at load time rejects zero or negative budgets.

**Rationale**: The adapter already reports `TokensUsed` in `AdapterResult`. Budget enforcement post-execution is simpler than mid-stream termination. The timeout provides a rough real-time bound.

## Decision 8: WebUI and Retro Integration

**Context**: FR-013/FR-014/FR-015 require progress events, dashboard display, and retro friction tracking.

**Decision**:
- **Events**: Add `review_started`, `review_completed`, `review_failed` states to event emission in the contract validation loop. Include reviewer persona, verdict, issue count, and token spend in event fields.
- **WebUI**: Add `ReviewVerdict`, `ReviewIssueCount`, `ReviewerPersona`, `ReviewTokens` fields to `StepDetail` in `webui/types.go`. Populate from step events in the runs handler.
- **Retro**: Add `FrictionReviewRework` friction type to `retro/types.go`. The retro generator checks for `review_rework` events and creates friction points with the review feedback detail.

**Rationale**: Follows the existing patterns for each subsystem. Events drive both dashboard and retro — single source of truth.
