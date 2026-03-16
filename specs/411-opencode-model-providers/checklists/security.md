# Security Quality Checklist: 411-opencode-model-providers

**Generated**: 2026-03-16
**Spec**: `specs/411-opencode-model-providers/spec.md`

## Environment Isolation

- [ ] CHK-S01 - Does FR-005 specify whether the curated environment blocks inherited env vars that share names with base vars (e.g., a malicious `PATH` override)? [Completeness]
- [ ] CHK-S02 - Is the credential leakage threat model documented — which specific variables are considered sensitive beyond API keys? [Completeness]
- [ ] CHK-S03 - Does the spec define whether `env_passthrough` values are validated or sanitized before being passed to the subprocess? [Completeness]
- [ ] CHK-S04 - Are requirements clear about whether `cfg.Env` can inject arbitrary variables that bypass the curated model? [Clarity]
- [ ] CHK-S05 - Does the spec address whether API key values are logged, traced, or visible in audit output? [Coverage]
- [ ] CHK-S06 - Is there a requirement ensuring the `.opencode/config.json` file does not contain credentials (only provider/model, not API keys)? [Coverage]
- [ ] CHK-S07 - Does the curated environment specification account for platform differences (e.g., `TMPDIR` behavior on macOS vs Linux)? [Completeness]

## Model Configuration Security

- [ ] CHK-S08 - Does the spec address whether the provider string in `provider/model` format is sanitized to prevent injection into config.json? [Completeness]
- [ ] CHK-S09 - Are there requirements about maximum length or character restrictions for model identifier strings? [Completeness]
- [ ] CHK-S10 - Does the spec define behavior when `.opencode/config.json` already exists in the workspace (overwrite vs merge)? [Coverage]
