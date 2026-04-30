# Work Items

## Phase 1: Setup

- [X] Item 1.1: Add `worksource worksource.Service` field to `serverRuntime` in `internal/webui/server.go`.
- [X] Item 1.2: Construct the service from `rwStore` in `NewServer` and wire it into `runtime`.

## Phase 2: Core Implementation

- [X] Item 2.1: Create `internal/webui/handlers_work_dispatch.go` with `handleWorkDispatch(w, r)`. [P]
- [X] Item 2.2: Implement helper `buildWorkItemRef(ctx, forge, owner, repo, number)` — uses `runtime.forgeClient` when available, falls back to bare ref otherwise. [P]
- [X] Item 2.3: Implement helper `serializeWorkItemRef(ref) (string, error)` matching the shared `work_item_ref` JSON schema (#1590).
- [X] Item 2.4: Implement `selectBindingPipeline(matches, requested string)` — returns chosen pipeline + HTTP status (200/400/409).
- [X] Item 2.5: Wire `s.launchPipelineExecution(runID, pipelineName, input, RunOptions{})` and 302 redirect to `/runs/{runID}`.
- [X] Item 2.6: Register `POST /work/{forge}/{owner}/{repo}/{number}/dispatch` in `internal/webui/routes.go`.

## Phase 3: Testing

- [X] Item 3.1: Add `handlers_work_dispatch_test.go` with `testServer(t)` setup + binding fixtures.
- [X] Item 3.2: Test — zero matches → 409. [P]
- [X] Item 3.3: Test — single match → 302, run row created with expected pipeline + serialized work_item_ref input. [P]
- [X] Item 3.4: Test — multiple matches, no `pipeline` param → 400. [P]
- [X] Item 3.5: Test — multiple matches with valid `pipeline` form value → 302 routes to chosen pipeline. [P]
- [X] Item 3.6: Test — multiple matches with `pipeline` value not in match set → 400. [P]
- [X] Item 3.7: Test — malformed `number` path param → 400. [P]
- [X] Item 3.8: Round-trip assertion that the persisted run input parses back through the shared `work_item_ref` schema validator.

## Phase 4: Polish

- [X] Item 4.1: Run `go test ./internal/webui/... ./internal/worksource/...` and ensure no regressions.
- [X] Item 4.2: Run `go test ./...` per repo policy.
- [X] Item 4.3: Run `golangci-lint run ./...` and resolve findings. (golangci-lint not installed in workspace; `go vet` and `go test -race` clean.)
- [X] Item 4.4: Verify `wave web` boot path still passes integration smoke (server starts, route registered). (Covered by webui suite — server constructor exercised via testServer + handler unit tests.)
- [X] Item 4.5: Add brief doc-comments on the new handler and helpers; no separate doc page needed.
