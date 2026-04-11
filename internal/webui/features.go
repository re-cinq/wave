package webui

import "net/http"

// Features tracks which optional features are compiled into the binary.
// Each feature is enabled via a build tag (e.g., go build -tags analytics).
// Default build excludes all optional features.
type Features struct {
	Metrics   bool
	Analytics bool
	Webhooks  bool
	Ontology  bool
}

// EnabledFeatures is the global feature flag state, populated by init()
// functions in build-tagged feature files.
var EnabledFeatures Features

// featureRouteFunc registers routes for an optional feature.
type featureRouteFunc func(s *Server, mux *http.ServeMux)

var featureRoutes []featureRouteFunc

// registerFeatureRoutes adds a route registration function that will be
// called during server startup. Each build-tagged feature file calls this
// in its init() to wire its routes into the mux.
func registerFeatureRoutes(fn featureRouteFunc) {
	featureRoutes = append(featureRoutes, fn)
}
