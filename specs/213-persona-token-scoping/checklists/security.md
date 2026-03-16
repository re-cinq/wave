# Security Requirements Quality: Persona Token Scoping

**Feature**: #213 — Persona Token Scoping
**Date**: 2026-03-16

## Credential Safety

- [ ] CHK026 - Does the spec explicitly state that token VALUES must never appear in logs, error messages, or audit trails? [Credential Safety]
- [ ] CHK027 - Is it specified how token introspection CLI output is sanitized before being included in error messages or events? [Credential Safety]
- [ ] CHK028 - Does the spec define whether token env var NAMES (not values) can appear in error messages and logs? [Credential Safety]
- [ ] CHK029 - Are requirements defined for what happens if `gh api user --include` output is captured in audit logs — is the response body sanitized? [Credential Safety]

## Threat Model

- [ ] CHK030 - Does the spec address the scenario where a malicious persona prompt attempts to read token env vars directly via `Bash(echo $GH_TOKEN)`? [Threat Model]
- [ ] CHK031 - Is the interaction between token scoping and the existing deny list system fully specified to prevent bypass (FR-009)? [Threat Model]
- [ ] CHK032 - Does the spec address whether scope validation results could leak information about token capabilities to untrusted personas? [Threat Model]
- [ ] CHK033 - Are requirements defined for handling token introspection API responses that return unexpected or malicious data? [Threat Model]

## Defense in Depth

- [ ] CHK034 - Does the spec clarify that token scope validation is a COMPLEMENTARY layer, not a REPLACEMENT for deny lists? [Defense in Depth]
- [ ] CHK035 - Are requirements specified for what happens when scope validation passes but the actual API call still fails (e.g., IP restrictions, rate limits)? [Defense in Depth]
- [ ] CHK036 - Does the spec define whether scope validation failures should be treated as security events in the audit log? [Defense in Depth]
