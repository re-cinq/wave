# Trust Center

Welcome to the Wave Trust Center. This resource provides enterprise security teams with comprehensive documentation of Wave's security architecture, compliance posture, and audit capabilities.

<TrustSignals />

## Security Overview

Wave is designed with security as a foundational principle. Our multi-agent orchestration platform enforces strict isolation boundaries, credential protection, and comprehensive audit logging at every layer.

### Core Security Principles

| Principle | Implementation |
|-----------|----------------|
| **Zero Credential Storage** | Credentials never touch disk. All secrets pass through environment variables only. |
| **Ephemeral Isolation** | Each pipeline step executes in an isolated workspace with fresh memory. |
| **Deny-First Permissions** | Persona permissions use deny patterns that take precedence over allow patterns. |
| **Contract Validation** | All inter-step communication is validated against defined contracts. |
| **Comprehensive Audit** | Every tool call and file operation can be logged with credential scrubbing. |

## Security Resources

### Architecture Documentation

- [Security Model](/trust-center/security-model) - Detailed documentation of Wave's security architecture, including credential handling, workspace isolation, and permission enforcement.

### Compliance

- [Compliance Roadmap](/trust-center/compliance) - Current compliance status and certification roadmap for SOC 2, HIPAA, GDPR, and other frameworks.

### Audit Capabilities

- [Audit Logging Specification](/trust-center/audit-logging) - Complete specification of Wave's audit logging system, including log formats, retention, and integration guidance.
- [Audit Log Schema (JSON)](/trust-center/downloads/audit-log-schema.json) - Downloadable JSON schema for audit log validation and integration.

## Security Contact

### Vulnerability Disclosure

We take security vulnerabilities seriously. If you discover a security issue in Wave, please report it responsibly.

**Reporting Process:**

1. **Do not** disclose the vulnerability publicly until it has been addressed.
2. Email security findings to: **security@re-cinq.com**
3. Include the following in your report:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact assessment
   - Any suggested remediation (optional)

**Response Timeline:**

| Stage | Target Time |
|-------|-------------|
| Initial acknowledgment | 24 hours |
| Preliminary assessment | 72 hours |
| Status update | 7 days |
| Resolution (critical) | 30 days |
| Resolution (non-critical) | 90 days |

**Recognition:**

We maintain a security acknowledgments page for researchers who responsibly disclose vulnerabilities. Please indicate in your report if you would like to be credited.

### Security Questions

For general security questions or to request additional security documentation for enterprise evaluation:

- Email: **security@re-cinq.com**
- Subject line: `[Security Inquiry] <Your Topic>`

## Additional Resources

- [Enterprise Patterns](/guides/enterprise) - Guide to deploying Wave in enterprise environments
- [Environment & Credentials](/reference/environment) - Complete environment variable reference
- [Audit Logging Guide](/guides/audit-logging) - Practical guide to configuring audit logging

---

*Last updated: February 2026*
