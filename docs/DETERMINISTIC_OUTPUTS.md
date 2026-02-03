# Deterministic Outputs in Wave

## Overview

Wave's deterministic output system ensures that AI-powered pipelines produce reliable, consistent, production-ready results every time. This system bridges the gap between stochastic AI behavior and the need for predictable, high-quality deliverables.

## Problem Statement

AI systems are inherently non-deterministic, but Wave pipelines need to guarantee:
- **Specific format compliance** - Outputs must match exact schemas and templates
- **Production quality** - All deliverables must be complete, validated, and ready to use
- **Consistent structure** - Same inputs should produce similarly structured outputs
- **Automatic recovery** - Failed attempts should be automatically corrected

## Core Components

### 1. Output Validation System

Wave uses a multi-layer validation approach:

```yaml
handover:
  contract:
    type: json_schema
    schema_path: .wave/contracts/strict-output.schema.json
    validate: true
    must_pass: true
    max_retries: 3
  quality_gates:
    - type: required_fields
      required: true
    - type: format
      required: true
    - type: verification
      required: true
```

**Validation Layers:**

1. **Schema Validation** - JSON Schema enforcement with strict type checking
2. **Format Validation** - Production-ready format requirements (title length, section presence, etc.)
3. **Quality Gates** - Content completeness, placeholder detection, cross-reference validation
4. **Verification** - Automated testing of generated code, link validation, file existence

### 2. Structured Templates

Templates enforce exact output structures:

```go
// Templates define the exact structure AI must produce
type TemplateConfig struct {
    Type        string                 // "json", "markdown", "yaml"
    Required    []string               // Required fields/sections
    Constraints map[string]interface{} // Field-specific constraints
}
```

**Template Types:**

- **JSON Templates** - Structured data with required fields and constraints
- **Markdown Templates** - Documents with required sections and formatting
- **YAML Templates** - Configuration files with validation

### 3. Adaptive Retry Strategy

When validation fails, Wave provides targeted repair guidance:

```go
// Retry with intelligent feedback
result, err := ValidateWithAdaptiveRetry(cfg, workspacePath)

// Failure types are classified:
// - schema_mismatch: Wrong types or missing fields
// - format_error: Invalid JSON/YAML syntax
// - missing_content: Incomplete or placeholder content
// - quality_gate: Failed quality checks
```

**Retry Mechanism:**
1. Detect failure type
2. Generate specific repair prompt
3. Exponential backoff with jitter
4. Track failure progression
5. Provide actionable guidance

### 4. Format Validators

Specialized validators ensure production-ready outputs:

```go
// Format-specific validation
validator := &FormatValidator{}

// Validates:
// - GitHub issues: Title format, required sections, no placeholders
// - Pull requests: Conventional commits, test evidence, issue references
// - Code implementations: Compilation, test passing, completeness
// - Analysis outputs: Comprehensive findings, recommendations
```

### 5. Verification Framework

Automated checks for generated outputs:

```yaml
quality_gates:
  - type: verification
    parameters:
      rules:
        - type: code_compilation
          command: "go build ./..."
        - type: test_execution
          command: "go test ./..."
        - type: link_validation
          target: artifact.json
        - type: cross_reference
          target: artifact.json
        - type: file_existence
          files: ["required.go", "required_test.go"]
```

### 6. Rollback Mechanism

State management for failed pipelines:

```go
// Create checkpoints for rollback
manager := NewRollbackManager(stateDir)
checkpoint, _ := manager.CreateCheckpoint(pipelineID, stepID, workspace, artifacts)

// Log operations that can be undone
manager.LogOperation(pipelineID, RollbackOperation{
    Type: "file_created",
    Target: "/path/to/file.go",
    CanRevert: true,
})

// Rollback on failure
manager.Rollback(pipelineID, checkpoint)

// Get rollback plan
plan, _ := manager.GetRollbackPlan(pipelineID)
```

## Usage Examples

### Example 1: Deterministic GitHub Issue Creation

```yaml
- id: create-issue
  persona: github-issue-creator
  exec:
    type: prompt
    source: |
      Create a GitHub issue.

      CRITICAL: Output ONLY valid JSON. No markdown, no explanations.

      Requirements:
      - Title: 10-200 characters, no placeholders
      - Body: Minimum 100 characters with sections
      - Labels: At least one from allowed list

      Output to artifact.json:
      {
        "title": "feat: Implement feature name",
        "body": "## Description\n...\n## Acceptance Criteria\n...",
        "labels": ["enhancement"]
      }
  handover:
    contract:
      type: json_schema
      schema_path: .wave/contracts/deterministic-github-issue.schema.json
      validate: true
      must_pass: true
      max_retries: 3
```

### Example 2: Deterministic PR Creation

```yaml
- id: create-pr
  persona: github-pr-creator
  exec:
    type: prompt
    source: |
      Create pull request.

      STRICT FORMAT:
      - Title: Conventional commit format (feat/fix: description)
      - Tests must pass
      - Must reference issues (Closes #123)

      Output ONLY valid JSON matching schema.
  handover:
    contract:
      type: json_schema
      schema_path: .wave/contracts/deterministic-pr-creation.schema.json
      validate: true
      must_pass: true
      max_retries: 3
    quality_gates:
      - type: verification
        parameters:
          rules:
            - type: cross_reference
            - type: link_validation
```

### Example 3: Code Implementation with Verification

```yaml
- id: implement
  persona: craftsman
  exec:
    type: prompt
    source: |
      Implement feature.

      CRITICAL:
      - Code MUST compile
      - All tests MUST pass
      - Add new tests
  handover:
    contract:
      type: json_schema
      schema_path: .wave/contracts/implementation-results.schema.json
      validate: true
      must_pass: true
    quality_gates:
      - type: verification
        required: true
        parameters:
          rules:
            - type: code_compilation
              command: "go build ./..."
            - type: test_execution
              command: "go test ./..."
```

## Best Practices

### 1. Always Use Strict Schemas

```json
{
  "type": "object",
  "required": ["field1", "field2"],
  "properties": {
    "field1": {
      "type": "string",
      "minLength": 10,
      "maxLength": 200,
      "pattern": "^[^TODO|PLACEHOLDER].*$"
    }
  }
}
```

### 2. Set must_pass and max_retries

```yaml
contract:
  validate: true
  must_pass: true    # Block pipeline if validation fails
  max_retries: 3     # Give AI 3 attempts to fix issues
```

### 3. Use Multiple Quality Gates

```yaml
quality_gates:
  - type: required_fields
  - type: content_completeness
  - type: verification
```

### 4. Provide Clear Format Requirements in Prompts

```yaml
source: |
  CRITICAL: Output ONLY valid JSON. No markdown, no explanations.

  Requirements:
  - Field X: 10-200 characters
  - Field Y: Must include sections A, B, C
  - No placeholder text (TODO, FIXME, etc.)
```

### 5. Enable Verification for Production Pipelines

```yaml
quality_gates:
  - type: verification
    required: true
    parameters:
      rules:
        - type: code_compilation
        - type: test_execution
        - type: link_validation
```

## Configuration Reference

### Contract Types

- `json_schema` - JSON Schema validation (RFC 7159)
- `typescript_interface` - TypeScript interface validation
- `test_suite` - Test execution validation
- `markdown_spec` - Markdown structure validation
- `template` - Structured template validation
- `format` - Production format validation

### Quality Gate Types

- `required_fields` - Check for required JSON fields
- `content_completeness` - Ensure content meets quality thresholds
- `json_structure` - Validate JSON formatting
- `markdown_structure` - Check markdown heading hierarchy
- `link_validation` - Verify all links are valid
- `verification` - Run automated verification checks

### Verification Rule Types

- `code_compilation` - Verify code compiles
- `test_execution` - Run test suite
- `link_validation` - Check URLs and references
- `cross_reference` - Validate cross-references
- `file_existence` - Ensure required files exist

## Error Handling

### Validation Errors

```go
type ValidationError struct {
    ContractType string   // Type of contract that failed
    Message      string   // Error summary
    Details      []string // Specific violations
    Retryable    bool     // Can this be retried?
    Attempt      int      // Current attempt number
    MaxRetries   int      // Maximum retries allowed
}
```

### Failure Classification

Wave automatically classifies failures:

- **schema_mismatch** - Wrong field types or missing required fields
- **format_error** - Invalid JSON/YAML syntax
- **missing_content** - Empty or placeholder content
- **quality_gate** - Failed quality checks
- **structure** - Incorrect document structure

### Repair Guidance

On failure, Wave provides specific guidance:

```
VALIDATION FAILURE - RETRY REQUIRED

Attempt 2 of 3
Failure Type: format_error

CRITICAL REQUIREMENTS:
1. Output ONLY valid JSON - no markdown code blocks
2. Start with { or [ and end with } or ]
3. Do NOT include explanatory text
4. Ensure all strings are properly quoted

Specific Suggestions:
1. Remove markdown code blocks around JSON
2. Verify all brackets are balanced
```

## Performance Considerations

- **Validation overhead**: 10-50ms per validation
- **Retry delays**: Exponential backoff (1s, 2s, 4s, ...)
- **Checkpoint creation**: < 100ms per checkpoint
- **Rollback time**: Depends on operation count, typically < 1s

## Monitoring and Debugging

### Debug Mode

```bash
wave run pipeline --debug
```

Shows:
- Validation results
- Retry attempts
- Failure classifications
- Repair guidance

### Logs

```
.wave/traces/
  pipeline-id/
    step-id/
      validation.log
      retry-1.log
      retry-2.log
```

## Migration Guide

### From Basic Contracts to Deterministic Outputs

**Before:**
```yaml
handover:
  contract:
    type: json_schema
    schema_path: schema.json
```

**After:**
```yaml
handover:
  contract:
    type: json_schema
    schema_path: deterministic-schema.json
    validate: true
    must_pass: true
    max_retries: 3
  quality_gates:
    - type: format
      required: true
    - type: verification
      required: true
```

## Troubleshooting

### Issue: AI keeps failing validation

**Solution:**
- Check schema is not too strict
- Verify prompt provides clear format requirements
- Increase max_retries
- Review failure types in logs

### Issue: Validation is too slow

**Solution:**
- Reduce number of quality gates
- Use simpler schemas
- Consider caching validation results

### Issue: False negatives

**Solution:**
- Adjust quality gate thresholds
- Review pattern matching rules
- Use more lenient constraints

## Future Enhancements

- Machine learning for failure prediction
- Automatic schema generation from examples
- Visual validation result display
- Integration with external validators
- Parallel validation for faster results
