# Feature Specification: Static Analysis for Unused/Redundant Go Code

**Feature Branch**: `103-static-analysis-ci`
**Created**: 2026-02-18
**Status**: Draft
**Input**: [GitHub Issue #72](https://github.com/re-cinq/wave/issues/72) — feat(ci): add static analysis for unused/redundant Go code

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Catch Unused Code in Pull Requests (Priority: P1)

A Wave contributor opens a pull request that introduces unused functions, variables, or parameters. The CI pipeline automatically runs static analysis and reports violations inline on the PR, preventing dead code from entering the codebase without requiring manual reviewer attention.

**Why this priority**: This is the core value proposition — automated detection of code rot at the PR boundary. Without CI integration, linting is ad-hoc and easily skipped. Catching issues before merge is the highest-leverage intervention point.

**Independent Test**: Can be fully tested by opening a PR that introduces an unused variable or function, and verifying that the CI check reports the violation with file path and line number.

**Acceptance Scenarios**:

1. **Given** a contributor opens a PR that introduces an unused function, **When** CI runs, **Then** the static analysis check fails and reports the specific unused function with file path and line number.
2. **Given** a contributor opens a PR with no linting violations in new or modified code, **When** CI runs, **Then** the static analysis check passes and does not block the merge.
3. **Given** a contributor opens a PR that modifies existing code without introducing new violations, **When** CI runs, **Then** only issues in new or modified code are reported (pre-existing violations in untouched code are not surfaced).
4. **Given** multiple linting violations exist in a single PR, **When** CI runs, **Then** all violations are reported in a single check run with individual file and line references.

---

### User Story 2 - Maintain Code Quality on Main Branch (Priority: P1)

A Wave maintainer pushes or merges to the main branch. The CI pipeline automatically runs static analysis to detect regressions, providing visibility into overall codebase health and establishing a baseline for incremental PR checks.

**Why this priority**: Main branch quality monitoring complements PR checks. Together they form a complete safety net. The main branch state also serves as the comparison baseline for incremental mode in PR checks.

**Independent Test**: Can be fully tested by pushing a commit to main and verifying that the lint workflow executes and the check status is visible in the GitHub Actions tab.

**Acceptance Scenarios**:

1. **Given** a commit is pushed to the main branch, **When** CI runs, **Then** the static analysis workflow executes and reports results.
2. **Given** the main branch has pre-existing violations from before the linter was adopted, **When** the lint workflow runs, **Then** only violations introduced by the current push are reported (incremental mode).
3. **Given** a merge to main introduces a new unused parameter, **When** the lint workflow runs, **Then** the violation is reported in the workflow summary.

---

### User Story 3 - Run Linting Locally Before Pushing (Priority: P2)

A Wave developer runs the same linting checks locally before pushing, catching violations in seconds rather than waiting for the CI round-trip. The local experience uses the same configuration as CI so results are consistent.

**Why this priority**: Local linting reduces the feedback loop significantly. It depends on the configuration file existing (P1 stories) but adds substantial developer experience value by enabling pre-push validation.

**Independent Test**: Can be tested by running the lint command locally and verifying it detects the same violations that CI would flag.

**Acceptance Scenarios**:

1. **Given** a developer has golangci-lint v2 installed, **When** they run `make lint` from the repository root, **Then** the same linters and configuration used in CI are applied.
2. **Given** a developer introduces an unused variable in a source file, **When** they run `make lint`, **Then** the violation is reported with the file path and line number.
3. **Given** a developer wants to auto-fix certain violations, **When** they run `golangci-lint run --fix ./...`, **Then** auto-fixable issues are corrected in place.

---

### User Story 4 - Suppress False Positives with Required Justification (Priority: P2)

A Wave contributor encounters a legitimate false positive (e.g., a function parameter required by an interface but unused in a specific implementation). They suppress the violation using a `//nolint` directive that requires both the specific linter name and a justification comment. Unjustified or vague suppressions are themselves flagged as violations.

**Why this priority**: Without a clear, enforced suppression mechanism, developers either fight false positives endlessly or disable linters entirely. Requiring structured justification prevents suppression abuse while respecting legitimate exceptions.

**Independent Test**: Can be tested by adding `//nolint` directives with and without justification, and verifying that unjustified ones are flagged.

**Acceptance Scenarios**:

1. **Given** a function parameter is unused but required by an interface, **When** the developer adds `//nolint:unparam // required by io.Writer interface`, **Then** the violation is suppressed and the suppression itself passes linting.
2. **Given** a developer adds a bare `//nolint` without specifying a linter name, **When** linting runs, **Then** the bare directive is flagged as a violation (specific linter name required).
3. **Given** a developer adds `//nolint:unused` without an explanation comment, **When** linting runs, **Then** the suppression is flagged as a violation (explanation required).

---

### User Story 5 - Gradual Adoption Without Blocking on Legacy Issues (Priority: P3)

A Wave maintainer adopts the linter without needing to fix every pre-existing violation upfront. The CI workflow uses incremental mode so that only new violations are flagged, while legacy issues can be addressed over time. A full-codebase scan remains available for local use.

**Why this priority**: Requiring all existing violations to be fixed before any CI value is delivered creates a high adoption barrier. Incremental mode provides immediate value while deferring legacy cleanup to be addressed organically.

**Independent Test**: Can be tested by enabling incremental mode and verifying that existing violations are not reported on PRs, while newly introduced violations are.

**Acceptance Scenarios**:

1. **Given** the codebase has pre-existing violations, **When** a PR introduces no new violations, **Then** CI passes without reporting legacy issues.
2. **Given** a maintainer wants to see all violations including legacy, **When** they run `golangci-lint run ./...` locally (without incremental mode), **Then** all violations across the entire codebase are reported.

---

### Edge Cases

- What happens when a developer has a different version of golangci-lint installed locally than CI uses? The CI workflow pins a specific version for reproducibility. Local version mismatches may produce slightly different results; the CI version is authoritative.
- What happens when golangci-lint encounters files with `//go:embed` directives (e.g., `internal/defaults/embed.go`, `internal/webui/embed.go`)? These files are hand-written Go code that uses embed directives — they are NOT generated code and SHOULD be linted normally. No path-based exclusions are needed for embed files.
- What happens when the linting step times out on large changesets? SSA-based linters (`unparam`, `unused`) are slower on cold cache. CI caching mitigates cold-cache penalties, and a reasonable timeout is configured.
- What happens when a new golangci-lint release changes linter defaults or behavior? The CI workflow pins a specific version to prevent unexpected breakage from upstream changes.
- What happens when a contributor does not have golangci-lint installed locally? The `make lint` target invokes `golangci-lint run ./...` directly; if the binary is not on `$PATH`, the shell reports "command not found." The CLAUDE.md documentation covers installation guidance. CI is the authoritative check regardless.
- What happens when `revive` rules overlap with `unparam` or `govet`? The configuration omits `revive` to reduce overlap — dedicated linters (`unparam`, `govet`, `staticcheck`) plus `gocritic` provide sufficient coverage without duplicated warnings.
- What happens with files behind `//go:build webui` (the entire `internal/webui/` package)? These files are excluded by the Go toolchain when no `webui` build tag is specified. The default `golangci-lint run ./...` and CI workflow lint only non-tagged code. Linting webui-tagged code is out of scope for this feature and can be added as a separate CI job later if desired.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: Project MUST include a `.golangci.yml` configuration file in the repository root using golangci-lint v2 format (`version: "2"`), since golangci-lint v2.x cannot parse v1 configuration files.
- **FR-002**: The configuration MUST use the `standard` linter preset as a baseline (`linters.default: standard`), which includes: `copyloopvar`, `errcheck`, `govet`, `ineffassign`, `staticcheck`, and `unused`.
- **FR-003**: The configuration MUST additionally enable these linters beyond the standard preset: `unparam` (unused function parameters), `wastedassign` (wasted assignment statements), `gocritic` (broad code quality checks), and `nolintlint` (nolint directive hygiene enforcement).
- **FR-004**: The `nolintlint` linter MUST be configured with `require-explanation: true` and `require-specific: true`, enforcing the format `//nolint:lintername // reason` on every suppression directive.
- **FR-005**: A GitHub Actions workflow file (`.github/workflows/lint.yml`) MUST run golangci-lint on every pull request targeting the `main` branch and on every push to the `main` branch.
- **FR-006**: The CI workflow MUST use the `golangci/golangci-lint-action` (v9 or later) with `only-new-issues: true` to report only violations in new or modified code.
- **FR-007**: The CI workflow MUST pin golangci-lint to a specific version (v2.9 or later) for reproducibility.
- **FR-008**: The CI workflow MUST use `actions/setup-go` with `go-version-file: go.mod` to match the project's Go version (currently Go 1.25.5).
- **FR-009**: The configuration SHOULD use the v2 `exclusions` section for any future generated file paths. The current codebase contains no generated Go files — `//go:embed` files are hand-written and should be linted normally. Files behind `//go:build webui` are excluded from default `go test`/`golangci-lint` runs by the Go toolchain (no explicit exclusion needed).
- **FR-010**: The Makefile `lint` target MUST be updated from `go vet ./...` to `golangci-lint run ./...`.
- **FR-011**: The CLAUDE.md file MUST be updated to document the lint command (`golangci-lint run ./...`) and the auto-fix command (`golangci-lint run --fix ./...`) in the Testing section.
- **FR-012**: The lint workflow MUST be a separate workflow file (`lint.yml`) that does not duplicate or interfere with the existing `release.yml` workflow.
- **FR-013**: The configuration MUST exclude standard error handling patterns and comment-related checks via the v2 exclusion presets (`std-error-handling`, `comments`).
- **FR-014**: The `revive` linter MUST NOT be enabled, to avoid significant overlap with `unparam`, `govet`, and `staticcheck` which already cover unused-parameter and unreachable-code detection.
- **FR-015**: The `gocritic` linter SHOULD use its default stable checks (which already include `dupSubExpr`, `assignOp`, `unnecessaryBlock`), without needing explicit per-check configuration.

### Key Entities

- **Linter Configuration** (`.golangci.yml`): The central configuration file defining which linters are enabled, their individual settings, file exclusion patterns, and the golangci-lint v2 format version. Checked into the repository root.
- **CI Workflow** (`.github/workflows/lint.yml`): A GitHub Actions workflow that orchestrates golangci-lint execution on PRs and main branch pushes. References the linter configuration, pins tool versions, and enables caching.
- **Lint Suppression** (`//nolint` directive): An inline code annotation that suppresses a specific linting violation, requiring both a linter name and a justification comment per FR-004.
- **Standard Preset**: The golangci-lint v2 `standard` preset that serves as the baseline linter set, covering the most broadly applicable code quality checks without per-project configuration.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: A `.golangci.yml` file exists in the repository root and is parseable by golangci-lint v2 without configuration errors.
- **SC-002**: The GitHub Actions lint workflow executes successfully on a PR with no new violations and reports a passing check status.
- **SC-003**: The GitHub Actions lint workflow detects and reports at least one deliberately introduced unused variable in a test PR, proving the analysis is functional.
- **SC-004**: Running `golangci-lint run ./...` locally with the repository configuration produces results consistent with CI (same linters, same rules).
- **SC-005**: All `//nolint` directives in the codebase include a specific linter name and an explanation comment — bare or unjustified `//nolint` directives are flagged by `nolintlint`.
- **SC-006**: The existing `release.yml` CI workflow continues to function identically — the new lint workflow does not interfere with test, build, or release jobs.
- **SC-007**: The Makefile `lint` target runs `golangci-lint run ./...` and exits with a non-zero code when violations are found.
- **SC-008**: CLAUDE.md documents both the lint command and auto-fix command in the Testing section.
- **SC-009**: The CI lint workflow leverages caching (via the golangci-lint-action defaults) to avoid cold-cache performance penalties on repeated runs.

## Clarifications

The following ambiguities were identified and resolved during spec refinement:

### C-001: Embed files are not generated code
**Ambiguity**: FR-009 and Edge Case #2 implied that `//go:embed` files (e.g., `internal/defaults/embed.go`) are generated code requiring exclusion from linting.
**Resolution**: Codebase analysis confirms all embed files are hand-written Go that happen to use `//go:embed` directives. No generated `.go` files exist in the repository (no `//go:generate` directives found). FR-009 was updated to remove the incorrect generated-file exclusion requirement and instead note that the v2 `exclusions` section is available for future use if generated files are added.
**Rationale**: Excluding hand-written code from linting would create blind spots. The `//go:embed` directive is a build instruction, not an indicator of generated code.

### C-002: Build-tagged files (webui) are implicitly excluded
**Ambiguity**: The entire `internal/webui/` package uses `//go:build webui` tags. The spec did not address whether these files would be linted in CI.
**Resolution**: The Go toolchain naturally excludes build-tagged files when the tag is not specified. Since `go test ./...` and `golangci-lint run ./...` both skip webui-tagged files by default, this matches existing project behavior. An edge case was added documenting this. Linting webui code is out of scope for this feature.
**Rationale**: Consistency with existing CI behavior (`release.yml` does not use `--tags=webui` either). Adding webui lint coverage is an independent concern.

### C-003: Makefile lint target needs no installation detection
**Ambiguity**: Edge Case #5 stated that `make lint` should "report that golangci-lint is not installed and provide a brief installation hint," but FR-010 specified a simple command replacement.
**Resolution**: The `make lint` target simply invokes `golangci-lint run ./...`. If the binary is absent, the shell's "command not found" error is sufficient. Installation guidance belongs in CLAUDE.md (covered by FR-011), not in Makefile logic.
**Rationale**: Adding installation detection to the Makefile would be over-engineering. The existing `make lint` target (`go vet ./...`) does not detect whether `go` is installed either — it relies on `$PATH`. The same convention applies.

### C-004: CI timeout relies on action defaults
**Ambiguity**: Edge Case #3 mentions "a reasonable timeout is configured" but no specific value appears in the requirements.
**Resolution**: The `golangci-lint-action` provides sensible default timeouts. The GitHub Actions job-level default (6 hours) serves as the upper bound. No explicit timeout configuration is needed in the workflow file unless experience shows cold-cache runs exceeding the action's default.
**Rationale**: Premature timeout tuning without empirical data risks either being too aggressive (failing valid runs) or too generous (wasting CI minutes). The action defaults are well-tested across thousands of Go projects.

### C-005: Version pinning uses minimum bounds
**Ambiguity**: FR-006 specifies "v9 or later" for the action and FR-007 specifies "v2.9 or later" for golangci-lint. These are minimum bounds rather than pinned versions, which could lead to non-reproducible builds if interpreted as "use latest."
**Resolution**: The requirements intentionally specify minimum bounds to allow the implementer to select the latest stable version at implementation time. The CI workflow MUST pin to a specific version (e.g., `version: "2.9.0"` not `version: ">=2.9"`), but the spec allows flexibility in which specific version is chosen as long as it meets the minimum.
**Rationale**: Pinning to a specific version at spec-writing time would cause the spec to become stale. The minimum bound ensures compatibility while the implementation pins the exact version for reproducibility.
