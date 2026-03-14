# Implementation Plan: Skill Worktree Provisioning

## Objective

Extend the existing partial DirectoryStore skill provisioning in `executor.go` to:
1. Copy resolved SKILL.md files + resource files into worktree workspaces at `.wave/skills/<name>/`
2. Inject skill content/references into the runtime CLAUDE.md assembly
3. Handle missing skills as hard errors (not warnings) before step execution begins
4. Ensure deduplication and scope precedence are enforced

## Current State Analysis

The codebase already has **partial skill provisioning** in `executor.go` (lines 1244-1337):
- DirectoryStore skills are read, SKILL.md body is written to `.wave/skills/<name>/SKILL.md` in the workspace
- Resource files (scripts/, references/, assets/) are copied with path containment checks
- Missing skills emit **warnings** and continue — not errors
- **CLAUDE.md assembly does NOT inject skill references** — skills are written to disk but the agent has no prompt telling it about them

The CLAUDE.md assembly in `adapter/claude.go` (lines 254-297) has numbered sections:
0. Base protocol preamble
1. Persona system prompt
2. Contract compliance section
2.5. Concurrency hint
3. Restriction section

There is **no skill section** in the CLAUDE.md assembly.

## Approach

### 1. Promote missing skill errors from warnings to hard errors

In `executor.go`, the skill provisioning at lines 1244-1337 emits warnings for missing skills. Change this to return an error that prevents step execution when a resolved skill is not found in the store (while still allowing `requires.skills`-based skills to use their own provisioning path).

### 2. Add skill content injection into CLAUDE.md assembly

Pass resolved skill information through the `AdapterRunConfig` to `claude.go`, which will inject a new section into the CLAUDE.md between the persona prompt and the contract compliance section. The section will list available skills with their descriptions and tell the agent where to find SKILL.md files.

### 3. Collect resolved skill metadata during provisioning

During the existing DirectoryStore provisioning loop, collect skill metadata (name, description) for each successfully provisioned skill. Pass this to the adapter config so CLAUDE.md assembly can reference it.

### 4. Keep existing `commands_glob` provisioning unchanged

The `requires.skills` (SkillConfig-backed) provisioning path remains as-is. It handles `commands_glob` patterns and copies to `.claude/commands/`. This is a separate mechanism from DirectoryStore skills.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/executor.go` | modify | Change missing-skill warnings to errors, collect skill metadata, pass to adapter config |
| `internal/adapter/adapter.go` | modify | Add `ResolvedSkills` field to `AdapterRunConfig` |
| `internal/adapter/claude.go` | modify | Add skill section to CLAUDE.md assembly |
| `internal/pipeline/executor_test.go` | modify | Add tests for missing-skill error behavior |
| `internal/adapter/claude_test.go` | modify | Add tests for CLAUDE.md skill section |
| `internal/skill/provision.go` | create | Extract worktree provisioning into dedicated function |
| `internal/skill/provision_test.go` | create | Tests for the new provisioning function |

## Architecture Decisions

### 1. Skill section in CLAUDE.md — inline descriptions, not full body

The CLAUDE.md skill section will include skill names and descriptions with paths to the SKILL.md files in the workspace. Full skill body content will NOT be inlined (it could be very large). Instead, the section tells the agent:
```
## Available Skills
The following skills are available in this workspace:
- **skill-name**: description (see `.wave/skills/skill-name/SKILL.md`)
```

This keeps CLAUDE.md concise while giving the agent enough context to discover and use skills.

### 2. Missing skills = hard error

A resolved skill that cannot be read from the store is a configuration error. The current behavior (warning + continue) can lead to silent failures where a step runs without expected skills. Changing to a hard error ensures misconfiguration is caught early.

### 3. Extract provisioning to `internal/skill/provision.go`

The DirectoryStore provisioning logic in executor.go (lines 1244-1337) is ~90 lines of file copying with path containment checks. Extracting it into a dedicated function in the skill package:
- Makes it independently testable
- Reduces executor.go complexity
- Provides a clean API: `ProvisionFromStore(store, workspacePath, skillNames) ([]SkillInfo, error)`

### 4. SkillInfo type for metadata passing

A new lightweight type `SkillInfo` with `Name` and `Description` fields will be used to pass provisioned skill metadata from executor to adapter. This avoids leaking the full `Skill` struct into the adapter package.

## Risks

| Risk | Mitigation |
|------|------------|
| Hard error for missing skills breaks existing pipelines with optional skills | Only error for DirectoryStore skills (not `requires.skills` which have their own check/install flow) |
| Large number of skills slows down provisioning | Skills are small files; copying is fast. No mitigation needed for current scale |
| CLAUDE.md skill section could conflict with persona prompt | Skill section is injected as a clearly delimited section with its own heading |

## Testing Strategy

1. **Unit tests for `ProvisionFromStore`**: success path, missing skill error, resource copying, deduplication, path traversal rejection
2. **Unit tests for CLAUDE.md assembly**: verify skill section is injected with correct content
3. **Executor integration tests**: verify end-to-end flow from resolved skills through provisioning to CLAUDE.md
4. **Regression tests**: ensure `commands_glob` provisioning still works unchanged
