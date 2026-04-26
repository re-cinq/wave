# Implementation Plan: webui Feature Registry

## 1. Objective

Remove the package-level `EnabledFeatures` global and `init()`-based route registration in `internal/webui`. Replace with an explicit `FeatureRegistry` constructed at startup and injected into `Server`, so feature flags and routes are owned by the server instance rather than global package state.

## 2. Approach

Use the canonical Go build-tag-toggle pattern: each feature has two files (enabled / disabled stub) selected by build tag. Each file defines a `register<Name>(*FeatureRegistry)` function. A constructor `NewFeatureRegistry()` calls every `register<Name>` and assembles flags + route hooks into a single owned struct.

`ServerConfig` gains a `Features *FeatureRegistry` field (nil → `NewFeatureRegistry()` default). `Server` stores the registry. `routes.go` iterates `s.features.routeFns`. The `featureEnabled` template func reads `s.features.Features.<Name>` instead of the global.

This eliminates: package-level `EnabledFeatures`, `featureRoutes` slice, `RegisterFeatureRoutes`, and all `init()` functions in `features_*.go`.

## 3. File Mapping

### Modified
- `internal/webui/features.go` — remove `EnabledFeatures` global, `featureRoutes`, `RegisterFeatureRoutes`. Add `FeatureRegistry` struct, `NewFeatureRegistry()` constructor, `addRoutes()` helper.
- `internal/webui/features_analytics.go` — drop `init()`; export `registerAnalytics(*FeatureRegistry)`.
- `internal/webui/features_metrics.go` — drop `init()`; export `registerMetrics(*FeatureRegistry)`.
- `internal/webui/features_webhooks.go` — drop `init()`; export `registerWebhooks(*FeatureRegistry)`.
- `internal/webui/features_ontology.go` — drop `init()`; export `registerOntology(*FeatureRegistry)`.
- `internal/webui/features_retros.go` — drop `init()`; export `registerRetros(*FeatureRegistry)`.
- `internal/webui/server.go` — add `features *FeatureRegistry` to `Server`, add `Features *FeatureRegistry` to `ServerConfig`. Default to `NewFeatureRegistry()` when nil. `featureEnabled` reads `srv.features.Features.<X>` (closure captures the registry built before `Server` literal).
- `internal/webui/routes.go` — iterate `s.features.routeFns` instead of package `featureRoutes`.

### Created
- `internal/webui/features_analytics_disabled.go` — `//go:build !analytics`; stub `registerAnalytics(*FeatureRegistry) {}`.
- `internal/webui/features_metrics_disabled.go` — `//go:build !metrics`; stub `registerMetrics(*FeatureRegistry) {}`.
- `internal/webui/features_webhooks_disabled.go` — `//go:build !webhooks`; stub `registerWebhooks(*FeatureRegistry) {}`.
- `internal/webui/features_ontology_disabled.go` — `//go:build !ontology`; stub `registerOntology(*FeatureRegistry) {}`.
- `internal/webui/features_retros_disabled.go` — `//go:build !retros`; stub `registerRetros(*FeatureRegistry) {}`.
- `internal/webui/features_test.go` — unit tests covering: empty registry, manual enable, route function execution, multi-feature composition. No build tag (uses default constructor + manual `addRoutes` for assertions).

### Deleted
- None.

### Untouched
- `cmd/wave/commands/serve.go` — `webui.NewServer(cfg)` call works unchanged because `Features` defaults to `NewFeatureRegistry()` when nil. Optional follow-up: pass an explicit registry from main, but not required by acceptance criteria.
- `tests/integration/server_test.go` — same reasoning; nil `Features` field defaults correctly.
- All `templates/*.html` — `featureEnabled` template func name unchanged.

## 4. Architecture Decisions

- **Stub-file pattern over reflection / map-based registration.** Build-tag stubs keep the call graph statically resolvable, preserve zero-cost when features are disabled, and avoid runtime registration ordering questions. This is idiomatic Go for binary-time feature toggles.
- **Registry owned by `Server`, not exported via `cmd/`.** `ServerConfig.Features` is exported so tests / future main wiring can inject custom sets, but the default path (`NewFeatureRegistry()`) means no caller must change.
- **Closure capture of registry in `featureEnabled` template func.** The func is defined inside `NewServer` and captures the local `features *FeatureRegistry` before storing it on `Server`. Avoids needing a method on `Server` and keeps template func setup co-located.
- **Per-feature register functions take `*FeatureRegistry`.** Each file mutates flags and appends route fns through a single seam (`addRoutes`). Symmetric with the disabled stubs.
- **Naming.** `registerAnalytics` (lowercase) — package-internal. Not exported because no caller outside the package needs to invoke individual feature setup.

## 5. Risks

- **Risk: stub file forgotten for a feature.** Build with default tags fails to compile (undefined `registerX`). Mitigation: stub files mirror enabled files 1:1, lint catches missing functions, CI default build (no tags) exercises stubs.
- **Risk: build matrix coverage.** Need to verify both `go build` (no tags) and `go build -tags 'analytics webhooks ontology retros metrics'` succeed. Mitigation: run both locally; existing CI already builds default tag.
- **Risk: external callers reading `webui.EnabledFeatures`.** Confirmed via grep: only `internal/webui` references it. No external callers.
- **Risk: template func registration timing.** Template parsing happens after `features` registry is built but before `Server` struct literal. Mitigation: hoist registry construction above `parseTemplates` call.

## 6. Testing Strategy

- **Unit (no build tag):** `features_test.go` — tests `NewFeatureRegistry()` returns zero-flag registry by default tags; tests `addRoutes` accumulates fns; tests calling each route fn against a `*http.ServeMux` registers expected paths.
- **Build-tag smoke:** `go build -tags <feature>` for each feature individually (existing pattern, no new tests required) — verified manually before PR.
- **Integration:** existing `tests/integration/server_test.go` continues to work unchanged because `ServerConfig.Features` defaults to `NewFeatureRegistry()`. No global state means `go test -race` is no longer racy on parallel server constructions.
- **Race:** `go test -race ./internal/webui/...` — no shared mutable globals after refactor.
- **Coverage target:** maintain or improve current `internal/webui` line coverage; new `features_test.go` adds direct coverage for the registry seam that was previously untested.
