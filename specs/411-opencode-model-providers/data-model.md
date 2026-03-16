# Data Model: Third-Party Model Providers via OpenCode Adapter

**Branch**: `411-opencode-model-providers` | **Date**: 2026-03-16

## Entities

### ProviderModel (new, internal to adapter)

Represents a parsed provider/model identifier.

```go
type ProviderModel struct {
    Provider string // e.g., "openai", "google", "anthropic"
    Model    string // e.g., "gpt-4o", "gemini-pro", "claude-sonnet-4-20250514"
}
```

**Source**: Parsed from `AdapterRunConfig.Model` string.

### Provider Inference Map (new, package-level)

```go
var knownModelPrefixes = map[string]string{
    "gpt-":    "openai",
    "gemini-": "google",
    "claude-": "anthropic",
}
```

**Usage**: When model string has no `/` prefix, iterate prefixes to infer provider.

### OpenCode Config (existing, modified)

The `.opencode/config.json` file generated in workspace:

```json
{
    "provider": "<resolved-provider>",
    "model": "<resolved-model>",
    "temperature": 0.7
}
```

**Change**: `provider` and `model` are now dynamic (from `cfg.Model`) instead of hardcoded.

### AdapterRunConfig (existing, no changes needed)

```go
type AdapterRunConfig struct {
    Model          string   // Already carries resolved model from executor
    EnvPassthrough []string // Already carries passthrough list from manifest
    Env            []string // Already carries step-specific env vars
    // ... other fields unchanged
}
```

No schema changes needed — all required fields already exist.

## Functions (new/modified)

### `ParseProviderModel(model string) ProviderModel` (new)

Exported function in `adapter` package. Parses `"provider/model"` format:
1. If `model` contains `/`, split on first `/` → provider + model name
2. If no `/`, check `knownModelPrefixes` for inference
3. If no match, return default provider `"anthropic"`

### `BuildCuratedEnvironment(cfg AdapterRunConfig) []string` (new)

Exported package-level function. Extracts shared logic from `ClaudeAdapter.buildEnvironment`:
- Base: `HOME`, `PATH`, `TERM`, `TMPDIR`
- Plus: `EnvPassthrough` vars present in host env
- Plus: `cfg.Env` step-specific vars

### `ClaudeAdapter.buildEnvironment` (modified)

Calls `BuildCuratedEnvironment(cfg)` then appends Claude-specific telemetry vars.

### `OpenCodeAdapter.prepareWorkspace` (modified)

Uses `ParseProviderModel(cfg.Model)` to resolve provider and model for `config.json`.

### `OpenCodeAdapter.Run` (modified)

Replaces `os.Environ()` with `BuildCuratedEnvironment(cfg)`.

## Relationships

```
AdapterRunConfig.Model → ParseProviderModel() → ProviderModel
                                                   ├── .Provider → config.json "provider"
                                                   └── .Model    → config.json "model"

AdapterRunConfig → BuildCuratedEnvironment() → []string (env vars)
                                                   ↓
                                           ClaudeAdapter.Run (+ telemetry vars)
                                           OpenCodeAdapter.Run (direct use)
```
