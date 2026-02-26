# Community Patterns

This page documents common pipeline patterns from the Wave community. These patterns are not part of a centralized registry - they're examples you can copy, adapt, and share through git.

## Code Review Patterns

### Basic Code Review

Analyzes changes and provides feedback:

```yaml
kind: WavePipeline
metadata:
  name: gh-pr-review
  description: "Basic code review"

input:
  source: cli

steps:
  - id: analyze
    persona: navigator
    memory:
      strategy: fresh
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Analyze the code changes: {{ input }}

        Provide:
        1. Summary of changes
        2. Potential issues
        3. Suggestions for improvement
    output_artifacts:
      - name: review
        path: .wave/output/review.md
        type: markdown
```

### Security-Focused Review

Adds dedicated security analysis:

```yaml
kind: WavePipeline
metadata:
  name: security-review
  description: "Security-focused code review"

input:
  source: cli

steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze changes: {{ input }}"
    output_artifacts:
      - name: analysis
        path: .wave/output/analysis.json
        type: json

  - id: security
    persona: auditor
    dependencies: [analyze]
    memory:
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: |
        Security review based on:
        {{ artifacts.context }}

        Check for OWASP Top 10 vulnerabilities.
    output_artifacts:
      - name: security
        path: .wave/output/security.md
        type: markdown
```

## Documentation Patterns

### API Documentation

Generates documentation from code:

```yaml
kind: WavePipeline
metadata:
  name: api-docs
  description: "Generate API documentation"

input:
  source: cli

steps:
  - id: extract
    persona: navigator
    workspace:
      mount:
        - source: ./src
          target: /code
          mode: readonly
    exec:
      type: prompt
      source: |
        Extract API endpoints from {{ input }}

        Output as JSON with endpoint details.
    output_artifacts:
      - name: endpoints
        path: .wave/output/endpoints.json
        type: json
    handover:
      contract:
        type: jsonschema
        schema_path: .wave/contracts/endpoints.schema.json

  - id: document
    persona: documenter
    dependencies: [extract]
    memory:
      inject_artifacts:
        - step: extract
          artifact: endpoints
          as: api
    exec:
      type: prompt
      source: |
        Generate markdown documentation for:
        {{ artifacts.api }}

        Include examples and error responses.
    output_artifacts:
      - name: docs
        path: .wave/output/api-docs.md
        type: markdown
```

### README Generation

Creates or updates README files:

```yaml
kind: WavePipeline
metadata:
  name: readme-update
  description: "Generate or update README"

input:
  source: cli

steps:
  - id: analyze-project
    persona: navigator
    workspace:
      mount:
        - source: ./
          target: /project
          mode: readonly
    exec:
      type: prompt
      source: |
        Analyze the project structure and purpose.
        Focus on: {{ input }}
    output_artifacts:
      - name: structure
        path: .wave/output/structure.json
        type: json

  - id: generate-readme
    persona: documenter
    dependencies: [analyze-project]
    memory:
      inject_artifacts:
        - step: analyze-project
          artifact: structure
          as: project
    exec:
      type: prompt
      source: |
        Generate a README.md based on:
        {{ artifacts.project }}

        Include: installation, usage, examples.
    output_artifacts:
      - name: readme
        path: .wave/output/README.md
        type: markdown
```

## Testing Patterns

### Test Generation

Generates tests for existing code:

```yaml
kind: WavePipeline
metadata:
  name: generate-tests
  description: "Generate unit tests"

input:
  source: cli

steps:
  - id: analyze
    persona: navigator
    workspace:
      mount:
        - source: ./src
          target: /src
          mode: readonly
    exec:
      type: prompt
      source: |
        Analyze {{ input }} for testable functions.
        Identify edge cases and test scenarios.
    output_artifacts:
      - name: analysis
        path: .wave/output/test-analysis.json
        type: json

  - id: generate
    persona: craftsman
    dependencies: [analyze]
    memory:
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: scenarios
    exec:
      type: prompt
      source: |
        Generate unit tests for:
        {{ artifacts.scenarios }}

        Use the project's existing test framework.
    output_artifacts:
      - name: tests
        path: .wave/output/tests.generated.ts
        type: typescript
    handover:
      contract:
        type: typescript
        source: .wave/output/tests.generated.ts
        validate: true
```

## Debugging Patterns

### Root Cause Analysis

Investigates issues systematically:

```yaml
kind: WavePipeline
metadata:
  name: debug
  description: "Debug issue analysis"

input:
  source: cli

steps:
  - id: gather-context
    persona: debugger
    workspace:
      mount:
        - source: ./
          target: /project
          mode: readonly
    exec:
      type: prompt
      source: |
        Investigate: {{ input }}

        Gather relevant:
        - Code paths
        - Log patterns
        - Recent changes
    output_artifacts:
      - name: context
        path: .wave/output/debug-context.json
        type: json

  - id: analyze
    persona: debugger
    dependencies: [gather-context]
    memory:
      inject_artifacts:
        - step: gather-context
          artifact: context
          as: info
    exec:
      type: prompt
      source: |
        Based on: {{ artifacts.info }}

        Provide:
        1. Most likely root cause
        2. Steps to verify
        3. Suggested fix
    output_artifacts:
      - name: analysis
        path: .wave/output/root-cause.md
        type: markdown
```

## Contributing Patterns

### Share Your Patterns

Found a useful pattern? Share it:

1. Document in your repository's README
2. Write a blog post or gist
3. Contribute to Wave discussions
4. Create a public repository with your pipelines

### Pattern Guidelines

Good patterns are:

- **Focused**: Solve one problem well
- **Documented**: Clear description and usage
- **Tested**: Include example inputs and expected outputs
- **Portable**: Minimal project-specific dependencies

### Example Repository Structure

```
my-wave-patterns/
├── README.md
├── gh-pr-review/
│   ├── basic.yaml
│   ├── security.yaml
│   └── README.md
├── documentation/
│   ├── api-docs.yaml
│   └── README.md
├── contracts/
│   ├── review.schema.json
│   └── endpoints.schema.json
└── personas/
    ├── navigator.md
    └── auditor.md
```

## Next Steps

- [Creating Pipelines](/workflows/creating-workflows) - Build your own patterns
- [Sharing Pipelines](/workflows/sharing-workflows) - Share with your team
- [Contracts](/paradigm/deliverables-contracts) - Ensure pattern reliability
