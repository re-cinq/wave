# Sandboxing Claude Code with Bubblewrap in a Nix Development Shell

## Comprehensive Research Report

**Date**: 2026-02-09
**Context**: Wave project -- exploring bubblewrap-based sandboxing for AI agent execution within Nix dev shells

---

## Table of Contents

1. [Analysis of the CFOAgent Example](#1-analysis-of-the-cfoagent-example)
2. [How Bubblewrap Works](#2-how-bubblewrap-works)
3. [Claude Code's Filesystem Requirements](#3-claude-codes-filesystem-requirements)
4. [Anthropic's Official Sandbox Runtime](#4-anthropics-official-sandbox-runtime)
5. [Nix Flake Patterns for Sandboxed Dev Shells](#5-nix-flake-patterns-for-sandboxed-dev-shells)
6. [Security Considerations for AI Agents](#6-security-considerations-for-ai-agents)
7. [Limitations and Gotchas](#7-limitations-and-gotchas)
8. [Recommendations for Wave](#8-recommendations-for-wave)
9. [Sources](#9-sources)

---

## 1. Analysis of the CFOAgent Example

The CFOAgent project (`/home/mwc/Coding/recinq/CFOAgent/flake.nix`) provides a working reference implementation of bubblewrap sandboxing integrated into a Nix dev shell. Here is how it works:

### Architecture

The flake defines multiple dev shells:
- **`default`** -- Sandboxed shell that auto-enters a bwrap jail on interactive sessions
- **`yolo`** -- Full filesystem access, no sandbox
- **`firebase`** / **`web`** -- Purpose-specific shells for service startup

### The Sandbox Script

The core sandbox is implemented as a `writeShellScriptBin "enter-sandbox"` that constructs bwrap arguments dynamically:

```nix
sandboxScript = pkgs.writeShellScriptBin "enter-sandbox" ''
  PROJECT_DIR="''${SANDBOX_PROJECT_DIR:-$PWD}"

  BWRAP_ARGS=(
    --unshare-all        # Unshare ALL namespaces (user, mount, PID, IPC, UTS, net)
    --share-net          # Re-enable network (overrides --unshare-all's --unshare-net)
    --die-with-parent    # Kill sandbox if parent dies

    --ro-bind / /        # Mount entire root filesystem as READ-ONLY
    --dev /dev           # Create minimal /dev (null, zero, urandom, etc.)
    --proc /proc         # Fresh /proc mount

    --tmpfs "$HOME"      # Overlay HOME with empty tmpfs (hides everything)

    # Selectively bind writable paths OVER the tmpfs
    --bind "$PROJECT_DIR" "$PROJECT_DIR"
    --bind "$HOME/.claude" "$HOME/.claude"
    --bind "$HOME/.claude.json" "$HOME/.claude.json"
    --bind "$HOME/.bun" "$HOME/.bun"

    # Read-only binds for git operations
    --ro-bind "$HOME/.gitconfig" "$HOME/.gitconfig"
    --ro-bind "$HOME/.ssh" "$HOME/.ssh"

    --bind /tmp /tmp     # Writable /tmp

    # Environment passthrough
    --setenv HOME "$HOME"
    --setenv PATH "$PATH"
    --setenv SANDBOX_ACTIVE "1"
    --chdir "$PROJECT_DIR"
  )

  exec bwrap "''${BWRAP_ARGS[@]}" bash
'';
```

### Key Design Decisions in CFOAgent

1. **`--ro-bind / /` then `--tmpfs $HOME`**: The entire root is mounted read-only, then `$HOME` is overlaid with a tmpfs. This means the sandbox can read system files (`/usr`, `/etc`, `/nix/store`) but sees an empty HOME directory.

2. **Selective bind-mounts over tmpfs**: Specific directories (`~/.claude`, project dir, `~/.bun`) are bind-mounted writable on top of the tmpfs overlay. Order matters -- bwrap processes arguments left to right, so later mounts overlay earlier ones.

3. **Network is shared**: `--unshare-all --share-net` unshares everything but then re-shares network. This is appropriate when the agent needs API access (Claude Code hits `api.anthropic.com`).

4. **SSH keys are read-only**: `--ro-bind "$HOME/.ssh" "$HOME/.ssh"` allows git operations over SSH without letting the agent modify or exfiltrate keys (though network access means exfiltration via network is still possible).

5. **Auto-entry for interactive sessions**: The `shellHook` detects interactive terminals and auto-executes `enter-sandbox`, making the sandbox transparent to the user.

6. **Escape hatch**: The `yolo` shell provides full access when needed.

### What CFOAgent Gets Right

- Simple, understandable bwrap invocation
- Proper ordering of mount overlays
- Pre-flight `mkdir -p` and `touch` to ensure bind targets exist
- Detection of interactive vs non-interactive sessions
- Clear user feedback about what is restricted

### What CFOAgent Could Improve

- No network filtering (full network access means exfiltration is possible)
- SSH keys readable means they could be sent over the network
- No seccomp filtering
- No `--new-session` flag (terminal escape sequences could leak)
- `--bind /tmp /tmp` shares host `/tmp` rather than creating isolated tmpfs

---

## 2. How Bubblewrap Works

Bubblewrap (`bwrap`) is a lightweight, unprivileged sandboxing tool that uses Linux kernel namespaces. It is the same technology that powers Flatpak.

### Core Mechanism

Bubblewrap creates a new, empty mount namespace where the root is on a tmpfs invisible from the host. The user specifies exactly what parts of the host filesystem to make visible inside the sandbox. Everything else is inaccessible.

### Key Flags Reference

| Flag | Purpose |
|------|---------|
| `--ro-bind SRC DEST` | Mount SRC at DEST, read-only |
| `--bind SRC DEST` | Mount SRC at DEST, read-write |
| `--dev DEST` | Create minimal `/dev` (null, zero, urandom, etc.) |
| `--proc DEST` | Mount fresh `/proc` |
| `--tmpfs DEST` | Create ephemeral tmpfs at DEST |
| `--symlink SRC DEST` | Create symlink inside sandbox |
| `--unshare-all` | Unshare all namespaces (user, mount, PID, IPC, UTS, net) |
| `--unshare-net` | Unshare only network namespace |
| `--share-net` | Re-share network (after `--unshare-all`) |
| `--unshare-pid` | Isolate process IDs |
| `--die-with-parent` | Kill sandbox when parent exits |
| `--new-session` | New terminal session (prevents keyboard escape) |
| `--clearenv` | Clear all environment variables |
| `--setenv KEY VALUE` | Set environment variable inside sandbox |
| `--chdir PATH` | Set working directory |

### Critical Ordering Rule

Arguments are processed left to right. Later mounts overlay earlier ones. For example:

```bash
# 1. Mount root read-only
--ro-bind / /
# 2. Overlay HOME with empty tmpfs (hides everything under HOME)
--tmpfs $HOME
# 3. Bind-mount specific dirs writable OVER the tmpfs
--bind $HOME/.claude $HOME/.claude
```

### Performance

Bubblewrap uses kernel namespaces, not virtualization. The overhead is negligible:
- **CPU**: Native speed (no instruction-level overhead)
- **I/O**: Minimal overhead from bind mounts
- **Startup**: Milliseconds to set up namespaces
- **Benchmark**: 100 commands via bwrap (0.374s) vs Docker (1.126s) -- 3x faster than Docker

---

## 3. Claude Code's Filesystem Requirements

### Directory Layout

Claude Code stores configuration in predictable locations:

```
~/.claude/                        # Primary config directory
├── settings.json                 # Global user settings
├── settings.local.json           # Local settings (not synced)
├── CLAUDE.md                     # Global instructions
├── .credentials.json             # API credentials (Linux/Windows)
├── statsig/                      # Analytics cache
├── commands/                     # Custom slash commands
├── projects/                     # Per-project conversation history
│   └── {encoded-project-path}/   # URL-encoded path
├── shell-snapshots/              # Shell state snapshots
└── todos/                        # Todo tracking

~/.claude.json                    # MCP server configuration (separate file!)
```

### Required Paths

| Path | Access | Purpose |
|------|--------|---------|
| `~/.claude/` | **Read-Write** | Config, credentials, conversation history, settings |
| `~/.claude.json` | **Read-Write** | MCP server configuration |
| Project directory | **Read-Write** | The actual codebase being worked on |
| `/tmp` | **Read-Write** | Temporary files, IPC |
| `/nix/store` | **Read-Only** | Nix packages (binaries, libraries) |
| `/etc/resolv.conf` | **Read-Only** | DNS resolution |
| `/etc/ssl/certs/` | **Read-Only** | TLS certificates for HTTPS |
| `/etc/hosts` | **Read-Only** | Hostname resolution |
| `~/.gitconfig` | **Read-Only** | Git configuration |
| `~/.ssh/` | **Read-Only** | SSH keys for git operations |

### CLAUDE_CONFIG_DIR Override

The `CLAUDE_CONFIG_DIR` environment variable allows overriding the config directory location. This is useful for:
- Running multiple Claude accounts
- Isolating config per-sandbox
- CI/CD environments

**Caveat**: Even with `CLAUDE_CONFIG_DIR` set, Claude Code may still create local `.claude/` directories in project workspaces.

### Config Directory History

- **Pre-v1.0.30**: `~/.claude/`
- **v1.0.30+**: Changed to `~/.config/claude/` (undocumented breaking change)
- **Current**: `CLAUDE_CONFIG_DIR` environment variable respected; both paths may be checked

### What About `~/.config/claude-code/`?

Claude Code does NOT use `~/.config/claude-code/`. This is a common misconception. The agent sometimes suggests this path based on XDG conventions, but Claude Code has its own specific paths.

---

## 4. Anthropic's Official Sandbox Runtime

Anthropic published an open-source [sandbox-runtime](https://github.com/anthropic-experimental/sandbox-runtime) (srt) that powers Claude Code's built-in sandbox mode.

### How It Works on Linux

The sandbox-runtime constructs bwrap arguments programmatically:

```typescript
const bwrapArgs: string[] = ['--new-session', '--die-with-parent'];

// Always unshare network
bwrapArgs.push('--unshare-net');
bwrapArgs.push('--unshare-pid');
bwrapArgs.push('--proc', '/proc');
bwrapArgs.push('--dev', '/dev');

// Start with read-only root, then allow specific writes
bwrapArgs.push('--ro-bind', '/', '/');

// Selectively allow writes
for (const path of allowedWritePaths) {
  bwrapArgs.push('--bind', path, path);
}

// Deny reads by overlaying with tmpfs/dev-null
for (const path of deniedReadPaths) {
  bwrapArgs.push('--tmpfs', path);  // for directories
  // or: bwrapArgs.push('--ro-bind', '/dev/null', path);  // for files
}
```

### Network Proxy Architecture

The official sandbox uses a sophisticated proxy-based network filtering system:

1. `--unshare-net` removes all network interfaces inside the sandbox
2. `socat` bridges Unix sockets from inside the sandbox to the host
3. HTTP/SOCKS proxy servers run on the host, filtering by domain
4. `HTTP_PROXY`, `HTTPS_PROXY` env vars point to the proxy inside the sandbox
5. `GIT_SSH_COMMAND` is configured to use the proxy for git-over-SSH

This enables per-domain network access control, unlike the CFOAgent approach of `--share-net` which allows all network traffic.

### Configuration Format (`~/.srt-settings.json`)

```json
{
  "network": {
    "allowedDomains": ["api.anthropic.com", "github.com", "*.github.com"],
    "deniedDomains": ["malicious.com"],
    "allowUnixSockets": [],
    "allowLocalBinding": false
  },
  "filesystem": {
    "denyRead": ["~/.ssh", "~/.aws", "~/.gnupg"],
    "allowWrite": [".", "/tmp"],
    "denyWrite": [".env", "secrets/"]
  }
}
```

### Key Design Differences from CFOAgent

| Aspect | CFOAgent | Anthropic srt |
|--------|----------|---------------|
| Network | `--share-net` (full access) | `--unshare-net` + proxy filtering |
| Domain control | None | Per-domain allow/deny |
| Filesystem | Static bwrap args | Dynamic from config |
| SSH keys | `--ro-bind ~/.ssh` | Denied by default |
| Platform | Linux only | Linux + macOS (Seatbelt) |

---

## 5. Nix Flake Patterns for Sandboxed Dev Shells

### Pattern 1: Inline Script (CFOAgent approach)

The simplest approach -- define the bwrap script inline in `flake.nix`:

```nix
{
  devShells.default = let
    sandboxScript = pkgs.writeShellScriptBin "enter-sandbox" ''
      BWRAP_ARGS=(
        --unshare-all --share-net --die-with-parent
        --ro-bind / /
        --dev /dev --proc /proc
        --tmpfs "$HOME"
        --bind "$PROJECT_DIR" "$PROJECT_DIR"
        --bind "$HOME/.claude" "$HOME/.claude"
        # ... more mounts
      )
      exec ${pkgs.bubblewrap}/bin/bwrap "''${BWRAP_ARGS[@]}" "$@"
    '';
  in pkgs.mkShell {
    buildInputs = [ pkgs.bubblewrap sandboxScript ];
    shellHook = ''
      exec enter-sandbox bash
    '';
  };
}
```

**Pros**: Self-contained, easy to understand.
**Cons**: Verbose, hard to reuse across projects.

### Pattern 2: bubblewrap-claude (Nix library)

The [bubblewrap-claude](https://github.com/matgawin/bubblewrap-claude) project provides a reusable Nix flake with profile-based sandboxing:

```nix
{
  inputs.bubblewrap-claude.url = "github:matgawin/bubblewrap-claude";

  outputs = { self, nixpkgs, bubblewrap-claude, ... }:
    let
      bwLib = bubblewrap-claude.lib.${system};
    in {
      packages.default = bwLib.mkSandbox {
        name = "my-sandbox";
        packages = with pkgs; [ go git ];
        allowList = [ "api.anthropic.com" "github.com" ];
        customPrompt = "You are a Go developer...";
      };

      devShells.default = bwLib.mkDevShell {
        packages = with pkgs; [ go git ];
      };
    };
}
```

**Pros**: Pre-built profiles (Go, Python, Rust, JS), network proxy filtering, telemetry disabled.
**Cons**: Opinionated, may not fit all workflows.

### Pattern 3: jail.nix (Declarative combinators)

[jail.nix](https://git.sr.ht/~alexdavid/jail.nix) provides high-level Nix combinators for bubblewrap:

```nix
{
  inputs = {
    jail-nix.url = "sourcehut:~alexdavid/jail.nix";
  };

  outputs = { self, nixpkgs, jail-nix, ... }:
    let
      jailLib = jail-nix.lib.${system};
      jailedClaude = jailLib.jail pkgs.claude-code [
        jailLib.mount-cwd           # Only access current directory
        (jailLib.add-pkg-deps [     # Explicit tool allowlist
          pkgs.git pkgs.curl pkgs.ripgrep
        ])
      ];
    in {
      devShells.default = pkgs.mkShell {
        packages = [ jailedClaude ];
      };
    };
}
```

**Pros**: Declarative, composable, understands Nix closures automatically.
**Cons**: Newer project, less documentation.

### Pattern 4: nixwrap (Simple wrapper)

[nixwrap](https://github.com/rti/nixwrap) wraps individual packages with sandbox restrictions:

```nix
{
  inputs.wrap.url = "github:rti/nixwrap";

  outputs = { self, nixpkgs, wrap, ... }: {
    devShells.default = pkgs.mkShell {
      buildInputs = [
        (wrap.lib.wrap pkgs.nodejs {
          network = true;
          cwd = true;
        })
      ];
    };
  };
}
```

**Pros**: Minimal, wraps individual tools.
**Cons**: Per-package rather than whole-environment sandboxing.

### Environment Variable Passthrough

A critical concern in all approaches -- environment variables must be explicitly passed through the sandbox:

```bash
# Required for Claude Code
--setenv HOME "$HOME"
--setenv PATH "$PATH"
--setenv TERM "$TERM"
--setenv ANTHROPIC_API_KEY "$ANTHROPIC_API_KEY"
--setenv CLAUDE_CONFIG_DIR "$CLAUDE_CONFIG_DIR"
--setenv GH_TOKEN "$GH_TOKEN"

# For Nix-built tools
--setenv LD_LIBRARY_PATH "$LD_LIBRARY_PATH"
--setenv NIX_PATH "$NIX_PATH"
--setenv NIX_PROFILES "$NIX_PROFILES"
```

Using `--clearenv` and explicitly setting only needed variables is more secure but requires careful enumeration.

### Nix Store Interaction

The `/nix/store` must be accessible read-only inside the sandbox. With `--ro-bind / /`, this happens automatically. If building a minimal sandbox (not binding all of `/`), you need to either:

1. Bind `/nix/store` explicitly: `--ro-bind /nix/store /nix/store`
2. Use `nix-bubblewrap` which automatically resolves and binds only the required store path closures
3. Use `writeReferencesToFile` or `closureInfo` from nixpkgs to compute needed paths at build time

---

## 6. Security Considerations for AI Agents

### What to Hide from the Agent

| Asset | Risk | Mitigation |
|-------|------|------------|
| `~/.ssh/` (private keys) | Key theft, unauthorized access | `--tmpfs ~/.ssh` or omit entirely |
| `~/.aws/`, `~/.gcloud/` | Cloud credential theft | Omit from sandbox |
| `~/.gnupg/` | GPG key theft | Omit from sandbox |
| Other project dirs | IP leakage, cross-contamination | `--tmpfs $HOME` hides all |
| `~/.bash_history` | Command history leakage | Hidden by `--tmpfs $HOME` |
| `/etc/shadow`, `/etc/passwd` | System credential access | Read-only via `--ro-bind / /` |
| `.env` files in project | Secret leakage | `--ro-bind /dev/null $PROJECT/.env` |

### Network Restrictions

Full network access (`--share-net`) is convenient but dangerous. A compromised agent can:
- Exfiltrate any readable file (including SSH keys if mounted)
- Download malicious payloads
- Contact C2 servers
- Make unauthorized API calls

**Recommendation**: Use the proxy-based approach from Anthropic's sandbox-runtime:
1. `--unshare-net` to block all network
2. Proxy via socat + Unix socket for filtered access
3. Allowlist only required domains:
   - `api.anthropic.com` (Claude API)
   - `github.com`, `*.github.com` (git operations)
   - `*.githubusercontent.com` (GitHub raw content)
   - `proxy.golang.org`, `sum.golang.org` (Go modules, for Wave)
   - Package registries as needed

### Git Operations Inside the Sandbox

Git needs several things to work properly:

```bash
# Required for git
--ro-bind "$HOME/.gitconfig" "$HOME/.gitconfig"    # Git config
--ro-bind /etc/ssl/certs /etc/ssl/certs            # TLS certs (via --ro-bind / /)

# For SSH-based git (optional, increases attack surface)
--ro-bind "$HOME/.ssh" "$HOME/.ssh"                # SSH keys
--setenv SSH_AUTH_SOCK "$SSH_AUTH_SOCK"             # SSH agent (if used)

# For GPG-signed commits (optional)
--ro-bind "$HOME/.gnupg" "$HOME/.gnupg"
--setenv GPG_TTY "$GPG_TTY"
```

**Safer alternative**: Use HTTPS-based git with `GH_TOKEN`:
```bash
--setenv GH_TOKEN "$GH_TOKEN"
# Git will use https:// URLs and token auth, no SSH keys needed
```

### Preventing System Config Modification

The `--ro-bind / /` + `--tmpfs $HOME` pattern is effective:
- System files (`/etc`, `/usr`, `/bin`) are read-only
- Home directory files (`.bashrc`, `.profile`) are hidden/inaccessible
- Only explicitly bound paths are writable

### Defense in Depth

Anthropic recommends layering:
1. **Permissions** (first gate): Claude's allow/ask/deny rules control which tools run
2. **Sandboxing** (second gate): OS-level enforcement of what those tools can access
3. **Network filtering** (third gate): Domain-level control of outbound connections

### Zero-Trust Principle

The OWASP AI Agent Security Top 10 (2026) recommends treating all AI-generated code as potentially malicious. The sandbox should operate on zero-trust principles where all actions are explicitly allowed rather than implicitly permitted.

---

## 7. Limitations and Gotchas

### macOS: Bubblewrap Does Not Work

Bubblewrap requires Linux kernel namespaces and does **not** work on macOS. Alternatives for macOS:
- **Seatbelt** (`sandbox-exec`): macOS-native sandboxing, used by Anthropic's srt
- **Docker Desktop**: Container-based isolation (heavier weight)
- The sandbox-runtime npm package handles both platforms transparently

### Nix Store Path Visibility

With `--ro-bind / /`, the entire Nix store is visible read-only, which is fine. But if you use a minimal sandbox (binding specific paths), you must ensure all transitive dependencies in `/nix/store` are available. The `nix-bubblewrap` project automates this using closure info.

### TMPDIR Gotcha on NixOS

When bubblewrap is installed setuid root (common on NixOS), glibc automatically filters `TMPDIR` from the environment via `UNSECURE_ENVVARS`. This means:
- `TMPDIR` will be unset inside the sandbox
- Fix: explicitly pass `--setenv TMPDIR /tmp`

### Unprivileged User Namespaces

On Ubuntu 24.04+, unprivileged user namespaces are restricted by default. bubblewrap may need the setuid bit. On NixOS, this is typically handled by the `bubblewrap` package.

### Docker Inside Sandbox

Docker cannot run inside a bubblewrap sandbox. Use `excludedCommands` in Claude Code's sandbox settings or mount the Docker socket (which weakens isolation significantly).

### Wayland / X11

Bubblewrap can forward Wayland/X11 access for graphical apps, but this is irrelevant for Claude Code (CLI-only).

### `--new-session` Flag

Without `--new-session`, a process inside the sandbox can use ANSI escape sequences or terminal ioctls to inject keystrokes into the parent terminal. Always include this flag for security.

### Race Condition with Mount Tables

A known bwrap issue: `--bind` can fail if the host mount table changes concurrently (e.g., USB mount/unmount during sandbox creation). This is rare but worth noting.

### `enableWeakerNestedSandbox`

Anthropic's sandbox-runtime includes this flag for running inside Docker (where user namespaces may not be available). It significantly weakens security and should only be used when outer isolation is already enforced.

### Interactive Terminal Detection

The CFOAgent example correctly checks `[ ! -t 0 ]` to avoid entering the sandbox for non-interactive commands (`nix develop --command ...`). This is important for CI/CD compatibility.

---

## 8. Recommendations for Wave

### Recommended Architecture

For Wave's `flake.nix`, I recommend a hybrid approach combining the CFOAgent pattern with network filtering:

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
        pkgs = import nixpkgs { inherit system; config.allowUnfree = true; };

        commonPackages = with pkgs; [
          go gh git jq curl sqlite
          bubblewrap socat  # For sandboxing
        ];

        sandboxScript = pkgs.writeShellScriptBin "wave-sandbox" ''
          PROJECT_DIR="''${WAVE_PROJECT_DIR:-$PWD}"

          BWRAP_ARGS=(
            --unshare-all
            --share-net          # TODO: Replace with proxy-based filtering
            --die-with-parent
            --new-session        # Prevent terminal escape attacks

            # Root filesystem - READ-ONLY
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

            # Writable: temp files
            --tmpfs /tmp

            # Read-only: git config
            --ro-bind "$HOME/.gitconfig" "$HOME/.gitconfig"

            # Read-only: SSH keys (for git operations)
            # NOTE: Consider using GH_TOKEN + HTTPS instead
            --ro-bind "$HOME/.ssh" "$HOME/.ssh"

            # Environment
            --setenv HOME "$HOME"
            --setenv PATH "$PATH"
            --setenv TERM "''${TERM:-xterm-256color}"
            --setenv TMPDIR "/tmp"
            --setenv SANDBOX_ACTIVE "1"

            # Pass through API keys
            --setenv ANTHROPIC_API_KEY "''${ANTHROPIC_API_KEY:-}"
            --setenv GH_TOKEN "''${GH_TOKEN:-}"

            --chdir "$PROJECT_DIR"
          )

          # Ensure bind targets exist
          mkdir -p "$HOME/.claude"
          touch "$HOME/.claude.json"

          if [ $# -gt 0 ]; then
            exec ${pkgs.bubblewrap}/bin/bwrap "''${BWRAP_ARGS[@]}" "$@"
          else
            exec ${pkgs.bubblewrap}/bin/bwrap "''${BWRAP_ARGS[@]}" \
              ${pkgs.bash}/bin/bash
          fi
        '';
      in {
        devShells = {
          default = pkgs.mkShell {
            buildInputs = commonPackages ++ [ sandboxScript ];
            shellHook = ''
              export WAVE_PROJECT_DIR="$PWD"

              if command -v gh &>/dev/null && gh auth status &>/dev/null 2>&1; then
                export GH_TOKEN=$(gh auth token 2>/dev/null)
              fi

              if [ -t 0 ] && [ -z "$SANDBOX_ACTIVE" ]; then
                echo ""
                echo "  WAVE Sandboxed Environment"
                echo "  WRITE: $PWD, ~/.claude, /tmp"
                echo "  READ:  everything else"
                echo ""
                exec wave-sandbox bash
              fi
            '';
          };

          # Unsandboxed shell for when you need full access
          unsandboxed = pkgs.mkShell {
            buildInputs = commonPackages;
            shellHook = ''
              echo "WAVE Development (NO SANDBOX)"
            '';
          };
        };
      }
    );
}
```

### Phase 1: Basic Filesystem Isolation (Start Here)

- `--ro-bind / /` + `--tmpfs $HOME` + selective writable mounts
- `--share-net` for simplicity (full network access)
- This matches what CFOAgent does today
- Protects against: filesystem damage, config modification, cross-project contamination

### Phase 2: Network Filtering (Future Enhancement)

- Replace `--share-net` with `--unshare-net` + socat proxy
- Domain allowlist: `api.anthropic.com`, `github.com`, `proxy.golang.org`
- Use Anthropic's `@anthropic-ai/sandbox-runtime` npm package or reimplement the proxy pattern
- Protects against: data exfiltration, unauthorized API calls

### Phase 3: Per-Persona Sandboxing (Wave-Specific)

Wave already has persona-level permission enforcement. This could be extended to configure different bwrap profiles per persona:
- **Navigator** (read-only analysis): No write access, no network
- **Implementer** (code changes): Project dir writable, network for package managers
- **Reviewer** (code review): Read-only project, no network

### What NOT to Do

1. **Do not use `--dangerously-skip-permissions` without a sandbox** -- this gives Claude Code unrestricted shell access
2. **Do not mount `~/.ssh` writable** -- agent could modify SSH config or keys
3. **Do not share `/tmp` with host** -- use `--tmpfs /tmp` for isolation
4. **Do not skip `--new-session`** -- terminal escape attacks are real
5. **Do not rely solely on Claude Code's built-in sandbox** -- it can be escaped via the `dangerouslyDisableSandbox` parameter (unless `allowUnsandboxedCommands: false` is set)

---

## 9. Sources

### Anthropic Official
- [Claude Code Sandboxing Documentation](https://code.claude.com/docs/en/sandboxing)
- [Making Claude Code More Secure and Autonomous](https://www.anthropic.com/engineering/claude-code-sandboxing)
- [Anthropic Sandbox Runtime (GitHub)](https://github.com/anthropic-experimental/sandbox-runtime)
- [Sandbox Runtime README](https://github.com/anthropic-experimental/sandbox-runtime/blob/main/README.md)
- [Claude Code Security Documentation](https://code.claude.com/docs/en/security)
- [Claude Code Settings Documentation](https://code.claude.com/docs/en/settings)

### Community Projects
- [bubblewrap-claude -- Nix flake for sandboxed Claude Code](https://github.com/matgawin/bubblewrap-claude)
- [blaude -- Claude Code in a bubblewrap sandbox](https://github.com/c0ffee0wl/blaude)
- [cco -- Thin protective layer for Claude Code](https://github.com/nikvdp/cco)
- [jail.nix -- Nix library for bubblewrap jails](https://git.sr.ht/~alexdavid/jail.nix)
- [nixwrap -- Easy application sandboxing on NixOS](https://github.com/rti/nixwrap)
- [nix-bwrapper -- User-friendly bubblewrap with portals](https://github.com/Naxdy/nix-bwrapper)
- [nix-bubblewrap -- Nix/bubblewrap integration](https://github.com/fgaz/nix-bubblewrap)
- [nix-sandbox-mcp -- Sandboxed code execution for LLMs](https://github.com/SecBear/nix-sandbox-mcp)

### Bubblewrap Reference
- [Bubblewrap GitHub Repository](https://github.com/containers/bubblewrap)
- [Bubblewrap -- Arch Wiki](https://wiki.archlinux.org/title/Bubblewrap)
- [Bubblewrap Examples -- Arch Wiki](https://wiki.archlinux.org/title/Bubblewrap/Examples)
- [bwrap(1) Man Page](https://www.mankier.com/1/bwrap)
- [Notes on Running Containers with Bubblewrap (Julia Evans)](https://jvns.ca/blog/2022/06/28/some-notes-on-bubblewrap/)
- [Sandboxing Applications with Bubblewrap](https://sloonz.github.io/posts/sandboxing-1/)

### Blog Posts and Analysis
- [A Better Way to Limit Claude Code Access to Secrets](https://patrickmccanna.net/a-better-way-to-limit-claude-code-and-other-coding-agents-access-to-secrets/)
- [How I Run LLM Agents in a Secure Nix Sandbox](https://dev.to/andersonjoseph/how-i-run-llm-agents-in-a-secure-nix-sandbox-1899)
- [Trying Out Bubblewrap in Claude Code's Sandbox Runtime](https://www.sambaiz.net/en/article/547/)
- [Claude Code Sandbox Guide (2026)](https://claudefa.st/blog/guide/sandboxing-guide)
- [Claude Code Config File Locations](https://inventivehq.com/knowledge-base/claude/where-configuration-files-are-stored)

### Security Standards
- [How to Sandbox AI Agents in 2026 (Northflank)](https://northflank.com/blog/how-to-sandbox-ai-agents)
- [NVIDIA: Practical Security Guidance for Sandboxing Agentic Workflows](https://developer.nvidia.com/blog/practical-security-guidance-for-sandboxing-agentic-workflows-and-managing-execution-risk)
- [OWASP AI Agent Security Top 10 (2026)](https://medium.com/@oracle_43885/owasps-ai-agent-security-top-10-agent-security-risks-2026-fc5c435e86eb)
- [Claude Code Security Best Practices (Backslash)](https://www.backslash.security/blog/claude-code-security-best-practices)

### Nix Community
- [Claude Code and Security Isolation -- NixOS Discourse](https://discourse.nixos.org/t/claude-code-and-security-isolation/71543)
- [Restricting /nix/store in a Mount Namespace -- NixOS Discourse](https://discourse.nixos.org/t/restricting-nix-store-in-a-mount-namespace/21185)
- [Development Shells with Nix (2025) -- Michael Stapelberg](https://michael.stapelberg.ch/posts/2025-07-27-dev-shells-with-nix-4-quick-examples/)
- [jail.nix NixCon 2025 Talk](https://talks.nixcon.org/nixcon-2025/talk/3QH3PZ/)

### GitHub Issues
- [CLAUDE_CONFIG_DIR Behavior Unclear (Issue #3833)](https://github.com/anthropics/claude-code/issues/3833)
- [TMPDIR Not Set in Sandbox with Setuid bwrap (Issue #10952)](https://github.com/anthropics/claude-code/issues/10952)
- [Allow Sandboxing Development Shells (NixOS/nix #11211)](https://github.com/NixOS/nix/issues/11211)
