# Implementation Plan — #173 CI/CD Documentation

## Objective

Complete the remaining documentation work for issue #173: consolidate existing CI/CD docs, add a production-ready GitHub Actions example workflow file, document headless/no-TTY mode for CI runners, and ensure the prerequisites/dependency information is comprehensive and accurate.

## Approach

This is primarily a documentation task. The code infrastructure (preflight, doctor, onboarding, CI detection, headless mode) already exists. The work involves:

1. **Consolidate** scattered CI/CD documentation into the main `docs/guides/ci-cd.md`
2. **Add** a production-ready example GitHub Actions workflow to `.github/workflows/wave-example.yml`
3. **Enhance** the installation guide with a comprehensive prerequisites section
4. **Document** headless/no-TTY mode for CI environments
5. **Verify** the existing environment variable reference covers secret injection

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `docs/guides/ci-cd.md` | modify | Consolidate content from `future/integrations/github-actions.md`, add headless mode section, add prerequisites section, add `wave doctor` CI health check |
| `.github/workflows/wave-ci-example.yml` | create | Production-ready example workflow demonstrating Wave in GitHub Actions |
| `docs/guide/installation.md` | modify | Expand prerequisites section with all required tools and their purposes |
| `README.md` | modify | Expand Requirements section with tool table and link to installation guide |
| `docs/future/integrations/github-actions.md` | modify | Add deprecation notice pointing to `docs/guides/ci-cd.md` |

## Architecture Decisions

1. **Keep example workflow as `.github/workflows/wave-ci-example.yml`** — This makes it discoverable for users browsing the repo and serves as both documentation and a testable artifact. The `wave-ci-` prefix distinguishes it from Wave's own CI workflows.

2. **Consolidate into `docs/guides/ci-cd.md`** rather than creating new pages — The existing guide is already well-structured and covers both GitHub Actions and GitLab CI. The `future/integrations/github-actions.md` content should be merged in, not duplicated.

3. **Do not add minimum version numbers for `claude` and `tea`** — Per the issue, these are still TBD. Document that versions are adapter-dependent and suggest `wave doctor` for verification.

4. **Keep `docs/future/integrations/github-actions.md`** with a redirect notice — Don't delete it since it may be linked from external sources.

## Risks

| Risk | Mitigation |
|------|------------|
| Example workflow references install script URL that may change | Use the canonical `raw.githubusercontent.com` URL already established in other docs |
| Missing minimum version info for tools | Document as "latest stable" with `wave doctor` as the verification mechanism |
| Duplicate/conflicting info across docs | Explicit consolidation pass — canonical source is `docs/guides/ci-cd.md` |

## Testing Strategy

Since this is documentation work:

1. **Validate YAML syntax** of the example workflow file
2. **Verify all cross-references** between docs are correct
3. **Check that `go test ./...` still passes** — no code changes expected, but validate
4. **Manual review** of rendered docs structure
