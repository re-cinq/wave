# 027: Sandbox Claude Code Sessions

## Problem Statement

Wave executes Claude Code as a subprocess with `--dangerously-skip-permissions` and relies on `--allowedTools` CLI flags for access control. The developer opens a terminal, runs `wave run`, and Claude Code gets full access to the host filesystem and network.

Two problems exist:

1. **No outer isolation**: The development environment itself is unsandboxed. Everything Wave spawns inherits the developer's full filesystem and network access.

2. **Manifest permissions not fully projected**: Wave's manifest defines per-persona `allowed_tools` and `deny` patterns, but the adapter only writes `Allow` to `settings.json` — the `Deny` list is silently dropped (`ClaudePermissions` has no `Deny` field). The generated `CLAUDE.md` contains only the persona's system prompt, with no mention of the restrictions the manifest defines.

### Current Flow (what happens today)

```
Developer terminal (full access)
  └─ wave run <pipeline>
       └─ executor.go reads manifest personas
            └─ passes AllowedTools + DenyTools to AdapterRunConfig
                 └─ claude.go prepareWorkspace:
                      ├─ settings.json: writes Allow only, Deny DROPPED
                      ├─ CLAUDE.md: persona system prompt only, no restrictions
                      └─ buildArgs: --dangerously-skip-permissions --allowedTools
                           └─ claude subprocess (full host access)
```

### Target Flow

```
nix develop  (enters bubblewrap sandbox)
  └─ wave run <pipeline>
       └─ executor.go reads manifest personas + sandbox config
            └─ passes AllowedTools + DenyTools + sandbox settings to AdapterRunConfig
                 └─ claude.go prepareWorkspace:
                      ├─ settings.json: Allow + Deny + sandbox + network domains
                      ├─ CLAUDE.md: persona prompt + restriction section from manifest
                      └─ buildArgs: --allowedTools (sandbox handles permissions)
                           └─ claude subprocess (sandboxed by outer bwrap + inner Claude sandbox)
```

## Goals

1. Provide a Nix flake dev shell that sandboxes the entire Wave development session via bubblewrap
2. Fix the adapter to write `Deny` rules from the manifest into `settings.json`
3. Generate `CLAUDE.md` that includes restriction directives derived from the manifest
4. Enable Claude Code's built-in sandbox with settings driven by the manifest
5. Curate environment variables instead of passing full `os.Environ()`

## Non-Goals

1. Docker/microVM per-step isolation (future work)
2. macOS Seatbelt integration (Claude Code handles this natively)
3. Custom seccomp profiles
4. OverlayFS workspace replacement (orthogonal optimization)

---

## Design

### Phase 1: Nix Flake Dev Shell Sandbox

**The outer container.** You `nix develop` into a bubblewrap sandbox, then everything — Wave, Claude Code, git — runs inside it.

#### 1.1 Bubblewrap Sandbox Script

```nix
sandboxScript = pkgs.writeShellScriptBin "wave-sandbox" ''
  PROJECT_DIR="''${WAVE_PROJECT_DIR:-$PWD}"

  # Ensure bind targets exist before bwrap
  mkdir -p "$HOME/.claude"
  touch -a "$HOME/.claude.json"

  BWRAP_ARGS=(
    --unshare-all
    --share-net          # Full net for now; Phase 1a adds proxy filtering
    --die-with-parent
    --new-session        # Prevent terminal escape sequence attacks

    # Root filesystem — READ-ONLY
    --ro-bind / /
    --dev /dev
    --proc /proc

    # Hide entire home directory
    --tmpfs "$HOME"

    # Writable: project directory only
    --bind "$PROJECT_DIR" "$PROJECT_DIR"

    # Writable: Claude Code config (session state, credentials)
    --bind "$HOME/.claude" "$HOME/.claude"
    --bind "$HOME/.claude.json" "$HOME/.claude.json"

    # Writable: isolated temp (NOT shared with host)
    --tmpfs /tmp

    # Read-only: git config for commits
    --ro-bind "$HOME/.gitconfig" "$HOME/.gitconfig"

    # Environment — curated, not inherited
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

#### 1.2 Flake.nix

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
          bubblewrap socat
          claude-code
        ];

        sandboxScript = /* as above */;
      in {
        devShells = {
          # Default: sandboxed
          default = pkgs.mkShell {
            buildInputs = commonPackages ++ [ sandboxScript ];
            shellHook = ''
              export WAVE_PROJECT_DIR="$PWD"

              if command -v gh &>/dev/null && gh auth status &>/dev/null 2>&1; then
                export GH_TOKEN=$(gh auth token 2>/dev/null)
              fi

              if [ -t 0 ] && [ -z "$SANDBOX_ACTIVE" ]; then
                echo ""
                echo "  WAVE Sandboxed Development Shell"
                echo "  WRITE: $PWD, ~/.claude, /tmp"
                echo "  READ:  / (read-only)"
                echo ""
                exec wave-sandbox bash
              fi
            '';
          };

          # Escape hatch
          yolo = pkgs.mkShell {
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

#### 1.3 What This Protects

- Agent cannot read `~/.ssh`, `~/.aws`, `~/.gnupg` or any other home directory contents
- Agent cannot write outside the project directory
- Agent cannot modify system config (`/etc`, `/usr`, `/bin`)
- Host `/tmp` is isolated — no cross-process snooping
- Environment is curated — no credential leakage from host env vars
- `--new-session` blocks terminal escape injection

#### 1.4 Platform Notes

- **Linux only**: Bubblewrap requires kernel namespaces. macOS users use `nix develop .#yolo` and rely on Claude Code's built-in Seatbelt sandbox.
- **NixOS TMPDIR**: Setuid bwrap strips `TMPDIR` via glibc. The `--setenv TMPDIR /tmp` handles this.
- **Docker**: Cannot run inside bwrap. Steps needing Docker must use the yolo shell.

---

### Phase 2: Manifest-Driven Claude Code Configuration

**Fix the adapter to fully project manifest settings into Claude Code's configuration files.**

#### 2.1 Fix: Write Deny Rules to settings.json

Bug: `ClaudePermissions` only has `Allow`. The executor passes `DenyTools` (from `persona.Permissions.Deny`) but `prepareWorkspace` drops it.

```go
// Current (broken):
type ClaudePermissions struct {
    Allow []string `json:"allow"`
}

// Fixed:
type ClaudePermissions struct {
    Allow []string `json:"allow"`
    Deny  []string `json:"deny,omitempty"`
}
```

In `prepareWorkspace`:

```go
settings := ClaudeSettings{
    Model:        model,
    Temperature:  cfg.Temperature,
    OutputFormat: "stream-json",
    Permissions: ClaudePermissions{
        Allow: normalizeAllowedTools(allowedTools),
        Deny:  cfg.DenyTools,  // Currently missing
    },
}
```

#### 2.2 Add Sandbox Settings to settings.json

Extend `ClaudeSettings` with sandbox configuration derived from the manifest:

```go
type ClaudeSettings struct {
    Model        string            `json:"model"`
    Temperature  float64           `json:"temperature"`
    OutputFormat string            `json:"output_format"`
    Permissions  ClaudePermissions `json:"permissions"`
    Sandbox      *SandboxSettings  `json:"sandbox,omitempty"`
}

type SandboxSettings struct {
    Enabled                  bool            `json:"enabled"`
    AllowUnsandboxedCommands bool            `json:"allowUnsandboxedCommands"`
    AutoAllowBashIfSandboxed bool            `json:"autoAllowBashIfSandboxed"`
    Network                  *NetworkSettings `json:"network,omitempty"`
}

type NetworkSettings struct {
    AllowedDomains []string `json:"allowedDomains,omitempty"`
}
```

The manifest drives these values. Example generated `settings.json` for an `implementer` step:

```json
{
  "model": "opus",
  "permissions": {
    "allow": ["Read", "Write", "Edit", "Bash", "Glob", "Grep"],
    "deny": ["Bash(rm -rf /*)"]
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
        "proxy.golang.org",
        "sum.golang.org"
      ]
    }
  }
}
```

For a `navigator` step (read-only):

```json
{
  "model": "opus",
  "permissions": {
    "allow": ["Read", "Glob", "Grep", "Bash(git log*)"],
    "deny": ["Write(*)", "Edit(*)"]
  },
  "sandbox": {
    "enabled": true,
    "allowUnsandboxedCommands": false,
    "autoAllowBashIfSandboxed": true,
    "network": {
      "allowedDomains": ["api.anthropic.com"]
    }
  }
}
```

#### 2.3 Generate CLAUDE.md with Restriction Directives

Currently `prepareWorkspace` writes only the persona's system prompt to `CLAUDE.md`. It should append a restriction section derived from the manifest:

```go
func (a *ClaudeAdapter) prepareWorkspace(workspacePath string, cfg AdapterRunConfig) error {
    // ... existing settings.json generation ...

    // Build CLAUDE.md: persona prompt + manifest-derived restrictions
    var claudeMd strings.Builder

    // 1. Persona system prompt (existing behavior)
    if cfg.SystemPrompt != "" {
        claudeMd.WriteString(cfg.SystemPrompt)
    } else if data, err := os.ReadFile(personaPath); err == nil {
        claudeMd.Write(data)
    } else {
        fmt.Fprintf(&claudeMd, "# %s\n\nYou are operating as the %s persona.\n", cfg.Persona, cfg.Persona)
    }

    // 2. Restriction section from manifest
    claudeMd.WriteString("\n\n---\n\n## Restrictions\n\n")
    claudeMd.WriteString("The following restrictions are enforced by the pipeline orchestrator.\n\n")

    if len(cfg.DenyTools) > 0 {
        claudeMd.WriteString("### Denied Tools\n\n")
        for _, deny := range cfg.DenyTools {
            fmt.Fprintf(&claudeMd, "- `%s`\n", deny)
        }
        claudeMd.WriteString("\n")
    }

    if len(cfg.AllowedTools) > 0 {
        claudeMd.WriteString("### Allowed Tools\n\n")
        claudeMd.WriteString("You may ONLY use the following tools:\n\n")
        for _, tool := range cfg.AllowedTools {
            fmt.Fprintf(&claudeMd, "- `%s`\n", tool)
        }
        claudeMd.WriteString("\n")
    }

    if len(cfg.AllowedDomains) > 0 {
        claudeMd.WriteString("### Network Access\n\n")
        claudeMd.WriteString("Network requests are restricted to:\n\n")
        for _, domain := range cfg.AllowedDomains {
            fmt.Fprintf(&claudeMd, "- `%s`\n", domain)
        }
        claudeMd.WriteString("\n")
    }

    // Write CLAUDE.md
    return os.WriteFile(claudeMdPath, []byte(claudeMd.String()), 0644)
}
```

This tells Claude Code at both the configuration level (settings.json enforces it) and the prompt level (CLAUDE.md makes the model aware of it) what it can and cannot do.

#### 2.4 Manifest Schema Extension

Add optional `sandbox` config to personas and runtime:

```go
type Persona struct {
    Adapter          string           `yaml:"adapter"`
    Description      string           `yaml:"description,omitempty"`
    SystemPromptFile string           `yaml:"system_prompt_file"`
    Temperature      float64          `yaml:"temperature,omitempty"`
    Model            string           `yaml:"model,omitempty"`
    Permissions      Permissions      `yaml:"permissions,omitempty"`
    Hooks            HookConfig       `yaml:"hooks,omitempty"`
    Sandbox          *PersonaSandbox  `yaml:"sandbox,omitempty"`
}

type PersonaSandbox struct {
    AllowedDomains []string `yaml:"allowed_domains,omitempty"`
    DenyReadPaths  []string `yaml:"deny_read_paths,omitempty"`
}

type Runtime struct {
    WorkspaceRoot        string          `yaml:"workspace_root"`
    // ... existing fields ...
    Sandbox              RuntimeSandbox  `yaml:"sandbox,omitempty"`
}

type RuntimeSandbox struct {
    Enabled              bool     `yaml:"enabled"`
    DefaultAllowedDomains []string `yaml:"default_allowed_domains,omitempty"`
    DenyReadPaths        []string `yaml:"deny_read_paths,omitempty"`
    EnvPassthrough       []string `yaml:"env_passthrough,omitempty"`
}
```

Example `wave.yaml`:

```yaml
runtime:
  workspace_root: .wave/workspaces
  sandbox:
    enabled: true
    default_allowed_domains:
      - api.anthropic.com
      - github.com
      - "*.github.com"
      - proxy.golang.org
      - sum.golang.org
    deny_read_paths:
      - "~/.ssh"
      - "~/.aws"
    env_passthrough:
      - ANTHROPIC_API_KEY
      - GH_TOKEN

personas:
  navigator:
    adapter: claude
    system_prompt_file: .wave/personas/navigator.md
    permissions:
      allowed_tools: ["Read", "Glob", "Grep", "Bash(git *)"]
      deny: ["Write(*)", "Edit(*)", "Bash(rm *)"]
    sandbox:
      allowed_domains:
        - api.anthropic.com
      # No github.com — navigator doesn't push

  implementer:
    adapter: claude
    system_prompt_file: .wave/personas/implementer.md
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

#### 2.5 Environment Variable Hygiene

Replace `os.Environ()` passthrough with a curated list driven by `runtime.sandbox.env_passthrough`:

```go
func (a *ClaudeAdapter) buildEnvironment(cfg AdapterRunConfig) []string {
    // Base environment (always needed)
    env := []string{
        "HOME=" + os.Getenv("HOME"),
        "PATH=" + os.Getenv("PATH"),
        "TERM=" + os.Getenv("TERM"),
        "TMPDIR=/tmp",
        "DISABLE_TELEMETRY=1",
        "DISABLE_ERROR_REPORTING=1",
        "CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY=1",
        "DISABLE_BUG_COMMAND=1",
    }

    // Add only explicitly allowed env vars from manifest
    for _, key := range cfg.EnvPassthrough {
        if val := os.Getenv(key); val != "" {
            env = append(env, key+"="+val)
        }
    }

    // Step-specific env vars
    env = append(env, cfg.Env...)
    return env
}
```

#### 2.6 AdapterRunConfig Extension

```go
type AdapterRunConfig struct {
    // ... existing fields ...
    AllowedDomains []string // Network domain allowlist from manifest
    DenyReadPaths  []string // Paths to deny reading
    EnvPassthrough []string // Env var names to pass through from host
}
```

The executor populates these from the manifest:

```go
// In executor.go, where cfg is built:
sandboxDomains := execution.Manifest.Runtime.Sandbox.DefaultAllowedDomains
if persona.Sandbox != nil && len(persona.Sandbox.AllowedDomains) > 0 {
    sandboxDomains = persona.Sandbox.AllowedDomains // persona overrides runtime default
}

cfg := adapter.AdapterRunConfig{
    // ... existing fields ...
    AllowedDomains: sandboxDomains,
    DenyReadPaths:  execution.Manifest.Runtime.Sandbox.DenyReadPaths,
    EnvPassthrough: execution.Manifest.Runtime.Sandbox.EnvPassthrough,
}
```

---

### Phase 3: Docker Container Per Step (Future)

Deferred. See research reports for architecture details.

---

## Security Model

### Defense in Depth (inside-out)

1. **CLAUDE.md restrictions** (prompt-level): Model is told what it can/cannot do
2. **settings.json permissions** (Claude Code enforcement): Allow/Deny rules enforced by Claude Code
3. **Claude Code sandbox** (OS-level for Bash): bubblewrap/seatbelt restricts what Bash commands can access
4. **Nix dev shell sandbox** (OS-level for everything): bubblewrap restricts the entire session
5. **Manifest as single source of truth**: All of the above are generated from `wave.yaml`

### Attack Vectors

| Attack Vector | Nix Sandbox | settings.json | CLAUDE.md |
|---|---|---|---|
| Read ~/.ssh | Blocked (tmpfs $HOME) | Can add denyRead | Persona told no access |
| Write outside project | Blocked (ro-bind) | Sandbox restricts to CWD | Persona told allowed tools |
| Network exfiltration | --share-net (gap) | allowedDomains filter | Persona told allowed domains |
| Env var leakage | --clearenv | N/A | N/A |
| Terminal escape | --new-session | N/A | N/A |
| Deny rule bypass | N/A | Enforced by Claude Code | Model aware of restrictions |

---

## Implementation Checklist

### Phase 1: Nix Flake

- [ ] Add `flake.nix` with sandboxed default shell and yolo escape hatch
- [ ] Test: filesystem isolation (can't read ~/.ssh, can't write /etc)
- [ ] Test: env isolation (AWS_SECRET_ACCESS_KEY not visible)
- [ ] Test: interactive detection (no sandbox for `nix develop --command`)

### Phase 2: Manifest-Driven Config

- [ ] Add `Deny` field to `ClaudePermissions` struct
- [ ] Write `cfg.DenyTools` to `settings.json` in `prepareWorkspace`
- [ ] Add `SandboxSettings` to `ClaudeSettings` struct
- [ ] Add `PersonaSandbox` and `RuntimeSandbox` to manifest types
- [ ] Executor reads sandbox config from manifest and passes to adapter
- [ ] Generate restriction section in `CLAUDE.md` from manifest permissions
- [ ] Replace `os.Environ()` with curated env from `runtime.sandbox.env_passthrough`
- [ ] Add `AllowedDomains`, `DenyReadPaths`, `EnvPassthrough` to `AdapterRunConfig`
- [ ] Table-driven tests for settings.json generation per persona
- [ ] Test: deny rules present in generated settings.json
- [ ] Test: CLAUDE.md contains restriction section
- [ ] Test: canary env var not leaked to subprocess

## Open Questions

1. **Should we remove `--dangerously-skip-permissions`?** With sandbox enabled and `autoAllowBashIfSandboxed: true`, sandboxed Bash auto-executes. But non-Bash tools (Read, Edit, Write) may still prompt without this flag. Needs testing in headless mode.

2. **Persona sandbox overrides vs runtime defaults**: Current design has persona-level `allowed_domains` override the runtime default entirely. Should it merge instead? (e.g., runtime provides `api.anthropic.com`, persona adds `github.com`)

3. **Network filtering in Phase 1**: The bwrap script uses `--share-net` (full network). True domain filtering requires a proxy (socat + `--unshare-net`). Should this be a Phase 1a sub-step, or deferred to Claude Code's built-in network sandbox handling it via settings.json?
