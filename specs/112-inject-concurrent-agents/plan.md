# Implementation Plan: Inject Concurrent Agent Count

## Objective

Add a `max_concurrent_agents` field to the pipeline step definition that, when set > 1, injects a concurrency permission line into the persona's generated CLAUDE.md. This allows Claude to know it can spawn subagents for parallel work.

## Approach

The change threads a new integer field through three layers:

1. **Pipeline types** (`internal/pipeline/types.go`): Add `MaxConcurrentAgents int` to the `Step` struct
2. **Adapter config** (`internal/adapter/adapter.go`): Add `MaxConcurrentAgents int` to `AdapterRunConfig`
3. **CLAUDE.md assembly** (`internal/adapter/claude.go`): Inject a concurrency hint section in `prepareWorkspace` between the contract compliance section and the restriction section
4. **Executor wiring** (`internal/pipeline/executor.go`): Pass the step's `MaxConcurrentAgents` value into the `AdapterRunConfig`

The concurrency value is capped at 10 (Claude Code's practical limit) for safety. Values of 0 or 1 produce no hint (default single-agent behavior).

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/types.go` | modify | Add `MaxConcurrentAgents int` field to `Step` struct |
| `internal/adapter/adapter.go` | modify | Add `MaxConcurrentAgents int` field to `AdapterRunConfig` |
| `internal/adapter/claude.go` | modify | Add concurrency hint injection in `prepareWorkspace` between contract and restrictions |
| `internal/pipeline/executor.go` | modify | Wire `step.MaxConcurrentAgents` into `AdapterRunConfig` |
| `internal/adapter/claude_test.go` | modify | Add table-driven tests for concurrency hint generation |
| `internal/pipeline/executor_test.go` or `stepcontroller_test.go` | modify | Add integration test verifying hint appears in adapter config |
| `docs/reference/manifest-schema.md` | modify | Document `max_concurrent_agents` field |
| `.wave/schemas/wave-manifest.schema.json` | modify | Add field to JSON schema |

## Architecture Decisions

### 1. Injection Point: Between Contract and Restrictions in CLAUDE.md

The CLAUDE.md assembly in `prepareWorkspace` (claude.go:254-293) builds four sections:
1. Base protocol preamble
2. Persona system prompt
3. Contract compliance (ContractPrompt)
4. **NEW: Concurrency hint** (when MaxConcurrentAgents > 1)
5. Restrictions (deny/allow tools, network domains)

This placement ensures the agent sees the concurrency permission after understanding its role and output requirements, but before the restrictions that constrain its behavior.

### 2. Field on Step (not Persona)

Concurrency is a per-step concern, not a per-persona property. The same persona (e.g., `implementer`) may run with different concurrency limits in different pipeline steps. Placing it on `Step` allows pipeline authors to tune concurrency per step.

### 3. Cap at 10

Claude Code has a practical limit of ~10 concurrent subagents. The system will cap the value at 10 and emit a warning if a higher value is configured. This is enforced in the adapter layer, not the YAML parser, so manifest validation stays simple.

### 4. Permission Language (not Prohibition)

Following the research recommendation, the prompt uses permission language:
```
You may spawn up to N concurrent sub-agents or workers for this step.
```
This is more effective than "Do not spawn more than N agents" for LLM instruction following.

### 5. Raw Int on AdapterRunConfig

The `AdapterRunConfig` carries a raw `int` rather than a formatted string. This keeps the adapter interface clean and allows different adapters (Claude, OpenCode, etc.) to format the hint differently if needed.

## Risks

| Risk | Mitigation |
|------|------------|
| Field name collision with existing `max_concurrent` on IterateConfig | Use distinct name `max_concurrent_agents` — different semantic (agent subprocesses vs parallel iteration workers) |
| Values > 10 causing Claude Code errors | Cap at 10 in adapter layer with warning log |
| Existing pipelines break on new field | Field is optional with zero value = no change in behavior |
| Prompt injection via the integer field | Value is a hardcoded template with `%d` formatting — no user string interpolation |

## Testing Strategy

### Unit Tests (adapter package)
- Table-driven test in `claude_test.go`:
  - `MaxConcurrentAgents = 0` → no concurrency section in CLAUDE.md
  - `MaxConcurrentAgents = 1` → no concurrency section in CLAUDE.md
  - `MaxConcurrentAgents = 3` → contains "You may spawn up to 3 concurrent sub-agents"
  - `MaxConcurrentAgents = 15` → capped at 10 in output
- Verify the hint appears between contract and restrictions sections

### Integration Test (pipeline package)
- Use `configCapturingAdapter` to verify `MaxConcurrentAgents` is correctly passed from step definition to `AdapterRunConfig`

### YAML Parsing Test
- Add a test case in `internal/manifest/parser_test.go` or `internal/pipeline/` for parsing `max_concurrent_agents` from YAML
