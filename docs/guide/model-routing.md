# Multi-Adapter Model Routing

Route different LLM models to different pipeline steps for cost optimization and provider resilience.

## Model Tier System

Wave classifies pipeline steps into three complexity tiers based on persona and step characteristics:

| Tier | Intent | Use Case |
|------|--------|----------|
| `cheapest` | Cost-optimized | Navigation, summarization, scanning |
| `fastest` | Latency-optimized | Balanced speed/cost for standard tasks |
| `strongest` | Capability-optimized | Complex reasoning, code generation |

## Adapter Tier Models

Each adapter can define tier-specific model mappings:

```yaml
adapters:
  claude:
    binary: claude
    default_model: sonnet
    tier_models:
      cheapest: haiku
      fastest: ""
      strongest: opus
  opencode:
    binary: opencode
    default_model: opencode/big-pickle
    tier_models:
      cheapest: opencode/big-pickle
      fastest: opencode/big-pickle
      strongest: opencode/big-pickle
```

When `auto_route: true` is enabled, Wave uses these tier mappings to select models automatically.

## Auto-Routing

Enable automatic model selection based on step complexity:

```yaml
routing:
  auto_route: true
  complexity_map:
    cheapest: haiku
    fastest: ""
    strongest: opus
```

Override specific tiers in `wave.yaml` routing section, or use adapter-level `tier_models` for per-adapter mappings.

## Per-Step Model Assignment

```yaml
steps:
  - id: analyze
    persona: navigator
    model: haiku          # explicit override

  - id: implement
    persona: craftsman
    # no model — uses adapter tier model based on complexity

  - id: review
    persona: reviewer
    model: haiku          # cheap model for review
```

## Adapter Resolution (strongest to weakest)

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

Use `--adapter <name>` to override the adapter for all steps in a run:

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
model: haiku
adapter: claude
```
