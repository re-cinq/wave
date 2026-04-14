# Implementation Plan: Progressive Skill Disclosure

## Objective

Reduce LLM context token waste by implementing 3-level progressive disclosure for Agent Skills. Primary target: `gh-cli` (2,281 lines / ~10,700 tokens) injected into 11 pipelines unconditionally. Overall savings: ~45K tokens per run for multi-skill pipelines.

## Approach

Two-pronged strategy:

**Prong 1 (Immediate — gh-cli split):** Restructure `gh-cli` SKILL.md into a lean core + reference file. This is a content-only change with zero code modifications, delivering the biggest single win.

**Prong 2 (Systematic — metadata-only provisioning):** Modify `ProvisionFromStore()` to support writing metadata stubs instead of full bodies. Add a `ReadMetadata()` method to `Store` interface. Update `buildSkillSection()` to differentiate Level 1 vs Level 2 skills. Add provisioning level control via manifest/pipeline config.

## File Mapping

### Files to Modify

| File | Change |
|------|--------|
| `skills/gh-cli/SKILL.md` | Reduce to core patterns (~100 lines) |
| `internal/skill/store.go` | Add `ReadMetadata(name string) (Skill, error)` to `Store` interface and `DirectoryStore` |
| `internal/skill/provision.go` | Add `ProvisionLevel` type; split `ProvisionFromStore` to support Level 1 (metadata stub) vs Level 2 (full body) |
| `internal/skill/parse.go` | No changes needed — `ParseMetadata()` already exists |
| `internal/adapter/claude.go` | Update `buildSkillSection()` to indicate when a skill is metadata-only vs fully loaded |
| `internal/adapter/adapter.go` | Add `Level` field to `SkillRef` struct |
| `internal/pipeline/executor.go` | Pass provisioning level when calling `ProvisionFromStore()` |
| `cmd/wave/commands/skills.go` | Add body-size warning to `publish`/`verify` subcommands |

### Files to Create

| File | Purpose |
|------|---------|
| `skills/gh-cli/references/full-reference.md` | Complete gh-cli command reference (moved from SKILL.md body) |
| `internal/skill/provision_test.go` additions | Tests for Level 1 provisioning |

### Files NOT Affected

- `internal/skill/resolve.go` — skill name resolution is orthogonal to loading level
- Pipeline YAML files — no config changes needed for initial implementation
- Other skill SKILL.md files — gh-cli is the priority; others can be optimized later

## Architecture Decisions

**AD-1: Metadata stub file instead of no file**
When provisioning at Level 1, write a stub SKILL.md containing frontmatter + a one-line instruction ("Use the Skill tool or read references/ for full content"). This ensures the `buildSkillSection` path reference remains valid and agents can discover the skill exists.

**AD-2: `ReadMetadata` on Store, not a flag on `Read`**
Separate method is cleaner than a boolean flag. `Read()` returns full skill. `ReadMetadata()` returns metadata-only. Matches the existing `Parse()` / `ParseMetadata()` split in parse.go.

**AD-3: Default to Level 2 for backwards compatibility**
No pipeline breaks. Teams opt into Level 1 by annotating skills in manifest or pipeline config. Future: make Level 1 the default once validated.

**AD-4: gh-cli split is the immediate win**
Restructuring gh-cli content requires no Go code changes. Can be validated independently. Delivers ~10K token savings across 11 pipelines immediately.

**AD-5: No trigger/callback mechanism for Level 2 promotion**
The issue suggests dynamically promoting Level 1 → Level 2 when a skill is "triggered." This is complex and fragile. Instead: let pipeline authors declare which skills need Level 2 per step. Keep it explicit and deterministic.

## Risks

**Risk 1: Agent reads metadata stub and gets confused**
- Likelihood: Low — stub explicitly says "request full content via Skill tool"
- Mitigation: Test with representative pipelines

**Risk 2: gh-cli split breaks pipelines expecting full content**
- Likelihood: Medium — agents currently rely on full SKILL.md being in context
- Mitigation: Keep core patterns (pr create, issue create, common flags) in SKILL.md; move only reference material to references/

**Risk 3: Store interface change breaks downstream**
- Likelihood: Low — `Store` interface is internal, only `DirectoryStore` implements it
- Mitigation: Add `ReadMetadata` as new method; don't change `Read` signature

## Testing Strategy

**Unit Tests:**
- `TestProvisionFromStore_Level1` — verify stub file written with metadata only
- `TestProvisionFromStore_Level2` — verify full body written (existing behavior)
- `TestReadMetadata` — verify metadata-only read from DirectoryStore
- `TestBuildSkillSection_WithLevel` — verify different rendering for L1 vs L2 skills

**Integration Tests:**
- `TestSkillLifecycle_FileAdapter` — extend existing test to cover Level 1 provisioning
- Verify gh-cli references/ files are copied correctly

**Validation:**
- Run `impl-issue` pipeline with modified gh-cli skill, verify no functional regression
- Compare token counts before/after (manual or via adapter stream events)
- Verify `wave skills verify` warns on oversized SKILL.md files
