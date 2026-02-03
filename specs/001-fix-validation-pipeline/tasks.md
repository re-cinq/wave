# Task Breakdown: Fix Wave Validation Pipeline Error Wrapping

## Phase 1: Deep Investigation (Priority: Critical)

### T1.1: Analyze Step Configuration Differences
**Status**: Pending
**Owner**: Unassigned
**Estimated Effort**: 4 hours
**Dependencies**: None

**Description**: Compare pipeline step configurations to understand why some steps correctly handle valid JSON while others wrap it in error metadata.

**Acceptance Criteria**:
- [ ] Create comparison matrix of working vs failing steps
- [ ] Document configuration differences between `plan-enhancements` (working) and `scan-issues`/`apply-enhancements` (failing)
- [ ] Identify patterns correlating with success/failure
- [ ] Generate hypothesis for root cause

**Tasks**:
1. Examine `.wave/workspaces/github-issue-enhancer/*/CLAUDE.md` files
2. Compare persona prompts and instructions
3. Check contract schema differences in `.wave/contracts/`
4. Analyze pipeline step configurations in `.wave/pipelines/github-issue-enhancer.yaml`
5. Create step comparison matrix

### T1.2: Map Validation Flow Execution Paths
**Status**: Pending
**Owner**: Unassigned
**Estimated Effort**: 6 hours
**Dependencies**: None
**Blocks**: T2.1, T3.1

**Description**: Trace the exact code execution path from AI output to schema validation for both success and failure cases.

**Acceptance Criteria**:
- [ ] Document validation flow for success path (plan-enhancements)
- [ ] Document validation flow for failure path (scan-issues, apply-enhancements)
- [ ] Identify decision points leading to error wrapping vs pass-through
- [ ] Create flow diagrams for both paths

**Tasks**:
1. Trace execution in `internal/pipeline/executor.go`
2. Follow validation logic in `internal/contract/jsonschema.go`
3. Examine error recovery in `internal/contract/json_recovery.go`
4. Map decision points and conditional logic
5. Create visual flow diagrams

### T1.3: Locate Error Wrapper Generation Code
**Status**: Pending
**Owner**: Unassigned
**Estimated Effort**: 3 hours
**Dependencies**: T1.2
**Blocks**: T2.1, T3.1

**Description**: Find the exact code location that generates error wrapper structures and understand what triggers it for valid JSON.

**Acceptance Criteria**:
- [ ] Identify function/method that creates error wrapper structure
- [ ] Document the conditions that trigger error wrapping
- [ ] Understand why valid JSON incorrectly triggers error path
- [ ] Map the data flow leading to error wrapper creation

**Tasks**:
1. Search codebase for error wrapper structure generation
2. Trace backwards from known error wrapper format
3. Identify conditional logic causing false positives
4. Document trigger conditions and data flow

## Phase 2: Quick Fix Implementation (Priority: High)

### T2.1: Implement Error Wrapper Detection
**Status**: Pending
**Owner**: Unassigned
**Estimated Effort**: 4 hours
**Dependencies**: T1.2, T1.3
**Blocks**: T2.2, T2.3

**Description**: Add logic to detect error wrapper structures in validation input and extract raw content for validation.

**Acceptance Criteria**:
- [ ] Create `DetectErrorWrapper()` function
- [ ] Implement wrapper structure identification
- [ ] Extract raw output from wrapper when detected
- [ ] Maintain backward compatibility with non-wrapped inputs

**Tasks**:
1. Create error wrapper detection function in `internal/contract/`
2. Add JSON unmarshaling logic for wrapper structure
3. Implement raw output extraction
4. Add error handling for malformed wrappers

### T2.2: Enhance Schema Validation Logic
**Status**: Pending
**Owner**: Unassigned
**Estimated Effort**: 3 hours
**Dependencies**: T2.1
**Blocks**: T2.3, T2.4

**Description**: Modify schema validation to automatically handle both wrapped and unwrapped inputs.

**Acceptance Criteria**:
- [ ] Integrate wrapper detection into validation flow
- [ ] Validate raw content when wrapper detected
- [ ] Preserve original validation for non-wrapped inputs
- [ ] Maintain error context and debugging information

**Tasks**:
1. Modify `ValidateAgainstSchema()` to use wrapper detection
2. Update validation entry points to handle both input types
3. Preserve error context for debugging
4. Ensure proper error propagation

### T2.3: Create Comprehensive Unit Tests
**Status**: Pending
**Owner**: Unassigned
**Estimated Effort**: 5 hours
**Dependencies**: T2.1, T2.2
**Blocks**: T2.4

**Description**: Build comprehensive test suite covering all input formats and validation scenarios.

**Acceptance Criteria**:
- [ ] Test valid JSON in error wrapper format
- [ ] Test valid JSON in direct format
- [ ] Test invalid JSON in both wrapper and direct formats
- [ ] Test edge cases and malformed inputs
- [ ] Achieve >95% code coverage for new logic

**Test Scenarios**:
1. Valid JSON wrapped in error structure
2. Valid JSON direct input
3. Invalid JSON syntax wrapped in error structure
4. Invalid JSON syntax direct input
5. Schema violations (valid syntax, missing fields)
6. Malformed wrapper structures
7. Empty and null inputs

### T2.4: Integration Testing with Real Artifacts
**Status**: Pending
**Owner**: Unassigned
**Estimated Effort**: 3 hours
**Dependencies**: T2.3

**Description**: Test the fix using actual workspace artifacts from failed pipeline runs.

**Acceptance Criteria**:
- [ ] Test with `scan-issues/artifact.json` (error wrapped)
- [ ] Test with `apply-enhancements/artifact.json` (error wrapped)
- [ ] Test with `plan-enhancements/artifact.json` (direct JSON)
- [ ] Verify all tests pass with enhanced validation
- [ ] Confirm no regression in working cases

**Tasks**:
1. Create integration tests using real artifact files
2. Test validation with actual error wrapper content
3. Verify raw output extraction and validation
4. Test performance with realistic data sizes

## Phase 3: Root Cause Resolution (Priority: Medium)

### T3.1: Analyze Step Execution Logic Differences
**Status**: Pending
**Owner**: Unassigned
**Estimated Effort**: 4 hours
**Dependencies**: T1.1, T1.2, T1.3
**Blocks**: T3.2

**Description**: Deep dive into why some steps trigger error wrapping while others don't, based on investigation findings.

**Acceptance Criteria**:
- [ ] Identify the specific code path causing inconsistent behavior
- [ ] Document configuration or logic differences between steps
- [ ] Understand the trigger condition for premature error wrapping
- [ ] Propose specific fix for root cause

**Tasks**:
1. Compare step execution contexts
2. Analyze persona prompt impact on output interpretation
3. Check timeout and retry configuration differences
4. Identify the specific conditional logic causing divergence

### T3.2: Fix Root Cause in Pipeline Logic
**Status**: Pending
**Owner**: Unassigned
**Estimated Effort**: 6 hours
**Dependencies**: T3.1
**Blocks**: T3.3

**Description**: Implement the fix for the underlying issue causing inconsistent step behavior.

**Acceptance Criteria**:
- [ ] Modify pipeline logic to prevent false positive error wrapping
- [ ] Ensure consistent behavior across all pipeline steps
- [ ] Preserve error handling for genuine failures
- [ ] Maintain backward compatibility

**Tasks**:
1. Implement fix based on root cause analysis
2. Modify pipeline executor or contract validation logic
3. Update step configuration if needed
4. Test fix with various pipeline scenarios

### T3.3: Standardize Step Configurations
**Status**: Pending
**Owner**: Unassigned
**Estimated Effort**: 3 hours
**Dependencies**: T3.2

**Description**: Create consistent templates and validation for step configurations to prevent future issues.

**Acceptance Criteria**:
- [ ] Create standardized step configuration templates
- [ ] Implement configuration validation tools
- [ ] Update existing step configurations for consistency
- [ ] Document best practices for step setup

**Tasks**:
1. Create step configuration templates
2. Build configuration validation tools
3. Update existing pipeline configurations
4. Document configuration guidelines

## Phase 4: Integration and Testing (Priority: Medium)

### T4.1: End-to-End Pipeline Testing
**Status**: Pending
**Owner**: Unassigned
**Estimated Effort**: 4 hours
**Dependencies**: T2.4, T3.2
**Blocks**: T4.2

**Description**: Verify the complete fix works across all pipeline scenarios and use cases.

**Acceptance Criteria**:
- [ ] GitHub issue enhancer pipeline completes successfully
- [ ] Microsoft TypeScript repository pipeline works end-to-end
- [ ] Edge cases and error conditions handled properly
- [ ] Performance meets requirements (<1 second for valid outputs)

**Tasks**:
1. Run complete GitHub issue enhancer pipeline
2. Test with various repository scenarios
3. Validate edge cases and error handling
4. Measure performance and success rates

### T4.2: Regression Testing Suite
**Status**: Pending
**Owner**: Unassigned
**Estimated Effort**: 3 hours
**Dependencies**: T4.1

**Description**: Comprehensive testing to ensure existing functionality remains intact.

**Acceptance Criteria**:
- [ ] All existing pipeline configurations work correctly
- [ ] Error handling for genuine validation failures preserved
- [ ] Retry mechanisms function for actual malformed output
- [ ] Debugging and audit capabilities maintained
- [ ] No performance regression in baseline scenarios

**Tasks**:
1. Test all existing pipeline configurations
2. Validate error handling for genuine failures
3. Test retry mechanisms with malformed input
4. Verify debugging and audit functionality
5. Performance benchmarking

## Phase 5: Documentation and Cleanup (Priority: Low)

### T5.1: Update Technical Documentation
**Status**: Pending
**Owner**: Unassigned
**Estimated Effort**: 3 hours
**Dependencies**: T4.2

**Description**: Update architecture documentation and create troubleshooting guides.

**Acceptance Criteria**:
- [ ] Updated validation flow documentation
- [ ] Troubleshooting guide for validation issues
- [ ] Best practices for pipeline configuration
- [ ] Migration guide for existing pipelines

**Tasks**:
1. Update architecture diagrams and documentation
2. Create troubleshooting guide
3. Document best practices
4. Write migration guide

### T5.2: Code Cleanup and Optimization
**Status**: Pending
**Owner**: Unassigned
**Estimated Effort**: 2 hours
**Dependencies**: T4.2

**Description**: Clean up temporary code and optimize performance based on testing results.

**Acceptance Criteria**:
- [ ] Remove temporary debugging code
- [ ] Optimize validation performance
- [ ] Clean up redundant logic
- [ ] Finalize code comments and documentation

**Tasks**:
1. Remove debugging and temporary code
2. Optimize performance bottlenecks
3. Clean up redundant validation logic
4. Update code comments and documentation

## Summary

**Total Tasks**: 15
**Critical Priority**: 3 tasks (Phase 1)
**High Priority**: 4 tasks (Phase 2)
**Medium Priority**: 6 tasks (Phases 3-4)
**Low Priority**: 2 tasks (Phase 5)

**Total Estimated Effort**: 56 hours
**Critical Path**: T1.2 → T1.3 → T2.1 → T2.2 → T2.3 → T2.4 → T4.1 → T4.2

**Dependencies**:
- Phase 2 (Quick Fix) can begin once Phase 1 investigation is complete
- Phase 3 (Root Cause) runs in parallel with Phase 2 testing
- Phase 4 (Integration) requires both quick fix and root cause resolution
- Phase 5 (Documentation) can only begin after full validation

**Risk Mitigation**:
- Quick fix provides immediate relief while root cause investigation continues
- Comprehensive testing at each phase ensures stability
- Regression testing prevents breaking existing functionality
- Documentation ensures maintainability and future prevention