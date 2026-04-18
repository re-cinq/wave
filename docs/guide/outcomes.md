# Outcomes

Outcomes extract structured deliverables from step artifacts and register them with the pipeline output summary. Use outcomes to surface PR URLs, issue links, deployment URLs, and other key results in the Wave UI.

## Basic Outcome

```yaml
steps:
  - id: create-pr
    persona: craftsman
    exec:
      type: prompt
      source: "Create a pull request"
    output_artifacts:
      - name: result
        path: .agents/output/pr-result.json
        type: json
    outcomes:
      - type: pr
        extract_from: output/pr-result.json
        json_path: ".url"
        label: "Pull Request"
```

When the step completes, Wave extracts the value at `.url` from the JSON artifact and registers it as a PR deliverable in the run summary.

## Outcome Types

| Type | Description | Requires `json_path` |
|------|-------------|---------------------|
| `pr` | Pull request URL | Yes |
| `issue` | Issue URL | Yes |
| `url` | Generic URL | Yes |
| `deployment` | Deployment URL | Yes |
| `file` | File deliverable | No |
| `artifact` | Artifact deliverable | No |

### URL-Based Outcomes

URL-based types (`pr`, `issue`, `url`, `deployment`) require `json_path` to extract a URL string from a JSON artifact:

```yaml
outcomes:
  - type: pr
    extract_from: output/result.json
    json_path: ".pr_url"
    label: "Feature PR"
  - type: issue
    extract_from: output/result.json
    json_path: ".issue_url"
    label: "Tracking Issue"
  - type: deployment
    extract_from: output/deploy.json
    json_path: ".url"
    label: "Staging"
```

### File and Artifact Outcomes

`file` and `artifact` types use `extract_from` directly as the deliverable path without JSON extraction:

```yaml
outcomes:
  - type: file
    extract_from: output/report.md
    label: "Analysis Report"
  - type: artifact
    extract_from: output/coverage.html
    label: "Coverage Report"
```

## Array Extraction

Extract multiple values from a JSON array using `[*]` in the json_path:

```yaml
outcomes:
  - type: url
    extract_from: output/deploy-result.json
    json_path: ".environments[*].url"
    json_path_label: ".environments[*].name"
    label: "Deployments"
```

This creates one deliverable per array element. The `json_path_label` field extracts a label for each item.

## Multiple Outcomes per Step

A step can declare multiple outcomes:

```yaml
steps:
  - id: publish
    persona: craftsman
    exec:
      type: prompt
      source: "Create PR and deployment"
    output_artifacts:
      - name: result
        path: .agents/output/publish.json
        type: json
    outcomes:
      - type: pr
        extract_from: output/publish.json
        json_path: ".pr_url"
        label: "Pull Request"
      - type: deployment
        extract_from: output/publish.json
        json_path: ".deploy_url"
        label: "Preview"
      - type: file
        extract_from: output/changelog.md
        label: "Changelog"
```

## Outcomes vs Pipeline Outputs

These serve different purposes:

| Feature | Outcomes | Pipeline Outputs |
|---------|----------|-----------------|
| Purpose | Surface results in UI | Compose pipelines |
| Scope | Human-facing display | Machine-facing composition |
| Location | Step-level | Pipeline-level |
| Consumer | Wave UI, CLI output | Parent pipelines |

A pipeline can use both: outcomes for display, pipeline_outputs for composition.

## See Also

- [Pipeline Schema: Outcomes](/reference/pipeline-schema#outcomes) - Field reference
- [Outcomes Concept](/concepts/outcomes) - Conceptual overview
- [Pipeline Outputs](/guide/pipeline-outputs) - Named output aliases for composition
