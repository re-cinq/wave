# Creating Wave Workflows

Transform your AI automation from ad-hoc prompts to reproducible, shareable infrastructure code. This guide shows you how to create declarative Wave workflows that your team can version control, share, and trust.

## The Declarative Approach

Wave workflows are **configuration files** that declare what you want to accomplish, not how to accomplish it. Think `docker-compose.yml` for AI workflows.

### Complete Example First

Here's a full workflow that implements automated code review:

```yaml
# code-review.yaml
apiVersion: v1
kind: WavePipeline
metadata:
  name: automated-code-review
  description: "Comprehensive code review with security and quality analysis"
  version: "1.2.0"
  author: "engineering-team"

input:
  type: git_diff
  required: true

steps:
  - id: analyze-changes
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: ./
          target: /codebase
          mode: readonly
    exec:
      type: prompt
      source: |
        Analyze the code changes in this diff for:
        1. Complexity and risk assessment
        2. Affected components and dependencies
        3. Test coverage requirements
        4. Performance implications

        Input: {{ input }}

        Output as structured JSON with keys:
        complexity_score, risk_level, affected_components, test_requirements
    output_artifacts:
      - name: change-analysis
        path: output/analysis.json
        type: json
    handover:
      contract:
        type: json_schema
        schema: .wave/contracts/change-analysis.schema.json
        source: output/analysis.json
        on_failure: retry
        max_retries: 2

  - id: security-review
    persona: auditor
    dependencies: [analyze-changes]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze-changes
          artifact: change-analysis
          as: change_context
    exec:
      type: prompt
      source: |
        Perform security review of code changes:

        Change Context: {{ artifacts.change_context }}
        Code Diff: {{ input }}

        Check for:
        - SQL injection vulnerabilities
        - XSS attack vectors
        - Authentication/authorization gaps
        - Sensitive data exposure
        - Input validation issues

        Output findings as structured security report.
    output_artifacts:
      - name: security-findings
        path: output/security-report.json
        type: json
    handover:
      contract:
        type: json_schema
        schema: .wave/contracts/security-findings.schema.json
        source: output/security-report.json

  - id: generate-review
    persona: reviewer
    dependencies: [analyze-changes, security-review]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze-changes
          artifact: change-analysis
        - step: security-review
          artifact: security-findings
    exec:
      type: prompt
      source: |
        Generate comprehensive code review comments based on:

        Analysis: {{ artifacts.change_analysis }}
        Security: {{ artifacts.security_findings }}

        Create review that is:
        - Constructive and specific
        - Includes code suggestions where appropriate
        - Prioritizes issues by severity
        - Follows team coding standards
    output_artifacts:
      - name: review-comments
        path: output/review.md
        type: markdown
    handover:
      contract:
        type: test_suite
        command: python .wave/validators/review-quality.py output/review.md
        must_pass: true

event_handlers:
  on_completion:
    - type: github_comment
      target: "{{ input.pr_number }}"
      source: output/review.md
  on_failure:
    - type: slack_notification
      channel: "#engineering"
      message: "Code review workflow failed for {{ input.pr_number }}"

contracts:
  change-analysis.schema.json: |
    {
      "$schema": "http://json-schema.org/draft-07/schema#",
      "type": "object",
      "required": ["complexity_score", "risk_level", "affected_components"],
      "properties": {
        "complexity_score": {"type": "integer", "minimum": 1, "maximum": 10},
        "risk_level": {"enum": ["low", "medium", "high", "critical"]},
        "affected_components": {"type": "array", "items": {"type": "string"}}
      }
    }

  security-findings.schema.json: |
    {
      "$schema": "http://json-schema.org/draft-07/schema#",
      "type": "object",
      "required": ["findings", "risk_score"],
      "properties": {
        "findings": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["type", "severity", "description"],
            "properties": {
              "type": {"type": "string"},
              "severity": {"enum": ["low", "medium", "high", "critical"]},
              "description": {"type": "string"},
              "file": {"type": "string"},
              "line": {"type": "integer"}
            }
          }
        },
        "risk_score": {"type": "integer", "minimum": 0, "maximum": 100}
      }
    }
```

## Understanding the Structure

### 1. Metadata Section
Every workflow starts with metadata:

```yaml
apiVersion: v1
kind: WavePipeline
metadata:
  name: my-workflow
  description: "Clear description of what this accomplishes"
  version: "1.0.0"
  author: "team-name"
```

**Why This Matters:**
- **Name**: Used for `wave run my-workflow` commands
- **Description**: Helps team members understand purpose
- **Version**: Enables workflow evolution and compatibility
- **Author**: Ownership and maintenance responsibility

### 2. Input Declaration
Define what data your workflow needs:

```yaml
input:
  type: file_path | git_diff | text | json
  required: true | false
  validation:
    schema: input-schema.json
```

**Input Types:**
- `file_path`: Path to file or directory
- `git_diff`: Git diff output (for code review workflows)
- `text`: Free-form text input
- `json`: Structured JSON data
- `custom`: Your own type with validation schema

### 3. Step Definitions
Each step is an isolated AI operation:

```yaml
steps:
  - id: unique-step-name
    persona: navigator | auditor | craftsman | reviewer
    dependencies: [list-of-step-ids]  # Optional
    memory:
      strategy: fresh | context | relay
      inject_artifacts: [...]  # Optional
    workspace:
      mount: [...]  # File access permissions
    exec:
      type: prompt | script
      source: "Task description with variables"
    output_artifacts: [...]
    handover:
      contract: ...  # Quality validation
```

### 4. Contract Validation
Ensure each step produces exactly what you need:

```yaml
handover:
  contract:
    type: json_schema | typescript_interface | test_suite
    schema: path/to/schema.json
    source: output/filename
    on_failure: retry | halt | continue
    max_retries: 3
```

## Creating Your First Workflow

### Step 1: Define the Goal
Start with a clear statement of what you want to accomplish:

```yaml
metadata:
  name: api-documentation
  description: "Generate comprehensive API documentation from codebase"
```

### Step 2: Identify the Steps
Break down the process into discrete operations:

1. **Analyze** the codebase to find API endpoints
2. **Extract** route definitions and parameter information
3. **Generate** OpenAPI specification
4. **Create** human-readable documentation
5. **Validate** that documentation is complete and accurate

### Step 3: Choose Personas
Match each step to the appropriate AI persona:

- **Navigator**: Analysis and discovery tasks
- **Auditor**: Validation and security checks
- **Craftsman**: Code generation and modification
- **Reviewer**: Quality assessment and documentation

### Step 4: Define Contracts
Specify exactly what each step must produce:

```yaml
# For the API analysis step
handover:
  contract:
    type: json_schema
    schema: |
      {
        "type": "object",
        "required": ["endpoints", "models"],
        "properties": {
          "endpoints": {
            "type": "array",
            "items": {
              "type": "object",
              "required": ["method", "path", "description"],
              "properties": {
                "method": {"enum": ["GET", "POST", "PUT", "DELETE"]},
                "path": {"type": "string"},
                "description": {"type": "string"},
                "parameters": {"type": "array"}
              }
            }
          }
        }
      }
```

### Step 5: Test and Iterate
Run your workflow and refine based on results:

```bash
# Test with sample input
wave run api-documentation --input ./src/api

# Debug if contracts fail
wave run api-documentation --input ./src/api --debug

# Validate schema correctness
wave validate workflow api-documentation.yaml
```

## Advanced Workflow Patterns

### Contract Modification Examples

Sometimes you need to change output formats. Here's how to modify contracts:

**Before: Simple JSON output**
```yaml
handover:
  contract:
    type: json_schema
    schema: simple-schema.json
```

**After: Rich TypeScript interfaces**
```yaml
handover:
  contract:
    type: typescript_interface
    source: output/generated-types.ts
    validate: true
  additional_contracts:
    - type: json_schema
      schema: enhanced-schema.json
      source: output/metadata.json
```

**Migration Strategy:**
1. Add new contract alongside existing one
2. Update downstream steps to handle new format
3. Remove old contract once migration complete
4. Version your workflow to track changes

### Conditional Steps

Execute different steps based on input or previous results:

```yaml
steps:
  - id: analyze-language
    persona: navigator
    # ... detect programming language

  - id: java-specific
    persona: craftsman
    dependencies: [analyze-language]
    condition: "{{ steps.analyze-language.output.language == 'java' }}"
    # ... Java-specific processing

  - id: javascript-specific
    persona: craftsman
    dependencies: [analyze-language]
    condition: "{{ steps.analyze-language.output.language == 'javascript' }}"
    # ... JavaScript-specific processing
```

### Error Handling and Retry Logic

Build resilient workflows that handle failures gracefully:

```yaml
steps:
  - id: external-api-call
    persona: navigator
    exec:
      type: prompt
      source: "Fetch data from external API: {{ input.api_endpoint }}"
    handover:
      contract:
        type: json_schema
        schema: api-response.schema.json
        on_failure: retry
        max_retries: 5
        backoff: exponential
        escalation:
          # After max retries, try different approach
          fallback_persona: senior-navigator
          manual_review: true

  - id: process-data
    dependencies: [external-api-call]
    persona: craftsman
    # This step only runs if external-api-call succeeds
```

### Parallel Processing

Run independent steps simultaneously for better performance:

```yaml
steps:
  - id: extract-functions
    persona: navigator
    # ... extract all functions from codebase

  # These three steps can run in parallel
  - id: analyze-complexity
    dependencies: [extract-functions]
    persona: auditor
    parallel: true

  - id: generate-tests
    dependencies: [extract-functions]
    persona: craftsman
    parallel: true

  - id: create-documentation
    dependencies: [extract-functions]
    persona: reviewer
    parallel: true

  # This step waits for all parallel steps to complete
  - id: final-report
    dependencies: [analyze-complexity, generate-tests, create-documentation]
    persona: reviewer
```

### Dynamic Contract Generation

Generate contracts based on the actual data structure discovered:

```yaml
steps:
  - id: schema-discovery
    persona: navigator
    exec:
      type: prompt
      source: |
        Analyze this API and generate a JSON schema that describes its responses.
        Input: {{ input }}
    output_artifacts:
      - name: discovered-schema
        path: output/dynamic-schema.json
        type: json

  - id: validate-responses
    dependencies: [schema-discovery]
    persona: auditor
    handover:
      contract:
        type: json_schema
        schema: "{{ steps.schema-discovery.outputs.discovered-schema }}"
        # Use the dynamically generated schema
```

## Workflow Composition Patterns

### Modular Workflows

Break large workflows into reusable components:

```yaml
# base-analysis.yaml
metadata:
  name: base-analysis
  description: "Common analysis steps for any codebase"

steps:
  - id: scan-files
    # ... file scanning logic

  - id: extract-metadata
    # ... metadata extraction

# Reuse in other workflows
# feature-development.yaml
metadata:
  name: feature-development
  imports: [base-analysis]

steps:
  - id: analyze-codebase
    workflow: base-analysis

  - id: implement-feature
    dependencies: [analyze-codebase]
    # ... feature implementation
```

### Pipeline Templates

Create templates for common workflow patterns:

```yaml
# Template: code-review-template.yaml
metadata:
  name: code-review-template
  template: true
  parameters:
    - name: language
      type: string
      required: true
    - name: security_level
      type: string
      default: "standard"

steps:
  - id: language-analysis
    persona: navigator
    exec:
      source: "Analyze {{ parameters.language }} code for review"

# Usage: my-review.yaml
metadata:
  name: my-java-review
  extends: code-review-template
  parameters:
    language: "java"
    security_level: "high"
```

### Workflow Libraries

Organize related workflows into reusable libraries:

```yaml
# workflows/security/owasp-audit.yaml
metadata:
  name: owasp-security-audit
  library: security
  tags: [security, audit, owasp]

# workflows/testing/integration-tests.yaml
metadata:
  name: integration-test-generator
  library: testing
  tags: [testing, integration, automation]

# Use library workflows
metadata:
  name: comprehensive-review

steps:
  - id: security-check
    workflow: security/owasp-audit

  - id: test-generation
    workflow: testing/integration-tests
```

## Testing Your Workflows

### Validation Commands

```bash
# Validate workflow syntax
wave validate workflow my-workflow.yaml

# Test with sample data (dry run)
wave run my-workflow --input sample-data.json --dry-run

# Run with debug output
wave run my-workflow --input sample-data.json --debug

# Validate specific contracts
wave validate contract .wave/contracts/my-schema.json sample-output.json
```

### Contract Testing

Create test suites for your contracts:

```yaml
# .wave/tests/contract-tests.yaml
tests:
  - name: "API analysis contract validation"
    contract: .wave/contracts/api-analysis.schema.json
    valid_examples:
      - test-data/valid-api-analysis.json
      - test-data/another-valid-example.json
    invalid_examples:
      - test-data/missing-required-field.json
      - test-data/wrong-data-type.json

  - name: "Security findings contract validation"
    contract: .wave/contracts/security-findings.schema.json
    valid_examples:
      - test-data/no-findings.json
      - test-data/critical-vulnerabilities.json
```

Run contract tests:
```bash
wave test contracts .wave/tests/contract-tests.yaml
```

### Integration Testing

Test complete workflows end-to-end:

```bash
# Test with known good input, verify expected output
wave run code-review --input test-fixtures/sample-pr-diff.txt
diff expected-output/review.md /tmp/wave/*/generate-review/output/review.md

# Regression testing - ensure changes don't break existing workflows
wave test regression workflows/ --baseline test-results/baseline/
```

## Best Practices

### 1. Start Simple, Add Complexity Gradually

**Good First Workflow:**
```yaml
steps:
  - id: analyze
    persona: navigator
    exec:
      source: "Analyze this code: {{ input }}"
    output_artifacts:
      - name: analysis
        path: output/analysis.txt
        type: text
```

**After It Works:**
```yaml
steps:
  - id: analyze
    persona: navigator
    exec:
      source: "Analyze this code: {{ input }}"
    output_artifacts:
      - name: analysis
        path: output/analysis.json
        type: json
    handover:
      contract:
        type: json_schema
        schema: analysis.schema.json
```

### 2. Design Contracts First

Before writing workflow steps, define what "success" looks like:

```yaml
# Define the contract first
contracts:
  user-story.schema.json: |
    {
      "type": "object",
      "required": ["title", "description", "acceptance_criteria"],
      "properties": {
        "title": {"type": "string", "minLength": 10},
        "description": {"type": "string", "minLength": 50},
        "acceptance_criteria": {
          "type": "array",
          "minItems": 3,
          "items": {"type": "string"}
        }
      }
    }

# Then write steps that produce this output
steps:
  - id: create-user-story
    handover:
      contract:
        type: json_schema
        schema: user-story.schema.json
```

### 3. Use Meaningful Step and Artifact Names

**Bad:**
```yaml
steps:
  - id: step1
    output_artifacts:
      - name: output
        path: out.json
```

**Good:**
```yaml
steps:
  - id: extract-requirements
    output_artifacts:
      - name: requirements-list
        path: output/extracted-requirements.json
```

### 4. Version Your Workflows

```yaml
metadata:
  name: api-documentation
  version: "2.1.0"
  # Document what changed
  changelog:
    - "2.1.0: Added TypeScript interface generation"
    - "2.0.0: Switched to OpenAPI 3.0 format"
    - "1.0.0: Initial version"
```

### 5. Document Your Personas

Create persona documentation that explains which to use when:

```yaml
# .wave/personas/README.md
# Persona Guide

## Navigator
**Use for:** Analysis, discovery, research, planning
**Good at:** Understanding codebases, identifying patterns, extracting information
**Avoid for:** Code generation, file modification

## Craftsman
**Use for:** Code generation, file creation, implementation
**Good at:** Writing code, creating files, following specifications
**Avoid for:** High-level analysis, security reviews

## Auditor
**Use for:** Security reviews, compliance checks, validation
**Good at:** Finding vulnerabilities, checking against standards
**Avoid for:** Creative tasks, code generation

## Reviewer
**Use for:** Quality assessment, documentation, summarization
**Good at:** Synthesizing information, writing human-readable content
**Avoid for:** Technical implementation, security analysis
```

## Common Patterns Library

### Code Review Workflow
```yaml
# Complete PR review with security analysis
metadata:
  name: comprehensive-code-review
steps:
  - id: analyze-changes
    persona: navigator
  - id: security-review
    persona: auditor
  - id: generate-feedback
    persona: reviewer
```

### Documentation Generation
```yaml
# API documentation from codebase
metadata:
  name: api-docs-generator
steps:
  - id: extract-endpoints
    persona: navigator
  - id: generate-openapi
    persona: craftsman
  - id: create-readme
    persona: reviewer
```

### Test Generation
```yaml
# Automated test creation
metadata:
  name: test-generator
steps:
  - id: analyze-functions
    persona: navigator
  - id: generate-unit-tests
    persona: craftsman
  - id: create-integration-tests
    persona: craftsman
  - id: validate-coverage
    persona: auditor
```

### Refactoring Assistant
```yaml
# Safe code refactoring with validation
metadata:
  name: refactoring-assistant
steps:
  - id: analyze-current-code
    persona: navigator
  - id: propose-refactoring
    persona: craftsman
  - id: validate-changes
    persona: auditor
  - id: generate-tests
    persona: craftsman
```

Wave workflows transform AI from unpredictable assistants into reliable infrastructure components. Start with simple workflows, add contracts for quality assurance, and build a library of reusable patterns your team can depend on.

## Next Steps

- [Sharing Workflows](/workflows/sharing-workflows) - Version control and team collaboration
- [Community Library](/workflows/community-library) - Discover and contribute workflows
- [Examples](/workflows/examples/) - Complete workflow specimens
- [Contracts](/concepts/contracts) - Deep dive into output validation