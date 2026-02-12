# Tasks

## Phase 1: Setup and Audit

- [X] Task 1.1: Create `docs/future/` directory structure mirroring `docs/` layout
- [X] Task 1.2: Cross-reference CLI docs (`docs/reference/cli.md`) against cobra command definitions in `cmd/wave/commands/*.go` to identify flags/options documented but not implemented [P]
- [X] Task 1.3: Cross-reference manifest/pipeline schema docs against `internal/manifest/` types to identify unimplemented fields [P]

## Phase 2: Relocate Entire Pages

- [X] Task 2.1: Move `docs/trust-center/compliance.md` to `docs/future/trust-center/compliance.md` [P]
- [X] Task 2.2: Move `docs/use-cases/incident-response.md` to `docs/future/use-cases/incident-response.md` [P]
- [X] Task 2.3: Move `docs/use-cases/security-audit.md` to `docs/future/use-cases/security-audit.md` [P]
- [X] Task 2.4: Move `docs/use-cases/multi-agent-review.md` to `docs/future/use-cases/multi-agent-review.md` [P]
- [X] Task 2.5: Move `docs/use-cases/api-design.md` to `docs/future/use-cases/api-design.md` [P]
- [X] Task 2.6: Move `docs/use-cases/migration.md` to `docs/future/use-cases/migration.md` [P]
- [X] Task 2.7: Move `docs/use-cases/docs-generation.md` to `docs/future/use-cases/docs-generation.md` (duplicate of documentation-generation) [P]
- [X] Task 2.8: Move `docs/guides/matrix-strategies.md` to `docs/future/guides/matrix-strategies.md` [P]
- [X] Task 2.9: Move `docs/integrations/github-actions.md` to `docs/future/integrations/github-actions.md` [P]
- [X] Task 2.10: Move `docs/integrations/gitlab-ci.md` to `docs/future/integrations/gitlab-ci.md` [P]

## Phase 3: Trim Sections from Remaining Pages

- [X] Task 3.1: Edit `docs/trust-center/index.md` — remove Compliance roadmap link and description
- [X] Task 3.2: Edit `docs/trust-center/security-model.md` — remove SIEM Integration section, remove compliance roadmap link
- [X] Task 3.3: Edit `docs/trust-center/audit-logging.md` — remove SIEM integration examples (Splunk, Elasticsearch), remove unverified performance percentages, remove compliance link
- [X] Task 3.4: Edit `docs/guides/enterprise.md` — remove unimplemented config references (`skill_mounts`, `max_concurrent_workers`, `meta_pipeline.max_total_tokens`, `output_format`); simplify to actual feature set
- [X] Task 3.5: Edit `docs/reference/cli.md` — audit confirmed CLI docs are accurate; no changes needed
- [X] Task 3.6: Edit `docs/quickstart.md` — verified commands/flags exist; OpenCode adapter is shipped so section kept; updated "security audits" mention
- [X] Task 3.7: Edit `docs/index.md` — remove comparison table with non-existent competitors; update feature description
- [X] Task 3.8: Edit `docs/use-cases/index.md` — remove gallery entries for relocated use cases; remove `--save` from `wave do` (flag doesn't exist)
- [X] Task 3.9: Edit `docs/integrations/index.md` — remove card grid, install script example, and links to relocated integration guides

## Phase 4: Update Navigation and Config

- [X] Task 4.1: Edit `docs/.vitepress/config.ts` — remove sidebar/nav entries for all relocated pages
- [X] Task 4.2: Scan all remaining docs for broken cross-references to relocated pages and fix or remove them (7 broken links fixed across 5 files)

## Phase 5: Validation

- [X] Task 5.1: Run `go test -race ./...` — all tests pass
- [X] Task 5.2: Manual review of remaining docs — cross-reference scan confirmed no remaining links to relocated content
- [X] Task 5.3: Change summary created for commit/PR description
