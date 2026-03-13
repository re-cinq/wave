# Tasks

## Phase 1: Critical Fixes

- [X] Task 1.1: [DOC-001] Remove `skills` from manifest-schema.md top-level fields table and SkillConfig section. Reframe as pipeline-level `requires.skills` documentation. Remove `skills:` block from the Complete Example at the bottom of the file.
- [X] Task 1.2: [DOC-002] Verify `on_failure` on `ContractConfig` — confirm it exists in code at `internal/pipeline/types.go:256` and mark as resolved (likely no doc change needed). If the contract-types.md examples show `on_failure` correctly, no edit required.
- [X] Task 1.3: [DOC-003] Remove phantom platform pipeline sections (gl-*, gt-*, bb-*) from `docs/guide/pipelines.md`. These 18 pipelines are documented but have no corresponding YAML files.

## Phase 2: High-Severity Fixes [P]

- [X] Task 2.1: [DOC-004] Add `--preserve-workspace` flag to `wave run` options in `docs/reference/cli.md` [P]
- [X] Task 2.2: [DOC-005] Add `wave migrate validate` subcommand to migrate section in `docs/reference/cli.md` [P]
- [X] Task 2.3: [DOC-006] Add `sandbox` field to Persona table in `docs/reference/manifest-schema.md`, including `PersonaSandbox.allowed_domains` sub-field [P]
- [X] Task 2.4: [DOC-007] Add `match_labels` to RoutingRule table, and `context_window`/`summarizer_persona` to RelayConfig table in `docs/reference/manifest-schema.md` [P]
- [X] Task 2.5: [DOC-008] Update pipeline count from "47" to "46" in `docs/guide/pipelines.md` and `README.md`

## Phase 3: Medium-Severity Fixes [P]

- [X] Task 3.1: [DOC-009] Add 17 undocumented pipelines to `docs/guide/pipelines.md` — consolidate, dead-code-issue, dead-code-review, dual-analysis, dx-audit, epic-runner, junk-code, quality-loop, release-harden, research-implement, ux-audit, wave-audit, wave-bugfix, wave-evolve, wave-review, wave-security-audit, wave-test-hardening [P]
- [X] Task 3.2: [DOC-010] Verify `--manifest` flag on `wave do` — already documented as global flag; confirm and close [P]
- [X] Task 3.3: [DOC-011] Document `GH_TOKEN`/`GITHUB_TOKEN` precedence order in `docs/reference/environment.md` [P]
- [X] Task 3.4: [DOC-012] Clarify persona count in README — 31 files but 30 personas (base-protocol.md is not a persona) [P]

## Phase 4: Low-Severity & Validation

- [X] Task 4.1: [DOC-013] Note in PR that comprehensive persona documentation (22 of 30 undocumented) is a separate effort — out of scope for this PR
- [X] Task 4.2: [DOC-014] Investigate `reviewer.yaml` in `internal/defaults/personas/` — verify it follows the same pattern as other persona YAML configs (it does — all personas have `.yaml` + `.md` pairs). Mark as not anomalous
- [X] Task 4.3: Run `go test ./...` to verify no regressions
- [X] Task 4.4: Final review — cross-reference all 14 DOC items against changes, verify markdown links resolve
