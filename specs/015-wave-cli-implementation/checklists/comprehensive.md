# Comprehensive Requirements Quality Checklist: Wave CLI Implementation

**Purpose**: Author self-check covering all areas (CLI, Security, Reliability) with probes for recovery/rollback gaps
**Created**: 2026-02-02
**Feature**: [spec.md](../spec.md)

## Requirement Completeness

- [ ] CHK001 - Are all 7 CLI commands (init, validate, run, do, list, resume, clean) specified with complete flag sets? [Completeness, Spec §FR-001 to §FR-007]
- [ ] CHK002 - Are requirements for all 7 built-in personas documented with their specific permission sets? [Completeness, Spec §Key Entities]
- [ ] CHK003 - Are workspace lifecycle requirements complete (creation, mounting, artifact injection, cleanup)? [Completeness, Spec §FR-016, §FR-027]
- [ ] CHK004 - Are all contract validation types (JSON schema, TypeScript, test suite) fully specified? [Completeness, Spec §FR-018]
- [ ] CHK005 - Are requirements for the Claude Code adapter subprocess invocation complete? [Completeness, Spec §FR-038 to §FR-040]
- [ ] CHK006 - Are audit logging requirements complete (what is logged, what is excluded)? [Completeness, Spec §FR-035 to §FR-037]

## Requirement Clarity

- [ ] CHK007 - Is "80% context utilization threshold" quantified with specific token counting method? [Clarity, Spec §US-7]
- [ ] CHK008 - Is "per-step timeout" specified with default value and configuration method? [Clarity, Spec §FR-020]
- [ ] CHK009 - Is "max_retries" default value and range explicitly defined? [Clarity, Spec §FR-019]
- [ ] CHK010 - Is the glob pattern syntax for permission allow/deny patterns documented? [Clarity, Spec §FR-021, §FR-022]
- [ ] CHK011 - Is "structured progress events" format (JSON schema) explicitly defined? [Clarity, Spec §FR-035]
- [ ] CHK012 - Is the checkpoint document structure for relay resumption specified? [Clarity, Spec §FR-031]
- [ ] CHK013 - Is "topological order" for DAG execution unambiguously defined for parallel branches? [Clarity, Spec §FR-013]

## Requirement Consistency

- [ ] CHK014 - Are permission enforcement requirements consistent between FR-021/FR-022 and User Story 5 scenarios? [Consistency]
- [ ] CHK015 - Are retry semantics consistent across contract failures, subprocess crashes, and timeouts? [Consistency, Spec §FR-019]
- [ ] CHK016 - Are workspace paths consistent between runtime config and step definitions? [Consistency]
- [ ] CHK017 - Are artifact injection requirements consistent between pipeline YAML schema and executor behavior? [Consistency, Spec §FR-017]
- [ ] CHK018 - Is error message format consistent across all validation and execution failures? [Consistency]

## Acceptance Criteria Quality

- [ ] CHK019 - Can SC-001 "under 10 minutes" be objectively measured with defined start/end points? [Measurability, Spec §SC-001]
- [ ] CHK020 - Can SC-002 "100% of configuration errors" be verified with a defined error taxonomy? [Measurability, Spec §SC-002]
- [ ] CHK021 - Can SC-005 "100% of denied tool calls" be tested with comprehensive deny pattern coverage? [Measurability, Spec §SC-005]
- [ ] CHK022 - Can SC-010 "10 parallel workers without resource contention" be measured? [Measurability, Spec §SC-010]
- [ ] CHK023 - Are all 12 user stories' acceptance scenarios written in testable Given/When/Then format? [Measurability]

## Scenario Coverage - Primary Flows

- [ ] CHK024 - Are requirements defined for pipeline execution with zero steps? [Coverage, Gap]
- [ ] CHK025 - Are requirements defined for `wave do` with empty task description? [Coverage, Gap]
- [ ] CHK026 - Are requirements defined for `wave run` with non-existent pipeline name? [Coverage, Gap]
- [ ] CHK027 - Are requirements defined for `wave resume` when pipeline already completed? [Coverage, Gap]
- [ ] CHK028 - Are requirements defined for concurrent `wave run` of the same pipeline? [Coverage, Gap]

## Scenario Coverage - Alternate Flows

- [ ] CHK029 - Are requirements defined for `wave init --merge` behavior with partial existing config? [Coverage, Spec §US-1]
- [ ] CHK030 - Are requirements defined for `wave validate` with warnings-only (no errors)? [Coverage]
- [ ] CHK031 - Are requirements defined for pipeline execution when optional artifacts are missing? [Coverage, Gap]
- [ ] CHK032 - Are requirements defined for matrix execution with zero tasks from items_source? [Coverage, Gap]

## Edge Case Coverage

- [ ] CHK033 - Are requirements defined for adapter binary found but not executable? [Edge Case, Gap]
- [ ] CHK034 - Are requirements defined for disk full during workspace creation? [Edge Case, Gap]
- [ ] CHK035 - Are requirements defined for SQLite database corruption during state persistence? [Edge Case, Gap]
- [ ] CHK036 - Are requirements defined for YAML with valid syntax but invalid schema? [Edge Case, partially in §US-2]
- [ ] CHK037 - Are requirements defined for circular artifact dependencies between steps? [Edge Case, Gap]
- [ ] CHK038 - Are requirements defined for system prompt file changed mid-execution? [Edge Case, Gap]

## Recovery & Rollback Requirements (Probing for Gaps)

- [ ] CHK039 - Are rollback requirements defined for partially completed `wave init`? [Recovery, Gap]
- [ ] CHK040 - Are recovery requirements defined for workspace mount failures mid-pipeline? [Recovery, Gap]
- [ ] CHK041 - Are cleanup requirements defined for orphaned workspaces after unclean shutdown? [Recovery, Gap]
- [ ] CHK042 - Are requirements defined for state database migration between Wave versions? [Recovery, Gap]
- [ ] CHK043 - Are requirements defined for recovering from relay compaction failures? [Recovery, Gap]
- [ ] CHK044 - Are requirements defined for matrix worker crash impact on other workers? [Recovery, Spec §US-9 partial]
- [ ] CHK045 - Are requirements defined for adapter subprocess hanging (not crashing)? [Recovery, partially in §FR-020]

## Security Requirements

- [ ] CHK046 - Are requirements clear on what constitutes "credentials" for audit log scrubbing? [Clarity, Spec §FR-037]
- [ ] CHK047 - Are requirements defined for permission bypass attempts via crafted tool arguments? [Security, Gap]
- [ ] CHK048 - Are requirements defined for workspace isolation between concurrent pipelines? [Security, Gap]
- [ ] CHK049 - Are requirements defined for audit log tampering prevention? [Security, Gap]
- [ ] CHK050 - Are requirements defined for hook script injection risks? [Security, Gap]
- [ ] CHK051 - Are requirements clear that credentials never appear in progress events? [Security, Spec §FR-035, §FR-037]

## Non-Functional Requirements

- [ ] CHK052 - Is "single static binary" constraint verifiable with specific build requirements? [NFR, Spec §SC-009]
- [ ] CHK053 - Are memory usage requirements specified for large pipeline executions? [NFR, Gap]
- [ ] CHK054 - Are disk space requirements specified for workspace and state storage? [NFR, Gap]
- [ ] CHK055 - Is startup time requirement specified for CLI commands? [NFR, Gap]
- [ ] CHK056 - Are requirements defined for graceful degradation under resource pressure? [NFR, Gap]

## Dependencies & Assumptions

- [ ] CHK057 - Is the assumption "adapter CLIs maintain stable interfaces" testable or version-pinned? [Assumption, Spec §Assumptions]
- [ ] CHK058 - Is the assumption about compilation tools availability validated with fallback behavior? [Assumption, Spec §Assumptions]
- [ ] CHK059 - Are SQLite version requirements documented? [Dependency, Gap]
- [ ] CHK060 - Are Go version requirements for building documented beyond "1.22+"? [Dependency, Spec §Assumptions]

## Ambiguities & Conflicts

- [ ] CHK061 - Is "fresh memory at every step boundary" reconciled with artifact injection requirements? [Ambiguity]
- [ ] CHK062 - Is the relationship between `--from-step` and state persistence clearly defined? [Ambiguity, Spec §US-4, §US-6]
- [ ] CHK063 - Are handover contract requirements clear on whether contracts are optional or mandatory per step? [Ambiguity, Spec §FR-018]
- [ ] CHK064 - Is the meta-pipeline recursion depth requirement consistent with max_total_steps? [Ambiguity, Spec §US-12]

## Notes

- Focus areas: All (CLI, Security, Reliability)
- Depth level: Author self-check (lightweight)
- Probing for: Additional recovery/rollback requirements
- Items marked [Gap] indicate potential missing requirements to address before planning
- Items marked [Ambiguity] may need clarification in spec before implementation
