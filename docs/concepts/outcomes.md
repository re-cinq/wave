# Outcomes

Outcomes extract structured results from step artifacts into the pipeline output summary. When a step produces a JSON artifact containing a PR URL, issue link, or deployment URL, outcomes let you surface those values automatically.

```yaml
steps:
  - id: create-pr
    persona: craftsman
    exec:
      type: prompt
      source: "Create a pull request"
    output_artifacts:
      - name: result
        path: .agents/output/result.json
        type: json
    outcomes:
      - type: pr
        extract_from: .agents/output/result.json
        json_path: ".pr_url"
        label: "Pull Request"
```

After the step completes, Wave extracts the value at `.pr_url` from the JSON artifact and registers it as a PR outcome in the pipeline summary.

## Outcome Types

| Type | Description | Example Value |
|------|-------------|---------------|
| `pr` | Pull request URL | `https://github.com/org/repo/pull/42` |
| `issue` | Issue URL | `https://github.com/org/repo/issues/99` |
| `url` | Generic URL | `https://example.com/report` |
| `deployment` | Deployment URL | `https://staging.example.com` |

## Field Reference

| Field | Required | Description |
|-------|----------|-------------|
| `type` | **yes** | Outcome type: `pr`, `issue`, `url`, `deployment` |
| `extract_from` | **yes** | Artifact path relative to workspace |
| `json_path` | **yes** | Dot notation path to extract the value |
| `json_path_label` | no | Label extraction path for array items (used with `[*]` in `json_path`) |
| `label` | no | Human-readable label for the output summary |

## Array Extraction

Extract multiple values from a JSON array using `[*]` in the `json_path`:

```yaml
outcomes:
  - type: url
    extract_from: .agents/output/deploy-result.json
    json_path: ".environments[*].url"
    json_path_label: ".name"
    label: "Deployments"
```

Given this artifact:

```json
{
  "environments": [
    { "name": "staging", "url": "https://staging.example.com" },
    { "name": "production", "url": "https://prod.example.com" }
  ]
}
```

Wave extracts both URLs and uses each item's `.name` as its display label.

## Multiple Outcomes

A single step can declare multiple outcomes to extract different result types:

```yaml
outcomes:
  - type: pr
    extract_from: .agents/output/result.json
    json_path: ".pr_url"
    label: "Pull Request"
  - type: issue
    extract_from: .agents/output/result.json
    json_path: ".issue_url"
    label: "Tracking Issue"
  - type: deployment
    extract_from: .agents/output/deploy.json
    json_path: ".deploy_url"
    label: "Preview"
```

## Next Steps

- [Pipelines](/concepts/pipelines) - Pipeline concepts and dependency patterns
- [Artifacts](/concepts/artifacts) - Output files that outcomes extract from
- [Pipeline Schema Reference](/reference/pipeline-schema) - Complete field reference including outcome fields
