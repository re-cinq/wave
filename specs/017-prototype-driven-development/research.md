# Research: Prototype-Driven Development Pipelines

**Feature**: 017-prototype-driven-development
**Date**: 2026-02-02
**Status**: Complete

## Research Questions

### Q1: How should speckit integration work?

**Decision**: External tool integration via prompt templates with conditional execution

**Rationale**:
- Speckit is an optional external tool, similar to how Claude is an external adapter
- Wave should not have a hard dependency on speckit availability
- The pipeline should work with or without speckit present

**Alternatives Considered**:
1. **Hard dependency on speckit** - Rejected because it violates P1 (zero runtime dependencies)
2. **Embedded speckit functionality** - Rejected because speckit is a separate, evolving tool
3. **API integration** - Rejected because speckit is a CLI tool, not a service

**Implementation**:
```yaml
exec:
  type: prompt
  source: |
    {{ if exists ".specify/" }}
    # Speckit detected - use speckit commands
    Run `/speckit.spec` to generate the specification.
    {{ else }}
    # No speckit - manual specification
    Generate specification following the spec-phase schema.
    {{ end }}
```

---

### Q2: What is the optimal phase granularity?

**Decision**: Four main phases (spec, docs, dummy, implement) with sub-steps

**Rationale**:
- Each phase produces distinct, reviewable artifacts
- Phase boundaries align with natural review points in development
- Sub-steps allow for navigator-first analysis within each phase
- Matches the user's original workflow description

**Alternatives Considered**:
1. **More phases (6-8)** - Rejected as too fragmented; harder to resume and manage
2. **Fewer phases (2-3)** - Rejected as insufficient artifact granularity
3. **Flat steps without phases** - Rejected because phases provide conceptual grouping

**Phase Breakdown**:
| Phase | Sub-steps | Primary Persona |
|-------|-----------|-----------------|
| spec | navigate, define | navigator, philosopher |
| docs | generate | philosopher |
| dummy | scaffold, verify | craftsman, auditor |
| implement | plan, code, review | planner, craftsman, auditor |

---

### Q3: How should stale artifact detection work?

**Decision**: Timestamp-based comparison using existing SQLite state

**Rationale**:
- Wave already persists step completion timestamps
- Simple comparison: if input artifact timestamp > step completion timestamp, mark stale
- No need for content hashing (complexity vs. value tradeoff)
- Aligns with existing state persistence patterns

**Alternatives Considered**:
1. **Content hashing** - Rejected as over-engineered for this use case
2. **Manual invalidation** - Rejected because users shouldn't track this manually
3. **Always re-run downstream** - Rejected as wasteful of resources

**Implementation**:
```go
func IsStale(state *StepState, inputArtifacts []Artifact) bool {
    for _, artifact := range inputArtifacts {
        if artifact.CreatedAt.After(state.CompletedAt) {
            return true
        }
    }
    return false
}
```

---

### Q4: How should the dummy phase ensure runnable prototypes?

**Decision**: Verification sub-step with auditor persona checks runnable criteria

**Rationale**:
- Craftsman creates the prototype; auditor verifies independently
- Clear separation of concerns (build vs. verify)
- Verification can fail and trigger retry without re-scaffolding
- Contract schema requires `runnable: true` with `entry_point`

**Alternatives Considered**:
1. **Self-verification by craftsman** - Rejected; personas shouldn't verify own work
2. **Automated test execution** - Partially adopted; contract can include test_suite type
3. **No verification** - Rejected; fails FR-010 (demonstrate interfaces without full logic)

**Verification Checklist**:
- All interfaces from spec are present
- Entry point exists and can be invoked
- Stub responses match documented formats
- No real business logic (only placeholders)

---

### Q5: How should re-running individual phases work?

**Decision**: Use existing `--from-step` flag with phase-aware step IDs

**Rationale**:
- Wave already supports `wave resume --from-step`
- Phase IDs are step ID prefixes (e.g., `spec-navigate`, `docs-generate`)
- No new CLI infrastructure needed

**Alternatives Considered**:
1. **New `--from-phase` flag** - Rejected; adds unnecessary complexity
2. **Interactive phase selection** - Rejected; Wave is non-interactive by design
3. **Always re-run from beginning** - Rejected; wasteful and poor UX

**Usage**:
```bash
# Re-run from docs phase
wave resume --from-step docs-generate

# Re-run just implementation
wave resume --from-step implement-plan
```

---

### Q6: What artifacts should flow to the implementation phase?

**Decision**: All prior phase artifacts are available, with explicit injection of critical ones

**Rationale**:
- Implementation needs full context: spec, docs, and prototype
- Explicit injection prevents context pollution
- Allows craftsman to reference original requirements

**Artifact Injection for implement-code**:
```yaml
memory:
  inject_artifacts:
    - step: spec-define
      artifact: specification
      as: spec
    - step: docs-generate
      artifact: feature_docs
      as: docs
    - step: dummy-scaffold
      artifact: prototype
      as: prototype
    - step: implement-plan
      artifact: implementation_plan
      as: plan
```

---

### Q7: How should contracts handle partial specification?

**Decision**: Required fields are minimal; optional fields allow incremental refinement

**Rationale**:
- Not all projects need all contract fields (e.g., API design for CLI-only tools)
- Required fields ensure minimum viability: title, description, user_stories, entities
- Optional fields: interfaces, edge_cases, success_metrics

**Alternatives Considered**:
1. **All fields required** - Rejected; too rigid for varied project types
2. **No required fields** - Rejected; contracts become meaningless
3. **Project-type templates** - Future enhancement; not MVP

---

### Q8: How should the pipeline handle external dependencies in dummy phase?

**Decision**: Fail gracefully with clear error messages; support retry with timeout

**Rationale**:
- Dummy phase may need to install dependencies, run build tools
- External failures shouldn't lose progress
- Clear errors guide resolution

**Implementation**:
- Use existing `on_failure: retry` with `max_retries: 2`
- Step timeout prevents hanging on network issues
- Error messages include failed command and exit code

---

## Technology Decisions

### Pipeline Definition Format

**Decision**: Standard Wave pipeline YAML

**Rationale**: Consistency with existing pipelines; no new parsing needed

### Contract Validation

**Decision**: JSON Schema (draft-07)

**Rationale**: Existing infrastructure in `internal/contract/jsonschema.go`

### Event Format

**Decision**: NDJSON with existing event structure

**Rationale**: Consistency with Wave's observable progress principle (P10)

---

## Dependencies and Risks

### Dependencies

| Dependency | Status | Notes |
|------------|--------|-------|
| Wave pipeline executor | Existing | No changes needed |
| JSON Schema validator | Existing | No changes needed |
| SQLite state persistence | Existing | No changes needed |
| Speckit (optional) | External | Graceful degradation if missing |

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Spec phase produces invalid JSON | Medium | High | Schema validation with retry |
| Dummy prototype not runnable | Medium | Medium | Auditor verification step |
| Implementation exceeds token limits | High | Medium | Summarizer relay configured |
| Speckit unavailable | Low | Low | Fallback to manual spec |

---

## Open Questions (None)

All research questions have been resolved. Ready for Phase 1 design.
