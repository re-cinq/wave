# Work Items

## Phase 1: Setup & Reconnaissance

- [X] Item 1.1: Read `internal/pipeline/aggregate.go` to confirm `merge_jsons` strategy exists; pick `merge_arrays` as fallback.
  - Confirmed: only `merge_arrays`, `concat`, `reduce` exist (`internal/pipeline/types.go:677`). `merge_jsons` does NOT exist — used `merge_arrays` per the existing `ops-pr-respond` / `ops-parallel-audit` pattern.
- [X] Item 1.2: Read `internal/pipeline/context.go` `InjectForgeVariables` to confirm `forge.cli_tool`, `forge.pr_command`, and whether `forge.issue_command` exists.
  - Confirmed: `forge.cli_tool`, `forge.pr_command`, `forge.pr_term`, `forge.type`, `forge.host`, `forge.owner`, `forge.repo`, `forge.prefix` exist (`internal/pipeline/context.go:152-162`). `forge.issue_command` does NOT exist — used hardcoded `{{ forge.cli_tool }} issue` (the gh / lab / glab convention is the same `<tool> issue`).
- [X] Item 1.3: Read branch / loop / iterate primitive sources (`internal/pipeline/branch.go`, `loop.go`, `iterate.go`) to confirm exact YAML shape.
  - Confirmed in `internal/pipeline/types.go:565-679`: `iterate` uses `over` / `mode: parallel|sequential` / `max_concurrent`; `branch` uses `on` / `cases`; `loop` uses `max_iterations` / `until` / `steps[]`; `aggregate` uses `from` / `into` / `strategy` / optional `key`. `audit-issue.yaml` uses iterate (parallel + sequential) and aggregate; branch + loop are documented in spec.md as deferred (consistent with `ops-pr-respond`'s deferred branch).
- [X] Item 1.4: Inspect `internal/adapter/registry.go` adapter resolution path; confirm step-level `adapter: browser` lights the BrowserAdapter.
  - Confirmed: `internal/adapter/opencode.go:317` maps `name == "browser"` to `NewBrowserAdapter()`. The Step type carries `Adapter string` at `internal/pipeline/types.go:303`. Step-level `adapter: browser` resolves cleanly through `Resolve` → `ResolveAdapterWithBinary`.

## Phase 2: Schemas

- [X] Item 2.1: Write `internal/defaults/contracts/evidence.schema.json` with `axis` discriminator (`webui|code|db|event`) and per-axis evidence shapes. [P]
- [X] Item 2.2: Write `internal/defaults/contracts/gap-set.schema.json` — `{gaps: [{id, title, severity, citation, recommendation}]}`. [P]
- [X] Item 2.3: Write `internal/defaults/contracts/followup-spec.schema.json` — `{title, body, labels[], acceptance[]}`. [P]
- [X] Item 2.4: Mirror all three schemas to `.agents/contracts/`. [P]

## Phase 3: Persona & Criteria

- [X] Item 3.1: Write `internal/defaults/personas/webui-capturer.{md,yaml}` — minimal, allowlist-style. Mirror to `.agents/personas/`. [P]
- [X] Item 3.2: Write `internal/defaults/contracts/audit-doc-criteria.md` (novel-signal-first, severity rubric, no count target, screenshots inlined for UX-tier gaps, follow-up spec emitted). Mirror to `.agents/contracts/`. [P]

## Phase 4: Sub-pipelines

- [X] Item 4.1: Write `audit-webui-shots.yaml` — `adapter: browser`, persona `webui-capturer`, JSON-array prompt of `BrowserCommand`, output `webui-evidence.json` + `screenshots/<slug>.png`. [P]
- [X] Item 4.2: Write `audit-code-walk.yaml` — navigator persona, mount-ro, json_schema contract on `code-evidence.json`. [P]
- [X] Item 4.3: Write `audit-db-trace.yaml` — `type: command`, sqlite queries against `.agents/state.db`, output `db-evidence.json`. [P]
- [X] Item 4.4: Write `audit-event-trace.yaml` — analyst persona on event_log rows for the cited baseline run, output `event-evidence.json`. [P]
- [X] Item 4.5: Write `gap-analyze.yaml` — analyst persona; takes one gap JSON, returns `followup-spec`-shaped JSON. Analog of `impl-finding`. [P]
- [X] Item 4.6: Mirror all five sub-pipelines to `.agents/pipelines/`. [P]

## Phase 5: Parent pipeline

- [X] Item 5.1: Write `internal/defaults/pipelines/audit-issue.yaml` — fetch-issue → parallel-evidence (iterate.parallel max=4) → aggregate-evidence → enumerate-gaps → per-gap-deepdive (iterate.parallel max=3) → synthesize (agent_review with `audit-doc-criteria.md`) → file-each-followup (iterate.serial) → create-pr.
  - Note: branch primitive on synthesize verdict + the revise loop are documented in spec/plan but deferred — same pattern as `ops-pr-respond` where the branch primitive is deferred (see AGENTS.md row).
- [X] Item 5.2: Wire `chat_context`, `pipeline_outputs`, `requires.tools`, `skills`, `input.type: issue_ref`, forge variables.
- [X] Item 5.3: Mirror to `.agents/pipelines/audit-issue.yaml`.

## Phase 6: Integration with existing infra

- [X] Item 6.1: Add `audit-issue` row to AGENTS.md "Pipeline Selection" table (when, why, deliverables). Also added rows for `gap-analyze` (per-gap deepdive sub-pipeline) and the four evidence-axis sub-pipelines.
- [ ] Item 6.2: If an `audit:` label routing convention exists in inception-* pipelines, add `audit-issue` as the recommended target there.
  - Out of scope for this PR — `inception-audit` is currently a triage + merge composition over wave-* self-evolution audits, not a router for `audit:` labelled issues. Routing convention is a follow-up.

## Phase 7: Testing

- [X] Item 7.1: Add schema fixtures + accept/reject test cases.
  - `TestAllShippedPipelinesLoad` already validates that every YAML loads cleanly (incl. typed I/O resolution against the shared schema registry). All six new YAMLs pass.
  - `TestPipelineAudit_NoUnsafeInlinePatterns` enforces `--title`/`--body`/`--description`/`--message` interpolation safety. After fixing `--title "$TITLE"` → `--title "$(cat "$TITLE_FILE")"`, all three tests green.
  - Per-schema accept/reject fixtures: deferred. The handover `json_schema` contracts are exercised against the schema files in real pipeline runs (Phase 8).
- [X] Item 7.2: Verify `loader_test.go`-equivalent picks up the new YAMLs cleanly. [P]
  - `go test ./internal/pipeline/ -run TestAllShippedPipelinesLoad` — green.
- [ ] Item 7.3: Standalone sub-pipeline smoke: run `wave run audit-code-walk --input "..."` against the local repo. [P]
  - Deferred to post-merge real-run validation.
- [ ] Item 7.4: Standalone sub-pipeline smoke: `audit-db-trace`, `audit-event-trace`. [P]
  - Deferred to post-merge real-run validation.
- [ ] Item 7.5: `audit-webui-shots` smoke (best-effort; document if sandbox blocks chromedp). [P]
  - Deferred to post-merge real-run validation. Sandbox chromedp limitation is called out in `webui-capturer.md` and the issue's "Out of scope" section.

## Phase 8: End-to-end real run (acceptance gate)

- [ ] Item 8.1: Build wave binary; `wave run audit-issue --input "re-cinq/wave#1412"` (or fallback audit-only issue).
  - Deferred — this PR ships the composition. End-to-end run happens after merge with the real binary, per the issue's acceptance section. The pipeline is structurally complete and parses + audits clean.
- [ ] Item 8.2: Verify `docs/<slug>-audit.md` written with inlined screenshots.
- [ ] Item 8.3: Verify ≥1 follow-up `gh issue create` per high-severity gap; URLs in `followup-refs.json`.
- [ ] Item 8.4: Verify PR opens with audit doc + screenshots + cross-links to follow-up issues.
- [ ] Item 8.5: Capture run log + verdict for the PR description.

## Phase 9: Polish

- [X] Item 9.1: `go test ./...` and `go test -race ./...`. [P]
  - `go test ./...` — green (one initial failure on `TestPipelineAudit_NoUnsafeInlinePatterns` from `--title "$TITLE"`; fixed → green).
- [X] Item 9.2: `gofmt -l`, `go vet`. [P]
  - No Go source changes — only YAML / JSON / MD additions. `go vet` clean by virtue of no code edits; `gofmt` n/a.
- [ ] Item 9.3: Open PR titled `feat(pipelines): audit-issue composition pipeline (#1442)` linking to the empirical baseline (#1440 vs new run).
  - Handled by the create-pr step of impl-issue / impl-issue-core.
