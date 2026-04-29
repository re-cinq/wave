package webui

import "net/http"

// Features tracks which optional features are compiled into the binary.
// Each feature is enabled via a build tag (e.g., go build -tags analytics).
// Default build excludes all optional features.
type Features struct {
	Metrics   bool
	Analytics bool
	Webhooks  bool
	Retros    bool
}

// featureRouteFunc registers routes for an optional feature.
type featureRouteFunc func(s *Server, mux *http.ServeMux)

// FeatureRegistry owns feature-flag state and the route hooks contributed by
// build-tagged feature files. Each Server instance gets its own registry,
// eliminating package-level mutable global state.
type FeatureRegistry struct {
	Features Features
	routeFns []featureRouteFunc
}

// NewFeatureRegistry constructs a registry by invoking every per-feature
// register<Name> function. With default build tags, those calls are no-ops
// (provided by features_<name>_disabled.go stubs); with the matching tag,
// the real file populates flags and route hooks.
func NewFeatureRegistry() *FeatureRegistry {
	r := &FeatureRegistry{}
	registerAnalytics(r)
	registerMetrics(r)
	registerWebhooks(r)
	registerRetros(r)
	return r
}

// addRoutes appends a route registration function. Called by per-feature
// register<Name> functions.
func (r *FeatureRegistry) addRoutes(fn featureRouteFunc) {
	r.routeFns = append(r.routeFns, fn)
}
