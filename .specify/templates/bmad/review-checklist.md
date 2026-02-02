# Code Review Checklist: [FEATURE/STORY NAME]

**Story**: [Link to story]
**PR**: [Link to PR]
**Reviewer**: [Name]
**Date**: [DATE]
**Status**: Pending | Approved | Changes Requested

## Summary

**Changes Overview**: [Brief description of what this PR does]
**Files Changed**: [Count]
**Lines Added/Removed**: +[X] / -[Y]

## Functional Correctness

### Requirements Alignment
- [ ] Implementation matches acceptance criteria
- [ ] All user scenarios are addressed
- [ ] Edge cases are handled
- [ ] Error states are handled gracefully

### Logic Verification
- [ ] Business logic is correct
- [ ] Data transformations are accurate
- [ ] State management is proper
- [ ] No off-by-one errors

## Code Quality

### Readability
- [ ] Code is self-documenting
- [ ] Variable/function names are clear and descriptive
- [ ] Complex logic has explanatory comments
- [ ] No unnecessary comments (commented-out code, TODOs without tickets)

### Structure
- [ ] Single responsibility principle followed
- [ ] Functions are appropriately sized
- [ ] Code duplication is minimized
- [ ] Abstractions are appropriate (not over/under-engineered)

### Style
- [ ] Follows project coding conventions
- [ ] Consistent formatting
- [ ] No linting errors/warnings
- [ ] Imports are organized

## Testing

### Test Coverage
- [ ] Unit tests cover new functionality
- [ ] Unit tests cover edge cases
- [ ] Unit tests cover error paths
- [ ] Integration tests added where appropriate

### Test Quality
- [ ] Tests are meaningful (not just for coverage)
- [ ] Test names describe behavior being tested
- [ ] Tests are isolated and repeatable
- [ ] No flaky tests introduced

### Existing Tests
- [ ] All existing tests pass
- [ ] No tests skipped without justification
- [ ] No regression in functionality

## Performance

### Efficiency
- [ ] No unnecessary computations
- [ ] Appropriate data structures used
- [ ] No N+1 query problems
- [ ] Resource cleanup handled (connections, file handles)

### Scalability
- [ ] Handles expected load
- [ ] No unbounded growth (memory, disk)
- [ ] Pagination used for large datasets

## Security

### Data Protection
- [ ] No sensitive data logged
- [ ] No credentials in code
- [ ] Input validation present
- [ ] Output encoding/escaping correct

### Authentication & Authorization
- [ ] Auth checks in place
- [ ] Permission checks correct
- [ ] No privilege escalation paths

### Common Vulnerabilities
- [ ] No SQL injection vectors
- [ ] No XSS vectors
- [ ] No CSRF vulnerabilities
- [ ] No path traversal vulnerabilities

## Error Handling

### Robustness
- [ ] Errors are caught appropriately
- [ ] Error messages are helpful (not exposing internals)
- [ ] Failures are graceful
- [ ] Recovery paths exist where appropriate

### Logging & Observability
- [ ] Appropriate logging added
- [ ] Log levels are correct
- [ ] Metrics/tracing added if needed
- [ ] No excessive logging

## Documentation

### Code Documentation
- [ ] Public APIs documented
- [ ] Complex algorithms explained
- [ ] Configuration options documented

### External Documentation
- [ ] README updated if needed
- [ ] API documentation updated
- [ ] Architecture docs updated if needed

## Deployment & Operations

### Backwards Compatibility
- [ ] No breaking changes to APIs
- [ ] Database migrations are safe
- [ ] Feature flags used appropriately

### Rollback Safety
- [ ] Can be rolled back safely
- [ ] No irreversible data changes
- [ ] Feature flag allows disabling

### Configuration
- [ ] New config documented
- [ ] Defaults are sensible
- [ ] Environment-specific values handled

## Review Notes

### Highlights
- [Something done particularly well]

### Concerns
- [Area of concern that needs discussion]

### Suggestions
- [Optional improvement suggestion]

### Blocking Issues
- [ ] [Issue that must be fixed before approval]

## Final Verdict

- [ ] **Approved** - Ready to merge
- [ ] **Approved with comments** - Minor issues, can merge after addressing
- [ ] **Changes Requested** - Must be revised before approval

**Comments**:
[Final reviewer comments]

---

## Revision History

| Review | Date | Reviewer | Verdict |
|--------|------|----------|---------|
| Initial | [Date] | [Name] | [Verdict] |
