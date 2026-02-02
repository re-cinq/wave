# Technical Architecture: [FEATURE NAME]

**PRD**: [Link to prd.md]
**Created**: [DATE]
**Last Updated**: [DATE]
**Owner**: [Architect Name]
**Status**: Draft | In Review | Approved

## Executive Summary

[2-3 sentences describing the technical approach and key decisions]

## System Context

### Current Architecture
```
[ASCII diagram or description of current system state]
```

### Target Architecture
```
[ASCII diagram or description of target system state]
```

### Scope of Changes
- **In Scope**: [Components being modified]
- **Out of Scope**: [Components not being touched]

## Technical Decisions

### Decision 1: [Decision Title]
**Context**: [Why this decision is needed]
**Options Considered**:
| Option | Pros | Cons |
|--------|------|------|
| Option A | [Pros] | [Cons] |
| Option B | [Pros] | [Cons] |
| Option C | [Pros] | [Cons] |

**Decision**: [Selected option]
**Rationale**: [Why this option was chosen]
**Consequences**: [What this means for the implementation]

### Decision 2: [Decision Title]
[Same structure as above]

## Component Design

### Component: [Component Name]
**Responsibility**: [What this component does]
**Interfaces**:
```
[Interface definition or API contract]
```

**Dependencies**:
- [Internal dependency]
- [External dependency]

**Data Flow**:
```
[Input] → [Processing] → [Output]
```

### Component: [Component Name]
[Same structure as above]

## Data Model

### Entity: [Entity Name]
```
[Schema definition or structure]
```

### Relationships
```
[Entity A] --[relationship]--> [Entity B]
```

### Data Migration
[Migration strategy if modifying existing data]

## API Design

### Endpoint: [Method] [Path]
**Purpose**: [What this endpoint does]
**Request**:
```json
{
  "field": "type"
}
```
**Response**:
```json
{
  "field": "type"
}
```
**Error Codes**:
| Code | Meaning |
|------|---------|
| 400 | [Description] |
| 404 | [Description] |

## Integration Points

### Internal Integrations
| System | Integration Type | Data Exchanged |
|--------|------------------|----------------|
| [System] | [Sync/Async/Event] | [Data format] |

### External Integrations
| Service | Purpose | Authentication |
|---------|---------|----------------|
| [Service] | [Purpose] | [Auth method] |

## Non-Functional Requirements

### Performance
- **Latency**: [Target latency]
- **Throughput**: [Target throughput]
- **Approach**: [How we'll achieve these targets]

### Scalability
- **Current load**: [Current metrics]
- **Target load**: [Expected growth]
- **Scaling strategy**: [How we'll scale]

### Security
- **Authentication**: [Auth approach]
- **Authorization**: [Authz approach]
- **Data protection**: [Encryption, masking, etc.]
- **Audit logging**: [What's logged]

### Reliability
- **Availability target**: [e.g., 99.9%]
- **Failure modes**: [What can fail and how we handle it]
- **Recovery strategy**: [How we recover from failures]

## Observability

### Logging
- [Log category]: [What's logged, format]

### Metrics
| Metric | Type | Alert Threshold |
|--------|------|-----------------|
| [Metric] | Counter/Gauge/Histogram | [Threshold] |

### Tracing
- [Spans to be instrumented]

## Testing Strategy

### Unit Testing
- [Approach and coverage targets]

### Integration Testing
- [Key integration tests]

### Performance Testing
- [Load testing approach]

## Deployment

### Rollout Strategy
- [ ] Feature flag: [Flag name]
- [ ] Gradual rollout: [Percentage steps]
- [ ] Canary deployment: [Canary metrics]

### Rollback Plan
- [How to rollback if issues arise]

### Feature Flags
| Flag | Purpose | Default | Cleanup Date |
|------|---------|---------|--------------|
| [Flag] | [Purpose] | true/false | [Date] |

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| [Technical risk] | [Impact] | [Mitigation] |

## Open Questions

- [ ] [Technical question 1]
- [ ] [Technical question 2]

## Appendix

### Glossary
- **Term**: Definition

### References
- [Link to relevant documentation]

### Revision History
| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | [Date] | [Name] | Initial draft |
