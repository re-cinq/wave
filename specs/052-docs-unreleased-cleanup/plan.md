# Implementation Plan: Remove or hide documentation for unreleased features

## Objective

Audit the Wave documentation site (`docs/`) against the actual codebase (`internal/`, `cmd/`) and remove or relocate documentation that references features not yet implemented. Preserve removed content in `docs/future/` for future use.

## Approach

The strategy is a three-pass approach:

1. **Audit** - Cross-reference every docs page against the Go implementation to identify documented-but-unshipped features.
2. **Relocate** - Move aspirational/unreleased documentation to `docs/future/`, preserving directory structure.
3. **Clean up** - Update the VitePress sidebar config, cross-references, and index pages to remove broken links.

### Relocation vs. Deletion

Per the issue requirements, removed content must be preserved. We will use `docs/future/` as a holding area. This is preferable to a separate branch because:
- Content remains discoverable in the same repo
- No branch maintenance burden
- The `.gitignore` in `docs/` already excludes build artifacts, not source files
- `docs/future/` can be excluded from VitePress builds via config

## File Mapping

### Files to move entirely to `docs/future/`

| Source | Destination | Reason |
|--------|-------------|--------|
| `docs/trust-center/compliance.md` | `docs/future/trust-center/compliance.md` | SOC 2, HIPAA, ISO 27001, GDPR, FedRAMP, PCI DSS — none are implemented |
| `docs/use-cases/incident-response.md` | `docs/future/use-cases/incident-response.md` | References unshipped personas and pipelines |
| `docs/use-cases/security-audit.md` | `docs/future/use-cases/security-audit.md` | References unshipped `security-audit` pipeline |
| `docs/use-cases/multi-agent-review.md` | `docs/future/use-cases/multi-agent-review.md` | References unshipped parallel review pipeline |
| `docs/use-cases/api-design.md` | `docs/future/use-cases/api-design.md` | References unshipped `philosopher` persona pipeline |
| `docs/use-cases/migration.md` | `docs/future/use-cases/migration.md` | References unshipped migration pipeline |
| `docs/use-cases/docs-generation.md` | `docs/future/use-cases/docs-generation.md` | Duplicate/aspirational |
| `docs/guides/matrix-strategies.md` | `docs/future/guides/matrix-strategies.md` | Matrix execution not fully implemented |
| `docs/integrations/github-actions.md` | `docs/future/integrations/github-actions.md` | References non-existent install script and unshipped features |
| `docs/integrations/gitlab-ci.md` | `docs/future/integrations/gitlab-ci.md` | References non-existent install script and unshipped features |

### Files to modify (trim unreleased sections)

| File | Sections to Remove/Modify |
|------|--------------------------|
| `docs/trust-center/index.md` | Remove Compliance link and description; update to reflect actual scope |
| `docs/trust-center/security-model.md` | Remove SIEM Integration section; tone down enterprise marketing language |
| `docs/trust-center/audit-logging.md` | Remove SIEM integration examples (Splunk, Elasticsearch); remove Performance Considerations with unverified numbers |
| `docs/guides/enterprise.md` | Remove references to `skill_mounts`, `max_concurrent_workers`, `meta_pipeline` if not implemented; simplify to what exists |
| `docs/index.md` | Update Trust Center badges; remove comparison table if competitors don't exist |
| `docs/reference/cli.md` | Audit each CLI flag/option against actual cobra command definitions; remove undocumented flags |
| `docs/quickstart.md` | Remove OpenCode adapter section if not shipped; verify all commands work |
| `docs/use-cases/index.md` | Remove gallery entries for relocated use cases |
| `docs/integrations/index.md` | Remove or simplify entries for relocated integration guides |
| `docs/.vitepress/config.ts` | Remove sidebar/nav entries for relocated pages |

### Files to leave as-is (verified implemented)

| File | Reason |
|------|--------|
| `docs/concepts/*.md` | Describes architecture concepts that are implemented |
| `docs/guides/sandbox-setup.md` | Nix sandbox is implemented in `flake.nix` |
| `docs/guides/state-resumption.md` | State/resumption implemented in `internal/state/` |
| `docs/guides/relay-compaction.md` | Relay implemented in `internal/relay/` |
| `docs/guides/audit-logging.md` | Audit logging implemented in `internal/audit/` |
| `docs/guides/ci-cd.md` | General CI/CD guide (references actual features) |
| `docs/guides/github-integration.md` | GitHub integration implemented in `internal/github/` |
| `docs/reference/environment.md` | Documents actual environment variables |
| `docs/reference/adapters.md` | Documents implemented adapter system |
| `docs/reference/events.md` | Documents implemented event system |
| `docs/migrations.md` | Documents implemented migration system |
| `docs/use-cases/code-review.md` | Core use case with implemented pipeline |
| `docs/use-cases/onboarding.md` | Basic use case |
| `docs/use-cases/refactoring.md` | Basic use case |
| `docs/use-cases/test-generation.md` | Basic use case |

## Architecture Decisions

1. **`docs/future/` over branch**: Keeping unreleased docs in the same branch is simpler and avoids merge conflicts when features are eventually implemented.
2. **Preserve directory structure**: `docs/future/trust-center/compliance.md` mirrors `docs/trust-center/compliance.md` for easy restoration.
3. **No VitePress config for `docs/future/`**: The future directory is not built or served — it's just a holding area.
4. **CLI audit via code**: Cross-reference `docs/reference/cli.md` against `cmd/wave/commands/*.go` cobra command definitions to identify undocumented flags.

## Risks

| Risk | Mitigation |
|------|------------|
| Removing too much content | Err on the side of keeping content that describes implemented features, even if the presentation is aspirational |
| Breaking internal doc links | Run link audit after changes; update cross-references |
| Incorrectly identifying features as unshipped | Cross-reference against actual Go code in `internal/` and `cmd/` |
| Missing some aspirational content | Systematic file-by-file audit with checklist |

## Testing Strategy

1. **Link audit**: After all changes, scan for broken internal links in remaining docs
2. **VitePress build**: Run `npm run docs:build` (or equivalent) to verify docs site builds without errors
3. **CLI verification**: For any CLI flags removed from docs, verify they don't exist in the actual cobra command definitions
4. **Manual review**: Read through remaining docs to ensure no references to relocated content remain
