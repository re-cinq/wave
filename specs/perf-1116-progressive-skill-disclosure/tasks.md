# Tasks

## Phase 1: Immediate Mitigation — gh-cli Split
- [X] Task 1.1: Extract reference material from `skills/gh-cli/SKILL.md` into `skills/gh-cli/references/full-reference.md`
- [X] Task 1.2: Rewrite `skills/gh-cli/SKILL.md` to contain only core patterns (~100 lines: pr create, issue create, common flags, auth basics)
- [X] Task 1.3: Validate gh-cli skill still parses correctly (`ParseMetadata` + `Parse` + `discoverResources`)

## Phase 2: Store & Provisioning Infrastructure
- [X] Task 2.1: Add `ReadMetadata(name string) (Skill, error)` method to `Store` interface in `internal/skill/store.go`
  - file_changes: [{path: "internal/skill/store.go", action: "modify"}]
- [X] Task 2.2: Implement `ReadMetadata` on `DirectoryStore` — calls `ParseMetadata` instead of `Parse`, still discovers resources
  - file_changes: [{path: "internal/skill/store.go", action: "modify"}]
- [X] Task 2.3: Add `ProvisionLevel` type (`Level1Metadata`, `Level2Instructions`) to `internal/skill/provision.go` [P]
  - file_changes: [{path: "internal/skill/provision.go", action: "modify"}]
- [X] Task 2.4: Add `ProvisionFromStoreWithLevel()` function that accepts `ProvisionLevel` parameter [P]
  - When Level 1: call `store.ReadMetadata()`, write stub SKILL.md (frontmatter + "Use Skill tool for full content")
  - When Level 2: current `ProvisionFromStore()` behavior (full body)
  - file_changes: [{path: "internal/skill/provision.go", action: "modify"}]
- [X] Task 2.5: Update `ProvisionFromStore()` to delegate to `ProvisionFromStoreWithLevel()` with Level 2 default
  - file_changes: [{path: "internal/skill/provision.go", action: "modify"}]

## Phase 3: Adapter & Executor Integration
- [X] Task 3.1: Add `Level int` field to `SkillRef` in `internal/adapter/adapter.go`
  - file_changes: [{path: "internal/adapter/adapter.go", action: "modify"}]
- [X] Task 3.2: Update `buildSkillSection()` in `internal/adapter/claude.go` to render Level 1 skills differently (indicate content available on-demand) [P]
  - file_changes: [{path: "internal/adapter/claude.go", action: "modify"}]
- [X] Task 3.3: Update executor skill provisioning in `internal/pipeline/executor.go` to pass `SkillRef.Level` from provisioning result [P]
  - file_changes: [{path: "internal/pipeline/executor.go", action: "modify"}]

## Phase 4: Testing
- [X] Task 4.1: Add `TestReadMetadata` to `internal/skill/store_test.go` — verify metadata-only read [P]
- [X] Task 4.2: Add `TestProvisionFromStoreWithLevel_Level1` to `internal/skill/provision_test.go` — verify stub written [P]
- [X] Task 4.3: Add `TestProvisionFromStoreWithLevel_Level2` — verify full body written (regression) [P]
- [X] Task 4.4: Update `TestBuildSkillSection` in `internal/adapter/claude_test.go` to cover Level 1 rendering [P]
- [X] Task 4.5: Extend `TestSkillLifecycle_FileAdapter` in `internal/skill/integration_test.go` to cover Level 1 path

## Phase 5: CLI & Validation
- [X] Task 5.1: Add body-size warning to `wave skills verify` — warn when SKILL.md body exceeds 500 lines
  - file_changes: [{path: "cmd/wave/commands/skills.go", action: "modify"}]
- [X] Task 5.2: Add body-size warning to `wave skills publish` — same threshold
  - file_changes: [{path: "cmd/wave/commands/skills.go", action: "modify"}]
- [X] Task 5.3: Run `go test ./internal/skill/... ./internal/adapter/...` — verify all tests pass
- [X] Task 5.4: Verify gh-cli split works end-to-end with a representative pipeline
