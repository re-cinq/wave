# Implementation Plan: LLM-as-Judge Contract Type

## Objective

Add a new `llm_judge` contract type that uses a cheap LLM (via Anthropic Messages API) to evaluate step output against human-defined criteria with threshold-based pass/fail semantics. This integrates into the existing contract validation pipeline without changing the `ContractValidator` interface.

## Approach

The LLM judge validator follows the same pattern as existing validators — it implements `ContractValidator.Validate(cfg, workspacePath)`. Internally it:

1. Reads the step output from `cfg.Source` (same as `jsonSchemaValidator`)
2. Builds a structured prompt with the evaluation criteria
3. Calls the Anthropic Messages API directly via `net/http` (no new dependencies)
4. Parses the structured JSON response
5. Evaluates the threshold: `(criteria_passed / total_criteria) >= threshold`
6. Returns `nil` on pass, `*ValidationError` on fail (with per-criterion reasoning in `Details`)

**Why direct HTTP instead of adapter subprocess?** The contract validator interface is `Validate(cfg, workspacePath) error` — it has no access to the adapter runner. Making a direct HTTP call to a well-documented API is simpler and faster than spawning a subprocess for a single API call. The `ANTHROPIC_API_KEY` env var is already available in the pipeline environment.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/contract/contract.go` | **modify** | Add `llm_judge` case to `NewValidator()` switch; add `Model`, `Criteria`, `Threshold` fields to `ContractConfig` |
| `internal/pipeline/types.go` | **modify** | Add `Model`, `Criteria`, `Threshold` fields to pipeline `ContractConfig` |
| `internal/contract/llm_judge.go` | **create** | `llmJudgeValidator` implementation: prompt construction, HTTP call, response parsing, threshold evaluation |
| `internal/contract/llm_judge_test.go` | **create** | Unit tests with HTTP test server mocking the Anthropic API |
| `internal/pipeline/executor.go` | **modify** | Pass new fields (`Model`, `Criteria`, `Threshold`) from pipeline ContractConfig to contract.ContractConfig in the validation block (~line 1786); add `llm_judge` case to `buildContractPrompt` |

## Architecture Decisions

### 1. Direct HTTP vs Adapter Subprocess
Use `net/http` to call the Anthropic Messages API. The adapter infrastructure is designed for long-running persona execution with workspace isolation, sandbox, event streaming, etc. An LLM judge call is a single, fast API request — subprocess overhead is unjustified.

### 2. Config Fields on ContractConfig
Add `Model`, `Criteria []string`, and `Threshold float64` directly to `ContractConfig` rather than creating a separate struct. This keeps the YAML flat and consistent with existing fields (`Command`, `Schema`, etc.) that are type-specific.

### 3. Prompt Design
The judge prompt asks for a JSON response matching the schema from the issue. The system prompt instructs the model to evaluate objectively and provide reasoning. The step output is included as context.

### 4. Threshold Semantics
`threshold` defaults to `1.0` (all criteria must pass) when omitted. This is the safe default — explicit opt-in to partial passing.

### 5. Error Handling
- Missing `ANTHROPIC_API_KEY`: returns `ValidationError` with install instructions
- API failure (non-200, timeout): returns `ValidationError` with retryable=true
- Malformed response: returns `ValidationError` with the raw response in details
- Missing criteria: returns `ValidationError` immediately (misconfiguration)

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| API key not available in sandbox | Judge can't call API | Check env var early, return clear error with setup instructions |
| LLM returns non-JSON response | Parse failure | Use structured prompt with JSON example; apply `cleanJSON()` recovery |
| Judge model disagrees with criteria intent | False positives/negatives | Include reasoning in `ValidationError.Details` for transparency |
| Network latency on API call | Slow validation | Use timeout context (30s default); this is a validation step, not interactive |
| API rate limits | Intermittent failures | Mark `Retryable: true` on API errors; existing retry infrastructure handles it |

## Testing Strategy

1. **Unit tests** (`llm_judge_test.go`):
   - Valid pass: all criteria pass, score >= threshold
   - Threshold failure: 2/4 criteria pass, threshold 0.8 requires 80%
   - Exact threshold boundary: score equals threshold (should pass)
   - Missing criteria config: returns error
   - Missing API key: returns clear error
   - API error (non-200): returns retryable error
   - Malformed API response: returns error with details
   - Default threshold (1.0) when omitted

2. **Integration with existing tests**:
   - `TestNewValidator` table: add `llm_judge` case
   - `TestValidate_AllTypes` table: add `llm_judge` case (with mock server)

3. **Mock approach**: Use `httptest.NewServer` to mock the Anthropic Messages API, returning canned judge responses. No real API calls in tests.
