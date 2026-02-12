---
title: Documentation Generation
description: Generate API docs, README files, and usage guides automatically from your code
---

# Documentation Generation

<div class="use-case-meta">
  <span class="complexity-badge beginner">Beginner</span>
  <span class="category-badge">Documentation</span>
</div>

Generate and update documentation from your code. Wave's docs pipeline analyzes your codebase, identifies public APIs, and produces structured documentation with examples.

## Prerequisites

- Wave installed and initialized (`wave init`)
- Codebase with exported functions, types, or APIs to document
- Basic understanding of YAML configuration

## Quick Start

```bash
wave run docs "generate API documentation"
```

Expected output:

```
[10:00:01] started   discover   (navigator)              Starting step
[10:00:28] completed discover   (navigator)   27s   2.8k Discovery complete
[10:00:29] started   generate   (philosopher)            Starting step
[10:01:15] completed generate   (philosopher)  46s   5.2k Generation complete
[10:01:16] started   review     (auditor)                Starting step
[10:01:35] completed review     (auditor)     19s   1.5k Review complete

Pipeline docs completed in 94s
Artifacts: output/generated-docs.md
```

## Complete Pipeline

This is the full `docs` pipeline from `.wave/pipelines/docs.yaml`:

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: docs
  description: "Generate or update documentation"

input:
  source: cli

steps:
  - id: discover
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
        Analyze the codebase for documentation needs: {{ input }}

        1. Find public APIs, exported functions, types
        2. Identify existing documentation (README, docs/, comments)
        3. Map package structure and dependencies
        4. Find usage examples in tests

        Output as JSON:
        {
          "public_apis": [{"package": "", "name": "", "type": "func|type|const", "documented": true|false}],
          "existing_docs": [],
          "package_structure": {},
          "examples_in_tests": []
        }
    output_artifacts:
      - name: discovery
        path: output/discovery.json
        type: json

  - id: generate
    persona: philosopher
    dependencies: [discover]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: discover
          artifact: discovery
          as: codebase
    workspace:
      mount:
        - source: ./
          target: /src
          mode: readwrite
    exec:
      type: prompt
      source: |
        Generate documentation for: {{ input }}

        Include:
        1. Package overview with purpose and usage
        2. API reference for public functions/types
        3. Code examples (extract from tests where possible)
        4. Configuration options
        5. Error handling and edge cases

        Write clear, concise documentation. Use code blocks for examples.
    output_artifacts:
      - name: docs
        path: output/generated-docs.md
        type: markdown

  - id: review
    persona: auditor
    dependencies: [generate]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: generate
          artifact: docs
          as: documentation
    exec:
      type: prompt
      source: |
        Review the generated documentation:

        1. Accuracy - does it match the actual code?
        2. Completeness - are all public APIs documented?
        3. Clarity - is it understandable for new users?
        4. Examples - do they work and demonstrate usage?
        5. Formatting - consistent style, proper markdown?

        Output: list of issues or "APPROVED"
    output_artifacts:
      - name: review
        path: output/doc-review.md
        type: markdown
```

</div>

## Expected Outputs

The pipeline produces three artifacts:

| Artifact | Path | Description |
|----------|------|-------------|
| `discovery` | `output/discovery.json` | JSON inventory of APIs and existing docs |
| `docs` | `output/generated-docs.md` | Generated documentation |
| `review` | `output/doc-review.md` | Review feedback and approval status |

### Example Output

The pipeline produces `output/generated-docs.md`:

```markdown
# Pipeline Package

The `pipeline` package executes multi-step AI workflows with dependency
resolution, artifact passing, and contract validation.

## Installation

` ` `go
import "github.com/recinq/wave/internal/pipeline"
` ` `

## Quick Start

` ` `go
executor := pipeline.NewExecutor(config)
result, err := executor.Run(ctx, pipelineDef, input)
if err != nil {
    log.Fatalf("pipeline failed: %v", err)
}
fmt.Printf("Completed %d steps\n", len(result.Steps))
` ` `

## API Reference

### Executor

` ` `go
type Executor struct {
    // contains filtered or unexported fields
}
` ` `

`Executor` runs pipeline workflows. Create one with `NewExecutor`.

#### func NewExecutor

` ` `go
func NewExecutor(config ExecutorConfig) *Executor
` ` `

NewExecutor creates a pipeline executor with the given configuration.

#### func (*Executor) Run

` ` `go
func (e *Executor) Run(ctx context.Context, pipeline Pipeline, input string) (*Result, error)
` ` `

Run executes the pipeline and returns the result. Steps are executed
in dependency order with parallel execution where possible.

### Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| MaxConcurrency | int | 4 | Maximum parallel steps |
| Timeout | time.Duration | 30m | Pipeline timeout |
| WorkspaceRoot | string | ".wave/workspaces" | Workspace directory |

## Error Handling

The executor returns structured errors:

` ` `go
result, err := executor.Run(ctx, pipeline, input)
if err != nil {
    var stepErr *pipeline.StepError
    if errors.As(err, &stepErr) {
        fmt.Printf("Step %s failed: %v\n", stepErr.StepID, stepErr.Cause)
    }
}
` ` `
```

## Customization

### Generate README

```bash
wave run docs "generate README.md for the project"
```

### Document specific package

```bash
wave run docs "document the internal/contract package"
```

### Generate changelog

```bash
wave run docs "generate changelog from git history since v1.0.0"
```

### Add to existing docs

Modify the generate step to update rather than replace:

<div v-pre>

```yaml
- id: generate
  exec:
    source: |
      Update existing documentation at docs/api.md with new APIs.
      Preserve existing content. Add new sections for undocumented APIs.
```

</div>

## Example: API Reference Pipeline

For comprehensive API documentation, create a specialized pipeline:

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: api-docs
  description: "Generate comprehensive API reference"

steps:
  - id: scan
    persona: navigator
    exec:
      source: |
        Scan for all exported symbols: {{ input }}
        Include: functions, types, constants, variables
        Output structured inventory with signatures and comments.
    output_artifacts:
      - name: api-inventory
        path: output/api-inventory.json
        type: json

  - id: document
    persona: philosopher
    dependencies: [scan]
    memory:
      inject_artifacts:
        - step: scan
          artifact: api-inventory
          as: apis
    exec:
      source: |
        Generate API reference documentation.
        Include: description, parameters, return values, examples, errors.
        Use godoc style formatting.
    output_artifacts:
      - name: api-reference
        path: output/api-reference.md
        type: markdown
```

</div>

## Related Use Cases

- [Code Review](/use-cases/code-review) - Review documentation changes in PRs
- [Test Generation](/use-cases/test-generation) - Generate tests from documented behavior
- [API Design](./api-design) - Design APIs with documentation-first approach

## Next Steps

- [Concepts: Artifacts](/concepts/artifacts) - Understand how docs are passed between steps
- [Concepts: Personas](/concepts/personas) - Learn about the philosopher persona

<style>
.use-case-meta {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
}
.complexity-badge {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 12px;
  text-transform: uppercase;
}
.complexity-badge.beginner {
  background: #dcfce7;
  color: #166534;
}
.complexity-badge.intermediate {
  background: #fef3c7;
  color: #92400e;
}
.complexity-badge.advanced {
  background: #fee2e2;
  color: #991b1b;
}
.category-badge {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 500;
  border-radius: 12px;
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
}
</style>
