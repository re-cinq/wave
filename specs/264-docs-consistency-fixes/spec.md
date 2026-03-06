# docs: documentation consistency report

**Issue**: [#264](https://github.com/re-cinq/wave/issues/264)
**Labels**: documentation
**Author**: nextlevelshit
**State**: OPEN

## Summary

Documentation consistency report with 25 itemized inconsistencies across 4 severity levels (5 critical, 8 high, 9 medium, 3 low). 8 items already fixed by PR #266, leaving 17 remaining items to address.

## Remaining Items (17)

### Critical (3)

- **DOC-002**: Pipeline count severely outdated — docs say 18, actual count is 47 pipeline YAML files
  - Files: `README.md`, `docs/guide/pipelines.md`
  - Fix: Update counts and add sections for undocumented pipeline families (bb-*, gl-*, gt-*, standalone)

- **DOC-004**: 7 documented pipelines do not exist as YAML files
  - Files: `docs/guide/pipelines.md`
  - Missing: `docs-to-impl`, `docs`, `migrate`, `github-issue-enhancer`, `gh-poor-issues`, `issue-research`, `umami`
  - Fix: Remove references to non-existent pipelines or note them as deprecated/planned

- **DOC-005**: `template` contract type documented but not implemented in code
  - Files: `docs/reference/contract-types.md` (lines 13, 331-345)
  - No `template.go` exists, no `"template"` case in contract switch
  - Fix: Remove from docs or mark as planned/unimplemented

### High (4)

- **DOC-007**: Multiple persona permissions documented incorrectly
  - Files: `docs/guide/personas.md`, `docs/concepts/personas.md`
  - Example: docs show `navigator` with `Bash(git log*)` allowed but wave.yaml shows different permissions; `philosopher` docs show restricted Write but wave.yaml gives full access
  - Fix: Update permission tables to match wave.yaml

- **DOC-008**: Multiple pipeline step counts incorrect
  - Files: `docs/guide/pipelines.md`
  - Fix: Update step counts and step listings for documented pipelines

- **DOC-010**: Event schema documentation missing 13+ fields
  - Files: `docs/reference/events.md`
  - Docs list 9 fields; actual Event struct has 22+ fields (tokens_in, tokens_out, progress, current_action, total_steps, completed_steps, estimated_time_ms, validation_phase, compaction_stats, failure_reason, remediation, tool_name, tool_target, etc.)
  - Fix: Add all missing fields to schema table

- **DOC-011**: Event states 5 of 10 documented
  - Files: `docs/reference/events.md`
  - Documented: started, running, completed, failed, retrying
  - Missing: step_progress, eta_updated, contract_validating, compaction_progress, stream_activity
  - Fix: Add missing states

### Medium (7)

- **DOC-014**: `wave init --reconfigure` and `--all` flags undocumented
  - Files: `docs/reference/cli.md`
  - Both flags exist in `cmd/wave/commands/init.go`

- **DOC-016**: `wave cancel --format` flag undocumented
  - Files: `docs/reference/cli.md`
  - Flag exists per `cancel.go` line 58

- **DOC-017**: Advanced contract configuration fields undocumented
  - Files: `docs/reference/contract-types.md`, `docs/guide/contracts.md`

- **DOC-018**: `NERD_FONT` and `NO_UNICODE` env var types incorrect
  - Files: `docs/reference/environment.md`
  - `NERD_FONT`: code checks `== "1"`, not generic bool
  - `NO_UNICODE`: code checks `!= ""`, any non-empty string enables

- **DOC-019**: Default step timeout inconsistency
  - wave.yaml: `default_timeout_minutes: 90`
  - Onboarding default: `30`
  - Fix: Reconcile and document actual default

- **DOC-020**: README shows `wave run --pipeline <name>` but examples use positional arg
  - Files: `README.md` line 140
  - Fix: Update to `wave run <pipeline>` matching actual usage

- **DOC-021**: No documentation for TUI and WebUI internal packages
  - Fix: Add TUI/WebUI overview docs

- **DOC-022**: Temperature values in docs but commented out in wave.yaml
  - All temperature values in wave.yaml are commented out with `#temperature:`
  - Fix: Note temperature as optional in docs examples, or uncomment in wave.yaml

### Low (3)

- **DOC-024**: Quick start guides inconsistently recommend first pipeline
  - `docs/quickstart.md` recommends `hello-world`
  - `docs/guide/quick-start.md` recommends `speckit-flow`
  - Fix: Standardize to `hello-world` as the recommended first pipeline

- **DOC-025**: Concepts personas page covers only 4 of 30 personas
  - Files: `docs/concepts/personas.md`
  - Fix: Expand or add cross-reference to complete list

## Acceptance Criteria

1. All 17 remaining DOC items are addressed
2. Documentation counts match actual codebase (47 pipelines, 30 personas)
3. Permission tables match wave.yaml definitions
4. Event schema docs match Event struct fields
5. CLI flag documentation matches registered flags
6. Environment variable descriptions match code behavior
7. No broken references to non-existent pipelines or personas
