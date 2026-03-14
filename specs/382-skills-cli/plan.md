# Implementation Plan: Wave Skills CLI

**Branch**: `382-skills-cli` | **Date**: 2026-03-14 | **Spec**: [specs/382-skills-cli/spec.md](spec.md)
**Input**: Feature specification from `/specs/382-skills-cli/spec.md`

## Summary

Add a `wave skills` top-level CLI command with 5 subcommands (`list`, `search`, `install`, `remove`, `sync`) for skill lifecycle management. The implementation wires existing `internal/skill` infrastructure (`DirectoryStore`, `SourceRouter`, all 7 source adapters) to a new Cobra command tree in `cmd/wave/commands/skills.go`. All subcommands support `--format table|json` with global flag precedence via `ResolveFormat()`. The `search` and `sync` subcommands delegate to the `tessl` CLI. The `remove` subcommand uses the `promptConfirm()` pattern from `doctor.go` for testable confirmation prompts.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `github.com/spf13/cobra`, `gopkg.in/yaml.v3`, `internal/skill` package
**Storage**: Filesystem — `DirectoryStore` reading from `.wave/skills/` and `.claude/skills/`
**Testing**: `go test -race ./cmd/wave/commands/...` — table-driven tests with mock store
**Target Platform**: Linux/macOS CLI (single static binary)
**Project Type**: Single Go binary (existing project)
**Constraints**: Single static binary, no new runtime dependencies

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new dependencies beyond existing Go stdlib + cobra |
| P2: Manifest as SSOT | PASS | Skills store uses `DirectoryStore` filesystem, not manifest. Command registered in root via `AddCommand()` |
| P3: Persona-Scoped Boundaries | N/A | CLI command, not a persona |
| P4: Fresh Memory | N/A | CLI command, not a pipeline step |
| P5: Navigator-First | N/A | CLI command, not a pipeline |
| P6: Contracts at Handover | N/A | CLI command, not a pipeline |
| P7: Relay via Summarizer | N/A | CLI command, not a pipeline |
| P8: Ephemeral Workspaces | N/A | CLI command, not a pipeline |
| P9: Credentials Never Touch Disk | PASS | No credentials involved — `tessl` CLI handles its own auth |
| P10: Observable Progress | PASS | Structured JSON output via `--format json`, error codes for machine parsing |
| P11: Bounded Recursion | N/A | No recursion |
| P12: Minimal State Machine | N/A | CLI command, not a pipeline step |
| P13: Test Ownership | PASS | All new code has unit tests, `go test -race` required |

**Post-Phase-1 Re-check**: All principles remain in compliance. No violations detected.

## Project Structure

### Documentation (this feature)

```
specs/382-skills-cli/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 research
├── data-model.md        # Phase 1 data model
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
cmd/wave/
├── main.go                      # Add rootCmd.AddCommand(commands.NewSkillsCmd())
└── commands/
    ├── errors.go                # Add 3 new error code constants
    ├── skills.go                # NEW: wave skills parent + all 5 subcommands
    └── skills_test.go           # NEW: unit tests for all subcommands
```

**Structure Decision**: All skill CLI code goes in a single `skills.go` file following the `migrate.go` pattern — parent command with private subcommand constructors. Tests in a paired `skills_test.go`. This keeps the feature contained in 2 new files plus 2 small edits to existing files (`main.go` registration, `errors.go` constants).

## Implementation Architecture

### File: `cmd/wave/commands/skills.go`

**Public API**:
- `NewSkillsCmd() *cobra.Command` — top-level `wave skills` command, registered in `main.go`

**Private subcommand constructors**:
- `newSkillsListCmd() *cobra.Command` — `wave skills list [--format table|json]`
- `newSkillsInstallCmd() *cobra.Command` — `wave skills install <source> [--format table|json]`
- `newSkillsRemoveCmd() *cobra.Command` — `wave skills remove <name> [--yes] [--format table|json]`
- `newSkillsSearchCmd() *cobra.Command` — `wave skills search <query> [--format table|json]`
- `newSkillsSyncCmd() *cobra.Command` — `wave skills sync [--format table|json]`

**Implementation functions** (separated from cobra wiring for testability):
- `runSkillsList(format string) error`
- `runSkillsInstall(source, format string) error`
- `runSkillsRemove(name, format string, yes bool, in io.Reader, out io.Writer) error`
- `runSkillsSearch(query, format string) error`
- `runSkillsSync(format string) error`

**Shared helpers**:
- `newSkillStore() *skill.DirectoryStore` — creates store with `.wave/skills/` (precedence 2) and `~/.claude/skills/` (precedence 1)
- `classifySkillError(err error) *CLIError` — maps `skill.ErrNotFound`, `*skill.DependencyError`, and router errors to `CLIError` with appropriate codes

### File: `cmd/wave/commands/errors.go`

Add 3 new constants:
```go
CodeSkillNotFound        = "skill_not_found"
CodeSkillSourceError     = "skill_source_error"
CodeSkillDependencyMissing = "skill_dependency_missing"
```

### File: `cmd/wave/main.go`

Add one line:
```go
rootCmd.AddCommand(commands.NewSkillsCmd())
```

### Subcommand Detail

#### `wave skills list`

1. Create `DirectoryStore` with project + user sources
2. Call `store.List()` → `[]skill.Skill`
3. Call `collectSkillPipelineUsage()` (reuse from `list.go`) for pipeline usage
4. Handle `*skill.DiscoveryError` — partial results with warnings
5. Format: table (columns: Name, Description, Source, Used By) or JSON (`SkillListOutput`)
6. Empty state: informative message with `wave skills install` hint

#### `wave skills install`

1. Validate source argument provided (exactly 1 arg)
2. Create `DirectoryStore` and `SourceRouter` via `skill.NewDefaultRouter(".")`
3. Call `router.Install(ctx, source, store)` → `*skill.InstallResult`
4. Map errors: `*skill.DependencyError` → `CodeSkillDependencyMissing`, router parse errors → `CodeSkillSourceError`
5. Format: table (success message with skill names) or JSON (`SkillInstallOutput`)

#### `wave skills remove`

1. Validate name argument provided (exactly 1 arg)
2. Create `DirectoryStore`
3. If `--yes` not set: call `promptConfirm(in, out, "Remove skill \"<name>\"? [Y/n] ")`
4. Call `store.Delete(name)`
5. Map `ErrNotFound` → `CodeSkillNotFound` with suggestion listing installed skills
6. Format: table (success message) or JSON (`SkillRemoveOutput`)

#### `wave skills search`

1. Validate query argument provided (exactly 1 arg)
2. Check `tessl` CLI dependency via `exec.LookPath("tessl")`
3. Run `tessl search <query>` → capture stdout
4. Parse output lines into `[]SkillSearchResult`
5. Format: table (columns: Name, Rating, Description) or JSON

#### `wave skills sync`

1. Check `tessl` CLI dependency via `exec.LookPath("tessl")`
2. Run `tessl install --project-dependencies` → capture stdout
3. Parse output for installed/updated skill names
4. Format: table (summary message) or JSON (`SkillSyncOutput`)

### Testing Strategy

**File**: `cmd/wave/commands/skills_test.go`

Tests use `t.TempDir()` for skill store isolation:

1. **TestSkillsListEmpty** — empty store shows "no skills installed" message
2. **TestSkillsListWithSkills** — populated store shows correct table columns
3. **TestSkillsListJSON** — JSON output parses into `SkillListOutput`
4. **TestSkillsListDiscoveryWarnings** — malformed SKILL.md included in warnings
5. **TestSkillsInstallFileSource** — `file:./path` dispatches correctly (mock or real file adapter)
6. **TestSkillsInstallUnknownPrefix** — error lists recognized prefixes
7. **TestSkillsInstallNoArgs** — error with usage hint
8. **TestSkillsInstallJSON** — JSON output structure
9. **TestSkillsRemoveExisting** — skill deleted successfully
10. **TestSkillsRemoveNonexistent** — `skill_not_found` error code
11. **TestSkillsRemoveConfirmation** — prompt confirms before delete (injectable reader)
12. **TestSkillsRemoveYesFlag** — skip confirmation with `--yes`
13. **TestSkillsRemoveJSON** — JSON output structure
14. **TestSkillsSearchMissingTessl** — `skill_dependency_missing` error when tessl not found
15. **TestSkillsSyncMissingTessl** — `skill_dependency_missing` error when tessl not found
16. **TestSkillsNoSubcommand** — shows help text with all 5 subcommands listed

Tests for `search` and `sync` that require the `tessl` CLI focus on the error paths (missing dependency). Integration tests with actual `tessl` are deferred to CI environments where `tessl` is available.

## Complexity Tracking

_No constitution violations. No complexity justifications needed._
