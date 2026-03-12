# ADR-002: Adapter Extensibility Model for Additional LLM Backends

## Status

Proposed

## Date

2026-03-12

## Context

Wave currently supports LLM backends through compiled-in adapter implementations. The `AdapterRunner` interface in `internal/adapter/adapter.go` defines the contract — `Run(ctx, cfg AdapterRunConfig) returns AdapterResult` — and concrete implementations exist for Claude Code (`claude.go`), GitHub (`github.go`), OpenCode (`opencode.go`), and interactive mode (`interactive.go`). Each adapter is compiled directly into the Wave binary.

As Wave matures beyond Claude Code, there is growing need to support additional LLM backends: OpenAI-compatible APIs, Google Gemini, local models via Ollama, and proprietary enterprise endpoints. The current model requires a Go code contribution, a PR to the main repository, and a new Wave release for every new backend. This creates friction for community adoption and limits Wave's utility in environments where Claude Code is not the primary LLM.

The decision is constrained by several architectural principles:

- **Single static binary** — Wave must remain a single binary with no runtime dependencies beyond adapter executables (Constitutional Principle 1).
- **Security model** — All inputs are sanitized, permissions enforced, and credentials never written to disk. Adapter extensions must not weaken these guarantees.
- **Subprocess execution model** — Wave already spawns adapter binaries as child processes. `ClaudeAdapter` runs `claude` via `ProcessGroupRunner` with NDJSON stream event parsing, SIGTERM/SIGKILL lifecycle management, and `settings.json` generation.
- **Prototype phase** — No backward compatibility constraint. Speed of iteration is prioritized over API stability.
- **Manifest as source of truth** — All configuration is declared in `wave.yaml` (Constitutional Principle 2).

## Decision

Adopt an **external process adapter protocol** using a versioned JSON-RPC specification over stdin/stdout.

Wave will define a standardized protocol for communicating with external adapter binaries. The orchestrator spawns the adapter process, sends the `AdapterRunConfig` as a JSON message on stdin, and reads back `AdapterResult` and streaming events as NDJSON on stdout. Adapter binaries are discovered via PATH or a directory configured in `wave.yaml`.

Built-in adapters (Claude Code, GitHub) continue as internal Go implementations. The external protocol is an additive extension point — it does not replace the existing `AdapterRunner` interface but rather provides a new implementation (`ExternalAdapter`) that bridges the protocol to the internal interface.

The protocol starts minimal — spawn, send config, read result — and grows incrementally as real adapter requirements emerge.

## Options Considered

### Option 1: Built-in Adapters Only

Keep the current model where all adapter implementations are compiled into the Wave binary. New backends are added as Go source files in `internal/adapter/` via pull requests.

**Pros:**
- Single binary constraint perfectly preserved — no runtime dependency resolution
- Full type safety and compile-time verification across all adapters
- Uniform testing with `go test ./...` catches regressions simultaneously
- Security model simplified — all adapter code is reviewed and trusted at compile time
- Stream event parsing can share internal utilities without serialization boundaries
- No protocol versioning concerns between adapter and orchestrator
- Proven pattern — four existing adapters demonstrate the approach works

**Cons:**
- Every new backend requires a code contribution, PR review, and Wave release
- Users cannot add proprietary or custom adapters without forking the project
- Binary size grows linearly with adapter count and their dependencies
- Community contribution friction is high — requires Go expertise and familiarity with internals
- Adapter-specific dependencies are pulled into all builds even when unused

**Assessment:** Low risk, trivial effort, but does not solve the extensibility problem. Appropriate as the sole model only if Wave remains a single-LLM tool.

### Option 2: External Process Adapter Protocol (stdin/stdout JSON-RPC) — Recommended

Define a versioned JSON-RPC protocol over stdin/stdout for external adapter processes. Wave spawns the adapter binary, sends configuration as JSON, and reads streaming events and results as NDJSON.

**Pros:**
- Adapters can be written in any language — Python for local model wrappers, shell scripts for custom APIs, Rust for performance-critical backends
- Decouples adapter release cycle from Wave releases
- Preserves single-binary constraint — Wave itself remains static, adapters are separate executables
- Natural extension of the existing subprocess model (Wave already spawns `claude` as a child process)
- Community can publish third-party adapters without PRs to the main repository
- Process isolation provides security boundaries between Wave and adapter code
- Protocol can be validated using Wave's existing contract validation infrastructure

**Cons:**
- Protocol design and versioning adds complexity — must handle schema evolution
- Cross-process debugging is harder than in-process adapter calls
- Stream event latency increases due to serialization/deserialization overhead
- Adapter discovery and dependency management becomes the user's responsibility
- Security model must extend to validate external adapter binaries
- Error handling across process boundaries is more complex (crashes, hangs, malformed output)

**Assessment:** Medium risk, medium effort. Best long-term extensibility with acceptable complexity cost.

### Option 3: Go Plugin System (hashicorp/go-plugin or Native Plugins)

Implement a plugin system where adapters are compiled as shared objects or use hashicorp/go-plugin's gRPC model. Plugins implement the `AdapterRunner` interface and are loaded at runtime.

**Pros:**
- Rich type-safe interface with full Go struct passing
- hashicorp/go-plugin provides battle-tested infrastructure with health checking
- Established pattern in Go ecosystem (Terraform providers, Vault backends)

**Cons:**
- Go native plugins require identical Go version and dependency versions — extremely fragile
- hashicorp/go-plugin adds gRPC dependency, significantly increasing binary size
- Breaks single static binary constraint — runtime dependency on `.so` files or plugin binaries
- CGO dependency conflicts with pure-Go cross-compilation goals
- Plugin API must be stabilized prematurely — incompatible with prototype phase
- Loading arbitrary code into the Wave process eliminates process-level security isolation
- Native Go plugin support is unreliable across platforms, especially macOS and Windows

**Assessment:** High risk, large effort. The fragility of Go's plugin ecosystem and the loss of process isolation make this a poor fit for Wave's security model and cross-platform goals.

### Option 4: Configuration-Driven Generic Adapter with Template Invocation

Define adapters entirely in `wave.yaml` using template-driven configuration — command templates, output parsing strategy, environment variables, and token counting method. No per-backend code required.

**Pros:**
- Zero code for new adapters — pure YAML configuration
- Aligns with manifest-as-source-of-truth principle
- Lowest community friction — no Go knowledge required
- Single binary preserved, no external protocol, just configuration
- Existing `ProcessGroupRunner` handles subprocess lifecycle

**Cons:**
- Limited expressiveness — complex adapter logic (NDJSON stream parsing, multi-turn conversations, auth refresh) cannot be captured in templates
- `ClaudeAdapter` has 400+ lines of specialized stream parsing that templates cannot replicate
- Error handling and failure classification require per-backend heuristics beyond template capability
- Token counting varies significantly between backends — not generalizable via config fields
- No type safety — template errors caught only at runtime
- Streaming events require backend-specific parsing that templates cannot generalize

**Assessment:** Low effort, medium risk. Useful as a rapid prototyping mechanism but insufficient as the primary extensibility model for production adapters with complex streaming requirements.

## Consequences

### Positive

- **Community extensibility** — third-party adapters can be published and installed independently of Wave releases, enabling an ecosystem of LLM backend integrations.
- **Language agnosticism** — adapter authors are not limited to Go. Python wrappers for Ollama, shell scripts for custom APIs, and compiled binaries for high-performance backends are all viable.
- **Preserved architecture** — Wave's single static binary, security model, and subprocess execution pattern are all maintained. The protocol is a formalization of what Wave already does.
- **Incremental adoption** — built-in adapters continue unchanged. The protocol is purely additive and can be adopted backend-by-backend.
- **Process isolation** — external adapters run in separate processes, providing natural security boundaries that align with Wave's defense-in-depth approach.

### Negative

- **Protocol maintenance burden** — the JSON-RPC protocol becomes a public API surface that must be documented, versioned, and maintained. Breaking changes require coordinated upgrades across adapter implementations.
- **Increased debugging complexity** — cross-process failures (adapter crashes, malformed NDJSON, hung processes) are harder to diagnose than in-process errors. Requires investment in adapter-side error reporting and Wave-side diagnostics.
- **User responsibility for adapter lifecycle** — users must install, update, and manage external adapter binaries. Wave cannot guarantee the availability or correctness of third-party adapters.
- **Latency overhead** — serialization/deserialization through stdin/stdout pipes adds marginal latency to every adapter invocation, though this is negligible relative to LLM response times.

### Neutral

- Built-in adapters (`claude.go`, `github.go`) remain as internal implementations and are unaffected by this change.
- The `AdapterRunner` interface in `internal/adapter/adapter.go` gains a new implementation (`ExternalAdapter`) but the interface itself does not change.
- Manifest schema (`wave.yaml`) extends to support external adapter declarations with binary path and protocol version.
- Adapter-specific configuration (model, temperature, API keys) continues to flow through `AdapterRunConfig` — the protocol simply serializes this struct.

## Implementation Notes

1. **Define the protocol specification** — Create `docs/adapter-protocol.md` specifying the JSON-RPC message format, NDJSON streaming event schema, lifecycle signals (start, heartbeat, complete, error), and versioning strategy. Start with protocol version `v1` and keep it minimal.

2. **Implement `ExternalAdapter`** — Add `internal/adapter/external.go` implementing `AdapterRunner`. This adapter spawns the configured binary, writes `AdapterRunConfig` as JSON to stdin, reads NDJSON events from stdout, and maps them to Wave's internal event types. Reuse `ProcessGroupRunner` for subprocess lifecycle management (SIGTERM/SIGKILL, timeout handling).

3. **Extend manifest schema** — Add an `external` adapter type to `wave.yaml` adapter declarations:
   ```yaml
   adapters:
     ollama:
       type: external
       command: wave-adapter-ollama
       protocol: v1
   ```
   The `command` field is resolved via PATH or an absolute path. Protocol version is mandatory.

4. **Adapter discovery** — Implement binary resolution in `internal/adapter/` that checks PATH and an optional `$WAVE_ADAPTER_DIR` directory. Validate that the binary exists and is executable before pipeline execution begins (preflight check in `internal/preflight/`).

5. **Protocol validation** — Use Wave's existing contract validation infrastructure to validate adapter responses against the protocol schema. Malformed responses should produce clear error messages indicating whether the issue is in the adapter or the protocol configuration.

6. **Security considerations** — External adapter binaries inherit Wave's sandbox constraints (Nix bubblewrap on Linux). Document that external adapters run with the same filesystem and network restrictions as built-in adapters. Add adapter binary path validation to `internal/security/` to prevent path traversal.

7. **Testing strategy** — Create a test adapter binary (`tests/testdata/mock-adapter`) that implements the protocol for integration testing. Add table-driven tests for protocol parsing, error handling (malformed JSON, process crashes, timeouts), and event stream mapping.

8. **Migration** — No migration required. This is purely additive. Existing `wave.yaml` configurations and built-in adapters are unaffected. Users opt in to external adapters by declaring them in their manifest.
