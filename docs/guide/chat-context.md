# Chat Context

Chat context configures what information to inject into post-pipeline interactive chat sessions. When a pipeline completes, Wave can start a chat session pre-loaded with pipeline results so users can explore outcomes conversationally.

## Basic Configuration

```yaml
kind: WavePipeline
metadata:
  name: audit-security

chat_context:
  artifact_summaries:
    - findings
    - recommendations
  suggested_questions:
    - "What are the critical vulnerabilities?"
    - "Which files are most affected?"
    - "What should we fix first?"
  focus_areas:
    - security
    - authentication
    - data-validation

steps:
  - id: scan
    persona: auditor
    exec:
      type: prompt
      source: "Scan for security vulnerabilities"
    output_artifacts:
      - name: findings
        path: .wave/output/findings.json
        type: json
      - name: recommendations
        path: .wave/output/recommendations.md
        type: markdown
```

## Fields

| Field | Default | Description |
|-------|---------|-------------|
| `artifact_summaries` | `[]` | Artifact names to summarize and inject into the chat context |
| `suggested_questions` | `[]` | Opening questions displayed to the user when the chat session starts |
| `focus_areas` | `[]` | Topic areas to highlight, helping the chat session stay relevant |
| `max_context_tokens` | `8000` | Token budget for injected context |

## Artifact Summaries

List the artifact names (from any step's `output_artifacts`) to include in the chat context. Wave summarizes these artifacts and injects them as background context for the chat session.

```yaml
chat_context:
  artifact_summaries:
    - analysis        # from step: analyze
    - test-results    # from step: test
    - implementation  # from step: implement
```

Only reference artifacts that provide useful background. Large artifacts are truncated to fit within `max_context_tokens`.

## Suggested Questions

Provide starting questions relevant to the pipeline's output. These appear as clickable suggestions when the chat session opens.

```yaml
chat_context:
  suggested_questions:
    - "Summarize the key findings"
    - "What patterns emerged from the analysis?"
    - "What are the recommended next steps?"
```

## Focus Areas

Focus areas guide the chat session toward relevant topics, reducing off-topic responses.

```yaml
chat_context:
  focus_areas:
    - performance
    - api-design
    - error-handling
```

## Token Budget

Control how much context is injected. Larger budgets provide more detail but consume more of the model's context window.

```yaml
chat_context:
  max_context_tokens: 16000
```

Default is 8000 tokens. Set higher for complex pipelines with many artifacts, lower for simple pipelines where you want the chat session to be more responsive.

## When to Use Chat Context

- **Exploratory analysis**: After audit or research pipelines, let users dig into findings
- **Implementation review**: After implementation pipelines, chat about the changes made
- **Decision support**: After planning pipelines, discuss recommendations interactively

## See Also

- [Pipeline Schema: Chat Context](/reference/pipeline-schema#chat-context) - Field reference
- [Outcomes](/guide/outcomes) - Structured deliverable extraction (complementary feature)
