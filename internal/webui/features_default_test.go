//go:build !analytics && !metrics && !webhooks && !retros && !webui_preview

package webui

import "testing"

// TestNewFeatureRegistryDefaultTagsZeroFlags verifies that under default build
// tags (no optional features) the registry reports every feature as disabled
// and contributes no route hooks. This locks in the "disabled stubs are no-ops"
// contract. Gated to default tags only — under any feature tag the matching
// register<Name> populates flags/routes by design.
func TestNewFeatureRegistryDefaultTagsZeroFlags(t *testing.T) {
	r := NewFeatureRegistry()
	if r.Features.Metrics {
		t.Error("default registry: Metrics should be false")
	}
	if r.Features.Analytics {
		t.Error("default registry: Analytics should be false")
	}
	if r.Features.Webhooks {
		t.Error("default registry: Webhooks should be false")
	}
	if r.Features.Retros {
		t.Error("default registry: Retros should be false")
	}
	if len(r.routeFns) != 0 {
		t.Errorf("default registry: expected 0 route fns, got %d", len(r.routeFns))
	}
}
