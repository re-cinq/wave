# Security Requirements Quality — Ecosystem Adapters for Skill Sources

**Feature**: #383 | **Date**: 2026-03-14

## Path Security

- [ ] CHK101 - Does the `file:` adapter spec require rejection of both direct symlinks AND intermediate symlink components in the path? [Completeness]
- [ ] CHK102 - Are path traversal rejection requirements defined for ALL adapters that write to disk (file, github, https), not just the `file:` adapter? [Coverage]
- [ ] CHK103 - Does the spec define whether absolute paths outside the project root are rejected by the `file:` adapter (e.g., `file:/etc/passwd`)? [Clarity]
- [ ] CHK104 - Is the containment boundary for the `file:` adapter explicitly defined as the `projectRoot` constructor parameter? [Clarity]
- [ ] CHK105 - Does the spec address TOCTOU (time-of-check-time-of-use) race conditions between path validation and file read for the `file:` adapter? [Coverage]

## Input Validation

- [ ] CHK106 - Does the spec define validation for the `ref` parameter passed to each adapter (e.g., max length, forbidden characters)? [Completeness]
- [ ] CHK107 - Is command injection prevention addressed for CLI adapters that pass user-supplied `ref` strings to subprocess arguments? [Coverage]
- [ ] CHK108 - Does the spec define validation for archive entry filenames beyond `..` rejection (e.g., absolute paths, null bytes, excessively long names)? [Completeness]
- [ ] CHK109 - Are the `github:` adapter reference parsing rules strict enough to prevent injection into the constructed clone URL? [Coverage]

## Network Security

- [ ] CHK110 - Does the `https://` adapter spec address SSRF (server-side request forgery) risks — e.g., requests to internal/localhost addresses? [Coverage]
- [ ] CHK111 - Is the maximum download size for the `https://` adapter specified to prevent denial-of-service via large archives? [Completeness]
- [ ] CHK112 - Does the spec define whether the `https://` adapter follows HTTP redirects, and if so, how many and to which domains? [Completeness]
- [ ] CHK113 - Is TLS verification behavior defined for the `https://` adapter (no insecure skip)? [Clarity]

## Credential Handling

- [ ] CHK114 - Does the spec confirm that no credentials are persisted to disk by any adapter (consistent with Wave's security model)? [Consistency]
- [ ] CHK115 - Is the `github:` adapter's authentication strategy defined (uses system git credentials, no Wave-managed credentials)? [Clarity]
