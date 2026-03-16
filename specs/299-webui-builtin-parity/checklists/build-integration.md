# Build Integration & Binary Constraints Checklist

**Feature**: #299 — Embed Web UI as Default Built-in with CLI/TUI Feature Parity
**Generated**: 2026-03-16

This checklist validates that build tag removal and binary constraints are fully specified.

---

## Build Tag Removal

- [ ] CHK201 - Is the complete list of files requiring build tag removal enumerated and verified against the actual codebase? (Plan says 22 files — is this current?) [Completeness]
- [ ] CHK202 - Are there any OTHER build tags in the webui package beyond `//go:build webui` that might affect compilation? [Completeness]
- [ ] CHK203 - Does the spec address what happens to any existing `_test.go` files that had `//go:build webui` — will they run in CI without the tag? [Consistency]
- [ ] CHK204 - Is the deletion of `serve_stub.go` specified with consideration for any other code that might reference the stub's types or functions? [Completeness]

## Binary Size

- [ ] CHK205 - Is the 2MB binary size budget (SC-002) justified with a measurement of current embedded asset sizes? [Clarity]
- [ ] CHK206 - Does the spec define how binary size is measured (compressed vs uncompressed, stripped vs unstripped)? [Clarity]
- [ ] CHK207 - Are asset optimization requirements (minification, compression of embedded HTML/CSS/JS) specified or explicitly deferred? [Completeness]

## Cross-Platform

- [ ] CHK208 - Does the spec verify that embedded assets work identically on all target platforms (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64)? [Coverage]
- [ ] CHK209 - Are there any platform-specific webui behaviors (e.g., file path handling in artifact URLs) that need specification? [Completeness]
