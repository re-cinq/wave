# Specification: Fix Wave Validation Pipeline Incorrect Error Wrapping

## Problem Statement

Wave's validation pipeline is incorrectly treating valid AI output as validation failures. When an AI persona produces JSON that perfectly matches the required schema, the pipeline wraps this correct output in error metadata and validates the error wrapper instead of the actual AI output.

### Evidence of the Problem

From recent pipeline execution:

**AI Output (correct):**
```json
{
  "repository": {"owner": "re-cinq", "name": "wave"},
  "total_issues": 0,
  "analyzed_count": 0,
  "poor_quality_issues": [],
  "quality_threshold": 70,
  "timestamp": "2026-02-03T15:30:00Z"
}
```

**What gets validated instead (incorrect):**
```json
{
  "attempts": 5,
  "contract_type": "json_schema",
  "error_type": "persistent_output_format_failure",
  "exit_code": 0,
  "final_error": "contract validation failed...",
  "raw_output": "[THE CORRECT JSON ABOVE]",
  "recommendations": ["Review the AI persona prompt..."],
  "step_id": "scan-issues",
  "timestamp": "2026-02-03T19:47:59+01:00",
  "tokens_used": 378
}
```

The validation system is validating the error wrapper against the schema instead of extracting and validating the `raw_output` field that contains the correct JSON.

## Current Behavior

1. AI persona executes and produces valid JSON output ✅
2. Pipeline incorrectly detects this as a "format failure" ❌
3. Pipeline wraps correct output in error metadata structure ❌
4. Validator attempts to validate error wrapper against schema ❌
5. Validation fails because error wrapper != expected schema ❌
6. Pipeline reports false failure ❌

## Required Behavior

1. AI persona executes and produces valid JSON output ✅
2. Pipeline extracts raw AI output ✅
3. Pipeline validates raw output against schema ✅
4. If validation passes, pipeline continues with raw output ✅
5. If validation fails, then apply error handling and retry logic ✅

## Acceptance Criteria

### Primary Success Criteria

1. **Valid JSON Detection**: Pipeline correctly identifies when AI output matches the required schema
2. **Pass-through Validation**: Valid AI output passes validation without being wrapped in error metadata
3. **Schema Compliance**: Only the actual AI output (not error wrappers) is validated against the expected schema
4. **Pipeline Continuation**: When AI output is valid, pipeline continues to next step immediately

### Secondary Success Criteria

1. **Error Handling Preservation**: Invalid AI output still triggers appropriate error handling and retry logic
2. **Debugging Support**: Raw AI output remains accessible for debugging in all cases
3. **Performance**: No additional latency for valid outputs
4. **Backward Compatibility**: Existing valid pipelines continue to work

### Success Metrics

- **Before**: Valid AI output wrapped in error metadata, 100% false failure rate
- **After**: Valid AI output passes through correctly, 0% false failure rate
- **Execution Time**: Valid outputs should process in < 1 second (no retry loops)

## Constraints and Assumptions

### Constraints

1. Must maintain Wave's constitutional principles (fresh memory, security, etc.)
2. Must not break existing error handling for actually invalid outputs
3. Must preserve all debugging and auditing capabilities
4. Changes must be backward compatible with existing manifests

### Assumptions

1. AI personas are producing correctly formatted JSON that matches schemas
2. The issue is in Wave's validation pipeline, not AI output quality
3. Schema files are correctly defined and accessible
4. Error wrapping happens before schema validation

### Out of Scope

- Changes to AI persona behavior or prompts
- Modifications to schema definitions
- Alternative output formats (non-JSON)
- Performance optimizations beyond fixing the core issue

## Technical Context

### Wave Architecture Components Involved

1. **Pipeline Executor** (`internal/pipeline/executor.go`)
2. **Contract Validation** (`internal/contract/`)
3. **JSON Schema Validator** (`internal/contract/jsonschema.go`)
4. **Error Handling System**

### Key Files and Functions

- Validation pipeline logic
- Contract validation system
- JSON schema validation
- Error wrapping/metadata generation

### Integration Points

- Step execution and handover validation
- Error handling and retry mechanisms
- Audit logging and debugging output
- Progress reporting and monitoring

## Examples

### Example 1: Valid GitHub Issue Analysis

**Input**: AI produces valid GitHub issue analysis JSON
**Expected**: Pipeline validates and continues
**Current**: Pipeline wraps in error metadata and fails

### Example 2: Invalid JSON Syntax

**Input**: AI produces malformed JSON
**Expected**: Pipeline detects error, applies recovery, retries
**Current**: Same (should continue working)

### Example 3: Missing Required Fields

**Input**: AI produces valid JSON syntax but missing schema fields
**Expected**: Pipeline detects schema violation, provides helpful errors, retries
**Current**: Same (should continue working)

## Success Definition

The fix is successful when:

1. AI personas that produce schema-compliant JSON see their output validated correctly
2. Pipeline execution continues immediately for valid outputs
3. Error handling continues to work for actually invalid outputs
4. GitHub issue enhancer pipeline completes successfully with valid AI output
5. Microsoft TypeScript repository pipeline runs end-to-end without false failures

This specification addresses the core issue: Wave incorrectly treating correct AI output as validation failures due to premature error wrapping.