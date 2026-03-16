# feat: register wave as a Claude Code skill during wave init

**Issue**: [#448](https://github.com/re-cinq/wave/issues/448)
**Labels**: enhancement
**Author**: nextlevelshit

## Description

When running `wave init`, Wave should register itself as a Claude Code skill so that users can invoke Wave pipelines directly from Claude Code sessions via slash commands.

### Current Behavior
- `wave init` creates `wave.yaml`, `.wave/` directory structure, personas, pipelines
- No integration with Claude Code's skill system

### Expected Behavior
- `wave init` should add a Wave skill entry to `.claude/commands/` (project-level)
- The skill should expose Wave's key commands as slash commands accessible from Claude Code:
  - `/wave-run <pipeline> -- <input>` — run a pipeline
  - `/wave-status` — show active runs
  - `/wave-list` — list available pipelines
- The skill definition should follow Claude Code's command format (`.md` file with YAML frontmatter)

### Implementation Notes
- Generate a `wave.md` command file during `wave init` in `.claude/commands/`
- Optionally update existing skill if Wave is already initialized
- The skill should work with Claude Code's `--agent` flag for deeper integration

### Acceptance Criteria
- [ ] `wave init` creates a Claude Code command file at `.claude/commands/wave.md`
- [ ] Command file is valid Claude Code command format (YAML frontmatter + markdown body)
- [ ] Command exposes wave run, status, and list operations
- [ ] Idempotent — re-running `wave init` updates rather than duplicates
- [ ] Tests for command file generation
