# Research: Third-Party Model Providers via OpenCode Adapter

**Branch**: `411-opencode-model-providers` | **Date**: 2026-03-16

## R-1: Provider/Model Identifier Parsing Strategy

**Decision**: Split on first `/` only. `provider/model-name` → provider=`provider`, model=`model-name`.

**Rationale**: OpenCode config.json uses separate `"provider"` and `"model"` fields. The spec explicitly states "only the first `/` should be treated as the provider delimiter" for cases like `provider/org/model`.

**Alternatives Rejected**:
- Split on all `/` — breaks model names containing organization prefixes
- Regex-based parsing — over-engineered for a simple split

## R-2: Provider Inference from Model Name Patterns

**Decision**: Use a static `map[string]string` mapping well-known prefixes to providers:
- `gpt-` → `openai`
- `gemini-` → `google`
- `claude-` → `anthropic`

**Rationale**: The spec mandates exactly 3 mappings (FR-003, SC-006). A Go map is trivially extensible later. Unknown model names without a provider prefix should fall back to the default provider (`anthropic`).

**Alternatives Rejected**:
- Dynamic lookup/registry — adds complexity for 3 entries
- CLI flag for provider — spec says use config.json only (C-4)

## R-3: Shared Base Environment Construction

**Decision**: Extract `BuildCuratedEnvironment(cfg AdapterRunConfig) []string` as a package-level function in `adapter.go` (or a new `environment.go`). Claude adapter calls it + appends telemetry vars. OpenCode adapter calls it directly.

**Rationale**: Claude's `buildEnvironment` already implements the pattern: base vars (`HOME`, `PATH`, `TERM`, `TMPDIR`) + `EnvPassthrough` + `cfg.Env`. Extracting shared logic follows DRY and makes future adapters consistent. Spec clarification C-3 and C-5 explicitly call for this.

**Alternatives Rejected**:
- Duplicate the logic in OpenCode adapter — violates DRY, risks drift
- Base class/embedding — Go favors composition; a shared function is simpler

## R-4: OpenCode Environment Security Gap

**Decision**: The current OpenCode adapter uses `os.Environ()` (line 55 of `opencode.go`), leaking the full host environment. Replace with the shared curated environment function.

**Rationale**: This is the FR-005 requirement. The Claude adapter already demonstrates the pattern. The docs (`docs/concepts/adapters.md:69`, `docs/concepts/adapters.md:149`) explicitly note that only the Claude adapter uses curated env — this is the gap to close.

**Alternatives Rejected**:
- Leave as-is with docs warning — violates the spec requirement and Constitution Principle 9

## R-5: Model Resolution Precedence

**Decision**: The existing `resolveModel` function in `executor.go:1635` already implements CLI > persona > default precedence. The OpenCode adapter just needs to read `cfg.Model` instead of hardcoding.

**Rationale**: FR-009 says precedence must match Claude adapter. The executor already handles this — the adapter just needs to consume it.

**Alternatives Rejected**:
- Implement precedence in the adapter — duplicates executor logic
