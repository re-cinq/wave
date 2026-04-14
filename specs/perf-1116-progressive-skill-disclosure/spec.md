# perf(skills): implement progressive disclosure – gh-cli injects 43K on every run

**Issue:** https://github.com/re-cinq/wave/issues/1116
**Author:** nextlevelshit
**State:** OPEN

## Problem

Every pipeline that declares a skill injects the **full SKILL.md body** into the LLM context at provisioning time, regardless of whether the skill content is actually needed for the current step.

`gh-cli` is the worst offender:

| Skill | File Size | Est. Tokens | Pipelines |
|-------|-----------|-------------|-----------|
| `gh-cli` | 43,077 bytes | ~10,700 | 11 pipelines |
| `spec-driven-development` | 20,811 bytes | ~5,200 | - |
| `agentic-coding` | 19,715 bytes | ~4,900 | - |
| `wave` | 18,944 bytes | ~4,700 | - |
| `software-architecture` | 17,780 bytes | ~4,400 | - |

All 13 skills combined: **~188K bytes / ~47K tokens**.

A pipeline declaring `gh-cli` + `golang` + `software-design` injects **~16K tokens** of skill content on every step — before the step prompt even starts. For a step that only runs `gh pr create`, 99% of the `gh-cli` content (auth flows, release management, gist commands, etc.) is wasted context.

**11 pipelines currently declare `gh-cli`:**
`plan-research`, `ops-pr-review-core`, `ops-refresh`, `ops-issue-quality`, `impl-issue`, `impl-issue-core`, `plan-scope`, `ops-rewrite`, `impl-review-loop`, `ops-pr-fix-review`, `ops-pr-review`

## Solution: 3-Level Progressive Disclosure

The Agent Skills specification defines three loading levels:

| Level | Content | Tokens | When Loaded |
|-------|---------|--------|-------------|
| 1 — Metadata | `name` + `description` frontmatter | ~100 | Always (startup) |
| 2 — Instructions | Full SKILL.md body | <5,000 | When skill is triggered |
| 3 — References | `references/`, `scripts/`, `assets/` | varies | On-demand only |

**Current Wave behavior:** Level 2 always. Level 3 never distinguished.
**Target behavior:** Level 1 always. Level 2 only when the step/agent actually invokes the skill. Level 3 on explicit reference.

## What Already Exists in the Codebase

- `ParseMetadata()` in `internal/skill/parse.go:173` — reads only frontmatter, returns `Skill` with empty `Body`
- `Parse()` in `internal/skill/parse.go:162` — full parse including body
- `ProvisionFromStore()` in `internal/skill/provision.go:25` — reads full skill via `store.Read()`, writes `s.Body` to `.wave/skills/<name>/SKILL.md` in workspace, copies resources
- `DirectoryStore.Read()` in `internal/skill/store.go:117` — calls `Parse()` (full body)
- `DirectoryStore.List()` in `internal/skill/store.go:201` — calls `ParseMetadata()` (metadata only)
- `ResolveSkills()` in `internal/skill/resolve.go` — merges skill names from global/persona/pipeline scopes
- `buildSkillSection()` in `internal/adapter/claude.go:769` — generates CLAUDE.md section listing skills with name, description, and path to SKILL.md
- `AdapterRunConfig.ResolvedSkills []SkillRef` in `internal/adapter/adapter.go:66` — carries `Name` + `Description` to adapter
- Executor integration at `internal/pipeline/executor.go:3194-3240` — resolves skills, provisions, converts to `[]SkillRef`

## Key Observation

The adapter injection path (`buildSkillSection`) already only injects metadata (name + description + path) into the CLAUDE.md system prompt. The full body goes into `.wave/skills/<name>/SKILL.md` on disk where the agent *can* read it via the `Read` tool. The problem is that `ProvisionFromStore()` unconditionally writes the full body to disk, and the `buildSkillSection` tells the agent about every skill file — meaning agents may preemptively read large skill files.

The fix is twofold:
1. For Level 1: write only a metadata stub to `.wave/skills/<name>/SKILL.md` (name + description + instruction to request full content if needed)
2. For Level 2: keep current behavior (full body written to disk)
3. Let step-level configuration or skill annotations control which level to use

## Acceptance Criteria

1. **Token Savings**: Level 1 metadata-only loads ~100 tokens per skill vs ~10K+ for `gh-cli`
2. **Functional Preservation**: All skill functionality remains accessible; agents can request full content when needed
3. **Backwards Compatibility**: Existing pipelines work without configuration changes (default to current behavior initially, with opt-in metadata-only)
4. **`gh-cli` Split**: `gh-cli` SKILL.md split into core (~500 tokens) + `references/full-reference.md` (on-demand)
5. **Body Limit Warning**: `wave skills publish` warns when body exceeds 500 lines per agentskills.io spec

## Immediate Mitigation

Split `gh-cli` into:
- `gh-cli/SKILL.md` — core patterns only (~500 tokens: pr create, issue create, common flags)
- `gh-cli/references/full-reference.md` — complete command reference (loaded on-demand)

This alone cuts 11 pipelines x ~10,000 tokens per step.
