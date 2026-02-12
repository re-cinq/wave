# Remove or hide documentation for unreleased features (ISO security, etc.)

**Issue**: [#52](https://github.com/re-cinq/wave/issues/52)
**Labels**: documentation
**Author**: nextlevelshit
**State**: OPEN

## Summary

Several documentation sections describe features that are not yet implemented or will not ship in the current release (e.g., ISO security compliance). The docs need to be aligned with the actual shipped feature set so users are not misled.

## Problem

The current documentation includes references to features that are not realistic for the initial release, including:
- ISO security compliance documentation
- Other features documented but not yet implemented

This creates a gap between what is documented and what is actually available, which can confuse users and set incorrect expectations.

## Action Plan

- [ ] Audit implementation code, CLI help text, and documentation for inconsistencies between documented and shipped features
- [ ] Identify specific documents and sections that reference unshipped features
- [ ] Create a list of all documented-but-unshipped features with their doc locations
- [ ] Remove or hide documentation sections that do not reflect the current release
- [ ] Relocate removed content to a `docs/future` directory or a dedicated branch for future reference
- [ ] Verify remaining documentation accurately describes the shipped feature set

## Acceptance Criteria

- [ ] All public documentation accurately reflects the shipped v0.x feature set
- [ ] No documentation references features that are not implemented
- [ ] CLI `--help` text matches actual available commands and options
- [ ] Removed documentation is preserved in a separate location for future use
- [ ] A summary of what was removed/hidden is documented in this issue or a linked PR

## Audit Findings

Based on analysis of the docs directory and codebase, the following categories of unreleased/aspirational content have been identified:

### 1. Trust Center - Compliance (`docs/trust-center/compliance.md`)
- **SOC 2 Type II** - Status "In Progress", expected Q3 2026. Not implemented.
- **HIPAA** - Status "Planned", expected Q1 2027. Not implemented.
- **ISO 27001** - Status "Planned", expected Q4 2026. Not implemented.
- **GDPR** - Claimed "Compliant" but Wave is a CLI tool, not a data processor. Overstates compliance posture.
- **FedRAMP**, **PCI DSS**, **State Privacy Laws** - References to compliance frameworks without implementations.
- **Compliance Documentation** table references documents like "Penetration Test Summary" and "SOC 2 Type II Report" that don't exist.
- **Contact emails** (security@re-cinq.com, enterprise@re-cinq.com, legal@re-cinq.com, federal@re-cinq.com) may not exist.

### 2. Trust Center - Security Model (`docs/trust-center/security-model.md`)
- Some content reflects the actual architecture (workspace isolation, credential handling, permission enforcement) but is presented with enterprise marketing language beyond current state.
- **SIEM Integration** section - references forwarding to SIEM systems (not implemented as a feature).
- **Security Event Schema** - the specific event format shown may not match actual implementation.

### 3. Trust Center - Audit Logging (`docs/trust-center/audit-logging.md`)
- Extensive specification with detailed JSON schemas for events. Some event types may not be fully implemented.
- **SIEM Integration** examples (Splunk, Elasticsearch) - aspirational.
- **Performance Considerations** with specific overhead percentages - likely not benchmarked.

### 4. Enterprise Guide (`docs/guides/enterprise.md`)
- References to enterprise deployment patterns that assume team-wide Wave deployment.
- `skill_mounts` configuration may not be implemented.
- `max_concurrent_workers` and parallel step execution may not be fully implemented.
- `meta_pipeline.max_total_tokens` may not be implemented.

### 5. Use Cases
- Several use cases reference personas that may not ship (e.g., `philosopher`, `summarizer`).
- Pipelines shown as runnable (`wave run security-audit`, `wave run incident-response`) may not be bundled.
- `wave do "..." --save` flag may not exist in the CLI.

### 6. Integrations (`docs/integrations/`)
- GitHub Actions and GitLab CI guides reference `curl -L https://wave.dev/install.sh` - this install script may not exist.
- `wave run --input`, `--timeout`, `--stop-after`, `--resume` flags may not all be implemented.
- `OPENAI_API_KEY` mentioned as required for OpenAI adapter - adapter may not be released.

### 7. CLI Reference (`docs/reference/cli.md`)
- Documents `wave do "..." --save` which may not exist.
- Documents `wave run --from-step --force` which may not be implemented.
- Documents `wave logs --follow`, `--since`, `--level` which may not be implemented.
- Documents output modes `--output json/text/quiet` which may not all be implemented.

### 8. Matrix Strategies (`docs/guides/matrix-strategies.md`)
- Matrix execution with fan-out parallel workers may not be fully implemented.
- Conflict detection between matrix workers may not exist.

### 9. Sandbox Setup (`docs/guides/sandbox-setup.md`)
- Documents Nix + bubblewrap sandbox integration - this may be partially implemented.
- `persona.sandbox.allowed_domains` configuration may not be fully enforced.

### 10. Reference Schemas (`docs/reference/manifest-schema.md`, `docs/reference/pipeline-schema.md`)
- May document fields that aren't yet parsed or enforced by the manifest/pipeline loaders.
