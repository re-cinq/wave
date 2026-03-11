# Research: Closed-Issue/PR Audit Pipeline (#305)

**Branch**: `305-audit-pipeline` | **Date**: 2026-03-11

## R1: Pipeline Step Decomposition and Persona Selection

**Decision**: 4-step pipeline using `github-analyst`, `navigator`, `navigator`, `craftsman` personas.

**Rationale**: The audit pipeline has two distinct data domains — GitHub API data (issues, PRs) and codebase data (file contents, git history). Each requires different tool permissions:

| Step | Persona | Why |
|------|---------|-----|
| `collect-inventory` | `github-analyst` | Needs `gh issue list`, `gh pr list`, `gh pr view` — only this persona has those Bash permissions |
| `audit-items` | `navigator` | Read-only codebase verification via Read, Glob, Grep, `git log` — matches FR-015 read-only requirement |
| `compose-triage` | `navigator` | Aggregation of findings into structured report — same tools as audit |
| `publish` | `craftsman` | Needs `gh issue create` — craftsman has full Bash access |

**Constitution P5 deviation**: P5 requires "The first step of every pipeline MUST be a Navigator persona." The first step uses `github-analyst` instead because navigator cannot run `gh` commands (its Bash permissions are limited to `git log*` and `git status*`). The `github-analyst` is read-only (denies `gh issue edit*`, `gh issue create*`, `gh issue close*`), and the audit step (navigator) performs codebase exploration during verification. This is documented in the Complexity Tracking table.

**Alternatives Rejected**:
- 5-step pipeline with navigator first: Adds a codebase mapping step that produces an artifact the audit step would barely use — every inventory item references different files, so a generic map has limited value
- Using `implementer` for inventory: Has full Bash access but is not read-only, violating FR-015
- Using `auditor` for audit-items: Lacks Glob and `git log*` Bash permissions needed for file discovery and revert detection

## R2: GitHub CLI Data Extraction Patterns

**Decision**: Use `gh issue list` and `gh pr list` with `--json` flag for structured output, persona handles scope parsing.

**Rationale**: The `gh` CLI supports structured JSON output via `--json` with field selection. Key queries:

```bash
# All closed issues (excluding "not planned")
gh issue list --state closed --json number,title,body,labels,closedAt,stateReason \
  --limit 500 --jq '[.[] | select(.stateReason != "NOT_PLANNED")]'

# All merged PRs
gh pr list --state merged --json number,title,body,files,mergeCommit,closedAt \
  --limit 500

# Scoped by time (persona-parsed)
gh issue list --state closed --search "closed:>2026-02-01" --json ...

# Scoped by label
gh issue list --state closed --label enhancement --json ...
```

Per spec C3, scope parsing is done by the persona, not Go code. This matches the `doc-audit` pattern where the navigator interprets "full" vs diff inputs directly.

**Pagination**: `gh` handles pagination automatically with `--limit`. For repos with 500+ items, the persona should use `--limit 1000` or paginate manually.

**Alternatives Rejected**:
- Go code for scope parsing: Would require a new input schema and parser, adding Go implementation complexity for a persona-driven pipeline
- GitHub REST API via `curl`: Less ergonomic than `gh` CLI, requires manual pagination and auth header management

## R3: Static Analysis Verification Methodology

**Decision**: File existence + content pattern matching using Glob/Grep/Read, plus `git log` for revert detection.

**Rationale**: Per spec C4, the audit uses static analysis only — no test execution, no compilation. The verification strategy for each fidelity category:

| Category | Verification Method |
|----------|-------------------|
| **verified** | Referenced files exist, key functions/types found via Grep, logic reads match description |
| **partial** | Some but not all acceptance criteria have matching code evidence |
| **regressed** | `git log --all --oneline -- <file>` shows the file was modified/deleted after the implementing PR |
| **obsolete** | Referenced files deleted at HEAD, or codebase has diverged significantly |
| **unverifiable** | No linked PRs, no commit SHAs, no file references in the issue body |

For revert detection specifically:
```bash
git log --oneline --all -- <file>   # Check if file was modified after implementing PR
git log --grep="Revert" --oneline   # Find revert commits
```

The navigator persona can run `Bash(git log*)` to perform these checks.

**Accuracy target**: SC-002 requires 90% accuracy for "verified" classifications. Static analysis can achieve this because:
- File existence is binary and deterministic
- Function/type name presence via Grep is highly reliable
- False negatives (marking something as partial when it's verified) are preferable to false positives

**Alternatives Rejected**:
- Running test suites: Violates FR-015 read-only constraint; test execution requires `go test` which navigator can't run
- Compilation checks: Requires `go build` which navigator can't run; also not available for non-Go items

## R4: Triage Report Structure

**Decision**: JSON report with `metadata`, `summary`, `findings` array, and `prioritized_actions` array.

**Rationale**: Follows the pattern established by `doc-consistency-report.schema.json` (summary + items array). The structure from spec C5:

```json
{
  "metadata": {
    "scope": "full | last 30 days | label:enhancement",
    "timestamp": "2026-03-11T10:00:00Z",
    "repository": "re-cinq/wave",
    "total_items_audited": 150
  },
  "summary": {
    "verified": 120,
    "partial": 15,
    "regressed": 5,
    "obsolete": 8,
    "unverifiable": 2
  },
  "findings": [
    {
      "item_number": 42,
      "item_type": "issue",
      "item_url": "https://github.com/re-cinq/wave/issues/42",
      "title": "Add verbose flag",
      "category": "partial",
      "evidence": ["cmd/wave/commands/run.go:45 — flag defined", "internal/pipeline/executor.go — no verbose output logic"],
      "remediation": "Add verbose output handling in executor.go when --verbose flag is set"
    }
  ],
  "prioritized_actions": [
    {
      "priority": 1,
      "item_number": 42,
      "action_description": "Complete verbose flag implementation in executor"
    }
  ]
}
```

**Alternatives Rejected**:
- Markdown report only: Doesn't satisfy SC-004 (valid JSON conforming to contract schema)
- Flat CSV: Loses the hierarchical grouping by category needed for FR-008

## R5: Workspace and Artifact Flow

**Decision**: All steps share a single worktree workspace; artifacts flow via `inject_artifacts` bindings.

**Rationale**: Following the doc-audit pattern, all steps use `workspace.type: worktree` with the same branch. Artifacts are written to `.wave/output/` and injected to downstream steps via `inject_artifacts`:

```
collect-inventory → inventory.json → audit-items
audit-items → audit-findings.json → compose-triage
compose-triage → triage-report.json → publish
compose-triage → triage-report.json (also used as report body)
```

Each step handover uses `json_schema` contract validation, consistent with P6.

**Alternatives Rejected**:
- Mount-based workspaces: Less isolated, more complex path resolution issues (per memory note about workspace path resolution)
- Separate worktrees per step: Unnecessary overhead for a read-only pipeline

## R6: Rate Limit and Pagination Handling

**Decision**: Persona handles rate limits via prompt instructions; `gh` CLI handles pagination natively.

**Rationale**: Per FR-012, the pipeline must handle GitHub API rate limits. The `gh` CLI has built-in rate limit handling — it automatically waits when rate-limited. The persona prompt instructs checking `X-RateLimit-Remaining` headers if raw API calls are needed.

For pagination (FR-005), `gh` CLI handles pagination automatically when using `--limit` up to 1000. For repositories with more items, the persona should make multiple calls with pagination.

The persona prompt in `collect-inventory` will include instructions for:
1. Using `--limit 500` for initial fetch
2. If the result count equals the limit, paginating for more
3. Rate limit awareness (gh handles this, but the prompt documents it)

**Alternatives Rejected**:
- Go-level rate limit retry logic: Pipeline is YAML-defined, not Go code; the persona handles this at runtime
