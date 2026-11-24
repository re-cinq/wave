package onboarding

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// initAdapterModels is the set of adapters `wave init` scaffolds, with the
// model each one is pinned to. Every complexity tier (cheapest, balanced,
// strongest) resolves to this model through the adapter's tier_models so a
// fresh project never silently falls back to an outdated or third-party
// model. Other adapters (opencode, gemini) remain usable but must be added
// to wave.yaml by hand.
var initAdapterModels = map[string]string{
	"claude": "sonnet",
	"codex":  "gpt-6.2-sol",
}

// initAdapterPreference is the detection order used when --adapter is not
// given: the first adapter whose binary is on PATH wins.
var initAdapterPreference = []string{"claude", "codex"}

// SupportedInitAdapters returns the adapter names `wave init` accepts, in
// preference order.
func SupportedInitAdapters() []string {
	out := make([]string, len(initAdapterPreference))
	copy(out, initAdapterPreference)
	return out
}

// DefaultModelFor returns the model `wave init` pins for adapter, or "" when
// the adapter is not one init scaffolds.
func DefaultModelFor(adapter string) string {
	return initAdapterModels[adapter]
}

// ValidateInitAdapter returns an error when adapter is not one `wave init`
// can scaffold.
func ValidateInitAdapter(adapter string) error {
	if _, ok := initAdapterModels[adapter]; ok {
		return nil
	}
	names := make([]string, 0, len(initAdapterModels))
	for n := range initAdapterModels {
		names = append(names, n)
	}
	sort.Strings(names)
	return fmt.Errorf("unsupported adapter %q: wave init supports %s (other adapters can be added to wave.yaml by hand)",
		adapter, strings.Join(names, ", "))
}

// LookPathFunc matches exec.LookPath; injectable for tests.
type LookPathFunc func(string) (string, error)

// DetectInstalledAdapters returns the supported init adapters whose binary is
// on PATH, in preference order.
func DetectInstalledAdapters(lookPath LookPathFunc) []string {
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	var found []string
	for _, name := range initAdapterPreference {
		if _, err := lookPath(name); err == nil {
			found = append(found, name)
		}
	}
	return found
}

// DefaultInitAdapter picks the adapter init uses when none is given: the
// first installed supported adapter, or claude when none is installed.
func DefaultInitAdapter(lookPath LookPathFunc) string {
	if found := DetectInstalledAdapters(lookPath); len(found) > 0 {
		return found[0]
	}
	return initAdapterPreference[0]
}

// TierModelsFor returns a tier_models map that pins every complexity tier to
// model.
func TierModelsFor(model string) map[string]interface{} {
	return map[string]interface{}{
		"cheapest":  model,
		"balanced":  model,
		"strongest": model,
	}
}

// AdapterOfManifest returns the adapter a parsed wave.yaml is configured
// for: the adapter most personas use, falling back to the only adapter
// entry. Returns "" when it cannot tell.
func AdapterOfManifest(m map[string]interface{}) string {
	counts := map[string]int{}
	if personas, ok := m["personas"].(map[string]interface{}); ok {
		for _, p := range personas {
			if pm, ok := p.(map[string]interface{}); ok {
				if a, ok := pm["adapter"].(string); ok && a != "" {
					counts[a]++
				}
			}
		}
	}
	best, bestN := "", 0
	for a, n := range counts {
		if n > bestN || (n == bestN && a < best) {
			best, bestN = a, n
		}
	}
	if best != "" {
		return best
	}
	if adapters, ok := m["adapters"].(map[string]interface{}); ok && len(adapters) == 1 {
		for a := range adapters {
			return a
		}
	}
	return ""
}

// ModelOfManifest returns adapters.<adapter>.default_model from a parsed
// wave.yaml, or "".
func ModelOfManifest(m map[string]interface{}, adapter string) string {
	adapters, ok := m["adapters"].(map[string]interface{})
	if !ok {
		return ""
	}
	a, ok := adapters[adapter].(map[string]interface{})
	if !ok {
		return ""
	}
	s, _ := a["default_model"].(string)
	return s
}

// ApplyReconfigure applies a `wave init --reconfigure` choice to a merged
// manifest map: the adapter entry gets model as default_model and for every
// tier, personas that used fromAdapter (or every persona when fromAdapter is
// empty) switch to adapter, and non-empty commands are written to project.
// Everything else in m is left as is.
func ApplyReconfigure(m map[string]interface{}, fromAdapter, adapter, model string, c ProjectCommands) map[string]interface{} {
	if m == nil {
		m = map[string]interface{}{}
	}
	adapters, _ := m["adapters"].(map[string]interface{})
	if adapters == nil {
		adapters = map[string]interface{}{}
		m["adapters"] = adapters
	}
	entry, _ := adapters[adapter].(map[string]interface{})
	if entry == nil {
		entry = map[string]interface{}{"binary": adapter, "mode": "headless"}
		adapters[adapter] = entry
	}
	if model != "" {
		entry["default_model"] = model
		entry["tier_models"] = TierModelsFor(model)
	}

	if personas, ok := m["personas"].(map[string]interface{}); ok {
		for _, p := range personas {
			pm, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			if cur, _ := pm["adapter"].(string); fromAdapter == "" || cur == fromAdapter || cur == "" {
				pm["adapter"] = adapter
			}
		}
	}

	// Drop the adapter we switched away from once no persona uses it, so
	// `wave doctor` does not keep checking a binary the project no longer
	// needs (often the reason for switching).
	if fromAdapter != "" && fromAdapter != adapter && !adapterInUse(m, fromAdapter) {
		delete(adapters, fromAdapter)
	}

	project, _ := m["project"].(map[string]interface{})
	if project = ApplyProjectCommands(project, c); len(project) > 0 {
		m["project"] = project
	}
	return m
}

func adapterInUse(m map[string]interface{}, adapter string) bool {
	personas, _ := m["personas"].(map[string]interface{})
	for _, p := range personas {
		if pm, ok := p.(map[string]interface{}); ok {
			if a, _ := pm["adapter"].(string); a == adapter {
				return true
			}
		}
	}
	return false
}
