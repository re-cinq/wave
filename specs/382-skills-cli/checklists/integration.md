# Integration Quality Checklist: Wave Skills CLI

**Feature**: `wave skills` CLI — list/search/install/remove/sync
**Focus**: Cross-component integration, dependency boundaries, and system contracts
**Generated**: 2026-03-14

---

## Dependency Boundaries

- [ ] CHK201 - Are the required interfaces from `internal/skill` (DirectoryStore, SourceRouter, InstallResult, DependencyError) pinned to specific types, or is there an abstraction boundary for testability? [Completeness]
- [ ] CHK202 - Is the `collectSkillPipelineUsage()` function's expected signature and return type documented for reuse from list.go? [Clarity]
- [ ] CHK203 - Does the spec define whether `collectSkillPipelineUsage()` should be extracted to a shared utility or called directly from list.go? [Clarity]
- [ ] CHK204 - Are the adapter dependencies (#381 DirectoryStore, #383 SourceRouter, #385 hierarchy config) confirmed as merged prerequisites? [Completeness]
- [ ] CHK205 - Is the `skill.NewDefaultRouter(projectRoot)` call's project root parameter resolution specified (cwd vs manifest location)? [Clarity]

## Cross-Command Consistency

- [ ] CHK206 - Does the spec address how `wave skills install` followed by `wave skills list` reflects the newly installed skill? [Consistency]
- [ ] CHK207 - Is the skill name matching semantics between install (by source), remove (by name), and list (all) consistently defined? [Consistency]
- [ ] CHK208 - Does the spec define whether `wave skills remove` can remove skills from user-level .claude/skills/ or only project-level .wave/skills/? [Completeness]
- [ ] CHK209 - Is the behavior specified when a removed skill is re-installed — does it go to the same precedence level? [Coverage]

## Testing Integration

- [ ] CHK210 - Are mock boundaries clearly defined — which components are mocked vs used directly in tests? [Clarity]
- [ ] CHK211 - Does the test strategy address testing the cobra command wiring (flag parsing, arg validation) separately from business logic? [Coverage]
- [ ] CHK212 - Are integration test requirements (e.g., tests requiring tessl CLI) clearly separated from unit tests? [Clarity]
- [ ] CHK213 - Is `go test -race` explicitly required in the task list, not just the success criteria? [Consistency]
