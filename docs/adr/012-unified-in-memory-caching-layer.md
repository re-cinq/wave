# ADR-012: Introduce a Unified In-Memory Caching Layer

## Status
Proposed

## Date
2026-04-20

## Context
Wave is a Go 1.25+ multi-agent pipeline orchestrator that wraps Claude Code and other LLM CLIs via subprocess execution. It ships as a single static binary with `modernc.org/sqlite` persistence and ephemeral, run-scoped workspaces. Today the codebase contains three unrelated ad-hoc caches: a TTL-based in-memory map in `internal/webui/cache.go` (RWMutex, prefix invalidation, `POST /api/cache/refresh` endpoint), prompt-cache token accounting in `internal/adapter/claude.go` (tracks `cache_read_input_tokens` and `cache_creation_input_tokens` for cost calculation), and memoized TypeScript schema parsing in `internal/contract/typescript.go`. There is no shared abstraction, no configurable cache block in `wave.yaml`, no persistent backend, and no consistent key/invalidation discipline across packages.

The cost of this fragmentation is rising. Several expensive hotspots are entirely uncached: Claude subprocess calls (5–120 s latency, $0.01–$1+ per call), relay context compaction/summarization, manifest/persona/pipeline parsing on every CLI invocation, the skill directory walk, artifact injection file I/O, and the TUI list providers (`internal/tui/contract_provider.go`, `health_provider.go`, `pipeline_detail_provider.go`). Each new hotspot currently acquires its own local memoization, diverging invalidation rules, and no visibility into hit/miss rates.

Several recently-proposed ADRs make this decision time-sensitive but also constrained. ADR-006 (cost-infrastructure) defers dynamic routing, the decision log, and Iron Rule enforcement pending real cost data — data that cache hit/miss counters would produce. ADR-007 (StateStore consolidation) mandates that any new SQLite access go through the single StateStore handle, so a persistent cache cannot simply open its own database. ADR-002 targets a 40–50% complexity reduction in `internal/pipeline/executor.go` (6 469 lines); threading caching through the executor before the StepExecutor extraction would enlarge that refactor. ADR-005 replaces `TopologicalSort` with a graph `NextSteps` model, meaning any invalidation strategy tied to step ordering must be graph-aware.

A decision is needed now to stop further divergence (every new hotspot ships its own cache), to unblock ADR-006 observability with a concrete hit/miss signal, and to pick an approach that does not pre-empt ADR-007, ADR-002, or ADR-005 sequencing.

## Decision
Adopt a unified in-memory cache package at `internal/cache`, extracted from the proven `internal/webui/cache.go` pattern. The package exposes typed `Get`/`Set`/`Invalidate` operations with per-tier TTL, prefix invalidation, `singleflight` deduplication of concurrent identical lookups, and hit/miss counters wired to the cost ledger. Configuration lives in a new `runtime.cache` block in `wave.yaml` (enable flags and TTLs per tier, mirroring the existing `runtime.timeouts` and `runtime.pricing` patterns). The existing webui and TypeScript caches migrate onto it; `internal/adapter`, `internal/relay`, `internal/manifest`, `internal/skill`, and the TUI providers opt in incrementally.

This option wins on risk/value against the current ADR queue. It uses only Go stdlib plus `golang.org/x/sync` (already in `go.mod`), preserves the single-binary deployment model, and introduces no SQLite schema — so it does not couple to ADR-007 StateStore sequencing or to the ADR-002 executor extraction. `singleflight` alone delivers a measurable win for parallel pipeline steps that issue identical Claude prompts within a run. Exposing hit/miss metrics is the concrete feed ADR-006 needs to decide whether a persistent backend is worth the correctness cost later. Persistence is deliberately out of scope: LLM cache-key correctness (prompt + context + model hash invalidating on codebase/manifest change) and Claude SDK prompt-cache coordination (avoiding double-count in `internal/cost`) are hard problems that should wait for real cost data from ADR-006 before being tackled.

## Options Considered

### Option 1: Status quo — keep ad-hoc per-package caches
Retain the three existing caches and let any new hotspot add its own local memoization, as `internal/contract/typescript.go` already does.

Pros: zero new code; no coupling to ADR-002/005/007; no interaction with the Claude SDK prompt cache, so no double-count risk in `internal/cost`; respects ADR-006's deferral of speculative optimization.

Cons: expensive hotspots stay uncached (Claude subprocess calls, relay compaction, manifest and skill parsing, artifact I/O); no persistence, so manifest parse and skill walk are cold on every CLI run; ADR-006 gets no hit/miss signal and cannot measure savings; invalidation discipline continues to diverge with every new local cache.

Effort: trivial. Reversibility: easy — this is the current state.

### Option 2: Unified in-memory cache package (`internal/cache`) — recommended
Extract the `internal/webui/cache.go` pattern into a reusable package with typed API, per-tier TTL, prefix invalidation, `singleflight`, and cost-ledger hooks. Add `runtime.cache` in `wave.yaml`. Migrate webui and TypeScript memoization; expose opt-in hooks for adapter, relay, manifest, and skill. No persistent backend.

Pros: single static binary preserved (stdlib + `golang.org/x/sync` only); leverages an already-accepted pattern, minimising novelty risk; `singleflight` collapses concurrent identical LLM prompts across parallel steps within a run; hit/miss counters feed ADR-006 directly; fits ADR-003 layering as a low-level utility package; touches no SQLite schema, so independent of ADR-007.

Cons: no cross-invocation persistence — manifest parse and skill walk are still cold on every `wave` CLI run; LLM response cache has limited value for one-shot CLI runs because the process exits immediately; the long-lived webui process needs a bounded eviction policy beyond TTL to avoid memory growth; integration in `internal/adapter/claude.go` must carefully separate SDK `cache_read` tokens from app-layer hits so `internal/cost` does not double-count.

Effort: medium. Reversibility: moderate — callers depend on the package, but the implementation can be swapped or disabled via `runtime.cache.enabled`.

### Option 3: Persistent SQLite-backed cache via StateStore
Add a `cache_entries` table behind `internal/state.StateStore` (per ADR-007) with key, BLOB value, tier tag, `created_at`, `expires_at`, `hit_count`. Build `internal/cache` with in-memory L1 and SQLite L2. Key schema hashes prompt+context+model for LLM entries, path+mtime for files, and config-hash for manifest. Database lives at a stable path (XDG cache or `.agents/cache.db`), not in an ephemeral workspace.

Pros: survives CLI invocations, so manifest parse, skill walk, and TypeScript schema warm on the second run — a real user-visible latency win; LLM response cache becomes genuinely useful for repeated dev loops that reissue the same prompt and context; routes through StateStore as ADR-007 mandates; hit/miss ledger joins trivially with the cost ledger in the same SQLite file; stable-path placement respects the ephemeral-workspace constraint.

Cons: couples directly to ADR-007 sequencing, which is still Proposed — this option either blocks or duplicates integration work; LLM cache-key correctness is semantically hard, and false hits produce stale replies (a correctness bug, not a performance bug); must coordinate with SDK prompt-cache tokens at `internal/adapter/claude.go:434-610` to avoid cost double-count; SQLite writes on every subprocess call add I/O to the hot path and need benchmarking; cache schema becomes part of StateStore migrations, which is effectively irreversible once users have populated caches; scope creep risk into `internal/pipeline/executor.go` that ADR-002 wants to extract first.

Effort: large. Reversibility: difficult.

### Option 4: Filesystem-keyed content-addressed cache (no DB)
Back `internal/cache` with content-addressed files under `XDG_CACHE_HOME/wave/` (`sha256(key)` → file) plus an in-memory L1 for hot keys. No SQLite dependency and no StateStore coupling. Pruning via LRU mtime sweep on startup. Configure via `runtime.cache.dir` and `max_size_mb`.

Pros: persistent without ADR-007 coupling, so ships independently; single binary preserved (only `os`, `path/filepath`, `crypto/sha256` from stdlib); content-addressing sidesteps invalidation for immutable inputs (file bytes, schemas keyed by content hash); natural fit for large LLM response blobs without bloating SQLite's page cache; `rm -rf` on the cache directory is a full reset.

Cons: two persistence mechanisms (SQLite for state, filesystem for cache) violate the ADR-007 consolidation spirit; no transactional guarantees — partial writes possible on crash, requiring tmp+rename discipline; LRU pruning is custom code, easy to get wrong and risking unbounded growth; no query surface, so the cost ledger cannot join hit statistics against run rows without bespoke indexing; per-entry metadata (hit counts, TTL) needs sidecar files or a JSON index, re-implementing what SQLite gives for free.

Effort: medium. Reversibility: easy.

### Option 5: Expand SDK-level prompt caching only (adapter-scoped)
Do not build a general cache. Instead, deepen use of the Claude SDK's native prompt caching in `internal/adapter/claude.go`: mark stable prefixes (system prompt, manifest digest, persona) with `cache_control`, let Anthropic's API handle TTL and invalidation, and surface `cache_read_input_tokens` savings in the ADR-006 cost ledger. Leave the webui and TypeScript caches untouched.

Pros: zero app-layer cache-correctness burden — Anthropic owns TTL and invalidation; infrastructure is already half-built, since `cache_read`/`cache_creation` token accounting exists at `claude.go:434-610`; immediate cost reduction on the most expensive hotspot (LLM calls) without introducing any new package; no coupling to ADR-002, ADR-005, or ADR-007; produces real cached-token telemetry for ADR-006.

Cons: does nothing for non-LLM hotspots (manifest parse, skill walk, artifact I/O, TUI providers stay cold); only applies to the Claude adapter — other LLM CLIs added per `docs/guides/adapter-development.md` gain nothing; Anthropic's server-side cache TTL (~5 minutes) limits cross-run benefit; does not address the underlying fragmentation of ad-hoc caches.

Effort: small. Reversibility: easy.

## Consequences

### Positive
- Ends cache fragmentation: `internal/webui/cache.go`, the `internal/contract/typescript.go` memoization, and future hotspots share one typed API, one invalidation model, and one TTL policy source.
- `singleflight` deduplication eliminates redundant concurrent Claude subprocess calls within a single pipeline run; parallel steps that compute the same prompt collapse to one 5–120 s invocation instead of N.
- Cache hit/miss counters flow into `internal/cost`, giving ADR-006 the concrete data it is waiting on to decide on dynamic routing and Iron Rule enforcement.
- TUI list providers (persona, pipeline, skill, contract) gain a process-scoped cache, cutting repeat-navigation latency inside a single TUI session to near-zero.
- No new runtime dependency, no SQLite schema change — the single static binary and existing `go.mod` stand unchanged.

### Negative
- No cross-invocation persistence: every `wave` CLI run re-parses `wave.yaml`, re-walks the skill directory, and re-parses TypeScript schemas. Mitigation: measure the aggregate cold-start cost; if it exceeds 50 ms per run, revisit option 3 once ADR-007 lands.
- LLM response caching in a one-shot CLI process has limited benefit (cache dies at exit). Mitigation: scope LLM caching to the webui and to long-running pipeline runs where `singleflight` within-run dedup is the primary win.
- Long-lived webui process memory grows with unique cache keys over time; TTL alone is insufficient. Mitigation: add a bounded max-entries policy with LRU eviction in the initial implementation; emit a cache-size metric.
- Integration in `internal/adapter/claude.go` must carefully separate SDK `cache_read_input_tokens` from app-layer hits to avoid double-counting cost. Mitigation: dedicated unit test covering a synthetic response with both `cache_read` tokens and an app-layer hit; assert `internal/cost` records each exactly once.

### Neutral
- `wave.yaml` gains a new top-level `runtime.cache` block (enable flag, per-tier TTL, max size); existing configs without it must default to current behavior.
- `POST /api/cache/refresh` endpoint moves from `internal/webui` to the shared package but keeps its URL and semantics.
- New package `internal/cache` enters the layered architecture below domain packages (consistent with ADR-003).
- An opt-in migration path for adapter/relay/manifest/skill means the ADR does not dictate ordering; those packages adopt the cache as their maintainers judge.

## Implementation Notes

Execution order:
1. Create `internal/cache` package. Lift the TTL map, RWMutex, prefix invalidation, and lazy expiration from `internal/webui/cache.go`. Add a typed generic API (`Get[T]`, `Set[T]`), `singleflight.Group` for dedup, bounded LRU eviction, and atomic hit/miss counters.
2. Add `runtime.cache` block to `wave.yaml` (fields: `enabled`, per-tier `ttl`, `max_entries`). Wire into `internal/manifest` config loading. Default `enabled: true` with conservative TTLs.
3. Migrate `internal/webui/cache.go` and `internal/webui/handlers_cache.go` to consume `internal/cache`. Preserve `POST /api/cache/refresh` semantics and route registration in `internal/webui/routes.go`.
4. Migrate `internal/contract/typescript.go` memoization to `internal/cache` with a `contract:` key prefix.
5. Wire hit/miss counters into `internal/cost`. Add a `cost.RecordCacheEvent` (or equivalent) and ensure it records separately from SDK `cache_read_input_tokens` already tracked in `internal/adapter/claude.go:434-610`.
6. Opt-in adopt in `internal/manifest` (parsed `wave.yaml`, persona, pipeline keyed by file hash), `internal/skill/store.go` (directory walk result keyed by dir mtime), and TUI providers (`internal/tui/contract_provider.go`, `health_provider.go`, `pipeline_detail_provider.go`).
7. Defer `internal/adapter/claude.go`, `internal/relay/relay.go`, and `internal/pipeline/executor.go` adoption. Relay and executor should wait on ADR-002's StepExecutor extraction to avoid enlarging that refactor.

Files changed or added:
- new: `internal/cache/cache.go`, `internal/cache/cache_test.go`, `internal/cache/singleflight.go`
- modified: `internal/webui/cache.go`, `internal/webui/handlers_cache.go`, `internal/webui/routes.go`, `internal/contract/typescript.go`, `internal/manifest/*`, `internal/skill/store.go`, `internal/tui/contract_provider.go`, `internal/tui/health_provider.go`, `internal/tui/pipeline_detail_provider.go`, `internal/cost/*`, `wave.yaml`, `docs/guides/adapter-development.md` (note for future adapter authors)
- not yet touched: `internal/adapter/claude.go`, `internal/relay/relay.go`, `internal/pipeline/executor.go`

Migration: callers of the old webui cache move to the new package in a single commit alongside the package creation; the public HTTP surface is preserved. No data migration needed (no persistence). If `runtime.cache.enabled: false`, all operations degrade to direct computation, giving a safe off-switch.

Testing:
- Unit tests in `internal/cache` for TTL expiry, prefix invalidation, `singleflight` dedup under concurrent callers, LRU eviction at `max_entries`, and nil-safe receiver.
- Regression test preserving existing `POST /api/cache/refresh` behavior.
- Cost double-count test: simulate a Claude response carrying `cache_read_input_tokens` and an `internal/cache` hit on the same call, assert `internal/cost` records each event exactly once.
- Benchmark: cold vs warm manifest parse and skill walk; record aggregate per-run cold-start cost as the signal for whether ADR-007-based persistence (option 3) is justified later.
- Race detector (`go test -race`) on concurrent `Get`/`Set`/`Invalidate` paths.
