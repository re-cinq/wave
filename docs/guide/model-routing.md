# Multi-Adapter Model Routing

Route different LLM models to different pipeline steps for cost optimization and provider resilience.

## Per-Step Model Assignment

```yaml
steps:
  - id: analyze
    persona: navigator
    model: claude-haiku          # cheap model for analysis

  - id: implement
    persona: craftsman
    # no model override — uses adapter default (Sonnet)

  - id: review
    persona: reviewer
    model: claude-haiku          # cheap model for review
```

## Cost Optimization Strategy

| Step Type | Recommended Model | Savings |
|-----------|------------------|---------|
| Navigation/exploration | claude-haiku | ~80% |
| Analysis/scanning | claude-haiku | ~70% |
| Synthesis/summary | claude-haiku | ~70% |
| Code generation | *(default — Sonnet)* | baseline |
| Complex reasoning | *(default — Sonnet)* | baseline |

Use unversioned names (`claude-haiku`, not `claude-haiku-4-5`) — the adapter resolves to the latest version.

## Adapter Registry

Wave supports multiple adapter backends:

```yaml
adapters:
  claude:
    binary: claude
    mode: headless
  codex:
    binary: codex
    mode: headless
  gemini:
    binary: gemini
    mode: headless
```

Override per step:

```yaml
- id: implement
  adapter: codex              # use OpenAI Codex for this step
  model: gpt-4o
```

## Fallback Chains

When a provider fails, try alternatives:

```yaml
runtime:
  fallbacks:
    anthropic: [openai, gemini]
    openai: [anthropic]
```

Fallback only triggers on transient failures (rate limits, timeouts). Permanent failures (auth errors, missing binary) do not fallback.

## Persona-Level Defaults

Personas can declare a preferred model, overridable at step level:

```yaml
# .wave/personas/analyst.yaml
name: analyst
model: claude-haiku
adapter: claude
```
