# Feature Specification: Closed-Issue/PR Audit Pipeline

**Feature Branch**: `305-audit-pipeline`
**Created**: 2026-03-11
**Status**: Draft
**Input**: User description: "https://github.com/re-cinq/wave/issues/305"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Run Full Implementation Audit (Priority: P1)

A project maintainer suspects that LLM-driven development has introduced implementation gaps — partially completed features, acceptance criteria that were never fully satisfied, or merged PRs whose changes later regressed. They run `wave run wave-audit` to get a zero-trust verification of every closed issue and merged PR against the current codebase.

**Why this priority**: This is the core value proposition. Without the end-to-end audit flow (inventory → per-item audit → triage report), no other stories deliver value. A single command that produces a prioritized list of implementation gaps is the minimum viable product.

**Independent Test**: Can be fully tested by running the pipeline against any repository with closed issues and merged PRs, then manually spot-checking 5-10 items from the report against the actual codebase to confirm classification accuracy.

**Acceptance Scenarios**:

1. **Given** a repository with 50 closed issues and 30 merged PRs, **When** the user runs `wave run wave-audit`, **Then** the pipeline completes and produces a triage report with every item categorized as verified, partial, regressed, obsolete, or unverifiable.
2. **Given** a repository with a known partially-implemented feature (some acceptance criteria unmet), **When** the audit runs, **Then** the report correctly identifies the item as "partial" with specific details about which criteria remain unmet.
3. **Given** a repository with a feature that was implemented but later refactored away, **When** the audit runs, **Then** the report categorizes the item as "regressed" with evidence of what changed.

---

### User Story 2 - Scoped Audit by Time or Label (Priority: P2)

A project maintainer wants to audit only recent work (e.g., issues closed in the last 30 days) or a specific label category (e.g., `enhancement` issues only) rather than the full history. They pass a scope parameter to limit the audit to a manageable subset.

**Why this priority**: Full-history audits on large projects may be prohibitively long. Scoping enables incremental adoption, targeted verification after sprints or releases, and routine use in CI/CD pipelines.

**Independent Test**: Can be tested by running the pipeline with a time-range or label filter and verifying the inventory only contains matching items, and that the resulting report excludes non-matching items.

**Acceptance Scenarios**:

1. **Given** a repository with issues spanning 6 months, **When** the user runs `wave run wave-audit -- "last 30 days"`, **Then** only issues and PRs closed within the last 30 days appear in the inventory and report.
2. **Given** a repository with issues labeled `enhancement` and `bug`, **When** the user runs `wave run wave-audit -- "label:enhancement"`, **Then** only enhancement-labeled items are audited.
3. **Given** an empty scope (no matching items), **When** the audit runs, **Then** the pipeline completes gracefully with a report indicating zero items to audit.

---

### User Story 3 - Actionable Remediation Output (Priority: P2)

A developer receives the audit report and wants to act on findings. Each "fixable gap" finding includes specific file paths, code references, and a description of what is missing so the developer can address the gap without re-reading the entire original issue.

**Why this priority**: An audit that identifies gaps without actionable details shifts the burden back to the developer. Specific, evidence-based findings make the audit results directly useful for remediation.

**Independent Test**: Can be tested by examining the report output for any non-verified finding and verifying it contains the original issue reference, unmet criteria, file paths, and a remediation description.

**Acceptance Scenarios**:

1. **Given** an audit finds a partially implemented feature, **When** the developer reads the finding, **Then** the finding includes the original issue's acceptance criteria, which criteria are unmet, relevant file paths, and a description of what needs to change.
2. **Given** an audit finds a regression, **When** the developer reads the finding, **Then** the finding includes the commit or PR that likely caused the regression and the affected code locations.

---

### User Story 4 - Resume After Interruption (Priority: P3)

The audit pipeline is long-running — auditing hundreds of items may take significant time. If it is interrupted (network failure, user cancellation, timeout), the user can resume from the last completed step without re-processing already-audited items.

**Why this priority**: Without resume capability, any interruption wastes all prior work. Wave already has resume infrastructure, so this story leverages existing capability with minimal additional effort.

**Independent Test**: Can be tested by running the pipeline, interrupting it mid-audit, then running `wave run wave-audit --from-step <step>` and verifying it continues from where it left off using persisted artifacts.

**Acceptance Scenarios**:

1. **Given** the pipeline is processing items and the inventory step has completed, **When** the user interrupts execution during the audit step, **Then** the completed inventory artifact is persisted and available for resume.
2. **Given** a previously interrupted run with a completed inventory, **When** the user resumes from the audit step, **Then** the pipeline uses the existing inventory artifact instead of re-fetching.

---

### Edge Cases

- What happens when a closed issue has no linked PR or commits? The audit should mark it as "unverifiable — no linked implementation artifacts" and include it in the report with a note explaining the lack of traceable code changes.
- How does the system handle issues closed as "not planned" or "wontfix"? These should be excluded from the audit inventory by default, since they represent intentional non-implementation.
- What happens when the GitHub API rate limit is exceeded during inventory collection? The pipeline should detect rate-limit responses (HTTP 429 / `X-RateLimit-Remaining: 0`), respect the reset window, and continue fetching after the cooldown.
- How does the system handle closed issues that reference code in deleted files or removed branches? The audit should categorize these as "obsolete" with evidence that the referenced code no longer exists at HEAD.
- What happens when a single issue maps to changes across dozens of files? The audit should produce a coherent finding summarizing the scope, highlighting the most important file paths rather than exhaustively listing every touched file.
- What if the repository has thousands of closed issues? The inventory step should paginate through all items using the GitHub CLI's built-in pagination. The audit step should process items in batches sized to fit within adapter context limits.
- What happens when an issue was closed by a PR that was later reverted? The audit should detect the revert and categorize the item as "regressed" with a reference to the reverting commit.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST provide a `wave-audit` pipeline definition in `.wave/pipelines/` that can be invoked via `wave run wave-audit`.
- **FR-002**: System MUST fetch all closed issues from the target repository via the `gh` CLI, extracting: number, title, body, labels, linked PRs/commits, close reason, and acceptance criteria (if present in the issue body).
- **FR-003**: System MUST fetch all merged pull requests from the target repository via the `gh` CLI, extracting: number, title, body, changed files, linked issues, and merge commit SHA.
- **FR-004**: System MUST produce a structured JSON inventory artifact containing all fetched issues and PRs with their extracted metadata.
- **FR-005**: System MUST handle GitHub API pagination to retrieve the complete set of closed issues and merged PRs, regardless of repository size.
- **FR-006**: System MUST audit each inventory item against the current codebase state at HEAD of the default branch, verifying that the described changes or features still exist and function as specified.
- **FR-007**: System MUST classify each audited item into exactly one fidelity category: **verified** (fully implemented and intact), **partial** (some acceptance criteria unmet or incomplete logic), **regressed** (was implemented but later broken or reverted), **obsolete** (codebase has diverged enough that the item no longer applies), or **unverifiable** (no linked PRs, commits, or traceable code changes to evaluate).
- **FR-008**: System MUST produce a triage report artifact that groups findings by fidelity category and includes a prioritized action list for non-verified items.
- **FR-009**: Each non-verified finding MUST include: the original issue/PR reference (number and URL), the assigned fidelity category, supporting evidence (file paths, code references, commit SHAs), and a remediation suggestion.
- **FR-010**: System MUST support scoped audits via CLI input, allowing filtering by time range (e.g., "last 30 days", "since 2026-01-01") or by label (e.g., "label:enhancement").
- **FR-011**: System MUST exclude issues closed as "not planned" from the audit inventory by default.
- **FR-012**: System MUST handle GitHub API rate limits gracefully — detecting rate-limit responses and pausing until the reset window before continuing.
- **FR-013**: System MUST be resumable using Wave's existing `--from-step` capability, allowing interrupted runs to continue from the last completed step with persisted artifacts.
- **FR-014**: All pipeline step outputs MUST be validated against contract schemas to ensure machine-parseable, structured results.
- **FR-015**: The pipeline MUST use read-only personas for analysis steps — no codebase modifications should occur during the audit. An optional final `publish` step using the `craftsman` persona MAY create GitHub issues for fixable gaps; this step is included in the pipeline definition but is not required for the core audit flow to deliver value.

### Key Entities

- **Inventory Item**: A closed issue or merged PR with its extracted metadata (number, title, body, labels, linked commits/PRs, close reason). Represents a single unit of work to audit.
- **Audit Finding**: The result of auditing one inventory item. Contains the item reference, fidelity category (verified/partial/regressed/obsolete), supporting evidence, and remediation suggestion.
- **Triage Report**: The aggregated output of all audit findings, organized by fidelity category with summary statistics (counts per category) and a prioritized action list.
- **Fidelity Category**: One of five classifications representing the current implementation status of a closed item: verified, partial, regressed, obsolete, or unverifiable.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: The pipeline successfully completes on repositories with up to 500 closed issues and 300 merged PRs without failure, timeout, or data loss.
- **SC-002**: At least 90% of "verified" classifications are confirmed correct when spot-checked against the codebase (low false-positive rate for the "all clear" category).
- **SC-003**: At least 80% of "partial" and "regressed" findings include actionable remediation details — file paths, specific unmet criteria, and a description of what needs to change.
- **SC-004**: The triage report is valid JSON conforming to its contract schema with zero validation errors across all runs.
- **SC-005**: Scoped audits (time range or label filter) reduce inventory size proportionally and complete faster than a full audit.
- **SC-006**: A resumed pipeline run does not re-fetch inventory or re-process steps that completed in the prior run.
- **SC-007**: Each pipeline step completes within the configured timeout (default 90 minutes per step).

## Clarifications

The following ambiguities were identified and resolved during spec refinement:

### C1: Auto-create GitHub issues for fixable gaps (FR-015)

**Question**: Should the pipeline optionally auto-create GitHub issues for non-verified findings that have actionable remediation steps?

**Resolution**: Yes — include an optional `publish` step using the `craftsman` persona that creates GitHub issues for fixable gaps (partial/regressed items with actionable remediation). This follows the established pattern in the `doc-audit` pipeline, which has a `publish` step using `craftsman` to create issues via `gh issue create`. The publish step is the final step in the pipeline DAG and only depends on the triage report being complete. All preceding analysis steps remain read-only.

**Rationale**: The `doc-audit` pipeline (`.wave/pipelines/doc-audit.yaml`) already establishes this exact pattern: read-only analysis steps followed by an optional write step for issue creation. Consistency with existing pipeline conventions is preferred.

### C2: Pipeline step decomposition and batching strategy

**Question**: How should the pipeline be decomposed into steps, and how should large inventories (hundreds of items) be processed within adapter context limits?

**Resolution**: The pipeline uses 4 steps: (1) `collect-inventory` — fetches all closed issues and merged PRs via `gh` CLI into a structured JSON inventory artifact; (2) `audit-items` — processes the inventory as a single step where the persona reads the inventory, then samples and verifies items against the codebase using Grep/Glob/Read (the persona handles batching internally by processing items sequentially within a single adapter session); (3) `compose-triage` — aggregates audit findings into a prioritized triage report; (4) `publish` (optional) — creates GitHub issues for actionable findings.

**Rationale**: Wave's `iterate` composition primitive exists but adds complexity. For the initial implementation, a single audit step that processes items within one adapter session is simpler and leverages the adapter's ability to maintain context across multiple file reads. If the inventory exceeds context limits, the persona should focus on the most impactful items (non-trivial issues with acceptance criteria) rather than exhaustively auditing every item. This matches the edge case guidance: "highlighting the most important file paths rather than exhaustively listing."

### C3: Scope parsing mechanism (FR-010)

**Question**: How are natural-language time ranges ("last 30 days") and label filters ("label:enhancement") parsed and applied to GitHub API queries?

**Resolution**: The inventory step persona is responsible for parsing the CLI input string and constructing appropriate `gh` CLI queries. Time-range expressions (e.g., "last 30 days", "since 2026-01-01") are translated by the persona into `gh issue list --search "closed:>YYYY-MM-DD"` and `gh pr list --search "merged:>YYYY-MM-DD"` queries. Label filters (e.g., "label:enhancement") are translated into `--label enhancement` flags. The persona handles the parsing in-prompt, not Go code, consistent with how `doc-audit` parses its scope input.

**Rationale**: The `doc-audit` pipeline establishes the pattern of persona-level input parsing (see its `scan-changes` step which interprets "full" vs empty vs git-ref inputs). Pushing scope parsing into Go code would require a new input schema and parser, adding implementation complexity for marginal benefit. The persona can handle natural-language time expressions more flexibly than a rigid parser.

### C4: Audit verification methodology (FR-006)

**Question**: How does a read-only persona verify that features "still exist and function as specified" without running tests or compiling code?

**Resolution**: The audit uses **static analysis only**. The persona reads the issue/PR description to identify what should exist in the codebase (specific functions, types, handlers, configuration options, test files), then uses Glob, Grep, and Read to verify presence. Specifically: (a) check that referenced files still exist; (b) grep for key function/type names mentioned in the issue; (c) read relevant code sections to verify the logic matches the described behavior; (d) check that related test files exist and contain assertions matching the acceptance criteria. The persona does NOT run tests, compile code, or execute any validation commands.

**Rationale**: FR-015 mandates read-only personas for analysis steps, and the `navigator` and `auditor` personas are both constrained to never modify source files. Static analysis is sufficient for the fidelity classification — it can detect missing code (partial), deleted code (regressed/obsolete), and present code (verified) without execution. False positives from static-only analysis are acceptable given the 90% accuracy target in SC-002.

### C5: Triage report top-level structure (FR-008, SC-004)

**Question**: What is the expected JSON structure of the triage report artifact?

**Resolution**: The triage report JSON contains: `metadata` (object with `scope`, `timestamp`, `repository`, `total_items_audited`), `summary` (object with counts per fidelity category: `verified`, `partial`, `regressed`, `obsolete`, `unverifiable`), `findings` (array of finding objects, each with `item_number`, `item_type` (issue/pr), `item_url`, `title`, `category`, `evidence` (array of strings with file paths and code references), `remediation` (string, empty for verified items)), and `prioritized_actions` (array of objects with `priority` (1-N), `item_number`, `action_description`, sorted by estimated impact). The exact schema will be defined as a JSON Schema contract file during the planning phase.

**Rationale**: This structure aligns with the existing contract schema patterns in `.wave/contracts/` (e.g., `doc-consistency-report.schema.json` uses a summary + items array pattern). Adding `unverifiable` as a fifth category in the summary accommodates the edge case of issues with no linked implementation artifacts, which was identified but not reflected in the fidelity categories of FR-007.
