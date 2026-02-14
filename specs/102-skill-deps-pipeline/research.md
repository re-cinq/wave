# Research: Skill Dependency Installation in Pipeline Steps

**Branch**: `102-skill-deps-pipeline` | **Date**: 2026-02-14

## Phase 0 — Outline & Research

### Unknowns Extracted from Spec

1. **Integration Point: Preflight → Executor** — How to wire the enhanced preflight (with event emission) into the existing `Execute()` flow
2. **StatePreflight Constant** — Where to add and how to integrate with existing event state constants
3. **Provisioner repoRoot Resolution** — The executor currently passes `""` as repoRoot to `skill.NewProvisioner` (executor.go:512), which means glob resolution will fail for command discovery
4. **Three Target Skills Configuration** — Concrete install/check/init commands for Speckit, BMAD, and OpenSpec

### Research Findings

---

#### R1: Preflight Event Emission Pattern

**Decision**: Add `StatePreflight` constant to `internal/event/emitter.go` and pass an `EventEmitter` to the preflight `Checker`.

**Rationale**: The executor currently emits preflight events with a string literal `"preflight"` (executor.go:173). FR-010 requires structured progress events per dependency. The preflight Checker needs access to the emitter to emit granular per-dependency events rather than only emitting after all checks complete.

**Alternatives Rejected**:
- *Return events from Checker.Run()*: Would require the caller to iterate and emit, adding coupling. The Checker knows when each check starts/succeeds/fails.
- *Log-based approach*: Doesn't satisfy the structured event requirement (FR-010).

**Implementation Approach**: Add an optional `EventEmitter` field to `Checker` (or a callback function). Add `StatePreflight = "preflight"` to event constants. The Checker emits one event per dependency check/install action.

---

#### R2: Provisioner repoRoot Bug Fix

**Decision**: The executor must pass the actual repository root path to `skill.NewProvisioner`, not an empty string.

**Rationale**: At executor.go:512, the code is `skill.NewProvisioner(execution.Manifest.Skills, "")`. The `Provisioner.Provision()` method resolves `commands_glob` relative to `repoRoot` (skill.go:52). An empty repoRoot means glob patterns resolve relative to the current working directory, which is unreliable during pipeline execution.

**Implementation Approach**: Derive the repo root from the manifest's location or from the project's working directory. The manifest loader already knows the path from `Load(path)` — propagate this to the executor context. The simplest approach: resolve the absolute path of the directory containing `wave.yaml` and use that as repoRoot.

---

#### R3: Two-Stage Skill Command Provisioning Chain

**Decision**: Keep the existing two-stage approach (provisioner → adapter) as-is. It is already implemented correctly in executor.go:509-528 and claude.go:278-283.

**Rationale**: The spec confirms this is the intended design (C1 clarification). The provisioner stages files into `.wave-skill-commands/.claude/commands/` within the workspace, and the adapter's `copySkillCommands()` copies them to the final `.claude/commands/` location.

**Alternatives Rejected**:
- *Direct provisioning to adapter directory*: Would couple the provisioner to adapter-specific paths, violating separation of concerns.
- *Symlink approach*: Would not work across filesystem boundaries and adds complexity for worktree workspaces.

---

#### R4: Manifest Validation for Skills Referenced by Pipelines

**Decision**: Add cross-validation between pipeline `requires.skills` and manifest `skills` map at pipeline load/validation time.

**Rationale**: FR-003 requires that all declared skills exist in the manifest's skills section before executing any step. Currently, the preflight Checker handles this at runtime (preflight.go:77-86), but the spec also calls for validation at load time for fail-fast behavior.

**Implementation Approach**: Two-layer validation:
1. **Runtime validation** (existing): `Checker.CheckSkills()` already rejects undeclared skills.
2. **Load-time validation** (new): When loading a pipeline, cross-reference `requires.skills` against the manifest's `skills` map. This is an enhancement to pipeline validation, not manifest validation (since pipelines reference the manifest).

---

#### R5: Target Skill Definitions (Speckit, BMAD, OpenSpec)

**Decision**: Define concrete skill configs for the three target skills in the Wave manifest.

**Rationale**: FR-011 requires support for at least Speckit, BMAD, and OpenSpec. These need `check`, `install` (optional), and `commands_glob` configurations.

**Concrete Definitions** (based on codebase research):

```yaml
skills:
  speckit:
    check: "test -d .specify"
    install: "npx -y @anthropic/speckit init"
    commands_glob: ".claude/commands/speckit.*.md"
  bmad:
    check: "test -f .claude/commands/bmad.*.md"
    commands_glob: ".claude/commands/bmad.*.md"
  openspec:
    check: "test -d .openspec"
    commands_glob: ".claude/commands/openspec.*.md"
```

**Note**: These definitions are illustrative. The actual install commands depend on the skill's distribution mechanism. The system's design is skill-agnostic — it only needs `check` and optionally `install`/`init`.

---

#### R6: Performance Constraint (SC-006: <500ms preflight overhead)

**Decision**: Sequential check execution with early-exit on failure. No parallelism needed.

**Rationale**: SC-006 requires <500ms overhead when all dependencies are pre-installed. Tool checks use `exec.LookPath()` (microseconds). Skill checks execute `sh -c <check_command>`. Typical check commands (`test -d`, `which`, `--version`) complete in <50ms. Even with 10 skills, sequential execution stays well under 500ms.

**Alternatives Rejected**:
- *Parallel skill checks*: Adds goroutine coordination complexity for marginal gain. Sequential checks of simple commands are fast enough.
- *Caching check results*: Unnecessary since preflight runs once per pipeline execution.

---

#### R7: Idempotent Installation (Edge Case: Concurrent Pipelines)

**Decision**: No locking mechanism. Rely on install command idempotency.

**Rationale**: The spec explicitly states (Edge Case 3) that no locking is required since install commands are expected to be idempotent. If two pipelines concurrently install the same skill, the second install is a no-op or succeeds redundantly.

**Implementation Approach**: Document the idempotency requirement in the `SkillConfig` type comments. The system trusts that install commands are safe to run concurrently.

---

### Codebase Integration Points

| Component | File | Current State | Required Changes |
|-----------|------|---------------|-----------------|
| `SkillConfig` | `internal/manifest/types.go:142-148` | Complete — has check, install, init, commands_glob | None |
| `Pipeline.Requires` | `internal/pipeline/types.go:14,20-23` | Complete — has Skills and Tools fields | None |
| `Checker` | `internal/preflight/preflight.go` | Complete — checks tools and skills | Add event emission support |
| `Provisioner` | `internal/skill/skill.go` | Complete — provisions commands | Fix repoRoot resolution in executor |
| `Executor` | `internal/pipeline/executor.go:158-181` | Partial — calls preflight but with string literal events | Add StatePreflight, fix repoRoot, enhance events |
| `Event states` | `internal/event/emitter.go:56-69` | Missing `StatePreflight` | Add constant |
| `Manifest skills` | `wave.yaml` | Missing `skills` section | Add speckit, bmad, openspec definitions |
| `Manifest validation` | `internal/manifest/parser.go:302-316` | Complete — validates skill check field | None |
| `Adapter integration` | `internal/adapter/claude.go:278-283` | Complete — copySkillCommands works | None |

### Summary

The feature's core infrastructure is **largely already implemented**. The main gaps are:
1. **Event emission integration** — Preflight Checker needs to emit per-dependency events
2. **StatePreflight constant** — Add to event package for consistency
3. **repoRoot propagation** — Fix the empty string bug in executor
4. **Manifest skill definitions** — Add the three target skills to wave.yaml
5. **Pipeline cross-validation** — Optional enhancement for fail-fast at load time
6. **Test coverage** — Extend existing tests to cover event emission and edge cases
