# Implementation Plan: Fix Wave Validation Pipeline Error Wrapping

## Executive Summary

This plan implements a combined approach to fix Wave's validation pipeline incorrectly wrapping valid AI output in error metadata. We start with deep investigation to understand step-specific behavior differences, then implement both immediate fixes and long-term root cause resolution.

## Phase 1: Deep Investigation (Days 1-2)

### 1.1 Map Step Behavior Differences

**Objective**: Understand why some pipeline steps correctly handle valid JSON while others wrap it in error metadata.

**Tasks**:
- Compare configurations between working steps (`plan-enhancements`) and failing steps (`scan-issues`, `apply-enhancements`)
- Examine step-specific personas, contracts, and execution contexts
- Identify configuration patterns that correlate with success/failure

**Deliverables**:
- Step comparison matrix showing differences
- Root cause hypothesis for inconsistent behavior

**Files to Examine**:
- `.wave/workspaces/github-issue-enhancer/*/CLAUDE.md` (persona configurations)
- `.wave/contracts/` (schema definitions used by each step)
- `.wave/pipelines/github-issue-enhancer.yaml` (step configurations)

### 1.2 Trace Validation Flow

**Objective**: Map the exact execution path from AI output to schema validation for both success and failure cases.

**Tasks**:
- Trace code execution in `internal/pipeline/executor.go`
- Follow validation logic in `internal/contract/` packages
- Identify decision points that lead to error wrapping vs. pass-through

**Deliverables**:
- Validation flow diagrams for both success and failure paths
- Identification of the specific condition triggering error wrapping

**Key Code Locations**:
- `internal/pipeline/executor.go` - Step execution logic
- `internal/contract/jsonschema.go` - Schema validation
- `internal/contract/json_recovery.go` - Error recovery system

### 1.3 Error Wrapper Generation Analysis

**Objective**: Identify what triggers creation of error wrapper structures for valid JSON.

**Tasks**:
- Find the code that generates the error wrapper structure
- Determine if this happens in retry logic, validation logic, or elsewhere
- Understand the conditions that incorrectly classify valid JSON as failed

**Deliverables**:
- Exact location of error wrapper generation
- Condition analysis showing why valid JSON triggers error path

## Phase 2: Quick Fix Implementation (Days 3-4)

### 2.1 Enhanced Validation System

**Objective**: Implement immediate fix to detect and handle error wrappers correctly.

**Implementation Strategy**:
```go
// Add to internal/contract/validation.go
func DetectErrorWrapper(input []byte) (*ErrorWrapper, []byte, error) {
    var wrapper ErrorWrapper
    if err := json.Unmarshal(input, &wrapper); err != nil {
        return nil, input, nil // Not an error wrapper, return original
    }

    // Check if this looks like an error wrapper
    if wrapper.ErrorType != "" && wrapper.RawOutput != "" {
        return &wrapper, []byte(wrapper.RawOutput), nil
    }

    return nil, input, nil
}

func ValidateWithWrapperDetection(input []byte, schema *Schema) error {
    wrapper, rawContent, err := DetectErrorWrapper(input)
    if err != nil {
        return err
    }

    // If we detected a wrapper, validate the raw content instead
    if wrapper != nil {
        return ValidateAgainstSchema(rawContent, schema)
    }

    // Otherwise validate the input directly
    return ValidateAgainstSchema(input, schema)
}
```

**Tasks**:
- Add error wrapper detection logic to contract validation
- Modify schema validation to check raw output when wrapper detected
- Maintain backward compatibility with non-wrapped inputs

**Deliverables**:
- Enhanced validation system that handles both wrapped and unwrapped inputs
- Unit tests covering all input formats
- Integration tests with actual pipeline scenarios

### 2.2 Testing and Validation

**Objective**: Ensure the quick fix resolves immediate pipeline failures without breaking existing functionality.

**Test Scenarios**:
1. **Valid JSON (wrapped)**: Error wrapper with correct JSON in `raw_output`
2. **Valid JSON (unwrapped)**: Direct JSON input matching schema
3. **Invalid JSON (wrapped)**: Error wrapper with malformed JSON in `raw_output`
4. **Invalid JSON (unwrapped)**: Direct malformed JSON input
5. **Schema violations**: Valid JSON syntax but missing required fields

**Deliverables**:
- Comprehensive test suite covering all scenarios
- Pipeline integration tests with real workspace artifacts
- Performance benchmarks ensuring no regression

## Phase 3: Root Cause Resolution (Days 5-7)

### 3.1 Step Configuration Analysis

**Objective**: Fix the underlying issue causing inconsistent step behavior.

**Investigation Focus**:
- Persona prompt differences that affect output interpretation
- Contract validation configuration differences
- Step-specific error handling settings
- Timeout and retry configuration variations

**Implementation**:
- Standardize step configurations for consistent behavior
- Fix the condition causing premature error wrapping
- Ensure all steps use the same validation approach

### 3.2 Pipeline Execution Logic Fix

**Objective**: Address the root cause in pipeline execution that incorrectly triggers error wrapping for valid outputs.

**Based on Investigation Results**:
- If issue is in retry logic: Modify retry conditions to avoid false positives
- If issue is in timeout handling: Adjust timeout detection for valid outputs
- If issue is in contract validation: Fix the validation trigger logic

**Implementation**:
- Modify pipeline executor to correctly identify validation failures
- Ensure error wrapping only occurs for actual failures, not valid outputs
- Preserve existing error handling for genuine failures

### 3.3 Configuration Standardization

**Objective**: Prevent future occurrences by standardizing step configurations.

**Tasks**:
- Create consistent templates for step configurations
- Implement validation for step configuration consistency
- Document best practices for pipeline step setup

**Deliverables**:
- Standardized step configuration templates
- Configuration validation tools
- Updated documentation and guidelines

## Phase 4: Integration and Testing (Days 8-9)

### 4.1 End-to-End Pipeline Testing

**Objective**: Verify the complete fix works across all pipeline scenarios.

**Test Plans**:
- GitHub issue enhancer pipeline with various issue types
- Microsoft TypeScript repository pipeline
- Edge cases and error conditions
- Performance and scalability testing

### 4.2 Regression Testing

**Objective**: Ensure existing functionality remains intact.

**Coverage**:
- All existing pipeline configurations
- Error handling for genuine validation failures
- Retry mechanisms for actual malformed output
- Debugging and audit capabilities

## Phase 5: Documentation and Cleanup (Day 10)

### 5.1 Documentation Updates

**Deliverables**:
- Updated architecture documentation
- Troubleshooting guide for validation issues
- Best practices for pipeline configuration
- Migration guide for existing pipelines

### 5.2 Code Cleanup

**Tasks**:
- Remove temporary debugging code
- Clean up any redundant validation logic
- Optimize performance based on testing results

## Risk Mitigation

### High-Priority Risks

1. **Breaking Existing Pipelines**
   - Mitigation: Extensive regression testing before deployment
   - Rollback plan: Feature flag to disable new validation logic

2. **Performance Regression**
   - Mitigation: Benchmark testing throughout development
   - Optimization: Cache validation results for repeated operations

3. **Incomplete Fix**
   - Mitigation: Test with production workspace artifacts
   - Validation: End-to-end pipeline testing with various scenarios

### Medium-Priority Risks

1. **Configuration Complexity**
   - Mitigation: Provide clear documentation and examples
   - Support: Automated configuration validation tools

2. **Edge Case Handling**
   - Mitigation: Comprehensive test suite covering known edge cases
   - Monitoring: Enhanced logging for debugging edge cases

## Success Metrics

### Primary Metrics
- **False Failure Rate**: Reduce from 100% to 0% for valid AI output
- **Pipeline Success Rate**: Increase successful completions for valid scenarios
- **Processing Time**: Maintain <1 second for valid outputs (no retry loops)

### Secondary Metrics
- **Error Handling Quality**: Maintain helpful error messages for actual failures
- **Debugging Capability**: Preserve or improve debugging information
- **Performance**: No regression in validation speed

## Dependencies

### Code Dependencies
- Understanding of current pipeline execution flow
- Access to production workspace artifacts for testing
- Wave's constitutional compliance requirements

### Resource Dependencies
- Development environment with Wave codebase
- Test pipelines for validation
- Production-like test scenarios

## Rollout Strategy

### Phase 1: Development and Testing
- Implement in development environment
- Test with historical workspace artifacts
- Validate against known failing scenarios

### Phase 2: Staging Deployment
- Deploy to staging environment
- Run complete test suites
- Performance and reliability testing

### Phase 3: Production Deployment
- Gradual rollout with feature flags
- Monitor pipeline success rates
- Rollback plan if issues detected

This implementation plan provides a structured approach to solving the validation pipeline issue while maintaining Wave's stability and performance requirements.