# Implementation Plan: README Public Documentation Update

## Objective

Update the README.md and supporting documentation so that external users and contributors can install Wave, run their first pipeline, and contribute — without any internal knowledge of the project's history or private tooling.

## Approach

This is a documentation-only change. The strategy is:

1. **Audit** existing README.md and `docs/guide/installation.md` for internal-only references
2. **Rewrite** the Installation section in README.md to cover all public install methods (install script, GitHub Releases, build from source, Nix)
3. **Revise** the Quick Start section to be self-contained and testable by a new user
4. **Create** a CONTRIBUTING.md with contributor guidance and link it from the README
5. **Create** a LICENSE file (resolve the MIT vs Apache-2.0 inconsistency — the README says MIT, goreleaser says Apache-2.0; implementation should follow the project owner's intent)
6. **Clean up** `docs/guide/installation.md` to remove the "Private Repository" warning block
7. **Review** all changed files for leaked internal context

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `README.md` | modify | Rewrite Installation and Quick Start sections; add Contributing section link; fix license badge if needed |
| `CONTRIBUTING.md` | create | New file with contributor guidance (build, test, PR workflow, commit conventions) |
| `LICENSE` | create | Add the actual license file (resolve MIT vs Apache-2.0 inconsistency) |
| `docs/guide/installation.md` | modify | Remove "Private Repository" warning; update to reflect public status |

## Architecture Decisions

1. **License resolution**: The README badge says MIT but `.goreleaser.yaml` says Apache-2.0. The implementer must determine which is correct (likely by checking with the maintainer or defaulting to what the README says — MIT). The LICENSE file and all references should be made consistent.

2. **Install methods to document**: The README should present install methods in order of ease:
   - Install script (one-liner curl pipe)
   - GitHub Releases (manual download)
   - Build from source (`make build`)
   - Nix dev shell (optional, for sandboxed development)

3. **CONTRIBUTING.md scope**: Should cover prerequisites, building, testing, commit conventions (conventional commits), and PR process. Should not duplicate the full README — link back to it for installation.

4. **Quick Start rewrite**: The current Quick Start uses `./install.sh` (a root-level convenience script). The rewrite should use the install script curl one-liner or build-from-source as the primary method, then walk through `wave init` → `wave run hello-world`.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| License inconsistency causes confusion | High | Medium | Resolve in this PR by picking one license and making all references consistent |
| Install script URL may be wrong after public release | Low | Low | Use the canonical raw.githubusercontent.com URL that the script already references |
| Existing doc links break | Low | Low | Only modifying content within existing files, not renaming or moving them |
| `go install` may not work if module path differs from GitHub URL | Medium | Medium | Check `go.mod` module path (`github.com/recinq/wave`) — note it uses `recinq` not `re-cinq`, which means `go install github.com/recinq/wave/cmd/wave@latest` is the correct invocation but users must know this differs from the GitHub org URL |

## Testing Strategy

Since this is a documentation-only change, testing is manual verification:

1. **README install script command**: Verify the curl one-liner URL is correct and the script exists at that path
2. **Build from source**: Run `git clone` + `make build` on a clean checkout and confirm the binary builds
3. **Quick Start flow**: Follow the README instructions from a fresh clone through `wave run hello-world` and confirm it works
4. **Link validation**: Check that all markdown links in README.md point to existing files
5. **License consistency**: Grep the repo for license references and confirm they all agree
6. **No internal references**: Search for private registry URLs, internal team names, or credentials in all modified files
