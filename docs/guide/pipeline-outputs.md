# Pipeline Outputs

Pipeline outputs define named aliases for pipeline results, making pipelines composable. Parent pipelines can reference child pipeline outputs without knowing internal step or artifact details.

## Defining Outputs

Add a `pipeline_outputs` block at the top level of a pipeline:

```yaml
kind: WavePipeline
metadata:
  name: impl-feature

pipeline_outputs:
  pr_url:
    step: create-pr
    artifact: result
    field: ".url"
  branch_name:
    step: create-pr
    artifact: result
    field: ".branch"
  summary:
    step: analyze
    artifact: report

steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze the feature request"
    output_artifacts:
      - name: report
        path: .agents/output/report.md
        type: markdown

  - id: implement
    persona: craftsman
    dependencies: [analyze]
    exec:
      type: prompt
      source: "Implement the feature"

  - id: create-pr
    persona: craftsman
    dependencies: [implement]
    exec:
      type: prompt
      source: "Create a pull request"
    output_artifacts:
      - name: result
        path: .agents/output/pr-result.json
        type: json
```

## Output Fields

| Field | Required | Description |
|-------|----------|-------------|
| `step` | **yes** | Source step ID that produces the artifact |
| `artifact` | **yes** | Artifact name from the source step's `output_artifacts` |
| `field` | no | JSON dot-notation path for field extraction (e.g., `.url`, `.data.name`) |

When `field` is set, Wave extracts a specific value from a JSON artifact. When omitted, the entire artifact is exposed.

## Consuming Outputs in Parent Pipelines

Parent pipelines access child outputs through artifact injection:

```yaml
steps:
  - id: run-impl
    pipeline: impl-feature
    input: "Build the login page"

  - id: notify
    persona: navigator
    dependencies: [run-impl]
    memory:
      inject_artifacts:
        - pipeline: impl-feature
          artifact: pr_url
          as: pr_link
    exec:
      type: prompt
      source: "Notify the team about the new PR"
```

The `pipeline` field in `inject_artifacts` references the child pipeline by name, and `artifact` references the named output (not the raw step artifact).

## When to Use Pipeline Outputs

- **Composability**: When a pipeline will be called as a sub-pipeline by others
- **Abstraction**: To hide internal step structure from consumers
- **Field extraction**: To expose a specific JSON field rather than the full artifact

## See Also

- [Pipeline Schema: Pipeline Outputs](/reference/pipeline-schema#pipeline-outputs) - Field reference
- [Composition](/guide/composition) - Sub-pipelines, iterate, and other composition primitives
- [Outcomes](/guide/outcomes) - Extracting deliverables for display (distinct from pipeline outputs)
