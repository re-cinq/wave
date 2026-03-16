# Adapter Parity Checklist: 411-opencode-model-providers

**Generated**: 2026-03-16
**Spec**: `specs/411-opencode-model-providers/spec.md`

## Claude-OpenCode Adapter Consistency

- [ ] CHK-P01 - Does the spec enumerate all behavioral differences that SHOULD exist between Claude and OpenCode curated environments (e.g., telemetry vars)? [Completeness]
- [ ] CHK-P02 - Is the shared `BuildCuratedEnvironment` interface defined precisely enough that both adapters produce predictable output? [Clarity]
- [ ] CHK-P03 - Does the spec define whether the OpenCode adapter should support `cfg.Env` step-specific overrides the same way the Claude adapter does? [Consistency]
- [ ] CHK-P04 - Are there requirements ensuring the refactored Claude adapter produces byte-identical environment output after the extraction? [Coverage]
- [ ] CHK-P05 - Does the spec address whether future adapters (beyond Claude and OpenCode) should also use `BuildCuratedEnvironment`? [Completeness]
- [ ] CHK-P06 - Is the model precedence rule (FR-009) testable for the OpenCode adapter specifically, or only stated generically? [Clarity]
- [ ] CHK-P07 - Does the documentation (FR-007, FR-008) clearly distinguish OpenCode-specific vs adapter-generic configuration? [Clarity]
