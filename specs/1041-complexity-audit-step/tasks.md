# Work Items — 1041 Complexity Audit Step

## Phase 1: Setup

- [X] Item 1.1: Confirm feature branch `1041-complexity-audit-step` exists and is checked out.
- [X] Item 1.2: Add `"complexity"` to the `type` enum in `internal/defaults/contracts/shared-findings.schema.json` and the mirrored `.agents/contracts/shared-findings.schema.json`. Run schema-validation tests to confirm no consumer regresses.
- [X] Item 1.3: Scaffold `internal/complexity/` package directory with empty `analyzer.go`, `cyclomatic.go`, `cognitive.go`, `findings.go`, `testdata/`. Add package doc.go describing the rule set.

## Phase 2: Core Implementation

- [X] Item 2.1: Implement cyclomatic visitor in `internal/complexity/cyclomatic.go` (counts `if`, `for`, `range`, `case`, `&&`, `||`, `select` clauses, named-result return paths). [P]
- [X] Item 2.2: Implement cognitive-complexity visitor in `internal/complexity/cognitive.go` (Sonar rules: nesting bonus, sequence penalty, recursion bonus, label-jump penalty). [P]
- [X] Item 2.3: Implement `Analyze(paths []string, opts Options) (Report, error)` in `analyzer.go` — discovers `.go` files (skip `_test.go` opt-out, skip `vendor/`), parses each with `go/parser`, runs both visitors, returns per-function scores. Per-file parallelism via `errgroup.SetLimit(runtime.NumCPU())`.
- [X] Item 2.4: Implement `ToSharedFindings(report Report, opts Options) ([]Finding, error)` in `findings.go` — maps over-threshold functions to `shared-findings.schema.json` entries (severity = `high` for fail-threshold breach, `medium` for warn-threshold breach).
- [X] Item 2.5: Implement `wave audit complexity [paths...]` Cobra subcommand in `cmd/wave/commands/audit_complexity.go`. Flags: `--max-cyclomatic` (default 15), `--max-cognitive` (default 15), `--warn-cyclomatic` (default 10), `--warn-cognitive` (default 10), `--output` (default `.agents/output/findings.json`), `--exclude` (repeatable glob), `--format` (default `json`, accept `summary` for stdout). Exit codes: 0 pass, 1 breach, ≥2 IO/parse error.
- [X] Item 2.6: Wire the `audit` parent command group in `cmd/wave/main.go` (or wherever subcommands register) and attach `complexity` as a child. [P with 2.5 once 2.5 lands]
- [X] Item 2.7: Author `internal/defaults/pipelines/audit-complexity.yaml` — single `command` step running `wave audit complexity {{ input }}`, `output_artifacts.findings`, `handover.contract: json_schema` against `shared-findings.schema.json` with `on_failure: fail`.
- [X] Item 2.8: Update `internal/defaults/pipelines/ops-parallel-audit.yaml` to add `audit-complexity` to the iterate list and bump `max_concurrent: 3 → 4`. Update report prompt's "three parallel audit sub-pipelines" wording.

## Phase 3: Testing

- [X] Item 3.1: Author `internal/complexity/testdata/` golden fixtures: `linear.go`, `branchy.go`, `nested.go`, `switchy.go`, `recursion.go`, `closures.go`, `broken.go` (parse-error case). [P]
- [X] Item 3.2: Write `cyclomatic_test.go` — table-driven test asserting exact integer score per fixture function. [P]
- [X] Item 3.3: Write `cognitive_test.go` — table-driven test against Sonar reference scores. [P]
- [X] Item 3.4: Write `analyzer_test.go` — integration: empty path → empty report; mixed Go/non-Go dir → only Go scored; broken file → typed error; race-safe under `-race`.
- [X] Item 3.5: Write `findings_test.go` — severity mapping; output validates against `shared-findings.schema.json` via `gojsonschema`.
- [X] Item 3.6: Write `cmd/wave/commands/audit_complexity_test.go` — exit-code matrix, JSON output shape, stderr summary on breach.
- [X] Item 3.7: Verify `internal/pipeline/all_pipelines_load_test.go` discovers and validates `audit-complexity.yaml` (no test-code change expected; just run).
- [X] Item 3.8: Run `go test ./... -race` end-to-end before PR.

## Phase 4: Polish

- [X] Item 4.1: Add package-level docstring to `internal/complexity/doc.go` documenting the cyclomatic and cognitive rule sets, citing the Sonar spec for cognitive.
- [X] Item 4.2: Update `docs/reference/cli.md` (or equivalent) with the new `wave audit complexity` subcommand, flags, exit codes, and a one-line example.
- [X] Item 4.3: Update `docs/guides/` audit-pipeline overview (if one exists) to mention `audit-complexity` alongside the LLM-driven audits and explain the deterministic-vs-LLM distinction.
- [X] Item 4.4: Run `wave run audit-complexity internal/complexity/` against the new package itself as a self-check and confirm it passes its own thresholds (dogfood smoke). [P]
- [X] Item 4.5: Run `wave run ops-parallel-audit internal/pipeline/` to verify the new audit fans out alongside the existing ones; capture screenshot/log evidence for the PR description.
- [X] Item 4.6: Final validation pass: re-read `spec.md` acceptance criteria; tick each box in the PR description with the test/file that proves it.
