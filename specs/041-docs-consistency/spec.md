# docs: documentation consistency report — 10 inconsistencies found

**Issue**: [#41](https://github.com/re-cinq/wave/issues/41)
**Repository**: re-cinq/wave
**Author**: nextlevelshit
**Labels**: documentation, ready-for-impl
**Complexity**: simple

## Summary

A documentation consistency audit found 10 inconsistencies between Wave's code and documentation. 7 have been fixed; 3 remain.

## Remaining Inconsistencies

### DOC-004 (MEDIUM)

README.md references "18 built-in pipelines" but 19 exist in `internal/defaults/pipelines/`.

- **Location**: `README.md:288,300`
- **Fix**: Update "18 built-in pipelines" to "19 built-in pipelines" in README.md (two occurrences).

### DOC-006 (MEDIUM)

The GitHub adapter (`internal/adapter/github.go`) exists in code but is not documented in the adapters reference.

- **Location**: `docs/reference/adapters.md`
- **Fix**: Add a GitHub adapter section documenting the `GitHubAdapter`, its purpose (GitHub API operations: issue management, PR creation, repo queries), required environment variables (`GH_TOKEN`/`GITHUB_TOKEN`), and how it differs from the LLM CLI adapters (direct API calls vs subprocess execution).

### DOC-007 (MEDIUM)

`GITHUB_TOKEN`/`GH_TOKEN` environment variables are used in code but not documented in the environment reference.

- **Location**: `docs/reference/environment.md`
- **Fix**: Add `GITHUB_TOKEN`/`GH_TOKEN` to the required environment variables section, documenting them as required for the GitHub adapter and GitHub-related personas/pipelines. Currently these only appear as an example in the audit redaction patterns table.

## Acceptance Criteria

- [ ] README.md pipeline count updated from 18 to 19 (both occurrences on lines 288 and 300)
- [ ] GitHub adapter documented in `docs/reference/adapters.md` with purpose, env vars, and API operations
- [ ] `GITHUB_TOKEN`/`GH_TOKEN` documented in `docs/reference/environment.md` required variables section
- [ ] No code changes required — purely documentation updates
