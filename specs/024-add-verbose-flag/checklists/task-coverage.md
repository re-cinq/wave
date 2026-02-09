# Task Coverage Review: Add --verbose Flag to Wave CLI

**Feature Branch**: `024-add-verbose-flag`
**Date**: 2026-02-06
**Artifacts Reviewed**: tasks.md cross-referenced against spec.md, plan.md

## Traceability

- [ ] CHK101 - Does any task explicitly trace to FR-002 (verbose propagation to ALL subcommands), or is FR-002 only implicitly covered by the combination of T001 and the individual command tasks? [Traceability]
- [ ] CHK102 - Does any task explicitly cover FR-008 (dual-stream model: stdout for NDJSON, stderr for human-readable verbose details), or is it assumed that extending the Event struct (T004) and rendering (T008) satisfy this without a dedicated verification step? [Traceability]
- [ ] CHK103 - The Task-to-Acceptance Mapping maps US1-S2 (no regression when --verbose omitted) solely to T019 (full test suite run), but no task explicitly creates a regression assertion for output-parity. Is relying on existing tests sufficient to cover SC-002 and FR-006? [Traceability]
- [ ] CHK104 - FR-007 (no conflict with validate's -v flag) maps to T011, which says "no code change expected; add a test." Does the task-to-acceptance mapping distinguish between verifying no conflict (FR-007) versus verifying verbose output content for validate (US2-S1)? [Traceability]
- [ ] CHK105 - SC-001 states verbose must produce "additional output lines" for ALL verbose-eligible commands. No single task verifies this cross-command property. Is the implicit union of T007-T011 sufficient? [Traceability]

## Granularity

- [ ] CHK106 - T007 bundles four distinct verbose enrichments (workspace path, injected artifacts, persona name, contract result) into one task touching three code regions. Should this be decomposed for independent verification? [Granularity]
- [ ] CHK107 - T012 covers both US3-S1 (debug supersedes verbose) and US3-S2 (verbose excludes debug traces) as a single task. Are these distinct enough behavioral assertions to warrant separate tasks? [Granularity]
- [ ] CHK108 - T008 (render verbose fields in human-readable emitter) does not mention the NDJSON emitter. Does T008 need a companion task for verifying verbose fields in NDJSON stdout output (FR-008)? [Granularity]
- [ ] CHK109 - T001 is mapped to both US1-S3 (persistent flag) and SC-003 (--help documentation). Should help text verification be a separate task from flag registration? [Granularity]

## Ordering

- [ ] CHK110 - The dependency graph shows T004 feeds into T007 and T008, but T008 does not depend on T007. Can T008 (rendering) be meaningfully implemented before T007 (field population), or is there a missing dependency? [Ordering]
- [ ] CHK111 - T005 (wire DisplayConfig.VerboseOutput) is listed in Phase 2 but no later task identifies a consumer. Does any Phase 3+ task actually read DisplayConfig.VerboseOutput, or is T005 dangling? [Ordering]
- [ ] CHK112 - Test tasks T014-T018 are grouped in Phase 6, but each depends on its implementation task. Would co-locating tests with implementation phases (e.g., T014 with Phase 1) better reflect the build order and enable earlier validation? [Ordering]
- [ ] CHK113 - T019 (full test suite) depends on T001-T013 but no task captures a build gate between phases. Could integration errors be caught earlier with per-phase build verification? [Ordering]

## Testability

- [ ] CHK114 - T011 states "no code change expected" for validate, but US2-S1 requires specific verbose content (validators, sections, pass/fail). Does the existing validate verbose output already satisfy US2-S1's acceptance criteria, and does T011 verify this? [Testability]
- [ ] CHK115 - Edge case 3 (--verbose with --no-logs) specifies verbose details appear on stderr when stdout is suppressed. No task or test covers this interaction. Is this edge case testable from the current task set? [Testability]
- [ ] CHK116 - Edge case 4 (--verbose in non-TTY/piped environment) specifies plain text without ANSI formatting. No task addresses non-TTY detection. Is this covered by existing infrastructure or does it need a task? [Testability]
- [ ] CHK117 - The data-model.md contract specifies WorkspacePath must be an absolute path and InjectedArtifacts must be filenames only. Does any test task (T015) verify these contract constraints? [Testability]
- [ ] CHK118 - SC-004 maps to T019, which is a single `go test ./... -race` run. Should T019 specify baseline verification (run tests before changes) to confirm any failures are regressions introduced by this feature? [Testability]
