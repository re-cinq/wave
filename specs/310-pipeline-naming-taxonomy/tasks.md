# Tasks

## Phase 1: Pipeline File Renames

Rename pipeline YAML files in `internal/defaults/pipelines/` and update `metadata.name` inside each.

- [X] Task 1.1: Rename audit-category pipelines [P]
  - `doc-audit.yaml` → `audit-docs.yaml` (update metadata.name to `audit-docs`)
  - `security-scan.yaml` → `audit-security.yaml` (update metadata.name)
  - `supervise.yaml` → `audit-supervise.yaml` (update metadata.name)
- [X] Task 1.2: Rename plan-category pipelines [P]
  - `plan.yaml` → `plan-feature.yaml` (update metadata.name to `plan-feature`)
  - `speckit-flow.yaml` → `plan-speckit.yaml` (update metadata.name)
- [X] Task 1.3: Rename impl-category pipelines [P]
  - `dead-code.yaml` → `impl-dead-code.yaml` (update metadata.name)
  - `debug.yaml` → `impl-debug.yaml` (update metadata.name)
  - `feature.yaml` → `impl-feature.yaml` (update metadata.name)
  - `hotfix.yaml` → `impl-hotfix.yaml` (update metadata.name)
  - `improve.yaml` → `impl-improve.yaml` (update metadata.name)
  - `prototype.yaml` → `impl-prototype.yaml` (update metadata.name)
  - `recinq.yaml` → `impl-recinq.yaml` (update metadata.name)
  - `refactor.yaml` → `impl-refactor.yaml` (update metadata.name)
- [X] Task 1.4: Rename doc-category pipelines [P]
  - `adr.yaml` → `doc-adr.yaml` (update metadata.name)
  - `changelog.yaml` → `doc-changelog.yaml` (update metadata.name)
  - `explain.yaml` → `doc-explain.yaml` (update metadata.name)
  - `onboard.yaml` → `doc-onboard.yaml` (update metadata.name)
- [X] Task 1.5: Rename test-category pipelines [P]
  - `smoke-test.yaml` → `test-smoke.yaml` (update metadata.name)
- [X] Task 1.6: Rename ops-category pipelines [P]
  - `hello-world.yaml` → `ops-hello-world.yaml` (update metadata.name)

## Phase 2: Prompt Directory Rename

- [X] Task 2.1: Rename `internal/defaults/prompts/speckit-flow/` → `internal/defaults/prompts/plan-speckit/`

## Phase 3: Go Source Hardcoded References

- [X] Task 3.1: Update `internal/pipeline/validation.go` — change `"prototype"` → `"impl-prototype"` (3 occurrences in PhaseSkipValidator + workspace path)
- [X] Task 3.2: Update `internal/pipeline/resume.go` — change `"prototype"` → `"impl-prototype"` (line 543)
- [X] Task 3.3: Update `internal/pipeline/validation.go` — change `"prototype"` workspace path references

## Phase 4: Test Fixture Updates

- [X] Task 4.1: Update `internal/pipeline/validation_test.go` — `"prototype"` → `"impl-prototype"` [P]
- [X] Task 4.2: Update `internal/pipeline/resume_test.go` — `"prototype"` → `"impl-prototype"`, `"speckit-flow"` → `"plan-speckit"` [P]
- [X] Task 4.3: Update `internal/pipeline/prototype_dummy_test.go` — `"prototype"` references [P]
- [X] Task 4.4: Update `internal/pipeline/prototype_e2e_test.go` — `"prototype"` references [P]
- [X] Task 4.5: Update `internal/pipeline/prototype_implement_test.go` — `"prototype"` references [P]
- [X] Task 4.6: Update `internal/pipeline/composition_test.go` — `"hotfix"` → `"impl-hotfix"`, `"speckit-flow"` → `"plan-speckit"` [P]
- [X] Task 4.7: Update `internal/doctor/optimize_test.go` — `"speckit-flow"` → `"plan-speckit"`, `"doc-audit"` → `"audit-docs"` [P]
- [X] Task 4.8: Update `internal/recovery/recovery_test.go` — `"speckit-flow"` → `"plan-speckit"`, `"feature"` → `"impl-feature"` [P]
- [X] Task 4.9: Update `internal/recovery/format_test.go` — `"feature"` → `"impl-feature"` [P]
- [X] Task 4.10: Update `internal/tui/issue_detail_test.go` — `"speckit-flow"` → `"plan-speckit"` [P]
- [X] Task 4.11: Update `cmd/wave/commands/doctor_test.go` — `"speckit-flow"` → `"plan-speckit"` [P]
- [X] Task 4.12: Update `internal/adapter/mock.go` — `"prototype"` → `"impl-prototype"` in mock data [P]

## Phase 5: User-Space Pipeline Renames

Rename `.wave/pipelines/` files to match the new taxonomy (same mapping as Phase 1, plus local-only pipelines).

- [X] Task 5.1: Rename `.wave/pipelines/` files matching defaults taxonomy [P]
- [X] Task 5.2: Categorize local-only `.wave/pipelines/` files:
  - `consolidate.yaml` → `impl-consolidate.yaml`
  - `dead-code-issue.yaml` → `audit-dead-code-issue.yaml`
  - `dead-code-review.yaml` → `audit-dead-code-review.yaml`
  - `dual-analysis.yaml` → `audit-dual-analysis.yaml`
  - `dx-audit.yaml` → `audit-dx.yaml`
  - `epic-runner.yaml` → `ops-epic-runner.yaml`
  - `junk-code.yaml` → `audit-junk-code.yaml`
  - `quality-loop.yaml` → `ops-quality-loop.yaml`
  - `release-harden.yaml` → `ops-release-harden.yaml`
  - `research-implement.yaml` → `impl-research.yaml`
  - `ux-audit.yaml` → `audit-ux.yaml`
  - `wave-*` pipelines: keep as-is (already prefixed)
  - `gh-implement-epic.yaml`: keep as-is (forge-prefixed)

## Phase 6: Documentation Updates

- [X] Task 6.1: Update `docs/guide/pipelines.md` — all pipeline name references + add taxonomy section [P]
- [X] Task 6.2: Update `docs/guide/quick-start.md` — `hello-world` → `ops-hello-world` [P]
- [X] Task 6.3: Update `docs/guide/tui.md` — `speckit-flow` → `plan-speckit` [P]
- [X] Task 6.4: Update `CLAUDE.md` — Pipeline Selection table (speckit-flow, hotfix, etc.) [P]

## Phase 7: Validation

- [X] Task 7.1: Run `go test ./...` and fix any failures
- [X] Task 7.2: Run `go test -race ./...`
- [X] Task 7.3: Run `golangci-lint run ./...` (skipped — not available in sandbox)
- [X] Task 7.4: Grep audit for remaining old pipeline names
- [X] Task 7.5: Verify `internal/defaults/embed_test.go` passes (embedded file loading)
