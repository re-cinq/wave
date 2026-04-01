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

Wave implements a multi-adapter registry that routes pipeline steps to different LLM backends. The `AdapterRegistry` resolves the active adapter per step using a precedence chain.

### Adapter Resolution (strongest to weakest)

| Priority | Source | Scope | Example |
|----------|--------|-------|---------|
| 1 | CLI `--adapter` flag | Entire run | `wave run impl-issue --adapter opencode` |
| 2 | Step-level `adapter:` | Single step | `adapter: codex` in pipeline YAML |
| 3 | Persona-level `adapter:` | Steps using that persona | `adapter: claude` in persona definition |
| 4 | Adapter default | Steps with no override | Falls back to first configured adapter |

```yaml
adapters:
  claude:
    binary: claude
    mode: headless
  opencode:
    binary: opencode
    mode: headless
  codex:
    binary: codex
    mode: headless
  gemini:
    binary: gemini
    mode: headless
```

### CLI Adapter Override

Use `--adapter <name>` to override the adapter for all steps in a run. Takes precedence over step-level and persona-level settings:

```bash
wave run ops-hello-world --adapter opencode --model "zai-coding-plan/glm-5-turbo"
```

Override per step in pipeline YAML:

```yaml
- id: implement
  adapter: codex              # use OpenAI Codex for this step
  model: gpt-4o
```

### Model Format Differences

Each adapter uses a different model identifier format:

| Adapter | Format | Examples |
|---------|--------|---------|
| claude | Short names | `sonnet`, `haiku`, `opus` |
| opencode | `provider/model` | `zai-coding-plan/glm-5-turbo`, `anthropic/claude-sonnet-4-20250514` |
| gemini | Plain names | `gemini-2.0-pro`, `gemini-2.5-flash` |
| codex | OpenAI identifiers | `gpt-4o`, `o3` |

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
