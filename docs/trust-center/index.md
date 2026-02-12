# Trust Center

This resource documents Wave's security architecture and audit capabilities.

## Security Overview

Wave enforces strict isolation boundaries, credential protection, and comprehensive audit logging at every layer.

### Core Security Principles

| Principle | Implementation |
|-----------|----------------|
| **Zero Credential Storage** | Credentials never touch disk. All secrets pass through environment variables only. |
| **Ephemeral Isolation** | Each pipeline step executes in an isolated workspace with fresh memory. |
| **Deny-First Permissions** | Persona permissions use deny patterns that take precedence over allow patterns. |
| **Contract Validation** | All inter-step communication is validated against defined contracts. |
| **Comprehensive Audit** | Every tool call and file operation can be logged with credential scrubbing. |

## Security Resources

- [Security Model](/trust-center/security-model) - Credential handling, workspace isolation, and permission enforcement
- [Audit Logging Guide](/guides/audit-logging) - Configuring audit logging

## Vulnerability Disclosure

If you discover a security issue in Wave, please report it via [GitHub Issues](https://github.com/re-cinq/wave/issues) with the `security` label, or open a private security advisory on the repository.

## Additional Resources

- [Enterprise Patterns](/guides/enterprise) - Guide to deploying Wave in enterprise environments
- [Environment & Credentials](/reference/environment) - Complete environment variable reference
- [Audit Logging Guide](/guides/audit-logging) - Practical guide to configuring audit logging
