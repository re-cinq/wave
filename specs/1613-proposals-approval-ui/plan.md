# Implementation Plan — #1613

## Objective

Provide a human approval gate for evolution proposals, exposing both a webui (`/proposals`) and CLI (`wave proposals ...`) surface that read from `evolution_proposal` rows, render diffs and signal summaries, and on approval activate a new `pipeline_version` atomically.

## Approach

1. Build webui handlers + templates that match the existing `internal/webui` Server/registerRoutes/template pattern. Reuse the per-session CSRF token already wired into `serverAuth.csrfToken` and `csrfMiddleware` (mutating method gate). Follow the `templates/preview/proposal.html` mockup as visual reference.
2. Extend the CLI surface with a `proposals` command following the cobra subcommand layout used by `decisions.go` / `persona.go`. Use `state.NewStateStore(dbPath)` to obtain an `EvolutionStore`.
3. The "approve" path is two writes inside one logical transaction: `DecideProposal(id, ProposalApproved, decidedBy)` then `CreatePipelineVersion(rec)` with `Active=true` (which already deactivates priors atomically inside its own tx). Compute the new version number as `latest_existing + 1` and source `sha256` + `yaml_path` from the proposal's diff/yaml-after on disk.
4. Render diffs by reading the `DiffPath` file from `EvolutionProposalRecord.DiffPath`; emit as plain `<pre>` HTML (escaped) with line-class spans matching the preview mockup classes (`diff-line-add`, `diff-line-del`, `diff-line-ctx`). No new chroma dep.

## File Mapping

### Created
- `internal/webui/handlers_proposals.go` — list/detail/approve/reject handlers; activation logic helper.
- `internal/webui/handlers_proposals_test.go` — table-driven tests against an in-memory `stateStore`; covers list filter, detail (with diff render), approve happy path, reject happy path, CSRF rejection, missing-id 404.
- `internal/webui/templates/proposals/list.html` — table of proposals with status badges + filter (status query param).
- `internal/webui/templates/proposals/detail.html` — header (pipeline, v before→after, status), diff `<pre>`, reason + signal_summary cards, approve/reject form posting via `fetch` with `X-CSRF-Token` (single inline `<script>`, no JS framework).
- `cmd/wave/commands/proposals.go` — cobra `proposals` parent + `list`/`show`/`approve`/`reject` subcommands.
- `cmd/wave/commands/proposals_test.go` — exercises the resolver + decision + activation logic via a temp sqlite DB; verifies that approve flips `active` and that reject leaves versions untouched.
- `specs/1613-proposals-approval-ui/spec.md`, `plan.md`, `tasks.md` (planning artifacts).

### Modified
- `internal/webui/routes.go` — register four `/proposals…` routes.
- `cmd/wave/main.go` — `rootCmd.AddCommand(commands.NewProposalsCmd())`.
- `internal/webui/templates/layout.html` — add nav link to `/proposals` (only if a global nav exists; otherwise skip).

### Not modified
- `internal/state/evolution.go` — `EvolutionStore` already exposes everything required (`DecideProposal`, `CreatePipelineVersion`, `ListProposalsByStatus`, `ListPipelineVersions`).
- DB schema — `pipeline_version` and `evolution_proposal` tables already exist.

## Architecture Decisions

- **CSRF reuse, not new middleware.** `Server.csrfMiddleware` already gates non-GET requests against `s.auth.csrfToken`. The `csrfToken` template func is already in scope. Adding a new mechanism would duplicate state.
- **Activation is two atomic ops, not one.** `DecideProposal` (status flip) and `CreatePipelineVersion(active=true)` each take their own short transaction. The window between them is sub-millisecond; if step 2 fails, the proposal is in `approved` status without an active row — surface that as an error and let the user retry via CLI/webui (idempotent: approve→activate is the only path that activates). Wrapping both in a single tx would leak transaction handling out of `EvolutionStore`. Acceptable trade-off.
- **Version number = latest+1.** `CreatePipelineVersion` does not auto-increment. Caller computes via `ListPipelineVersions(name)[0].Version + 1` (or `1` if empty). Race: two concurrent approvals on the same pipeline could both compute `N+1`; the unique key on `(pipeline_name, version)` will reject the second with a constraint error. Surface as 409 Conflict.
- **Diff render: no chroma.** Reason: existing `internal/webui/diff.go` uses git-derived line-class HTML; chroma would pull a new dep for syntax highlighting that is not in the acceptance criteria. Plain escaped diff with class spans matches the preview mockup and ships zero new deps.
- **CLI `--reason` on reject only.** Issue spec says `reject <id> --reason "..."`. For approve, `decided_by` is taken from `os.Getenv("USER")` (fallback `"cli"`). Reason field on approve is allowed but optional.
- **Source of yaml_path/sha256 on approve.** `EvolutionProposalRecord` carries `DiffPath` (the diff blob). The post-diff yaml file path convention is `<DiffPath>.after.yaml` (set by pipeline-evolve when it writes the proposal). On approve, hash that file with sha256 and store the path in the new `pipeline_version` row. If the after-yaml file is absent, fail loudly — do not silently activate a stale version.

## Risks

- **Concurrent approvals** racing on the same pipeline → unique-constraint error on `pipeline_version`. Mitigation: 409 in webui, exit code 2 + clear message in CLI; no data loss.
- **after-yaml path convention not yet set by pipeline-evolve.** If the upstream pipeline writes the file under a different name, approve will fail. Mitigation: probe both `<DiffPath>.after.yaml` and `<DiffPath>` (.yaml suffix) and pick the one that exists; fall back with a clear error if neither.
- **CSRF token in templates.** The `csrfToken` template func is registered during `parseTemplates`. If the new templates aren't included in the embed glob, the func is missing → template error. Mitigation: confirm `parseTemplates` glob matches `templates/proposals/*.html` (verify in `internal/webui/embed.go`).
- **DB schema drift.** Tests must run against the real schema, not a hand-crafted one. Mitigation: tests use `state.NewStateStore(filepath.Join(t.TempDir(), "state.db"))` so migrations run.

## Testing Strategy

- **Unit (handlers)**: in-memory rwStore, seed three proposals (proposed/approved/rejected), verify `GET /proposals` lists by status filter, `GET /proposals/{id}` renders diff + reason, `POST /proposals/{id}/approve` without `X-CSRF-Token` returns 403, with token returns 200 and flips `pipeline_version.active`. Use `httptest.NewRecorder`.
- **Unit (CLI)**: `runProposalsApprove` against temp sqlite — seed proposal + a v1 active row, run approve, assert `GetActiveVersion(pipeline)` returns the new version, original v1 has `active=false`. Mirror for reject (active stays on v1).
- **Acceptance gate (integration)**: a single test that wires CLI approve → reads back via `EvolutionStore.GetActiveVersion`, then constructs an executor with the new version's yaml_path and confirms the loader picks it up. Lives in `cmd/wave/commands/proposals_test.go` to keep dependencies minimal.
- `go test ./... -race` and `golangci-lint run ./...` must be clean before PR.
