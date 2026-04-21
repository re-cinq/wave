# Requirements Quality Checklist: Farewell Function

Purpose: validate requirements quality before implementation. Each item tests the spec, not the code.

## Completeness

- [ ] CHK001 - Is the exact farewell string (or template) specified as a fixed literal? [Completeness, Spec §FR-009]
- [ ] CHK002 - Is the placeholder/format for interpolating the recipient name defined (e.g. `"Goodbye, {name}!"`)? [Completeness, Spec §FR-002]
- [ ] CHK003 - Is the generic fallback wording (no-name case) defined verbatim? [Completeness, Spec §FR-003, Edge Cases]
- [ ] CHK004 - Is "successful interactive command execution" defined precisely (which commands, which exit paths)? [Completeness, Spec §FR-004]
- [ ] CHK005 - Is "end of command" defined (before/after final flush, before/after TUI teardown)? [Completeness, Spec §AS-1.2]
- [ ] CHK006 - Is behavior on signal-based exit (SIGINT/SIGTERM) specified beyond "MAY be skipped"? [Completeness, Spec §Edge Cases]
- [ ] CHK007 - Is the TTY-detection target specified (stdout only, or both stdout+stderr)? [Completeness, Spec §FR-005]
- [ ] CHK008 - Is the exact name of the existing global quiet flag identified? [Completeness, Spec §FR-011]
- [ ] CHK009 - Are non-interactive commands (e.g. `--detach`, server mode, `wave run` in CI) explicitly classified as suppress-or-emit? [Completeness, Spec §US3]

## Clarity

- [ ] CHK010 - Is "interactive" unambiguously defined (TTY on stdout? stdin? both?)? [Clarity, Spec §FR-004, FR-005]
- [ ] CHK011 - Is "successful" defined in terms of process exit code = 0, or something else? [Clarity, Spec §FR-007]
- [ ] CHK012 - Does "recipient name" have a max length / sanitization rule (control chars, newlines)? [Clarity, Spec §FR-002]
- [ ] CHK013 - Is "before process exit" ordered relative to other exit-time output (metrics, telemetry flush)? [Clarity, Spec §AS-1.1]
- [ ] CHK014 - Is "reuse existing global quiet signal" traced to a concrete flag/config key? [Clarity, Spec §FR-011]

## Consistency

- [ ] CHK015 - Do FR-005 (non-TTY suppress) and FR-004 (print on success) agree on precedence order? [Consistency, Spec §FR-004, FR-005]
- [ ] CHK016 - Do FR-007 (no farewell on failure) and AS-1.1 ("completes successfully") use the same success definition? [Consistency]
- [ ] CHK017 - Does FR-008 (single source) align with US2's programmatic `Farewell` function as that single source? [Consistency, Spec §FR-008, §US2]
- [ ] CHK018 - Does SC-003 (determinism) hold given FR-002 name interpolation from `$USER` which varies per environment? [Consistency, Spec §SC-003, FR-010]
- [ ] CHK019 - Do edge cases (Ctrl+C, pipe redirect) match FR-005/FR-007 without contradiction? [Consistency]

## Coverage

- [ ] CHK020 - Is there a requirement covering TUI farewell rendering after alt-screen teardown? [Coverage, Spec §AS-1.2]
- [ ] CHK021 - Is there a requirement for empty `$USER` → generic message path (observable)? [Coverage, Spec §FR-010, Edge Cases]
- [ ] CHK022 - Is there a requirement covering pipeline-run completion (not just CLI wrapper commands)? [Coverage, Spec §US1]
- [ ] CHK023 - Is there acceptance criteria for the stderr vs stdout channel? [Coverage, Spec §FR-006]
- [ ] CHK024 - Is there a performance ceiling (SC-004 covers 50 ms) and is it measurable with a defined baseline command? [Coverage, Spec §SC-004]
- [ ] CHK025 - Is there a testable assertion that CLI, TUI, and embedder paths produce byte-identical output? [Coverage, Spec §FR-008]

## Testability

- [ ] CHK026 - Can each FR be verified by a black-box test without reading source code? [Testability]
- [ ] CHK027 - Is "100% of successful runs" (SC-001) measurable across a finite, enumerated command set? [Testability, Spec §SC-001]
- [ ] CHK028 - Is the SC-004 50 ms ceiling reproducible (baseline hardware/command defined)? [Testability, Spec §SC-004]

## Scope & Non-Goals

- [ ] CHK029 - Are i18n/localization, random-pool variants, and per-user greeting profiles all explicitly out of scope? [Scope, Spec §Clarifications, §FR-009]
- [ ] CHK030 - Is it explicit that no new CLI flag is introduced? [Scope, Spec §FR-011]

## Critical Gaps

Critical = ambiguity that would block implementation or cause divergent implementations across CLI/TUI/API.

- CHK001, CHK002, CHK003 — exact string/template not in spec (implementor must invent wording).
- CHK008, CHK014 — global quiet flag name unidentified.
- CHK006 — signal-exit behavior under-specified ("MAY" leaves it open).
- CHK018 — SC-003 determinism claim conflicts with env-derived `$USER`.
