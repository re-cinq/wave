# 027: Sandbox Claude Code Sessions

## Problem Statement

Wave executes Claude Code as a subprocess but currently provides no OS-level isolation. The `ClaudeAdapter` runs `claude` with `--dangerously-skip-permissions` and relies solely on `--allowedTools` flags for access control. A compromised or misbehaving agent can read/write anywhere on the host filesystem, access any network endpoint, and modify system configuration. Wave's existing workspace isolation (copy-based with `chmod 0555` for read-only mounts) is enforced at the application layer, not the OS layer.

### Current State

From `internal/adapter/claude.go`, the adapter:
1. Creates `.claude/settings.json` in the workspace with `ClaudePermissions` (allow list only)
2. Passes `--dangerously-skip-permissions` on every invocation
3. Sets `cmd.Dir` to the workspace path but does not restrict filesystem access beyond that
4. Merges host environment variables into the subprocess (full `os.Environ()`)
5. Has no sandbox, network filtering, or resource limits

### Gap Analysis

| Capability | Current | With Sandboxing |
|---|---|---|
| Filesystem isolation | Application-level (copy + chmod) | OS-enforced (bubblewrap bind mounts) |
| Network isolation | None | Proxy-based domain filtering |
| Process isolation | None (shared host) | PID/mount/user namespaces |
| Syscall filtering | None | seccomp-BPF (pre-built filters) |
| Resource limits | None | cgroups via container or bwrap |
| Escape hatch | `--dangerously-skip-permissions` always set | `allowUnsandboxedCommands: false` blocks retries outside sandbox |

## Goals

1. Enable Claude Code's built-in sandbox for all adapter subprocess invocations with zero escape hatches
2. Generate per-persona `settings.json` files that map Wave's permission model to Claude Code's sandbox configuration
3. Restrict network access to a known domain allowlist per persona/step
4. Provide a bubblewrap-based Nix dev shell sandbox for Wave's own development environment
5. Define the adapter interface for future Docker-based per-step isolation

## Non-Goals

1. **Full Docker/microVM isolation** -- deferred to Phase 3 as future work
2. **macOS Seatbelt integration** -- Linux-only for now; Claude Code handles macOS natively
3. **io_uring hardening** -- acknowledged as a blind spot; mitigation deferred to kernel-level config
4. **OverlayFS workspace replacement** -- interesting optimization but orthogonal to sandboxing
5. **MCP server sandboxing** -- no MCP servers in current Wave architecture
6. **Custom seccomp profiles** -- Claude Code's sandbox-runtime ships pre-built BPF filters
7. **Multi-tenant/untrusted pipeline isolation** -- enterprise concern, not prototype scope

---

## Design

### Phase 1: Native Sandbox Integration (Immediate)

**Effort**: Low. Changes confined to `internal/adapter/claude.go` and `internal/adapter/adapter.go`.

#### 1.1 Generate Sandbox-Aware settings.json

Extend `ClaudeAdapter.prepareWorkspace` to write a `settings.json` that enables Claude Code's built-in sandbox. The current `ClaudeSettings` struct only covers `model`, `temperature`, `output_format`, and `permissions.allow`. Add sandbox configuration:

```go
type ClaudeSettings struct {
    Model        string            `json:"model"`
    Temperature  float64           `json:"temperature"`
    OutputFormat string            `json:"output_format"`
    Permissions  ClaudePermissions `json:"permissions"`
    Sandbox      SandboxSettings   `json:"sandbox"`
}

type SandboxSettings struct {
    Enabled                   bool            `json:"enabled"`
    AllowUnsandboxedCommands  bool            `json:"allowUnsandboxedCommands"`
    AutoAllowBashIfSandboxed  bool            `json:"autoAllowBashIfSandboxed"`
    Network                   NetworkSettings `json:"network"`
}

type NetworkSettings struct {
    AllowedDomains   []string `json:"allowedDomains"`
    AllowLocalBinding bool    `json:"allowLocalBinding"`
}
```

The generated `settings.json` for each step workspace:

```json
{
  "model": "opus",
  "permissions": {
    "allow": ["Read", "Write", "Edit", "Bash", "Glob", "Grep"],
    "deny": []
  },
  "sandbox": {
    "enabled": true,
    "allowUnsandboxedCommands": false,
    "autoAllowBashIfSandboxed": true,
    "network": {
      "allowedDomains": [
        "api.anthropic.com",
        "github.com",
        "*.github.com",
        "*.githubusercontent.com",
        "proxy.golang.org",
        "sum.golang.org"
      ],
      "allowLocalBinding": false
    }
  }
}
```

Key decisions:
- `allowUnsandboxedCommands: false` -- eliminates the escape hatch where Claude retries commands outside the sandbox on failure
- `autoAllowBashIfSandboxed: true` -- reduces permission prompts by 84% (Anthropic's internal data) since the sandbox enforces boundaries
- `allowLocalBinding: false` -- prevents the agent from starting local servers that could be used as exfiltration channels

#### 1.2 Per-Persona Permission Mapping

Map Wave's `Permissions` struct (`AllowedTools` + `Deny`) to Claude Code's `permissions` field in `settings.json`. This is already partially implemented but the `Deny` list is not currently written:

```go
type ClaudePermissions struct {
    Allow []string `json:"allow"`
    Deny  []string `json:"deny"`
}
```

Persona-specific sandbox network rules. In `prepareWorkspace`, derive the domain allowlist from the persona's role:

| Persona | Network Domains | Rationale |
|---|---|---|
| navigator | `api.anthropic.com` | Read-only analysis, no git push needed |
| implementer | `api.anthropic.com`, `github.com`, `*.github.com`, `*.githubusercontent.com`, `proxy.golang.org`, `sum.golang.org` | Needs git operations and Go module fetching |
| reviewer | `api.anthropic.com`, `github.com`, `*.github.com` | Needs to read PRs/issues but not fetch modules |
| craftsman | Same as implementer | Full build/push capability |
| philosopher | `api.anthropic.com` | Pure reasoning, no external access needed |
| auditor | `api.anthropic.com` | Read-only security analysis |

#### 1.3 Filesystem Deny Rules

Add filesystem deny rules to the sandbox settings to protect sensitive host paths, even though the sandbox already restricts writes to the working directory:

```json
{
  "sandbox": {
    "filesystem": {
      "denyRead": ["~/.ssh", "~/.aws", "~/.gnupg", "~/.config/gh"],
      "allowWrite": [".", "/tmp"],
      "denyWrite": [".env", "*.key", "*.pem"]
    }
  }
}
```

#### 1.4 Remove Reliance on --dangerously-skip-permissions

Currently `buildArgs` unconditionally adds `--dangerously-skip-permissions`. With the sandbox enabled and `autoAllowBashIfSandboxed: true`, this flag becomes unnecessary for sandboxed tools. However, during the transition:

1. Keep `--dangerously-skip-permissions` but pair it with explicit sandbox enforcement
2. The sandbox settings written to `.claude/settings.json` take effect regardless of this flag
3. Future: remove `--dangerously-skip-permissions` once sandbox coverage is validated

#### 1.5 Environment Variable Hygiene

Stop passing the full host environment. In `ClaudeAdapter.Run`, replace `os.Environ()` with a curated allowlist:

```go
// Instead of: mergedEnv := append(os.Environ(), cfg.Env...)
// Use a curated base environment:
baseEnv := []string{
    "HOME=" + os.Getenv("HOME"),
    "PATH=" + os.Getenv("PATH"),
    "TERM=" + os.Getenv("TERM"),
    "ANTHROPIC_API_KEY=" + os.Getenv("ANTHROPIC_API_KEY"),
    "TMPDIR=/tmp",
    "DISABLE_TELEMETRY=1",
    "DISABLE_ERROR_REPORTING=1",
    "CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY=1",
    "DISABLE_BUG_COMMAND=1",
}
// Conditionally add GH_TOKEN only for personas that need git
if personaNeedsGit(cfg.Persona) {
    if token := os.Getenv("GH_TOKEN"); token != "" {
        baseEnv = append(baseEnv, "GH_TOKEN="+token)
    }
}
mergedEnv := append(baseEnv, cfg.Env...)
```

This prevents leaking credentials from the host environment (AWS keys, database URLs, etc.) into the agent subprocess.

#### 1.6 AdapterRunConfig Extension

Add sandbox configuration to the run config so the pipeline executor can override defaults per step:

```go
type AdapterRunConfig struct {
    // ... existing fields ...
    SandboxEnabled    bool     // Enable Claude Code's built-in sandbox (default: true)
    AllowedDomains    []string // Network domain allowlist for this step
    DenyReadPaths     []string // Filesystem paths to deny reading
}
```

---

### Phase 2: Nix Flake Dev Shell Sandbox

**Effort**: Moderate. New `flake.nix` file (or extension of existing Nix config). No Go code changes.

This phase sandboxes the Wave development environment itself -- the human developer's shell where they run Wave, Claude Code, and other tools. It does not affect Wave's subprocess execution.

#### 2.1 Bubblewrap Sandbox Script

Based on the CFOAgent pattern with improvements from the Anthropic sandbox-runtime:

```nix
sandboxScript = pkgs.writeShellScriptBin "wave-sandbox" ''
  PROJECT_DIR="''${WAVE_PROJECT_DIR:-$PWD}"

  # Ensure bind targets exist before bwrap
  mkdir -p "$HOME/.claude"
  touch -a "$HOME/.claude.json"

  BWRAP_ARGS=(
    --unshare-all
    --share-net          # Phase 2a: replace with proxy-based filtering
    --die-with-parent
    --new-session        # Prevent terminal escape sequence attacks

    # Root filesystem -- READ-ONLY
    --ro-bind / /
    --dev /dev
    --proc /proc

    # Hide entire home directory
    --tmpfs "$HOME"

    # Writable: project directory
    --bind "$PROJECT_DIR" "$PROJECT_DIR"

    # Writable: Claude Code config
    --bind "$HOME/.claude" "$HOME/.claude"
    --bind "$HOME/.claude.json" "$HOME/.claude.json"

    # Writable: isolated temp (NOT shared with host)
    --tmpfs /tmp

    # Read-only: git config
    --ro-bind "$HOME/.gitconfig" "$HOME/.gitconfig"

    # Environment -- curated, not inherited
    --clearenv
    --setenv HOME "$HOME"
    --setenv PATH "$PATH"
    --setenv TERM "''${TERM:-xterm-256color}"
    --setenv TMPDIR "/tmp"
    --setenv SANDBOX_ACTIVE "1"
    --setenv ANTHROPIC_API_KEY "''${ANTHROPIC_API_KEY:-}"
    --setenv GH_TOKEN "''${GH_TOKEN:-}"

    --chdir "$PROJECT_DIR"
  )

  if [ $# -gt 0 ]; then
    exec ${pkgs.bubblewrap}/bin/bwrap "''${BWRAP_ARGS[@]}" "$@"
  else
    exec ${pkgs.bubblewrap}/bin/bwrap "''${BWRAP_ARGS[@]}" \
      ${pkgs.bash}/bin/bash
  fi
'';
```

Key differences from CFOAgent:
- `--new-session` prevents ANSI terminal escape injection
- `--clearenv` + explicit `--setenv` instead of inheriting all env vars
- `--tmpfs /tmp` instead of `--bind /tmp /tmp` (isolated, not shared with host)
- No `--ro-bind ~/.ssh` by default -- use `GH_TOKEN` + HTTPS for git operations
- No `--bind ~/.bun` -- Wave is a Go project, not a Bun project

#### 2.2 Flake.nix Additions

```nix
{
  description = "WAVE - Multi-agent pipeline orchestrator";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        commonPackages = with pkgs; [
          go_1_25 gh git jq curl sqlite
          bubblewrap
        ];

        sandboxScript = /* as defined above */;
      in {
        devShells = {
          # Default: sandboxed development shell
          default = pkgs.mkShell {
            buildInputs = commonPackages ++ [ sandboxScript ];
            shellHook = ''
              export WAVE_PROJECT_DIR="$PWD"

              # Auto-detect GH_TOKEN from gh CLI
              if command -v gh &>/dev/null && gh auth status &>/dev/null 2>&1; then
                export GH_TOKEN=$(gh auth token 2>/dev/null)
              fi

              # Auto-enter sandbox for interactive sessions
              if [ -t 0 ] && [ -z "$SANDBOX_ACTIVE" ]; then
                echo ""
                echo "  WAVE Sandboxed Development Shell"
                echo "  WRITE: $PWD, ~/.claude, /tmp"
                echo "  READ:  / (read-only)"
                echo "  NET:   full (use 'nix develop .#restricted' for filtered)"
                echo ""
                exec wave-sandbox bash
              fi
            '';
          };

          # Escape hatch: full access when you need it
          unsandboxed = pkgs.mkShell {
            buildInputs = commonPackages;
            shellHook = ''
              echo "WAVE Development Shell (NO SANDBOX)"
            '';
          };
        };
      }
    );
}
```

#### 2.3 Network Filtering (Phase 2a -- future enhancement within Phase 2)

Replace `--share-net` with proxy-based network filtering. Two approaches:

**Option A: Use Anthropic's sandbox-runtime directly**

```bash
npx @anthropic-ai/sandbox-runtime --config ~/.srt-settings.json -- claude ...
```

Where `~/.srt-settings.json` contains the domain allowlist. This handles the socat/proxy plumbing automatically.

**Option B: Manual socat + proxy setup in the bwrap script**

1. `--unshare-net` to remove all network interfaces inside the sandbox
2. Start an HTTP/SOCKS5 proxy on the host that filters by domain
3. Bridge via `socat` over a Unix domain socket into the sandbox
4. Set `HTTP_PROXY`/`HTTPS_PROXY` environment variables inside the sandbox

Option A is recommended for simplicity. Option B gives more control but requires maintaining proxy infrastructure.

#### 2.4 Gotchas and Platform Notes

- **NixOS TMPDIR**: When bubblewrap is setuid root, glibc strips `TMPDIR`. Always pass `--setenv TMPDIR /tmp`.
- **Ubuntu 24.04+**: Unprivileged user namespaces restricted by default. May need `sysctl kernel.unprivileged_userns_clone=1` or setuid bwrap.
- **macOS**: Bubblewrap does not work on macOS. Developers on macOS should use `nix develop .#unsandboxed` or rely on Claude Code's built-in Seatbelt sandbox.
- **Docker inside sandbox**: Docker cannot run inside bubblewrap. If a step needs Docker, it must be excluded or use an alternative isolation approach.

---

### Phase 3: Docker Container Per Step (Future)

**Effort**: High. New `DockerAdapter` in `internal/adapter/`, Dockerfile, container lifecycle management.

This phase is documented for architectural planning but not targeted for implementation in the prototype.

#### 3.1 DockerAdapter Interface

```go
type DockerAdapter struct {
    Image         string
    NetworkMode   string         // "none", "bridge", custom network
    Mounts        []Mount
    Resources     ResourceLimits
    SecurityOpts  []string
}

type Mount struct {
    Source   string
    Target   string
    ReadOnly bool
}

type ResourceLimits struct {
    MemoryMB  int
    CPUs      float64
    PidsLimit int
}
```

#### 3.2 Hardened Container Profile

```bash
docker run -it --rm \
  --user 1000:1000 \
  --read-only \
  --cap-drop ALL \
  --security-opt no-new-privileges:true \
  --memory=4g \
  --cpus=2 \
  --pids-limit=256 \
  --tmpfs /tmp:size=200M,noexec,nosuid \
  --tmpfs /home/wave/.cache:size=500M \
  -v "${WAVE_WORKSPACE}:/workspace:rw" \
  -v "${WAVE_ARTIFACTS}:/artifacts:ro" \
  -e ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" \
  --network=wave-net \
  wave-step:latest
```

#### 3.3 Artifact Injection/Extraction

- Previous step artifacts mounted read-only at `/artifacts/`
- Step output written to `/workspace/output/`
- After step completion, Wave copies output artifacts from the stopped container before removal
- Container filesystem is ephemeral -- no persistent state between steps

#### 3.4 Network Policy

Use Docker's internal network + iptables allowlisting (same pattern as Anthropic's devcontainer `init-firewall.sh`) or a CoreDNS sidecar that only resolves allowed domains.

---

## Configuration

### wave.yaml Sandbox Section

Add an optional `sandbox` section to the runtime configuration:

```yaml
runtime:
  workspace_root: .wave/workspaces
  sandbox:
    enabled: true                    # Enable sandbox for all steps (default: true)
    allow_unsandboxed_commands: false # Block escape hatch (default: false)
    default_allowed_domains:
      - api.anthropic.com
      - github.com
      - "*.github.com"
      - "*.githubusercontent.com"
      - proxy.golang.org
      - sum.golang.org
    deny_read_paths:
      - "~/.ssh"
      - "~/.aws"
      - "~/.gnupg"
    env_passthrough:                 # Host env vars to pass through (curated)
      - ANTHROPIC_API_KEY
      - GH_TOKEN
      - TERM
```

### Per-Persona Overrides

```yaml
personas:
  navigator:
    adapter: claude
    permissions:
      allowed_tools: ["Read", "Glob", "Grep", "Bash(git log*)"]
      deny: ["Write(*)", "Edit(*)"]
    sandbox:
      allowed_domains:
        - api.anthropic.com
      # No github.com -- navigator is read-only analysis

  implementer:
    adapter: claude
    permissions:
      allowed_tools: ["Read", "Write", "Edit", "Bash", "Glob", "Grep"]
    sandbox:
      allowed_domains:
        - api.anthropic.com
        - github.com
        - "*.github.com"
        - proxy.golang.org
        - sum.golang.org
```

---

## Security Model

### Attack Vectors Mitigated by Phase

| Attack Vector | Phase 1 | Phase 2 | Phase 3 |
|---|---|---|---|
| Host filesystem read (SSH keys, credentials) | Yes -- `denyRead` paths | Yes -- `--tmpfs $HOME` hides all | Yes -- container isolation |
| Host filesystem write (system config, dotfiles) | Yes -- sandbox restricts writes to CWD | Yes -- `--ro-bind / /` | Yes -- `--read-only` + tmpfs |
| Cross-project contamination | Partial -- per-step workspace | Yes -- only project dir mounted | Yes -- only workspace volume |
| Data exfiltration via network | Yes -- domain allowlist | Partial -- `--share-net` still allows all (until Phase 2a) | Yes -- iptables/CoreDNS filtering |
| Environment variable leakage | Yes -- curated env passthrough | Yes -- `--clearenv` | Yes -- explicit `-e` flags only |
| Privilege escalation | Partial -- no-new-privileges via sandbox | Yes -- user namespace isolation | Yes -- `--cap-drop ALL` |
| Terminal escape injection | No | Yes -- `--new-session` | N/A (no terminal) |
| Process inspection/signaling | No | Yes -- PID namespace (`--unshare-pid`) | Yes -- PID namespace |
| Fork bomb / resource exhaustion | No | No (bwrap has no cgroups) | Yes -- `--pids-limit`, `--memory` |

### Defense in Depth Layers

1. **Wave permissions** (`PermissionChecker`): Application-level tool allow/deny patterns
2. **Claude Code permissions** (`settings.json`): Tool-level allow/deny rules enforced by Claude Code itself
3. **Claude Code sandbox** (bubblewrap/seatbelt): OS-level filesystem and network isolation for the Bash tool
4. **Nix dev shell sandbox** (bubblewrap): OS-level isolation for the entire development session
5. **Docker container** (future): Kernel namespace isolation with resource limits

### What Is NOT Protected

- **DNS tunneling**: All phases allow DNS resolution. Exfiltration via DNS encoding is possible. Mitigation: CoreDNS proxy (Phase 3) or DNS query monitoring.
- **Domain fronting**: An allowed domain like `github.com` could be used to relay data to unauthorized endpoints via CDN fronting. Mitigation: HTTPS inspection (complex, deferred).
- **io_uring bypass**: Processes can perform I/O operations via io_uring without triggering seccomp-BPF filters. Mitigation: Disable io_uring via sysctl on kernel 6.6+ (`io_uring_disabled=2`).
- **Side channels**: Timing attacks, CPU cache side channels, etc. Out of scope for process-level sandboxing.

---

## Testing Strategy

### Phase 1 Tests

1. **settings.json generation**: Verify that `prepareWorkspace` writes correct sandbox config for each persona type. Table-driven tests covering navigator, implementer, reviewer, craftsman, philosopher, auditor.

2. **Permission mapping**: Verify that Wave's `Permissions.Deny` list is correctly written to Claude Code's `permissions.deny` in settings.json (currently missing).

3. **Environment hygiene**: Verify that the subprocess environment does not contain unexpected variables. Set a canary env var (e.g., `AWS_SECRET_ACCESS_KEY=test`), run the adapter, and verify it is not in the child process environment.

4. **Domain allowlist per persona**: Verify that persona-specific domain lists are correctly generated. Navigator should not have `github.com`; implementer should.

5. **Integration test**: Run a Claude Code subprocess with sandbox enabled in a temp workspace. Verify that:
   - Writes outside the workspace directory fail
   - Network requests to non-allowed domains fail
   - The step completes successfully for allowed operations

### Phase 2 Tests

1. **Sandbox script smoke test**: Run `wave-sandbox echo "hello"` and verify output.

2. **Filesystem isolation**: Inside sandbox, verify:
   - `ls ~/.ssh` fails or shows empty directory
   - `ls ~/.aws` fails or shows empty directory
   - Writing to `/etc/test` fails (read-only root)
   - Writing to workspace directory succeeds
   - Writing to `/tmp` succeeds

3. **Environment isolation**: Inside sandbox, verify:
   - `env | grep AWS` returns nothing (curated env)
   - `ANTHROPIC_API_KEY` is set (explicitly passed)
   - `SANDBOX_ACTIVE=1` is set

4. **Interactive detection**: Verify sandbox auto-entry only triggers for interactive (`-t 0`) sessions, not for `nix develop --command ...`.

### Phase 3 Tests (Future)

1. Docker container lifecycle (create, run, extract artifacts, destroy)
2. Resource limit enforcement (OOM kill at memory limit, PID limit)
3. Network policy enforcement (iptables rules block unauthorized traffic)
4. Artifact injection from previous steps (read-only mount)

---

## Open Questions

1. **Should Wave generate `.claude/settings.json` or `.claude/settings.local.json`?** The `settings.local.json` file is not synced and takes precedence over `settings.json`. Using it would avoid conflicting with any user-level settings, but Claude Code's sandbox documentation primarily references `settings.json`.

2. **How to handle adapters other than Claude Code?** The `opencode` adapter and future adapters may have their own sandboxing mechanisms. Should the sandbox config in `wave.yaml` be adapter-agnostic, or should each adapter define its own sandbox integration?

3. **Should `--dangerously-skip-permissions` be removed?** With `autoAllowBashIfSandboxed: true`, sandboxed Bash commands auto-execute without prompting. Non-sandboxed tools (Read, Edit, Write) still go through Claude Code's permission flow. Removing the flag would add safety but might cause prompts that block headless execution. Needs testing.

4. **Network proxy for Phase 2a**: Should Wave ship its own proxy, use Anthropic's `@anthropic-ai/sandbox-runtime` npm package, or defer network filtering to the Docker phase? The npm dependency would be the first non-Go dependency in Wave.

5. **Per-step vs per-pipeline sandbox config**: Should sandbox settings be configurable at the pipeline step level (more granular) or only at the persona level (simpler)? The current design uses persona-level defaults with runtime override via `AdapterRunConfig`.

6. **bubblewrap availability**: Should Wave detect bubblewrap at startup and warn/fail if not available? Or should it gracefully degrade to unsandboxed execution with a warning?
