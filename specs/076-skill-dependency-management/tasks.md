# Tasks

## Phase 1: Manifest Schema — Skills and Requires

- [ ] Task 1.1: Replace `SkillMount` with `SkillConfig` in `internal/manifest/types.go` — add `Install`, `Init`, `Check`, `CommandsGlob` fields; add `Skills map[string]SkillConfig` to `Manifest`; update YAML tag from `skill_mounts` to `skills`
- [ ] Task 1.2: Add `Requires` struct to `Pipeline` in `internal/pipeline/types.go` — fields: `Skills []string`, `Tools []string`
- [ ] Task 1.3: Add `Type` field to `WorkspaceConfig` in `internal/pipeline/types.go` — supports `""` (default/legacy) and `"worktree"`; add `Branch` field for worktree branch name
- [ ] Task 1.4: Update `validateSkillMountsWithFile()` → `validateSkillsWithFile()` in `internal/manifest/parser.go` — validate new SkillConfig fields
- [ ] Task 1.5: Add manifest parser tests for skills config in `internal/manifest/parser_test.go` [P]
- [ ] Task 1.6: Update pipeline YAML loader (`internal/pipeline/meta.go`) to parse `requires:` and `workspace.type`/`workspace.branch` [P]

## Phase 2: Preflight Validation

- [ ] Task 2.1: Create `internal/preflight/preflight.go` — `PreflightChecker` with `CheckTools(tools []string)` and `CheckSkills(skills []string, configs map[string]SkillConfig)` methods
- [ ] Task 2.2: Implement tool checking — use `exec.LookPath()` for each tool, return structured errors with actionable messages
- [ ] Task 2.3: Implement skill checking — run `check:` command, detect missing skills, attempt auto-install via `install:` + `init:` commands
- [ ] Task 2.4: Wire preflight into `DefaultPipelineExecutor.Execute()` in `internal/pipeline/executor.go` — call before workspace creation, emit preflight events
- [ ] Task 2.5: Write unit tests for `internal/preflight/preflight_test.go` — mock exec, test pass/fail/auto-install flows [P]
- [ ] Task 2.6: Write integration test for preflight in executor pipeline flow [P]

## Phase 3: Git Worktree Workspaces

- [ ] Task 3.1: Create `internal/worktree/worktree.go` — `WorktreeManager` with `Create(repoRoot, workspacePath, branch string)` and `Remove(workspacePath string)` methods
- [ ] Task 3.2: Implement worktree creation — `git worktree add <path> <branch>`, create branch if needed, handle errors
- [ ] Task 3.3: Implement worktree cleanup — `git worktree remove <path>`, handle force removal if dirty
- [ ] Task 3.4: Integrate worktree into `createStepWorkspace()` in `internal/pipeline/executor.go` — when `workspace.type == "worktree"`, use `WorktreeManager` instead of directory creation
- [ ] Task 3.5: Update `WorkspaceManager` interface in `internal/workspace/workspace.go` to support worktree type [P]
- [ ] Task 3.6: Write unit tests for `internal/worktree/worktree_test.go` — temp git repos, create/remove lifecycle [P]
- [ ] Task 3.7: Write race condition tests for concurrent worktree operations [P]

## Phase 4: Skill Provisioning

- [ ] Task 4.1: Create `internal/skill/skill.go` — `SkillProvisioner` that discovers and copies skill command files (`.claude/commands/*.md`) into a target workspace
- [ ] Task 4.2: Update `prepareWorkspace()` in `internal/adapter/claude.go` — accept skill commands source path, copy matching files to workspace `.claude/commands/`
- [ ] Task 4.3: Pass skill provisioning config through `AdapterRunConfig` — add `SkillCommandsDir` field
- [ ] Task 4.4: Wire skill provisioning in `runStepExecution()` in `internal/pipeline/executor.go` — resolve skill commands dir from manifest, pass to adapter config
- [ ] Task 4.5: Write unit tests for `internal/skill/skill_test.go` — test file discovery and copy [P]
- [ ] Task 4.6: Write integration test verifying skill commands appear in workspace `.claude/commands/` [P]

## Phase 5: Prompt Simplification and Cleanup

- [ ] Task 5.1: Add `exec.type: slash_command` support in pipeline types and executor — resolve slash command to prompt
- [ ] Task 5.2: Update sample pipeline YAML (`speckit-flow.yaml`) to use worktree workspaces and `requires:` block
- [ ] Task 5.3: Update `gh-issue-impl.yaml` pipeline to use worktree workspaces
- [ ] Task 5.4: Simplify pipeline prompts in `.wave/prompts/` — remove `REPO_ROOT`, `cd`, worktree boilerplate

## Phase 6: Testing and Validation

- [ ] Task 6.1: Run full test suite `go test -race ./...` and fix any failures
- [ ] Task 6.2: Verify existing pipelines without `requires:` still work (backward compatibility)
- [ ] Task 6.3: Update documentation in `docs/` for new manifest schema fields
