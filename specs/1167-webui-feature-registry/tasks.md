# Work Items

## Phase 1: Setup
- [ ] 1.1: Read current `internal/webui/features.go`, `features_*.go`, `server.go`, `routes.go` to confirm no drift since plan.
- [ ] 1.2: Confirm `EnabledFeatures` / `RegisterFeatureRoutes` have zero external callers (`grep -r "webui\.EnabledFeatures\|webui\.RegisterFeatureRoutes"`).

## Phase 2: Core Implementation
- [ ] 2.1: Rewrite `internal/webui/features.go` — define `Features` struct (kept), `FeatureRegistry` struct with `Features` field + `routeFns []func(*Server, *http.ServeMux)`, `NewFeatureRegistry()` constructor calling all `register<Name>` funcs, `addRoutes()` helper. Remove `EnabledFeatures` global, `featureRoutes` slice, `RegisterFeatureRoutes`.
- [ ] 2.2: Convert `internal/webui/features_analytics.go` — replace `init()` with `registerAnalytics(r *FeatureRegistry)` setting `r.Features.Analytics = true` and calling `r.addRoutes(...)`. [P]
- [ ] 2.3: Convert `internal/webui/features_metrics.go` — `registerMetrics(r *FeatureRegistry)` (flag only, no routes). [P]
- [ ] 2.4: Convert `internal/webui/features_webhooks.go` — `registerWebhooks(r *FeatureRegistry)`. [P]
- [ ] 2.5: Convert `internal/webui/features_ontology.go` — `registerOntology(r *FeatureRegistry)`. [P]
- [ ] 2.6: Convert `internal/webui/features_retros.go` — `registerRetros(r *FeatureRegistry)`. [P]
- [ ] 2.7: Create `features_analytics_disabled.go` with `//go:build !analytics` and empty `registerAnalytics`. [P]
- [ ] 2.8: Create `features_metrics_disabled.go` with `//go:build !metrics`. [P]
- [ ] 2.9: Create `features_webhooks_disabled.go` with `//go:build !webhooks`. [P]
- [ ] 2.10: Create `features_ontology_disabled.go` with `//go:build !ontology`. [P]
- [ ] 2.11: Create `features_retros_disabled.go` with `//go:build !retros`. [P]
- [ ] 2.12: Update `internal/webui/server.go` — add `features *FeatureRegistry` to `Server`, add `Features *FeatureRegistry` to `ServerConfig`. In `NewServer`, build `features := cfg.Features; if features == nil { features = NewFeatureRegistry() }` BEFORE `parseTemplates`. Rewrite `featureEnabled` closure to read `features.Features.<X>`. Store `features` on `srv`.
- [ ] 2.13: Update `internal/webui/routes.go` — iterate `s.features.routeFns` instead of package `featureRoutes`.

## Phase 3: Testing
- [ ] 3.1: Create `internal/webui/features_test.go` — assert `NewFeatureRegistry()` returns non-nil registry, `addRoutes`/registered fn invocation correctness. Build manual registries with flags + route fns and verify `*http.ServeMux` receives the expected paths.
- [ ] 3.2: Run `go build ./...` (default tags) — must compile.
- [ ] 3.3: Run `go build -tags 'analytics metrics webhooks ontology retros' ./...` — must compile (all stubs swapped for real impls).
- [ ] 3.4: Run `go test -race ./internal/webui/...` — must pass.
- [ ] 3.5: Run `go test -race ./tests/integration/...` for `server_test.go` to confirm `ServerConfig{}` (nil `Features`) still works.

## Phase 4: Polish
- [ ] 4.1: Run `golangci-lint run ./internal/webui/...` — clean.
- [ ] 4.2: Run `go vet ./...` — clean.
- [ ] 4.3: Verify acceptance criteria — `grep -rn "EnabledFeatures\|RegisterFeatureRoutes\|featureRoutes" internal/webui/` returns no hits in non-test code.
- [ ] 4.4: Commit with `refactor(webui):` scope; PR body links #1167 and lists acceptance-criteria checkboxes.
