# Tasks: Wave Skills CLI

**Feature**: `wave skills` CLI command — list/search/install/remove/sync
**Branch**: `382-skills-cli`
**Generated**: 2026-03-14
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Data Model**: [data-model.md](data-model.md)

---

## Phase 1: Setup — Error Codes & Registration

- [X] T001 [P1] [Setup] Add 3 skill error code constants (`CodeSkillNotFound`, `CodeSkillSourceError`, `CodeSkillDependencyMissing`) to `cmd/wave/commands/errors.go`
- [X] T002 [P1] [Setup] Add `rootCmd.AddCommand(commands.NewSkillsCmd())` to `cmd/wave/main.go` init function (line ~154, after existing AddCommand calls)

## Phase 2: Foundational — Parent Command & Output Structs

- [X] T003 [P1] [Foundational] Create `cmd/wave/commands/skills.go` with package declaration, imports, and CLI output structs: `SkillListItem`, `SkillListOutput`, `SkillInstallOutput`, `SkillRemoveOutput`, `SkillSearchResult`, `SkillSyncOutput` — as defined in [data-model.md](data-model.md)
- [X] T004 [P1] [Foundational] Implement `NewSkillsCmd() *cobra.Command` parent command in `cmd/wave/commands/skills.go` — Use `wave skills`, Short description, Long description listing all 5 subcommands, wires `AddCommand()` for each subcommand constructor. Follow `NewMigrateCmd()` pattern from `cmd/wave/commands/migrate.go`
- [X] T005 [P1] [Foundational] Implement `newSkillStore() *skill.DirectoryStore` helper in `cmd/wave/commands/skills.go` — creates `DirectoryStore` with `.wave/skills/` (precedence 2) and `~/.claude/skills/` (precedence 1) sources using `os.UserHomeDir()` for home directory resolution

## Phase 3: User Story 1 — List Installed Skills (P1)

- [X] T006 [P1] [Story1] Implement `newSkillsListCmd() *cobra.Command` in `cmd/wave/commands/skills.go` — `wave skills list`, local `--format` flag (table/json, default table), calls `runSkillsList`. Use `cobra.NoArgs` for argument validation
- [X] T007 [P1] [Story1] Implement `runSkillsList(cmd *cobra.Command, format string) error` in `cmd/wave/commands/skills.go` — calls `newSkillStore().List()`, calls `collectSkillPipelineUsage()` (reuse from `cmd/wave/commands/list.go:1372`), maps `[]skill.Skill` + pipeline usage to `SkillListOutput`. Handle `*skill.DiscoveryError` for partial results with warnings. Empty state prints hint to use `wave skills install`
- [X] T008 [P1] [Story1] Implement table formatting for `list` output in `runSkillsList` — columns: Name, Description, Source, Used By. Use `fmt.Fprintf` with `text/tabwriter` or manual column alignment matching existing CLI patterns
- [X] T009 [P1] [Story1] Implement JSON formatting for `list` output in `runSkillsList` — marshal `SkillListOutput` to stdout via `json.NewEncoder(os.Stdout).Encode()`. Resolve format via `ResolveFormat(cmd, localFormat)` per `cmd/wave/commands/output.go:119`

## Phase 4: User Story 2 — Install a Skill from Any Source (P1)

- [X] T010 [P1] [Story2] Implement `newSkillsInstallCmd() *cobra.Command` in `cmd/wave/commands/skills.go` — `wave skills install <source>`, local `--format` flag, calls `runSkillsInstall`. Use `cobra.ExactArgs(1)` for argument validation
- [X] T011 [P1] [Story2] Implement `runSkillsInstall(cmd *cobra.Command, source, format string) error` in `cmd/wave/commands/skills.go` — creates `DirectoryStore` and `SourceRouter` via `skill.NewDefaultRouter(".")`, calls `router.Install(ctx, source, store)`. Maps `*skill.DependencyError` → `CodeSkillDependencyMissing`, router parse errors → `CodeSkillSourceError` using `classifySkillError()`. Format table (success message with installed skill names) or JSON (`SkillInstallOutput`)
- [X] T012 [P1] [Story2] Implement `classifySkillError(err error) *CLIError` helper in `cmd/wave/commands/skills.go` — maps `skill.ErrNotFound` → `CodeSkillNotFound`, `*skill.DependencyError` → `CodeSkillDependencyMissing` with install instructions, unrecognized prefix errors → `CodeSkillSourceError` with recognized prefix list

## Phase 5: User Story 3 — Remove an Installed Skill (P2)

- [X] T013 [P2] [Story3] Implement `newSkillsRemoveCmd() *cobra.Command` in `cmd/wave/commands/skills.go` — `wave skills remove <name>`, local `--format` flag, `--yes` flag for skip confirmation. Use `cobra.ExactArgs(1)`. Options struct holds `in io.Reader` and `out io.Writer` fields, defaulting to `os.Stdin`/`os.Stderr`
- [X] T014 [P2] [Story3] Implement `runSkillsRemove(cmd *cobra.Command, name, format string, yes bool, in io.Reader, out io.Writer) error` in `cmd/wave/commands/skills.go` — if `--yes` not set, call `promptConfirm(in, out, "Remove skill \"<name>\"? [Y/n] ")` (reuse from `cmd/wave/commands/doctor.go:255`). Call `newSkillStore().Delete(name)`. Map `ErrNotFound` → `CodeSkillNotFound` with suggestion listing installed skills. Format table or JSON (`SkillRemoveOutput`)

## Phase 6: User Story 4 — Search the Tessl Registry (P3)

- [X] T015 [P3] [Story4] Implement `newSkillsSearchCmd() *cobra.Command` in `cmd/wave/commands/skills.go` — `wave skills search <query>`, local `--format` flag. Use `cobra.ExactArgs(1)`
- [X] T016 [P3] [Story4] Implement `runSkillsSearch(cmd *cobra.Command, query, format string) error` in `cmd/wave/commands/skills.go` — check `tessl` via `exec.LookPath("tessl")`, return `CodeSkillDependencyMissing` error if missing. Run `exec.CommandContext(ctx, "tessl", "search", query)`, parse stdout lines into `[]SkillSearchResult`. Format table (Name, Rating, Description columns) or JSON

## Phase 7: User Story 5 — Sync Project Dependencies (P3)

- [X] T017 [P3] [Story5] Implement `newSkillsSyncCmd() *cobra.Command` in `cmd/wave/commands/skills.go` — `wave skills sync`, local `--format` flag. Use `cobra.NoArgs`
- [X] T018 [P3] [Story5] Implement `runSkillsSync(cmd *cobra.Command, format string) error` in `cmd/wave/commands/skills.go` — check `tessl` via `exec.LookPath("tessl")`, return `CodeSkillDependencyMissing` if missing. Run `exec.CommandContext(ctx, "tessl", "install", "--project-dependencies")`, parse output for installed/updated skill names. Format table (summary) or JSON (`SkillSyncOutput`)

## Phase 8: Tests

- [X] T019 [P1] [Tests] [P] Create `cmd/wave/commands/skills_test.go` with `TestSkillsListEmpty` — populate `t.TempDir()` with empty skill directories, verify "no skills installed" message with install hint
- [X] T020 [P1] [Tests] [P] Add `TestSkillsListWithSkills` — create SKILL.md files in temp dir, verify table output contains correct Name, Description, Source columns
- [X] T021 [P1] [Tests] [P] Add `TestSkillsListJSON` — verify `--format json` output parses into `SkillListOutput` via `json.Unmarshal`
- [X] T022 [P1] [Tests] [P] Add `TestSkillsListDiscoveryWarnings` — create malformed SKILL.md, verify warnings included in output without command failure
- [X] T023 [P1] [Tests] [P] Add `TestSkillsInstallUnknownPrefix` — call install with `unknown:something`, verify `skill_source_error` code and recognized prefix list in error
- [X] T024 [P1] [Tests] [P] Add `TestSkillsInstallNoArgs` — run without args, verify error with usage hint
- [X] T025 [P1] [Tests] [P] Add `TestSkillsInstallFileSource` — call `file:./path` with valid skill dir, verify success or correct dispatch (mock or real `FileAdapter`)
- [X] T026 [P1] [Tests] [P] Add `TestSkillsInstallJSON` — verify `--format json` output parses into `SkillInstallOutput`
- [X] T027 [P2] [Tests] [P] Add `TestSkillsRemoveExisting` — create skill in temp dir, remove it, verify deleted
- [X] T028 [P2] [Tests] [P] Add `TestSkillsRemoveNonexistent` — verify `skill_not_found` error code
- [X] T029 [P2] [Tests] [P] Add `TestSkillsRemoveConfirmation` — inject `strings.NewReader("y\n")` as stdin, verify prompt shown and skill deleted. Also test "n\n" aborts
- [X] T030 [P2] [Tests] [P] Add `TestSkillsRemoveYesFlag` — verify `--yes` skips prompt
- [X] T031 [P2] [Tests] [P] Add `TestSkillsRemoveJSON` — verify JSON output structure
- [X] T032 [P3] [Tests] [P] Add `TestSkillsSearchMissingTessl` — verify `skill_dependency_missing` error when `tessl` not in PATH
- [X] T033 [P3] [Tests] [P] Add `TestSkillsSyncMissingTessl` — verify `skill_dependency_missing` error when `tessl` not in PATH
- [X] T034 [P1] [Tests] Add `TestSkillsNoSubcommand` — run `wave skills` with no args, verify help text lists all 5 subcommands

## Phase 9: Polish & Cross-Cutting

- [X] T035 [P1] [Polish] Verify `go build ./cmd/wave/...` compiles without errors
- [X] T036 [P1] [Polish] Run `go test -race ./cmd/wave/commands/...` and fix any failures or data races
- [X] T037 [P2] [Polish] Run `golangci-lint run ./cmd/wave/commands/...` and fix any lint violations
