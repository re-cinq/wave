# Wave Resilient Contract Validation System

## Overview

Wave's contract validation system has been enhanced to be more resilient to minor JSON formatting issues while maintaining strict validation requirements. The system now provides comprehensive error recovery, detailed error messages, and progressive validation modes.

## Key Features Implemented

### 1. Intelligent JSON Recovery

**Location**: `internal/contract/json_recovery.go`

The new `JSONRecoveryParser` provides three levels of JSON recovery:

- **Conservative Recovery**: Fixes safe, obvious issues like trailing commas and comments
- **Progressive Recovery**: Handles unquoted keys, single quotes, and missing commas
- **Aggressive Recovery**: Attempts reconstruction from fragments and infers missing structure

**Supported Recovery Operations**:
- Extract JSON from markdown code blocks (````json` ... ````)
- Remove single-line (`//`) and multi-line (`/* */`) comments
- Remove trailing commas before `}` or `]`
- Quote unquoted object keys (`{name: "value"}` → `{"name": "value"}`)
- Convert single quotes to double quotes (`'value'` → `"value"`)
- Extract JSON from surrounding text
- Add missing commas between properties/array elements
- Fix unbalanced braces and brackets
- Reconstruct objects from key-value pairs
- Infer missing object/array wrappers

### 2. Enhanced Error Messages

**Location**: `internal/contract/validation_error_formatter.go`

The new error formatter provides:
- **Detailed error analysis** with specific error type classification
- **Actionable suggestions** based on the type of validation failure
- **Common pitfalls** warnings to help avoid future issues
- **Examples** showing correct vs incorrect formats
- **Recovery context** indicating what fixes were applied

**Error Types Analyzed**:
- Missing required fields
- Type mismatches (string vs number, etc.)
- Enum violations
- Additional properties not allowed
- Array validation issues
- String format violations

### 3. Progressive Validation Modes

**Configuration Options**:
```yaml
contract:
  type: json_schema
  schema: "..."
  # Recovery settings
  allow_recovery: true                    # Enable/disable JSON recovery
  recovery_level: "progressive"           # "conservative", "progressive", "aggressive"
  warn_on_recovery: true                  # Generate warnings for recovery

  # Progressive validation
  progressive_validation: true            # Convert errors to warnings
  must_pass: false                       # Make validation non-blocking
```

**Validation Modes**:
- **Strict Mode**: `must_pass: true` - Validation failures block pipeline execution
- **Lenient Mode**: `must_pass: false` - Validation failures are logged but non-blocking
- **Progressive Mode**: `progressive_validation: true` - Errors become warnings

### 4. Format Normalization

**Location**: `internal/contract/json_cleaner.go`

Enhanced JSON cleaning capabilities:
- Integration with recovery parser for comprehensive fixing
- Format normalization with consistent indentation
- Structural validation beyond basic JSON parsing
- Extract JSON from AI-generated text with explanations

### 5. Comprehensive Testing

**Test Files**:
- `json_resilience_test.go` - Tests JSON recovery with malformed inputs
- `validation_error_formatter_test.go` - Tests enhanced error messages
- `resilient_validation_integration_test.go` - End-to-end workflow tests

**Test Coverage**:
- 15+ JSON recovery scenarios with various malformed inputs
- Error message quality and comprehensiveness
- Progressive validation behavior
- Real-world AI output scenarios
- Performance benchmarks for recovery operations

## Usage Examples

### Basic Configuration

```yaml
# Conservative recovery for production
pipeline:
  steps:
    - name: validate-output
      contract:
        type: json_schema
        schema_path: "schema.json"
        allow_recovery: true
        recovery_level: "conservative"
        must_pass: true
```

### Progressive Validation for Development

```yaml
# Progressive validation for development/testing
pipeline:
  steps:
    - name: validate-output
      contract:
        type: json_schema
        schema_path: "schema.json"
        allow_recovery: true
        recovery_level: "aggressive"
        progressive_validation: true
        must_pass: false
```

### Example Error Output

**Before (old system)**:
```
contract validation failed [json_schema]: does not match schema
```

**After (new system)**:
```
contract validation failed [json_schema]: Required fields are missing from the JSON output
  Details:
    - File: /workspace/artifact.json
    - JSON Recovery Applied: removed_trailing_commas, quoted_unquoted_keys
    - Schema Validation Errors:
      • - at '': missing property 'category'
    - Suggested Fixes:
      1. Check the schema to identify all required fields
      2. Ensure all mandatory properties are included in the output
      3. Verify that field names match the schema exactly (case-sensitive)
    - Common Issues to Check:
      ⚠ Field names with typos or incorrect casing
      ⚠ Fields with null values when null is not allowed
    - Example Fix:
      Required: {"name": "string", "category": "string"}
      Invalid: {"name": "example"}  // missing 'category'
      Valid:   {"name": "example", "category": "bug"}
```

## Performance Characteristics

- **Recovery Operations**: Typically complete in < 10ms for moderate JSON (< 100KB)
- **Memory Usage**: Bounded - original and recovered JSON kept in memory during processing
- **Fallback Strategy**: If recovery fails, original validation error is preserved
- **Caching**: Recovery strategies are stateless and reusable

## Security Considerations

- **Input Sanitization**: All recovery operations are safe and don't execute code
- **Path Validation**: File paths are validated and sanitized
- **Content Limits**: Recovery has built-in limits to prevent DoS attacks
- **Error Scrubbing**: Sensitive data is not leaked in error messages

## Configuration Best Practices

1. **Production**: Use `conservative` recovery with `must_pass: true`
2. **Development**: Use `progressive` or `aggressive` recovery with progressive validation
3. **CI/CD**: Use `progressive` recovery with `must_pass: true` for thorough validation
4. **Monitoring**: Enable `warn_on_recovery: true` to track recovery frequency

## Migration Guide

Existing configurations will continue to work. To enable the new features:

1. Add `allow_recovery: true` to enable JSON recovery
2. Set `recovery_level` based on your needs
3. Consider enabling `progressive_validation` for development workflows
4. Update error handling code to use the enhanced error details

## Future Enhancements

The resilient validation system provides a foundation for:
- Custom recovery strategies for domain-specific formats
- AI-assisted error correction suggestions
- Integration with linting and formatting tools
- Automatic schema inference from valid examples
- Real-time validation feedback in development environments