# Research: Manifest & Pipeline Design

**Branch**: `014-manifest-pipeline-design`
**Date**: 2026-02-01

## R1: SQLite Driver — CGo vs Pure Go

**Decision**: Use `modernc.org/sqlite` (pure Go SQLite implementation)

**Rationale**: Constitution Principle 1 requires a single binary with
no runtime dependencies. The standard `github.com/mattn/go-sqlite3`
requires CGo and links against the C SQLite library, making
cross-compilation harder and breaking the zero-dependency guarantee.
`modernc.org/sqlite` is a transpilation of SQLite to Go — no C
compiler needed, fully static binary.

**Alternatives considered**:
- `go-sqlite3` (CGo): Better performance but violates single-binary
  principle due to C dependency chain.
- BoltDB/bbolt: Key-value store, not relational. Pipeline state
  queries (e.g., "find all steps in Retrying state") are naturally
  relational and awkward in KV.
- Flat files (JSON): No concurrent access safety, no transactional
  guarantees for state updates during parallel matrix workers.

## R2: CLI Framework

**Decision**: Use `github.com/spf13/cobra`

**Rationale**: Cobra is the de facto standard for Go CLIs. It provides
subcommand routing, flag parsing, help generation, and shell
completions out of the box. The muzzle CLI has multiple subcommands
(`init`, `validate`, `run`, `do`, `resume`, `clean`) which map
directly to cobra's command tree model.

**Alternatives considered**:
- `github.com/urfave/cli/v2`: Similar capabilities but less ecosystem
  adoption. Cobra's integration with `pflag` (POSIX flags) is cleaner.
- Standard `flag` package: No subcommand support. Would require manual
  routing.
- No framework (raw `os.Args`): Unnecessary complexity for a CLI with
  6+ subcommands.

## R3: YAML Parsing

**Decision**: Use `gopkg.in/yaml.v3`

**Rationale**: yaml.v3 supports YAML 1.2, handles anchors/aliases
(useful for DRY manifest definitions), and provides a node-level API
for better error messages with line numbers. Manifest validation errors
need to point to specific YAML locations.

**Alternatives considered**:
- `sigs.k8s.io/yaml`: Converts YAML to JSON first, losing YAML-specific
  features (anchors, multi-line strings). Manifest format uses these.
- `github.com/goccy/go-yaml`: Good performance but less mature. yaml.v3
  is battle-tested.

## R4: JSON Schema Validation

**Decision**: Use `github.com/santhosh-tekuri/jsonschema/v6`

**Rationale**: Supports JSON Schema draft 2020-12, is pure Go, and
handles `$ref` resolution. Handover contracts with `type: json_schema`
need full spec compliance to validate navigator output, task lists, etc.

**Alternatives considered**:
- `github.com/xeipuuv/gojsonschema`: Only supports draft-07. Less
  actively maintained.
- Manual validation: Error-prone and doesn't scale as contracts grow
  in complexity.

## R5: Claude Code Subprocess Integration

**Decision**: Wrap `claude -p` via `os/exec.Command` with JSON stdout
parsing, process group management, and environment variable forwarding.

**Rationale**: Claude Code's headless mode (`claude -p`) accepts a
prompt via argument or stdin, outputs JSON events to stdout, and exits
with code 0 on success. The Go adapter:
1. Builds the command with `--allowedTools`, `--output-format json`,
   and persona-specific flags.
2. Sets up `.claude/settings.json` in the ephemeral workspace to
   configure hooks and permissions.
3. Projects the persona's system prompt as `CLAUDE.md` in the workspace.
4. Streams stdout for progress events and token count monitoring.
5. Uses process groups (`syscall.SysProcAttr{Setpgid: true}`) so
   timeout kills the entire process tree, not just the parent.

**Alternatives considered**:
- Claude Code SDK (TypeScript): Would require Node.js runtime,
  violating Principle 1.
- Direct Anthropic API: Bypasses Claude Code's tool use, hooks, and
  permission model. We'd have to reimplement all of that.
- WebSocket/gRPC to a running Claude Code server: Claude Code doesn't
  expose a server mode. Subprocess is the only integration surface.

## R6: Workspace Isolation Strategy

**Decision**: Use filesystem copy + symlinks for workspace creation.
Readonly mounts via filesystem perm issions (chmod), not OS-level mount
namespaces.

**Rationale**: OS-level mount namespaces (Linux `unshare`) require
root or specific capabilities. Muzzle must run as a normal user. The
simpler approach:
1. Create workspace directory in configurable root (default `/tmp/muzzle/`).
2. For `readonly` mounts: symlink the source directory and set the
   workspace permission bits to read-only for the subprocess user.
3. For `readwrite` mounts: copy the relevant files into the workspace.
4. For artifact injection: copy artifact files into the workspace at
   the specified paths.

This doesn't provide kernel-level isolation but satisfies the safety
goal: agents work in their own directory, not the source repo directly.

**Alternatives considered**:
- Docker/container-per-step: Heavy overhead, requires Docker daemon,
  violates single-binary principle.
- Linux namespaces (`unshare`): Requires elevated privileges.
- FUSE overlay filesystem: Complex, platform-specific, requires FUSE
  installation.

## R7: Token Count Monitoring for Relay

**Decision**: Parse Claude Code's JSON output stream for token usage
events and track cumulative tokens against the model's context window.

**Rationale**: Claude Code emits token usage in its JSON event stream.
The adapter monitors these events in real-time. When cumulative tokens
reach the configured threshold (default 80% of model context window),
the adapter signals the executor to trigger relay.

**Alternatives considered**:
- Approximate by message count: Inaccurate. Messages vary wildly in
  token count.
- External tokenizer (tiktoken equivalent): Adds dependency and may
  not match the model's actual tokenization.
- Fixed message limit: Too rigid. Some messages are 10 tokens, others
  are 10,000.

## R8: Structured Event Format

**Decision**: Newline-delimited JSON (NDJSON) to stdout. Each line is
a self-contained JSON object with fields: `timestamp`, `pipeline_id`,
`step_id`, `state`, `duration_ms`, `message`.

**Rationale**: NDJSON is parseable by `jq`, CI log processors, and
monitoring tools. It's also human-scannable (one event per line). The
format supports both terminal display (via a pretty-printer flag) and
machine ingestion.

**Alternatives considered**:
- Structured logging to stderr: Mixes with adapter subprocess stderr.
  Keeping events on stdout and errors on stderr maintains clean
  separation.
- Custom protocol: No tooling support. NDJSON is a standard.
