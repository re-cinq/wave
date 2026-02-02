# Epics & Story Breakdown: [FEATURE NAME]

**PRD**: [Link to prd.md]
**Architecture**: [Link to architecture.md]
**Created**: [DATE]
**Owner**: [Product Manager / Tech Lead]
**Status**: Draft | Refined | Sprint Ready

## Overview

### Feature Summary
[Brief description of the feature being broken down]

### Total Estimation
- **Story Points**: [Total]
- **Estimated Sprints**: [Count]
- **Team Capacity**: [Points per sprint]

## Epic 1: [Epic Title]

**Description**: [What this epic delivers]
**Business Value**: [Why this matters]
**Dependencies**: [Other epics or external dependencies]
**Priority**: P0 | P1 | P2

### Stories

#### Story 1.1: [Story Title]
**ID**: [PROJ-XXX]
**Priority**: P0/P1/P2
**Points**: [Estimate]
**Assignee**: [Name or TBD]

**User Story**:
As a [persona], I want to [action] so that [benefit].

**Acceptance Criteria**:
- [ ] Given [context], when [action], then [result]
- [ ] Given [context], when [action], then [result]
- [ ] Given [context], when [action], then [result]

**Technical Notes**:
- [Implementation hint or constraint]

**Dependencies**:
- Blocked by: [Story ID or None]
- Blocks: [Story ID or None]

---

#### Story 1.2: [Story Title]
**ID**: [PROJ-XXX]
**Priority**: P0/P1/P2
**Points**: [Estimate]
**Assignee**: [Name or TBD]

**User Story**:
As a [persona], I want to [action] so that [benefit].

**Acceptance Criteria**:
- [ ] Given [context], when [action], then [result]

**Technical Notes**:
- [Implementation hint]

**Dependencies**:
- Blocked by: [Story ID]
- Blocks: [None]

---

### Epic 1 Summary
| Story | Points | Priority | Blocked By | Status |
|-------|--------|----------|------------|--------|
| 1.1 | [X] | P0 | None | Pending |
| 1.2 | [X] | P1 | 1.1 | Pending |
| **Total** | **[X]** | | | |

---

## Epic 2: [Epic Title]

**Description**: [What this epic delivers]
**Business Value**: [Why this matters]
**Dependencies**: [Epic 1 or other]
**Priority**: P0 | P1 | P2

### Stories

#### Story 2.1: [Story Title]
[Same structure as Epic 1 stories]

---

### Epic 2 Summary
| Story | Points | Priority | Blocked By | Status |
|-------|--------|----------|------------|--------|
| 2.1 | [X] | P1 | 1.2 | Pending |
| **Total** | **[X]** | | | |

---

## Epic 3: [Epic Title]

[Continue same structure...]

---

## Dependency Graph

```
Epic 1
  └── Story 1.1 ──┬──> Story 1.2
                  │
Epic 2            │
  └── Story 2.1 <─┘──> Story 2.2

Epic 3 (parallel)
  └── Story 3.1 ──> Story 3.2
```

## Release Planning

### MVP (Must Have)
| Epic | Stories | Total Points |
|------|---------|--------------|
| Epic 1 | 1.1, 1.2 | [X] |
| Epic 2 | 2.1 | [X] |
| **Total** | | **[X]** |

### Phase 2 (Should Have)
| Epic | Stories | Total Points |
|------|---------|--------------|
| Epic 2 | 2.2, 2.3 | [X] |
| Epic 3 | 3.1, 3.2 | [X] |
| **Total** | | **[X]** |

### Future (Nice to Have)
| Epic | Stories | Total Points |
|------|---------|--------------|
| [Epic] | [Stories] | [X] |

## Sprint Allocation (Proposed)

### Sprint 1
- [ ] Story 1.1 - [X] points
- [ ] Story 3.1 - [X] points (parallel track)
- **Total**: [X] points

### Sprint 2
- [ ] Story 1.2 - [X] points
- [ ] Story 2.1 - [X] points
- [ ] Story 3.2 - [X] points
- **Total**: [X] points

### Sprint 3
- [ ] Story 2.2 - [X] points
- [ ] Story 2.3 - [X] points
- **Total**: [X] points

## Risks & Blockers

| Risk/Blocker | Impact | Mitigation | Owner |
|--------------|--------|------------|-------|
| [Risk] | [Stories affected] | [Action] | [Name] |

## Open Questions

- [ ] [Question about scope or priority]
- [ ] [Question about technical approach]

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | [Date] | [Name] | Initial breakdown |
