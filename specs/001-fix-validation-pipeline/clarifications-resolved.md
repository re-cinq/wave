# Resolved Clarifications for Validation Pipeline Fix

## Investigation Results from Workspace Analysis

Through examination of actual production workspace outputs, several key clarifications have been resolved:

### 1. Root Cause Location - RESOLVED ✅

**Finding**: The error wrapping occurs **inconsistently** across different pipeline steps.

**Evidence**:
- `plan-enhancements/artifact.json` - Contains pure valid JSON (no wrapper)
- `scan-issues/artifact.json` - Contains error wrapper with valid JSON in `raw_output`
- `apply-enhancements/artifact.json` - Contains error wrapper with valid JSON in `raw_output`

**Conclusion**: The issue is **step-specific**, not universal. Some steps correctly handle valid JSON while others incorrectly wrap it.

### 2. Validation Flow Architecture - PARTIALLY RESOLVED ⚠️

**Current Understanding**:
```
AI Output → [Step-Specific Logic] → [Validation OR Error Wrapper] → Schema Validation
```

**Pattern Identified**:
- **Success Path**: `AI Output → Direct Validation → Pass`
- **Failure Path**: `AI Output → Error Wrapper → Wrapper Validation → Fail`

**Still Needs Clarification**: What determines which path a step takes?

### 3. Error Detection Logic - RESOLVED ✅

**Finding**: Valid JSON is NOT being detected as invalid during parsing. The AI consistently produces correct JSON.

**Evidence**: In all error-wrapped artifacts, the `raw_output` field contains perfectly valid JSON that matches the required schema.

**Conclusion**: The problem is **not** with error detection but with **validation target selection** - the system validates the wrapper instead of the raw output.

### 4. Recovery and Retry Integration - PARTIALLY RESOLVED ⚠️

**Understanding**: The existing retry/recovery system appears to be creating the error wrappers, but it's doing so prematurely for valid outputs.

**Still Needs Clarification**:
- Is the retry system being triggered incorrectly?
- Or is the validation system not checking the `raw_output` field when present?

### 5. Debugging and Audit Preservation - RESOLVED ✅

**Finding**: Raw AI output is already preserved in the `raw_output` field of error wrappers.

**Conclusion**: The debugging infrastructure is intact. We just need to ensure validation uses this preserved raw output.

## Remaining Critical Questions

### 1. Step Behavior Inconsistency

**Question**: Why do some pipeline steps (like `plan-enhancements`) correctly handle valid JSON while others (like `scan-issues`, `apply-enhancements`) wrap it in error metadata?

**Investigation Needed**:
- Compare step configurations between working and non-working steps
- Examine step-specific validation logic
- Identify what triggers error wrapping vs pass-through

### 2. Validation Logic Selection

**Question**: When an error wrapper is present, why doesn't the validation system extract and validate the `raw_output` field?

**Investigation Needed**:
- Examine contract validation logic in `internal/contract/`
- Identify where validation target is selected
- Determine if wrapper detection logic exists or needs to be added

### 3. Error Wrapper Generation

**Question**: What specific condition triggers the creation of error wrappers for valid JSON output?

**Investigation Needed**:
- Trace the code path that generates error wrapper structure
- Identify the false positive condition
- Determine if this is in retry logic or validation logic

## Implementation Strategy Implications

Based on these clarifications:

### Preferred Approach: Option B - Fix at Validation Execution

**Rationale**:
- Error wrappers already preserve raw output correctly
- Debugging infrastructure is intact
- Minimal disruption to working steps
- Maintains backward compatibility

**Implementation**: Enhance validation system to:
1. Detect when input is an error wrapper structure
2. Extract `raw_output` field for validation
3. If `raw_output` validates successfully, continue with raw content
4. If `raw_output` validation fails, proceed with error handling

### Validation Target Detection

Add logic to identify error wrapper structure:
```json
{
  "attempts": number,
  "contract_type": string,
  "error_type": string,
  "raw_output": string,
  ...
}
```

When this structure is detected, validate the `raw_output` content instead of the wrapper.

## Next Steps for Implementation Planning

1. **Code Investigation**: Map the specific differences between working and non-working pipeline steps
2. **Validation Logic Enhancement**: Design wrapper detection and extraction logic
3. **Integration Testing**: Ensure fix works for both wrapped and non-wrapped inputs
4. **Regression Prevention**: Ensure actual validation failures still trigger proper error handling

These clarifications provide sufficient understanding to proceed with implementation planning while focusing investigation on the most critical remaining unknowns.