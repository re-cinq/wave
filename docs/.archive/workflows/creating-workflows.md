# Creating Pipelines

This guide walks you through creating Wave pipelines from scratch. You'll learn the pipeline structure, step configuration, artifact flow, and contract validation.

## Quick Start

Create your first pipeline in `.wave/pipelines/hello.yaml`:

```yaml
kind: WavePipeline
metadata:
  name: hello
  description: "A simple hello world pipeline"

input:
  source: cli

steps:
  - id: greet
    persona: navigator
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: "Say hello to: {{ input }}"
    output_artifacts:
      - name: greeting
        path: .wave/output/greeting.txt
        type: text
```

Run it:

```bash
wave run hello "Wave Developer"
```

## Pipeline Structure

Every pipeline follows this structure:

```yaml
kind: WavePipeline
metadata:
  name: <pipeline-name>
  description: "<description>"

input:
  source: cli
  # Additional input configuration

steps:
  - id: <step-id>
    persona: <persona-name>
    # Step configuration
```

### Metadata

```yaml
metadata:
  name: code-review
  description: "Automated code review with security analysis"
```

- **name**: Used in `wave run <name>` commands (required)
- **description**: Documents the pipeline's purpose

### Input Configuration

```yaml
input:
  source: cli
  label_filter: "type=feature"
  batch_size: 10
```

- **source**: Where input comes from (`cli` for command line)
- **label_filter**: Filter inputs by labels (optional)
- **batch_size**: Process inputs in batches (optional)

## Step Configuration

Steps are the building blocks of pipelines. Each step runs a persona to accomplish a specific task.

### Basic Step

```yaml
steps:
  - id: analyze
    persona: navigator
    memory:
      strategy: fresh
    exec:
      type: prompt
      source: "Analyze the codebase: {{ input }}"
    output_artifacts:
      - name: analysis
        path: .wave/output/analysis.json
        type: json
```

### Step Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique identifier for the step |
| `persona` | Yes | Which persona executes this step |
| `dependencies` | No | Steps that must complete first |
| `memory` | No | Memory and artifact injection settings |
| `workspace` | No | File system mount configuration |
| `exec` | Yes | What the step executes |
| `output_artifacts` | No | Files produced by the step |
| `handover` | No | Contract validation configuration |

### Dependencies

Steps wait for dependencies before executing:

```yaml
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze: {{ input }}"
    output_artifacts:
      - name: analysis
        path: .wave/output/analysis.json
        type: json

  - id: implement
    persona: craftsman
    dependencies: [analyze]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze
          artifact: analysis
          as: context
    exec:
      type: prompt
      source: |
        Based on the analysis:
        {{ artifacts.context }}

        Implement the changes.
```

Parallel execution happens automatically when steps share a dependency:

```yaml
steps:
  - id: analyze

  - id: security-review
    dependencies: [analyze]

  - id: quality-review
    dependencies: [analyze]

  - id: summary
    dependencies: [security-review, quality-review]
```

`security-review` and `quality-review` run in parallel since they only depend on `analyze`.

### Memory Configuration

The `memory` section controls what context a step receives:

```yaml
memory:
  strategy: fresh
  inject_artifacts:
    - step: previous-step
      artifact: artifact-name
      as: local-name
```

- **strategy**: Always `fresh` - steps start with clean context
- **inject_artifacts**: Explicitly include outputs from previous steps

Access injected artifacts in prompts:

```yaml
exec:
  type: prompt
  source: |
    Previous analysis: {{ artifacts.local-name }}

    Continue with...
```

### Workspace Configuration

Mount source files into the step's workspace:

```yaml
workspace:
  mount:
    - source: ./src
      target: /code
      mode: readonly
    - source: ./tests
      target: /tests
      mode: readonly
```

- **source**: Path relative to project root
- **target**: Path inside the workspace
- **mode**: `readonly` or `readwrite`

### Execution Configuration

```yaml
exec:
  type: prompt
  source: |
    Your task instructions here.

    Input: {{ input }}
    Context: {{ artifacts.analysis }}
```

- **type**: `prompt` for AI execution
- **source**: The prompt template (supports variable substitution)

### Output Artifacts

Declare files the step produces:

```yaml
output_artifacts:
  - name: analysis
    path: .wave/output/analysis.json
    type: json
    required: true
  - name: report
    path: .wave/output/report.md
    type: markdown
```

- **name**: Identifier for referencing in `inject_artifacts`
- **path**: Location in the workspace
- **type**: Content type (`json`, `markdown`, `text`, etc.)
- **required**: Whether the artifact must exist (default: false)

## Contract Validation

Contracts ensure outputs meet requirements before continuing:

```yaml
handover:
  contract:
    type: jsonschema
    schema_path: .wave/contracts/analysis.schema.json
    on_failure: retry
    max_retries: 2
```

### Contract Types

**JSON Schema**
```yaml
handover:
  contract:
    type: jsonschema
    schema_path: .wave/contracts/output.schema.json
```

**TypeScript**
```yaml
handover:
  contract:
    type: typescript
    source: .wave/output/types.ts
    validate: true
```

**Test Suite**
```yaml
handover:
  contract:
    type: testsuite
    command: "npm test"
    must_pass: true
```

**Markdown Spec**
```yaml
handover:
  contract:
    type: markdownspec
    source: .wave/output/docs.md
```

### Failure Handling

```yaml
handover:
  contract:
    type: jsonschema
    schema_path: .wave/contracts/output.schema.json
    on_failure: retry
    max_retries: 3
  on_review_fail: retry
  target_step: analyze  # Jump back to specific step
```

## Complete Example: Code Review Pipeline

```yaml
kind: WavePipeline
metadata:
  name: code-review
  description: "Security-focused automated code review"

input:
  source: cli

steps:
  - id: diff-analysis
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

        1. Identify all modified files
        2. Map change scope (modules affected)
        3. Find related tests
        4. Check for breaking changes

        Output as JSON:
        {
          "files_changed": [...],
          "modules_affected": [...],
          "related_tests": [...],
          "breaking_changes": [...]
        }
    output_artifacts:
      - name: diff
        path: .wave/output/diff-analysis.json
        type: json
    handover:
      contract:
        type: jsonschema
        schema_path: .wave/contracts/diff-analysis.schema.json
        on_failure: retry
        max_retries: 2

  - id: security-review
    persona: auditor
    dependencies: [diff-analysis]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: diff-analysis
          artifact: diff
          as: changes
    exec:
      type: prompt
      source: |
        Security review of changes:
        {{ artifacts.changes }}

        Check for:
        1. SQL injection, XSS, CSRF
        2. Hardcoded secrets
        3. Insecure deserialization
        4. Missing input validation
        5. Auth/authz gaps

        Output findings with severity.
    output_artifacts:
      - name: security
        path: .wave/output/security-review.md
        type: markdown

  - id: quality-review
    persona: auditor
    dependencies: [diff-analysis]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: diff-analysis
          artifact: diff
          as: changes
    exec:
      type: prompt
      source: |
        Quality review of changes:
        {{ artifacts.changes }}

        Check for:
        1. Error handling
        2. Edge cases
        3. Code duplication
        4. Missing tests
        5. Performance issues

        Output findings with suggestions.
    output_artifacts:
      - name: quality
        path: .wave/output/quality-review.md
        type: markdown

  - id: summary
    persona: summarizer
    dependencies: [security-review, quality-review]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: security-review
          artifact: security
          as: security_findings
        - step: quality-review
          artifact: quality
          as: quality_findings
    exec:
      type: prompt
      source: |
        Synthesize review findings:

        Security: {{ artifacts.security_findings }}
        Quality: {{ artifacts.quality_findings }}

        Provide:
        1. Overall verdict (APPROVE / REQUEST_CHANGES)
        2. Critical issues
        3. Suggested improvements
        4. Positive observations

        Format as PR review comment.
    output_artifacts:
      - name: verdict
        path: .wave/output/review-summary.md
        type: markdown
```

## Directory Structure

Organize pipelines and related files:

```
.wave/
├── pipelines/
│   ├── code-review.yaml
│   ├── documentation.yaml
│   └── testing.yaml
├── personas/
│   ├── navigator.md
│   ├── auditor.md
│   └── craftsman.md
├── contracts/
│   ├── diff-analysis.schema.json
│   └── review.schema.json
└── traces/           # Audit logs (auto-generated)
```

## Testing Pipelines

### Validate Configuration

```bash
wave validate
```

Checks syntax, references, and schema compliance.

### Dry Run

Test with actual input:

```bash
wave run code-review "Review changes to auth module"
```

### Debug Mode

Enable verbose output:

```bash
wave run code-review "test input" --debug
```

### Monitor Execution

```bash
# List runs
wave list

# Check status
wave status <run-id>

# View logs
wave logs <run-id>
```

## Best Practices

### 1. Start Simple

Begin with a single-step pipeline, add complexity incrementally:

```yaml
# Start here
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze {{ input }}"
```

### 2. Clear Step Boundaries

Each step should have a single, clear purpose:

- **Good**: "Analyze code changes" → "Review security" → "Generate summary"
- **Avoid**: "Analyze, review, and summarize everything"

### 3. Explicit Artifact Flow

Always explicitly inject artifacts:

```yaml
memory:
  inject_artifacts:
    - step: previous
      artifact: output
      as: context
```

Don't rely on implicit data passing.

### 4. Contract Everything

Add contracts for any structured output:

```yaml
handover:
  contract:
    type: jsonschema
    schema_path: .wave/contracts/output.schema.json
    on_failure: retry
    max_retries: 2
```

### 5. Use Read-Only Mounts

Default to read-only access:

```yaml
workspace:
  mount:
    - source: ./src
      target: /code
      mode: readonly
```

Only use `readwrite` when the step must modify files.

## Next Steps

- [Pipeline Execution](/concepts/pipeline-execution) - How pipelines run
- [Contracts](/paradigm/deliverables-contracts) - Output validation
- [Personas](/concepts/personas) - Configuring AI agents
- [Sharing Pipelines](/workflows/sharing-workflows) - Team collaboration
