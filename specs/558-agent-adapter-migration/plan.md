# Implementation Plan: Migrate Adapter to Agent-Based Execution

**Branch**: `558-agent-adapter-migration` | **Date**: 2026-03-23 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/558-agent-adapter-migration/spec.md`

## Summary

Migrate `ClaudeAdapter` from the three-surface permission model (`settings.json` + CLI flags + CLAUDE.md) to self-contained agent `.md` files using Claude Code's `--agent` flag. This collapses tool permissions, model selection, and system prompts into a single artifact, eliminating `normalizeAllowedTools`, `UseAgentFlag`, and the full `ClaudeSettings`/`ClaudePermissions` types.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `os/exec`, `encoding/json`, `strings`, `path/filepath`
**Storage**: Filesystem (agent .md files in workspace `.claude/` directory)
**Testing**: `go test ./...` with table-driven tests
**Target Platform**: Linux (primary), macOS (secondary)
**Project Type**: Single Go binary
**Constraints**: No backward compatibility constraint (prototype phase per constitution)
**Scale/Scope**: 4 modified files, 2 deleted types, 1 deleted function, ~20 test modifications

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new dependencies. Agent .md is plain text. |
| P2: Manifest as SSOT | PASS | Persona definitions still in wave.yaml. Agent file is a runtime compilation artifact. |
| P3: Persona-Scoped Execution | PASS | Each step still gets exactly one persona. Agent file is the new binding mechanism. |
| P4: Fresh Memory | PASS | Agent files are per-workspace. No chat history inheritance. |
| P5: Navigator-First | N/A | Pipeline structure unchanged. |
| P6: Contracts at Handover | PASS | Contract section still assembled into agent file body. |
| P7: Relay via Summarizer | N/A | Relay mechanism unchanged. |
| P8: Ephemeral Workspaces | PASS | Agent files live inside workspace `.claude/` directory. |
| P9: Credentials Never Touch Disk | PASS | No credential changes. Agent files contain only tool names, not secrets. |
| P10: Observable Progress | PASS | Stream parsing unchanged. |
| P11: Bounded Recursion | N/A | Resource limits unchanged. |
| P12: Minimal State Machine | N/A | Step state transitions unchanged. |
| P13: Test Ownership | PASS | All existing tests must pass. Tests for deleted code are deleted. New tests added for new behavior. |

**Post-Phase 1 re-check**: No violations found. The migration simplifies the adapter without introducing new architectural concepts.

## Project Structure

### Documentation (this feature)

```
specs/558-agent-adapter-migration/
├── plan.md              # This file
├── research.md          # Phase 0: Decision rationale for each design choice
├── data-model.md        # Phase 1: Entity definitions and type changes
├── spec.md              # Feature specification (from specify step)
└── checklists/          # Verification checklists
```

### Source Code (repository root)

```
internal/adapter/
├── adapter.go           # MODIFY: Remove UseAgentFlag from AdapterRunConfig
├── claude.go            # MODIFY: Core migration — prepareWorkspace, buildArgs, types
└── claude_test.go       # MODIFY: Update/delete tests for removed code paths

cmd/wave/commands/
├── agent.go             # VERIFY: wave agent export still works (no changes expected)
└── agent_test.go        # VERIFY: agent CLI tests still pass

internal/pipeline/
└── executor.go          # VERIFY: No changes needed (UseAgentFlag not set here)

docs/decisions/
└── adr-agent-migration.md  # UPDATE: Mark migration as complete, not PoC
docs/guides/
└── adapter-development.md  # UPDATE: Remove references to UseAgentFlag and normalizeAllowedTools
```

**Structure Decision**: All changes are within the existing project structure. No new packages or files needed.

## Implementation Phases

### Phase A: Type and Config Cleanup (P1 — Foundation)

**Goal**: Remove `UseAgentFlag` from `AdapterRunConfig` and simplify types.

**Files modified**:
- `internal/adapter/adapter.go` — Remove `UseAgentFlag` field from `AdapterRunConfig`
- `internal/adapter/claude.go` — Replace `ClaudeSettings`/`ClaudePermissions` with `SandboxOnlySettings`

**Changes**:
1. Delete `UseAgentFlag bool` field and its comment from `AdapterRunConfig`
2. Add `SandboxOnlySettings` struct with only `Sandbox *SandboxSettings` field
3. Keep `SandboxSettings` and `NetworkSettings` structs unchanged
4. Delete `ClaudeSettings` struct
5. Delete `ClaudePermissions` struct

**Tests impacted**:
- `TestPrepareWorkspaceAgentMode` — Remove `UseAgentFlag: true` from config
- `TestSettingsJSONFormat` — Will need rewrite (no longer writes full settings.json)
- `TestSettingsJSONDenyRules` — Will need rewrite (deny rules are in agent frontmatter, not settings.json)
- `TestSettingsJSONSandboxSettings` — Update to verify minimal sandbox-only settings.json
- `TestSettingsJSONPerPersona` — Will need rewrite

### Phase B: prepareWorkspace Migration (P1 — Core)

**Goal**: Make agent mode unconditional. Remove CLAUDE.md branch. Remove settings.json generation for non-sandbox cases.

**Files modified**:
- `internal/adapter/claude.go` — `prepareWorkspace()` method

**Changes**:
1. Remove the `if cfg.UseAgentFlag { ... } else { ... }` branch — agent path becomes the only path
2. Add TodoWrite injection: before constructing PersonaSpec, append `"TodoWrite"` to `cfg.DenyTools` if not already present (FR-006, C-004)
3. Remove the settings.json generation block that writes full `ClaudeSettings`
4. Add conditional sandbox-only settings.json: if `cfg.SandboxEnabled`, write `SandboxOnlySettings` to `.claude/settings.json`
5. The base protocol loading, system prompt loading, skill section, contract section, concurrency hint, and restriction section assembly all remain — they feed into `PersonaToAgentMarkdown()`

**Before** (simplified):
```go
func (a *ClaudeAdapter) prepareWorkspace(workspacePath string, cfg AdapterRunConfig) error {
    // ... create .claude dir ...

    // Write full settings.json (REMOVE)
    settings := ClaudeSettings{Model: ..., Permissions: ...}
    writeSettingsJSON(settings)

    // Load base protocol, system prompt, contract, restrictions (KEEP)

    if cfg.UseAgentFlag {
        // Agent mode (MAKE UNCONDITIONAL)
        spec := PersonaSpec{...}
        agentMd := PersonaToAgentMarkdown(spec, ...)
        writeAgentFile(agentMd)
    } else {
        // CLAUDE.md mode (DELETE)
        assembleCLAUDEMD(...)
    }
}
```

**After** (simplified):
```go
func (a *ClaudeAdapter) prepareWorkspace(workspacePath string, cfg AdapterRunConfig) error {
    // ... create .claude dir ...

    // Write sandbox-only settings.json (NEW — conditional)
    if cfg.SandboxEnabled {
        settings := SandboxOnlySettings{Sandbox: &SandboxSettings{...}}
        writeSettingsJSON(settings)
    }

    // Load base protocol, system prompt, contract, restrictions (KEEP)

    // Inject TodoWrite into deny list (NEW)
    denyTools := cfg.DenyTools
    if !containsTodoWrite(denyTools) {
        denyTools = append(denyTools, "TodoWrite")
    }

    // Agent mode (UNCONDITIONAL)
    spec := PersonaSpec{Model: cfg.Model, AllowedTools: cfg.AllowedTools, DenyTools: denyTools}
    agentMd := PersonaToAgentMarkdown(spec, ...)
    writeAgentFile(agentMd)
}
```

### Phase C: PersonaToAgentMarkdown Cleanup (P1 — Remove normalizeAllowedTools)

**Goal**: Remove `normalizeAllowedTools` from both `PersonaToAgentMarkdown` and the codebase.

**Files modified**:
- `internal/adapter/claude.go` — `PersonaToAgentMarkdown()` and `normalizeAllowedTools()`

**Changes**:
1. In `PersonaToAgentMarkdown()`: replace `normalized := normalizeAllowedTools(persona.AllowedTools)` with direct use of `persona.AllowedTools`
2. Delete the `normalizeAllowedTools()` function entirely
3. Remove the call to `normalizeAllowedTools` in the old settings.json generation (already removed in Phase B)

### Phase D: buildArgs Simplification (P2 — CLI flags)

**Goal**: Simplify CLI argument assembly to use `--agent` unconditionally.

**Files modified**:
- `internal/adapter/claude.go` — `buildArgs()` method

**Changes**:
1. Remove `--model` flag (model is in agent frontmatter)
2. Remove `if cfg.UseAgentFlag` branch — always pass `--agent .claude/wave-agent.md`
3. Remove the `else` branch (no more `--allowedTools`, `--disallowedTools`)
4. Remove `--dangerously-skip-permissions` (replaced by `permissionMode: dontAsk` in frontmatter)
5. Keep: `--output-format stream-json`, `--verbose`, `--no-session-persistence`

**Before**:
```go
func (a *ClaudeAdapter) buildArgs(cfg AdapterRunConfig) []string {
    args := []string{"-p"}
    args = append(args, "--model", model)
    if cfg.UseAgentFlag {
        args = append(args, "--agent", agentFilePath)
    } else {
        args = append(args, "--allowedTools", ...)
        args = append(args, "--disallowedTools", "TodoWrite")
    }
    args = append(args, "--output-format", "stream-json")
    args = append(args, "--verbose")
    args = append(args, "--dangerously-skip-permissions")
    args = append(args, "--no-session-persistence")
}
```

**After**:
```go
func (a *ClaudeAdapter) buildArgs(cfg AdapterRunConfig) []string {
    args := []string{"-p"}
    args = append(args, "--agent", agentFilePath)
    args = append(args, "--output-format", "stream-json")
    args = append(args, "--verbose")
    args = append(args, "--no-session-persistence")
    if cfg.Prompt != "" {
        args = append(args, cfg.Prompt)
    }
    return args
}
```

### Phase E: Test Migration (P1 — Parallel with implementation)

**Goal**: Update all tests to reflect the new unconditional agent mode.

**Tests to DELETE**:
- `TestNormalizeAllowedTools` (8 test cases) — function deleted
- `TestBuildArgsNormalizesAllowedTools` — tests removed code path
- `TestBuildArgsDisallowsTodoWrite` — TodoWrite is now in agent frontmatter, not CLI args

**Tests to REWRITE**:
- `TestSettingsJSONFormat` → `TestNoSettingsJSONWhenSandboxDisabled` — verify no settings.json created
- `TestSettingsJSONDenyRules` → `TestDenyRulesInAgentFrontmatter` — verify deny rules in agent .md
- `TestSettingsJSONPerPersona` → `TestAgentFilePerPersona` — verify agent file per persona config
- `TestContractPromptInClaudeMD` → `TestContractPromptInAgentFile` — verify contract section in agent .md

**Tests to UPDATE**:
- `TestPrepareWorkspaceAgentMode` — Remove `UseAgentFlag: true` from config (now default)
- `TestSettingsJSONSandboxSettings` — Verify sandbox-only settings.json format
- `TestPersonaToAgentMarkdown` / "scoped tools are normalized" subtest — Remove or change to verify passthrough behavior
- `TestCLAUDEMDRestrictionSection` — Verify restrictions appear in agent file, not CLAUDE.md

**Tests to ADD**:
- `TestBuildArgsAgentMode` — verify `--agent` present, no `--allowedTools`, `--dangerously-skip-permissions`, or `--model`
- `TestTodoWriteInjection` — verify TodoWrite added to disallowedTools in agent frontmatter
- `TestTodoWriteNoDuplication` — verify no duplicate when persona already denies TodoWrite
- `TestEmptyToolLists` — verify agent frontmatter omits `tools:` and `disallowedTools:` (edge case)
- `TestSandboxOnlySettingsJSON` — verify minimal settings.json with sandbox-only config

### Phase F: Documentation and ADR Update (P3)

**Files modified**:
- `docs/decisions/adr-agent-migration.md` — Update status from "PoC implemented" to "Complete". Remove "opt-in via UseAgentFlag" language.
- `docs/guides/adapter-development.md` — Remove references to `UseAgentFlag`, `normalizeAllowedTools`, and the legacy CLAUDE.md code path.

## Dependency Order

```
Phase A (types) → Phase B (prepareWorkspace) → Phase C (normalizeAllowedTools)
                                              ↘
                                               Phase D (buildArgs)
                                              ↘
                                               Phase E (tests — can run in parallel)
Phase F (docs — after all code changes)
```

Phases B, C, D can be done as a single commit since they form a logical unit. Phase E tests should be updated alongside each phase.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Agent frontmatter missing a field Claude Code requires | Low | Medium | Research spec #557 thoroughly documented the schema. PoC already validated. |
| Sandbox settings.json format change breaks bubblewrap | Low | High | Only the outer fields change; sandbox sub-object stays identical. |
| `wave agent export` produces different output | Low | Medium | Same `PersonaToAgentMarkdown` function — just removing normalization. Verify with test. |
| Tests referencing deleted types fail to compile | Certain | Low | Expected — update tests as part of Phase E. |
| `--dangerously-skip-permissions` removal breaks execution | Low | High | `permissionMode: dontAsk` in frontmatter is the documented replacement. Verified in PoC. |

## Complexity Tracking

No constitution violations found. All changes simplify existing code (net deletion).
