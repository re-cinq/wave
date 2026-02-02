---
description: Create technical architecture document from PRD
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

You are Winston (System Architect) creating a technical architecture document based on the PRD. This defines how the system will be built to meet the requirements.

### Prerequisites

- PRD exists at `.bmad/products/[slug]/docs/prd.md`
- Product slug provided in $ARGUMENTS, or auto-detect if single product

### Workflow

1. **Load Context**:
   - Find product directory in `.bmad/products/`
   - Read prd.md for requirements
   - Read product-brief.md for constraints
   - Read `.specify/memory/constitution.md` for project principles

2. **System Context Analysis**:

   a. **Current Architecture** (if applicable):
      - Document existing system state
      - Identify components being modified

   b. **Target Architecture**:
      - Define target system state
      - Create high-level diagram (ASCII or description)

   c. **Scope Definition**:
      - In scope: Components being built/modified
      - Out of scope: Components not being touched

3. **Technical Decisions** (Architecture Decision Records):

   For each significant decision:

   ```markdown
   ### Decision 1: [Decision Title]
   **Context**: [Why this decision is needed]
   **Options Considered**:
   | Option | Pros | Cons |
   |--------|------|------|
   | Option A | [Pros] | [Cons] |
   | Option B | [Pros] | [Cons] |

   **Decision**: [Selected option]
   **Rationale**: [Why chosen]
   **Consequences**: [Implications]
   ```

4. **Component Design**:

   For each major component:

   ```markdown
   ### Component: [Name]
   **Responsibility**: [What it does]
   **Interfaces**: [API/contract definition]
   **Dependencies**: [What it depends on]
   **Data Flow**: [Input] → [Processing] → [Output]
   ```

5. **Data Model**:

   - Entity definitions with schemas
   - Relationships between entities
   - Migration strategy if modifying existing data

6. **API Design**:

   For each endpoint:
   - Method and path
   - Request/response schemas
   - Error codes and meanings

7. **Integration Points**:

   | System | Integration Type | Data Exchanged |
   |--------|------------------|----------------|
   | [System] | [Sync/Async/Event] | [Format] |

8. **Non-Functional Architecture**:

   - **Performance**: How targets will be achieved
   - **Scalability**: Scaling strategy
   - **Security**: Auth, authz, encryption
   - **Reliability**: Availability, failure handling

9. **Observability**:

   - Logging strategy
   - Metrics to collect
   - Tracing approach

10. **Testing Strategy**:

    - Unit testing approach
    - Integration testing approach
    - Performance testing approach

11. **Deployment**:

    - Rollout strategy
    - Rollback plan
    - Feature flags

12. **Constitution Compliance Check**:

    Verify against Wave principles:
    - [ ] Single binary, zero dependencies
    - [ ] Manifest as single source of truth
    - [ ] Persona-scoped execution
    - [ ] Fresh memory at step boundaries
    - [ ] Contracts at every handover
    - [ ] Credentials never touch disk

13. **Write Architecture Document**:
    Update `.bmad/products/[slug]/docs/architecture.md`

14. **Report**:

    ```markdown
    ## Architecture Document Created

    **Product**: [Name]
    **Path**: [ARCH_FILE]

    ### Key Decisions
    1. [Decision 1]: [Choice]
    2. [Decision 2]: [Choice]

    ### Components
    - [Component 1]: [Purpose]
    - [Component 2]: [Purpose]

    ### Constitution Compliance
    - [Status of each principle]

    ### Next Steps
    1. Review with development team
    2. Run `/bmad.epics` to create story breakdown
    ```

### Error Handling

- If no PRD: Prompt to run `/bmad.prd` first
- If constitution violations: Document justification or flag as blocker
- If unclear requirements: Reference back to PRD, ask for clarification
