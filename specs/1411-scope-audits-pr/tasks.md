# Work Items

## Phase 1: Setup
- [X] Item 1.1: Create feature branch `1411-scope-audits-pr` from `main` (already done)
- [X] Item 1.2: Re-read `ops-pr-respond.yaml`, `impl-finding.yaml`, `triaged-findings.schema.json` to confirm exact line anchors before editing
- [X] Item 1.3: Locate any existing fixtures / tests that exercise `triaged-findings.schema.json` so additions don't break them — none found; schema not referenced in Go code

## Phase 2: Schema work (foundation)
- [X] Item 2.1: Extend `internal/defaults/contracts/triaged-findings.schema.json` with optional `design_questions[]` array (id, description, question, source_finding_id, suggested_followup)
- [X] Item 2.2: Mirror the schema change to `.agents/contracts/triaged-findings.schema.json`
- [X] Item 2.3: Create `internal/defaults/contracts/scope-filter-stats.schema.json` (per-axis kept/dropped counts + total + dropped sample)
- [X] Item 2.4: Mirror new schema to `.agents/contracts/scope-filter-stats.schema.json`
- [X] Item 2.5: Schema fixtures unchanged — additions are optional, existing fixtures remain valid (covered indirectly by `internal/contract` tests)

## Phase 3: Pipeline edits — ops-pr-respond
- [X] Item 3.1: Edit `internal/defaults/pipelines/ops-pr-respond.yaml` `parallel-review` step — add `config.inject: ["pr-context"]` so each audit child receives `.agents/artifacts/pr-context`
- [X] Item 3.2: Insert new `filter-scope` step (between `merge-findings` and `triage`) — `command` type, deterministic `jq` filter, output `.agents/output/scoped-findings.json` and `.agents/output/scope-filter-stats.json`, handover contract `json_schema` against `scope-filter-stats.schema.json`
- [X] Item 3.3: Rewire `triage.dependencies` → `[filter-scope, fetch-pr]`; switch its `inject_artifacts` source from `merged-findings` to `scoped-findings`
- [X] Item 3.4: Update `triage` prompt — drop "verify scope" step, add "populate design_questions[] for deferred-with-design entries", short-circuit on empty input array
- [X] Item 3.5: Update `comment-back` prompt — add "Design Questions" section template; update sanitisation cap-shedding order to drop `design_questions` last (after deferred/rejected)
- [X] Item 3.6: Mirror all edits to `.agents/pipelines/ops-pr-respond.yaml`

## Phase 4: Pipeline edits — audit-* fleet [P]
Each audit-* pipeline gets the same conditional PR-scope preamble injected at the top of its `scan` step prompt. All six can be edited in parallel.
- [X] Item 4.1: `internal/defaults/pipelines/audit-security.yaml` + mirror [P]
- [X] Item 4.2: `internal/defaults/pipelines/audit-architecture.yaml` (preamble in `internal/defaults/prompts/audit/architecture-scan.md` since pipeline uses `source_path`) + mirror [P]
- [X] Item 4.3: `internal/defaults/pipelines/audit-tests.yaml` (preamble in `internal/defaults/prompts/audit/tests-scan.md`) + mirror [P]
- [X] Item 4.4: `internal/defaults/pipelines/audit-duplicates.yaml` + mirror [P]
- [X] Item 4.5: `internal/defaults/pipelines/audit-doc-scan.yaml` + mirror [P]
- [X] Item 4.6: `internal/defaults/pipelines/audit-dead-code-scan.yaml` + mirror [P]

## Phase 5: Pipeline edits — impl-finding hardening
- [X] Item 5.1: Edit `internal/defaults/pipelines/impl-finding.yaml` apply-fix prompt — explicit instruction "do NOT shell out to `gh pr view` or `gh pr diff`; the parent injected `.agents/artifacts/pr-context`"
- [X] Item 5.2: Mirror to `.agents/pipelines/impl-finding.yaml`

## Phase 6: Testing
- [X] Item 6.1: `go test ./internal/pipeline/... -run Load` — passed
- [X] Item 6.2: `go test ./internal/contract/...` — passed (schema registry resolves both new schemas)
- [X] Item 6.3: Contract registry test — passed (additive schema field, no fixture changes needed)
- [X] Item 6.4: `go test ./...` (full suite, no `-race` per project default) — all packages pass
- [ ] Item 6.5: Build wave binary — deferred to validation step (separate run after merge per project rule)
- [ ] Item 6.6: Run `wave run ops-pr-respond <pr-url>` against a representative ≤10-file PR — deferred to post-merge validation per `feedback_pipeline_validation_means_run.md`
- [ ] Item 6.7: Run one standalone `wave run audit-security <topic>` — deferred to post-merge validation

## Phase 7: Polish
- [ ] Item 7.1: AGENTS.md update — not needed; audit pipelines unchanged for standalone use, parent injection is documented in pipeline YAML comments
- [ ] Item 7.2: `git diff --stat` review — handled at commit time
- [ ] Item 7.3: No `.wave/`, `.claude/`, `.agents/output/`, `.agents/artifacts/` staged — handled by `git reset HEAD` step
- [ ] Item 7.4: Conventional commit: `feat(pipelines): scope audits to PR diff and surface design questions`
- [ ] Item 7.5: Push branch + open PR — handled by next pipeline step
