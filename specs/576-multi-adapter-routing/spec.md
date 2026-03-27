# feat: multi-adapter model routing with per-step model assignment

**Issue**: [re-cinq/wave#576](https://github.com/re-cinq/wave/issues/576)
**Labels**: enhancement
**Author**: nextlevelshit
**State**: OPEN

## Context

Fabro routes different LLM models to different workflow steps via CSS-like "model stylesheets" — achieving up to 75x cost savings by using cheap models for simple tasks and expensive models for complex ones. They support 7 providers (Anthropic, OpenAI, Gemini, Kimi, GLM, Minimax, Mercury) with two backends: API-direct (Fabro manages the agent loop) and CLI (delegates to Claude Code, Codex, Gemini CLI).

Wave currently uses a single adapter (Claude Code) for all steps. This is the biggest competitive gap.

## Design Goals — Best of Both Worlds

Wave should combine Fabro's multi-model routing with Wave's persona isolation and contract validation:

### Per-Step Model Assignment (Wave-native approach)

In `wave.yaml` pipeline steps, allow model/adapter override:

```yaml
steps:
  - name: plan
    persona: navigator
    adapter: claude        # default
    model: claude-haiku-4-5

  - name: implement
    persona: craftsman
    adapter: claude
    model: claude-sonnet-4-5

  - name: review
    persona: navigator
    adapter: openai        # different provider
    model: gpt-5.3-codex
```

### Adapter Registry

Extend the existing `internal/adapter/` to support multiple backends:

- `claude` — Claude Code CLI (existing, default)
- `codex` — OpenAI Codex CLI
- `gemini` — Gemini CLI
- `api` — Direct API calls (future: Wave manages the agent loop itself like Fabro does)

Each adapter implements the existing `Adapter` interface. The executor selects the adapter per-step based on manifest config.

### Persona-Level Defaults

Personas can declare a preferred model/adapter, overridable at the step level:

```yaml
# .wave/personas/analyst.yaml
name: analyst
model: claude-haiku-4-5     # cheap by default
adapter: claude
```

### Fallback Chains

```yaml
runtime:
  fallbacks:
    anthropic: [openai, gemini]
    openai: [anthropic]
```

When primary provider fails with transient/quota errors, try fallbacks in order.

## What Wave Keeps (USPs)

- **Persona isolation** — each step still gets fresh memory, persona-specific CLAUDE.md, per-persona permissions
- **Contract validation** — output still validated against schemas regardless of which model produced it
- **Workspace isolation** — ephemeral worktrees regardless of adapter
- **Forge-agnostic design** — template variables work across all adapters

## What Wave Gains

- **Cost optimization** — use Haiku for planning/analysis, Sonnet/Opus for implementation
- **Model diversity** — reduce single-provider risk, use best model for each task
- **Provider resilience** — fallback chains handle outages

## Implementation Scope

1. Add `adapter` and `model` fields to step and persona manifest schemas
2. Implement Codex CLI adapter (`internal/adapter/codex.go`)
3. Implement Gemini CLI adapter (`internal/adapter/gemini.go`)
4. Extend executor to select adapter per-step
5. Add fallback chain logic
6. Update preflight to validate adapter/model combinations

## Acceptance Criteria

1. Pipeline steps can specify `adapter` and `model` fields that override persona defaults
2. Model resolution follows 4-tier precedence: CLI `--model` > step.model > persona.model > adapter default
3. Adapter resolution follows 3-tier precedence: step.adapter > persona.adapter > manifest default
4. Codex CLI adapter (`internal/adapter/codex.go`) implements `AdapterRunner` with workspace prep and output parsing
5. Gemini CLI adapter (`internal/adapter/gemini.go`) implements `AdapterRunner` with workspace prep and output parsing
6. Executor resolves adapter per-step instead of using a single global runner
7. Fallback chains in `runtime.fallbacks` trigger on transient/quota/rate-limit errors
8. Preflight validates all referenced adapter binaries are available
9. Manifest validation enforces adapter names reference defined adapters
10. All existing tests pass, new adapters have unit tests
