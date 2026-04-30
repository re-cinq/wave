# Implementation Plan — 1595 Dispatch Wiring

## 1. Objective

Add a `POST /work/{forge}/{owner}/{repo}/{number}/dispatch` endpoint that turns a work item into a real pipeline run by resolving matching `worksource` bindings, selecting a pipeline, serializing the `work_item_ref` as run input, and launching via the existing `runner.LaunchInProcess` plumbing.

## 2. Approach

1. Build a `worksource.WorkItemRef` from path params, querying the configured forge client for the live issue (title/labels/state).
2. Call `worksource.Service.MatchBindings(ctx, ref)` to get candidate bindings.
3. Pick the pipeline:
   - 0 matches → 409 Conflict.
   - 1 match → use that binding's `PipelineName`.
   - >1 matches → require a `pipeline` form/query param; 400 if missing/not in match set.
4. Marshal the `work_item_ref` (shared schema shape, #1590) as the run input string.
5. Create the run via `runtime.rwStore.CreateRun(pipelineName, input)`.
6. Launch via `s.launchPipelineExecution(runID, pipelineName, input, opts)` — same path as `/api/issues/start`, so `runner.BuildExecutorOptions` is preserved.
7. Respond with `302 Found` to `/runs/{runID}`.

## 3. File Mapping

### Created

- `internal/webui/handlers_work_dispatch.go` — new handler + helpers (`buildWorkItemRef`, `serializeWorkItemRef`, `selectBindingPipeline`).
- `internal/webui/handlers_work_dispatch_test.go` — round-trip handler tests.
- `specs/1595-dispatch-wiring/{spec,plan,tasks}.md` — planning artifacts.

### Modified

- `internal/webui/server.go` — add `worksource worksource.Service` to `serverRuntime`; construct in `NewServer` from `rwStore`.
- `internal/webui/routes.go` — register `POST /work/{forge}/{owner}/{repo}/{number}/dispatch`.

### Not Touched

- `internal/worksource/*` — already merged (#1591).
- `internal/runner/*` — reused as-is.
- `internal/contract/schemas/shared/work_item_ref.json` — input artifact only.

## 4. Architecture Decisions

- **Path namespace**: `/work/...` (not `/api/work/...`) per the issue spec, because the success path returns a 302 — implying browser-form usage. JSON callers can still POST and follow the redirect.
- **WorkItemRef construction**: the handler is the source of truth. It populates `Forge`, `Repo`, `Kind=issue`, `ID=number`, and enriches with `Title/Labels/State/URL` via `runtime.forgeClient.GetIssue` when available. If the forge client is unconfigured, fall back to the bare-bones ref (kind+id+repo) — matches will work for `repoPattern`/`forge` rules but label filters won't.
- **Run input shape**: serialized JSON matching the shared `work_item_ref` schema (snake_case fields per `internal/contract/schemas/shared/work_item_ref.json`). The persona sees the same shape it would in the contract input.
- **Service caching**: `worksource.Service` is constructed once in `NewServer` and stored on `serverRuntime`, mirroring the pattern used for `forgeClient`.
- **Disambiguation source**: `pipeline` is read from the form body first, then query string — the `/work/...` endpoint is browser-friendly.

## 5. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Forge client unconfigured → ref missing labels → label-filter bindings won't match | Document behavior in handler doc-comment; tests cover both with-forge and without-forge paths. |
| Multiple bindings + caller picks one not in match set | Explicit 400 with allowed pipeline list in the error body. |
| Path `/work/...` collides with future page route | Reserve only the `/dispatch` suffix; future GET pages can use the same prefix safely. |
| Serialization drift from shared schema | Handler builds JSON using the documented field names; a test asserts the run input parses back to a matching `WorkItemRef`. |

## 6. Testing Strategy

- **Unit / handler tests** in `handlers_work_dispatch_test.go`:
  - 0 bindings → 409.
  - 1 binding → 302 redirect to `/runs/{id}`, run row exists with expected pipeline + JSON input.
  - 2 bindings, no `pipeline` param → 400.
  - 2 bindings, `pipeline=A` (in match set) → 302 with pipeline A.
  - 2 bindings, `pipeline=Bogus` (not in match set) → 400.
  - Path param parsing: missing parts → 404 (handled by mux), malformed `number` → 400.
- **Round-trip assertion**: after dispatch, `rwStore.GetRun(runID)` returns a row whose `Input` is JSON that round-trips through the shared `work_item_ref` schema validator (using `internal/contract/schemas/shared/work_item_ref_test.go` style helpers).
- Reuse `testServer(t)` pattern from `handlers_test.go`. Insert bindings directly into `rwStore` (already exposes `WorksourceStore` via interface composition).
- No forge client needed for tests — pass labels/state via query params or use a fake forgeClient.

## 7. Out of Scope

- The work-board UI (#2.3) — the handler is callable but the button lives in another issue.
- Detach mode — the issue mentions "(or detached)" parenthetically; default is in-process via `launchPipelineExecution`. A `detach=true` form field can be honored if trivial; otherwise deferred.
- `on-open`/`scheduled`-trigger automatic dispatch — this issue covers only manual `/dispatch`.
