# Artifacts

An artifact is a file or directory produced by a pipeline step and passed to dependent steps. Artifacts are the primary way steps communicate with each other.

```yaml
output_artifacts:
  - name: analysis
    path: .wave/output/analysis.json
    type: json
```

Use artifacts when one step needs data from a previous step - analysis results, generated code, or structured output.

## Producing Artifacts

Declare what a step produces in `output_artifacts`:

```yaml
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze the codebase"
    output_artifacts:
      - name: analysis
        path: .wave/output/analysis.json
        type: json
      - name: files
        path: .wave/output/files.md
        type: markdown
```

## Consuming Artifacts

Inject artifacts from previous steps using `inject_artifacts`:

```yaml
steps:
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
      source: "Implement based on the analysis"
```

The artifact is copied into the step's workspace at `.wave/artifacts/context`.

## Artifact Types

| Type | Description |
|------|-------------|
| `json` | Structured JSON data |
| `markdown` | Documentation or analysis |
| `file` | Any single file |
| `directory` | Folder with multiple files |

## Complete Example

A two-step pipeline with artifact passing:

```yaml
kind: WavePipeline
metadata:
  name: analyze-implement
steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze: {{ input }}"
    output_artifacts:
      - name: report
        path: .wave/output/report.json

  - id: implement
    persona: craftsman
    dependencies: [analyze]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: analyze
          artifact: report
          as: analysis
    exec:
      type: prompt
      source: "Implement based on analysis"
```

## Viewing Artifacts

List artifacts from a pipeline run:

```bash
wave artifacts run-abc123
```

**Output:**
```
STEP      ARTIFACT      TYPE    PATH
analyze   report        json    .wave/workspaces/.../.wave/output/report.json
```

## Next Steps

- [Pipelines](/concepts/pipelines) - Build multi-step workflows
- [Contracts](/concepts/contracts) - Validate artifact content
- [Pipelines](/concepts/pipelines) - How artifacts flow between steps
