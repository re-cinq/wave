# Docker-Based Sandboxing for Claude Code: Research Report

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Claude Code's Native Sandboxing](#claude-codes-native-sandboxing)
3. [Docker Sandboxes (Official Docker Feature)](#docker-sandboxes-official-docker-feature)
4. [Anthropic's Official Devcontainer](#anthropics-official-devcontainer)
5. [Community Docker Solutions](#community-docker-solutions)
6. [Cloud Sandbox Platforms](#cloud-sandbox-platforms)
7. [OCI Runtime Alternatives](#oci-runtime-alternatives)
8. [Security Hardening Techniques](#security-hardening-techniques)
9. [Docker Compose Patterns](#docker-compose-patterns)
10. [Practical Considerations](#practical-considerations)
11. [Recommendations for Wave](#recommendations-for-wave)
12. [References](#references)

---

## Executive Summary

Running AI coding agents like Claude Code in isolated environments is no longer optional -- it is a baseline requirement for production use. The ecosystem has matured rapidly through 2025-2026, with multiple tiers of isolation available:

| Approach | Isolation Level | Startup Time | Overhead | Complexity |
|----------|----------------|-------------|----------|------------|
| Claude Code native sandbox (bubblewrap/seatbelt) | Process-level | Instant | None | Built-in |
| Docker containers | Kernel-shared | ~50ms | Minimal | Low |
| Docker Sandboxes (microVM) | VM-level | ~125ms | <5MB/VM | Low (Docker Desktop) |
| gVisor | User-space kernel | Fast | 10-30% I/O | Medium |
| Firecracker microVMs | Hardware-level | ~125ms | <5MB/VM | High (DIY) |
| Kata Containers | Hardware-level | 150-300ms | Tens of MB | Medium (K8s-native) |
| Cloudflare Sandbox | VM-level (edge) | Fast | N/A (managed) | Low |
| E2B | Firecracker microVM | <200ms | N/A (managed) | Low |

**Key finding**: Docker's new Sandbox feature (microVM-based, available since Docker Desktop 4.57+) provides the best balance of security, developer experience, and operational simplicity for running Claude Code. For self-hosted production pipelines like Wave, a hardened Docker container with network allowlisting is the most practical starting point, with Firecracker/Kata as the upgrade path for untrusted workloads.

---

## Claude Code's Native Sandboxing

Claude Code includes built-in sandboxing that uses OS-level primitives to enforce both filesystem and network isolation.

### Implementation

- **Linux**: Uses [bubblewrap](https://github.com/containers/bubblewrap) (bwrap) for process sandboxing
- **macOS**: Uses Apple's Seatbelt (sandbox-exec) framework
- Sandbox applies to the **Bash tool only** -- filesystem and network restrictions for Read/Edit/WebFetch use permission rules instead

### Filesystem Isolation

- Read/write access to the current working directory
- Blocks modification of files outside the working directory
- Prevents compromised agents from altering system files or reading sensitive data like SSH keys

### Network Isolation

- Internet access is routed through a Unix domain socket connected to a proxy server running **outside** the sandbox
- The proxy enforces domain-level restrictions and handles user confirmation for new domains
- Users can customize the proxy to enforce arbitrary rules on outgoing traffic

### Configuration (`settings.json`)

```json
{
  "sandbox": {
    "enabled": true,
    "allowUnsandboxedCommands": false,
    "excludedCommands": ["docker"],
    "network": {
      "allowedDomains": [
        "github.com",
        "*.npmjs.org",
        "registry.yarnpkg.com",
        "api.anthropic.com"
      ],
      "allowLocalBinding": true,
      "allowUnixSockets": ["/var/run/docker.sock"]
    }
  }
}
```

Key settings:

| Setting | Default | Purpose |
|---------|---------|---------|
| `enabled` | `true` | Enable/disable sandbox |
| `allowUnsandboxedCommands` | `true` | Allow retry outside sandbox on failure |
| `autoAllowBashIfSandboxed` | `false` | Skip bash permission prompts when sandboxed |
| `excludedCommands` | `[]` | Commands that bypass sandbox |
| `enableWeakerNestedSandbox` | `false` | Linux-only: weaker sandbox inside Docker |
| `network.allowedDomains` | `[]` | Allowlisted domains for outbound traffic |

### Security Caveat

The native sandbox includes an **escape hatch**: when a command fails due to sandbox restrictions, Claude can retry it outside the sandbox (with user permission). Disable this with `"allowUnsandboxedCommands": false`. Internal testing shows sandboxing **reduces permission prompts by 84%**.

### Limitations

- Only applies to the Bash tool
- Does not protect against exfiltration via DNS tunneling
- The `enableWeakerNestedSandbox` option exists for running inside Docker but "weakens security considerably"
- When running inside a Docker container, the container itself should provide the primary isolation boundary

---

## Docker Sandboxes (Official Docker Feature)

Docker Sandboxes is Docker's first-party solution for running AI coding agents in isolated microVM environments. Available since Docker Desktop 4.57+.

### Architecture

- Each sandbox runs in a **dedicated microVM** (not a standard container)
- Each microVM has its own private Docker daemon
- The agent runs inside the VM and cannot access the host Docker daemon, containers, or files outside the workspace
- Your workspace directory syncs between host and sandbox at the same absolute path

### Usage

```bash
# Create a sandbox from a project directory
docker sandbox create claude ~/project

# Run Claude Code inside the sandbox
docker sandbox run <sandbox-name>

# Pass a prompt directly
docker sandbox run <sandbox-name> -- "Refactor the auth module"

# Continue a previous session
docker sandbox run <sandbox-name> -- --continue
```

### Claude Code Template

Docker provides a pre-built sandbox template for Claude Code that includes:
- Ubuntu-based environment
- Development tools: Docker CLI, GitHub CLI, Node.js, Go, Python 3, Git, ripgrep, jq
- Non-root `agent` user with sudo privileges
- Private Docker daemon
- Claude launches with `--dangerously-skip-permissions` by default (safe because the VM provides isolation)

### Network Isolation

Network policy is configured per-sandbox:

```bash
# Allow specific domains
docker sandbox create claude ~/project --allow-host github.com --allow-host api.anthropic.com

# Block specific domains
docker sandbox create claude ~/project --block-host evil.com
```

Configuration lives at `~/.docker/sandboxes/vm/<vm-name>/proxy-config.json`:

```json
{
  "network": {
    "allowedDomains": ["github.com", "api.anthropic.com", "*.npmjs.org"],
    "blockedDomains": []
  }
}
```

### Git Credential Handling

Docker automatically:
1. Discovers `git user.name` and `user.email` configuration
2. Injects them into the sandbox
3. Prompts for authentication on first run
4. Stores credentials in a persistent Docker volume (`docker-claude-sandbox-data`)
5. Reuses credentials for future sandboxes

### Platform Support

| Platform | Support |
|----------|---------|
| macOS | Full (microVM-based) |
| Windows | Experimental (microVM-based) |
| Linux | Legacy container-based (Docker Desktop 4.57+) |

### Custom Templates

Build custom sandbox templates based on the official one:

```dockerfile
FROM docker/sandbox-templates:claude-code

# Add project-specific tools
RUN apt-get update && apt-get install -y \
    postgresql-client \
    redis-tools

# Add custom configuration
COPY my-settings.json /home/agent/.claude/settings.json
```

---

## Anthropic's Official Devcontainer

Anthropic maintains a reference devcontainer setup in the [claude-code repository](https://github.com/anthropics/claude-code/tree/main/.devcontainer). This is the officially recommended approach for teams needing consistent, secure environments.

### Components

Three files make up the devcontainer:

#### 1. `devcontainer.json`

```jsonc
{
  "build": {
    "dockerfile": "Dockerfile",
    "args": {
      "TZ": "America/Los_Angeles",
      "CLAUDE_CODE_VERSION": "latest"
    }
  },
  "runArgs": ["--cap-add=NET_ADMIN"],  // Required for firewall
  "remoteUser": "node",
  "containerEnv": {
    "NODE_OPTIONS": "--max-old-space-size=4096"
  },
  "mounts": [
    // Bash history persistence
    "source=claude-code-bashhistory,target=/commandhistory,type=volume",
    // Claude configuration persistence
    "source=claude-code-claude-config,target=/home/node/.claude,type=volume",
    // Workspace
    "source=${localWorkspaceFolder},target=/workspace,type=bind,consistency=delegated"
  ],
  "customizations": {
    "vscode": {
      "extensions": [
        "anthropic.claude-code",
        "dbaeumer.vscode-eslint",
        "esbenp.prettier-vscode",
        "eamodio.gitlens"
      ],
      "settings": {
        "editor.formatOnSave": true,
        "editor.defaultFormatter": "esbenp.prettier-vscode",
        "terminal.integrated.defaultProfile.linux": "zsh"
      }
    }
  },
  "postStartCommand": "bash /usr/local/bin/init-firewall.sh",
  "waitFor": "postStartCommand"
}
```

#### 2. `Dockerfile`

```dockerfile
FROM node:20

ARG TZ=America/Los_Angeles
ARG CLAUDE_CODE_VERSION=latest

# System dependencies
RUN apt-get update && apt-get install -y \
    less git procps sudo fzf zsh man-db unzip gnupg2 \
    gh iptables ipset iproute2 dnsutils aggregate jq \
    nano vim \
    && rm -rf /var/lib/apt/lists/*

# Install git-delta for better diffs
RUN wget -qO /tmp/git-delta.deb \
    https://github.com/dandavison/delta/releases/download/0.18.2/git-delta_0.18.2_amd64.deb \
    && dpkg -i /tmp/git-delta.deb && rm /tmp/git-delta.deb

# Create directories
RUN mkdir -p /commandhistory /home/node/.npm-global /workspace /home/node/.claude

# Zsh with powerline10k theme
RUN sh -c "$(wget -O- https://github.com/deluan/zsh-in-docker/releases/download/v1.2.1/zsh-in-docker.sh)" -- \
    -t powerlevel10k/powerlevel10k

# Install Claude Code CLI
RUN npm install -g @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}

# Firewall script
COPY init-firewall.sh /usr/local/bin/init-firewall.sh
RUN chmod +x /usr/local/bin/init-firewall.sh

# Grant node user sudo for firewall init
RUN echo "node ALL=(ALL) NOPASSWD: /usr/local/bin/init-firewall.sh" >> /etc/sudoers.d/node-firewall

USER node
WORKDIR /workspace
SHELL ["/bin/zsh", "-c"]

ENV NPM_CONFIG_PREFIX=/home/node/.npm-global
ENV EDITOR=nano
ENV VISUAL=nano
```

#### 3. `init-firewall.sh`

The firewall script implements a **default-deny** network policy:

```bash
#!/bin/bash
set -euo pipefail

# Preserve Docker DNS rules before flushing
# ... (saves existing DOCKER rules)

# Flush and set default policies
iptables -F OUTPUT
iptables -P OUTPUT DROP

# Allow established connections
iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow loopback
iptables -A OUTPUT -o lo -j ACCEPT

# Allow DNS (UDP 53)
iptables -A OUTPUT -p udp --dport 53 -j ACCEPT

# Allow SSH (TCP 22)
iptables -A OUTPUT -p tcp --dport 22 -j ACCEPT

# Allowlisted domains (resolved to IPs via dig, stored in ipsets)
ALLOWED_DOMAINS=(
    "registry.npmjs.org"
    "api.anthropic.com"
    "sentry.io"
    "statsig.anthropic.com"
    "statsig.com"
    "marketplace.visualstudio.com"
    "vscode.blob.core.windows.net"
    "update.code.visualstudio.com"
)

# Create ipset and resolve each domain
ipset create allowed_ips hash:net
for domain in "${ALLOWED_DOMAINS[@]}"; do
    for ip in $(dig +short "$domain" | grep -E '^[0-9]'); do
        ipset add allowed_ips "$ip/32" 2>/dev/null || true
    done
done

# Allow GitHub IP ranges (fetched from GitHub API)
for range in $(curl -s https://api.github.com/meta | jq -r '.git[],.api[],.web[]' | sort -u); do
    ipset add allowed_ips "$range" 2>/dev/null || true
done

# Allow traffic to resolved IPs
iptables -A OUTPUT -m set --match-set allowed_ips dst -j ACCEPT

# Verification tests
# Confirm blocked: curl -s --max-time 5 http://example.com (should fail)
# Confirm allowed: curl -s https://api.github.com (should succeed)
```

**Allowlisted domains**:
- `registry.npmjs.org` -- npm packages
- `api.anthropic.com` -- Claude API
- `sentry.io` -- error tracking
- `statsig.anthropic.com`, `statsig.com` -- analytics
- `marketplace.visualstudio.com`, `vscode.blob.core.windows.net`, `update.code.visualstudio.com` -- VS Code extensions
- GitHub IP ranges (dynamically fetched from `api.github.com/meta`)

### Security Properties

| Property | Status |
|----------|--------|
| Network default-deny | Yes (iptables DROP policy) |
| Domain allowlisting | Yes (ipset + iptables) |
| Non-root execution | Yes (runs as `node` user) |
| Filesystem isolation | Yes (only `/workspace` mounted) |
| Credential isolation | Partial (Claude config in volume) |
| DNS exfiltration prevention | No (UDP 53 is allowed) |

### Limitations

- Requires `--cap-add=NET_ADMIN` for iptables (grants network configuration capability)
- Does not prevent exfiltration via DNS tunneling
- Claude Code credentials in the container could be accessed by a malicious project
- Shared host kernel (standard Docker container, not a microVM)

---

## Community Docker Solutions

### textcortex/claude-code-sandbox

[GitHub Repository](https://github.com/textcortex/claude-code-sandbox)

A third-party tool that creates isolated Docker containers for running Claude Code autonomously.

**Key features:**
- Files are **copied into** containers rather than mounted (true isolation)
- Automatic isolated branch creation per session
- Real-time commit notifications with syntax-highlighted diffs
- Interactive approval before pushing branches or creating PRs
- Web UI at `http://localhost:3456` for monitoring
- Supports both Docker and Podman (automatic detection)

**Configuration** (`claude-sandbox.config.json`):

```json
{
  "setupCommands": [
    "npm install",
    "pip install -r requirements.txt"
  ],
  "mounts": [
    {
      "source": "./data",
      "target": "/workspace/data",
      "readonly": true
    }
  ],
  "environment": {
    "NODE_ENV": "development"
  },
  "envFile": ".env",
  "maxThinkingTokens": 10000,
  "bashTimeout": 300
}
```

**Usage:**
```bash
claude-sandbox start          # Begin new container
claude-sandbox attach [id]    # Connect to existing
claude-sandbox list           # View all containers
claude-sandbox stop           # Terminate
claude-sandbox clean          # Remove stopped containers
```

### RchGrav/claudebox

[GitHub Repository](https://github.com/RchGrav/claudebox)

A comprehensive Docker development environment with pre-configured language profiles.

**Key features:**
- 15+ pre-configured development profiles (C/C++, Python, Rust, Go, Java, etc.)
- Per-project Docker image isolation (named `claudebox-<project-name>`)
- Project-specific firewall allowlists via `/allowlist` command
- UID/GID matching with host user for permission alignment
- Persistent authentication, shell history, and tool configurations
- Multi-instance support for parallel projects

**Volume mounting strategy:**
- Current working directory mounted to `/workspace`
- Per-project `.claude/` for auth state persistence
- `.zsh_history` for shell history
- `firewall/allowlist` for network rules
- Global `~/.claude/` mounted **read-only**

**Usage:**
```bash
claudebox                     # Launch Claude CLI
claudebox shell              # Open Zsh container shell
claudebox profile python ml  # Install Python + ML tools
claudebox allowlist          # Manage firewall rules
claudebox rebuild            # Force image rebuild
```

### Zeeno-atl/claude-code

[GitHub Repository](https://github.com/Zeeno-atl/claude-code)

A minimal containerized Claude Code -- always installs the latest CLI version on startup.

```bash
docker pull ghcr.io/zeeno-atl/claude-code:latest
docker run -it --rm \
  -v "$(pwd):/app" \
  -e ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" \
  ghcr.io/zeeno-atl/claude-code:latest
```

---

## Cloud Sandbox Platforms

### E2B (e2b.dev)

[Website](https://e2b.dev) | [GitHub](https://github.com/e2b-dev/E2B)

**Architecture**: Firecracker microVMs with <200ms cold start. Used by 88% of Fortune 100 companies.

**Key properties:**
- Each sandbox is a Firecracker microVM with full kernel isolation
- Jailer companion process uses cgroups and namespaces for defense-in-depth
- Sandboxes can run up to 24 hours
- Full filesystem I/O via Python/JavaScript SDK
- Unrestricted internet access by default (configurable)
- Multi-cloud: AWS, GCP, Azure, or your VPC
- LLM-agnostic (works with any model)

**SDK example (Python):**
```python
from e2b_code_interpreter import Sandbox

sandbox = Sandbox()
execution = sandbox.run_code("print('hello from microVM')")
print(execution.text)
sandbox.close()
```

**Self-hosting**: Terraform scripts for GCP (AWS in progress).

**Cost model**: Pay per sandbox-second. Variable costs at scale vs. self-hosted.

### Cloudflare Sandbox SDK

[Documentation](https://developers.cloudflare.com/sandbox/) | [GitHub](https://github.com/cloudflare/sandbox-sdk)

**Architecture**: Firecracker microVMs running on Cloudflare's edge network, orchestrated via Workers and Durable Objects.

**Claude Code integration:**
```bash
npm create cloudflare@latest -- claude-code-sandbox \
  --template=cloudflare/sandbox-sdk/examples/claude-code
```

**Key properties:**
- Sandboxes are Firecracker microVMs at the edge
- Managed via Cloudflare Workers (Durable Objects for lifecycle)
- Claude Code runs headless on any repo
- Also offers V8 isolate-based execution for lighter workloads (microsecond startup, ~few MB overhead)
- Currently in Beta

### Modal

[Website](https://modal.com)

Python-first serverless container platform. Each function call runs in its own container with:
- gVisor-based isolation
- GPU support (NVIDIA)
- Sub-second cold starts
- Pay-per-second billing
- Git and pip/conda integration

### Daytona

[Website](https://daytona.io)

Agent-native sandboxes that persist like real workspaces. Useful for long-running development sessions where the agent needs a stable environment.

---

## OCI Runtime Alternatives

### Standard Docker Containers

| Property | Value |
|----------|-------|
| Isolation | Shared kernel (namespaces + cgroups) |
| Startup | ~50ms |
| Overhead | Minimal |
| Compatibility | Full Linux |
| Security | Weakest for untrusted code |

Containers share the host kernel. The Linux kernel averages 300+ CVEs annually. A single kernel exploit compromises every container on the host (cf. CVE-2019-5736 runc escape).

**When appropriate**: Running your own trusted code, or as the outermost layer with additional isolation inside (e.g., bubblewrap inside Docker).

### gVisor

[Website](https://gvisor.dev)

| Property | Value |
|----------|-------|
| Isolation | User-space kernel (Sentry process) |
| Startup | Fast |
| Overhead | 10-30% I/O penalty, minimal compute |
| Compatibility | ~70-80% of Linux syscalls |
| Security | Medium (dual-codebase attack required) |

gVisor intercepts system calls via ptrace or KVM and handles them in a Go-based "Sentry" process. Only a tiny, audited subset of syscalls reach the host kernel. An attacker must exploit both the Go Sentry and the host kernel -- two completely different codebases.

**Limitations**: Does not support systemd, Docker-in-Docker, or certain networking features. Some build tools and package managers may fail.

**Usage with Docker:**
```bash
# Install gVisor runtime
# Then run containers with it
docker run --runtime=runsc myimage
```

**Who uses it**: Modal chose gVisor for their serverless containers.

### Firecracker

[Website](https://firecracker-microvm.github.io)

| Property | Value |
|----------|-------|
| Isolation | Hardware-level (KVM hypervisor) |
| Startup | ~125ms |
| Overhead | <5 MiB per microVM |
| Compatibility | Full (own kernel) |
| Security | Strongest (hardware boundary) |

Written in ~50,000 lines of Rust, Firecracker emulates exactly 5 virtual devices. Each microVM runs its own guest kernel, completely separate from the host. The companion "jailer" process sets up cgroups and namespaces around the VMM itself for defense-in-depth.

**Density**: Up to 150 VMs/second/host. AWS Lambda runs millions in production.

**Limitations**: No GPU support, no pause/resume, no live migration. Requires building custom kernel images, root filesystems, and networking configuration.

**Who uses it**: E2B, Docker Sandboxes, Cloudflare Sandbox, AWS Lambda, AWS Fargate.

### Kata Containers

[Website](https://katacontainers.io)

| Property | Value |
|----------|-------|
| Isolation | Hardware-level (KVM via QEMU/Cloud Hypervisor) |
| Startup | 150-300ms |
| Overhead | Tens of MB per VM |
| Compatibility | Full (own kernel, OCI-compliant) |
| Security | Strongest (hardware boundary) |

OCI-compliant -- each container runs in its own VM while keeping Docker and Kubernetes workflows intact. Best for teams that want Firecracker-level security with Kubernetes-native integration.

**Advantages over raw Firecracker**: Native Kubernetes RuntimeClass support, standard container tooling, lower operational complexity.

**Who uses it**: Organizations needing production K8s with strong multi-tenant isolation.

### Podman

[Website](https://podman.io)

| Property | Value |
|----------|-------|
| Isolation | Shared kernel (same as Docker) |
| Startup | Up to 50% faster than Docker |
| Overhead | Minimal |
| Key advantage | Rootless by design, daemonless |

Podman's rootless architecture means containers execute under your user account, not root. Even if an attacker escapes the container, they are confined to your unprivileged user's permissions. No daemon means no daemon-based attack surface and no cross-site forgery exploits.

**Audit trail**: Since each container is instantiated directly through a user login session, `auditd` can accurately identify which user started each container process.

**Docker compatibility**: CLI-compatible (`alias docker=podman`), OCI-compliant images, supports Docker Compose via `podman-compose`.

**Recommended for**: CI/CD pipelines (no Docker-in-Docker hacks needed), regulated environments, zero-trust infrastructures.

### Kubernetes Agent Sandbox

[GitHub](https://github.com/kubernetes-sigs/agent-sandbox)

An open-source Kubernetes controller providing a declarative API for managing isolated, stateful, singleton workloads -- ideal for AI agent runtimes. Provides:
- Persistent storage
- Stable pod identity
- OCI-compliant isolation
- Integration with Kata Containers or gVisor RuntimeClasses

### Decision Matrix

| Use Case | Recommended Runtime |
|----------|-------------------|
| Local development (trusted code) | Docker or Podman |
| Local development (untrusted repos) | Docker Sandboxes (microVM) |
| CI/CD pipelines | Podman (rootless, daemonless) |
| Production multi-tenant | Kata Containers on K8s |
| Maximum density serverless | Firecracker (custom) |
| Edge/cloud managed | E2B or Cloudflare Sandbox |

---

## Security Hardening Techniques

### Capability Dropping

Drop all Linux capabilities and add back only what is needed:

```bash
docker run \
  --cap-drop ALL \
  --cap-add NET_BIND_SERVICE \
  myimage
```

For Claude Code, the only capability that might be needed is `NET_ADMIN` (for firewall rules). If using external network policy (Docker network or host firewall), even this can be dropped.

### Read-Only Filesystem

Run with a read-only root filesystem, using tmpfs for writable paths:

```bash
docker run \
  --read-only \
  --tmpfs /tmp:size=100M \
  --tmpfs /home/agent/.npm:size=200M \
  --tmpfs /home/agent/.cache:size=200M \
  -v ./project:/workspace \
  myimage
```

### Seccomp Profiles

Docker's default seccomp profile blocks ~44 dangerous syscalls. For AI agents, consider a more restrictive custom profile:

```json
{
  "defaultAction": "SCMP_ACT_ERRNO",
  "architectures": ["SCMP_ARCH_X86_64"],
  "syscalls": [
    {
      "names": ["read", "write", "open", "close", "stat", "fstat",
                "mmap", "mprotect", "munmap", "brk", "ioctl",
                "access", "pipe", "select", "clone", "execve",
                "wait4", "kill", "getpid", "socket", "connect",
                "sendto", "recvfrom", "bind", "listen", "accept"],
      "action": "SCMP_ACT_ALLOW"
    }
  ]
}
```

Apply it:
```bash
docker run --security-opt seccomp=/path/to/profile.json myimage
```

### AppArmor Profiles

AppArmor provides finer-grained control than capabilities -- it can restrict file operations on specific paths:

```
#include <tunables/global>
profile claude-code-agent flags=(attach_disconnected) {
  #include <abstractions/base>
  #include <abstractions/nameservice>

  # Allow read/write to workspace only
  /workspace/** rw,
  /tmp/** rw,

  # Allow executing Claude and Node
  /usr/local/bin/node ix,
  /usr/local/bin/claude ix,

  # Block access to sensitive paths
  deny /etc/shadow r,
  deny /etc/passwd w,
  deny /root/** rw,
  deny /home/*/.ssh/** rw,

  # Network: allow TCP/UDP outbound
  network tcp,
  network udp,
}
```

### No New Privileges

Prevent privilege escalation via setuid/setgid binaries:

```bash
docker run --security-opt no-new-privileges:true myimage
```

### Resource Limits

Prevent fork bombs, memory exhaustion, and disk filling:

```bash
docker run \
  --memory=4g \
  --memory-swap=4g \
  --cpus=2 \
  --pids-limit=256 \
  --storage-opt size=10G \
  --ulimit nofile=1024:1024 \
  --ulimit nproc=256:256 \
  myimage
```

### Network Isolation

#### Option A: Docker Network with No External Access

```bash
# Create an internal-only network
docker network create --internal agent-net

# Run container on internal network (no internet)
docker run --network=agent-net myimage
```

#### Option B: iptables Allowlisting (Inside Container)

Requires `--cap-add=NET_ADMIN`. See the Anthropic devcontainer `init-firewall.sh` above.

#### Option C: Host-Level Firewall

Use the host's nftables/iptables to restrict container traffic by container subnet:

```bash
# Get container subnet
SUBNET=$(docker network inspect bridge -f '{{range .IPAM.Config}}{{.Subnet}}{{end}}')

# Allow only specific destinations
iptables -I FORWARD -s $SUBNET -d 140.82.112.0/20 -j ACCEPT   # GitHub
iptables -I FORWARD -s $SUBNET -d 160.16.0.0/16 -j ACCEPT     # Anthropic
iptables -A FORWARD -s $SUBNET -j DROP                          # Block rest
```

#### Option D: DNS-Level Filtering

Use a DNS proxy that only resolves allowlisted domains:

```yaml
services:
  dns-proxy:
    image: coredns/coredns
    volumes:
      - ./Corefile:/etc/coredns/Corefile
    networks:
      - agent-net

  claude-agent:
    dns:
      - dns-proxy
    networks:
      - agent-net
```

### Combined Hardened Container

```bash
docker run -it --rm \
  --name claude-agent \
  --user 1000:1000 \
  --read-only \
  --cap-drop ALL \
  --security-opt no-new-privileges:true \
  --security-opt seccomp=/path/to/seccomp.json \
  --memory=4g \
  --memory-swap=4g \
  --cpus=2 \
  --pids-limit=256 \
  --tmpfs /tmp:size=100M,noexec,nosuid \
  --tmpfs /home/agent/.npm:size=200M \
  --tmpfs /home/agent/.cache:size=200M \
  -v "$(pwd)/project:/workspace:rw" \
  -e ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" \
  --network=agent-net \
  claude-code-hardened:latest
```

---

## Docker Compose Patterns

### Minimal: Claude Code with Network Isolation

```yaml
# docker-compose.yml
version: "3.8"

services:
  claude-agent:
    build:
      context: .
      dockerfile: Dockerfile.claude
    container_name: claude-agent
    user: "1000:1000"
    read_only: true
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    tmpfs:
      - /tmp:size=100M
      - /home/agent/.npm:size=200M
      - /home/agent/.cache:size=500M
    volumes:
      - ./project:/workspace:rw
      - claude-config:/home/agent/.claude
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
    mem_limit: 4g
    cpus: 2
    pids_limit: 256
    networks:
      - agent-net
    dns:
      - 8.8.8.8

volumes:
  claude-config:

networks:
  agent-net:
    driver: bridge
```

### With Git Credential Helper

```yaml
services:
  claude-agent:
    build:
      context: .
      dockerfile: Dockerfile.claude
    volumes:
      - ./project:/workspace:rw
      - claude-config:/home/agent/.claude
      # Git config (read-only)
      - ${HOME}/.gitconfig:/home/agent/.gitconfig:ro
      # GitHub CLI config (read-only)
      - ${HOME}/.config/gh:/home/agent/.config/gh:ro
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - GIT_AUTHOR_NAME=${GIT_AUTHOR_NAME}
      - GIT_AUTHOR_EMAIL=${GIT_AUTHOR_EMAIL}
      - GIT_COMMITTER_NAME=${GIT_COMMITTER_NAME}
      - GIT_COMMITTER_EMAIL=${GIT_COMMITTER_EMAIL}
```

### Multi-Service: Agent + Database + DNS Proxy

```yaml
version: "3.8"

services:
  dns-proxy:
    image: coredns/coredns:1.11
    volumes:
      - ./coredns/Corefile:/etc/coredns/Corefile:ro
    networks:
      agent-net:
        ipv4_address: 172.28.0.2

  claude-agent:
    build:
      context: .
      dockerfile: Dockerfile.claude
    user: "1000:1000"
    read_only: true
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    tmpfs:
      - /tmp:size=100M
      - /home/agent/.npm:size=200M
      - /home/agent/.cache:size=500M
    volumes:
      - ./project:/workspace:rw
      - claude-config:/home/agent/.claude
      - ./artifacts:/workspace/output:rw
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - DATABASE_URL=postgres://agent:pass@db:5432/devdb
    depends_on:
      - db
      - dns-proxy
    dns:
      - 172.28.0.2
    mem_limit: 4g
    cpus: 2
    pids_limit: 256
    networks:
      - agent-net

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: agent
      POSTGRES_PASSWORD: pass
      POSTGRES_DB: devdb
    volumes:
      - pgdata:/var/lib/postgresql/data
    networks:
      - agent-net
    # No external network access
    # Database is only reachable within agent-net

volumes:
  claude-config:
  pgdata:

networks:
  agent-net:
    driver: bridge
    ipam:
      config:
        - subnet: 172.28.0.0/16
```

### CoreDNS Configuration for Domain Allowlisting

```
# Corefile
. {
    # Only forward queries for allowed domains
    forward github.com 8.8.8.8
    forward api.anthropic.com 8.8.8.8
    forward registry.npmjs.org 8.8.8.8
    forward pypi.org 8.8.8.8
    forward golang.org 8.8.8.8
    forward proxy.golang.org 8.8.8.8
    forward sum.golang.org 8.8.8.8

    # Block everything else
    template IN A . {
        rcode NXDOMAIN
    }
    template IN AAAA . {
        rcode NXDOMAIN
    }

    log
    errors
}
```

---

## Practical Considerations

### Performance Overhead

| Approach | Startup | Runtime Overhead | Memory Overhead |
|----------|---------|-----------------|-----------------|
| Native sandbox (bubblewrap) | Instant | None | None |
| Docker container | ~50ms | ~0% | ~10MB |
| Docker + gVisor | ~100ms | 10-30% I/O | ~50MB |
| Docker Sandbox (microVM) | ~125ms | ~2-5% | ~5MB |
| Firecracker (raw) | ~125ms | ~2-5% | ~5MB |
| Kata Containers | 150-300ms | ~5% | ~30-50MB |

For Claude Code specifically, the dominant cost is API latency (seconds per turn), so container startup and runtime overhead are negligible in practice.

### File Permission Mapping (UID/GID)

The most common problem with Docker volume mounts is file ownership mismatches.

**Solution 1: Match host UID/GID in Dockerfile**
```dockerfile
ARG HOST_UID=1000
ARG HOST_GID=1000
RUN groupadd -g ${HOST_GID} agent && \
    useradd -u ${HOST_UID} -g ${HOST_GID} -m agent
USER agent
```

Build with: `docker build --build-arg HOST_UID=$(id -u) --build-arg HOST_GID=$(id -g) .`

**Solution 2: Use user namespace remapping**
```json
// /etc/docker/daemon.json
{
  "userns-remap": "default"
}
```

**Solution 3: Podman (automatic)**
Podman automatically maps the container root to your unprivileged user via user namespaces.

### API Key Security

**DO**: Pass via environment variable at runtime
```bash
docker run -e ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" myimage
```

**DO**: Use Docker secrets (Swarm) or mounted secret files
```yaml
services:
  claude-agent:
    secrets:
      - anthropic_api_key
    environment:
      - ANTHROPIC_API_KEY_FILE=/run/secrets/anthropic_api_key

secrets:
  anthropic_api_key:
    file: ./secrets/api_key.txt
```

**DO NOT**: Bake keys into the Docker image (`ENV` in Dockerfile)
**DO NOT**: Mount your entire home directory or `.env` file with other secrets
**DO NOT**: Use `--privileged` mode

### Session State Persistence

To persist Claude Code session state across container restarts:

```yaml
volumes:
  # Claude configuration and session state
  - claude-config:/home/agent/.claude
  # Command history
  - claude-history:/home/agent/.bash_history
  # Workspace changes
  - ./project:/workspace:rw
```

The `claude-config` volume preserves:
- Authentication tokens
- Session history
- User preferences
- Conversation state (for `--continue`)

### DNS Exfiltration Risk

All container-based approaches (except fully internal networks) allow DNS resolution, which can be exploited for data exfiltration. Mitigations:
1. Use a DNS proxy that only resolves allowlisted domains (CoreDNS example above)
2. Monitor DNS query logs for anomalous patterns
3. Use `--dns` to point at a controlled resolver
4. For maximum security, use `--network=none` and provide a SOCKS/HTTP proxy for API access only

---

## Recommendations for Wave

Wave executes multi-step pipelines where each step runs Claude Code (or other LLM CLIs) as a subprocess. The sandboxing strategy should align with Wave's existing workspace isolation model.

### Tier 1: Immediate (Low Effort)

**Use Claude Code's native sandbox + Wave's workspace isolation**

Wave already creates ephemeral workspaces per step at `.wave/workspaces/<pipeline>/<step>/`. Configure Claude Code's native sandbox for each step execution:

```json
{
  "sandbox": {
    "enabled": true,
    "allowUnsandboxedCommands": false,
    "network": {
      "allowedDomains": [
        "api.anthropic.com",
        "github.com",
        "*.github.com"
      ]
    }
  }
}
```

This requires no Docker setup and provides process-level filesystem + network isolation. The `allowUnsandboxedCommands: false` setting eliminates the escape hatch.

### Tier 2: Standard (Moderate Effort)

**Docker container per pipeline step**

Extend Wave's adapter to optionally run steps inside Docker containers:

```go
// internal/adapter/docker.go
type DockerAdapter struct {
    Image       string
    Mounts      []Mount
    NetworkMode string
    Resources   ResourceLimits
}
```

Each step would:
1. Build/pull the container image (with Claude Code pre-installed)
2. Mount only the step's workspace directory
3. Inject artifacts from previous steps
4. Pass `ANTHROPIC_API_KEY` via environment variable
5. Run with `--cap-drop ALL --read-only --security-opt no-new-privileges`
6. Extract output artifacts after completion
7. Destroy the container

This provides kernel-level namespace isolation, resource limits, and clean teardown.

### Tier 3: High Security (Higher Effort)

**Firecracker microVMs or Docker Sandboxes for untrusted pipelines**

For pipelines processing untrusted code (e.g., community contributions, automated PRs):
- Use Docker Sandboxes (if Docker Desktop is available)
- Or deploy Kata Containers on a Kubernetes cluster
- Or build Firecracker microVM orchestration (highest effort, strongest isolation)

### Dockerfile for Wave Steps

```dockerfile
FROM node:20-slim

# Security: non-root user
RUN groupadd -g 1000 wave && \
    useradd -u 1000 -g wave -m -s /bin/bash wave

# System dependencies for common build tasks
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl ca-certificates jq \
    && rm -rf /var/lib/apt/lists/*

# Install Claude Code CLI
RUN npm install -g @anthropic-ai/claude-code

# Install Go (for Wave projects)
ARG GO_VERSION=1.25
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | \
    tar -C /usr/local -xzf -
ENV PATH="/usr/local/go/bin:${PATH}"

# Switch to non-root user
USER wave
WORKDIR /workspace

# Default: run Claude Code in headless mode
ENTRYPOINT ["claude", "--dangerously-skip-permissions"]
```

### Docker Compose for Wave Pipeline Execution

```yaml
version: "3.8"

services:
  wave-step:
    build:
      context: .
      dockerfile: Dockerfile.wave-step
      args:
        GO_VERSION: "1.25"
    user: "1000:1000"
    read_only: true
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    tmpfs:
      - /tmp:size=200M
      - /home/wave/.npm:size=200M
      - /home/wave/.cache:size=500M
      - /home/wave/.claude:size=100M
    volumes:
      # Step workspace (ephemeral)
      - ${WAVE_WORKSPACE}:/workspace:rw
      # Artifacts from previous steps (read-only)
      - ${WAVE_ARTIFACTS}:/artifacts:ro
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - WAVE_STEP_NAME=${WAVE_STEP_NAME}
      - WAVE_PIPELINE_ID=${WAVE_PIPELINE_ID}
    mem_limit: 4g
    cpus: 2
    pids_limit: 256
    networks:
      - wave-net

networks:
  wave-net:
    driver: bridge
```

---

## References

### Official Documentation
- [Claude Code Sandboxing Documentation](https://code.claude.com/docs/en/sandboxing)
- [Claude Code Development Containers](https://code.claude.com/docs/en/devcontainer)
- [Anthropic Engineering: Claude Code Sandboxing](https://www.anthropic.com/engineering/claude-code-sandboxing)
- [Docker Sandboxes Documentation](https://docs.docker.com/ai/sandboxes)
- [Configure Claude Code in Docker Sandbox](https://docs.docker.com/ai/sandboxes/claude-code/)

### Official Repositories
- [anthropics/claude-code Devcontainer](https://github.com/anthropics/claude-code/tree/main/.devcontainer)
- [anthropics/devcontainer-features](https://github.com/anthropics/devcontainer-features)
- [docker/compose-for-agents](https://github.com/docker/compose-for-agents)

### Community Tools
- [textcortex/claude-code-sandbox](https://github.com/textcortex/claude-code-sandbox)
- [RchGrav/claudebox](https://github.com/RchGrav/claudebox)
- [Zeeno-atl/claude-code](https://github.com/Zeeno-atl/claude-code)

### Cloud Platforms
- [E2B - Enterprise AI Agent Cloud](https://e2b.dev/)
- [Cloudflare Sandbox SDK](https://developers.cloudflare.com/sandbox/)
- [Cloudflare: Run Claude Code on a Sandbox](https://developers.cloudflare.com/sandbox/tutorials/claude-code/)

### Isolation Technologies
- [Firecracker MicroVM](https://firecracker-microvm.github.io)
- [gVisor](https://gvisor.dev)
- [Kata Containers](https://katacontainers.io)
- [Podman](https://podman.io)
- [Kubernetes Agent Sandbox](https://github.com/kubernetes-sigs/agent-sandbox)

### Security References
- [Docker Seccomp Profiles](https://docs.docker.com/engine/security/seccomp/)
- [OWASP Docker Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Docker_Security_Cheat_Sheet.html)
- [Docker Security: Audit to AI Protection](https://dzone.com/articles/docker-security-audit-to-ai-protection)

### Comparison Articles
- [How to Sandbox LLMs & AI Shell Tools (CodeAnt.ai)](https://www.codeant.ai/blogs/agentic-rag-shell-sandboxing)
- [Choosing a Workspace for AI Agents: gVisor vs Kata vs Firecracker](https://dev.to/agentsphere/choosing-a-workspace-for-ai-agents-the-ultimate-showdown-between-gvisor-kata-and-firecracker-b10)
- [Firecracker vs Docker: Security Tradeoffs for Agentic Workloads](https://nextkicklabs.substack.com/p/firecracker-vs-docker-security-tradeoffs)
- [How to Sandbox AI Agents in 2026 (Northflank)](https://northflank.com/blog/how-to-sandbox-ai-agents)
- [Code Sandboxes for LLMs and AI Agents](https://amirmalik.net/2025/03/07/code-sandboxes-for-llm-ai-agents)
- [Podman vs Docker 2026 (Last9)](https://last9.io/blog/podman-vs-docker/)

### Blog Posts and Tutorials
- [Docker Blog: Docker Sandboxes for Coding Agents](https://www.docker.com/blog/docker-sandboxes-run-claude-code-and-other-coding-agents-unsupervised-but-safely/)
- [Docker Blog: Secure AI Agents at Runtime](https://www.docker.com/blog/secure-ai-agents-runtime-security/)
- [Docker Blog: Build AI Agents with Docker Compose](https://www.docker.com/blog/build-ai-agents-with-docker-compose/)
- [Running Claude Code in a Container (Substack)](https://expandmapping.substack.com/p/running-claude-code-in-a-container)
- [Running Claude Code Agents in Docker (Medium)](https://medium.com/@dan.avila7/running-claude-code-agents-in-docker-containers-for-complete-isolation-63036a2ef6f4)
- [How to Safely Run AI Agents Inside a DevContainer](https://codewithandrea.com/articles/run-ai-agents-inside-devcontainer/)
- [Docker Sandboxes + Claude Code: What Works, What Breaks (Arcade)](https://blog.arcade.dev/using-docker-sandboxes-with-claude-code)
- [Running Claude Code Safely in Devcontainers (Solberg)](https://www.solberg.is/claude-devcontainer)
