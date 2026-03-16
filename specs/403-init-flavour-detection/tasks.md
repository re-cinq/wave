# Tasks

## Phase 1: Manifest Type Updates
- [X] Task 1.1: Add `Flavour` and `FormatCommand` fields to `Project` struct in `internal/manifest/types.go`
- [X] Task 1.2: Update `ProjectVars()` to include `project.flavour` and `project.format_command`

## Phase 2: Flavour Detection System
- [X] Task 2.1: Create `internal/onboarding/flavour.go` with `FlavourInfo` struct and `DetectFlavour()` function implementing the full 25+ language detection matrix [P]
- [X] Task 2.2: Create `internal/onboarding/flavour_test.go` with table-driven tests for each language/flavour [P]
- [X] Task 2.3: Create `internal/onboarding/metadata.go` with `ExtractProjectMetadata()` to read project name/description from go.mod, Cargo.toml, package.json, etc. [P]
- [X] Task 2.4: Create `internal/onboarding/metadata_test.go` with tests for metadata extraction [P]

## Phase 3: Cold-Start Git Init
- [X] Task 3.1: Add `ensureGitRepo()` function to `cmd/wave/commands/init.go` that runs `git init` when `.git` is absent
- [X] Task 3.2: Add `createInitialCommit()` function to create initial commit after writing wave files (targeted `git add wave.yaml .wave/`)
- [X] Task 3.3: Integrate `ensureGitRepo()` at the start of `runInit()` and `runWizardInit()`, and `createInitialCommit()` at the end

## Phase 4: Wire Flavour into Init & Onboarding
- [X] Task 4.1: Replace `detectProject()` in `init.go` with call to `onboarding.DetectFlavour()` and convert `FlavourInfo` to manifest map
- [X] Task 4.2: Replace `detectProjectType()` in `steps.go` with call to `DetectFlavour()` and add `format_command` to `TestConfigStep` result
- [X] Task 4.3: Add `Flavour` and `FormatCommand` fields to `WizardResult` in `onboarding.go` and pass them through step results into `buildManifest()`
- [X] Task 4.4: Delete unused `detectProject()`, `detectNodeProject()` from `init.go` and `detectProjectType()` from `steps.go`

## Phase 5: Smart Init — Forge Filtering & Metadata
- [X] Task 5.1: Use `forge.DetectFromGitRemotes()` in init to filter personas via naming convention (forge-prefixed personas only for detected forge) [P]
- [X] Task 5.2: Use `ExtractProjectMetadata()` to populate manifest `metadata.name` and `metadata.description` instead of hardcoded defaults [P]
- [X] Task 5.3: Add first-run suggestion logic: empty project suggests `ops-bootstrap`, existing code suggests `audit-dx` [P]

## Phase 6: Testing & Validation
- [X] Task 6.1: Write integration test for cold-start scenario (init in empty dir creates git repo with commit)
- [X] Task 6.2: Verify `go test ./...` passes
- [X] Task 6.3: Verify `golangci-lint run ./...` passes
- [X] Task 6.4: Verify flavour and format_command fields appear correctly in generated `wave.yaml`
