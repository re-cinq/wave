# Adapter Integration Requirements Checklist

**Feature**: #116 — Browser Automation Capability
**Generated**: 2026-03-16
**Focus**: Integration with Wave's existing adapter, executor, and pipeline systems

## AdapterRunner Interface Compliance

- [ ] CHK201 - Is the mapping from `AdapterRunConfig` fields to browser behavior fully specified — which fields are used (Prompt, WorkDir, AllowedDomains, SandboxEnabled) and which are ignored? [Completeness]
- [ ] CHK202 - Is the `AdapterResult.ExitCode` semantics defined — does any single command error make ExitCode=1, or only if all commands fail? [Clarity]
- [ ] CHK203 - Are cancellation semantics defined — when `ctx` is cancelled mid-command-sequence, does the adapter return partial results or nothing? [Completeness]
- [ ] CHK204 - Is the adapter's behavior specified when `cfg.Prompt` is empty (no commands)? [Completeness]

## Executor Integration

- [ ] CHK205 - Is it specified how the executor constructs the `BrowserCommand` JSON array — does the pipeline step author write raw JSON in the `prompt` field, or is there a higher-level abstraction? [Clarity]
- [ ] CHK206 - Are contract validation requirements defined for browser adapter output — can standard JSON schema contracts validate the `[]BrowserResult` array? [Coverage]
- [ ] CHK207 - Is the interaction with artifact injection specified — can browser results be written as artifacts for downstream steps? [Completeness]

## Manifest & Configuration

- [ ] CHK208 - Is the manifest schema for `adapters.browser` fully defined — which fields are required vs optional? [Completeness]
- [ ] CHK209 - Is per-step browser configuration addressed (e.g., one step needs viewport 1920x1080, another 375x667 for mobile testing)? [Coverage]
- [ ] CHK210 - Is the `binary` field in adapter config validated against the preflight-detected binary, or are they independent? [Consistency]
