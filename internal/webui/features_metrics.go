//go:build metrics

package webui

func registerMetrics(r *FeatureRegistry) {
	r.Features.Metrics = true
	// Metrics has no standalone routes — it's a tab in the run detail page.
	// The template conditionally renders the tab based on Features.Metrics.
}
