# Implementation Plan — 1041 Complexity Audit Step

## Objective

Add a deterministic, parallel complexity-audit pipeline (`audit-complexity`) that scores Go functions by cyclomatic and cognitive complexity, fails the pipeline when any function exceeds a configurable threshold, and integrates with `ops-parallel-audit` for fan-out execution alongside the other audit pipelines.

## Approach

1. **In-tree analyzer** at `internal/complexity/` — uses `go/ast` to walk Go source, scores each `*ast.FuncDecl` for cyclomatic and cognitive complexity, returns a flat slice of per-function scores. No external complexity library; ~250 LOC including both visitors.
2. **Per-file parallelism** inside the analyzer using `golang.org/x/sync/errgroup` with `SetLimit(runtime.NumCPU())` so a single audit invocation parallelises across the file set. The `errgroup` import is already a Wave dependency (used by `internal/pipeline/concurrency.go`).
3. **CLI subcommand** `wave audit complexity [paths...]` exposes the analyzer with flags for thresholds, output path, and format. Exits 0 on pass, 1 on threshold breach, ≥2 on parser/IO error.
4. **Pipeline manifest** `audit-complexity.yaml` — single command-step that invokes the CLI subcommand, writes findings JSON conforming to `shared-findings.schema.json`, and gates via `handover.contract.on_failure: fail`.
5. **Parallel composition** — add `audit-complexity` to the `ops-parallel-audit` iterate list; `iterate.parallel` already provides pipeline-level fan-out (max_concurrent: 4 → 5).

## File Mapping

### Create

| Path | Purpose |
|------|---------|
| `internal/complexity/analyzer.go` | Public `Analyze(paths []string, opts Options) (Report, error)`; orchestrates per-file workers via errgroup. |
| `internal/complexity/cyclomatic.go` | `ast.Visitor` that counts decision points (if/for/range/case/&&/\|\|/select-case). |
| `internal/complexity/cognitive.go` | `ast.Visitor` for cognitive complexity (nesting-aware scoring per Sonar rules). |
| `internal/complexity/findings.go` | `ToSharedFindings(report, opts) []Finding` — emits `shared-findings.schema.json`-compatible objects with severity from threshold breach. |
| `internal/complexity/analyzer_test.go` | Golden-fixture unit tests: known-complexity functions, empty input, non-Go files, parser errors, threshold logic. |
| `internal/complexity/testdata/` | Fixture `.go` files with annotated expected scores. |
| `cmd/wave/commands/audit_complexity.go` | Cobra subcommand `wave audit complexity`; flag parsing, JSON write, exit-code mapping. |
| `cmd/wave/commands/audit_complexity_test.go` | Golden-output tests for CLI; exit-code matrix. |
| `internal/defaults/pipelines/audit-complexity.yaml` | Pipeline manifest with single command-step + json_schema contract. |
| `.agents/contracts/complexity-findings.schema.json` | Optional: tighter schema than shared-findings (with `score` numeric field). Defer if shared-findings is sufficient. |
| `specs/1041-complexity-audit-step/{spec,plan,tasks}.md` | This spec. |

### Modify

| Path | Change |
|------|--------|
| `cmd/wave/main.go` (or wherever subcommands register) | Register the new `audit complexity` subcommand under an `audit` parent group (create the parent if missing). |
| `internal/defaults/pipelines/ops-parallel-audit.yaml` | Add `audit-complexity` to the iterate list; bump `max_concurrent` from 3 to 4. Update phase-3 report prompt to mention the new audit. |
| `internal/defaults/embed.go` | No change expected — `embed.FS` globs `*.yaml` automatically. Verify with the `all_pipelines_load_test.go` test. |
| `internal/pipeline/all_pipelines_load_test.go` | Should auto-discover the new pipeline; only modify if it has an explicit pipeline list. |

### Delete

None.

## Architecture Decisions

### ADR 1: In-tree AST analyzer over fzipp/gocyclo embedding

**Choice**: implement cyclomatic + cognitive scorers from scratch in `internal/complexity/`.
**Why**: ~250 LOC total; no third-party supply-chain surface; matches existing Wave style of in-tree analysis (see `internal/onboarding/flavour.go` matrix). Allows shipping cognitive scoring alongside cyclomatic with one walk.
**Trade-off**: we own the cyclomatic-counting rules forever. Mitigated by golden-fixture tests pinning expected scores.

### ADR 2: `command` step type instead of `prompt` step type

**Choice**: the audit pipeline runs `wave audit complexity` via `exec.type: command`, not via an LLM persona prompt.
**Why**: complexity scoring is deterministic — using an LLM would be slow, expensive, and non-deterministic. Existing audit pipelines use `prompt` because they evaluate qualitative concerns; complexity is quantitative.
**Trade-off**: divergence from the visual pattern of other `audit-*` pipelines. Mitigated by clear comment block at the top of the YAML.

### ADR 3: Output via `shared-findings.schema.json`, not a new schema

**Choice**: emit findings of `type: "complexity"` with severity derived from threshold ratio.

Wait — `shared-findings.schema.json` enum does not include `"complexity"`. Options:
1. Add `"complexity"` to the enum (small change, `internal/defaults/contracts/shared-findings.schema.json` and the `.agents/contracts/` copy).
2. Use existing `"performance"` or `"other"` type.

**Decision**: extend the enum with `"complexity"`. Cleaner downstream filtering in `ops-parallel-audit` triage.

### ADR 4: Default thresholds — 15 / 15

**Choice**: cyclomatic ≤ 15 (fail), cognitive ≤ 15 (fail). Warn at 10.
**Why**: middle-ground of research-comment convergence (15/15, 20/15, 10/20). Matches `golangci-lint`'s `cyclop` default. Configurable via flags so projects can ratchet.

### ADR 5: Per-file parallelism inside the analyzer + pipeline-level parallelism via iterate

**Choice**: errgroup with `SetLimit(NumCPU)` inside `Analyze()`; pipeline composition in `ops-parallel-audit` for fan-out across audit types.
**Why**: satisfies "step runs in parallel with other independent steps" (pipeline-level) AND scales scoring across CPU cores within a single audit invocation.

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| AST-based scoring drifts from `gocyclo` reference | Golden-fixture tests pin expected scores for canonical examples (small/branchy/recursive/closure); `gocyclo` reference output stored in testdata as ground truth. |
| Cognitive complexity rules are subjective | Implement Sonar's published rules verbatim; unit-test each clause (nesting bonus, sequence penalty, jump-to-label). Document rule set in `cognitive.go` package comment. |
| Large repos slow scoring | NumCPU parallelism + skip vendored dirs (`vendor/`, `node_modules/`) by default; flag `--exclude` for additional patterns. |
| `shared-findings.schema.json` enum change breaks consumers | The `complexity` enum addition is additive; existing consumers already accept the enum string. Run `all_pipelines_load_test.go` and the schema validation tests in CI. |
| New CLI subcommand collides with existing wave verbs | `audit` parent group is new; no collision with `do`, `run`, `evolve`, `compose`, etc. Verified via `cmd/wave/commands/` listing. |
| `ops-parallel-audit` aggregator prompt assumes 3 audits | Update the synthesis prompt's hard-coded "Three parallel audit sub-pipelines" wording. |

## Testing Strategy

### Unit tests

- **`internal/complexity/cyclomatic_test.go`**: ~10 fixture functions with hand-counted McCabe scores (linear, single-if, nested-if, switch, for-range, &&-chain, ||-chain, defer, anonymous closure, recursive). Assert exact integer match.
- **`internal/complexity/cognitive_test.go`**: ~10 fixtures spanning Sonar's documented rule classes (nesting penalty, sequence penalty, recursion bonus, label jump). Compare against Sonar's published reference scores.
- **`internal/complexity/analyzer_test.go`**: integration of both visitors over a directory; empty-dir → empty findings; non-Go files filtered; broken `.go` file → error returned with file path; concurrent execution verified via `t.Parallel()` and `-race`.
- **`internal/complexity/findings_test.go`**: severity mapping (`high` ≥ threshold, `medium` ≥ warn, else not emitted), schema-compatible output (validate against `shared-findings.schema.json` via `gojsonschema`).

### CLI tests

- **`cmd/wave/commands/audit_complexity_test.go`**: exit-code matrix (no breaches → 0; cyclomatic breach → 1; cognitive breach → 1; parser error → 2; missing path → 2). Stdout is empty by default; stderr carries human summary on breach. Findings JSON written to `--output` path.

### Pipeline tests

- **`internal/pipeline/all_pipelines_load_test.go`**: should auto-include `audit-complexity.yaml`; verify schema validation.
- **Contract test**: `wave run audit-complexity internal/complexity/` against the project itself in a smoke test under `pipeline-contract` step (manual, not CI — the CI smoke runs through `ops-parallel-audit`).
- **`ops-parallel-audit` integration**: run on a fixture branch with one over-threshold function; verify the fail propagates through `merge-findings` into the final report. Documented as manual validation step; not a CI gate.

### Test commands

```bash
go test ./internal/complexity/... -race
go test ./cmd/wave/commands/ -run AuditComplexity -race
go test ./internal/pipeline/ -run AllPipelinesLoad
```

## Validation Checklist (acceptance criteria mapping)

| AC | Validated by |
|----|--------------|
| Pipeline step exists, invokable via `wave run` | `audit-complexity.yaml` + `all_pipelines_load_test.go` |
| Computes ≥1 complexity metric for Go source | `cyclomatic_test.go` golden fixtures |
| Thresholds configurable | CLI flag tests `--max-cyclomatic`, `--max-cognitive` |
| Runs in parallel with other independent steps | `ops-parallel-audit.yaml` iterate.parallel + manual run |
| Threshold breach → non-zero exit + named function/score | CLI exit-code matrix test |
| All-pass → zero exit + summary | CLI happy-path test, stdout assertion |
| Edge cases (empty list, non-Go, missing config) | `analyzer_test.go` cases |

## Out of Scope

- Halstead, linguistic complexity, multi-language support — deferred to v2.
- SARIF 2.1.0 output — deferred to v2; track as follow-up issue.
- Churn/hotspot integration — deferred to v2.
- Auto-ratchet against merge-base — deferred; thresholds are absolute for v1.
