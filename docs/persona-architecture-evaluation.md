# Persona Architecture Evaluation: Claude Code Teams vs Wave

**Issue**: [#131](https://github.com/re-cinq/wave/issues/131)
**Date**: 2026-02-22
**Status**: Implemented

## Executive Summary

This document evaluates Wave's persona architecture against Claude Code's team-based multi-agent system, identifies consolidation opportunities, and proposes actionable improvements. The analysis covers role specialization, coordination patterns, permission models, and pipeline orchestration.

**Key finding**: Wave's 17 personas can be consolidated to ~15 by merging overlapping roles (craftsman+implementer, auditor+reviewer) while preserving specialization where it matters. Persona prompts benefit from anti-pattern guidance and cross-persona awareness.

---

## 1. Claude Code Teams Architecture Summary

Source: [Claude Code Teams Architecture Gist](https://gist.github.com/kieranklaassen/d2b35569be2c7f1412c64861a219d51f) (analysis of Claude Code v2.1.19 binary)

### Coordination Patterns

| Pattern | Description | Wave Equivalent |
|---------|-------------|-----------------|
| **Leader-Worker** | Single orchestrator spawns and manages specialists | No direct equivalent (Wave uses manifest-defined pipelines) |
| **Swarm** | Workers self-assign from shared task queue | Not compatible (violates fresh-memory constraint) |
| **Pipeline** | Sequential agents with blocking dependencies | Direct match — Wave's primary model |
| **Council** | Multiple agents propose solutions; leader selects best | Achievable via parallel steps + synthesizer merge |
| **Watchdog** | Safety-check agents monitor critical operations | Achievable via parallel read-only verification steps |

### Role Definition

Claude Code defines roles through:
- **Environment variables**: `CLAUDE_CODE_TEAM_NAME`, `CLAUDE_CODE_AGENT_ID`, `CLAUDE_CODE_AGENT_TYPE`
- **Spawn-time prompts**: Natural language specialization at agent creation
- **Plan approval gates**: `approvePlan`/`rejectPlan` for execution control

### Communication Model

- **File-based**: `~/.claude/teams/{team-name}/` with config.json, messages/, tasks/
- **Point-to-point**: `write()` sends to specific teammate
- **Broadcast**: `broadcast()` sends to all team members
- **13 team operations**: spawn, discover, join, approve/reject, write, broadcast, shutdown, plan approval, cleanup

### Permission Model

- **Plan approval workflow**: Agents submit plans before acting on sensitive operations
- **Negotiated shutdown**: Agents can refuse shutdown (with timeout override)
- **Team membership gating**: Join requests require leader approval
- No per-tool permission enforcement (unlike Wave's allow/deny patterns)

---

## 2. Wave Persona Architecture Summary

### Current State: 17 Personas

| Category | Personas | Pipeline Usage |
|----------|----------|----------------|
| **High-use** (12+ pipelines) | navigator, craftsman | Core workflow personas |
| **Medium-use** (4-9 pipelines) | auditor, planner, philosopher, summarizer, implementer | Specialized workflow personas |
| **Low-use** (1-3 pipelines) | reviewer, debugger, researcher, validator, synthesizer, provocateur, supervisor | Single-pipeline or niche personas |
| **GitHub-specific** | github-analyst, github-enhancer, github-commenter | GitHub integration trio |

### Permission Model

Wave enforces permissions at two levels:
1. **settings.json**: Claude Code sandbox enforcement with allowed_tools/deny patterns
2. **CLAUDE.md restrictions**: Prompt-level restriction section generated from manifest

Permission granularity ranges from read-only (planner, summarizer) to near-full access (craftsman, implementer).

### Coordination Model

- **Pipeline DAG**: Steps execute in dependency order
- **Artifact injection**: Inter-step communication via files (JSON, markdown)
- **Fresh memory**: No chat history inheritance between steps
- **Contract validation**: Outputs validated against schemas at step boundaries
- **Relay compaction**: Token-based summarization via summarizer persona

---

## 3. Pattern Comparison

### What Wave Already Does Well

1. **Contract validation at handovers** — Claude Code Teams has no equivalent output validation. Wave's JSON schema contracts, test suite validation, and retry mechanisms are more rigorous.

2. **Fine-grained permission enforcement** — Wave's per-tool allow/deny patterns are more granular than Claude Code's team-level plan approval.

3. **Fresh memory guarantees** — Wave's constitutional enforcement of fresh context at step boundaries prevents context drift and hallucination accumulation.

4. **Ephemeral workspace isolation** — Bubblewrap sandboxing + worktree isolation provides stronger security than Claude Code's shared filesystem model.

### What Wave Can Adopt

1. **Watchdog pattern** — Can be implemented as a parallel read-only step that validates critical operations. Already partially present in auditor/reviewer verify steps.

2. **Council pattern** — Can be implemented as parallel steps (e.g., multiple reviewers) with a synthesizer merge step. The `recinq` pipeline already uses a variant of this.

3. **Anti-pattern guidance in roles** — Claude Code Teams' failure handling table suggests documenting what NOT to do is as important as documenting responsibilities.

### What Is Incompatible

1. **Leader-Worker dynamic orchestration** — Violates fresh-memory constraint. Wave's manifest-defined pipelines cannot be dynamically modified at runtime.

2. **Swarm self-assignment** — Requires shared state between agents, incompatible with fresh context at step boundaries.

3. **Peer-to-peer communication** — Wave uses orchestrator-mediated artifact passing, not direct agent-to-agent messaging.

---

## 4. Persona Overlap Analysis

### High Overlap (Consolidation Recommended)

#### Craftsman + Implementer (90% overlap)

| Aspect | Craftsman | Implementer |
|--------|-----------|-------------|
| **Core role** | Code implementation and testing | Code execution and artifact generation |
| **Permissions** | Read, Write, Edit, Bash | Read, Write, Edit, Bash, Glob, Grep |
| **Pipeline usage** | 17 pipelines (23 steps) | 3 pipelines (7 steps) |
| **Differentiation** | Spec-driven, TDD emphasis | Execution-focused, artifact production |

**Recommendation**: Merge implementer into craftsman. The craftsman is already the primary implementation persona across most pipelines. The implementer adds only Glob/Grep permissions and a slightly different prompt focus, both of which can be absorbed.

#### Auditor + Reviewer (85% overlap)

| Aspect | Auditor | Reviewer |
|--------|---------|---------|
| **Core role** | Security and quality review | Quality review and validation |
| **Permissions** | Read, Grep, Bash(go vet/npm audit) | Read, Glob, Grep, Write(artifacts), Bash(tests) |
| **Pipeline usage** | 10 pipelines (11 steps) | 4 pipelines (4 steps) |
| **Differentiation** | OWASP focus, security-first | Correctness focus, can run tests |

**Recommendation**: Merge auditor into reviewer. The merged persona retains security audit capabilities while gaining the reviewer's broader quality assessment scope and test-running ability.

### Medium Overlap (Clarify Boundaries)

#### Planner + Philosopher (65% overlap)

| Aspect | Planner | Philosopher |
|--------|---------|-------------|
| **Core role** | Task breakdown and planning | Architecture design and specification |
| **Permissions** | Read, Glob, Grep (read-only) | Read, Write(.wave/specs/*) |
| **Pipeline usage** | 9 pipelines | 9 pipelines |
| **Differentiation** | Atomic tasks, dependencies, parallelization | User stories, data models, API design |

**Recommendation**: Keep separate with explicit scope boundaries. The planner focuses on HOW to break work into steps (task decomposition), while the philosopher focuses on WHAT to build (specification). These are complementary, not overlapping roles.

### Low Overlap (No Action)

- **Navigator vs Researcher**: Navigator explores local codebase; researcher searches the web. Complementary roles.
- **GitHub trio** (analyst, enhancer, commenter): Already well-separated by read/write/comment operations.
- **Niche personas** (debugger, provocateur, supervisor, validator, synthesizer): Each serves a specific pipeline with distinct responsibilities.

---

## 5. Actionable Proposals

### Proposal 1: Consolidate Craftsman + Implementer

**Rationale**: The implementer persona was created for gh-implement and speckit-flow pipelines but is functionally a subset of craftsman. Both write code, follow patterns, and produce artifacts. Maintaining two similar implementation personas creates confusion for pipeline authors.

**Changes**:
- Merge implementer's capabilities into craftsman prompt
- Add Glob and Grep to craftsman's permissions (from implementer)
- Update `speckit-flow.yaml` and `gh-implement.yaml` to use `craftsman`
- Remove `implementer.md` persona file
- Remove `implementer` entry from `wave.yaml`

**Impact**: 3 pipeline files updated, 1 persona removed, no test regressions expected (persona name is dynamically resolved).

### Proposal 2: Consolidate Auditor + Reviewer

**Rationale**: Both personas review code without modifying it, both produce severity-rated findings, and both cite file paths and line numbers. The auditor's security focus and the reviewer's quality focus are complementary aspects of a single review role.

**Changes**:
- Merge auditor's security review capabilities into reviewer prompt
- Add auditor's Bash permissions (go vet, npm audit) to reviewer
- Update 7 pipeline files that reference `auditor` to use `reviewer`
- Remove `auditor.md` persona file
- Remove `auditor` entry from `wave.yaml`

**Impact**: 7 pipeline files updated, 1 persona removed. The `gh-pr-review` pipeline's security-review and quality-review steps retain distinct prompts even though both use the unified reviewer persona.

### Proposal 3: Clarify Planner vs Philosopher Boundaries

**Rationale**: Rather than consolidating, these personas benefit from explicit scope documentation. The planner decomposes tasks; the philosopher designs systems. Making this distinction explicit prevents role confusion.

**Changes**:
- Add scope boundary documentation to both persona prompts
- Add "What I Don't Do" section to each
- Add cross-references between the two roles

### Proposal 4: Enhance Persona Prompts with Anti-Patterns

**Rationale**: Claude Code Teams documents failure modes and handling. Wave personas currently describe what TO do but not what NOT to do. Adding anti-patterns improves output quality.

**Changes**:
- Add anti-pattern sections to high-use personas (navigator, craftsman, reviewer, summarizer, debugger)
- Add output quality checklists
- Add cross-persona awareness (expected artifact inputs/outputs)

### Proposal 5: Enhance Base Protocol with Coordination Guidance

**Rationale**: The base protocol describes operational context but not inter-step communication conventions. Adding artifact I/O expectations helps personas produce well-structured handoffs.

**Changes**:
- Add artifact format conventions to base-protocol.md
- Add common output structure guidance
- Add quality expectations for handoff artifacts

---

## 6. Constitutional Compliance

All proposed changes preserve Wave's constitutional properties:

| Property | Status |
|----------|--------|
| Fresh memory at step boundaries | Preserved — persona changes do not affect memory isolation |
| Contract validation at handovers | Preserved — contract schemas are independent of persona names |
| Permission enforcement | Preserved — consolidated personas get the union of necessary permissions |
| Ephemeral workspace isolation | Preserved — workspace management is orthogonal to personas |
| Single binary deployment | Preserved — no new dependencies |
| Observable progress events | Preserved — event system is persona-agnostic |

---

## 7. Risk Assessment

| Risk | Mitigation |
|------|------------|
| Pipeline breakage from persona renames | Update all YAML references; run full test suite |
| Tests reference specific persona names | Search and update all test assertions |
| Consolidated personas lose specialization | Keep distinct step prompts even with unified persona; permission union preserves capability |
| Hardcoded persona constants in Go code | navigator, philosopher, summarizer are hardcoded; these are NOT being renamed |
