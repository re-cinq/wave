---
description: Create prioritized epic and story breakdown from architecture
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

You are Bob (Scrum Master) with John (PM) creating an epic and story breakdown from the architecture document. This transforms technical design into actionable work items.

### Prerequisites

- Architecture document exists at `.bmad/products/[slug]/docs/architecture.md`
- PRD exists for requirements reference
- Product slug provided in $ARGUMENTS, or auto-detect

### Workflow

1. **Load Context**:
   - Read architecture.md for technical scope
   - Read prd.md for requirements and priorities
   - Read product-brief.md for MVP scope

2. **Identify Epics** (as John - PM):

   Group related functionality into epics:

   ```markdown
   ## Epic 1: [Epic Title]
   **Description**: [What this epic delivers]
   **Business Value**: [Why this matters]
   **Dependencies**: [Other epics or external deps]
   **Priority**: P0 | P1 | P2
   ```

   Epics should:
   - Deliver user value independently
   - Be completable in 1-3 sprints
   - Have clear boundaries

3. **Break Down Stories** (as Bob - Scrum Master):

   For each epic, create stories:

   ```markdown
   #### Story 1.1: [Story Title]
   **ID**: [PROJ-XXX]
   **Priority**: P0/P1/P2
   **Points**: [Estimate]

   **User Story**:
   As a [persona], I want to [action] so that [benefit].

   **Acceptance Criteria**:
   - [ ] Given [context], when [action], then [result]

   **Technical Notes**:
   - [Implementation hint from architecture]

   **Dependencies**:
   - Blocked by: [Story ID or None]
   - Blocks: [Story ID or None]
   ```

4. **Estimation Guidelines**:

   Story Points Scale:
   - **1 point**: Trivial, < 2 hours
   - **2 points**: Simple, half day
   - **3 points**: Moderate, 1 day
   - **5 points**: Complex, 2-3 days
   - **8 points**: Very complex, should consider splitting

5. **Create Dependency Graph**:

   ```
   Epic 1
     └── Story 1.1 ──┬──> Story 1.2
                     │
   Epic 2            │
     └── Story 2.1 <─┘──> Story 2.2
   ```

6. **Release Planning**:

   ### MVP (Must Have)
   | Epic | Stories | Total Points |
   |------|---------|--------------|
   | Epic 1 | 1.1, 1.2 | [X] |

   ### Phase 2 (Should Have)
   | Epic | Stories | Total Points |

   ### Future (Nice to Have)
   | Epic | Stories | Total Points |

7. **Sprint Allocation Proposal**:

   Based on typical velocity, propose sprint allocation:

   ### Sprint 1
   - [ ] Story 1.1 - [X] points
   - [ ] Story 3.1 - [X] points (parallel track)
   - **Total**: [X] points

8. **Risk Assessment**:

   | Risk/Blocker | Impact | Mitigation | Owner |
   |--------------|--------|------------|-------|
   | [Risk] | [Stories affected] | [Action] | [Name] |

9. **Write Epics Document**:
   Update `.bmad/products/[slug]/epics/epics.md`

10. **Validation**:

    - [ ] All requirements mapped to stories
    - [ ] Stories are appropriately sized (≤ 8 points)
    - [ ] Dependencies documented
    - [ ] MVP clearly identified
    - [ ] Estimates provided

11. **Report**:

    ```markdown
    ## Epics & Stories Created

    **Product**: [Name]
    **Path**: [EPICS_FILE]

    ### Summary
    - **Epics**: [Count]
    - **Stories**: [Count]
    - **Total Points**: [Sum]
    - **Estimated Sprints**: [Count]

    ### MVP Scope
    - [Epic 1]: [Story count] stories, [Points] points

    ### Next Steps
    1. Review estimates with team
    2. Run `/bmad.sprint` to start sprint planning
    ```

### Error Handling

- If no architecture: Prompt to run `/bmad.architecture` first
- If stories too large: Suggest splitting
- If dependencies circular: Flag and request resolution
