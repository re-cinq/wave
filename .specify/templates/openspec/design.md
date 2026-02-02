# Design Document: [CHANGE NAME]

**Proposal**: [Link to proposal.md]
**Created**: [DATE]
**Author**: [Name]
**Status**: Draft | In Review | Approved

## Overview

[Brief description of the technical approach to implement this change]

## Architecture

### Current State
```
[ASCII diagram or description of current architecture]
```

### Target State
```
[ASCII diagram or description of target architecture]
```

### Changes Summary
[What's being added, modified, or removed]

## Detailed Design

### Component 1: [Component Name]

**Purpose**: [What this component does]

**Interface**:
```
[API/function signatures]
```

**Implementation Notes**:
- [Note 1]
- [Note 2]

### Component 2: [Component Name]

[Same structure as above]

## Data Model

### New Entities
```
[Entity definition]
```

### Modified Entities
```
[Changes to existing entities]
```

### Data Migration
[Migration strategy if applicable]

## API Changes

### New Endpoints

#### [METHOD] [Path]
**Purpose**: [What it does]
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

### Modified Endpoints
[Changes to existing endpoints]

## Error Handling

| Error Condition | Response | Recovery |
|-----------------|----------|----------|
| [Condition] | [Error response] | [How to recover] |

## Testing Strategy

### Unit Tests
- [ ] [Test case 1]
- [ ] [Test case 2]

### Integration Tests
- [ ] [Test case 1]

### Manual Testing
- [ ] [Test scenario]

## Rollout Plan

### Phase 1: [Phase name]
- [ ] [Step 1]
- [ ] [Step 2]

### Rollback Plan
[How to rollback if issues arise]

## Security Considerations

- [Security consideration 1]
- [Security consideration 2]

## Performance Considerations

- [Performance consideration 1]
- [Performance consideration 2]

## Open Questions

- [ ] [Technical question]

## References

- [Link to relevant documentation]

---

## Review Checklist

- [ ] Design addresses all requirements in proposal
- [ ] Architecture is consistent with existing system
- [ ] Error handling is comprehensive
- [ ] Testing strategy is adequate
- [ ] Security considerations addressed
- [ ] Performance considerations addressed
