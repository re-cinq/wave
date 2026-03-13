# Implementation Plan: Pipeline Naming Taxonomy

## Objective

Rename all non-forge built-in pipelines to use consistent category prefixes (audit-, plan-, impl-, doc-, test-, ops-) so they group logically in alphabetical listings and communicate purpose at a glance.

## Approach

**Clean break** — this is prototype phase (per CLAUDE.md: "No backward compatibility constraint during prototype phase"). Rename files, update metadata, fix all references in a single pass. No aliases, no deprecation layer.

### Proposed Name Mapping

**22 non-forge pipelines in `internal/defaults/pipelines/`:**

| Current Name | New Name | Category | Rationale |
|---|---|---|---|
| `adr` | `doc-adr` | doc- | Produces architecture decision records |
| `changelog` | `doc-changelog` | doc- | Generates changelogs from git history |
| `dead-code` | `impl-dead-code` | impl- | Finds and removes dead code (writes code) |
| `debug` | `impl-debug` | impl- | Systematic debugging with hypothesis testing (writes fixes) |
| `doc-audit` | `audit-docs` | audit- | Read-only documentation consistency analysis |
| `doc-fix` | `doc-fix` | doc- | Already prefixed correctly |
| `explain` | `doc-explain` | doc- | Deep-dive explanations of code/architecture |
| `feature` | `impl-feature` | impl- | Plans, implements, tests, and commits a feature |
| `hello-world` | `ops-hello-world` | ops- | Smoke test / example pipeline |
| `hotfix` | `impl-hotfix` | impl- | Quick investigation and fix |
| `improve` | `impl-improve` | impl- | Analyze and apply targeted improvements |
| `onboard` | `doc-onboard` | doc- | Generates onboarding guide |
| `plan` | `plan-feature` | plan- | Breaks down features into tasks |
| `prototype` | `impl-prototype` | impl- | Prototype-driven development |
| `recinq` | `impl-recinq` | impl- | Double Diamond code simplification |
| `refactor` | `impl-refactor` | impl- | Safe refactoring with test coverage |
| `security-scan` | `audit-security` | audit- | Security vulnerability audit |
| `smoke-test` | `test-smoke` | test- | Minimal pipeline for contract/artifact testing |
| `speckit-flow` | `plan-speckit` | plan- | Full speckit workflow (spec→plan→tasks→implement) |
| `supervise` | `audit-supervise` | audit- | Review work and process quality |
| `test-gen` | `test-gen` | test- | Already prefixed correctly |

**No change needed**: `doc-fix`, `test-gen` (already follow taxonomy).

**20 renames total** across defaults. Plus corresponding `.wave/pipelines/` user-space files.

### Forge pipelines (EXCLUDED)

All `gh-*`, `gl-*`, `bb-*`, `gt-*` pipelines are excluded. They will be handled by #241 (wave flavours).

## File Mapping

### Files to rename (create new + delete old)

**Pipeline YAML files** (`internal/defaults/pipelines/`):
- `adr.yaml` → `doc-adr.yaml`
- `changelog.yaml` → `doc-changelog.yaml`
- `dead-code.yaml` → `impl-dead-code.yaml`
- `debug.yaml` → `impl-debug.yaml`
- `doc-audit.yaml` → `audit-docs.yaml`
- `explain.yaml` → `doc-explain.yaml`
- `feature.yaml` → `impl-feature.yaml`
- `hello-world.yaml` → `ops-hello-world.yaml`
- `hotfix.yaml` → `impl-hotfix.yaml`
- `improve.yaml` → `impl-improve.yaml`
- `onboard.yaml` → `doc-onboard.yaml`
- `plan.yaml` → `plan-feature.yaml`
- `prototype.yaml` → `impl-prototype.yaml`
- `recinq.yaml` → `impl-recinq.yaml`
- `refactor.yaml` → `impl-refactor.yaml`
- `security-scan.yaml` → `audit-security.yaml`
- `smoke-test.yaml` → `test-smoke.yaml`
- `speckit-flow.yaml` → `plan-speckit.yaml`
- `supervise.yaml` → `audit-supervise.yaml`

**Prompt directories** (`internal/defaults/prompts/`):
- `speckit-flow/` → `plan-speckit/`

### Files to modify (update references)

**Go source with hardcoded pipeline names**:
- `internal/pipeline/validation.go` — `"prototype"` → `"impl-prototype"` (3 occurrences)
- `internal/pipeline/resume.go` — `"prototype"` → `"impl-prototype"` (1 occurrence)

**Tests with pipeline name strings**:
- `internal/pipeline/validation_test.go` — prototype references
- `internal/pipeline/resume_test.go` — prototype, speckit-flow references
- `internal/pipeline/prototype_dummy_test.go` — prototype references
- `internal/pipeline/prototype_e2e_test.go` — prototype references
- `internal/pipeline/prototype_implement_test.go` — prototype references
- `internal/pipeline/composition_test.go` — speckit-flow, hotfix references
- `internal/doctor/optimize_test.go` — speckit-flow, doc-audit references
- `internal/recovery/recovery_test.go` — speckit-flow, feature references
- `internal/recovery/format_test.go` — feature references
- `internal/tui/issue_detail_test.go` — speckit-flow references
- `cmd/wave/commands/doctor_test.go` — speckit-flow reference

**Documentation**:
- `docs/guide/pipelines.md` — all pipeline name references + taxonomy section
- `docs/guide/quick-start.md` — hello-world references
- `docs/guide/tui.md` — speckit-flow references
- `CLAUDE.md` — Pipeline Selection table

**Suggest engine** (no changes needed — uses `resolvePipeline()` with base names like "debug", "improve", "refactor" which are resolved dynamically, not hardcoded to full names).

## Architecture Decisions

1. **`doc-audit` → `audit-docs`**: The current name is `doc-audit` which suggests it's a documentation pipeline. But its function is auditing (read-only analysis). Rename to `audit-docs` to correctly reflect it's an audit that targets docs.

2. **`speckit-flow` → `plan-speckit`**: Despite having an implementation step, the pipeline's identity is spec-driven planning. The `plan-` prefix communicates "start here for new features."

3. **`prototype` → `impl-prototype`**: This pipeline has special behavior hardcoded in `validation.go` and `resume.go`. The string `"prototype"` must be updated everywhere. This is the highest-risk rename.

4. **No aliases/redirects**: Clean break. No backward compat mapping. Tests and docs are the safety net.

5. **`.wave/pipelines/` user-space files**: Updated in the same PR for consistency. These are not embedded but are versioned in the repo.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| `prototype` hardcoded in validation/resume logic | Pipeline breaks at runtime | Thorough grep + tests cover these paths |
| Prompt directory rename breaks prompt injection | Steps lose their external prompts | Rename directory + verify embed works |
| User scripts/muscle memory breaking | Users type old names | Clean break per CLAUDE.md policy; update docs |
| Missing a reference somewhere | Runtime errors | `go test ./...` catches most; grep audit before merge |
| `.wave/pipelines/` divergence from defaults | Confusion | Update both in same commit |

## Testing Strategy

1. **Existing test suite**: `go test ./...` must pass — this catches most hardcoded string mismatches
2. **Race detector**: `go test -race ./...` as required for PR
3. **Embed verification**: Ensure `internal/defaults/embed_test.go` passes (validates all embedded files load)
4. **Manual verification**: `wave list` output should show pipelines grouped by prefix
5. **Grep audit**: Final `grep -r` for old names to catch stragglers
