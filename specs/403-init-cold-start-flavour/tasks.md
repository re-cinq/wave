# Tasks: Init Cold-Start Fix, Flavour Auto-Detection

**Branch**: `403-init-cold-start-flavour` | **Date**: 2026-03-16
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

---

## Phase 1: Setup

- [X] T001 [P1] Setup — Create `internal/flavour/` package directory and `detect.go` with package declaration, `DetectionResult` struct, `DetectionRule` struct, and the exported `Detect(dir string) DetectionResult` function signature (empty body returning zero value) — `internal/flavour/detect.go`
- [X] T002 [P1] Setup — Create `internal/flavour/metadata.go` with `MetadataResult` struct and exported `DetectMetadata(dir string) MetadataResult` function signature (empty body returning zero value) — `internal/flavour/metadata.go`

---

## Phase 2: Foundation — Flavour Detection Engine (Story 2, FR-004/005/013)

- [X] T003 [P1] [Story2] Implement priority-ordered detection rules slice with all 25+ language flavours as specified in data-model.md: go, rust, deno, bun, pnpm, yarn, node, python (modern/legacy), dotnet, maven, gradle, kotlin, elixir, dart, cmake, make, php, ruby, swift, zig, scala, cabal, stack, typescript-standalone. Each rule maps marker files to a `DetectionResult` with flavour, language, test/lint/build/format commands, and source glob — `internal/flavour/detect.go`
- [X] T004 [P1] [Story2] Implement `Detect(dir string) DetectionResult` function body: iterate rules in priority order, check each marker (exact filename via `os.Stat`, glob patterns via `filepath.Glob`), return first match. Return zero value if no match — `internal/flavour/detect.go`
- [X] T005 [P1] [Story2] Implement Node refinement logic in `Detect`: after matching any Node-based flavour (node, bun, pnpm, yarn), check for `tsconfig.json` to override Language to `"typescript"` and SourceGlob to `"*.{ts,tsx}"`. Read `package.json` scripts to derive actual test/lint/build commands (port logic from existing `detectNodeProject()` in init.go:902) — `internal/flavour/detect.go`
- [X] T006 [P1] [Story2] [P] Write table-driven tests for `Detect()`: one test case per flavour (25+), specificity ordering tests (bun.lock before package.json, deno.json before package.json), no-match test, Node+tsconfig refinement test, glob marker tests (*.csproj, *.cabal) — `internal/flavour/detect_test.go`

---

## Phase 3: Manifest Extension (Story 3, FR-006/007/011)

- [X] T007 [P1] [Story3] Add `Flavour string` and `FormatCommand string` fields (both `yaml:"...,omitempty"`) to the `Project` struct in `internal/manifest/types.go` — `internal/manifest/types.go:8-14`
- [X] T008 [P1] [Story3] Extend `ProjectVars()` to include `project.flavour` and `project.format_command` entries when non-empty — `internal/manifest/types.go:171-192`
- [X] T009 [P1] [Story3] [P] Add `Flavour` and `FormatCommand` fields to `WizardResult` struct in `internal/onboarding/onboarding.go` — `internal/onboarding/onboarding.go:25-36`
- [X] T010 [P1] [Story3] Update `buildManifest()` to propagate `Flavour` and `FormatCommand` from `WizardResult` into the manifest project section — `internal/onboarding/onboarding.go:175`
- [X] T011 [P1] [Story3] [P] Write tests for `ProjectVars()` with new fields: verify `project.flavour` and `project.format_command` are present when set, absent when empty — `internal/manifest/types_test.go`

---

## Phase 4: Cold-Start Git Bootstrap (Story 1, FR-001/002/003)

- [X] T012 [P1] [Story1] Add `ensureGitRepo()` helper function in `cmd/wave/commands/init.go` that: (1) checks for `.git` existence and runs `git init` if absent, (2) checks for commits via `git rev-parse --verify HEAD` and returns a boolean `needsInitialCommit` flag — `cmd/wave/commands/init.go`
- [X] T013 [P1] [Story1] Wire `ensureGitRepo()` into `runInit()` (line 267): call before any other logic. If `needsInitialCommit` is true, defer creating an initial commit after Wave files are written (stage only `wave.yaml` and `.wave/`, commit message `chore: initialize wave project`) — `cmd/wave/commands/init.go:267`
- [X] T014 [P1] [Story1] Wire `ensureGitRepo()` into `runWizardInit()` (line 1165): same logic as T013 — `cmd/wave/commands/init.go:1165`

---

## Phase 5: Integration — Flavour into Init (Stories 2, 5, FR-009)

- [X] T015 [P1] [Story2] Replace `detectProject()` function body (init.go:850) with a call to `flavour.Detect(".")` and convert `DetectionResult` to `map[string]interface{}` including the new `flavour` and `format_command` keys — `cmd/wave/commands/init.go:850`
- [X] T016 [P1] [Story2] Replace `detectProjectType()` function body (onboarding/steps.go:165) with a call to `flavour.Detect(".")` and convert `DetectionResult` to `map[string]string`. Add `format_command` to the TestConfigStep result — `internal/onboarding/steps.go:165`
- [X] T017 [P1] [Story2] Update `createDefaultManifest()` (init.go:994) to include `flavour` and `format_command` keys in the project section when present — `cmd/wave/commands/init.go:994`

---

## Phase 6: Metadata Extraction (Story 5, FR-009)

- [X] T018 [P2] [Story5] Implement `DetectMetadata(dir string) MetadataResult` function body: parse `go.mod` (extract repo name from module path), `package.json` (JSON unmarshal name/description), `Cargo.toml` (line scan `name =` under `[package]`), `pyproject.toml` (line scan under `[project]`). Fallback: use directory name as project name — `internal/flavour/metadata.go`
- [X] T019 [P2] [Story5] [P] Write table-driven tests for `DetectMetadata()`: go.mod, package.json, Cargo.toml, pyproject.toml, no-manifest-fallback-to-dirname — `internal/flavour/detect_test.go`
- [X] T020 [P2] [Story5] Wire `flavour.DetectMetadata(".")` into `runInit()` and `runWizardInit()` to populate `metadata.name` and `metadata.description` in the manifest when not already set — `cmd/wave/commands/init.go`

---

## Phase 7: Forge Filtering & Required Pipelines (Stories 4, 7, FR-008/010)

- [X] T021 [P2] [Story4] Add forge detection call (`forge.DetectFromGitRemotes()`) in `getFilteredAssets()` (init.go:116) and apply `forge.FilterPipelinesByForge()` to the pipeline name list after release filtering — `cmd/wave/commands/init.go:116`
- [X] T022 [P2] [Story7] [P] Add `requiredPipelines` safeguard in `getFilteredAssets()`: after all filtering, ensure `impl-issue` is always present in the pipeline set — `cmd/wave/commands/init.go:116`

---

## Phase 8: First-Run Suggestion (Story 6, FR-012)

- [X] T023 [P3] [Story6] Add `suggestFirstPipeline(dir string) string` helper that checks for non-Wave source files: returns `"audit-dx"` if source files exist, `"ops-bootstrap"` if empty — `cmd/wave/commands/init.go`
- [X] T024 [P3] [Story6] Update `printInitSuccess()` (init.go:790) and `printWizardSuccess()` (init.go:1332) to call `suggestFirstPipeline()` and include the suggestion in the success message — `cmd/wave/commands/init.go:790`

---

## Phase 9: Integration Tests & Polish

- [X] T025 [P1] [P] Write integration tests for cold-start scenarios: (1) empty dir no .git, (2) .git but no commits, (3) .git + commits + no remote, (4) .git + commits + remote — `cmd/wave/commands/init_test.go`
- [X] T026 [P2] [P] Write integration tests for flavour detection in init: run `detectProject()` with go.mod present → verify flavour=go in result — `cmd/wave/commands/init_test.go`
- [X] T027 [P2] [P] Write integration tests for forge-filtered pipeline selection and requiredPipelines safeguard — `cmd/wave/commands/init_test.go`
- [X] T028 [P1] Run `go test ./...` and `go vet ./...` to verify all tests pass and no regressions — full codebase
- [X] T029 [P1] Run `golangci-lint run ./...` and fix any lint issues — full codebase

---

## Dependency Graph

```
T001, T002 (setup, parallel)
  ├── T003 → T004 → T005 (detection engine, sequential)
  │   └── T006 (tests, after T005)
  ├── T007 → T008 (manifest types, sequential)
  │   ├── T009 (wizard result, parallel with T008)
  │   ├── T010 (buildManifest, after T008+T009)
  │   └── T011 (type tests, parallel with T010)
  └── T012 → T013 → T014 (cold-start, sequential)
T005 + T008 → T015 → T016 → T017 (integration, needs both flavour + manifest)
T005 → T018 → T019, T020 (metadata, after detection engine)
T015 → T021 → T022 (forge filtering, after integration)
T015 → T023 → T024 (suggestion, after integration)
T013 + T015 → T025, T026, T027 (integration tests, after cold-start + flavour wired)
T025..T027 → T028 → T029 (final validation, sequential)
```

## Task Summary

| Phase | Tasks | Parallelizable |
|-------|-------|----------------|
| 1: Setup | T001-T002 | 2 |
| 2: Flavour Engine | T003-T006 | 1 (T006) |
| 3: Manifest Extension | T007-T011 | 2 (T009, T011) |
| 4: Cold-Start | T012-T014 | 0 |
| 5: Flavour Integration | T015-T017 | 0 |
| 6: Metadata | T018-T020 | 1 (T019) |
| 7: Forge Filtering | T021-T022 | 1 (T022) |
| 8: First-Run Suggestion | T023-T024 | 0 |
| 9: Polish | T025-T029 | 3 (T025-T027) |
| **Total** | **29** | **10** |
