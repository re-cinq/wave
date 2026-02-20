# Runtime Integration Quality Checklist

**Feature**: Persona Prompt Optimization (Issue #96)
**Focus**: Quality of requirements governing the adapter code change, parity enforcement, and testing
**Date**: 2026-02-20

## Adapter Code Change (FR-002, CLR-003)

- [ ] CHK-RI001 - Does the spec define the exact read path for base-protocol.md — is it always `.wave/personas/base-protocol.md` relative to workspace root, or could it be resolved from a different location? [Clarity]
- [ ] CHK-RI002 - Does the spec address whether the base protocol file path should be configurable or hardcoded? [Completeness]
- [ ] CHK-RI003 - Does the spec define the error type and message format when base-protocol.md is missing — is `fmt.Errorf` sufficient or should it use a structured error type? [Clarity]
- [ ] CHK-RI004 - Does the spec address whether the base protocol should be cached in memory across steps within a single pipeline run, or re-read from disk for each step? [Completeness]
- [ ] CHK-RI005 - Does FR-015's constraint ("only permitted change is adding the base protocol prepend logic") account for all the code that will actually need to change — e.g., error handling, file reading, string concatenation? [Completeness]
- [ ] CHK-RI006 - Does the spec define the expected content of the generated CLAUDE.md with enough precision to write a deterministic assertion in tests? [Coverage]

## Parity Enforcement (FR-011, FR-012, SC-006)

- [ ] CHK-RI007 - Does the spec define whether parity is checked at build time (via go:embed comparison), at CI time (via Go test), or both? [Completeness]
- [ ] CHK-RI008 - Does the spec define whether the parity test should fail on missing files in either direction (file in defaults but not in .wave, or vice versa)? [Coverage]
- [ ] CHK-RI009 - Does the spec address who is responsible for maintaining parity — is it a manual step in the development workflow, or should tooling enforce it? [Completeness]
- [ ] CHK-RI010 - Does the spec define whether parity includes file permissions, line endings, and BOM markers, or only content bytes? [Clarity]

## Testing Requirements (SC-005, SC-008)

- [ ] CHK-RI011 - Does the spec define the minimum test cases for the base protocol injection — happy path, missing file, empty file, inline prompt, restriction section interaction? [Coverage]
- [ ] CHK-RI012 - Does the spec address whether existing adapter tests need modification to account for the new base protocol preamble in CLAUDE.md output? [Coverage]
- [ ] CHK-RI013 - Does the spec define whether SC-008 (runtime injection verification) requires a new integration test, or can it be verified via existing unit tests? [Clarity]
- [ ] CHK-RI014 - Does the spec address race condition testing requirements for the adapter change — could concurrent pipeline steps read base-protocol.md simultaneously? [Coverage]

## Init & Upgrade Path

- [ ] CHK-RI015 - Does the spec define the behavior of `wave init` on a project that already has `.wave/personas/` without `base-protocol.md` (upgrade scenario)? [Coverage]
- [ ] CHK-RI016 - Does the spec address whether `PersonaNames()` should filter out `base-protocol.md` or whether callers are expected to handle it? [Clarity]
- [ ] CHK-RI017 - Does the spec define whether the default manifest template should be updated to acknowledge the base protocol's existence, even if no manifest entry is created for it? [Completeness]
