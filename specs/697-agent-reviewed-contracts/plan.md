# Implementation Plan: Agent-Reviewed Contracts

**Branch**: `697-agent-reviewed-contracts` | **Date**: 2026-03-30 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/697-agent-reviewed-contracts/spec.md`

## Summary

Add an `agent_review` contract type that spawns a separate agent (via the adapter runner) to evaluate step output against configurable review criteria, producing structured `ReviewFeedback`. Support plural `contracts` lists per step with ordered execution, early termination, and independent `on_failure` policies. Integrate rework-with-feedback loops, dashboard display, and retro friction tracking.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `gopkg.in/yaml.v3`, `github.com/spf13/cobra`, Wave internal adapter/contract/pipeline packages
**Storage**: SQLite (existing schema, no migrations needed), filesystem for artifacts
**Testing**: `go test ./...`, `go test -race ./...`
**Target Platform**: Linux (primary), macOS (secondary)
**Project Type**: Single binary CLI
**Performance Goals**: Agent reviews complete within 60s for diffs <20KB (SC-007); <$0.02/step token cost with Haiku (SC-002)
**Constraints**: Single static binary, no new runtime dependencies, backward compatible with singular `contract` field

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new runtime dependencies. Adapter runner already exists. |
| P2: Manifest as SSOT | PASS | All agent_review config lives in wave.yaml pipeline definitions. |
| P3: Persona-Scoped Execution | PASS | Reviewer runs as a separate persona with its own permissions. Self-review prevented by validation. |
| P4: Fresh Memory at Step Boundaries | PASS | Reviewer agent starts with fresh context. No chat history from the implementation step. |
| P5: Navigator-First Architecture | N/A | Agent review is a validation mechanism, not an implementation step. |
| P6: Contracts at Every Handover | PASS | This feature strengthens contract coverage by adding agent-powered validation. |
| P7: Relay via Dedicated Summarizer | N/A | Reviewer agent is short-lived; unlikely to hit context limits. |
| P8: Ephemeral Workspaces | PASS | Reviewer operates in the step's existing workspace with read-only access. |
| P9: Credentials Never Touch Disk | PASS | Reviewer uses adapter runner which handles credentials via env vars. |
| P10: Observable Progress | PASS | Review lifecycle events emitted (started/completed/failed). Dashboard display. |
| P11: Bounded Recursion | PASS | Review→rework→re-review bounded by `max_retries`. Token budget enforced. |
| P12: Minimal Step State Machine | PASS | No new step states. Review is a sub-phase of the existing "validating" state. |
| P13: Test Ownership | PASS | All changes require `go test ./...` to pass. New tests for every new code path. |

## Project Structure

### Documentation (this feature)

```
specs/697-agent-reviewed-contracts/
├── plan.md                                    # This file
├── spec.md                                    # Feature specification
├── research.md                                # Phase 0 research output
├── data-model.md                              # Phase 1 entity definitions
├── contracts/
│   ├── review-feedback-schema.json            # ReviewFeedback JSON schema
│   ├── agent-review-contract-config.yaml      # Agent review YAML contract reference
│   └── plural-contracts-config.yaml           # Plural contracts usage example
└── tasks.md                                   # Phase 2 output (from /speckit.tasks)
```

### Source Code (repository root)

```
internal/
├── contract/
│   ├── contract.go              # MODIFY: add agent_review to registry, ValidateWithRunner()
│   ├── agent_review.go          # NEW: agentReviewValidator, ReviewFeedback, context assembly
│   └── agent_review_test.go     # NEW: unit tests for agent review validator
├── pipeline/
│   ├── types.go                 # MODIFY: Contracts field on HandoverConfig, new ContractConfig fields
│   ├── executor.go              # MODIFY: plural contract validation loop, adapter runner passthrough
│   ├── executor_test.go         # MODIFY: tests for plural contracts, agent review integration
│   ├── validation.go            # MODIFY: self-review prevention, criteria path validation
│   └── dag.go                   # MODIFY: rework validation for contract-level rework_step
├── retro/
│   └── types.go                 # MODIFY: add FrictionReviewRework constant
├── event/
│   └── emitter.go               # MODIFY: add review-specific event fields (optional)
└── webui/
    ├── types.go                 # MODIFY: add review verdict fields to StepDetail
    └── handlers_runs.go         # MODIFY: populate review fields from events
```

**Structure Decision**: All changes are modifications to existing packages. One new file (`agent_review.go`) plus its test file. No new packages or directories in `internal/`.

## Implementation Tiers

### Tier 1 — Core Agent Review (FR-001, FR-002, FR-003, FR-005, FR-008, FR-016)

The foundation: `agent_review` contract type, validator, ReviewFeedback extraction, self-review prevention.

**Files modified**:
- `internal/contract/contract.go` — Add `agent_review` to `NewValidator()`, add `ValidateWithRunner()` function
- `internal/contract/agent_review.go` — **NEW**: `agentReviewValidator`, `ReviewFeedback`, `ReviewIssue`, `ReviewContextSource` types; `buildReviewPrompt()`, `parseReviewFeedback()`; uses `extractJSON()` from llm_judge
- `internal/pipeline/types.go` — Add `Persona`, `CriteriaPath`, `Context`, `TokenBudget`, `Timeout`, `ReworkStep` to `ContractConfig`
- `internal/pipeline/validation.go` — Self-review prevention check (`contract.Persona != step.Persona`), criteria path existence check, reviewer persona existence check
- `internal/pipeline/executor.go` — Pass adapter runner to `ValidateWithRunner()` for agent_review type

**Key design**:
```go
// internal/contract/contract.go — new interface
type AgentContractValidator interface {
    ValidateWithRunner(cfg ContractConfig, workspacePath string,
        runner adapter.AdapterRunner, manifest *manifest.Manifest) (*ReviewFeedback, error)
}

// Top-level function used by executor
func ValidateWithRunner(cfg ContractConfig, workspacePath string,
    runner adapter.AdapterRunner, manifest *manifest.Manifest) (*ReviewFeedback, error)
```

The executor calls `ValidateWithRunner()` when the contract type is `agent_review`, falling back to `Validate()` for all other types. The `agentReviewValidator`:
1. Loads criteria from `CriteriaPath`
2. Assembles context (artifacts + git_diff)
3. Builds a user prompt with criteria + context + ReviewFeedback JSON schema
4. Calls `runner.Run()` with reviewer persona config from the manifest
5. Parses stdout for ReviewFeedback JSON via `extractJSON()`
6. Returns the feedback (caller decides pass/fail based on verdict)

### Tier 2 — Plural Contracts & Rework (FR-009, FR-010, FR-011, FR-012)

Contract composition and rework-with-feedback loops.

**Files modified**:
- `internal/pipeline/types.go` — Add `Contracts []ContractConfig` to `HandoverConfig`, add `EffectiveContracts()` method
- `internal/pipeline/executor.go` — Replace single contract validation block with contract list loop; handle per-contract `on_failure`; implement contract-level rework (write ReviewFeedback artifact, trigger rework step, re-run all contracts)
- `internal/pipeline/dag.go` — Validate rework targets referenced from contract-level `rework_step` fields

**Key design**:
```go
// Executor contract validation loop (pseudocode)
contracts := step.Handover.EffectiveContracts()
for retryRound := 0; retryRound <= maxContractRetries; retryRound++ {
    allPassed := true
    for _, c := range contracts {
        result := validateContract(c, workspacePath, runner, manifest)
        if result.failed {
            allPassed = false
            switch c.OnFailure {
            case "rework":
                writeFeedbackArtifact(result.feedback)
                executeReworkStep(...)
                break // re-run all contracts
            case "fail":
                return error
            case "skip":
                break // skip remaining contracts
            case "continue":
                continue // next contract
            }
            break // for rework: exit inner loop to restart
        }
    }
    if allPassed { break }
}
```

### Tier 3 — Context Sources (FR-004, FR-006, FR-007)

Token budget enforcement and configurable context assembly.

**Files modified**:
- `internal/contract/agent_review.go` — `assembleContext()` function: git_diff capture via `git diff HEAD`, artifact reading, truncation logic
- `internal/pipeline/validation.go` — Token budget positive validation

**Key design**:
- `git diff HEAD` executed via `exec.CommandContext` in workspace directory
- Truncation at `MaxSize` bytes (default 50KB) with `[... truncated at 50KB ...]` notice
- Token budget checked post-execution against `AdapterResult.TokensUsed`

### Tier 4 — Observability (FR-013, FR-014, FR-015)

Events, dashboard, retros.

**Files modified**:
- `internal/pipeline/executor.go` — Emit `review_started`, `review_completed`, `review_failed` events with reviewer persona, verdict, issue count, token spend
- `internal/retro/types.go` — Add `FrictionReviewRework` constant
- `internal/retro/generator.go` — Detect review_rework events, create friction points
- `internal/webui/types.go` — Add `ReviewVerdict`, `ReviewIssueCount`, `ReviewerPersona`, `ReviewTokens` to `StepDetail`
- `internal/webui/handlers_runs.go` — Populate review fields from stored events

### Tier 5 — Wave Pipeline Upgrades (User Story 7)

Upgrade Wave's own pipelines with `agent_review` contracts.

**Files modified**:
- `.wave/pipelines/impl-issue.yaml` — Add `agent_review` contract to implementation step
- `.wave/pipelines/impl-speckit.yaml` — Add `agent_review` contract to implementation step
- `.wave/contracts/impl-review-criteria.md` — **NEW**: Review criteria for implementation steps

**Note**: This tier depends on all previous tiers being complete and tested. The criteria file and false-positive rate (SC-004) require iterative tuning based on real pipeline runs.

## Complexity Tracking

_No constitution violations found._

| Concern | Mitigation |
|---------|-----------|
| `executor.go` is already 4000+ lines | New code is isolated in the contract validation loop. The `agent_review` validator logic lives in its own file. Only the loop orchestration touches executor.go. |
| Two ContractConfig types (pipeline vs contract package) | Fields are copied explicitly in executor.go (existing pattern). Adding 6 new fields to both is manageable. Consider unifying in a future refactor. |
| Adapter runner dependency in contract package | Passed via function parameter, not stored globally. Contract package imports adapter package (new dependency direction). Acceptable: contract validators are consumers of adapter capabilities. |
