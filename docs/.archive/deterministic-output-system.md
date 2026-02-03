# Deterministic Output System

Wave's deterministic output system addresses the core challenge of achieving consistent, reliable AI responses from inherently non-deterministic AI systems. This system provides a comprehensive framework for enforcing structured output formats, validating responses, and continuously improving output reliability through feedback loops.

## Overview

The deterministic output system consists of several key components:

1. **Output Formatters** - Define and enforce response structures
2. **Response Validation** - Multi-layer validation with detailed feedback
3. **Template Engine** - Dynamic prompt enhancement for format guidance
4. **Processing Pipeline** - Response normalization and cleanup
5. **Metrics Collection** - Consistency tracking and trend analysis
6. **Feedback Loops** - Adaptive improvement mechanisms
7. **Wave Integration** - Seamless pipeline and contract integration

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   AI Response   │───▶│  Output Formatter │───▶│  Validated      │
│  (Raw Output)   │    │                  │    │  Response       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │   Validation     │
                    │   Pipeline       │
                    │                  │
                    │ ┌─────────────┐  │
                    │ │ Structure   │  │
                    │ │ Validation  │  │
                    │ └─────────────┘  │
                    │ ┌─────────────┐  │
                    │ │ Schema      │  │
                    │ │ Validation  │  │
                    │ └─────────────┘  │
                    │ ┌─────────────┐  │
                    │ │ Format      │  │
                    │ │ Validation  │  │
                    │ └─────────────┘  │
                    └──────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │ Post-Processing  │
                    │ & Normalization  │
                    └──────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │ Metrics &        │
                    │ Feedback         │
                    └──────────────────┘
```

## Key Features

### 1. Structured Output Formats

Define precise output structures that AI responses must follow:

```yaml
formats:
  github_issue:
    id: "github_issue"
    name: "GitHub Issue Format"
    description: "Structured format for GitHub issue creation"
    template:
      type: "json"
      schema: |
        {
          "type": "object",
          "properties": {
            "title": {"type": "string", "minLength": 10, "maxLength": 200},
            "body": {"type": "string", "minLength": 50},
            "labels": {"type": "array", "items": {"type": "string"}},
            "assignees": {"type": "array", "items": {"type": "string"}}
          },
          "required": ["title", "body"]
        }
      instructions: "Create a GitHub issue with clear title and detailed body including acceptance criteria"
      examples:
        - name: "Bug Report"
          output: |
            {
              "title": "Bug: Application crashes on startup",
              "body": "## Description\nDetailed bug description...\n\n## Acceptance Criteria\n- [ ] Fix applied\n- [ ] Tests pass",
              "labels": ["bug", "priority-high"]
            }
```

### 2. Multi-Layer Validation

Comprehensive validation with multiple validators:

```go
validation:
  strict_mode: true
  validators:
    - type: "json_schema"
      config:
        schema: "..."
      weight: 0.7
      fail_on_error: false
    - type: "format"
      config:
        format_type: "github_issue"
      weight: 0.3
      fail_on_error: false
  failure_policy: "retry"
  timeout_ms: 5000
```

### 3. Intelligent Retry System

Adaptive retry with enhanced prompts:

```go
retry:
  max_retries: 3
  backoff_ms: 1000
  backoff_multiplier: 1.5
  retry_conditions: ["validation_failed", "parse_error"]
  prompt_adjustment:
    add_examples: true
    increase_specificity: true
    highlight_errors: true
    custom_instructions: "Focus on exact format compliance"
```

### 4. Response Processing Pipeline

Automatic cleanup and normalization:

```go
post_processing:
  - type: "json_normalizer"
    config:
      extract_from_markdown: true
      fix_syntax_errors: true
    order: 1
    enabled: true
  - type: "field_extractor"
    config:
      extract_fields: ["title", "body", "labels"]
    order: 2
    enabled: true
```

## Usage Examples

### Basic Usage

```go
// Create formatter
factory := NewFormatterFactory()
formatter, err := factory.CreateDefaultFormatter()

// Register a format
jsonFormat := factory.CreateJSONFormat("user_data", "User Data Format",
    `{"type": "object", "properties": {"name": {"type": "string"}}}`)
formatter.RegisterFormat(jsonFormat)

// Process AI response
request := FormatRequest{
    FormatID:    "user_data",
    RawResponse: `{"name": "John Doe", "extra": "data"}`,
    PipelineID:  "test_pipeline",
    StepID:      "extract_user",
}

analysis, err := formatter.FormatResponse(context.Background(), request)
if analysis.IsValid {
    fmt.Printf("Valid response: %s\n", analysis.ParsedResponse)
}
```

### Pipeline Integration

```go
// Create system integrator
integrator, err := NewSystemIntegrator()

// Integrate with pipeline executor
err = integrator.IntegrateWithPipelineExecutor(executor, eventEmitter)

// Process step output
processedResult, err := integrator.ProcessStepOutput(
    ctx, step, adapterResult, pipelineID, retryCount)
```

### Custom Format Creation

```go
// Create custom analysis format
format := &OutputFormat{
    ID: "code_analysis",
    Name: "Code Analysis Results",
    Description: "Structure for code analysis outputs",
    Template: &Template{
        Type: "json",
        Schema: `{
            "type": "object",
            "properties": {
                "summary": {"type": "string"},
                "issues": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "severity": {"type": "string", "enum": ["low", "medium", "high"]},
                            "description": {"type": "string"},
                            "recommendation": {"type": "string"}
                        }
                    }
                },
                "confidence": {"type": "number", "minimum": 0, "maximum": 1}
            },
            "required": ["summary", "issues"]
        }`,
        Instructions: "Provide structured code analysis with severity levels and recommendations",
        Examples: []Example{
            {
                Name: "Security Analysis",
                Output: `{
                    "summary": "Found 3 security issues requiring attention",
                    "issues": [
                        {
                            "severity": "high",
                            "description": "SQL injection vulnerability in user input handling",
                            "recommendation": "Use parameterized queries"
                        }
                    ],
                    "confidence": 0.95
                }`,
            },
        },
    },
    Validation: &ValidationConfig{
        StrictMode: true,
        Validators: []ValidatorConfig{
            {
                Type: "json_schema",
                Config: map[string]interface{}{
                    "schema": "...", // Schema from above
                },
                FailOnError: true,
            },
        },
    },
}

formatter.RegisterFormat(format)
```

## Configuration

### Environment Variables

- `WAVE_DETERMINISTIC_ENABLED` - Enable deterministic output processing (default: true)
- `WAVE_DETERMINISTIC_CACHE_TTL` - Cache TTL in minutes (default: 60)
- `WAVE_DETERMINISTIC_MAX_RETRIES` - Default max retries (default: 3)
- `WAVE_DETERMINISTIC_METRICS_ENABLED` - Enable metrics collection (default: true)
- `WAVE_DETERMINISTIC_FORMATS_PATH` - Path to format definitions (default: .wave/formats)

### Manifest Configuration

```yaml
# wave.yaml
runtime:
  deterministic:
    enabled: true
    cache_ttl: "1h"
    metrics:
      enabled: true
      retention: "24h"
      alert_thresholds:
        success_rate: 0.85
        retry_rate: 0.25
    feedback:
      enabled: true
      loops:
        - id: "low_success_rate"
          trigger: "success_rate_below"
          threshold: 0.8
          action: "enhance_prompt"
          auto_apply: false
```

### Step-Level Configuration

```yaml
# Pipeline step with deterministic output
steps:
  - id: "create_github_issue"
    persona: "github_manager"
    exec:
      source: "Create a GitHub issue for the bug report: {{ input }}"
    handover:
      output_format:
        id: "github_issue"
      contract:
        type: "json_schema"
        schema_path: "schemas/github-issue.json"
        must_pass: true
        max_retries: 3
```

## Monitoring and Metrics

### Consistency Metrics

The system tracks several key metrics:

- **Success Rate** - Percentage of responses that pass validation
- **Retry Rate** - Percentage of responses requiring retries
- **Consistency Score** - Measure of output structure similarity
- **Processing Latency** - Time for validation and processing
- **Error Distribution** - Categorized error patterns

### Metrics Collection

```go
// Get metrics for a format
window := TimeWindow{
    Start: time.Now().Add(-24 * time.Hour),
    End:   time.Now(),
}

metrics, err := formatter.GetMetrics("github_issue", window)
fmt.Printf("Success Rate: %.2f%%\n", metrics.SuccessRate * 100)
fmt.Printf("Consistency Score: %.2f\n", metrics.ConsistencyScore)
```

### Alert Thresholds

Configure automatic alerts for quality degradation:

```go
// Set alert thresholds
metricsCollector.SetThreshold("github_issue", "success_rate", 0.85)
metricsCollector.SetThreshold("github_issue", "retry_rate", 0.25)

// Check for threshold violations
alerts, err := metricsCollector.CheckThresholds("github_issue")
for _, alert := range alerts {
    fmt.Printf("Alert: %s - %s\n", alert.Metric, alert.Message)
}
```

## Error Handling and Debugging

### Validation Error Analysis

```go
// Analyze validation errors
analyzer := NewDefaultErrorAnalyzer()
errors := []string{
    "JSON parsing failed: invalid syntax",
    "Missing required field: title",
    "Type mismatch: age should be number",
}

analysis, err := analyzer.AnalyzeErrors(errors, context)
fmt.Printf("Root causes: %v\n", analysis.RootCauses)
fmt.Printf("Recommendations: %v\n", analysis.RecommendedActions)
```

### Debugging Failed Responses

1. **Enable Debug Logging**
   ```bash
   export WAVE_DEBUG=true
   export WAVE_DETERMINISTIC_DEBUG=true
   ```

2. **Check Validation Details**
   ```go
   for _, result := range analysis.ValidationResults {
       if !result.Passed {
           fmt.Printf("Validator %s failed: %v\n", result.ValidatorType, result.Errors)
           fmt.Printf("Details: %v\n", result.Details)
       }
   }
   ```

3. **Review Processing Steps**
   ```go
   for _, step := range analysis.ProcessingSteps {
       fmt.Printf("Step %s: %s\n", step.StepType, step.Success)
       if len(step.Changes) > 0 {
           fmt.Printf("Changes: %v\n", step.Changes)
       }
   }
   ```

## Best Practices

### 1. Format Design

- **Be Specific** - Define precise schemas with clear constraints
- **Include Examples** - Provide multiple examples showing correct format
- **Use Clear Instructions** - Write unambiguous formatting guidance
- **Test Thoroughly** - Validate formats with various AI responses

### 2. Validation Strategy

- **Layer Validations** - Use multiple validators for comprehensive checking
- **Weight Appropriately** - Balance strict compliance with practical usability
- **Handle Edge Cases** - Account for AI response variability
- **Monitor Performance** - Track validation success rates

### 3. Retry Configuration

- **Reasonable Limits** - Don't retry indefinitely (3-5 attempts max)
- **Progressive Enhancement** - Increase specificity with each retry
- **Error-Specific Logic** - Tailor retry strategies to error types
- **Fallback Options** - Always have graceful degradation paths

### 4. Performance Optimization

- **Use Caching** - Cache parsed and validated responses
- **Batch Processing** - Process multiple responses together when possible
- **Async Validation** - Run non-critical validations asynchronously
- **Sampling** - Sample metrics collection for high-volume scenarios

## Troubleshooting

### Common Issues

1. **High Retry Rates**
   - Review prompt clarity and format instructions
   - Add more examples to format definitions
   - Check if schema requirements are too strict
   - Verify AI model capabilities match format complexity

2. **Low Success Rates**
   - Analyze error patterns for common issues
   - Simplify format requirements if possible
   - Improve prompt specificity
   - Consider model temperature adjustments

3. **Inconsistent Outputs**
   - Review format schema for ambiguities
   - Add structure validation rules
   - Check post-processing configuration
   - Monitor for model behavior changes

4. **Performance Issues**
   - Enable response caching
   - Optimize validation chains
   - Use appropriate sampling rates
   - Profile validation latencies

### Debug Commands

```bash
# Check system status
wave deterministic status

# List registered formats
wave deterministic formats list

# Validate format configuration
wave deterministic formats validate github_issue

# Test format with sample input
wave deterministic test --format github_issue --input sample.json

# View metrics for format
wave deterministic metrics --format github_issue --window 24h

# Generate compliance report
wave deterministic report --pipeline my_pipeline --window 7d
```

## API Reference

### Core Interfaces

- `OutputFormatter` - Main interface for format processing
- `Validator` - Response validation interface
- `Processor` - Response processing interface
- `TemplateEngine` - Prompt enhancement interface
- `MetricsCollector` - Metrics collection interface

### Key Types

- `OutputFormat` - Format definition structure
- `ResponseAnalysis` - Validation and processing results
- `ValidationResult` - Individual validator results
- `OutputConsistencyMetrics` - Performance metrics
- `FeedbackLoop` - Adaptive improvement configuration

See the [API documentation](./api/deterministic.md) for complete interface details.

## Integration Examples

### GitHub Workflow

```yaml
# .wave/pipelines/github-issue-creator.yaml
name: "GitHub Issue Creator"
description: "Creates structured GitHub issues from bug reports"

steps:
  - id: "analyze_bug"
    persona: "bug_analyst"
    exec:
      source: "Analyze this bug report and extract key information: {{ input }}"
    handover:
      output_format:
        id: "bug_analysis"
      contract:
        type: "json_schema"
        schema_path: "schemas/bug-analysis.json"

  - id: "create_issue"
    persona: "github_manager"
    memory:
      inject_artifacts:
        - step: "analyze_bug"
          artifact: "analysis"
          as: "bug_data.json"
    exec:
      source: "Create a GitHub issue from this analysis: {{ artifacts/bug_data.json }}"
    handover:
      output_format:
        id: "github_issue"
      contract:
        type: "json_schema"
        schema_path: "schemas/github-issue.json"
        must_pass: true
        max_retries: 3
```

### Code Review Pipeline

```yaml
# .wave/pipelines/code-review.yaml
steps:
  - id: "analyze_changes"
    persona: "senior_developer"
    exec:
      source: "Review these code changes for issues: {{ input }}"
    handover:
      output_format:
        id: "code_analysis"
      contract:
        type: "json_schema"
        schema_path: "schemas/code-review.json"

  - id: "create_review"
    persona: "reviewer"
    memory:
      inject_artifacts:
        - step: "analyze_changes"
          artifact: "analysis"
    exec:
      source: "Create detailed code review comments"
    handover:
      output_format:
        id: "github_pr_review"
```

The deterministic output system provides Wave with production-ready reliability for AI-generated content, ensuring consistent, valid outputs that integrate seamlessly with downstream systems and workflows.