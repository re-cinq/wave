# bug: wave init --all missing gitea-analyst and gitea-enhancer personas required by gt-rewrite pipeline

**Issue**: [#179](https://github.com/re-cinq/wave/issues/179)
**Feature Branch**: `179-missing-gitea-personas`
**Labels**: bug, personas
**Complexity**: simple
**Status**: Planning

## Summary

`wave init --all` does not scaffold the `gitea-analyst` and `gitea-enhancer` personas in the manifest, which are required by the bundled `gt-rewrite` pipeline. Running the pipeline after init fails immediately at the first step because the persona cannot be resolved from the manifest.

## Root Cause

The `createDefaultManifest()` function in `cmd/wave/commands/init.go` defines only 18 persona entries in the manifest, but there are 27 embedded persona `.md` files (26 personas + `base-protocol.md`). The persona `.md` files ARE embedded and written to `.wave/personas/` during init, but without corresponding manifest entries the pipeline executor cannot resolve them.

## Missing Personas (10 total)

The following personas have embedded `.md` files but no entries in `createDefaultManifest()`:

1. `gitea-analyst` — used by gt-rewrite, gt-refresh, gt-research
2. `gitea-enhancer` — used by gt-rewrite, gt-refresh
3. `gitea-commenter` — used by gt-implement, gt-research
4. `gitlab-analyst` — used by gl-rewrite, gl-refresh, gl-research
5. `gitlab-enhancer` — used by gl-rewrite, gl-refresh
6. `gitlab-commenter` — used by gl-implement, gl-research
7. `bitbucket-analyst` — used by bb-rewrite, bb-refresh, bb-research
8. `bitbucket-enhancer` — used by bb-rewrite, bb-refresh
9. `bitbucket-commenter` — used by bb-implement, bb-research
10. `supervisor` — used by supervise pipeline

## Affected Pipelines

- `gt-rewrite`, `gt-refresh`, `gt-research`, `gt-implement`
- `gl-rewrite`, `gl-refresh`, `gl-research`, `gl-implement`
- `bb-rewrite`, `bb-refresh`, `bb-research`, `bb-implement`
- `supervise`

## Steps to Reproduce

1. Create a new project directory
2. Run `wave init --all`
3. Run `wave run gt-rewrite --input <repo>`
4. Error: `pipeline execution failed: step "scan-issues" failed: persona "gitea-analyst" not found in manifest`

## Expected Behavior

`wave init --all` should create all personas referenced by bundled pipelines, so that all bundled pipelines can run without additional configuration.

## Acceptance Criteria

1. **AC-1**: `createDefaultManifest()` includes entries for all 10 missing personas with correct adapter, description, system_prompt_file, temperature, and permissions
2. **AC-2**: Each persona's permissions follow the pattern of existing forge personas (e.g., `github-analyst` pattern for `gitea-analyst`, `gitlab-analyst`, `bitbucket-analyst`)
3. **AC-3**: Each persona's CLI tool reference matches the forge-specific CLI (tea for Gitea, glab for GitLab, bb for Bitbucket)
4. **AC-4**: `wave validate` passes after `wave init --all`
5. **AC-5**: Existing tests continue to pass
6. **AC-6**: All bundled pipelines have their persona dependencies satisfied by the manifest
