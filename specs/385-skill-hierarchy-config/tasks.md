# Tasks: Hierarchical Skill Configuration

**Feature**: #385 — Skill Hierarchy Config
**Branch**: `385-skill-hierarchy-config`
**Generated**: 2026-03-14
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

---

## Phase 1: Data Model — YAML Struct Fields (FR-001, FR-002, FR-003, FR-010)

- [X] T001 [P1] [US1] Add `Skills []string` field to `Manifest` struct — `internal/manifest/types.go:16` — Add `Skills []string \`yaml:"skills,omitempty"\`` to the `Manifest` struct after the `Personas` field. Verify absent/null/empty YAML all parse to nil slice.
- [X] T002 [P1] [US2] Add `Skills []string` field to `Persona` struct — `internal/manifest/types.go:41` — Add `Skills []string \`yaml:"skills,omitempty"\`` to the `Persona` struct after the `Sandbox` field.
- [X] T003 [P2] [US3] Add `Skills []string` field to `Pipeline` struct — `internal/pipeline/types.go:21` — Add `Skills []string \`yaml:"skills,omitempty"\`` to the `Pipeline` struct after `ChatContext`. This is a top-level field (NOT under `Requires`), per C1 resolution.

## Phase 2: Skill Resolution Function (FR-004, FR-005, FR-009, FR-012)

- [X] T004 [P1] [US5] Create `ResolveSkills` function — `internal/skill/resolve.go` (NEW) — Implement `func ResolveSkills(global, persona, pipeline []string) []string` that merges three scope lists into a single deduplicated, alphabetically sorted slice. Iterate pipeline first (highest precedence), then persona, then global using a `map[string]bool` for dedup. Return sorted result.
- [X] T005 [P1] [US5] Add table-driven tests for `ResolveSkills` — `internal/skill/resolve_test.go` (NEW) — Cover: (1) all empty → empty, (2) single scope → sorted, (3) all three scopes → merged/deduped/sorted, (4) duplicate across scopes → appears once, (5) same input order → deterministic output, (6) large input with many duplicates.

## Phase 3: Skill Validation (FR-006, FR-007, FR-011)

- [X] T006 [P1] [US4] Create `ValidateSkillRefs` function — `internal/skill/validate.go` (NEW) — Implement `func ValidateSkillRefs(names []string, scope string, store Store) []error` that validates each name via `ValidateName()` for format, then checks existence via `store.Read(name)` if store is non-nil. Aggregate all errors (don't fail-fast). Each error message must include the scope label and invalid name.
- [X] T007 [P1] [US4] Add table-driven tests for `ValidateSkillRefs` — `internal/skill/validate_test.go` (NEW) — Cover: (1) empty names → no errors, (2) valid names with nil store → no errors, (3) invalid format name → format error with scope, (4) valid format but nonexistent in store → existence error with scope, (5) multiple invalid names → all reported, (6) mixed valid/invalid → only invalid reported, (7) store is nil → only format validation runs.

## Phase 4: Manifest Validation Integration (FR-007, FR-011, SC-003, SC-006)

- [X] T008 [P1] [US4] Create `ValidateManifestSkills` function — `internal/skill/validate.go` — Add `func ValidateManifestSkills(skills []string, personas map[string]struct{ Skills []string }, store Store) []error` that calls `ValidateSkillRefs` for global skills (scope="global") and each persona's skills (scope="persona:<name>"). Return aggregated errors from all scopes.
- [X] T009 [P1] [US4] Integrate manifest skill validation into `Load()` — `internal/manifest/parser.go` — After `ValidateWithFile` in the `Load()` function, call `skill.ValidateManifestSkills` with the parsed `Manifest.Skills` and persona skills. The `Load()` function needs a `skill.Store` parameter (or create a new `LoadWithStore` wrapper). Append any skill validation errors to the error list.
- [X] T010 [P] [P1] [US4] Add integration tests for manifest skill validation — `internal/manifest/parser_test.go` — Test: (1) manifest with valid global skills passes validation, (2) manifest with invalid skill name format produces error, (3) manifest with nonexistent skill produces error referencing scope, (4) persona with invalid skill produces error with "persona:<name>" scope, (5) errors aggregated across global and persona scopes, (6) absent `skills:` field → no error.

## Phase 5: Pipeline Validation Integration (FR-007)

- [X] T011 [P] [P2] [US4] Add pipeline skill validation to pipeline loading path — `internal/pipeline/dag.go` — In `ValidateDAG` (or a new `ValidatePipelineSkills` function called from executor), validate `pipeline.Skills` via `ValidateSkillRefs(pipeline.Skills, "pipeline:<name>", store)`. The caller needs to pass the store. Since `ValidateDAG` currently takes only `*Pipeline`, either add a new method or validate separately in the executor before execution.
- [X] T012 [P] [P2] [US4] Add tests for pipeline skill validation — `internal/pipeline/dag_test.go` — Test: (1) pipeline with valid skills passes, (2) pipeline with invalid skill format → error, (3) pipeline with nonexistent skill → error with "pipeline:" scope, (4) pipeline with no skills field → no error.

## Phase 6: Executor Integration — Per-Step Resolution & Provisioning (FR-012, FR-013, FR-009)

- [X] T013 [P1] [US1,US2,US3,US5] Add per-step skill resolution to `buildAdapterRunConfig` — `internal/pipeline/executor.go:1192` — Before the existing skill provisioning block, resolve skills from all three scopes: `global := execution.Manifest.Skills`, `personaSkills := persona.Skills` (from the resolved persona), `pipelineSkills := append(execution.Pipeline.Skills, execution.Pipeline.Requires.SkillNames()...)`. Call `skill.ResolveSkills(global, personaSkills, pipelineSkills)` to get the merged set. Use this resolved set for provisioning instead of only `Requires.SkillNames()`.
- [X] T014 [P1] [US1,US2,US3] Provision DirectoryStore skills for name-only references — `internal/pipeline/executor.go` — For each skill in the resolved set that is NOT in `requires.skills`, call `DirectoryStore.Read(name)` to get the `Skill` struct. Write the skill's `Body` content as a SKILL.md file into the workspace. Copy any resource files from `Skill.ResourcePaths`. The executor needs a `skill.Store` dependency (add to `Executor` struct or pass via `ExecutionContext`).
- [X] T015 [P1] [US5] Ensure `requires.skills` + new `skills:` unification works — `internal/pipeline/executor.go` — When both `requires.skills` has a key and the new `skills:` list has the same name, the resolved set should contain the name once. The `SkillConfig` from `requires.skills` drives install/check/init via preflight, while DirectoryStore drives SKILL.md provisioning. Existing `Provisioner.Provision()` call should use skill names from `requires.skills` only (for command file provisioning), while DirectoryStore provisioning handles the rest.

## Phase 7: Backward Compatibility & Round-Trip Tests (SC-001, SC-004)

- [X] T016 [P] [P1] [US1] Add YAML round-trip test for Manifest with `skills:` — `internal/manifest/types_test.go` or new test file — Marshal a `Manifest` with `Skills: ["speckit", "golang"]`, unmarshal back, verify identical. Also test with empty/nil skills.
- [X] T017 [P] [P2] [US2] Add YAML round-trip test for Persona with `skills:` — Same test file — Marshal a `Persona` with `Skills: ["speckit"]`, unmarshal back, verify identical.
- [X] T018 [P] [P2] [US3] Add YAML round-trip test for Pipeline with `skills:` — `internal/pipeline/types_test.go` — Marshal a `Pipeline` with `Skills: ["golang", "testing"]`, unmarshal back, verify identical. Also verify existing `Requires.Skills` map is unaffected.
- [X] T019 [P] [P1] [US1] Backward compatibility test — existing pipelines without `skills:` — `internal/pipeline/executor_test.go` or integration test — Verify that a pipeline with no `skills:` fields at any scope produces the same behavior as before (empty resolved set, existing `requires.skills` flow unchanged).

## Phase 8: Edge Case Tests (Edge Cases from spec.md)

- [X] T020 [P] [P3] [US4] Test invalid skill name characters at all scopes — `internal/skill/validate_test.go` — Verify `ValidateSkillRefs` catches names with spaces, uppercase, special chars, and names exceeding 64 chars at all three scopes.
- [X] T021 [P] [P3] [US4] Test missing `.wave/skills/` directory — `internal/skill/validate_test.go` — When store's source directory doesn't exist, all skill references should produce validation errors.
- [X] T022 [P] [P3] [US1] Test `skills: null` and `skills: []` YAML parsing — `internal/manifest/parser_test.go` — Verify both parse cleanly to nil/empty slice with no error at all three scopes.
- [X] T023 [P] [P3] [US5] Test same skill at all three scopes — `internal/skill/resolve_test.go` — Verify `ResolveSkills(["x"], ["x"], ["x"])` returns `["x"]` (one entry).
- [X] T024 [P] [P3] [US4] Test skill directory exists but has no SKILL.md — `internal/skill/validate_test.go` — DirectoryStore.Read returns error → `ValidateSkillRefs` should report it as invalid.

## Phase 9: Polish & Final Validation

- [X] T025 [P2] Run full test suite — Run `go test ./...` and `go test -race ./...` to verify no regressions across the entire codebase.
- [X] T026 [P3] Run linter — Run `golangci-lint run ./...` to verify no lint issues in new or modified code.

---

## Dependency Graph

```
T001 ──┐
T002 ──┤
T003 ──┼── T004 ──── T005
       │
       ├── T006 ──── T007
       │     │
       │     ├── T008 ── T009 ── T010
       │     │
       │     └── T011 ── T012
       │
       └── T013 ── T014 ── T015
                     │
T016 ────────────────┤
T017 ────────────────┤
T018 ────────────────┤
T019 ────────────────┤
T020 ────────────────┤
T021 ────────────────┤
T022 ────────────────┤
T023 ────────────────┤
T024 ────────────────┘
                     │
                     └── T025 ── T026
```

## Parallelization Opportunities

- **T001, T002, T003** can all run in parallel (independent struct changes in different packages)
- **T004 and T006** can run in parallel after Phase 1 completes (independent new files)
- **T005 and T007** can run in parallel (test files for their respective functions)
- **T008 and T011** can run in parallel (manifest vs pipeline validation, both depend on T006)
- **T010 and T012** can run in parallel (test files for their respective validation points)
- **T016, T017, T018, T019, T020, T021, T022, T023, T024** are all parallelizable (independent test cases)
- Total parallel opportunities: **9 tasks** can run concurrently in the widest parallelization band

## Summary

| Metric | Value |
|--------|-------|
| Total tasks | 26 |
| New files | 4 (`resolve.go`, `resolve_test.go`, `validate.go`, `validate_test.go`) |
| Modified files | 5 (`types.go` ×2, `parser.go`, `dag.go`, `executor.go`) |
| P1 tasks | 13 |
| P2 tasks | 7 |
| P3 tasks | 6 |
