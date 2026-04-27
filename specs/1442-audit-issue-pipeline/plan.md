# Implementation Plan — `audit-issue` composition pipeline

## 1. Objective

Ship a new composition pipeline (`audit-issue`) that turns a GitHub issue ref into an audit doc + screenshots + auto-filed follow-up issues + PR — replacing the impl-issue mismatch on audit-only work (PR #1440 baseline: 60% acceptance, no screenshots, no follow-up issues).

## 2. Approach

Mirror the proven `ops-pr-respond` composition pattern:

1. One **parent pipeline** (`audit-issue`) orchestrates all phases.
2. Five **single-step sub-pipelines** carry the per-axis work — invoked via `iterate.parallel` (4 evidence axes) and `iterate.parallel` over gaps (`gap-analyze`).
3. **Three new JSON schemas** carry typed I/O between phases (`evidence`, `gap-set`, `followup-spec`).
4. **One new persona** (`webui-capturer`) gates the chromedp adapter to the screenshot step.
5. **One agent_review criteria file** (`audit-doc-criteria.md`) enforces the synthesize step's quality bar (novel-signal-first, severity rubric, no count target).
6. AGENTS.md gets a row in the "Pipeline Selection" table.

Files land in **both** `internal/defaults/` (shipped defaults) and `.agents/` (project workspace overrides) per existing parity convention used by `ops-pr-respond` + audit-* family.

## 3. File Mapping

### Create — Pipeline YAMLs (12 = 6 × 2 trees)

- `internal/defaults/pipelines/audit-issue.yaml`
- `.agents/pipelines/audit-issue.yaml`
- `internal/defaults/pipelines/audit-webui-shots.yaml`
- `.agents/pipelines/audit-webui-shots.yaml`
- `internal/defaults/pipelines/audit-code-walk.yaml`
- `.agents/pipelines/audit-code-walk.yaml`
- `internal/defaults/pipelines/audit-db-trace.yaml`
- `.agents/pipelines/audit-db-trace.yaml`
- `internal/defaults/pipelines/audit-event-trace.yaml`
- `.agents/pipelines/audit-event-trace.yaml`
- `internal/defaults/pipelines/gap-analyze.yaml`
- `.agents/pipelines/gap-analyze.yaml`

### Create — Schemas (6 = 3 × 2 trees)

- `internal/defaults/contracts/evidence.schema.json`
- `.agents/contracts/evidence.schema.json`
- `internal/defaults/contracts/gap-set.schema.json`
- `.agents/contracts/gap-set.schema.json`
- `internal/defaults/contracts/followup-spec.schema.json`
- `.agents/contracts/followup-spec.schema.json`

### Create — Persona (4 files = (md + yaml) × 2 trees)

- `internal/defaults/personas/webui-capturer.yaml`
- `internal/defaults/personas/webui-capturer.md`
- `.agents/personas/webui-capturer.yaml`
- `.agents/personas/webui-capturer.md`

### Create — Criteria (2 = 1 × 2 trees)

- `internal/defaults/contracts/audit-doc-criteria.md`
- `.agents/contracts/audit-doc-criteria.md`

### Modify

- `AGENTS.md` — append `audit-issue` row to "Pipeline Selection" table.

### Out of scope (deferred per issue)

- chromedp sandbox profile changes for `webui-capturer`.
- `audit-pr` generalisation.

## 4. Architecture Decisions

### A. Adapter selection for screenshots

Step-level `adapter: browser` (already wired in `internal/adapter/registry.go`). `audit-webui-shots` step writes a JSON array of `BrowserCommand` to its prompt source — the browser adapter parses that and runs chromedp. Output: PNG files plus a `webui-evidence.json` listing `{route, screenshot_path, viewport, captured_at}`.

### B. Evidence aggregation

Use `aggregate.merge_jsons` (or `merge_arrays`) with strategy that pulls each child's typed primary output into a unified `evidence.json` shape. The schema's `axis` discriminator (`webui|code|db|event`) lets `enumerate-gaps` slice by axis.

### C. Branch on synthesize verdict

The synthesize step writes `verdict.json` with `{verdict: "pass" | "warn", reason}`. A `branch` block on that artifact routes to either `file-each-followup` (pass) or a `revise` loop (warn, `max_iterations: 2`).

### D. Per-gap fan-out

`per-gap-deepdive` uses `iterate.parallel max_concurrent: 3` over `enumerate-gaps.out.gap-set.gaps[]`. Each child runs `gap-analyze` which returns a `followup-spec` shaped JSON. Aggregate the followup-specs back, then file them serially.

### E. Filing follow-ups

`file-each-followup` is `type: command` with `iterate.serial` over the high-severity entries from the followup-spec set. Each iteration calls `{{ forge.cli_tool }} issue create --title <t> --body-file <b> --label <labels>`. Output: `followup-refs.json` carrying the URL of every filed issue. Serial (not parallel) to keep gh-CLI rate within bounds.

### F. Persona scoping

`webui-capturer` is a thin allowlist-only persona — its prompt path runs through the browser adapter, so its `permissions.allowed_tools` only need to permit the implicit chromedp commands. No Bash/Write/Read needed at the model layer (the adapter is the executor). Pattern mirrors how `command` steps don't need a model persona at all, but the issue calls for a persona record so we ship a minimal one.

### G. Workspace modes

- Evidence axes: `mount: ./` `mode: readonly` — read-only access to the project tree, plus `.agents/state.db` for DB trace.
- `synthesize`: same.
- `create-pr`: `worktree` (writes the audit doc + commits the screenshots + opens PR).

### H. Forge interpolation

Use `{{ forge.cli_tool }}` and `{{ forge.pr_command }}` for issue + PR commands so the pipeline is forge-agnostic from day one.

### I. Skip speckit boilerplate

Do NOT emit `specs/<branch>/{spec,plan,tasks}.md` from the pipeline runtime. The audit deliverable is the markdown doc + screenshots + follow-up issues + PR — speckit boilerplate is `impl-issue`'s shape, not ours.

## 5. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| chromedp sandbox profile blocks browser adapter under bwrap | Issue acceptance flags this as out-of-scope; pipeline will still ship and run outside sandbox. Document in pipeline README the known sandbox limitation. |
| `aggregate.merge_jsons` may not exist as a strategy name | Confirm against `internal/pipeline/aggregate.go`. Fall back to `merge_arrays` (used by `ops-pr-respond`) if no merge_jsons. |
| Forge variable for `gh issue create` — does `forge.issue_command` exist? | Inspect `internal/pipeline/context.go` `InjectForgeVariables`. If not, hardcode `{{ forge.cli_tool }} issue` (gh-only path) and file follow-up to add forge.issue_command. |
| `branch` primitive syntax — confirm shape | Inspect `internal/pipeline/branch.go` (or the equivalent), and any existing pipeline that uses `branch` (none in `.agents/pipelines/`); rely on type-safe yaml schema and unit tests around the executor. |
| Browser adapter prompt format vs LLM personas — different shape | The `audit-webui-shots` step uses `adapter: browser` + the prompt is JSON. No LLM persona invocation; the persona block is metadata only. Verify this routing path in adapter wire-up. |
| End-to-end real run requires #1412 still open or another suitable audit-only issue | Run against #1412 if open; otherwise pick another `audit:` labelled issue. |
| Defaults parity test (deleted per memory) — but onboarding flavour tests may still flag missing parity | Ship to BOTH trees per existing audit-* convention. |

## 6. Testing Strategy

### Unit / contract level

- **Schema validation tests** — every new schema gets a test fixture under `internal/pipeline/testdata/contracts/` validated by an existing JSON-schema test runner (pattern: `internal/pipeline/contracts_test.go` if present, else add). Verify accept-cases and reject-cases.
- **Pipeline load tests** — `internal/pipeline/loader_test.go` already round-trips every YAML in `internal/defaults/pipelines/`; ensure new pipelines parse cleanly.

### Integration

- **Sub-pipeline executes standalone**: `wave run audit-code-walk --input "..."` produces a valid `code-evidence.json`. Same drill for `audit-db-trace`, `audit-event-trace`, `audit-webui-shots` (skip if sandbox blocks chromedp; document).
- **Composition smoke**: Run `audit-issue` against a tiny dummy issue (or local `--input "re-cinq/wave#1412"`) to validate phase wiring (fetch → parallel-evidence → aggregate → enumerate → per-gap → synthesize → file-each-followup → create-pr). Don't actually file new GH issues — use a dry-run flag or stop after `synthesize`.

### End-to-end (real run, gated by acceptance)

- `wave run audit-issue --input "re-cinq/wave#1412"` (or another audit-only issue if #1412 closed). Acceptance: doc lands, ≥1 follow-up issue filed per high-severity gap, PR opens. URLs captured in `followup-refs.json` and `pr-result.json`.

### Lint / style gates

- `golangci-lint run` (no Go changes — but run anyway).
- `go test ./...` and `go test -race ./...` (yaml round-trip tests live under Go).
- `go vet`, `gofmt -l` clean.
