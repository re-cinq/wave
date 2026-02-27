# Implementation Plan: Concurrent Agent Count Injection

## Objective

Inject a concurrency hint into the persona's runtime CLAUDE.md when a pipeline step sets `max_concurrent_agents > 1`, so the agent knows how many sub-agents it may spawn via the Task tool.

## Approach

The change follows the existing CLAUDE.md assembly pipeline: data flows from pipeline YAML → `Step` struct → `AdapterRunConfig` → `prepareWorkspace()` where CLAUDE.md is assembled. A new section is inserted between the contract compliance section and the restrictions section.

### Data Flow

1. **YAML parse** → `Step.MaxConcurrentAgents int` (new field on `Step` struct in `internal/pipeline/types.go`)
2. **Executor** → Reads `step.MaxConcurrentAgents`, passes it to `AdapterRunConfig.MaxConcurrentAgents int` (new field)
3. **Claude adapter** → `prepareWorkspace()` checks `cfg.MaxConcurrentAgents > 1` and writes a concurrency section into CLAUDE.md between the contract prompt and restrictions
4. **Validation** → Cap at 10 (Claude Code hard limit), reject values < 0

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/types.go` | modify | Add `MaxConcurrentAgents int` field to `Step` struct |
| `internal/adapter/adapter.go` | modify | Add `MaxConcurrentAgents int` field to `AdapterRunConfig` struct |
| `internal/pipeline/executor.go` | modify | Pass `step.MaxConcurrentAgents` to `AdapterRunConfig` when building the config |
| `internal/adapter/claude.go` | modify | Add concurrency section to CLAUDE.md in `prepareWorkspace()` between contract prompt and restrictions |
| `internal/adapter/claude_test.go` | modify | Add test cases for concurrency hint in CLAUDE.md (following `TestCLAUDEMDRestrictionSection` pattern) |
| `internal/pipeline/executor_test.go` | modify | Add test for `MaxConcurrentAgents` propagation to `AdapterRunConfig` |
| `internal/pipeline/validation.go` | modify | Add validation: `MaxConcurrentAgents` must be 0–10, reject negative values |
| `internal/pipeline/validation_test.go` | modify | Add test cases for validation of `MaxConcurrentAgents` bounds |

## Architecture Decisions

### AD-1: Field on Step, not Persona
The concurrency count is per-step, not per-persona. The same persona (e.g., "implementer") may run in different steps with different concurrency limits. This matches the existing pattern where `MatrixStrategy.MaxConcurrency` is on `Step`, not `Persona`.

### AD-2: CLAUDE.md section placement
Insert between contract compliance (`cfg.ContractPrompt`) and restrictions (`buildRestrictionSection`). This follows the research recommendation and maintains the existing layered prompt architecture: base protocol → persona → contract → **concurrency** → restrictions.

### AD-3: Permission language
Use "You may spawn up to N concurrent sub-agents or workers for this step." — positive permission language, per research finding that permission language works better than prohibition.

### AD-4: Cap at 10
Claude Code's Task tool has a hard cap of 10 concurrent subagents. Validate at pipeline load time that `max_concurrent_agents <= 10`.

### AD-5: No hint when unset or <= 1
When `MaxConcurrentAgents` is 0 (default/unset) or 1, no section is added to CLAUDE.md. This avoids prompt bloat and is semantically correct: the default behavior is single-agent execution.

## Risks

| Risk | Mitigation |
|------|------------|
| Prompt section ordering affects agent behavior | Place after contract (most critical) but before restrictions (least likely to be read) |
| Future adapters may not support concurrency | Field is on `AdapterRunConfig` as raw int; other adapters can ignore it |
| Value > 10 silently accepted | Validation at pipeline load rejects values > 10 with clear error |

## Testing Strategy

1. **Unit test: `TestBuildConcurrencySection`** — Test the concurrency section builder function directly with values 0, 1, 3, 10
2. **Unit test: `TestCLAUDEMDConcurrencySection`** — Test CLAUDE.md contains/doesn't contain concurrency hint via `prepareWorkspace()`, following existing `TestCLAUDEMDRestrictionSection` pattern
3. **Unit test: `TestMaxConcurrentAgentsValidation`** — Test pipeline validation rejects -1, 11, accepts 0, 1, 10
4. **Unit test: `TestMaxConcurrentAgentsPropagation`** — Test executor passes the field through to `AdapterRunConfig`
5. **Integration test** — Verify hint appears in adapter invocation by checking the assembled CLAUDE.md content in a full step execution scenario
