# Tasks

## Phase 1: Critical Fixes

- [X] Task 1.1: Update pipeline documentation in `docs/guide/pipelines.md` — update count from 18 to actual (47), add pipeline family sections for bb-*, gl-*, gt-* and standalone pipelines, remove 7 non-existent pipeline references (docs-to-impl, docs, migrate, github-issue-enhancer, gh-poor-issues, issue-research, umami), fix step counts for remaining pipelines (DOC-002, DOC-004, DOC-008)
- [X] Task 1.2: Remove `template` contract type from `docs/reference/contract-types.md` — remove the template section (lines 331-345) and its entry from the Quick Reference table since no implementation exists in code (DOC-005)
- [X] Task 1.3: Update pipeline count in `README.md` and fix `wave run` command syntax — update pipeline reference count, change line 140 from `wave run --pipeline <name>` to `wave run <pipeline>` (DOC-002, DOC-020)

## Phase 2: High Priority Fixes

- [X] Task 2.1: Fix persona permissions in `docs/guide/personas.md` — update persona count from 14 to 30, fix permission tables for navigator (add Glob), philosopher (full access not restricted), planner (full access not restricted), craftsman (broad Bash not restricted), debugger (add Glob), summarizer (full access not restricted). Update YAML examples to match wave.yaml. Note temperature as optional with commented-out examples (DOC-007, DOC-022) [P]
- [X] Task 2.2: Fix persona count and permissions in `docs/concepts/personas.md` — update count from 14 to 30, fix permission examples for navigator/auditor/implementer/craftsman to match wave.yaml, add cross-reference to complete persona list (DOC-007, DOC-025) [P]
- [X] Task 2.3: Update event schema documentation in `docs/reference/events.md` — add 13+ missing fields (tokens_in, tokens_out, progress, current_action, total_steps, completed_steps, estimated_time_ms, validation_phase, compaction_stats, failure_reason, remediation, tool_name, tool_target), add 5 missing states (step_progress, eta_updated, contract_validating, compaction_progress, stream_activity) (DOC-010, DOC-011) [P]

## Phase 3: Medium Priority Fixes

- [X] Task 3.1: Add missing CLI flags to `docs/reference/cli.md` — add `--reconfigure` and `--all` to wave init options section, add `--format` to wave cancel options section (DOC-014, DOC-016) [P]
- [X] Task 3.2: Fix environment variable docs in `docs/reference/environment.md` — update NERD_FONT type from "bool" to note it requires "1" to enable, update NO_UNICODE to note any non-empty string enables it, reconcile default timeout (wave.yaml=90min, onboarding=30min) (DOC-018, DOC-019) [P]
- [X] Task 3.3: Document advanced contract configuration in `docs/reference/contract-types.md` and `docs/guide/contracts.md` — ensure all contract fields used in pipeline YAML files are documented (DOC-017) [P]
- [X] Task 3.4: Add TUI documentation — create brief `docs/guide/tui.md` with overview of TUI mode, how to enable/disable, and environment variables (DOC-021) [P]
- [X] Task 3.5: Standardize quick start guide in `docs/guide/quick-start.md` — change recommended first pipeline from speckit-flow to hello-world to match docs/quickstart.md (DOC-024)
- [X] Task 3.6: Note temperature as optional in persona docs — update YAML examples in `docs/guide/personas.md` and `docs/concepts/personas.md` to show temperature as commented-out optional field matching wave.yaml convention (DOC-022)

## Phase 4: Validation

- [X] Task 4.1: Cross-verify all count references — grep for "18 pipelines", "14 personas", "14 built-in" across all docs to ensure no stale counts remain
- [X] Task 4.2: Verify no broken cross-references — check that removed pipeline/persona references don't leave dangling links
- [X] Task 4.3: Run `go test ./...` to confirm no regressions from documentation-only changes
