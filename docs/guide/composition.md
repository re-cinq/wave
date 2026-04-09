# Composition Primitives

Wave provides five composition primitives for building complex workflows from simpler parts: sub-pipelines, iterate, branch, loop, and aggregate.

## Sub-Pipelines

Execute a child pipeline as a step. Use sub-pipelines to encapsulate reusable workflow components.

```yaml
steps:
  - id: implement
    persona: craftsman
    exec:
      type: prompt
      source: "Implement the feature"

  - id: test-suite
    pipeline: test-validate
    input: "Validate the implementation"
    config:
      inject: [implementation]
      extract: [test-results]
      timeout: "3600s"
    dependencies: [implement]
```

The `config` block controls artifact flow between parent and child:
- **inject**: Parent artifacts passed into the child pipeline
- **extract**: Child artifacts pulled back to the parent
- **timeout**: Hard limit on child execution time
- **max_cycles**: Iteration cap for child loop steps
- **stop_condition**: Template expression for early termination

## Iterate

Fan out over a collection of items, executing a child pipeline for each item.

### Sequential Iteration

```yaml
steps:
  - id: plan
    persona: navigator
    exec:
      type: prompt
      source: "List files to review"
    output_artifacts:
      - name: file-list
        path: .wave/output/files.json
        type: json

  - id: review-each
    iterate:
      over: "{{ steps.plan.artifacts.file-list }}"
      mode: sequential
    pipeline: review-single-file
    input: "{{ item }}"
    dependencies: [plan]
```

### Parallel Iteration

```yaml
steps:
  - id: process-all
    iterate:
      over: "{{ steps.discover.artifacts.items }}"
      mode: parallel
      max_concurrent: 4
    pipeline: process-item
    input: "{{ item }}"
    config:
      extract: [result]
    dependencies: [discover]
```

### Iterate Fields

| Field | Required | Description |
|-------|----------|-------------|
| `over` | **yes** | Template expression resolving to a JSON array |
| `mode` | **yes** | `sequential` or `parallel` |
| `max_concurrent` | no | Max parallel workers (parallel mode only) |

## Branch

Route execution to different pipelines based on a runtime value.

```yaml
steps:
  - id: classify
    persona: navigator
    exec:
      type: prompt
      source: "Classify as: bug, feature, or docs"
    output_artifacts:
      - name: classification
        path: .wave/output/classification.json
        type: json

  - id: route
    branch:
      on: "{{ steps.classify.artifacts.classification.type }}"
      cases:
        bug: impl-bugfix
        feature: impl-feature
        docs: doc-update
        _default: skip
    dependencies: [classify]
```

The `on` expression is evaluated at runtime. The matching case value determines which pipeline to execute. Use `skip` as the pipeline name for a no-op case.

## Loop

Execute sub-steps repeatedly until a condition is met or the iteration limit is reached.

```yaml
steps:
  - id: initial
    persona: craftsman
    exec:
      type: prompt
      source: "Write the first draft"

  - id: refine
    loop:
      max_iterations: 5
      until: "{{ steps.validate.outcome == 'success' }}"
      steps:
        - id: improve
          persona: craftsman
          exec:
            type: prompt
            source: "Improve based on feedback"
        - id: validate
          type: command
          script: "go test ./..."
          dependencies: [improve]
    dependencies: [initial]
```

### Loop Fields

| Field | Required | Description |
|-------|----------|-------------|
| `max_iterations` | **yes** | Hard limit on iterations |
| `until` | no | Template condition for early exit |
| `steps` | no | Sub-steps executed per iteration |

Loops also work with graph-mode edges for more flexible control flow. See [Graph Loops](/guide/graph-loops).

## Aggregate

Collect and merge outputs from fan-out steps (iterate or matrix).

```yaml
steps:
  - id: review-all
    iterate:
      over: "{{ steps.plan.artifacts.items }}"
      mode: parallel
      max_concurrent: 4
    pipeline: review-item
    input: "{{ item }}"
    config:
      extract: [finding]
    dependencies: [plan]

  - id: collect
    aggregate:
      from: "{{ steps.review-all.results }}"
      into: .wave/output/all-findings.json
      strategy: merge_arrays
    dependencies: [review-all]
```

### Aggregation Strategies

| Strategy | Description |
|----------|-------------|
| `merge_arrays` | Merge JSON arrays from all items into one array |
| `concat` | Concatenate text outputs |
| `reduce` | Custom reduction logic |

### Key Extraction

When sub-pipelines produce JSON objects that wrap an array (e.g., `{"findings": [...], "summary": "..."}`), use the `key` field to extract and merge only the array values:

```yaml
  - id: collect
    aggregate:
      from: "{{ steps.run-audits.results }}"
      into: .wave/output/merged-findings.json
      strategy: merge_arrays
      key: findings
    dependencies: [run-audits]
```

This extracts the `findings` array from each object and merges them into a single flat array. Without `key`, `merge_arrays` expects each element to already be an array.

## Combining Primitives

Composition primitives can be combined in a single pipeline:

```yaml
steps:
  - id: discover
    persona: navigator
    exec:
      type: prompt
      source: "Discover items to process"

  - id: classify
    iterate:
      over: "{{ steps.discover.artifacts.items }}"
      mode: parallel
      max_concurrent: 3
    pipeline: classify-item
    dependencies: [discover]

  - id: route-bugs
    branch:
      on: "{{ steps.classify.artifacts.summary.has_bugs }}"
      cases:
        "true": impl-bugfix-batch
        "false": skip
    dependencies: [classify]

  - id: collect
    aggregate:
      from: "{{ steps.classify.results }}"
      into: .wave/output/report.json
      strategy: merge_arrays
    dependencies: [classify]
```

## See Also

- [Pipeline Schema: Composition](/reference/pipeline-schema#iterate) - Field reference
- [Graph Loops](/guide/graph-loops) - Edge-based loops and conditional routing
- [Pipeline Outputs](/guide/pipeline-outputs) - Exposing results for parent pipelines
