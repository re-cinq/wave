# feat(skills): worktree provisioning — copy resolved skills into workspaces and CLAUDE.md assembly

**Issue**: [#386](https://github.com/re-cinq/wave/issues/386)
**Parent**: [#239](https://github.com/re-cinq/wave/issues/239)
**Labels**: enhancement, personas, pipeline

## Summary

Implement worktree provisioning of resolved skills into each pipeline step's workspace. After skill resolution (merging global, persona, and pipeline scopes), copy the corresponding SKILL.md files from `.wave/skills/` into the step's worktree and inject skill references into the assembled runtime CLAUDE.md. This ensures each agent has access to exactly the skills configured for its execution context.

## Acceptance Criteria

- [ ] Resolved skills (from hierarchical merge) are copied into each step's worktree at `.wave/skills/` or the appropriate skill discovery path
- [ ] Skill SKILL.md content is injected/referenced in the assembled runtime CLAUDE.md for each step
- [ ] Skills are deduplicated before provisioning — no duplicate files in the worktree
- [ ] File conflicts from overlapping scopes are resolved using precedence (more specific scope wins)
- [ ] Provisioning integrates with the existing workspace creation flow in `internal/workspace/`
- [ ] Provisioning integrates with the existing CLAUDE.md assembly in `internal/pipeline/executor.go`
- [ ] Existing per-step skill behavior (`commands_glob` pattern) is preserved and not broken
- [ ] Missing skills (referenced but not installed) produce clear errors before step execution begins
- [ ] Performance: skill provisioning does not significantly slow down step startup

## Dependencies

- **Skill store core** (sub-issue #1 / PR #389) — reads skills from the store (MERGED)
- **Hierarchical configuration** (sub-issue #5 / PR #390) — receives resolved skill sets (MERGED)

## Scope Notes

- **In scope**: Worktree file copying, CLAUDE.md injection, conflict resolution, integration with executor
- **Out of scope**: Skill resolution logic (sub-issue #5), CLI commands (sub-issue #2), ecosystem adapters (sub-issue #3)
- **Reference**: `internal/workspace/` for workspace creation, `internal/pipeline/executor.go` for CLAUDE.md assembly, `internal/skill/skill.go` for existing `Provisioner`
