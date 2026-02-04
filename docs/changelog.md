# Changelog

All notable changes to Wave are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Enterprise documentation with Trust Center
- Interactive pipeline visualization components
- YAML playground for configuration validation
- Use case gallery with 10 production-ready patterns
- Platform-specific installation guides (macOS, Linux, Windows)
- CI/CD integration guides (GitHub Actions, GitLab CI)
- Error codes reference with resolution steps

### Changed
- Redesigned landing page with hero section and feature cards
- Enhanced quickstart with troubleshooting callouts
- Improved persona documentation with interactive permission matrix

## [0.1.0] - 2026-02-01

### Added
- Initial release of Wave
- Multi-agent pipeline orchestration via YAML configuration
- Built-in personas: Navigator, Auditor, Implementer, Tester
- Contract validation (JSON schema, TypeScript, test suites)
- Ephemeral workspace isolation for each step
- Fresh memory at step boundaries
- Relay/compaction via dedicated Summarizer persona
- Audit logging with credential scrubbing
- State persistence and pipeline resumption
- CLI commands: `wave init`, `wave run`, `wave validate`, `wave status`

### Security
- Credentials passed via environment variables only
- Deny-first permission model for personas
- Workspace isolation prevents cross-step data leakage
- Input validation for prompt injection prevention
- Path traversal protection on all file operations

---

## Version History

| Version | Date | Highlights |
|---------|------|------------|
| 0.1.0 | 2026-02-01 | Initial release with core pipeline features |

## Upgrade Guide

### From Pre-release to 0.1.0

If you were using a pre-release version:

1. **Backup your manifests**
   ```bash
   cp wave.yaml wave.yaml.backup
   ```

2. **Update Wave**
   ```bash
   wave self-update
   ```

3. **Validate your configuration**
   ```bash
   wave validate
   ```

4. **Review breaking changes**
   - Manifest schema has been finalized
   - Some CLI flags have been renamed for consistency

## Release Notes

For detailed release notes including migration guides, see [GitHub Releases](https://github.com/re-cinq/wave/releases).

## Deprecation Policy

- Features are deprecated with at least one minor version warning
- Deprecated features are removed in the next major version
- Security fixes may require immediate breaking changes

## Reporting Issues

Found a bug or have a feature request?

- [GitHub Issues](https://github.com/re-cinq/wave/issues)
- [Security vulnerabilities](/trust-center/#vulnerability-disclosure)
