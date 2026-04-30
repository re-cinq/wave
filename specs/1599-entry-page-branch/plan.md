# Implementation Plan — #1599 Entry-page smart redirect

## Objective

Replace the static `GET /{$}` → `/runs` redirect in the webui with a sentinel-aware handler that routes `/` to `/onboard` (when `.agents/.onboarding-done` is missing) or `/work` (when present).

## Approach

- Extract the `GET /{$}` handler from `internal/webui/routes.go` into a new file `internal/webui/handlers_root.go` as `Server.handleRoot`.
- `handleRoot` joins `s.runtime.repoDir` with `onboarding.SentinelFile` (`.agents/.onboarding-done`) and `os.Stat`s it. Missing → 302 `/onboard`; present → 302 `/work`.
- Update `routes.go` to wire `mux.HandleFunc("GET /{$}", s.handleRoot)`.
- Add `handlers_root_test.go` covering both branches via `t.TempDir()` fixtures plus a real `httptest.NewRecorder` round-trip on a `Server` whose `runtime.repoDir` points at the fixture.

## File Mapping

Created:
- `internal/webui/handlers_root.go` — `handleRoot` method.
- `internal/webui/handlers_root_test.go` — table test for sentinel present / missing / stat-error fallback.

Modified:
- `internal/webui/routes.go` — replace inline lambda with `s.handleRoot` reference.

Deleted: none.

## Architecture Decisions

- **Sentinel detection lives in handler, not middleware.** Only `/` cares about the sentinel; a middleware would add per-request stat overhead to every route.
- **Fallback on stat error → `/onboard`.** Permission errors or unreadable parent dirs should treat the project as not-yet-onboarded (safest default — operator can finish onboarding rather than land on an empty work board).
- **Reuse `onboarding.SentinelFile` constant** from `internal/onboarding/service.go` rather than hardcoding the path again.
- **`repoDir` is already on `serverRuntime`** (server.go:71, populated from `git rev-parse --show-toplevel`). No new plumbing needed.
- **No build-tag gating.** This is a routing concern, not a feature-flagged surface.

## Risks

- **Sentinel created mid-session.** If onboarding completes while the user has `/` open, they'd need to refresh; acceptable — `/` is not a long-lived page.
- **`repoDir` empty in test contexts.** Mitigation: handler treats empty `repoDir` as `"."` (matches existing `handlers_onboard.go:352` pattern).
- **Symlinked `.agents` dirs.** `os.Stat` follows symlinks, which is what we want.

## Testing Strategy

Unit test in `handlers_root_test.go`:
1. **Sentinel present** → expect `302 /work`. Setup: `t.TempDir()` + `os.MkdirAll(.agents)` + create the sentinel file. Build a minimal `Server{runtime: serverRuntime{repoDir: tmp}}` and invoke `handleRoot` directly.
2. **Sentinel missing** → expect `302 /onboard`. Setup: bare `t.TempDir()`.
3. **Verify no `/runs` references** appear in the redirect target for either path.

Use `httptest.NewRecorder` + `httptest.NewRequest("GET", "/", nil)`. Assert `rec.Code == 302` and `rec.Header().Get("Location")`.

No integration test needed — `routes.go` already wires the handler and the existing route-registration test (`features_default_test.go`) confirms the mux compiles.
