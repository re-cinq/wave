# Security Requirements Checklist

**Feature**: Dashboard Inspection, Rendering, Statistics & Run Introspection
**Spec**: specs/091-dashboard-introspection/spec.md
**Generated**: 2026-02-14

This checklist validates that security-related requirements are sufficiently specified before implementation.

---

## XSS Prevention (FR-031)

- [ ] CHK-S01 - Are sanitization requirements specified for each distinct content rendering path: markdown output, syntax-highlighted code, raw text display, artifact previews, and error messages? [Completeness]
- [ ] CHK-S02 - Does the spec define whether the markdown parser should strip or escape HTML tags embedded in markdown source (e.g., `<script>` inside a system prompt file)? [Clarity]
- [ ] CHK-S03 - Is it clear that the syntax highlighter must HTML-escape content BEFORE tokenizing (not after), to prevent injection via crafted YAML/JSON values? [Clarity]
- [ ] CHK-S04 - Are workspace file contents required to be HTML-escaped on the server side, client side, or both? Is the responsibility clearly assigned? [Clarity]
- [ ] CHK-S05 - Does the spec address XSS risk in recovery hint display — could a crafted error message containing HTML be rendered unsafely as a recovery hint? [Coverage]
- [ ] CHK-S06 - Are event log messages (which may contain arbitrary text) specified to be escaped before display in the event timeline? [Coverage]

## Path Traversal (FR-024, C-007)

- [ ] CHK-S07 - Is the path validation mechanism for workspace browsing clearly specified — `filepath.Rel` check, prefix check, or canonical path comparison? [Clarity]
- [ ] CHK-S08 - Does the spec explicitly prohibit symlink following in the workspace browser to prevent symlink-based path traversal? [Completeness]
- [ ] CHK-S09 - Is the behavior specified when a workspace path from `step_state` points to a directory outside the expected workspace root? [Coverage]
- [ ] CHK-S10 - Are URL-encoded path traversal attempts (e.g., `%2e%2e%2f`) addressed in the path validation requirements? [Coverage]

## Authentication (FR-032)

- [ ] CHK-S11 - Does the spec enumerate all new API endpoints that must be protected by bearer token auth, or does it rely on a blanket statement? [Completeness]
- [ ] CHK-S12 - Is it specified whether the HTML page endpoints (not just /api/ JSON endpoints) also require authentication when non-localhost? [Clarity]
- [ ] CHK-S13 - Are the workspace file content endpoints specifically called out as requiring auth, given they serve potentially sensitive file contents? [Coverage]

## Data Exposure

- [ ] CHK-S14 - Does the spec address whether system prompt content displayed via the persona detail API could leak sensitive information (API keys, internal URLs embedded in prompts)? [Coverage]
- [ ] CHK-S15 - Is the existing credential redaction (`RedactCredentials`) required to be applied to system prompt content before serving via the API? [Completeness]
- [ ] CHK-S16 - Are workspace file content responses required to redact sensitive patterns (credentials, tokens) or is raw content acceptable? [Clarity]
