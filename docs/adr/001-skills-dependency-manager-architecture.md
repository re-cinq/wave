# ADR-001: Skills and Dependency Manager Architecture

- **Status:** Proposed
- **Date:** 2026-02-28
- **Decision:** Adopt **Option 1: Minimal Enhancement of the Current System** — extend `SkillConfig` with `version` and `update` fields, add `wave skills update` CLI command. No registry, no transitive dependencies, no remote fetching.

## Options Evaluated

| # | Option | Effort | Risk | Verdict |
|---|--------|--------|------|---------|
| 1 | Minimal Enhancement (recommended) | Small | Low | Best fit for current adoption (1 skill) and prototype phase |
| 2 | Embedded Skill Registry + Lockfile | Medium | Medium | Good upgrade path when demand materializes |
| 3 | Remote Skill Registry | Large | High | Violates single-binary constraint |
| 4 | Plugin Architecture / Bundles | Medium | Medium | Adds ceremony without solving core gaps |
| 5 | Nix-Native Dependency Mgmt | Medium | High | Hard platform dependency, low portability |

## Rationale

Only one skill (`speckit`) exists today. The current `SkillConfig` extension point is the natural place to add version awareness. The prototype-phase no-backward-compatibility policy means graduating to Option 2 later requires no migration shims.

## Implementation Scope

Implementation touches 7 areas:

1. `SkillConfig` type extension
2. Preflight version validation
3. `wave skills update` command
4. `wave list skills` updates
5. `wave.yaml` schema
6. Recovery hints
7. Tests
