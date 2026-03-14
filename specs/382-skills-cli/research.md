# Research: Wave Skills CLI

**Feature**: `wave skills` CLI command with subcommands for skill lifecycle management
**Date**: 2026-03-14

## Decision 1: Command Structure Pattern

**Decision**: Use Cobra parent command with `AddCommand()` subcommands, following the `wave migrate` pattern.

**Rationale**: The `migrate.go` command already demonstrates the exact pattern needed — a parent command grouping related subcommands (`up`, `down`, `status`, `validate`). This pattern:
- Uses `NewMigrateCmd()` returning `*cobra.Command` with `AddCommand()` for each sub
- Each subcommand follows `newMigrateXxxCmd()` private constructor convention
- The parent command with no args shows help text automatically

**Alternatives Rejected**:
- Flat top-level commands (`wave skills-list`, `wave skills-install`) — violates CLI ergonomics, clutters root command namespace
- Single command with positional arg (`wave skills list|install|remove`) without Cobra subcommands — loses flag isolation per subcommand, worse help text

## Decision 2: Output Format Integration

**Decision**: Each subcommand gets a local `--format` flag (table/json) with `ResolveFormat(cmd, localFormat)` for global flag precedence.

**Rationale**: This is the exact pattern established in `list.go:141`:
```go
opts.Format = ResolveFormat(cmd, opts.Format)
```
The `ResolveFormat()` function in `output.go:119` handles `--json > --quiet > --output > local` precedence correctly.

**Alternatives Rejected**:
- Custom flag resolution — would duplicate existing logic and risk inconsistency
- Global-only format flag — subcommands need different default columns/layout

## Decision 3: Store Initialization

**Decision**: Create `DirectoryStore` with project-level `.wave/skills/` (precedence 2) and user-level `.claude/skills/` (precedence 1) sources.

**Rationale**: The `DirectoryStore` in `store.go` already handles multi-source precedence. Higher precedence sources are checked first. Project-level skills take precedence over user-level.

**Code reference**: `store.go:107` — `NewDirectoryStore(sources ...SkillSource)` sorts by precedence descending.

**Alternatives Rejected**:
- Single-source store — loses the project vs user precedence model
- Manifest-based store — `DirectoryStore` is the authoritative SKILL.md store, not manifest `SkillConfig`

## Decision 4: Install Command Source Routing

**Decision**: Use `NewDefaultRouter(projectRoot)` from `source.go:132` to create a pre-registered router with all 7 adapters.

**Rationale**: The `SourceRouter` and all adapters (Tessl, BMAD, OpenSpec, SpecKit, GitHub, File, URL) are already implemented in `internal/skill/`. The `Install()` method handles parse → dispatch → write.

**Alternatives Rejected**:
- Manual prefix parsing in the CLI layer — duplicates `SourceRouter.Parse()` logic
- Lazy adapter registration — over-engineering, all 7 are lightweight to construct

## Decision 5: Error Code Strategy

**Decision**: Add 3 new error codes to `errors.go`: `skill_not_found`, `skill_source_error`, `skill_dependency_missing`.

**Rationale**: Follows the established pattern of domain-specific codes (`pipeline_not_found`, `adapter_not_found`, etc.). These codes enable machine-parseable `--format json` error responses.

**Mapping**:
- `skill_not_found` ← `store.ErrNotFound` wrapper (remove non-existent skill)
- `skill_source_error` ← `SourceRouter.Parse()` errors (unrecognized prefix, adapter failure)
- `skill_dependency_missing` ← `*DependencyError` from adapter (missing `tessl`, `npx`, etc.)

## Decision 6: Confirmation Prompt Pattern

**Decision**: Reuse `promptConfirm(in io.Reader, out io.Writer, prompt string)` from `doctor.go:255`.

**Rationale**: This function already exists, accepts injectable I/O for testability, and handles Y/n parsing with default yes on empty input.

**Implementation**: The `remove` subcommand options struct holds `in io.Reader` and `out io.Writer` fields, defaulting to `os.Stdin`/`os.Stderr` in the cobra command, overridable in tests.

## Decision 7: Search and Sync Delegation

**Decision**: Shell out to `tessl search <query>` and `tessl install --project-dependencies` respectively, capturing stdout for parsing.

**Rationale**: The Tessl CLI is the authoritative registry interface. Wave wraps it rather than reimplementing registry protocol. This matches the adapter pattern in `source_cli.go` where `TesslAdapter.Install()` already shells out to `tessl install`.

**Implementation**:
- `search`: `exec.CommandContext(ctx, "tessl", "search", query)` → parse stdout table → format for display
- `sync`: `exec.CommandContext(ctx, "tessl", "install", "--project-dependencies")` → capture stdout → report results

**Alternatives Rejected**:
- Direct Tessl API calls — would require understanding the Tessl API protocol, fragile coupling
- Generic registry interface — over-engineering for a single registry
