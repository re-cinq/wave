# Wave Retry Mechanisms for Malformed AI Outputs

Wave implements comprehensive retry mechanisms to handle malformed AI outputs gracefully, ensuring pipeline resilience while maintaining output quality.

## Overview

The retry system consists of several integrated components:

1. **Adaptive Retry Strategy** - Intelligent failure classification and retry decisions
2. **Output Format Correction** - Automatic correction of common format issues
3. **Progressive Prompt Enhancement** - Increasingly specific guidance on retry attempts
4. **Graceful Degradation** - Fallback strategies when all retries fail

## Architecture

### Core Components

#### 1. Output Format Corrector (`internal/adapter/retry.go`)

Provides multiple strategies for fixing malformed output:

- **Direct Validation** - Validates and reformats valid content
- **Markdown Extraction** - Extracts JSON/YAML from code blocks
- **Regex Extraction** - Finds JSON patterns in mixed content
- **Heuristic Recovery** - Fixes common syntax errors
- **Partial Recovery** - Reconstructs valid content from fragments
- **Template-based Recovery** - Uses fallback templates
- **Structured Error Reports** - Generates diagnostic reports

```go
corrector := adapter.NewOutputFormatCorrector(adapter.DefaultRetryConfig())
result, err := corrector.CorrectOutput(rawOutput, "json", attempt)
```

#### 2. Progressive Prompt Enhancer (`internal/pipeline/prompt_enhancement.go`)

Enhances prompts with retry-specific guidance:

- **Enhancement Levels**: Minimal → Moderate → Aggressive → Maximal
- **Error Analysis** - Pattern detection across attempts
- **Targeted Guidance** - Format-specific instructions
- **Enforcement Directives** - Mandatory requirements

```go
enhancer := NewPromptEnhancer(basePrompt, step)
enhanced := enhancer.EnhanceForRetry(attempt, maxRetries, lastError)
```

#### 3. Retry Policy Enforcer (`internal/pipeline/prompt_enhancement.go`)

Manages retry decisions and timing:

- **Adaptive Max Retries** - Adjusts retry count based on step complexity
- **Progressive Delays** - Exponential backoff with jitter
- **Error Classification** - Determines retryable vs non-retryable errors

```go
enforcer := NewRetryPolicyEnforcer()
shouldRetry := enforcer.ShouldRetry(attempt, maxRetries, err, step)
delay := enforcer.GetRetryDelay(attempt, baseDelay)
```

## Configuration

### Step-Level Configuration

```yaml
steps:
  - id: generate-spec
    persona: architect
    handover:
      max_retries: 5                    # Override default retry count
      contract:
        type: json_schema
        max_retries: 3                  # Contract-specific retries
        must_pass: true                 # Strict validation
      on_review_fail: retry             # Action on validation failure
```

### Retry Configuration Options

```yaml
# Default retry configuration
retry_config:
  max_attempts: 3
  base_delay: 1s
  max_delay: 30s
  backoff_multiplier: 2.0
  enable_jitter_delay: true

  # Output correction features
  enable_json_recovery: true
  enable_structure_recovery: true
  enable_content_extraction: true

  # Progressive enhancement
  progressive_enhancement: true
  incremental_guidance: true

  # Graceful degradation
  allow_partial_results: true
  generate_error_reports: true
```

## Retry Flow

### 1. Initial Attempt
- Execute step with base prompt
- Validate output against format/contract requirements
- If successful, continue pipeline

### 2. Format Correction (if validation fails)
- Apply output format correction strategies
- Re-validate corrected output
- If successful after correction, continue pipeline

### 3. Retry Decision
- Classify error type (retryable vs non-retryable)
- Check attempt count against maximum
- Apply retry policy logic

### 4. Enhanced Retry Attempt
- Generate enhanced prompt with:
  - Retry attempt header with urgency level
  - Error analysis and pattern detection
  - Specific guidance for the error type
  - Progressive instructions based on attempt number
  - Enforcement directives
- Apply progressive delay before execution
- Execute with enhanced prompt

### 5. Graceful Degradation (after max retries)
- Extract partial valid content if possible
- Generate structured error report
- Apply fallback templates
- Continue pipeline with degraded output if allowed

## Error Classification

### Retryable Errors
- JSON parsing/syntax errors
- Schema validation failures
- Missing required fields
- Format compliance issues
- Contract validation failures

### Non-Retryable Errors
- Network timeouts
- Permission denied
- File not found
- System resource exhaustion
- Memory issues

## Progressive Prompt Enhancement

### Enhancement Levels

#### Minimal (Early Attempts)
- Basic reminders about format requirements
- Simple guidance text

#### Moderate (Middle Attempts)
- Important reminders section
- Specific requirement lists
- Validation instructions

#### Aggressive (Later Attempts)
- Critical requirements with emphasis
- Strict adherence warnings
- Step-by-step validation process

#### Maximal (Final Attempts)
- Emergency protocol activation
- Pipeline failure warnings
- Detailed step-by-step approach
- Mandatory enforcement directives

### Guidance Types

#### JSON-Specific Guidance
```
JSON OUTPUT REQUIREMENTS:
• Start your response with { or [
• End your response with } or ]
• Use double quotes for all strings
• Do NOT include trailing commas
• Do NOT wrap in markdown code blocks
```

#### Schema Validation Guidance
```
SCHEMA VALIDATION REQUIREMENTS:
• Every field specified in the schema MUST be present
• Field types must match exactly (string, number, boolean, array, object)
• Required fields cannot be null or empty
• Enum values must be exactly as specified
```

## Output Format Correction

### Correction Strategies (Applied in Order)

1. **Direct JSON Validation** - Parse and reformat valid JSON
2. **Markdown Code Block Extraction** - Extract JSON from ```json blocks
3. **Regex JSON Extraction** - Find JSON patterns using regex
4. **Heuristic JSON Recovery** - Fix common syntax issues:
   - Remove trailing commas
   - Fix single quotes to double quotes
   - Add missing quotes to object keys
   - Remove explanatory text prefixes/suffixes
5. **Partial JSON Recovery** - Reconstruct from valid fragments
6. **Template-Based Recovery** - Use format-specific templates
7. **Structured Error Report** - Generate diagnostic JSON

### Correction Example

Input (malformed):
```
Here's the result you requested:
{
  'name': 'test',
  'values': [1, 2, 3,],
}
```

Corrected output:
```json
{
  "name": "test",
  "values": [1, 2, 3]
}
```

## Integration with Pipeline

### Event Emission

The retry system emits detailed progress events:

```go
// Retry attempt events
step_retry: "Retry attempt 2/3"
step_retry_needed: "Output validation failed"
step_retry_delay: "Waiting 2s before retry"

// Format correction events
step_format_correction: "Attempting output format correction"
step_format_corrected: "Output corrected using heuristic_json_recovery"

// Degradation events
step_degraded: "Applied graceful degradation after 3 attempts"
step_retry_exhausted: "Failed after 3 attempts"
```

### State Management

- Retry attempts are tracked in pipeline execution state
- Failed steps are recorded with failure reasons
- Successful corrections are logged for debugging
- Degraded outputs are marked for review

## Best Practices

### For Pipeline Authors

1. **Set Appropriate Retry Limits**
   - Simple steps: 1-2 retries
   - Complex JSON generation: 3-5 retries
   - Critical contract validation: 5+ retries

2. **Design Clear Contracts**
   - Use specific JSON schemas
   - Provide examples in contract documentation
   - Set appropriate `must_pass` flags

3. **Monitor Retry Patterns**
   - Review retry logs to identify common failures
   - Adjust persona instructions based on retry patterns
   - Update schemas to be more forgiving where appropriate

### For Persona Authors

1. **Clear Format Instructions**
   - Specify exact output format requirements
   - Provide examples of valid output
   - Include format validation in system prompts

2. **Contract Awareness**
   - Reference contract requirements in persona instructions
   - Include schema validation guidance
   - Emphasize format compliance

## Monitoring and Debugging

### Logs and Events

- Retry attempts with enhanced prompts are logged
- Format correction results are tracked
- Error patterns are analyzed and reported
- Degradation strategies are documented

### Metrics

- Retry success rates by step and persona
- Format correction effectiveness by strategy
- Common failure patterns and trends
- Pipeline reliability improvements

### Debugging Tools

- Enhanced prompt inspection
- Correction strategy analysis
- Error pattern visualization
- Retry timing analysis

## Performance Considerations

### Efficiency Optimizations

- Early validation prevents unnecessary retries
- Format correction reduces retry count
- Adaptive delays prevent resource exhaustion
- Pattern detection improves guidance quality

### Resource Management

- Retry delays prevent overwhelming external APIs
- Maximum retry limits prevent infinite loops
- Graceful degradation maintains pipeline flow
- Memory usage is bounded during retry attempts

## Future Enhancements

### Planned Features

1. **AI-Assisted Recovery** - Use AI to fix malformed output
2. **Learning-Based Enhancement** - Improve prompts based on historical failures
3. **Cross-Step Pattern Detection** - Learn from failures across different steps
4. **Dynamic Retry Policies** - Adjust retry behavior based on step performance
5. **Advanced Correction Strategies** - Domain-specific output correction

### Research Areas

- Machine learning for error prediction
- Natural language prompt optimization
- Automated schema evolution
- Cross-pipeline failure analysis

## Conclusion

Wave's comprehensive retry mechanisms provide robust handling of malformed AI outputs while maintaining pipeline reliability and output quality. The multi-layered approach ensures that temporary AI formatting issues don't block entire workflows, while progressive enhancement helps AI models learn to produce better output over time.

The system balances resilience with performance, providing multiple correction strategies and graceful degradation paths while preventing resource exhaustion through intelligent retry policies and adaptive timing.