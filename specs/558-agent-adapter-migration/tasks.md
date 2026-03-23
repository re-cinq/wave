# Tasks: Migrate Adapter to Agent-Based Execution

**Feature**: #558 — Agent Adapter Migration
**Branch**: `558-agent-adapter-migration`
**Generated**: 2026-03-23

---

## Phase 1: Setup

- [X] T001 [P1] [Setup] Create feature branch and verify existing tests pass — run `go test ./internal/adapter/...` to establish green baseline before modifications

## Phase 2: Foundational — Type and Config Cleanup (US5, US3)

- [X] T002 [P1] [US5] Remove `UseAgentFlag` field and its comment from `AdapterRunConfig` struct in `internal/adapter/adapter.go:72-75`
- [X] T003 [P1] [US3] Add `SandboxOnlySettings` struct with single `Sandbox *SandboxSettings` field in `internal/adapter/claude.go` (after `NetworkSettings` type, ~line 47)
- [X] T004 [P1] [US3] Delete `ClaudeSettings` struct (`internal/adapter/claude.go:30-36`) and `ClaudePermissions` struct (`internal/adapter/claude.go:49-52`)

## Phase 3: Core Migration — prepareWorkspace (US1, US3)

- [X] T005 [P1] [US1] Remove full `settings.json` generation block in `prepareWorkspace` (`internal/adapter/claude.go:224-257`) — replace with conditional sandbox-only `settings.json` using `SandboxOnlySettings` written only when `cfg.SandboxEnabled` is true
- [X] T006 [P1] [US1] Add TodoWrite injection in `prepareWorkspace` before constructing `PersonaSpec` — append `"TodoWrite"` to `cfg.DenyTools` if not already present (`internal/adapter/claude.go`, between restriction section and agent file generation, ~line 297)
- [X] T007 [P1] [US1] Remove `if cfg.UseAgentFlag { ... } else { ... }` branch in `prepareWorkspace` (`internal/adapter/claude.go:298-334`) — make agent path unconditional (remove the CLAUDE.md assembly `else` branch entirely)
- [X] T008 [P1] [US1] Update `agentFilePath` comment to remove "when UseAgentFlag mode is active" wording (`internal/adapter/claude.go:207-210`)

## Phase 4: normalizeAllowedTools Removal (US2)

- [X] T009 [P1] [US2] Remove `normalizeAllowedTools` call in `PersonaToAgentMarkdown` — replace `normalized := normalizeAllowedTools(persona.AllowedTools)` with direct use of `persona.AllowedTools` (`internal/adapter/claude.go:914`)
- [X] T010 [P1] [US2] Delete `normalizeAllowedTools` function entirely (`internal/adapter/claude.go:957-989`)

## Phase 5: CLI Flag Simplification — buildArgs (US4)

- [X] T011 [P2] [US4] Remove `--model` flag from `buildArgs` — delete lines setting model (`internal/adapter/claude.go:407-412`); model is now in agent frontmatter
- [X] T012 [P2] [US4] Remove `if cfg.UseAgentFlag { ... } else { ... }` branch in `buildArgs` — make `--agent .claude/wave-agent.md` unconditional (`internal/adapter/claude.go:414-426`)
- [X] T013 [P2] [US4] Remove `--dangerously-skip-permissions` from `buildArgs` — replaced by `permissionMode: dontAsk` in agent frontmatter (`internal/adapter/claude.go:431`)
- [X] T014 [P2] [US4] Verify retained flags: `--output-format stream-json`, `--verbose`, `--no-session-persistence` remain in `buildArgs` (`internal/adapter/claude.go:428-435`)

## Phase 6: Test Migration (US1, US2, US3, US4, US5)

### Tests to DELETE
- [X] T015 [P] [P1] [US2] Delete `TestNormalizeAllowedTools` (8 test cases) — function removed (`internal/adapter/claude_test.go:41-102`)
- [X] T016 [P] [P1] [US4] Delete `TestBuildArgsNormalizesAllowedTools` — tests removed `--allowedTools` code path (`internal/adapter/claude_test.go:200-239`)
- [X] T017 [P] [P1] [US4] Delete `TestBuildArgsDisallowsTodoWrite` — TodoWrite now injected in prepareWorkspace, not CLI args (`internal/adapter/claude_test.go:242-261`)

### Tests to REWRITE
- [X] T018 [P1] [US3] Rewrite `TestSettingsJSONFormat` → `TestNoSettingsJSONWhenSandboxDisabled` — verify no `settings.json` created when sandbox disabled (`internal/adapter/claude_test.go:139-198`)
- [X] T019 [P1] [US3] Rewrite `TestSettingsJSONDenyRules` → `TestDenyRulesInAgentFrontmatter` — verify deny rules appear in agent `.md` frontmatter `disallowedTools:`, not in settings.json (`internal/adapter/claude_test.go:265-331`)
- [X] T020 [P1] [US3] Rewrite `TestSettingsJSONPerPersona` → `TestAgentFilePerPersona` — verify agent file generated per persona config with correct tools/deny/model in frontmatter (`internal/adapter/claude_test.go:875-979`)
- [X] T021 [P1] [US1] Rewrite `TestContractPromptInClaudeMD` → `TestContractPromptInAgentFile` — verify contract section in agent `.md` body, not in CLAUDE.md (`internal/adapter/claude_test.go:104-137`)
- [X] T022 [P1] [US1] Rewrite `TestCLAUDEMDRestrictionSection` → `TestAgentFileRestrictionSection` — verify restrictions appear in agent file body (`internal/adapter/claude_test.go:436-523`)
- [X] T023 [P1] [US1] Rewrite `TestBaseProtocolPrepended` → `TestBaseProtocolInAgentFile` — verify base protocol in agent file body with correct ordering (`internal/adapter/claude_test.go:1665-1720`)
- [X] T024 [P1] [US1] Rewrite `TestBaseProtocolWithInlinePrompt` → `TestBaseProtocolWithInlinePromptInAgentFile` — verify base protocol + inline prompt in agent file (`internal/adapter/claude_test.go:1745-1778`)

### Tests to UPDATE
- [X] T025 [P1] [US5] Update `TestPrepareWorkspaceAgentMode` — remove `UseAgentFlag: true` from config since agent mode is now default (`internal/adapter/claude_test.go:2226`)
- [X] T026 [P1] [US3] Update `TestSettingsJSONSandboxSettings` — change to verify sandbox-only settings.json format using `SandboxOnlySettings` instead of `ClaudeSettings` (`internal/adapter/claude_test.go:333-434`)
- [X] T027 [P1] [US2] Update `TestPersonaToAgentMarkdown` subtest "scoped tools are normalized" → change to verify passthrough behavior (both `Write` and `Write(.wave/output/*)` appear in output) (`internal/adapter/claude_test.go:2179-2191`)

### Tests to ADD
- [X] T028 [P] [P2] [US4] Add `TestBuildArgsAgentMode` — verify `--agent` present, no `--allowedTools`, `--disallowedTools`, `--dangerously-skip-permissions`, or `--model` in args (`internal/adapter/claude_test.go`)
- [X] T029 [P] [P1] [US1] Add `TestTodoWriteInjection` — verify TodoWrite added to `disallowedTools` in agent frontmatter when not in persona deny list (`internal/adapter/claude_test.go`)
- [X] T030 [P] [P1] [US1] Add `TestTodoWriteNoDuplication` — verify no duplicate when persona already denies TodoWrite (`internal/adapter/claude_test.go`)
- [X] T031 [P] [P1] [US1] Add `TestEmptyToolLists` — verify agent frontmatter omits `tools:` and `disallowedTools:` when lists are empty (edge case) (`internal/adapter/claude_test.go`)
- [X] T032 [P] [P1] [US3] Add `TestSandboxOnlySettingsJSON` — verify minimal settings.json with sandbox-only config (no model/temperature/permissions fields) (`internal/adapter/claude_test.go`)

## Phase 7: CLI Verification (US5)

- [X] T033 [P2] [US5] Verify `wave agent export` in `cmd/wave/commands/agent.go` still works — `PersonaSpecFromManifest` mapping and `PersonaToAgentMarkdown` call at line 303-311 need no changes (agent export uses same function, just verify no normalization call)

## Phase 8: Documentation Update (US5)

- [X] T034 [P] [P3] [US5] Update `docs/decisions/adr-agent-migration.md` — change status from "PoC implemented" to "Complete", remove "opt-in via UseAgentFlag" language
- [X] T035 [P] [P3] [US5] Update `docs/guides/adapter-development.md` — remove references to `UseAgentFlag`, `normalizeAllowedTools`, and the legacy CLAUDE.md code path (line 63 `UseAgentFlag bool` reference, line 477 `normalizeAllowedTools` reference)

## Phase 9: Final Validation

- [X] T036 [P1] [Final] Run `go test ./internal/adapter/...` — all adapter tests must pass
- [X] T037 [P1] [Final] Run `go test ./...` — full project test suite must pass
- [X] T038 [P1] [Final] Run `go build ./...` — verify clean compilation
- [X] T039 [P1] [Final] Verify SC-002: `grep -r "normalizeAllowedTools" --include="*.go"` returns zero results in non-spec files
- [X] T040 [P1] [Final] Verify SC-007: `grep -r "UseAgentFlag" --include="*.go"` returns zero results in non-spec files
- [X] T041 [P1] [Final] Verify SC-005: `buildArgs` output contains `--agent` and does NOT contain `--allowedTools`, `--disallowedTools`, `--dangerously-skip-permissions`, or `--model`

---

## Dependency Graph

```
T001 (baseline)
  ├→ T002, T003, T004 (types — can be parallel)
  │    └→ T005 (settings.json removal — depends on T003, T004)
  │         └→ T006 (TodoWrite injection)
  │              └→ T007 (UseAgentFlag branch removal — depends on T002)
  │                   └→ T008 (comment cleanup)
  │
  ├→ T009, T010 (normalizeAllowedTools — depends on T005 removing settings.json call)
  │
  ├→ T011, T012, T013, T014 (buildArgs — depends on T002)
  │
  ├→ T015-T032 (tests — depends on T002-T014 code changes)
  │    T015, T016, T017 can run in parallel (deletions)
  │    T018-T024 can run in parallel (rewrites)
  │    T025-T027 can run in parallel (updates)
  │    T028-T032 can run in parallel (additions)
  │
  ├→ T033 (CLI verification — depends on T009, T010)
  │
  ├→ T034, T035 (docs — can run in parallel, after all code changes)
  │
  └→ T036-T041 (final validation — after everything)
```

## Summary

| Phase | Tasks | Parallel? |
|-------|-------|-----------|
| 1: Setup | T001 | No |
| 2: Types | T002-T004 | Yes (3 parallel) |
| 3: prepareWorkspace | T005-T008 | Sequential |
| 4: normalizeAllowedTools | T009-T010 | Sequential |
| 5: buildArgs | T011-T014 | Sequential |
| 6: Tests | T015-T032 | Partially (groups parallel) |
| 7: CLI Verification | T033 | No |
| 8: Documentation | T034-T035 | Yes (2 parallel) |
| 9: Final Validation | T036-T041 | Partially |

**Total tasks**: 41
**Files modified**: `internal/adapter/adapter.go`, `internal/adapter/claude.go`, `internal/adapter/claude_test.go`, `docs/decisions/adr-agent-migration.md`, `docs/guides/adapter-development.md`
**Files verified (no changes)**: `cmd/wave/commands/agent.go`, `internal/pipeline/executor.go`
