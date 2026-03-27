# Multi-Adapter Model Routing with Per-Step Model Assignment

**Issue**: [re-cinq/wave#576](https://github.com/re-cinq/wave/issues/576)
**Labels**: enhancement
**Author**: nextlevelshit
**State**: OPEN

## Context

Wave currently uses a single adapter (Claude Code) for all pipeline steps. This issue adds multi-adapter routing: per-step adapter/model selection, new CLI adapters (Codex, Gemini), persona-level defaults, and fallback chains for provider resilience.

## Design Goals

### Per-Step Model Assignment

Pipeline steps can override the adapter and model:

```yaml
steps:
  - name: plan
    persona: navigator
    adapter: claude
    model: claude-haiku-4-5

  - name: implement
    persona: craftsman
    adapter: claude
    model: claude-sonnet-4-5

  - name: review
    persona: navigator
    adapter: openai
    model: gpt-5.3-codex
```

### Adapter Registry

Extend `internal/adapter/` to support multiple backends:

- `claude` -- Claude Code CLI (existing, default)
- `codex` -- OpenAI Codex CLI
- `gemini` -- Gemini CLI
- `api` -- Direct API calls (future, out of scope for this issue)

Each adapter implements the existing `AdapterRunner` interface. The executor selects the adapter per-step.

### Persona-Level Defaults

Personas already have `adapter` and `model` fields. Step-level fields override persona defaults.

### Fallback Chains

```yaml
runtime:
  fallbacks:
    anthropic: [openai, gemini]
    openai: [anthropic]
```

When primary provider fails with transient/quota errors, try fallbacks in order.

## What Wave Keeps

- Persona isolation -- fresh memory, per-persona CLAUDE.md, per-persona permissions
- Contract validation -- output validated regardless of which model produced it
- Workspace isolation -- ephemeral worktrees regardless of adapter
- Forge-agnostic design -- template variables work across all adapters

## What Wave Gains

- Cost optimization -- use cheap models for planning, expensive for implementation
- Model diversity -- reduce single-provider risk
- Provider resilience -- fallback chains handle outages

## Acceptance Criteria

1. Steps can specify `adapter` and `model` fields that override persona defaults
2. Codex CLI adapter (`codex.go`) implements `AdapterRunner`
3. Gemini CLI adapter (`gemini.go`) implements `AdapterRunner`
4. Executor resolves adapter per-step: step.Adapter > persona.Adapter > first manifest adapter
5. Model resolution: CLI --model > step.Model > persona.Model > empty
6. Fallback chain config in `runtime.fallbacks`, triggered on transient/quota failures
7. Preflight validates adapter references and binary availability
8. All existing tests pass; new tests cover per-step routing and fallbacks
