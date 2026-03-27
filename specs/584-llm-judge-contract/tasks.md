# Tasks

## Phase 1: Schema Extension
- [X] Task 1.1: Add LLM judge fields to contract.ContractConfig (`Model`, `Criteria`, `Threshold`)
- [X] Task 1.2: Add LLM judge fields to pipeline.ContractConfig (`Model`, `Criteria`, `Threshold`) [P]
- [X] Task 1.3: Pass new fields from pipeline ContractConfig to contract.ContractConfig in executor.go validation block

## Phase 2: Core Implementation
- [X] Task 2.1: Create `internal/contract/llm_judge.go` with `llmJudgeValidator` struct and `Validate()` method
- [X] Task 2.2: Implement prompt construction — system prompt + criteria list + step output content
- [X] Task 2.3: Implement Anthropic Messages API HTTP client (request building, response parsing)
- [X] Task 2.4: Implement threshold evaluation and `ValidationError` construction with per-criterion details
- [X] Task 2.5: Register `llm_judge` in `NewValidator()` switch in `contract.go`

## Phase 3: Executor Integration
- [X] Task 3.1: Add `llm_judge` case to `buildContractPrompt()` in executor.go — inform persona that output will be evaluated by LLM judge with listed criteria

## Phase 4: Testing
- [X] Task 4.1: Create `internal/contract/llm_judge_test.go` with httptest mock server [P]
- [X] Task 4.2: Test valid pass scenario (all criteria pass, above threshold) [P]
- [X] Task 4.3: Test threshold failure scenario (below threshold) [P]
- [X] Task 4.4: Test threshold boundary (score equals threshold — should pass) [P]
- [X] Task 4.5: Test error cases (missing API key, missing criteria, API error, malformed response) [P]
- [X] Task 4.6: Add `llm_judge` to `TestNewValidator` and `TestValidate_AllTypes` table-driven tests

## Phase 5: Validation
- [X] Task 5.1: Run `go test ./internal/contract/...` to verify all contract tests pass
- [X] Task 5.2: Run `go test ./internal/pipeline/...` to verify executor tests pass
- [X] Task 5.3: Run `go vet ./...` and verify no issues
