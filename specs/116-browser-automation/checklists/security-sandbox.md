# Security & Sandbox Requirements Checklist

**Feature**: #116 — Browser Automation Capability
**Generated**: 2026-03-16
**Focus**: Security model completeness for browser automation within Wave's sandbox

## Domain Enforcement

- [ ] CHK101 - Are requirements defined for DNS rebinding attacks — can a page resolve an allowed domain to a non-allowed IP? [Completeness]
- [ ] CHK102 - Is the domain matching specification precise — does `localhost` match `localhost:8080`? Does it match `127.0.0.1`? IPv6 `[::1]`? [Clarity]
- [ ] CHK103 - Are WebSocket connections subject to the same domain allowlist as HTTP requests? [Coverage]
- [ ] CHK104 - Are `data:`, `blob:`, and `javascript:` URI schemes addressed in the domain filtering specification? [Coverage]
- [ ] CHK105 - Is the behavior specified when an allowed page loads a service worker that makes requests to non-allowed domains? [Coverage]

## Process Isolation

- [ ] CHK106 - Is the watchdog goroutine's behavior specified — polling interval, detection mechanism, cleanup actions? [Completeness]
- [ ] CHK107 - Are requirements defined for what happens if the browser spawns child processes (e.g., GPU process, renderer process) — are all killed on cleanup? [Completeness]
- [ ] CHK108 - Is the temp user-data-dir cleanup specified — is it deleted synchronously before step completion, or asynchronously? [Clarity]
- [ ] CHK109 - Is `/dev/shm` usage addressed for containers where shared memory is limited? [Coverage]

## Data Leakage Prevention

- [ ] CHK110 - Are requirements defined for browser cache — is it disabled, or just isolated to the temp dir? [Completeness]
- [ ] CHK111 - Is clipboard access addressed — can the browser read/write system clipboard? [Coverage]
- [ ] CHK112 - Are geolocation, camera, microphone, and notification permission defaults specified (should all be denied)? [Coverage]
- [ ] CHK113 - Is the URL sanitization requirement defined for audit logging — URLs may contain auth tokens, API keys, or PII? [Completeness]
