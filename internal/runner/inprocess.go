package runner

import (
	"github.com/recinq/wave/internal/config"
	"github.com/recinq/wave/internal/manifest"
)

// manifestDefaultsFromManifest adapts a *manifest.Manifest into the
// config.ManifestDefaults interface used by config.Resolve. The interface
// indirection exists because internal/config cannot import internal/manifest
// (manifest transitively imports config via hooks/scope, which would cycle).
//
// A nil manifest yields a ManifestDefaults that returns zero values; Resolve
// treats that the same as a zero-valued manifest, falling through to the
// hard-coded adapter and timeout fallbacks.
func manifestDefaultsFromManifest(m *manifest.Manifest) config.ManifestDefaults {
	if m == nil {
		return nil
	}
	return config.ManifestDefaultsFunc{
		FirstAdapterFn: func() string {
			// Map iteration order is non-deterministic; the previous
			// implementation relied on this. Preserve semantics by
			// returning the first key Go yields — callers that pin a
			// specific adapter must use --adapter explicitly.
			for adapterName := range m.Adapters {
				return adapterName
			}
			return ""
		},
		DefaultTimeoutFn: m.Runtime.GetDefaultTimeout,
	}
}
