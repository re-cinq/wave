package adapter

import (
	"os"
	"strings"
)

// ProviderModel represents a parsed provider/model identifier.
type ProviderModel struct {
	Provider string // e.g., "openai", "google", "anthropic"
	Model    string // e.g., "gpt-4o", "gemini-pro", "claude-sonnet-4-20250514"
}

// knownModelPrefixes maps well-known model name prefixes to their providers.
var knownModelPrefixes = map[string]string{
	"gpt-":    "openai",
	"gemini-": "google",
	"claude-": "anthropic",
}

// defaultProvider is used when no provider can be inferred.
const defaultProvider = "anthropic"

// defaultModel is used when no model is specified.
const defaultModel = "claude-sonnet-4-20250514"

// ParseProviderModel splits a model identifier into provider and model components.
// Supported formats:
//   - "provider/model" → ProviderModel{Provider: "provider", Model: "model"}
//   - "provider/org/model" → ProviderModel{Provider: "provider", Model: "org/model"}
//   - "gpt-4o" → ProviderModel{Provider: "openai", Model: "gpt-4o"} (inferred)
//   - "" → ProviderModel{Provider: "anthropic", Model: "claude-sonnet-4-20250514"} (defaults)
func ParseProviderModel(model string) ProviderModel {
	if model == "" {
		return ProviderModel{Provider: defaultProvider, Model: defaultModel}
	}

	// Split on first "/" for explicit provider prefix
	if idx := strings.Index(model, "/"); idx > 0 {
		return ProviderModel{
			Provider: model[:idx],
			Model:    model[idx+1:],
		}
	}

	// Infer provider from well-known model name prefixes
	for prefix, provider := range knownModelPrefixes {
		if strings.HasPrefix(model, prefix) {
			return ProviderModel{Provider: provider, Model: model}
		}
	}

	// Unknown model without prefix — default to anthropic
	return ProviderModel{Provider: defaultProvider, Model: model}
}

// BuildCuratedEnvironment constructs a curated environment for adapter subprocesses.
// It includes base variables (HOME, PATH, TERM, TMPDIR), explicitly allowed
// passthrough variables from the manifest, and step-specific env vars.
func BuildCuratedEnvironment(cfg AdapterRunConfig) []string {
	env := []string{
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + os.Getenv("PATH"),
		"TERM=" + getenvDefault("TERM", "xterm-256color"),
		"TMPDIR=/tmp",
	}

	// Add explicitly allowed env vars from manifest
	for _, key := range cfg.EnvPassthrough {
		if val := os.Getenv(key); val != "" {
			env = append(env, key+"="+val)
		}
	}

	// Step-specific env vars (from pipeline config)
	env = append(env, cfg.Env...)
	return env
}
