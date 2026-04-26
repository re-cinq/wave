# Contract: Recommendation Document

**Deliverable**: `specs/1106-webui-framework-research/recommendation.md`  
**Type**: Structural (document completeness)  
**Validates**: FR-006, SC-004, SC-005, SC-006, SC-007

## Acceptance Criteria

### Winner Selection (SC-004)

1. **Names exactly one framework** as the primary recommendation
2. **Justification references at least 3 evaluation criteria by name** from the matrix
3. **Justification cites specific evidence** (not just ratings — references findings)
4. **Optionally names a runner-up** with rationale for why it was not selected

### Migration Strategy (FR-006, SC-005)

5. **Declares strategy type**: incremental or big-bang
6. **If incremental**:
   - Specifies at least 3 pages to migrate first, in priority order
   - Each page has rationale for its position in the migration order
   - Explains coexistence strategy (how old Go templates and new framework coexist during migration)
7. **If big-bang**:
   - Justifies why incremental is not feasible
   - Provides estimated scope and timeline

### Risk Assessment (SC-006)

8. **Identifies at least 3 risks** with:
   - Title and severity (High / Medium / Low)
   - Description of what could go wrong
   - Proposed mitigation strategy
9. **Must address** (from spec):
   - Build pipeline complexity and CI impact
   - New runtime dependencies introduced (should be none, but document verification)
   - Developer onboarding / learning curve

### Constraints (SC-007)

10. **No backend API changes proposed** — recommendation is frontend-only
11. **Single-binary distribution preserved** — recommendation must maintain go:embed compatibility

### Cross-References

12. **References matrix findings**: at least 3 links/citations to specific matrix cells
13. **References PoC results**: cites empirical findings from PoC implementation
14. **Addresses edge cases** from the spec:
    - Template function migration strategy (30 custom functions)
    - Authentication mode compatibility (4 auth modes, CSRF)
    - Handler test suite impact (~1,900 lines)

## Validation Method

Manual review checklist — verify all 14 criteria. Count named criteria in justification (≥3), count risks (≥3), verify migration phases (≥3 if incremental).
