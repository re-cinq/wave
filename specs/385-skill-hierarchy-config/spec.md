# Feature Specification: Hierarchical Skill Configuration

**Feature Branch**: `385-skill-hierarchy-config`
**Created**: 2026-03-14
**Status**: Clarified
**Input**: [GitHub Issue #385](https://github.com/re-cinq/wave/issues/385) — Hierarchical skill configuration. Add `skills:` field at global (wave.yaml top-level), persona, and pipeline scope. Merge logic: pipeline > persona > global. Validate against `.wave/skills/` DirectoryStore.

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Global Default Skills (Priority: P1)

As a Wave user, I want to declare a set of default skills at the top level of `wave.yaml` so that every pipeline step automatically has access to commonly-used skills without repeating them in each pipeline definition.

**Why this priority**: This is the foundation of the hierarchy. Without global skill declarations, the other scopes have nothing to merge against. It also immediately reduces duplication for projects that use the same skill across many pipelines.

**Independent Test**: Can be fully tested by adding a `skills:` list to `wave.yaml` and verifying that the manifest loads successfully, the skills are parsed, and they are available to any pipeline step at resolution time.

**Acceptance Scenarios**:

1. **Given** a `wave.yaml` with a top-level `skills: ["speckit", "lint-rules"]`, **When** the manifest is loaded, **Then** the parsed `Manifest` struct contains the global skills list with both entries.
2. **Given** a `wave.yaml` with a top-level `skills: ["speckit"]` and a pipeline that declares no skills of its own, **When** skills are resolved for a step in that pipeline, **Then** the resolved skill set includes `"speckit"`.
3. **Given** a `wave.yaml` with no top-level `skills:` field, **When** the manifest is loaded, **Then** the global skills list defaults to an empty list and no error is raised.

---

### User Story 2 - Per-Persona Skill Declarations (Priority: P2)

As a Wave user, I want to attach skills to specific personas so that any step using that persona inherits its skills, merged with global defaults.

**Why this priority**: Personas represent agent roles — some roles naturally require specific skills (e.g., a `planner` persona always needs `speckit`). Persona-level skills reduce repetition across pipelines that share the same persona.

**Independent Test**: Can be tested by adding `skills:` to a persona definition in `wave.yaml`, loading the manifest, and verifying the persona's skills are parsed and included in the resolved set when that persona is used by a step.

**Acceptance Scenarios**:

1. **Given** a persona `planner` with `skills: ["speckit"]` and global `skills: ["common-tool"]`, **When** skills are resolved for a step using the `planner` persona, **Then** the resolved set is `["common-tool", "speckit"]` (merged, deduplicated).
2. **Given** a persona with `skills: ["speckit"]` and global `skills: ["speckit"]`, **When** skills are resolved, **Then** `"speckit"` appears only once (deduplicated).
3. **Given** a persona with no `skills:` field, **When** skills are resolved for a step using that persona, **Then** only global skills are included.

---

### User Story 3 - Per-Pipeline Skill Declarations (Priority: P2)

As a Wave user, I want to declare skills at the pipeline level so that all steps in a pipeline share the same skill set, merged with persona and global skills.

**Why this priority**: Tied with persona scope — pipelines often require specific skills regardless of which persona runs each step. This completes the three-tier hierarchy.

**Independent Test**: Can be tested by adding a `skills:` list to a pipeline YAML, loading the pipeline, and verifying that the pipeline's skills appear in the resolved set for any step, merged with persona and global skills.

**Acceptance Scenarios**:

1. **Given** a pipeline with `skills: ["audit-tool"]`, persona `navigator` with `skills: ["nav-skill"]`, and global `skills: ["common"]`, **When** skills are resolved for a step in that pipeline using `navigator`, **Then** the resolved set is `["audit-tool", "common", "nav-skill"]` (all three scopes merged, deduplicated, order is deterministic).
2. **Given** a pipeline with `skills: ["speckit"]` that overrides a global `skills: ["speckit"]`, **When** resolved, **Then** `"speckit"` appears once (deduplicated).
3. **Given** a pipeline with no `skills:` field, **When** resolved, **Then** only persona + global skills are included.

---

### User Story 4 - Validation Against Skill Store (Priority: P1)

As a Wave user, I want immediate, clear validation errors when I reference a skill name that does not exist in the `.wave/skills/` DirectoryStore so that I catch misconfigurations at load time (manifest loading for global/persona scopes, pipeline loading for pipeline scope) rather than at step execution time.

**Why this priority**: Validation prevents silent failures. A typo in a skill name should fail fast with a clear message, not silently skip skill provisioning.

**Independent Test**: Can be tested by adding an invalid skill name to any scope, loading the manifest, and verifying that a clear validation error is produced referencing the invalid name.

**Acceptance Scenarios**:

1. **Given** a global `skills: ["nonexistent-skill"]` and a `.wave/skills/` directory that does not contain `nonexistent-skill`, **When** the manifest is validated, **Then** a validation error is returned naming the invalid skill and the scope where it was declared.
2. **Given** a persona with `skills: ["valid-skill"]` and `valid-skill` exists in `.wave/skills/`, **When** the manifest is validated, **Then** no error is raised for that skill.
3. **Given** multiple invalid skill references across different scopes, **When** validated, **Then** all invalid references are reported in a single error (not just the first one).

---

### User Story 5 - Merge Precedence Override (Priority: P3)

As a Wave user, I want pipeline-scope skills to take precedence over persona-scope, and persona-scope to take precedence over global, so that more specific configurations can override less specific ones.

**Why this priority**: Precedence matters for future extensibility when skills may carry configuration beyond just names (e.g., skill-specific settings). For now, since skills are string references with deduplication, precedence primarily determines the canonical source of a skill in the resolved set.

**Independent Test**: Can be tested by verifying that the resolution function applies pipeline > persona > global ordering, and that the resolved set is deterministic regardless of input order.

**Acceptance Scenarios**:

1. **Given** identical skill names at all three scopes, **When** resolved, **Then** the skill appears exactly once in the result (deduplication).
2. **Given** skills declared at all three scopes, **When** resolved, **Then** the resolution function is called with (global, persona, pipeline) arguments and returns a merged, deduplicated, sorted list.

---

### Edge Cases

- What happens when a skill name contains invalid characters (e.g., spaces, uppercase, special chars)? The existing `ValidateName` function in the skill package enforces lowercase, hyphenated, max 64 chars — this validation MUST be applied to all three scopes.
- What happens when the `.wave/skills/` directory does not exist? Validation should treat this as all skill references being invalid and produce an error for each referenced skill.
- What happens when global `skills:` is set to `null` or an empty array in YAML? Both should be treated as "no global skills" with no error.
- What happens when the same skill is referenced at all three scopes? It should appear exactly once in the resolved set.
- What happens when a pipeline `requires.skills` (the existing `SkillConfig` map) and the new pipeline-level `skills:` list both reference the same skill? The `requires.skills` map provides install/check/init commands while the new `skills:` list is a name-only reference. These are complementary, not conflicting — the resolved skill set identifies WHICH skills to provision, and `requires.skills` provides HOW to install them. Both sources should be unified during resolution. Specifically: skill names from `requires.skills` keys are included in the resolved set alongside names from the new `skills:` lists. The `SkillConfig` metadata is used for preflight checks and installation, while DirectoryStore content is used for SKILL.md provisioning into the workspace.
- What happens when a skill exists in `.wave/skills/` but has no `SKILL.md`? The DirectoryStore's `List()` method already handles this — it only returns directories containing a valid `SKILL.md`. A name-only reference to such a directory should produce a validation error.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The manifest (`wave.yaml`) MUST support a top-level `skills:` field that accepts a list of skill name strings.
- **FR-002**: Each persona definition in `wave.yaml` MUST support a `skills:` field that accepts a list of skill name strings.
- **FR-003**: Each pipeline definition (`.wave/pipelines/*.yaml`) MUST support a top-level `skills:` field (a `[]string` on the `Pipeline` struct, separate from the existing `Requires.Skills` map) that accepts a list of skill name strings.
- **FR-004**: The system MUST provide a skill resolution function that accepts three skill lists (global, persona, pipeline) and returns a single merged, deduplicated, deterministically-ordered list.
- **FR-005**: Merge precedence MUST be pipeline > persona > global (more specific overrides less specific).
- **FR-006**: Skill name validation (lowercase, hyphenated, max 64 characters) MUST be applied to skill references at all three scopes during parsing (manifest parsing for global/persona scopes, pipeline parsing for pipeline scope).
- **FR-007**: At load time, every skill name referenced in any scope MUST be validated against the `.wave/skills/` DirectoryStore. For global and persona scopes, this happens during manifest validation. For pipeline scope, this happens during pipeline loading (since pipelines are loaded from separate `.wave/pipelines/*.yaml` files, not from `wave.yaml`). Invalid references MUST produce a clear error message identifying the invalid name and the scope.
- **FR-008**: The existing `requires.skills` map (pipeline-level `SkillConfig` entries with install/check/init) MUST remain backward-compatible and continue to function unchanged.
- **FR-009**: When both `requires.skills` (keyed SkillConfig map) and the new `skills:` list reference the same skill name in a pipeline, the system MUST unify them — the name appears in the resolved set, and the SkillConfig provides installation metadata.
- **FR-010**: YAML parsing MUST handle absent, null, and empty `skills:` fields at all three scopes without error, defaulting to an empty list.
- **FR-011**: Validation errors for invalid skill references MUST be aggregated — all invalid names across all scopes MUST be reported in a single validation pass, not fail-fast on the first error.
- **FR-012**: Skill resolution (merging global + persona + pipeline into a single set) MUST happen per-step at execution time in the executor, since each step may use a different persona. Name format validation and DirectoryStore existence checks happen earlier at load time (FR-006, FR-007).
- **FR-013**: Name-only skill references (from the new `skills:` lists) that lack a corresponding `SkillConfig` entry in `requires.skills` are provisioned via the DirectoryStore — the executor reads the skill from the store and provisions its SKILL.md content and resource files into the workspace. Skills that have BOTH a name-only reference AND a `SkillConfig` entry use the `SkillConfig` for install/check/init and the DirectoryStore for content provisioning.
- **FR-014**: Per-step skill declarations are explicitly out of scope for this feature. Steps inherit the resolved skill set from the pipeline + persona + global merge. Per-step overrides may be added in a future iteration if needed.

### Key Entities

- **Skill Reference**: A string name (e.g., `"speckit"`) declared at a scope level, validated against the DirectoryStore. Attributes: name (string, validated), scope (global/persona/pipeline).
- **Resolved Skill Set**: The final merged, deduplicated, sorted list of skill names for a given step execution context. Derived from the three-tier merge of global + persona + pipeline skill references.
- **DirectoryStore**: The existing filesystem-backed skill registry at `.wave/skills/` that provides the source of truth for valid skill names. Already supports multi-source precedence lookup.
- **SkillConfig**: The existing per-skill installation metadata (install/check/init/commands_glob). Remains a pipeline-level `requires.skills` map entry — unchanged by this feature.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: All three skill scope levels (global, persona, pipeline) are parseable from YAML and round-trip correctly through marshal/unmarshal.
- **SC-002**: The skill resolution function produces identical output for identical inputs regardless of call order (deterministic).
- **SC-003**: A manifest or pipeline file referencing a nonexistent skill name produces a validation error during the loading phase, before any pipeline step execution begins.
- **SC-004**: Existing pipelines that use only `requires.skills` (without the new `skills:` fields) continue to function identically with no changes required (backward compatibility).
- **SC-005**: The resolved skill set for a step using all three scopes contains no duplicates and is sorted alphabetically.
- **SC-006**: All skill name validation rules (format, length, character restrictions) are enforced identically across all three scopes.

## Clarifications _(resolved)_

### C1: Pipeline-level `skills:` field placement — top-level `Pipeline` struct vs nested under `Requires`

**Question**: Should the new pipeline-level `skills: []string` be a direct field on the `Pipeline` struct or nested under the existing `Requires` struct?

**Resolution**: Top-level `Pipeline` struct field (`Skills []string \`yaml:"skills,omitempty"\``). **Rationale**: The `Requires` struct holds operational metadata (install/check/init commands) for the preflight system. The new `skills:` list is a declarative intent field — it says WHAT skills to provision, not HOW. Keeping it at the top level mirrors the global (`Manifest.Skills`) and persona (`Persona.Skills`) placements for structural consistency across all three scopes. The existing `Requires.Skills` map remains unchanged (FR-008).

### C2: How name-only skill references are provisioned at execution time

**Question**: When the resolved skill set contains a name-only reference (from the new `skills:` lists) without a corresponding `SkillConfig` in `requires.skills`, how is that skill provisioned into the workspace?

**Resolution**: Via DirectoryStore lookup — the executor reads the skill's SKILL.md and resource files from `.wave/skills/<name>/` and provisions them into the workspace. **Rationale**: The DirectoryStore already provides `Read(name)` which returns the full `Skill` struct including `Body`, `ResourcePaths`, and `AllowedTools`. The existing `Provisioner` handles command file discovery from `SkillConfig.CommandsGlob`. For name-only references, the DirectoryStore content IS the provisioning payload. This is encoded as FR-013.

### C3: Validation scope — manifest and pipeline files

**Question**: FR-007 originally said "at manifest load time" but pipeline-level skills live in separate `.wave/pipelines/*.yaml` files. When does validation happen for pipeline-scope skills?

**Resolution**: Validation happens at each file's load time — global and persona skills are validated during `manifest.ValidateWithFile()`, pipeline skills are validated during pipeline loading. **Rationale**: The existing codebase validates manifests and pipelines in separate load paths (`internal/manifest/parser.go` for manifests, pipeline loading in the executor). Adding DirectoryStore validation at each load point is consistent with the existing architecture. FR-007 has been updated to reflect both validation points.

### C4: Resolution timing — load time vs step execution time

**Question**: When does the three-tier skill merge (global + persona + pipeline → resolved set) happen?

**Resolution**: Per-step at execution time in the executor. **Rationale**: Each step uses a specific persona, so the persona's skills vary per step within a pipeline. Validation of individual names (format + DirectoryStore existence) happens early at load time, but the actual merge into a resolved set must happen per-step when the executor knows which persona the step uses. This is encoded as FR-012.

### C5: Per-step skill declarations

**Question**: Should individual pipeline steps be able to declare their own `skills:` list in addition to the three existing scopes?

**Resolution**: Explicitly out of scope. **Rationale**: The issue (#385) specifies three scopes: global, persona, pipeline. Adding a fourth (per-step) scope increases complexity without demonstrated need — steps already inherit from pipeline + persona. If needed, this can be added as a backward-compatible extension later. This is encoded as FR-014.
