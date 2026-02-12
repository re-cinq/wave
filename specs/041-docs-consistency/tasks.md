# Tasks

## Phase 1: DOC-004 — Update Pipeline Count in README

- [X] Task 1.1: Update "18 built-in pipelines" to "19 built-in pipelines" on README.md line 288
- [X] Task 1.2: Update "A selection of the 18 built-in pipelines" to "A selection of the 19 built-in pipelines" on README.md line 300

## Phase 2: DOC-006 — Document GitHub Adapter

- [X] Task 2.1: Add GitHub adapter section to `docs/reference/adapters.md` after the OpenCode Adapter section, documenting: purpose (GitHub API operations — issue listing/analysis/updates, PR creation, repo queries, branch creation), how it differs from LLM adapters (direct API calls, no subprocess CLI invocation), required env vars (`GITHUB_TOKEN` or `GH_TOKEN`), supported operations (list_issues, analyze_issues, get_issue, update_issue, create_pr, get_repo, create_branch), and usage context (used by github-related pipelines like `gh-poor-issues`, `github-issue-enhancer`, `github-issue-impl`)

## Phase 3: DOC-007 — Document GITHUB_TOKEN/GH_TOKEN

- [X] Task 3.1: Add `GITHUB_TOKEN`/`GH_TOKEN` row to the Required Environment Variables table in `docs/reference/environment.md`, documenting it as required for the GitHub adapter and GitHub-related pipelines

## Phase 4: Validation

- [X] Task 4.1: Verify pipeline count matches actual files in `internal/defaults/pipelines/`
- [X] Task 4.2: Run `go test ./...` to confirm no test breakage from documentation changes
