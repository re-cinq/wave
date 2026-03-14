# Research: Hierarchical Skill Configuration

**Feature**: #385 — Skill Hierarchy Config
**Date**: 2026-03-14

## Research Questions

### RQ-1: Where should the `skills:` field be added in each scope?

**Decision**: Direct field on the struct at each scope level.

**Rationale**: The spec (C1) already resolved this — `Skills []string` as a top-level field
on each struct (`Manifest`, `Persona`, `Pipeline`) for structural consistency. This mirrors
how `Permissions` and `Sandbox` are sibling fields rather than nested.

**Alternatives Rejected**:
- Nested under `Requires` (pipeline scope): The `Requires` struct holds operational metadata
  (install/check/init). The new `skills:` list is declarative intent. Mixing concerns would
  create confusion about what `Requires.Skills` means vs `Skills`.

### RQ-2: How should skill resolution be implemented?

**Decision**: Pure function `ResolveSkills(global, persona, pipeline []string) []string` in a
new file `internal/skill/resolve.go`.

**Rationale**:
- Placed in `internal/skill/` because it operates on skill domain concepts (merge, dedup, sort).
- Pure function with no side effects — easy to test, easy to call from the executor.
- Returns a merged, deduplicated, sorted `[]string`.
- The executor calls this per-step in `buildAdapterRunConfig()` since each step may use a
  different persona.

**Alternatives Rejected**:
- Method on `Pipeline` struct: Would require passing persona skills as argument anyway.
  A free function is simpler.
- Method on `Manifest`: Same issue — manifest doesn't know which persona a step uses.
- Resolve at pipeline load time: Can't — persona varies per step (FR-012).

### RQ-3: When and how should validation happen?

**Decision**: Two validation points:

1. **Manifest load time** (`internal/manifest/parser.go`): Validate global and persona skill
   names via `skill.ValidateName()` and DirectoryStore existence check.
2. **Pipeline load time** (`internal/pipeline/dag.go`): Validate pipeline-level skill names
   via the same functions.

**Rationale**: Matches the existing validation architecture — manifest validation in
`ValidateWithFile()`, pipeline validation in `ValidateDAG()`. Both aggregate errors
(FR-011) rather than fail-fast.

**Alternatives Rejected**:
- Single validation point at execution time: Too late — fails only when pipeline runs,
  not at load/validate time. Spec explicitly requires fail-fast (FR-007).
- Validation in the resolver function: Mixes resolution with validation; resolver should
  assume inputs are already validated.

### RQ-4: How to handle DirectoryStore availability for validation?

**Decision**: Validation functions accept a `skill.Store` interface parameter. At manifest
load time, the caller provides the DirectoryStore. When the store is `nil` (e.g., unit
tests without filesystem), name format validation still runs but existence checks are skipped.

**Rationale**: The `manifest.ValidateWithFile()` currently only takes `basePath` and
`filePath`. Adding a store parameter requires a signature change, but this is acceptable
during prototype phase (no backward compatibility constraint per constitution).

**Alternatives Rejected**:
- Global store singleton: Violates testability and dependency injection patterns.
- Defer all existence checks to preflight: Loses the fail-fast property at load time.

### RQ-5: How to unify `requires.skills` (SkillConfig) with new `skills:` list?

**Decision**: During resolution, include both sources:
- Names from `Pipeline.Skills` (new field)
- Names from `Pipeline.Requires.Skills` map keys (existing field)
Both are merged into the resolved skill set. The `SkillConfig` metadata continues to
drive preflight install/check/init. The DirectoryStore content drives SKILL.md provisioning.

**Rationale**: FR-009 requires unification. The two sources serve complementary purposes:
`SkillConfig` = HOW to install, `skills:` list + DirectoryStore = WHAT to provision.

**Implementation**: The resolution function accepts an optional `[]string` from
`requires.skills` keys, merged alongside the three-tier hierarchy sources.

### RQ-6: How should DirectoryStore content provisioning work for name-only references?

**Decision**: Extend the existing provisioning path in `executor.go` (lines 1192-1213).
After resolving the skill set per-step, for each name-only reference (not in
`requires.skills`), call `DirectoryStore.Read(name)` to get the `Skill` struct, then
write `SKILL.md` content and resource files into the workspace.

**Rationale**: The executor already provisions `SkillConfig`-backed skills via the
`Provisioner`. Name-only references use the DirectoryStore's existing `Read()` method.
This keeps provisioning logic centralized in the executor.

**Alternatives Rejected**:
- New `StoreProvisioner` type: Over-engineering for what's a few lines of code.
- Provision at pipeline load time: Can't — workspace doesn't exist yet.
