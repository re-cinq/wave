# Requirements Quality Review: 411-opencode-model-providers

**Generated**: 2026-03-16
**Spec**: `specs/411-opencode-model-providers/spec.md`

## Completeness

- [ ] CHK001 - Are all supported OpenCode providers enumerated, or is the set intentionally open-ended? [Completeness]
- [ ] CHK002 - Does FR-003 define behavior when a model name matches multiple known prefixes (e.g., a hypothetical `gpt-gemini-v1`)? [Completeness]
- [ ] CHK003 - Is the full list of base environment variables (`HOME`, `PATH`, `TERM`, `TMPDIR`) justified — are there other essential vars (e.g., `LANG`, `USER`, `SHELL`) that adapters commonly need? [Completeness]
- [ ] CHK004 - Does the spec define what happens when `env_passthrough` contains duplicate entries? [Completeness]
- [ ] CHK005 - Are error messages or user-facing feedback specified for invalid model identifier formats? [Completeness]
- [ ] CHK006 - Does the spec address whether `cfg.Env` step-specific variables can override base variables or passthrough variables? [Completeness]
- [ ] CHK007 - Is the behavior for whitespace or malformed model strings (e.g., `" "`, `"/"`, `"openai/"`) defined? [Completeness]

## Clarity

- [ ] CHK008 - Is the term "well-known model name patterns" in FR-003 precise enough — does it mean prefix matching, regex, or exact map lookup? [Clarity]
- [ ] CHK009 - Does "sensible defaults" in acceptance scenario 1.2 have a concrete definition, or could it be misinterpreted? [Clarity]
- [ ] CHK010 - Is the phrase "pass through as-is" for unrecognized providers (edge case 2) unambiguous about what field receives the value? [Clarity]
- [ ] CHK011 - Does FR-005 clearly distinguish between "base variables" and "passthrough variables" in terms of override precedence? [Clarity]
- [ ] CHK012 - Is the scope of "curated environment model" in FR-005 clear about whether it applies to ALL subprocess invocations or only the main OpenCode CLI call? [Clarity]

## Consistency

- [ ] CHK013 - Does the default model in FR-004 (`claude-sonnet-4-20250514`) match the default referenced in edge case 3 and acceptance scenario 1.2? [Consistency]
- [ ] CHK014 - Is the precedence order (CLI > persona > adapter default) in FR-009 consistent with how the Claude adapter implements precedence today? [Consistency]
- [ ] CHK015 - Does C-3 resolution (no Claude telemetry vars for OpenCode) conflict with any requirement or acceptance scenario? [Consistency]
- [ ] CHK016 - Are Story 4 acceptance scenarios consistent with FR-005 base variable list — do both enumerate the same set? [Consistency]
- [ ] CHK017 - Does the plan's Phase 1 shared extraction (C-5) match the spec's FR-005 + FR-006 requirements without adding unstated scope? [Consistency]

## Coverage

- [ ] CHK018 - Does the spec cover backward compatibility — what happens to existing `wave.yaml` files that use `adapter: opencode` without a `model` field? [Coverage]
- [ ] CHK019 - Are there acceptance scenarios that test the multi-slash edge case (`provider/org/model`)? [Coverage]
- [ ] CHK020 - Does the spec address interaction between `--model` CLI flag and non-opencode adapters (e.g., does `--model openai/gpt-4o` on a Claude adapter cause an error)? [Coverage]
- [ ] CHK021 - Is the documentation requirement (FR-007, FR-008) covered by acceptance scenarios, or only by success criteria SC-005? [Coverage]
- [ ] CHK022 - Does the spec address what happens when OpenCode itself rejects the provider/model combination at runtime? [Coverage]
- [ ] CHK023 - Are there test scenarios covering the `BuildCuratedEnvironment` shared function being used by both adapters simultaneously? [Coverage]
