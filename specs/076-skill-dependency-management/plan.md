# Implementation Plan: Skill Dependency Management, Git Worktree Workspaces, and Preflight Validation

## Objective

Add declarative skill dependency management, preflight validation with auto-install, and git worktree-based workspaces to Wave. This eliminates manual skill provisioning, prevents dependency failures mid-execution, and provides true git isolation for concurrent pipeline runs.

## Approach

Follow the 5-phase approach outlined in the issue, implementing each phase as a cohesive unit with its own tests. Each phase builds on the previous, but phases are independently testable.

The implementation introduces three new packages (`internal/skill`, `internal/preflight`, `internal/worktree`) and modifies four existing packages (`internal/manifest`, `internal/workspace`, `internal/pipeline`, `internal/adapter`).

## File Mapping

### New Files (Create)

| File | Purpose |
|------|---------|
| `internal/skill/skill.go` | Skill type definitions, check/install/init logic |
| `internal/skill/skill_test.go` | Unit tests for skill operations |
| `internal/preflight/preflight.go` | Preflight validation engine (tools + skills) |
| `internal/preflight/preflight_test.go` | Unit tests for preflight checks |
| `internal/worktree/worktree.go` | Git worktree create/remove lifecycle |
| `internal/worktree/worktree_test.go` | Unit tests for worktree management |

### Modified Files

| File | Changes |
|------|---------|
| `internal/manifest/types.go` | Add `Skills` map to `Manifest`, expand `SkillMount` → `SkillConfig` |
| `internal/manifest/parser.go` | Add `validateSkills()`, update `ValidateWithFile()` |
| `internal/manifest/parser_test.go` | Tests for skills parsing and validation |
| `internal/pipeline/types.go` | Add `Requires` struct to `Pipeline`, add `WorkspaceType` field |
| `internal/pipeline/executor.go` | Call preflight before execution, handle worktree workspace creation |
| `internal/pipeline/executor_test.go` | Tests for preflight integration and worktree workspace flow |
| `internal/workspace/workspace.go` | Add worktree workspace creation method, skill provisioning |
| `internal/workspace/workspace_test.go` | Tests for worktree and skill provisioning |
| `internal/adapter/claude.go` | Update `prepareWorkspace()` to copy skill commands into `.claude/commands/` |
| `internal/adapter/claude_test.go` | Tests for skill command provisioning |
| `internal/pipeline/meta.go` | Update pipeline loader to parse `requires:` |

## Architecture Decisions

### AD-1: New packages vs inline code
**Decision**: Create `internal/skill`, `internal/preflight`, and `internal/worktree` as separate packages.
**Rationale**: Each concern is independently testable and follows the single-responsibility principle already present in the codebase (e.g., `internal/security`, `internal/contract`).

### AD-2: Skill check via shell commands
**Decision**: Skills declare their own `check:` command (e.g., `specify --version` or `ls .claude/commands/bmad.*.md`).
**Rationale**: Each skill has different verification mechanisms. A generic "does binary exist" check is insufficient — BMAD installs files, not a binary.

### AD-3: Worktree as workspace type, not replacement
**Decision**: Add `workspace.type: worktree` as a new option, keeping existing `mount`-based workspaces working.
**Rationale**: Backward compatibility — existing pipelines should not break. The `worktree` type is opt-in per step.

### AD-4: Preflight runs once per pipeline, not per step
**Decision**: Preflight validation runs before the first step, validating all declared `requires:` at once.
**Rationale**: Installing skills mid-pipeline would be confusing. Fail-fast at the start gives the user a clear signal.

### AD-5: Skill provisioning during adapter workspace prep
**Decision**: Skill commands (`.claude/commands/*.md`) are copied into the workspace during `prepareWorkspace()` in the adapter, not during workspace directory creation.
**Rationale**: The adapter is responsible for workspace file layout (settings.json, CLAUDE.md). Skill commands are part of that layout.

### AD-6: SkillMount becomes SkillConfig
**Decision**: Rename and expand the existing `SkillMount` type to `SkillConfig` with `install`, `init`, `check`, and `commands_glob` fields. Retain backward compatibility by still accepting the old `skill_mounts` YAML key (as an alias or deprecation path).
**Rationale**: The current `SkillMount` is dead code with only a `path` field. We need the richer structure, and this is prototype phase where backward compatibility is not a constraint.

## Risks

| Risk | Mitigation |
|------|------------|
| `git worktree add` fails if branch doesn't exist | Create branch if needed; handle error with actionable message |
| Skill install commands are interactive | Document requirement for non-interactive flags; timeout after configurable duration |
| Concurrent worktree creation on same branch | Use unique branch names with run-ID suffix or reject collisions |
| `skipDirs` blocks `.claude/commands/` in mount-based workspaces | Modify `skipDirs` to allow `.claude/commands/` subdirectory or use worktree type which doesn't copy |
| Stale skill commands after upgrade | `check:` runs on every preflight; if check fails, re-install |

## Testing Strategy

### Unit Tests
- `internal/skill`: Test check/install/init with mock commands (using `exec.Command` with test helper binaries)
- `internal/preflight`: Test tool path lookup, skill check, auto-install flow, error messages
- `internal/worktree`: Test create/remove lifecycle with temp git repos
- `internal/manifest`: Test skills parsing, validation errors for malformed configs
- `internal/pipeline`: Test requires parsing, preflight integration with executor

### Integration Tests
- End-to-end preflight + workspace creation flow with mock adapter
- Worktree lifecycle across pipeline execution
- Skill provisioning verification (commands present in workspace)

### Race Condition Tests
- Concurrent worktree creation on different branches
- Concurrent preflight checks (shared skill state)

### All tests run with `-race` flag
