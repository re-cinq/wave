# Implementation Plan: Register Wave as Claude Code Skill

## Objective

Add a new onboarding wizard step that generates a `.claude/commands/wave.md` command file during `wave init`, enabling users to invoke Wave pipelines directly from Claude Code sessions via `/wave` slash command.

## Approach

Add a new `WaveCommandStep` as Step 7 in the onboarding wizard, executed **after** skill selection and **before** manifest writing. This step generates a `.claude/commands/wave.md` file following Claude Code's command format (YAML frontmatter with `description` field + markdown body with `$ARGUMENTS` placeholder). The step is idempotent: it overwrites the file on re-init/reconfigure since the content is deterministic.

The command file will be a single `/wave` command that accepts subcommands as arguments (e.g., `/wave run impl-issue -- "fix bug"`, `/wave status`, `/wave list`), rather than separate command files per operation. This keeps the slash-command namespace clean and matches how the `wave` CLI itself works.

## File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `internal/onboarding/wave_command_step.go` | **create** | New wizard step: `WaveCommandStep` implementing `WizardStep` |
| `internal/onboarding/wave_command_step_test.go` | **create** | Unit tests for command file generation and idempotency |
| `internal/onboarding/onboarding.go` | **modify** | Register Step 7 in `RunWizard()`, add `WaveCommandGenerated` to `WizardResult` |
| `cmd/wave/commands/init.go` | **modify** | Ensure `.claude/commands/` directory is created in `runWizardInit()` |

## Architecture Decisions

1. **Single command file, not three**: The issue suggests `/wave-run`, `/wave-status`, `/wave-list` as separate commands. Instead, use a single `/wave` command that accepts subcommands via `$ARGUMENTS`. This follows the existing CLI pattern (`wave run`, `wave status`, `wave list`) and avoids polluting the slash-command namespace.

2. **Project-level `.claude/commands/`**: The issue mentions both `~/.claude/skills/` and `.claude/commands/`. The project-level `.claude/commands/` is the correct location — it's what the existing speckit commands use, it's version-controlled, and it's portable across team members.

3. **Wizard step, not post-wizard hook**: Implementing as a formal `WizardStep` follows the established pattern, makes testing straightforward, and lets us control ordering and error handling consistently.

4. **Overwrite-on-update**: Since the content is fully deterministic (no user-customizable parts), idempotency is achieved by simply overwriting. No merge logic needed.

5. **No manifest integration needed**: The wave command file is self-contained — it doesn't need to be listed in `manifest.Skills` since it's not a skill ecosystem package. It lives directly in `.claude/commands/` and is auto-discovered by Claude Code.

## Risks

| Risk | Mitigation |
|------|-----------|
| Claude Code command format changes | Use the same frontmatter format as existing speckit commands (proven working) |
| Users may want to customize the command | The file is in `.claude/commands/` and can be edited; we only overwrite on explicit `wave init` |
| Merge mode should also generate the file | Handle in `runMergeInit()` path as well, not just wizard path |

## Testing Strategy

1. **Unit tests** (`wave_command_step_test.go`):
   - Test `Run()` generates correct file at expected path
   - Test file content has valid YAML frontmatter
   - Test idempotency: running twice produces same result
   - Test non-interactive mode still generates the file
   - Test reconfigure mode overwrites existing file
   - Test `WaveDir` path is respected

2. **Integration**: Covered by existing `go test ./internal/onboarding/...` — the step follows the same `WizardStep` interface
