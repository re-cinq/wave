# Evaluate and Improve Persona Architecture Based on Claude Code Teams

**Issue**: [#131](https://github.com/re-cinq/wave/issues/131)
**Feature Branch**: `131-persona-architecture`
**Labels**: enhancement, personas
**Status**: Draft
**Author**: nextlevelshit

## Context

Claude Code uses agent teams to handle different aspects of development work. Wave has a similar concept with personas that specialize in different roles. This issue tracks an evaluation of how we can learn from Claude Code's approach to improve our persona architecture.

**Reference**: [Claude Code Teams Architecture](https://gist.github.com/kieranklaassen/d2b35569be2c7f1412c64861a219d51f)

## Objective

Analyze Claude Code's team-based architecture and identify opportunities to enhance Wave's persona system, including what we can adopt, modify, or potentially remove.

## Current State

### Wave Personas (18 total in `.wave/personas/`)

| Persona | Role | Pipeline Usage |
|---------|------|----------------|
| `navigator` | Read-only codebase exploration | High (12+ pipelines) |
| `craftsman` | Code implementation and testing | High (12+ pipelines) |
| `auditor` | Security and quality review | Medium (5+ pipelines) |
| `planner` | Task breakdown and planning | Medium (5 pipelines) |
| `philosopher` | Architecture design and spec | Medium (4 pipelines) |
| `summarizer` | Context compaction for relay | Medium (5 pipelines) |
| `implementer` | Code execution for pipeline steps | Medium (gh-issue, speckit) |
| `reviewer` | Quality review and validation | Low (3 pipelines) |
| `debugger` | Issue diagnosis | Low (1 pipeline) |
| `researcher` | Web research | Low (1 pipeline) |
| `validator` | Skeptical findings verification | Low (1 pipeline) |
| `synthesizer` | Analysis synthesis into proposals | Low (1 pipeline) |
| `provocateur` | Creative challenger | Low (1 pipeline) |
| `supervisor` | Work quality evaluation | Low (1 pipeline) |
| `github-analyst` | GitHub issue analysis | Low (3 pipelines) |
| `github-enhancer` | GitHub issue improvement | Low (2 pipelines) |
| `github-commenter` | Post GitHub comments | Low (2 pipelines) |
| `base-protocol` | Shared base protocol (not a persona) | All (injected) |

### Claude Code Teams Architecture (Reference)

Key patterns from the gist:
- **Leader-Worker hierarchy**: Single orchestrator manages specialists
- **Swarm Pattern**: Workers self-assign from shared task queue
- **Pipeline Pattern**: Sequential agents with blocking dependencies (most similar to Wave)
- **Council Pattern**: Multiple agents propose solutions; leader selects best
- **Watchdog Pattern**: Safety-check agents monitor critical operations
- **File-based coordination**: Teams store data at `~/.claude/teams/{team-name}/`
- **13 team operations**: spawn, discover, join, approve/reject, write, broadcast, shutdown, plan approval, cleanup

## Tasks

- [ ] Review the Claude Code Teams architecture from the reference gist
- [ ] Document key insights and patterns from Claude Code's approach
- [ ] Analyze Wave's current persona system (located in `.wave/personas/`)
- [ ] Identify gaps or opportunities in Wave's persona architecture
- [ ] Propose specific improvements to persona definitions and interactions
- [ ] Evaluate whether any current personas should be consolidated or removed
- [ ] Consider how team coordination patterns could apply to Wave pipelines

## Acceptance Criteria

- [ ] Comprehensive comparison document created
- [ ] Specific recommendations for persona improvements documented
- [ ] At least 3 actionable proposals for enhancing the persona system
- [ ] Clear rationale provided for any proposed removals or consolidations

## Focus Areas

- Persona specialization and role clarity
- Inter-persona communication patterns
- Permission and capability modeling
- Pipeline orchestration improvements

## Analysis Areas

### 1. Persona Overlap and Consolidation Candidates

Several personas have overlapping responsibilities:
- **auditor** vs **reviewer**: Both do quality/security review; auditor is security-focused, reviewer is broader
- **craftsman** vs **implementer**: Both write code; craftsman is spec-driven, implementer is pipeline-step-focused
- **navigator** vs **researcher**: Both explore/gather information; navigator is codebase-only, researcher is web-enabled
- **planner** vs **philosopher**: Both do design work; planner focuses on task breakdown, philosopher on architecture

### 2. Coordination Patterns

Wave currently uses:
- **Pipeline Pattern**: Sequential steps with dependencies (DAG-based)
- **Artifact injection**: Inter-step communication via files
- **Fresh memory**: No chat history inheritance between steps

Claude Code adds:
- **Leader-Worker**: Dynamic orchestration
- **Swarm**: Self-assigned work queues
- **Council**: Multi-agent proposal evaluation
- **Watchdog**: Monitoring and rollback

### 3. Permission Model

Wave personas have fine-grained tool permissions (allow/deny patterns) projected into both `settings.json` and `CLAUDE.md` restriction sections. This is more granular than Claude Code Teams which relies on role-based capabilities.

### 4. Architecture Alignment

Wave's persona system is designed around the constitutional constraint of "fresh memory at step boundaries" and "contract validation at handovers." Any improvements must preserve these properties.
