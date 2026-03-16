# Feature Specification: Third-Party Model Providers via OpenCode Adapter

**Feature Branch**: `411-opencode-model-providers`
**Created**: 2026-03-16
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/376

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Configure a Custom Model for OpenCode Persona (Priority: P1)

A Wave user wants to run a pipeline step using the OpenCode adapter with a non-default model (e.g., GPT-4, Gemini, GLM 5) instead of the hardcoded `claude-sonnet-4-20250514`. They configure the model in their `wave.yaml` persona definition and expect the OpenCode adapter to use it.

**Why this priority**: This is the core ask of the issue. Without dynamic model configuration, the OpenCode adapter is locked to a single Anthropic model, defeating the purpose of using OpenCode (which supports multiple providers).

**Independent Test**: Can be tested by configuring a persona with `adapter: opencode` and `model: <any-model>`, running a pipeline, and verifying the generated `.opencode/config.json` contains the correct provider and model values.

**Acceptance Scenarios**:

1. **Given** a persona with `adapter: opencode` and `model: openai/gpt-4o`, **When** the pipeline step executes, **Then** the generated `.opencode/config.json` contains `"provider": "openai"` and `"model": "gpt-4o"`
2. **Given** a persona with `adapter: opencode` and no model specified, **When** the pipeline step executes, **Then** the adapter uses sensible defaults (current behavior: `anthropic` / `claude-sonnet-4-20250514`)
3. **Given** a persona with `adapter: opencode` and `model: claude-sonnet-4-20250514` (no prefix), **When** the pipeline step executes, **Then** the adapter infers `"provider": "anthropic"` from the model name
4. **Given** the CLI flag `--model openai/gpt-4o` is passed, **When** the pipeline step executes with an opencode persona, **Then** the CLI override takes precedence over the persona manifest model

---

### User Story 2 - Pass Provider API Keys to OpenCode Subprocess (Priority: P1)

A Wave user configures a third-party model provider (e.g., OpenAI) and needs the corresponding API key environment variable (e.g., `OPENAI_API_KEY`) to reach the OpenCode subprocess. They add the variable to `runtime.sandbox.env_passthrough` in their manifest and expect it to be available.

**Why this priority**: Without API key passthrough, third-party providers cannot authenticate, making model configuration useless. This is a hard dependency of Story 1.

**Independent Test**: Can be tested by adding `OPENAI_API_KEY` to `env_passthrough`, running a pipeline with an OpenCode persona, and verifying the subprocess environment contains the key.

**Acceptance Scenarios**:

1. **Given** `runtime.sandbox.env_passthrough` includes `OPENAI_API_KEY` and the variable is set in the host environment, **When** the OpenCode adapter runs, **Then** the subprocess environment contains `OPENAI_API_KEY` with the correct value
2. **Given** `runtime.sandbox.env_passthrough` does NOT include `OPENAI_API_KEY`, **When** the OpenCode adapter runs, **Then** the subprocess environment does NOT contain `OPENAI_API_KEY`
3. **Given** `env_passthrough` includes a variable that is not set in the host environment, **When** the OpenCode adapter runs, **Then** the missing variable is silently skipped (no error)

---

### User Story 3 - Document OpenCode Multi-Provider Setup (Priority: P2)

A Wave user wants to understand how to configure and run third-party models through the OpenCode adapter. They consult the adapter documentation and find clear instructions covering manifest configuration, API key setup, and supported provider/model formats.

**Why this priority**: Documentation enables adoption but is not a functional blocker.

**Independent Test**: Can be tested by following the documentation from scratch to configure and run a pipeline with a non-Anthropic model through the OpenCode adapter.

**Acceptance Scenarios**:

1. **Given** the adapter reference documentation, **When** a user reads it, **Then** they find a complete example of configuring a persona with a third-party model provider
2. **Given** the adapter concepts documentation, **When** a user reads it, **Then** they understand the model identifier format (`provider/model-name`) and which providers are supported

---

### User Story 4 - Curated Environment for OpenCode Adapter (Priority: P3)

The OpenCode adapter currently inherits the full host environment via `os.Environ()`, which risks leaking credentials not intended for the subprocess. The adapter should adopt the same curated environment model as the Claude adapter, passing only base variables plus explicitly allowed passthrough variables.

**Why this priority**: This is a security hardening measure. The adapter works without it, but it closes a credential leakage vector.

**Independent Test**: Can be tested by running an OpenCode adapter step and verifying the subprocess environment contains only base variables plus passthrough variables, not the full host environment.

**Acceptance Scenarios**:

1. **Given** an OpenCode adapter step runs, **When** the subprocess starts, **Then** only base environment variables (`HOME`, `PATH`, `TERM`, `TMPDIR`) plus `env_passthrough` variables are present
2. **Given** a host environment with `SECRET_TOKEN` set but NOT in `env_passthrough`, **When** the OpenCode adapter runs, **Then** the subprocess does NOT have access to `SECRET_TOKEN`

---

### Edge Cases

- What happens when the model identifier contains no provider prefix (e.g., `gpt-4o` instead of `openai/gpt-4o`)? The system should attempt inference from known model name patterns or fall back to a default provider.
- What happens when an unrecognized provider prefix is used (e.g., `custom/my-model`)? The system should pass it through as-is to OpenCode, which handles provider resolution.
- What happens when `cfg.Model` is empty and no persona model is set? The adapter should fall back to its current defaults (`anthropic` / `claude-sonnet-4-20250514`).
- What happens when the OpenCode binary is not installed? The existing error handling in `NewOpenCodeAdapter()` already covers this.
- What happens when provider-specific API keys are missing at runtime? The OpenCode CLI itself should surface the authentication error; Wave does not need to pre-validate.
- What happens when the model string contains multiple `/` characters (e.g., `provider/org/model`)? Only the first `/` should be treated as the provider delimiter.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The OpenCode adapter MUST read `cfg.Model` from `AdapterRunConfig` instead of hardcoding the model in `.opencode/config.json`
- **FR-002**: The OpenCode adapter MUST support a provider/model identifier format (e.g., `openai/gpt-4o`) where the prefix before the first `/` specifies the provider and the remainder specifies the model name
- **FR-003**: The OpenCode adapter MUST infer the provider from well-known model name patterns when no explicit provider prefix is given (e.g., `gpt-*` implies `openai`, `gemini-*` implies `google`, `claude-*` implies `anthropic`)
- **FR-004**: The OpenCode adapter MUST fall back to its current defaults (`anthropic` / `claude-sonnet-4-20250514`) when no model is specified in the persona or CLI override
- **FR-005**: The OpenCode adapter MUST adopt a curated environment model, passing only base variables (`HOME`, `PATH`, `TERM`, `TMPDIR`) plus explicitly listed `env_passthrough` variables to the subprocess
- **FR-006**: The OpenCode adapter MUST support the existing `runtime.sandbox.env_passthrough` mechanism to pass provider-specific API keys (e.g., `OPENAI_API_KEY`, `GOOGLE_API_KEY`) to the subprocess
- **FR-007**: The adapter reference documentation MUST include examples of configuring third-party model providers via the OpenCode adapter
- **FR-008**: The adapter concepts documentation MUST describe the provider/model identifier format and environment passthrough for multi-provider usage
- **FR-009**: The model resolution precedence MUST remain consistent with the Claude adapter: CLI `--model` flag > persona `model` field > adapter default

### Key Entities

- **Provider/Model Identifier**: A string in the format `provider/model-name` (e.g., `openai/gpt-4o`) used to configure which LLM provider and model the OpenCode adapter should use. The provider prefix maps to OpenCode's internal provider configuration.
- **AdapterRunConfig.Model**: The existing configuration field that carries the resolved model string from manifest persona or CLI override to the adapter's `Run()` method.
- **OpenCode Config** (`.opencode/config.json`): The configuration file generated by the adapter in the workspace, containing provider, model, and temperature settings consumed by the OpenCode CLI.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: A pipeline step using the OpenCode adapter with `model: openai/gpt-4o` generates a `.opencode/config.json` with `"provider": "openai"` and `"model": "gpt-4o"`
- **SC-002**: The model resolution precedence (CLI flag > persona config > adapter default) produces correct results for all three precedence levels
- **SC-003**: Environment variables listed in `env_passthrough` are present in the OpenCode subprocess environment; variables not listed are absent
- **SC-004**: All existing OpenCode adapter tests continue to pass (backward compatibility with current defaults)
- **SC-005**: The adapter documentation contains at least one complete example of configuring a third-party model provider end-to-end
- **SC-006**: Provider inference correctly maps at least 3 well-known model name prefixes (`gpt-*` -> openai, `gemini-*` -> google, `claude-*` -> anthropic) to their providers

## Clarifications

The following ambiguities were identified and resolved during specification refinement:

### C-1: What exact provider strings does OpenCode expect in config.json?

**Ambiguity**: The spec references provider values like `"openai"`, `"google"`, `"anthropic"` but doesn't confirm these match OpenCode's actual provider identifiers.

**Resolution**: Use OpenCode's documented provider identifiers as-is. The provider prefix in the `provider/model` format is passed directly to OpenCode's `config.json` `"provider"` field. Wave does not validate provider names — OpenCode handles provider resolution internally. This means unknown providers (e.g., `custom/my-model`) work as long as OpenCode supports them. The inference mapping only applies when no explicit prefix is given.

**Rationale**: OpenCode is an external tool that evolves independently. Passing provider strings through rather than validating them keeps Wave decoupled and forward-compatible.

### C-2: Should the provider inference mapping include providers beyond the 3 listed?

**Ambiguity**: The issue mentions GLM 5 as a use case, and providers like DeepSeek, Mistral, and others exist. FR-003 only mandates 3 mappings (`gpt-*`, `gemini-*`, `claude-*`).

**Resolution**: Implement only the 3 mandated well-known prefixes initially (`gpt-*` → `openai`, `gemini-*` → `google`, `claude-*` → `anthropic`). Users with other providers (GLM, DeepSeek, Mistral, etc.) use the explicit `provider/model` format (e.g., `glm/glm-5`). The inference map should be a simple Go map so it's trivially extensible later.

**Rationale**: The explicit `provider/model` format already covers all providers. Inference is a convenience for the 3 most common providers. Adding speculative mappings risks incorrect provider resolution.

### C-3: What base environment variables should the curated OpenCode environment include?

**Ambiguity**: FR-005 lists `HOME`, `PATH`, `TERM`, `TMPDIR` as base variables. The Claude adapter also includes telemetry suppression variables (`DISABLE_TELEMETRY=1`, `DISABLE_ERROR_REPORTING=1`, `CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY=1`, `DISABLE_BUG_COMMAND=1`). Should OpenCode get equivalent suppression vars?

**Resolution**: The curated environment for OpenCode should include the same 4 base variables (`HOME`, `PATH`, `TERM`, `TMPDIR`) and should NOT include Claude-specific telemetry suppression variables (those are meaningless to OpenCode). If OpenCode has its own telemetry suppression mechanism, that can be addressed in a follow-up. The implementation should extract the base environment logic into a shared helper (e.g., `buildBaseEnvironment`) used by both adapters, with adapter-specific additions layered on top.

**Rationale**: The Claude telemetry vars are Claude Code-specific. Passing them to OpenCode is harmless but noisy. A shared base helper avoids code duplication and ensures consistency as new adapters are added.

### C-4: Should the model be delivered to OpenCode via config.json only, or also as a CLI argument?

**Ambiguity**: The current OpenCode adapter writes model to `.opencode/config.json` and does not pass `--model` as a CLI argument. The spec doesn't specify the delivery mechanism.

**Resolution**: Write the provider and model to `.opencode/config.json` only, matching the current approach. Do NOT add `--model` as a CLI argument. OpenCode reads its configuration from `config.json`, and the adapter already generates this file in `prepareWorkspace`.

**Rationale**: OpenCode's CLI interface uses `config.json` for model configuration. Adding a CLI flag would require verifying that OpenCode supports it and could conflict with the config file. The current config.json approach is proven.

### C-5: How should the `buildEnvironment` logic be shared between Claude and OpenCode adapters?

**Ambiguity**: FR-005 says "adopt the same curated environment model as the Claude adapter." Should the OpenCode adapter duplicate the Claude adapter's `buildEnvironment` method or share it?

**Resolution**: Extract the base environment construction (`HOME`, `PATH`, `TERM`, `TMPDIR` + `EnvPassthrough` + `cfg.Env`) into a package-level function (e.g., `BuildCuratedEnvironment(cfg AdapterRunConfig) []string`) in `adapter.go`. Both Claude and OpenCode adapters call this shared function. Claude then appends its adapter-specific vars (telemetry suppression). This avoids duplication and ensures both adapters stay in sync.

**Rationale**: The Claude adapter's `buildEnvironment` already implements the exact pattern needed. Extracting the shared parts follows the DRY principle and makes adding future adapters easier.
