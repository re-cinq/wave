# Requirements Quality Review: Agent-Reviewed Contracts

**Feature**: #697 | **Date**: 2026-03-30

## Completeness

- [ ] CHK001 - Does the spec define what happens when both `contract` and `contracts` are set AND the singular contract has `on_failure: rework`? The EffectiveContracts() method silently drops the singular — is a warning sufficient or should this be an error? [Completeness]
- [ ] CHK002 - Does the spec define the content and format of the reviewer's system prompt beyond "standard persona system prompt"? C2 says CLAUDE.md layering, but is the reviewer's CLAUDE.md assembly documented enough for an implementer to build without guesswork? [Completeness]
- [ ] CHK003 - Does the spec define what `AdapterRunConfig` fields are populated for the reviewer agent? The plan mentions "persona config from the manifest" but the specific mapping from `ContractConfig` fields to `AdapterRunConfig` fields is not enumerated [Completeness]
- [ ] CHK004 - Does the spec define the artifact name/key used when ReviewFeedback is written for rework injection? The plan says `review_feedback.json` but the spec doesn't specify whether this is a fixed name or contract-index-specific (important for plural contracts where multiple agent_reviews could each trigger rework) [Completeness]
- [ ] CHK005 - Does the spec define behavior when the adapter runner is unavailable or misconfigured for the reviewer persona? E.g., the reviewer persona references an adapter that isn't installed [Completeness]
- [ ] CHK006 - Does the spec define how `threshold` interacts with `verdict`? The contract config YAML reference shows a `threshold` field for confidence-based pass/fail, but the spec's acceptance scenarios only test verdict-based pass/fail. Is a `warn` verdict with confidence above threshold a pass or fail? [Completeness]

## Clarity

- [ ] CHK007 - Is the distinction between `Timeout` (contract config) and the adapter's own timeout unambiguous? The spec says "review is terminated" on timeout but doesn't clarify whether this is the contract-level timeout or the adapter's built-in timeout, or how they interact if both are set [Clarity]
- [ ] CHK008 - Is the meaning of `max_retries` for rework loops clear when multiple agent_review contracts in the same list each have their own `max_retries`? Does one contract's retry counter affect another's? [Clarity]
- [ ] CHK009 - Is the `on_failure: rework` behavior when the triggering contract is NOT `agent_review` defined? FR-009 describes rework for agent_review, but `on_failure: rework` could theoretically appear on any contract type — is this intentionally limited to agent_review or general? [Clarity]
- [ ] CHK010 - Is "read-only mode" for the reviewer workspace precisely defined? C2 says "deny Edit, Write tools unless explicitly allowed in the reviewer persona" — does this mean read-only is a convention enforced by persona config, not a filesystem-level enforcement? [Clarity]
- [ ] CHK011 - Is the relationship between `contract.Model` (existing field from llm_judge) and `agent_review.Model` unambiguous? Both use the same `Model` field on `ContractConfig` but for different purposes (API model vs adapter model) [Clarity]

## Consistency

- [ ] CHK012 - Are the `ReviewContextSource` types consistent between the contract package and pipeline package? The data model shows the type in `internal/contract` but the pipeline types.go also needs it for YAML deserialization — is there a single source of truth or are they duplicated? [Consistency]
- [ ] CHK013 - Is the `on_failure` enum consistent across singular contracts, plural contracts, and the existing `RetryConfig`? The spec references "fail", "skip", "continue", "rework", "retry" — do all these map cleanly to the existing `OnFailure` constants (`OnFailureFail`, `OnFailureSkip`, `OnFailureContinue`, `OnFailureRework`, `OnFailureRetry`)? [Consistency]
- [ ] CHK014 - Are the event names (`review_started`, `review_completed`, `review_failed`) consistent with the existing event naming pattern in the codebase? Existing events use patterns like `step_started`, `contract_validated` — do the review events follow the same prefix conventions? [Consistency]
- [ ] CHK015 - Is the `ContractConfig` duplication between `internal/pipeline` and `internal/contract` packages explicitly acknowledged as intentional? The plan notes "Two ContractConfig types" as a concern but the spec doesn't address whether the new fields must be manually kept in sync [Consistency]

## Coverage

- [ ] CHK016 - Does the spec cover concurrent contract execution? All contracts are described as sequential, but is there a requirement explicitly stating they MUST be sequential, or could a future optimization parallelize independent non-agent contracts? [Coverage]
- [ ] CHK017 - Does the spec define behavior when a rework step itself fails (not the re-review, but the rework execution)? The review->rework->re-review loop assumes rework succeeds, but the rework step could crash, timeout, or produce no changes [Coverage]
- [ ] CHK018 - Does the spec define what happens to the ReviewFeedback artifact from a previous rework cycle when a new rework cycle starts? Is it overwritten, versioned, or accumulated? [Coverage]
- [ ] CHK019 - Does the spec cover the interaction between contract-level rework (`ReworkStep` on `ContractConfig`) and step-level rework (`RetryConfig.ReworkStep`)? Can both be configured simultaneously? Which takes precedence? [Coverage]
- [ ] CHK020 - Are non-functional requirements for agent review (latency, reliability, cost) testable as stated? SC-002 says "<$0.02 per step" and SC-007 says "<60 seconds" — but these depend on external model pricing and performance which may change. Are the thresholds pinned to specific conditions? [Coverage]
- [ ] CHK021 - Does the spec cover what information is logged/audited for agent review executions? The existing audit system scrubs credentials — are review criteria, feedback, and context sources subject to the same audit logging? [Coverage]
- [ ] CHK022 - Does the spec define the git state assumptions for `git diff HEAD`? If the step has made commits (not just uncommitted changes), `git diff HEAD` shows nothing. Should the diff be against the workspace's initial state instead? [Coverage]
