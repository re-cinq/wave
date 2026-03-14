# Tasks

## Phase 1: Extract Provisioning Logic

- [X] Task 1.1: Create `internal/skill/provision.go` with `SkillInfo` type and `ProvisionFromStore(store Store, workspacePath string, skillNames []string) ([]SkillInfo, error)` function
  - Move the DirectoryStore provisioning logic from executor.go lines 1244-1337 into this function
  - `SkillInfo` holds `Name`, `Description`, `SourcePath` for metadata passing
  - Return an error (not warning) when a resolved skill is not found in the store
  - Preserve existing path containment checks for resource files
  - Handle deduplication: skip skills already in `requires.skills` (caller responsibility)

- [X] Task 1.2: Create `internal/skill/provision_test.go` with tests for `ProvisionFromStore` [P]
  - Test: successfully provisions SKILL.md and resources into workspace
  - Test: returns error for missing skill
  - Test: handles empty skill list (no-op)
  - Test: path traversal in resource files is blocked
  - Test: returns correct SkillInfo metadata for each provisioned skill

## Phase 2: CLAUDE.md Skill Section

- [X] Task 2.1: Add `ResolvedSkills` field to `AdapterRunConfig` in `internal/adapter/adapter.go`
  - Define `SkillRef` struct with `Name`, `Description` fields in adapter package (or reuse from skill package)
  - Add `ResolvedSkills []SkillRef` to `AdapterRunConfig`

- [X] Task 2.2: Add skill section builder in `internal/adapter/claude.go` [P]
  - Add `buildSkillSection(skills []SkillRef) string` function
  - Section format: `## Available Skills\n\nThe following skills are available...\n- **name**: description (see .wave/skills/name/SKILL.md)\n`
  - Insert after persona system prompt (section 1) and before contract compliance (section 2) in CLAUDE.md assembly

- [X] Task 2.3: Add tests for skill section in `internal/adapter/claude_test.go` [P]
  - Test: CLAUDE.md contains skill section when ResolvedSkills is non-empty
  - Test: CLAUDE.md has no skill section when ResolvedSkills is empty
  - Test: skill section contains correct names and descriptions

## Phase 3: Executor Integration

- [X] Task 3.1: Refactor executor.go to use `ProvisionFromStore` and pass skill metadata to adapter
  - Replace inline DirectoryStore provisioning (lines 1244-1337) with call to `skill.ProvisionFromStore`
  - Convert returned `[]SkillInfo` to `[]adapter.SkillRef` for adapter config
  - Change warning events to a hard error return when `ProvisionFromStore` fails
  - Set `cfg.ResolvedSkills` in the `AdapterRunConfig` construction

- [X] Task 3.2: Add executor tests for skill provisioning integration
  - Test: executor passes resolved skills to adapter config
  - Test: executor returns error when store read fails for a resolved skill
  - Test: `requires.skills` provisioning still works (regression)

## Phase 4: Validation and Polish

- [X] Task 4.1: Run full test suite and fix any regressions
  - `go test ./...`
  - `go test -race ./...`
  - `golangci-lint run ./...`

- [X] Task 4.2: Verify existing `commands_glob` behavior is preserved
  - Confirm `Provisioner.Provision` (skill.go) is untouched
  - Confirm `SkillCommandsDir` flow in adapter is unchanged
