# Model Routing

Wave supports model selection at multiple levels, enabling cost optimization by routing cheap analysis tasks to smaller models while reserving expensive models for complex implementation work.

## Model Precedence

Model resolution follows a four-tier precedence chain. The first non-empty value wins:

| Priority | Source | Set via | Example |
|----------|--------|---------|---------|
| 1 (highest) | CLI flag | `--model` | `wave run impl-issue --model claude-haiku-4-5 -- "issue #42"` |
| 2 | Step-level | `model:` in pipeline YAML | `model: claude-haiku-4-5` |
| 3 | Persona-level | `model:` in `wave.yaml` persona | `model: claude-sonnet-4-20250514` |
| 4 (lowest) | Adapter default | Adapter binary default | Depends on adapter |

The CLI `--model` flag overrides everything, making it useful for testing or one-off cost control. Step-level pinning is the primary mechanism for per-step optimization in pipeline YAML.

## Step-Level Model Override

Pin a specific model to a step directly in the pipeline YAML:

<div v-pre>

```yaml
steps:
  - id: fetch-assess
    persona: implementer
    model: claude-haiku-4-5  # Cheap model for data extraction
    workspace:
      type: worktree
      branch: "{{ pipeline_id }}"
    exec:
      type: prompt
      source_path: .wave/prompts/implement/fetch-assess.md

  - id: implement
    persona: craftsman
    dependencies: [fetch-assess]
    # No model override -- uses persona default or adapter default
    workspace:
      type: worktree
      branch: "{{ pipeline_id }}"
    exec:
      type: prompt
      source_path: .wave/prompts/implement/implement.md
```

</div>

Steps without a `model:` field inherit from their persona or the adapter default.

## Persona-Level Model Default

Set a default model for all steps using a persona in `wave.yaml`:

```yaml
# wave.yaml
personas:
  navigator:
    adapter: claude
    system_prompt_file: .wave/personas/navigator.md
    model: claude-haiku-4-5  # Default for all navigator steps
    permissions:
      allowed_tools: ["Read", "Glob", "Grep"]

  craftsman:
    adapter: claude
    system_prompt_file: .wave/personas/craftsman.md
    # No model -- uses adapter default (typically the most capable model)
    permissions:
      allowed_tools: ["Read", "Glob", "Grep", "Write", "Edit", "Bash"]
```

This establishes a baseline: all `navigator` steps use Haiku unless overridden at the step level, while `craftsman` steps use the adapter's default model.

## Multi-Adapter Registry

Wave's adapter registry maps adapter names to runner implementations. Configure adapters in `wave.yaml`:

```yaml
# wave.yaml
adapters:
  claude:
    binary: claude
    mode: oneshot
  codex:
    binary: codex
    mode: oneshot
```

Steps can select an adapter explicitly using the `adapter:` field:

<div v-pre>

```yaml
steps:
  - id: analyze
    persona: navigator
    adapter: claude           # Explicitly select adapter
    model: claude-haiku-4-5
    exec:
      type: prompt
      source: "Analyze the codebase"
```

</div>

When no `adapter:` is set on the step, the adapter is inherited from the persona's `adapter:` field.

## Cost Optimization Patterns

### Analysis and Navigation Steps

Read-only steps that analyze code, extract data, or assess issues rarely need the most capable model:

<div v-pre>

```yaml
  - id: fetch-assess
    persona: implementer
    model: claude-haiku-4-5  # Fast, cheap -- good for structured extraction
    exec:
      type: prompt
      source_path: .wave/prompts/implement/fetch-assess.md
    handover:
      contract:
        type: json_schema
        source: .wave/output/issue-assessment.json
        schema_path: .wave/contracts/issue-assessment.schema.json
```

</div>

The contract validation catches quality issues regardless of the model used. If the cheaper model fails the contract, the retry policy handles re-execution.

### Implementation Steps

Complex implementation steps benefit from the most capable model. Leave them at the default:

```yaml
  - id: implement
    persona: craftsman
    # No model override -- default Opus for complex reasoning
    exec:
      type: prompt
      source_path: .wave/prompts/implement/implement.md
```

### PR Creation and Commenting

PR description writing and comment posting are structured tasks that work well with mid-tier models:

<div v-pre>

```yaml
  - id: create-pr
    persona: "{{ forge.type }}-commenter"
    model: claude-sonnet-4-20250514  # Good balance of quality and cost
    exec:
      type: prompt
      source_path: .wave/prompts/implement/create-pr.md
```

</div>

### LLM-as-Judge Model Selection

Contract validation with `type: llm_judge` accepts a `model` field to control which model evaluates step output:

```yaml
  - id: implement
    persona: craftsman
    handover:
      contract:
        type: llm_judge
        model: claude-haiku-4-5  # Cheap model for pass/fail evaluation
        criteria:
          - "Code compiles without errors"
          - "All acceptance criteria are addressed"
          - "No security vulnerabilities introduced"
        threshold: 1.0
```

Using a cheaper model for judge evaluation keeps costs down while still providing structured quality checks.

## Cost Optimization Summary

| Step Type | Recommended Model | Rationale |
|-----------|------------------|-----------|
| Fetch / assess / extract | `claude-haiku-4-5` | Structured data extraction, schema-validated |
| Code review / analysis | `claude-haiku-4-5` | Read-only, findings validated by contracts |
| Implementation | Default (most capable) | Complex reasoning, code generation |
| PR creation / commenting | `claude-sonnet-4-20250514` | Structured writing, moderate complexity |
| LLM judge evaluation | `claude-haiku-4-5` | Binary pass/fail against criteria |
| Relay compaction | `claude-haiku-4-5` | Summarization, not generation |

## CLI Model Override

The `--model` flag overrides all step and persona model settings for a single run. This is useful for testing pipelines with different models or controlling costs on a per-run basis:

```bash
# Run entire pipeline with Haiku (cheap testing)
wave run impl-issue --model claude-haiku-4-5 -- "re-cinq/wave 42"

# Run with default model resolution (step/persona/adapter defaults)
wave run impl-issue -- "re-cinq/wave 42"
```

The CLI override applies to every step in the pipeline. For per-step control, use the `model:` field in the pipeline YAML.

## Further Reading

- [Custom Personas](/guides/custom-personas) -- Persona configuration including model defaults
- [Pipeline Configuration](/guides/pipeline-configuration) -- Step configuration and adapters
- [Adapter Development](/guides/adapter-development) -- Building custom adapters
