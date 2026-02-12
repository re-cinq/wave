# Compliance Roadmap

This document outlines Wave's current compliance status and certification roadmap. We are committed to meeting the security and privacy requirements of enterprise organizations across industries.

## Compliance Status Overview

<TrustSignals />

## Certification Details

### SOC 2 Type II {#soc-2}

**Status:** In Progress | **Expected:** Q3 2026

SOC 2 Type II certification demonstrates that Wave maintains effective controls over security, availability, processing integrity, confidentiality, and privacy.

#### Scope

The SOC 2 audit covers:

- **Security** - Protection against unauthorized access
- **Availability** - System availability for operation and use
- **Confidentiality** - Protection of confidential information

#### Current Progress

| Milestone | Status | Target Date |
|-----------|--------|-------------|
| Control documentation | Complete | Q4 2025 |
| Gap assessment | Complete | Q1 2026 |
| Remediation | In Progress | Q2 2026 |
| Type II audit period | Planned | Q2-Q3 2026 |
| Report issuance | Planned | Q3 2026 |

#### Wave Security Controls

Wave's architecture inherently supports SOC 2 requirements:

| SOC 2 Criteria | Wave Implementation |
|----------------|---------------------|
| CC6.1 - Logical access | Persona-scoped permissions with deny-first model |
| CC6.6 - System boundaries | Ephemeral workspace isolation |
| CC6.7 - Information transmission | Credential scrubbing in all logs |
| CC7.2 - Security monitoring | Comprehensive audit logging |
| CC7.3 - Security incidents | Security event detection and logging |

---

### GDPR {#gdpr}

**Status:** Compliant

Wave is designed to support GDPR compliance for organizations processing EU personal data.

#### Data Processing Principles

| GDPR Principle | Wave Implementation |
|----------------|---------------------|
| Data minimization | No credential storage; ephemeral workspaces |
| Storage limitation | Configurable workspace cleanup; no persistent user data |
| Integrity and confidentiality | Workspace isolation; credential scrubbing |
| Accountability | Comprehensive audit logging |

#### Technical Measures

1. **No Personal Data Storage** - Wave does not store personal data. All processing occurs in ephemeral workspaces.

2. **Audit Trail** - Complete logging of all operations for accountability and compliance verification.

3. **Data Location** - Wave runs on-premises or in your cloud environment. You control data residency.

4. **Right to Erasure** - Workspace cleanup commands support complete data removal:
   ```bash
   wave clean --all
   wave clean --older-than 30d
   ```

#### Data Processing Agreement

For enterprise customers requiring a DPA, contact: **legal@re-cinq.com**

---

### HIPAA {#hipaa}

**Status:** Planned | **Expected:** Q1 2027

Wave is planning HIPAA compliance for healthcare organizations handling Protected Health Information (PHI).

#### Planned Controls

| HIPAA Requirement | Planned Implementation |
|-------------------|------------------------|
| Access controls | Enhanced persona permission auditing |
| Audit controls | Extended retention for PHI audit logs |
| Integrity controls | Contract validation for PHI handling |
| Transmission security | Existing credential protection extends to PHI |

#### Architecture Alignment

Wave's existing security model aligns with HIPAA requirements:

- **Minimum necessary** - Persona permissions enforce least privilege
- **Audit trail** - All access logged with credential scrubbing
- **Encryption** - Credentials and sensitive data protected in transit
- **Access management** - Deny-first permission model

#### Timeline

| Milestone | Target Date |
|-----------|-------------|
| Gap assessment | Q3 2026 |
| Control implementation | Q4 2026 |
| BAA availability | Q1 2027 |

---

### ISO 27001 {#iso-27001}

**Status:** Planned | **Expected:** Q4 2026

ISO 27001 certification will demonstrate Wave's commitment to information security management best practices.

#### Scope

The ISMS certification will cover:

- Wave software development lifecycle
- Security controls implementation
- Operational security procedures
- Incident management processes

#### Timeline

| Milestone | Target Date |
|-----------|-------------|
| ISMS documentation | Q2 2026 |
| Internal audit | Q3 2026 |
| Certification audit | Q4 2026 |

#### Current Security Practices

Wave already implements key ISO 27001 controls:

| Control Domain | Wave Implementation |
|----------------|---------------------|
| A.9 Access control | Persona permissions, deny-first model |
| A.12 Operations security | Audit logging, workspace isolation |
| A.13 Communications security | Credential protection, no disk storage |
| A.14 System acquisition | Contract validation, input sanitization |

---

## Additional Compliance Considerations

### FedRAMP

Wave's architecture supports FedRAMP requirements. For federal government deployment inquiries, contact: **federal@re-cinq.com**

### PCI DSS

Wave does not process, store, or transmit cardholder data. For environments handling payment data, Wave operates outside the cardholder data environment (CDE).

### State Privacy Laws

Wave's data minimization approach (no credential storage, ephemeral workspaces) supports compliance with state privacy regulations including:

- California Consumer Privacy Act (CCPA)
- Virginia Consumer Data Protection Act (VCDPA)
- Colorado Privacy Act (CPA)

---

## Compliance Documentation

### Available Documents

| Document | Availability |
|----------|--------------|
| Security Architecture Overview | Available upon request |
| Penetration Test Summary | Available under NDA |
| SOC 2 Type II Report | Available Q3 2026 |
| Data Processing Agreement | Available for enterprise |

### Request Documentation

To request compliance documentation for enterprise evaluation:

**Email:** security@re-cinq.com
**Subject:** `[Compliance Request] <Your Organization>`

Please include:

- Organization name
- Specific documents requested
- Intended use case
- Timeline requirements

---

## Security Practices

### Development Security

| Practice | Description |
|----------|-------------|
| Code review | All changes require peer review |
| Static analysis | Automated security scanning |
| Dependency management | Regular dependency updates and vulnerability scanning |
| Security testing | Regular penetration testing |

### Operational Security

| Practice | Description |
|----------|-------------|
| Access management | Role-based access to development infrastructure |
| Monitoring | Security event monitoring and alerting |
| Incident response | Documented incident response procedures |
| Business continuity | Backup and recovery procedures |

---

## Compliance Contact

For compliance questions or to schedule a security review:

**Email:** security@re-cinq.com
**Response time:** 2 business days

For enterprise sales with compliance requirements:

**Email:** enterprise@re-cinq.com

---

## Further Reading

- [Security Model](/trust-center/security-model) - Detailed security architecture
- [Audit Logging](/trust-center/audit-logging) - Audit log specification
- [Enterprise Patterns](/guides/enterprise) - Enterprise deployment guide

---

*Last updated: February 2026*
