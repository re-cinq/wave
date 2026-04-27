# feat(pipeline): add parallel complexity audit step with quality gates

**Issue**: [re-cinq/wave#1041](https://github.com/re-cinq/wave/issues/1041)
**Author**: nextlevelshit
**State**: OPEN
**Labels**: `enhancement`, `pipeline`
**Branch**: `1041-complexity-audit-step`

---

## Problem Statement

The Wave pipeline currently lacks a complexity audit step that can run in parallel and enforce quality gates. There is no automated mechanism to measure code complexity (cyclomatic, Halstead, or linguistic metrics) and gate pipeline progression based on thresholds.

## Background / Original Content

The original issue references [this blog post on code complexity](https://philodev.one/posts/2026-04-code-complexity/) by Sofia Fischer, which surveys complexity metrics including:

- **Computational complexity** (Big O) — resource growth as input scales
- **Cyclomatic complexity** — linearly independent paths through code
- **Halstead complexity** — mental effort based on operator/operand vocabulary
- **Linguistic complexity** — psycholinguistic predictors of reading difficulty (familiarity, working memory load, coherence)
- **Practical usage** — combining complexity with churn and coupling metrics for refactoring prioritization

The post argues that complexity metrics are most valuable when used to drive data-based decision-making and visualize refactoring impact, not as forced targets.

## Expected Behavior

- A new pipeline step computes complexity metrics for changed files in parallel.
- Configurable thresholds define quality gates (e.g., max cyclomatic complexity per function).
- The step runs concurrently with other independent pipeline steps to minimize wall-clock time.
- Results are reported in a structured format consumable by downstream steps.
- When a gate threshold is exceeded, the pipeline fails with a clear diagnostic message identifying the offending functions/files.

## Acceptance Criteria

- [ ] A complexity audit step exists in the pipeline manifest and can be invoked via `wave run`.
- [ ] The step computes at least one complexity metric (e.g., cyclomatic complexity) for Go source files.
- [ ] Complexity thresholds are configurable in the pipeline manifest or a config file.
- [ ] The step runs in parallel with other independent steps (not sequentially blocking).
- [ ] When a function exceeds the configured threshold, the step exits non-zero with a message naming the function and its score.
- [ ] When all functions pass, the step exits zero and outputs a summary of scores.
- [ ] The step handles edge cases gracefully: empty file list, non-Go files, missing config defaults.

## Technical Context

- Pipeline orchestration: `internal/pipeline/` and manifest configuration.
- Parallel step execution: existing concurrency model (`internal/pipeline/concurrency.go`, `iterate.parallel` composition primitive).
- Tooling decision: in-tree `go/ast` analyzer (no external binary dependency, no fzipp/gocyclo embedding for v1). Rationale: ~200 LOC, no supply-chain surface, identical accuracy for cyclomatic+cognitive on Go AST.
- Gate mechanism: command-step exit codes + `handover.contract.on_failure: fail` for pipeline-level gating. Structured JSON output conforms to `shared-findings.schema.json` for cross-pipeline aggregation.

## Research Notes (from issue comments)

Four research comments on the issue converge on:

- Per-function cyclomatic + cognitive complexity over Go AST
- Parallel via `errgroup` + `SetLimit(NumCPU)`
- Configurable thresholds, default ~15
- Structured JSON output (Wave-typed primary, optional SARIF v2)
- Exit-code gating
- Defer Halstead, churn/hotspot, multi-language to v2

## Acceptance Decisions (resolves missing-info from assessment)

1. **Implementation**: in-tree Go AST analyzer (no third-party complexity library) for v1.
2. **Output format**: `shared-findings.schema.json` JSON only. SARIF deferred to v2.
3. **Default thresholds**: cyclomatic ≤ 15 (fail), cognitive ≤ 15 (fail). Warn at 10.

## Out of Scope (v2)

- Halstead complexity
- Linguistic complexity
- Multi-language support (Python, TS, Rust, etc.)
- Churn/hotspot analysis
- SARIF 2.1.0 output

## Original Issue URL

<https://github.com/re-cinq/wave/issues/1041>
