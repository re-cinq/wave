# Tasks: Third-Party Model Providers via OpenCode Adapter

**Branch**: `411-opencode-model-providers` | **Generated**: 2026-03-16
**Spec**: `specs/411-opencode-model-providers/spec.md`
**Plan**: `specs/411-opencode-model-providers/plan.md`

## Phase 1: Setup

- [X] T001 [P1] Verify branch and existing adapter source compiles: `go build ./internal/adapter/...`

## Phase 2: Foundational — Shared Environment & Model Parsing (Story 1 + Story 2 prerequisite)

- [X] T002 [P1] [Story1] Create `internal/adapter/environment.go` with `ProviderModel` struct and `ParseProviderModel(model string) ProviderModel` function that splits on first `/`, infers provider from `knownModelPrefixes` map (`gpt-`→`openai`, `gemini-`→`google`, `claude-`→`anthropic`), and defaults to `anthropic`/`claude-sonnet-4-20250514` when empty
- [X] T003 [P1] [Story2] Add `BuildCuratedEnvironment(cfg AdapterRunConfig) []string` to `internal/adapter/environment.go` — extracts base vars (`HOME`, `PATH`, `TERM`, `TMPDIR`) + `EnvPassthrough` + `cfg.Env` into a shared function
- [X] T004 [P1] [P] Create `internal/adapter/environment_test.go` with table-driven tests for `ParseProviderModel`: explicit prefix (`openai/gpt-4o`), inferred prefix (`gpt-4o`, `gemini-pro`, `claude-sonnet-4-20250514`), unknown model without prefix (defaults to `anthropic`), multi-slash (`provider/org/model`→provider=`provider`,model=`org/model`), empty string (defaults), and unrecognized prefix passthrough (`custom/my-model`)
- [X] T005 [P1] [P] Add table-driven tests for `BuildCuratedEnvironment` in `internal/adapter/environment_test.go`: base vars present, passthrough vars included when set in host env, missing passthrough vars silently skipped, `cfg.Env` step-specific vars appended, canary var NOT leaked

## Phase 3: Story 1 — Dynamic Model Configuration for OpenCode Persona (P1)

- [X] T006 [P1] [Story1] Modify `OpenCodeAdapter.prepareWorkspace` in `internal/adapter/opencode.go` to call `ParseProviderModel(cfg.Model)` and write resolved `provider` and `model` values to `.opencode/config.json` instead of hardcoded `"anthropic"`/`"claude-sonnet-4-20250514"`
- [X] T007 [P1] [Story1] [P] Add tests in `internal/adapter/opencode_test.go` for `prepareWorkspace` model resolution: verify `.opencode/config.json` contains correct provider/model for explicit prefix, inferred prefix, no model (defaults), and multi-slash model identifier

## Phase 4: Story 2 — API Key Passthrough via Curated Environment (P1)

- [X] T008 [P1] [Story2] Refactor `ClaudeAdapter.buildEnvironment` in `internal/adapter/claude.go` to call `BuildCuratedEnvironment(cfg)` and then append Claude-specific telemetry suppression vars (`DISABLE_TELEMETRY`, `DISABLE_ERROR_REPORTING`, `CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY`, `DISABLE_BUG_COMMAND`)
- [X] T009 [P1] [Story2] Modify `OpenCodeAdapter.Run` in `internal/adapter/opencode.go` to replace `os.Environ()` (line 55) with `BuildCuratedEnvironment(cfg)` for curated subprocess environment
- [X] T010 [P1] [Story2] [P] Add tests in `internal/adapter/opencode_test.go` verifying OpenCode curated environment: base vars present, passthrough vars included, canary vars NOT leaked, missing passthrough silently skipped
- [X] T011 [P1] [Story2] [P] Update existing `ClaudeAdapter.buildEnvironment` tests in `internal/adapter/claude_test.go` to verify refactored method still produces identical output (telemetry vars present, passthrough vars present, canary vars absent)

## Phase 5: Story 3 — Documentation (P2)

- [X] T012 [P2] [Story3] Update `docs/reference/adapters.md` OpenCode section: add model configuration examples showing `provider/model` format, `env_passthrough` for API keys, complete end-to-end persona configuration example
- [X] T013 [P2] [Story3] Update `docs/concepts/adapters.md`: remove "OpenCode inherits full host env" caveat, describe curated environment parity with Claude adapter, document provider/model identifier format

## Phase 6: Story 4 — Security Hardening Validation (P3)

- [X] T014 [P3] [Story4] Already covered by T009 + T010 — verify in integration that `os.Environ()` is fully removed from OpenCode adapter and only `BuildCuratedEnvironment` is used

## Phase 7: Polish & Cross-Cutting

- [X] T015 [P] Run `go test ./internal/adapter/...` to verify all existing and new tests pass
- [X] T016 [P] Run `go vet ./internal/adapter/...` and `go build ./...` to verify no compilation errors
- [X] T017 Verify success criteria: SC-001 (config.json provider/model), SC-002 (precedence), SC-003 (env passthrough), SC-004 (backward compat), SC-005 (docs example), SC-006 (3 prefix inferences)
