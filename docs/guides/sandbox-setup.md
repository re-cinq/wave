# Sandbox Setup

Wave provides defense-in-depth isolation for AI agent sessions through two complementary layers:

1. **Outer sandbox** (Nix + bubblewrap) — isolates the entire development session at the OS level
2. **Adapter sandbox** (manifest-driven) — projects persona permissions into Claude Code's `settings.json` and `CLAUDE.md`

## Quick Start

```bash
# Install Nix if you haven't already
# https://nixos.org/download.html

# Enter sandboxed development shell
nix develop

# Everything you run inside is sandboxed:
wave run speckit-flow "add user authentication"
```

## Nix Dev Shell (Outer Sandbox)

The `flake.nix` at the project root defines two dev shells:

### Default Shell (Sandboxed)

```bash
nix develop
```

On Linux, this automatically enters a bubblewrap sandbox with:

| Protection | Detail |
|------------|--------|
| Read-only filesystem | `--ro-bind / /` — entire root is mounted read-only |
| Hidden home directory | `--tmpfs $HOME` — `~/.aws`, `~/.gnupg`, etc. are invisible |
| Selective home read-only | `~/.ssh`, `~/.gitconfig`, `~/.config/gh`, `~/.npmrc` mounted read-only |
| Writable project dir | `--bind $PROJECT_DIR` — the project directory is writable |
| Writable Go cache | `--bind ~/go` — Go module cache persists across steps |
| Writable Wave binary | `--bind ~/.local/bin/wave` — build target for `go build` |
| Writable Claude config | `--bind ~/.claude` — Claude Code session state persists |
| Shared temp | `--bind /tmp` — shared with host (Nix tooling needs it) |
| Inherited environment | Nix-provided environment inherited (no `--clearenv`) |
| Process isolation | `--die-with-parent` ensures sandbox dies with the shell |

On macOS, bwrap requires kernel namespaces not available on Darwin. The default shell runs unsandboxed — Claude Code's built-in Seatbelt sandbox provides OS-level isolation for Bash commands.

### Yolo Shell (No Sandbox)

```bash
nix develop .#yolo
```

Use this when you need Docker (can't run inside bwrap), or for debugging sandbox-related issues.

### What the Sandbox Protects Against

| Attack Vector | Protected? | Mechanism |
|---------------|-----------|-----------|
| Write outside project | Yes | `--ro-bind /` makes filesystem read-only |
| Access `~/.aws` credentials | Yes | `--tmpfs $HOME` hides home directory |
| Access `~/.gnupg` keys | Yes | `--tmpfs $HOME` hides home directory |
| Modify `~/.ssh` keys | Partial | `--ro-bind-try` mounts read-only (readable for git) |
| Modify git config | Partial | `--ro-bind-try` mounts read-only |
| Process escape | Yes | `--unshare-all` + `--die-with-parent` |

### What the Sandbox Allows

The sandbox is intentionally permissive for things AI agents need:

| Access | Why |
|--------|-----|
| `~/.ssh` (read-only) | Git push/pull over SSH |
| `~/.gitconfig` (read-only) | Git commit identity |
| `~/.config/gh` (read-only) | GitHub CLI authentication |
| `~/.local/bin/wave` (read-write) | Build target for `go build` / `make install` |
| `~/.local/bin/notesium` (read-only) | Local tooling |
| `~/.local/bin/claudit` (read-only) | Local tooling |
| `~/go` (read-write) | Go module cache avoids re-downloads |
| `~/.claude` (read-write) | Claude Code session state and OAuth |
| `/tmp` (shared) | Nix store paths and tooling |
| Full network | API calls, package downloads, git remotes |

### Passed Environment Variables

The sandbox inherits the Nix-provided environment (no `--clearenv`), which includes:

- All Nix dev shell variables (`PATH`, `GOPATH`, etc.)
- `SANDBOX_ACTIVE=1` (set inside sandbox)
- `ANTHROPIC_API_KEY` (if set before `nix develop`)
- `GH_TOKEN` (if set, or derived from `gh auth token`)

## Manifest-Driven Adapter Config

The second layer projects manifest permissions into Claude Code's configuration files.

### Persona Sandbox Configuration

```yaml
personas:
  navigator:
    adapter: claude
    permissions:
      allowed_tools: [Read, Glob, Grep, "Bash(git log*)"]
      deny: ["Write(*)", "Edit(*)"]
    sandbox:
      allowed_domains:
        - api.anthropic.com

  implementer:
    adapter: claude
    permissions:
      allowed_tools: [Read, Write, Edit, Bash, Glob, Grep]
    sandbox:
      allowed_domains:
        - api.anthropic.com
        - github.com
        - "*.github.com"
        - proxy.golang.org
        - sum.golang.org
```

### Runtime Sandbox Configuration

```yaml
runtime:
  sandbox:
    enabled: true
    default_allowed_domains:
      - api.anthropic.com
      - github.com
    env_passthrough:
      - ANTHROPIC_API_KEY
      - GH_TOKEN
```

| Field | Description |
|-------|-------------|
| `enabled` | Enable sandbox configuration generation |
| `default_allowed_domains` | Default network domains for all personas (persona config overrides) |
| `env_passthrough` | Environment variables to pass to adapter subprocesses |

### How Permissions Flow

```
wave.yaml                    →  settings.json                →  CLAUDE.md
─────────────────────────────────────────────────────────────────────────
persona.permissions.allowed  →  permissions.allow            →  "Allowed Tools" section
persona.permissions.deny     →  permissions.deny             →  "Denied Tools" section
persona.sandbox.allowed_domains → sandbox.network.allowedDomains → "Network Access" section
runtime.sandbox.env_passthrough → curated subprocess env     →  (not in CLAUDE.md)
```

Claude Code reads both `settings.json` (enforced) and `CLAUDE.md` (model awareness), providing defense-in-depth.

### Environment Hygiene

The adapter constructs a curated environment instead of passing the full host environment:

**Always included:**
- `HOME`, `PATH`, `TERM`, `TMPDIR=/tmp`
- `DISABLE_TELEMETRY=1`, `DISABLE_ERROR_REPORTING=1`
- `CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY=1`, `DISABLE_BUG_COMMAND=1`

**Conditionally included:**
- Variables listed in `runtime.sandbox.env_passthrough` (if set in host env)
- Step-specific env vars from pipeline configuration

**Never included:**
- Any other host environment variable (e.g., `AWS_SECRET_ACCESS_KEY`, `DATABASE_PASSWORD`)

**Note:** The curated environment model applies to the Claude adapter. Other adapters using the `ProcessGroupRunner` currently inherit the full host environment.

## Defense-in-Depth Summary

From innermost to outermost:

| Layer | What It Does | Enforced By |
|-------|-------------|-------------|
| `CLAUDE.md` restrictions | Model is told what it can/cannot do | Prompt-level (advisory) |
| `settings.json` permissions | Allow/deny rules enforced by Claude Code | Claude Code runtime |
| Claude Code sandbox | Bubblewrap/Seatbelt restricts Bash commands | OS-level (Bash only) |
| Nix dev shell sandbox | Bubblewrap restricts the entire session | OS-level (everything) |
| Manifest | Single source of truth for all above | Wave configuration |

## Troubleshooting

### "bwrap: No permissions to create new namespace"

Your kernel may not support unprivileged user namespaces. Check:

```bash
cat /proc/sys/kernel/unprivileged_userns_clone
# Should be 1
```

If 0, enable it (requires root):

```bash
sudo sysctl -w kernel.unprivileged_userns_clone=1
```

Or use the yolo shell: `nix develop .#yolo`

### "command not found" inside sandbox

The sandbox inherits the Nix-provided `PATH`. If a tool isn't found, add the package to `commonPackages` in `flake.nix`.

### Docker doesn't work inside sandbox

Docker cannot run inside bubblewrap (it needs its own namespaces). Use `nix develop .#yolo` for steps that require Docker.
