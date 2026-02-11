# Tasks: Release-Gated Pipeline Embedding

**Branch**: `029-release-gated-embedding`
**Spec**: `specs/029-release-gated-embedding/spec.md`
**Plan**: `specs/029-release-gated-embedding/plan.md`

---

## Phase 1: Setup

- [X] T001 [P1] Verify branch and dependencies
  - Confirm `029-release-gated-embedding` branch is checked out
  - Run `go build ./...` to verify baseline compiles
  - Run `go test ./...` to verify all existing tests pass
  - **File**: (project root)

---

## Phase 2: Foundational — Pipeline Metadata Extension (US3)

These tasks are blocking prerequisites for all other phases.

- [X] T002 [P1] [US3] Add `Release` and `Disabled` fields to `PipelineMetadata` struct
  - Add `Release bool \`yaml:"release,omitempty"\`` to `PipelineMetadata`
  - Add `Disabled bool \`yaml:"disabled,omitempty"\`` to `PipelineMetadata`
  - Go zero-value `false` provides correct default behavior
  - **File**: `internal/pipeline/types.go:18-21`

- [X] T003 [P1] [US3] Add unit tests for `PipelineMetadata` YAML parsing
  - Test `release: true` parses to `Release == true`
  - Test `release: false` parses to `Release == false`
  - Test absent `release` field defaults to `false`
  - Test `disabled: true` is independent of `release: true`
  - Test invalid value `release: "yes"` produces YAML unmarshal error
  - Table-driven test format
  - **File**: `internal/pipeline/types_test.go` (new file)

---

## Phase 3: Defaults API — Release Pipeline Queries (US5)

- [X] T004 [P1] [US5] Implement `GetReleasePipelines()` in defaults package
  - Add `GetReleasePipelines() (map[string]string, error)` function
  - Call `GetPipelines()`, unmarshal each YAML into `pipeline.Pipeline`
  - Filter to entries where `Metadata.Release == true`
  - Return filtered `map[string]string` (filename → YAML content)
  - If unmarshal fails for a pipeline, exclude it (log warning, not hard error)
  - Return empty map (not nil) when no pipelines have `release: true`
  - **File**: `internal/defaults/embed.go`

- [X] T005 [P] [P1] [US5] Implement `ReleasePipelineNames()` in defaults package
  - Add `ReleasePipelineNames() []string` function
  - Calls `GetReleasePipelines()` and extracts keys
  - Returns strict subset of `PipelineNames()`
  - **File**: `internal/defaults/embed.go`

- [X] T006 [P1] [US5] Add unit tests for release filtering in defaults package
  - Test `GetReleasePipelines()` returns strict subset of `GetPipelines()`
  - Test `ReleasePipelineNames()` returns strict subset of `PipelineNames()`
  - Test that only pipelines with `metadata.release: true` appear in result
  - Test that pipelines without `release` field are excluded
  - Test that `release: true` + `disabled: true` pipelines are included (FR-009)
  - Verify current known release pipelines: `doc-loop.yaml`, `github-issue-enhancer.yaml`, `issue-research.yaml`
  - **File**: `internal/defaults/embed_test.go`

---

## Phase 4: Init Filtering — Release-Only Pipeline Distribution (US1)

- [X] T007 [P1] [US1] Add `All` field to `InitOptions` and register `--all` flag
  - Add `All bool` to `InitOptions` struct
  - Register `--all` flag: `cmd.Flags().BoolVar(&opts.All, "all", false, "Include all pipelines regardless of release status")`
  - Update command long description to mention `--all` flag (SC-007)
  - **File**: `cmd/wave/commands/init.go:16-49`

- [X] T008 [P1] [US1] Modify `runInit()` to use release filtering
  - When `opts.All` is false: call `defaults.GetReleasePipelines()` instead of `defaults.GetPipelines()`
  - When `opts.All` is true: call `defaults.GetPipelines()` (existing behavior)
  - Pass filtered pipelines map to `createExamplePipelines()` (refactor to accept parameter)
  - When filtered pipelines is empty, emit warning on stderr (FR-011)
  - **File**: `cmd/wave/commands/init.go:51-138`

- [X] T009 [P1] [US1] Refactor `createExamplePipelines()` to accept a pipelines map parameter
  - Change signature from `createExamplePipelines() error` to `createExamplePipelines(pipelines map[string]string) error`
  - Remove internal `defaults.GetPipelines()` call — use the passed map
  - Apply same refactoring to `createExamplePipelinesIfMissing()`
  - **File**: `cmd/wave/commands/init.go:563-597`

---

## Phase 5: Transitive Dependency Exclusion (US2)

- [X] T010 [P1] [US2] Implement `filterTransitiveDeps()` function
  - Create `filterTransitiveDeps(pipelines, allContracts, allPrompts map[string]string) (contracts, prompts map[string]string, err error)`
  - Parse each pipeline YAML into `pipeline.Pipeline`
  - Walk steps to extract `Handover.Contract.SchemaPath` values
  - Walk steps to extract `Exec.SourcePath` values
  - Normalize contract refs: strip `.wave/contracts/` prefix to match embedded keys
  - Normalize prompt refs: strip `.wave/prompts/` prefix to match embedded keys
  - Ignore empty `SchemaPath` and `SourcePath` values
  - Ignore inline `Source` blocks (no file dependency)
  - Filter `allContracts` and `allPrompts` to only keys in reference sets
  - Emit warning for referenced but missing schema files (not error)
  - **File**: `cmd/wave/commands/init.go` (new function)

- [X] T011 [P1] [US2] Integrate transitive filtering into `runInit()`
  - When `opts.All` is false: call `filterTransitiveDeps()` with release pipelines
  - Pass filtered contracts to `createExampleContracts()` (refactor to accept parameter)
  - Pass filtered prompts to `createExamplePrompts()` (refactor to accept parameter)
  - Personas always use `defaults.GetPersonas()` unfiltered (FR-005)
  - When `opts.All` is true: use `defaults.GetContracts()` and `defaults.GetPrompts()` directly
  - **File**: `cmd/wave/commands/init.go:51-138`

- [X] T012 [P] [P1] [US2] Refactor `createExampleContracts()` and `createExamplePrompts()` to accept map parameters
  - Change `createExampleContracts()` to accept `contracts map[string]string`
  - Change `createExampleContractsIfMissing()` to accept `contracts map[string]string`
  - Change `createExamplePrompts()` to accept `prompts map[string]string`
  - Change `createExamplePromptsIfMissing()` to accept `prompts map[string]string`
  - Remove internal `defaults.Get*()` calls — use passed maps
  - **File**: `cmd/wave/commands/init.go:599-675`

---

## Phase 6: Merge Mode Filtering (US1 + US2)

- [X] T013 [P1] [US1] [US2] Apply release filtering to `runMerge()`
  - When `opts.All` is false: use `defaults.GetReleasePipelines()` and `filterTransitiveDeps()`
  - When `opts.All` is true: use `defaults.GetPipelines()` with all contracts/prompts
  - Pass filtered maps to `*IfMissing()` functions
  - Existing files are never deleted (preserve non-release files from prior `--all`)
  - `--all` and `--merge` compose naturally
  - **File**: `cmd/wave/commands/init.go:140-205`

---

## Phase 7: Display Updates (US1)

- [X] T014 [P1] [US1] Update `printInitSuccess()` to show filtered counts
  - Change signature to accept extracted asset maps/counts instead of querying `defaults.Get*()`
  - Display count of actually extracted pipelines, contracts, prompts
  - Display only extracted pipeline names (not all embedded)
  - CLR-005: user sees accurate summary of what was initialized
  - **File**: `cmd/wave/commands/init.go:274-314`

---

## Phase 8: Include All Pipelines for Contributors (US4)

- [X] T015 [P] [P2] [US4] Add tests for `--all` flag behavior
  - Test `wave init --all` extracts all pipelines regardless of release status
  - Test `wave init --all` extracts all contracts and prompts
  - Test `wave init` (without `--all`) extracts only release pipelines
  - Test `wave init --all --merge` adds all missing pipelines
  - Compare counts: `--all` count equals `defaults.GetPipelines()` count
  - **File**: `cmd/wave/commands/init_test.go`

---

## Phase 9: Integration Tests — Release Filtering (US1 + US2)

- [X] T016 [P1] [US1] Add test: `wave init` writes only release pipelines
  - Run `wave init` in temp dir
  - Verify `.wave/pipelines/` contains only files corresponding to `release: true` pipelines
  - Verify non-release pipelines are absent
  - Verify pipeline count matches `defaults.GetReleasePipelines()` count
  - **File**: `cmd/wave/commands/init_test.go`

- [X] T017 [P] [P1] [US2] Add test: transitive contract exclusion
  - Run `wave init` in temp dir
  - Verify contracts referenced only by non-release pipelines are absent from `.wave/contracts/`
  - Verify contracts shared by release and non-release pipelines are present
  - **File**: `cmd/wave/commands/init_test.go`

- [X] T018 [P] [P1] [US2] Add test: transitive prompt exclusion
  - Run `wave init` in temp dir
  - Verify prompts referenced only by non-release pipelines are absent from `.wave/prompts/`
  - Verify prompts shared by release pipelines are present
  - **File**: `cmd/wave/commands/init_test.go`

- [X] T019 [P] [P1] [US2] Add test: personas are never transitively excluded
  - Run `wave init` in temp dir (without `--all`)
  - Verify `.wave/personas/` contains all personas from `defaults.GetPersonas()`
  - **File**: `cmd/wave/commands/init_test.go`

- [X] T020 [P] [P1] [US1] Add test: `release: true` + `disabled: true` pipeline is included
  - Verify that a pipeline with both flags is included in `wave init` output (SC-008)
  - **File**: `cmd/wave/commands/init_test.go`

---

## Phase 10: Edge Cases and Polish

- [X] T021 [P3] Add test: zero release pipelines produces warning
  - Mock or configure scenario where no pipelines have `release: true`
  - Verify `wave init` succeeds (no error)
  - Verify warning message is emitted on stderr
  - Verify `.wave/pipelines/` directory exists but is empty
  - **File**: `cmd/wave/commands/init_test.go`

- [X] T022 [P] [P3] Add test: missing referenced contract emits warning
  - Configure a pipeline that references a non-existent schema file
  - Verify `wave init` succeeds with a warning (not an error)
  - **File**: `cmd/wave/commands/init_test.go`

- [X] T023 [P3] Update existing init tests for release filtering compatibility
  - `TestInitCreatesPipelineFiles` must expect only release pipelines (or use `--all`)
  - `TestInitCreatesContractFiles` must expect only release-referenced contracts (or use `--all`)
  - Ensure `TestInitIdempotence` still passes with filtering
  - Ensure `TestInitOutputValidatesWithWaveValidate` still passes
  - **File**: `cmd/wave/commands/init_test.go`

---

## Phase 11: Final Validation

- [X] T024 [P1] Run full test suite and fix any failures
  - Run `go test ./...` from project root
  - Run `go test -race ./...` for race condition detection
  - Fix any test failures introduced by the changes
  - **File**: (project root)

- [X] T025 [P1] Run `go vet ./...` and fix any issues
  - Verify no static analysis warnings
  - **File**: (project root)
