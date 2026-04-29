# Implementation Plan — Onboarder agent + onboard-project meta-pipeline

## 1. Objective

Ship a forge-agnostic, project-introspecting `onboard-project` meta-pipeline plus a dedicated `onboarder` persona. Running `wave run onboard-project` on any existing project must detect its flavour, propose a tailored `.agents/*` overlay (personas, pipelines, prompts), write those files, and mark the project onboarded via the existing sentinel.

## 2. Approach

Mirror the proven `ops-bootstrap` shape (assess → generate → commit), but flip the polarity from **scaffolding source code** to **scaffolding agent configuration**:

1. **detect** (read-only) — `navigator` persona surveys the project, emits `detection.json` matching `detection.schema.json`.
2. **propose** — `onboarder` persona reads `detection.json` and emits a manifest of `.agents/*` files to write (paths + intended purpose), allowing later steps to be deterministic about what gets generated.
3. **generate** — `onboarder` persona writes the `.agents/*` files into the workspace based on the proposal. Includes at minimum one tailored persona, one tailored pipeline, and one prompt file referenced by that pipeline.
4. **finalize** — `craftsman` persona validates the generated YAML/MD with `wave validate` (or equivalent), then writes the sentinel via Go helper invocation (`go run ./cmd/wave doctor` style) or directly via shell (`mkdir -p .agents && touch .agents/.onboarding-done`) — preserving the same end-state `MarkDoneAt` produces.

Rationale for splitting **detect** from **propose**: forge-agnostic detection is reusable (also used by `wave init` non-interactive baseline); proposal is the LLM-creative step. Keeping them separate lets future steps cache detection output and re-run proposal cheaply.

The pipeline ships in `internal/defaults/pipelines/` so it is embedded into the binary. Local overlays (`.agents/pipelines/onboard-project.yaml`) still take precedence per existing loader behaviour.

## 3. File Mapping

### Create

| Path | Purpose |
|------|---------|
| `internal/defaults/personas/onboarder.md` | Persona spec for the onboarder agent — read-write, scoped to `.agents/*` and project introspection |
| `internal/defaults/contracts/detection.schema.json` | JSON schema validating the detect-step output |
| `internal/defaults/pipelines/onboard-project.yaml` | The 4-step meta-pipeline (detect → propose → generate → finalize) |
| `internal/defaults/prompts/onboard/detect.md` | Detect-step prompt (project introspection → JSON) |
| `internal/defaults/prompts/onboard/propose.md` | Propose-step prompt (detection JSON → file manifest) |
| `internal/defaults/prompts/onboard/generate.md` | Generate-step prompt (file manifest → write files) |
| `internal/defaults/prompts/onboard/finalize.md` | Finalize-step prompt (validate + sentinel) |
| `specs/1576-onboarder-meta-pipeline/{spec,plan,tasks}.md` | This planning artefact set |

### Modify

| Path | Reason |
|------|--------|
| (none required at planning time) | Pipeline + persona + contract auto-discovered by existing embed-FS / registry mechanisms |

If the pipeline registry or contract registry uses an explicit `embed.go` listing, that file will need adding to — to be confirmed during implementation.

### Delete

None.

## 4. Architecture Decisions

- **AD-1 — Four-step shape over three.** Splitting detect from propose makes the schema-validated handoff clean and lets the contract enforce structure on a small, easy-to-validate JSON. Three steps would either bloat one prompt or skip schema validation.
- **AD-2 — Sentinel write via shell, not Go.** The pipeline is a runtime artefact; calling `onboarding.MarkDoneAt` requires Go bindings the persona doesn't have. Shelling `mkdir -p .agents && : > .agents/.onboarding-done` produces the identical filesystem state and is what the sentinel constant documents (path-only marker).
- **AD-3 — `onboarder` persona is read-write but workspace-scoped.** It must be allowed to create `.agents/*` files. Model after `craftsman` (full tool access) but with a tight scope rule in the persona body.
- **AD-4 — Contract `must_pass: true` on detect step.** The schema validates the contract; downstream steps depend on it. Subsequent steps use `on_failure: warn` per existing `ops-bootstrap` precedent.
- **AD-5 — Workspace mount mode mirrors `ops-bootstrap`.** Detect step uses `readonly` mount of `./`; later steps use a `worktree` so generated files land in a branch (or, for live onboarding via `wave init`, the working tree directly — pipeline runner picks).
- **AD-6 — No commit step.** `ops-bootstrap` commits because it scaffolds source. `onboard-project` writes config the user reviews; pushing/committing belongs to the driver (`wave init` CLI driver / webui driver) per the plan doc.

## 5. Risks

| Risk | Mitigation |
|------|-----------|
| LLM generates invalid pipeline YAML | Finalize step runs `wave validate` (or `go run ./cmd/wave validate`) on each generated file; failure forces retry per existing handover contract |
| Onboarder writes outside `.agents/` (touches `wave.yaml`, source files) | Hard rule in persona body; generate-step prompt explicitly forbids non-`.agents/*` paths |
| Detection schema too rigid → fails on unusual projects | Schema accepts `flavour: "unknown"` and an `additional_signals` free-form object for cases we haven't enumerated |
| Sentinel write in different cwd than expected | Pipeline asserts `pwd` equals project root (mount target) before writing sentinel; finalize prompt makes this explicit |
| `embed.go` files-listing not auto-updated | Implementation must check `internal/defaults/embed.go` (or equivalent) and add new paths if the pattern requires it |
| Smoke runs require real LLM calls (cost / time) | Use `--adapter mock` for the schema-validation half of validation; full smoke runs with `--model cheapest` for the LLM half |

## 6. Testing Strategy

### Unit / package tests
- Schema validation tests for `detection.schema.json` — valid + invalid fixtures (mirror existing contract test pattern).
- Defaults registry test — confirm `onboard-project` loads, `onboarder` persona loads, prompt paths resolve.

### Integration / smoke
- `wave run onboard-project` against a throwaway Go repo (e.g. `git init` in `t.TempDir()` with a `go.mod`).
- `wave run onboard-project` against a throwaway Node repo (`package.json` only).
- Both runs must:
  1. Exit 0
  2. Produce ≥1 file in each of `.agents/personas/`, `.agents/pipelines/`, `.agents/prompts/`
  3. Produce `.agents/.onboarding-done`
  4. Detection JSON matches schema

### Manual verification (per memory)
- Build binary, run pipeline against the local `code-crispies` clone or a fresh `mktemp -d` with a Go scaffold. Inspect `.agents/*` content for sanity (not just file existence).

### Mock-adapter validation
- Pipeline structure validates with `--adapter mock` (no real LLM cost) — this catches YAML / contract / persona-resolution bugs before any LLM run.
