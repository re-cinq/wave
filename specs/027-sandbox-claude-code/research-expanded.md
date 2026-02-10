# Sandboxing AI Coding Agents: Expanded Research Report

> Research date: 2026-02-09
> Focus: Claude Code sandboxing, emerging isolation technologies, and applicability to Wave
> Supplements: `research.md` (Docker-focused research)

---

## Table of Contents

1. [Claude Code Built-In Permission Controls](#1-claude-code-built-in-permission-controls)
2. [Claude Code Native Sandboxing (Deep Dive)](#2-claude-code-native-sandboxing-deep-dive)
3. [Anthropic Official Recommendations](#3-anthropic-official-recommendations)
4. [OS-Level Sandboxing Primitives](#4-os-level-sandboxing-primitives)
5. [Filesystem Virtualization Approaches](#5-filesystem-virtualization-approaches)
6. [Network-Level Sandboxing](#6-network-level-sandboxing)
7. [Cloud Sandbox Platforms for AI Agents](#7-cloud-sandbox-platforms-for-ai-agents)
8. [Container and MicroVM Isolation Technologies](#8-container-and-microvm-isolation-technologies)
9. [Multi-Agent Orchestration Sandboxing](#9-multi-agent-orchestration-sandboxing)
10. [Applicability to Wave](#10-applicability-to-wave)
11. [Comparison Matrix](#11-comparison-matrix)
12. [Recommendations](#12-recommendations)
13. [Sources](#13-sources)

---

## 1. Claude Code Built-In Permission Controls

### Permission Model

Claude Code implements a layered permission system with four tool categories:

- **Bash Commands**: Terminal instructions matched by exact string or wildcard patterns (e.g., `Bash(git *)`, `Bash(npm test)`)
- **Read/Edit**: File access controls with path glob patterns (e.g., `Read(.env*)` can be denied)
- **WebFetch**: Domain-level controls for HTTP access
- **MCP Tools**: Model Context Protocol tool invocations, matched as `mcp__<server>__<tool>`

### Configuration Locations

Permissions are configured at multiple levels with cascading precedence:

1. **Enterprise managed settings** (highest priority): `~/.claude/managed-settings.json`
2. **User settings**: `~/.claude/settings.json`
3. **Project settings**: `.claude/settings.json` and `.claude/settings.local.json`
4. **CLI flags**: `--allowedTools` and `--disallowedTools` for session-level overrides

Example `settings.json`:

```json
{
  "permissions": {
    "allowedTools": ["Read", "Write(src/**)", "Bash(git *)", "Bash(npm *)"],
    "deny": ["Read(.env*)", "Write(production.config.*)", "Bash(rm *)", "Bash(sudo *)"]
  }
}
```

### Permission Modes

- **Default (interactive)**: Each potentially unsafe tool call requires user approval
- **Accept edits**: Auto-approves file edits but prompts for bash commands
- **YOLO mode** (`--dangerously-skip-permissions`): Bypasses all permission checks entirely. A study cited by eesel AI revealed that 32% of developers using this flag encountered at least one unintended file modification.

### Hooks System

Hooks provide deterministic control over Claude Code's lifecycle through user-defined shell commands:

| Hook Event | Trigger Point | Control Capability |
|---|---|---|
| `PreToolUse` | Before tool execution | Exit 0 to proceed, Exit 2 to block (with reason via stderr) |
| `PostToolUse` | After tool completion | Can add context, trigger follow-up actions |
| `UserPromptSubmit` | Before prompt processing | Can inject additional context, validate requests |
| `SessionStart` | Session initialization | Environment setup, validation |
| `Stop` | Response completion | Cleanup, notifications |
| `Notification` | When Claude sends notifications | Custom notification routing |

Hooks receive JSON data via stdin containing session information and can return structured JSON for fine-grained control:

```json
{ "continue": true, "stopReason": "string", "suppressOutput": true }
```

Hooks also support MCP tool matching with patterns like `mcp__<server>__<tool>`.

### Known Security Vulnerabilities (Patched)

Historical vulnerabilities highlight the importance of keeping Claude Code updated:

- **Claude Code <=1.0.119**: Symlink bypass in permission deny rules (Oct 2025)
- **Claude Code <1.0.105**: Confirm prompt bypass (Sept 2025)
- **Claude Code <1.0.20**: Command parsing bypass (Aug 2025)
- **npm supply chain attack** (Aug 2025): Malicious `nx` package postinstall script targeted Claude Code config files

---

## 2. Claude Code Native Sandboxing (Deep Dive)

### Architecture Overview

Claude Code's native sandboxing creates two security boundaries enforced at the OS level:

1. **Filesystem isolation**: Controls which directories can be read/written
2. **Network isolation**: Controls which domains/hosts can be accessed

Both boundaries are enforced using OS-level primitives, covering not just Claude Code's direct interactions but all scripts, programs, and subprocesses spawned by commands.

### Platform Implementations

**macOS (Seatbelt)**:
- Uses Apple's `sandbox-exec` with dynamically generated Seatbelt profiles
- Profiles specify allowed read/write paths using regex patterns converted from gitignore-style globs (via `globToRegex` function)
- Network restricted to specific localhost ports where proxies listen
- Violation monitoring via `startMacOSSandboxLogMonitor` which streams macOS system logs, filtering sandbox violations from the current session
- Works out of the box with no additional dependencies
- Single native tool (unlike Linux which requires multiple: bubblewrap, socat, seccomp)

**Linux (Bubblewrap)**:
- Uses bubblewrap (`bwrap`) for containerization with bind mounts
- Directories marked as read-only or read-write based on configuration
- Network namespace is removed entirely from the bubblewrap container
- All network traffic routed through Unix domain sockets (via `socat`) to host-side proxies
- Pre-generated seccomp BPF filters for x86-64 and ARM architectures (~104 bytes each, stored in `vendor/seccomp/x64/` and `vendor/seccomp/arm64/`)
- Violation detection requires `strace` (bubblewrap does not provide built-in violation reporting)
- Requires: `bubblewrap`, `socat`, `ripgrep`

**WSL2**: Uses bubblewrap, same as Linux. WSL1 is not supported (bubblewrap requires kernel features only available in WSL2).

### Filesystem Isolation Model

- **Read access** (deny-only pattern): Full read access by default. Users deny specific sensitive paths (e.g., `~/.ssh`, `~/.aws`). Empty deny list = full read access.
- **Write access** (allow-only pattern): No write access by default. Must explicitly allow directories (e.g., `.`, `/tmp`). `denyWrite` takes precedence over `allowWrite`.

### Network Isolation Model

All network access is denied by default. Traffic flows through a dual proxy architecture:

- **HTTP/HTTPS proxy**: Runs on localhost (macOS) or Unix domain socket (Linux)
- **SOCKS5 proxy**: Handles other TCP traffic with same domain enforcement
- Domain allowlists control which hosts are reachable
- New domain requests trigger interactive permission prompts
- Custom proxy support enables organizations to implement HTTPS inspection, logging, and integration with existing security infrastructure (e.g., mitmproxy for traffic inspection)

### Sandbox Modes

1. **Auto-allow mode**: Sandboxed bash commands auto-execute without prompting. Commands that cannot be sandboxed fall back to regular permission flow. Explicit ask/deny rules are always respected.
2. **Regular permissions mode**: All bash commands go through standard permission flow even when sandboxed.

Both modes enforce the same filesystem and network restrictions. Auto-allow mode works independently of the permission mode setting -- sandboxed bash commands run automatically even when file edit tools would normally require approval.

### Escape Hatch

An intentional escape mechanism exists: when commands fail due to sandbox restrictions, Claude can retry with `dangerouslyDisableSandbox` parameter, which goes through normal permission flow. Disable with `"allowUnsandboxedCommands": false`.

### Security Limitations

- Network filtering operates at domain level only; traffic content is not inspected
- Broad domains like `github.com` can enable data exfiltration
- Domain fronting can bypass network filtering
- `allowUnixSockets` can grant access to powerful system services (e.g., `/var/run/docker.sock` grants effective host access)
- Overly broad filesystem write permissions can enable privilege escalation (writing to `$PATH` directories, `.bashrc`, `.zshrc`)
- `enableWeakerNestedSandbox` mode (for Docker) considerably weakens security -- should only be used when the Docker container itself enforces additional isolation

### Performance Impact

Anthropic reports sandboxing reduces permission prompts by **84%** in internal usage, with minimal performance overhead on filesystem operations.

### Open Source Package

The sandbox runtime is published as `@anthropic-ai/sandbox-runtime` (version 0.0.28, Apache-2.0 license) and can sandbox arbitrary processes, agents, and MCP servers:

```bash
npx @anthropic-ai/sandbox-runtime <command-to-sandbox>
```

**Project structure**:
- `sandbox-manager.ts` -- SandboxManager orchestrator
- `sandbox-config.ts` -- Configuration schema and validation
- `sandbox-network.ts` -- HTTP/SOCKS5 proxy servers
- `macos-sandbox-utils.ts` -- macOS sandbox-exec wrapper
- `linux-sandbox-utils.ts` -- Linux bubblewrap wrapper
- `generate-seccomp-filter.ts` -- BPF filter generation

**Key API**:
- `SandboxManager.initialize(config)` -- Starts proxy servers
- `SandboxManager.wrapWithSandbox(command)` -- Wraps commands with restrictions
- `SandboxManager.reset()` -- Cleanup operations

Source: [github.com/anthropic-experimental/sandbox-runtime](https://github.com/anthropic-experimental/sandbox-runtime)

---

## 3. Anthropic Official Recommendations

### Layered Security Approach

Anthropic recommends treating Claude Code's configuration settings as one layer in a defense-in-depth strategy:

1. **OS-level sandboxing** (bubblewrap/seatbelt) as the primary enforcement boundary
2. **Permission rules** (allow/deny patterns) for tool-level access control
3. **Docker/DevContainers** as an additional isolation layer
4. **Enterprise policies** via managed settings for organization-wide enforcement

### Cloud Session Security (Claude Code on the Web)

For cloud environments, each session runs in an isolated, Anthropic-managed VM with:

- Network access limited by default, configurable to disable or allow specific domains
- Authentication through a secure proxy using scoped credentials inside the sandbox
- Git push restricted to the current working branch
- Proxy validates authentication tokens, branch names, and repository destinations
- Sensitive credentials (git credentials, signing keys) remain outside the sandbox
- All operations logged for compliance and audit

### Enterprise Best Practices

1. **Deny-by-default permissions**: Start with minimal access, expand as needed
2. **Never run as root**: AI agents should never have admin privileges
3. **Credential management**: API key rotation, centralized secrets management, no plaintext `.env` files. Use vaults for secrets management.
4. **Regular auditing**: Monthly review of managed-settings.json for drift
5. **Keep Claude Code updated**: Security patches are frequent (multiple CVEs patched in 2025)
6. **Monitor logs**: Review sandbox violation attempts
7. **Environment-specific configs**: Different sandbox rules for dev vs production
8. **Test configurations**: Verify sandbox settings in safe environment before production rollout
9. **External SAST/DAST tools**: Implement as additional security layer
10. **CASB/DNS controls**: Enforce technical controls to block consumer Claude use for work content

### Encryption and Compliance

- TLS 1.2+ for all network requests in transit
- AES-256 encryption for stored logs, model outputs, and files at rest
- SAML 2.0 and OIDC-based SSO for enterprise
- BYOK (Bring Your Own Key) support arriving in early 2026
- Enterprise contracts exclude training on customer content

### DevContainer Recommendation

The official security documentation recommends DevContainers for isolation:

- Claude Code can only access mounted project files
- `--dangerously-skip-permissions` runs safely within container isolation
- Host machine remains protected from prompt injection attacks

---

## 4. OS-Level Sandboxing Primitives

### Bubblewrap (Linux)

Bubblewrap (`bwrap`) is a lightweight sandboxing tool that uses Linux kernel features:

- **User namespaces**: Process isolation without root privileges
- **Mount namespaces**: Custom filesystem views with bind mounts
- **Network namespaces**: Complete network isolation (can remove network entirely)
- **PID namespaces**: Process ID isolation
- **Cgroups**: Resource limiting

Claude Code uses bubblewrap to create isolated execution environments where:
- System directories are mounted read-only
- Only specified workspace directories are writable
- Network namespace is removed entirely, forcing all traffic through Unix socket proxies

A practical blog post by Senko Rasic demonstrates a minimal bubblewrap script for AI agents:
- Read-only bindings for system directories (bin, lib, usr)
- Selective write access via bind mounts (only /tmp, ~/.claude/, project dir)
- API credentials injected via file descriptor redirection without persisting
- Recommends using `strace` to trace syscalls for identifying required bindings

### Seatbelt (macOS)

Apple's Seatbelt framework (`sandbox-exec`) provides:

- Dynamic security profile generation at runtime
- Fine-grained restrictions on file access, network, and system calls
- Regex-based path matching for allow/deny rules
- Violations logged to macOS system log for monitoring
- No additional dependencies needed (built into macOS)

### Seccomp-BPF

Seccomp-BPF provides system call filtering at the kernel level:

- **Immutable**: Once enabled, cannot be disabled; inherited by all child processes
- **TOCTOU-resistant**: BPF programs cannot dereference pointers, preventing time-of-check/time-of-use attacks
- **Not a complete sandbox**: Designed as a building block for sandbox developers
- Claude Code's sandbox-runtime includes pre-generated seccomp BPF filters for x86-64 and ARM
- Docker's default seccomp profile blocks ~44 dangerous syscalls

**Limitation**: Seccomp-BPF cannot apply fine-grained filters to `io_uring` operations -- it is "all or nothing" for that interface.

### Landlock LSM

Landlock is a Linux Security Module (since kernel 5.13) based on eBPF that enables unprivileged process self-sandboxing:

- **Unprivileged**: Processes can restrict themselves without root access
- **Stackable**: Layers on top of existing access controls (DAC, other LSMs)
- **Immutable**: Once applied, restrictions cannot be removed (only added to). Every new thread from `clone(2)` inherits Landlock domain restrictions.
- **Fine-grained**: Controls file read/write/execute, directory creation/removal, network TCP bind/connect
- **Minimal attack surface**: Does not interfere with other access-controls, only adds more restrictions

Key tools:
- **landrun** ([GitHub](https://github.com/Zouuup/landrun)): CLI wrapper for Landlock v5 with filesystem and TCP restrictions. No root, no containers, no SELinux/AppArmor configs. Lightweight and auditable.
- **island** ([GitHub](https://github.com/landlock-lsm/island)): High-level Landlock wrapper and policy manager from the official Landlock project.

**OverlayFS interaction**: Landlock treats OverlayFS layers and merge hierarchies as standalone. A policy restricting an OverlayFS layer does NOT restrict the merged hierarchy, and vice versa. Access rights can propagate with bind mounts but not with OverlayFS.

### io_uring Security Concerns

`io_uring` (Linux kernel 5.1+) presents a significant security gap for all sandboxing approaches:

- Bypasses traditional system call interface for I/O operations
- Google reported **60% of kernel exploits** in their 2022 bug bounty targeted `io_uring` ($1M in bounties paid for io_uring alone)
- The **"Curing" rootkit** (April 2025, ARMO): Demonstrated complete bypass of syscall-based security tools including Falco, Tetragon, and Microsoft Defender in default configurations. The rootkit communicates with C2 servers and executes commands without making any system calls.
- The fundamental problem: io_uring is impossible to apply fine-grained seccomp filters to because the actual functionality is opaque to BPF. Management calls look innocent; malicious activity happens in the shared queues.

**Google's response**: Disabled io_uring in Chrome OS, Android apps (via seccomp-bpf), GKE AutoPilot, and production servers.

**Mitigations**:
1. Disable io_uring entirely (possible since kernel 6.6)
2. Use KRSI (Kernel Runtime Security Instrumentation) with eBPF programs attached to LSM hooks
3. LSM hooks are part of the kernel's internal enforcement logic, much harder to bypass than syscall monitoring

### nsjail

nsjail (by Google) combines multiple Linux security features:
- Linux namespaces (PID, mount, network, user)
- Seccomp-BPF system call filtering
- Cgroups resource limits
- Lightweight alternative to full virtualization
- Used in Google production environments

---

## 5. Filesystem Virtualization Approaches

### OverlayFS

OverlayFS (Linux kernel 3.18+) provides kernel-level copy-on-write filesystem stacking:

**How it works**:
- A read-only **lower layer** contains the base filesystem
- A writable **upper layer** captures all modifications
- A **merged view** presents a unified directory to the process
- Original files are never modified; changes are isolated in the upper layer
- Copy-up on first write: reads data and metadata from lower, copies to upper, then presents upper copy

**Use cases for agent sandboxing**:
- **Ephemeral sandboxes**: Mount workspace read-only as lower, tmpfs as upper. All changes vanish on restart.
- **Memory efficiency**: AI agent sandboxes using OverlayFS report **75% reduction in memory usage** (Blaxel) because the base system is not fully loaded into RAM. A base Next.js image would require 4GB in memory; with OverlayFS, only modified files consume memory.
- **Container foundation**: Docker's `overlay2` storage driver uses OverlayFS for layered images, allowing multiple containers to share common base image layers without duplication.
- **Rollback**: Changes can be inspected, selectively applied, or discarded entirely.
- **CI/CD**: Base environments with overlay per build.

**Practical tools**:
- **poof** ([GitHub](https://github.com/Jarred-Sumner/poof)): Ephemeral filesystem isolation via OverlayFS. Commands run in isolated environment where changes never affect host. Changes can persist to a directory or be reviewed selectively. Linux only. Note: designed for reversibility, not full security sandboxing (commands can still make network requests, read env vars, access hardware).

**Security considerations**:
- Firejail had a vulnerability (race condition in OverlayFS check) allowing creation of arbitrary files
- Landlock policies on OverlayFS layers do not propagate to merged hierarchy
- Requires root for mount (or user namespace with unprivileged mounts)

### FUSE Filesystems

FUSE (Filesystem in Userspace) enables custom filesystem implementations without kernel modification:

**Agent-specific FUSE filesystems**:

1. **AgentFS** ([GitHub](https://github.com/tursodatabase/agentfs)) by Turso:
   - SQLite-backed agent state as a POSIX filesystem
   - Every file operation, tool call, and state change recorded in SQLite for audit/debugging/compliance
   - **Filesystem-level copy-on-write isolation** that is system-wide and cannot be bypassed (unlike git worktree isolation which is purely conventional)
   - FUSE mount on Linux, NFS mount on macOS
   - Built with Rust (fuser crate)
   - Architecture: Agent -> Linux kernel VFS -> FUSE module -> AgentFS userspace process

2. **"FUSE is All You Need" pattern** (Jakob Emmerling):
   - Uses FUSE as a UI/interface layer between agent and backend data
   - Agent runs in sandbox, sees mounted filesystem representing backend resources
   - Replaces multiple search/write/list tools with a single Bash tool -- reduces tool space significantly
   - Agents can chain operations intuitively using Unix paradigms
   - Think of filesystem as "a frontend for the agent"

3. **SecretFS** ([GitHub](https://github.com/obormot/secretfs)):
   - Security-focused FUSE filesystem for fine-grained access control to secrets
   - Mirrors directory tree into read-only FUSE volume
   - ACLs restrict access by specific process, command line, user/group, and time limit
   - Supports Linux, macOS, and FreeBSD

4. **Agentic File System (AFS)** -- Formal abstraction:
   - Multi-agent namespace isolation via sandboxed processes and scoped ACLs
   - SystemFS provides uniform namespace with fine-grained access control
   - Full context traceability and accountability
   - Designed to replace fragmented practices (prompt injection, RAG, ad hoc tool integration)

---

## 6. Network-Level Sandboxing

### Domain-Level Proxy Filtering (Claude Code Approach)

Claude Code's network isolation uses a proxy-based architecture:

- All outbound traffic forced through HTTP/SOCKS5 proxies
- Proxies enforce domain allowlists
- On Linux, network namespace removal ensures all traffic routes through Unix domain sockets
- On macOS, Seatbelt profiles restrict socket connections to proxy ports

**Limitations**:
- Domain-level filtering only (no payload inspection by default)
- Domain fronting can bypass restrictions
- Broad domain allowlists (e.g., `github.com`) create exfiltration risk

### Docker Sandbox Network Policies

Docker Sandboxes implement HTTP/HTTPS filtering proxies:

- Each sandbox has a dedicated proxy enforcing outbound policies
- HTTPS traffic subject to **MITM inspection** (proxy terminates TLS and re-encrypts with own CA)
- Sandbox container trusts proxy's CA certificate automatically
- Raw TCP/UDP connections blocked entirely
- All communication must go through HTTP proxy or remain within sandbox

### eBPF-Based Network Filtering

eBPF enables high-performance network filtering in the kernel:

- **dae** ([GitHub](https://github.com/daeuniverse/dae)): eBPF-based transparent proxy for traffic splitting. Direct traffic bypasses proxy forwarding with minimal performance loss and negligible resource consumption.

- **Transparent proxy with eBPF and Go**: Three eBPF programs work together:
  1. `cgroup/connect4`: Intercepts connect syscall, redirects to local proxy (transparent to client)
  2. `sockops`: Records source address/port on connection establishment
  3. `cgroup/getsockopt`: Retrieves original destination when proxy queries via getsockopt

  Performance: Adds approximately **1ms constant overhead** -- eBPF proves an excellent fit for transparent proxies.

- **Related eBPF security tools**:
  - **BPFJailer**: eBPF-based process jailing with mandatory access control
  - **Bombini**: eBPF-based security agent in Rust using Aya library, built on LSM BPF hooks
  - **bpfilter**: Translates filtering rules into optimized BPF programs

### WireGuard for Controlled Network Access

WireGuard provides encrypted tunnel-based network control:

- **Cryptokey Routing Table**: Associates public keys with allowed IPs -- functions as both routing table (sending) and access control list (receiving)
- Administrators can match on source IP and interface without complex firewall rules
- Useful for sandboxed environments where only specific endpoints should be reachable

**Tools**:
- **NetBird** ([GitHub](https://github.com/netbirdio/netbird)): WireGuard overlay network with granular access policies and SSO/MFA
- **Netmaker**: WireGuard network management with access controls for multi-environment setups

**Sandbox pattern** (from honeypot isolation): WireGuard as the only allowed communication path. iptables rules block all RFC1918 addresses while allowing WireGuard tunnel traffic first. Rule ordering is critical: WireGuard exception must come before RFC1918 blocks.

### DNS-Based Restrictions

- Restrict DNS resolution to prevent DNS-based data exfiltration
- Log all DNS queries for audit (timestamps, destinations, response codes)
- Combine with proxy-based filtering for defense in depth
- CoreDNS can selectively forward only allowed domains and return NXDOMAIN for everything else

---

## 7. Cloud Sandbox Platforms for AI Agents

### E2B (Enterprise AI Agent Cloud)

**Architecture**: Cloud-based sandboxes powered by Firecracker microVMs

| Property | Value |
|---|---|
| Isolation | Firecracker microVM with dedicated kernel |
| Cold start | ~150ms (pre-warmed snapshots for near-instant) |
| Memory overhead | 3-5 MB per instance |
| Throughput | Up to 150 microVMs/second/host |
| Session duration | Up to 24 hours |
| SDKs | Python, JavaScript, MCP server |
| License | Apache-2.0 (~8,900 GitHub stars) |
| Integrations | LangChain, OpenAI, Anthropic, Claude Code via MCP |

Defense-in-depth: Firecracker's companion "jailer" process sets up cgroups and namespaces to isolate the VMM process itself before dropping privileges.

### Daytona

**Focus**: Fastest sandbox provisioning, agent-agnostic

| Property | Value |
|---|---|
| Cold start | Sub-90ms (fastest in market) |
| Isolation | Docker by default; Kata Containers available |
| Statefulness | Filesystem, env vars, process memory persist |
| Special features | Built-in LSP support |
| Trade-off | Docker default weaker than microVMs |

Pivoted in early 2025 from CDEs to AI agent infrastructure. Agent-agnostic architecture works with any AI model or agent framework.

### Fly.io Sprites

**Philosophy**: "Ephemeral sandboxes are obsolete" -- agents want persistent computers

| Property | Value |
|---|---|
| Isolation | Firecracker microVMs with KVM hardware isolation |
| Cold start | Under 1 second |
| Persistence | Checkpoint/restore entire environments |
| Billing | Per-second CPU and memory |
| Default tools | Comes with Claude installed |
| Launch date | January 2026 |

Quote from Thomas Ptacek: "Agents don't want containers. They don't want 'sandboxes'. They want computers."

Zero Trust approach: even if a vulnerability is exploited, attacker remains trapped within the microVM. Cannot access host filesystem or internal network.

Open-source local version planned for near-term release.

### Modal

**Focus**: Python-first serverless compute with gVisor isolation

| Property | Value |
|---|---|
| Autoscaling | Zero to 20,000+ concurrent containers |
| Isolation | gVisor user-space kernel (optimized for Python ML) |
| GPU support | T4 through H200 |
| Snapshot | Save/restore sandbox state |
| Cold starts | Sub-second (but 2-5s for some ops) |
| Pricing | $0.047/vCPU-hour (managed-only) |
| Users | Lovable, Quora (millions of snippets daily) |

Limitations: No BYOC/on-prem. SDK-defined images create vendor lock-in. Optimizes heavily for Python.

### Northflank

**Focus**: Multi-isolation-technology platform, enterprise-grade

| Property | Value |
|---|---|
| Isolation | Kata Containers (Cloud Hypervisor) primary, gVisor fallback |
| Session duration | Unlimited (unique advantage) |
| OCI compatible | Any standard container image |
| BYOC | AWS, GCP, Azure, bare-metal |
| Scale | 2M+ isolated workloads monthly (since 2019) |
| Pricing | $0.01667/vCPU-hour (65% cheaper than Modal) |

Only AI sandbox platform offering both Kata Containers AND gVisor. Active open-source contributor to Kata Containers, QEMU, containerd, Cloud Hypervisor.

On infrastructure without nested virtualization, falls back to gVisor for syscall-level isolation.

### Microsandbox

**Focus**: Self-hosted, maximum control

| Property | Value |
|---|---|
| Isolation | Hardware-level via libkrun (KVM-based library) |
| Boot time | Under 200ms |
| OCI compatible | Standard container images |
| Hosting | Self-hosted only |
| License | Apache-2.0 (~3,300 GitHub stars) |
| Launch | May 2025 (v0.1.0) |
| MCP integration | Yes |

Designed to solve the security-speed-control trade-off: hardware isolation + container-like speed + full self-hosted control. libkrun is a library-based virtualization solution providing KVM VMs with minimal overhead.

### GitHub Codespaces

| Property | Value |
|---|---|
| Isolation | Container-based with VM backing |
| Integration | Tight GitHub repository integration |
| Agent use | Effective filesystem isolation |
| Limitation | No network-level isolation by default |
| Cost | Paid (per-hour) |

Recommended by GitHub Security Lab for running AI agent frameworks. Agent is jailed within container and cannot access local drives. However, agents with network access can still exfiltrate data from the repository.

### Koyeb

Ephemeral, fully isolated environments for AI agents and code generation. Managed platform with edge deployment capabilities.

---

## 8. Container and MicroVM Isolation Technologies

### Technology Comparison

| Technology | Isolation Level | Boot Time | Memory Overhead | Security Strength | Compatibility |
|---|---|---|---|---|---|
| **Docker Containers** | Process (shared kernel) | Milliseconds | Minimal (~10MB) | Weakest -- kernel exploit = escape | Full Linux |
| **gVisor** | Syscall interception (user-space kernel) | Milliseconds | ~50MB | Medium -- must exploit Go Sentry + host kernel | ~70-80% syscalls |
| **Firecracker** | Hardware (dedicated kernel) | ~125ms | <5 MiB/VM | Strongest -- must escape guest kernel + hypervisor | Full (own kernel) |
| **Kata Containers** | Hardware (VMM-orchestrated) | ~200ms | ~30-50MB | Strongest + K8s native | Full (own kernel, OCI) |
| **Docker Sandboxes** | MicroVM (Firecracker-based) | ~125ms | ~5MB | Strong -- private Docker daemon per VM | Full |

### Standard Docker Containers

Containers share the host kernel. A single kernel exploit compromises every container on the host. The Linux kernel averages 300+ CVEs annually.

**Verdict**: Suitable only for trusted, vetted code in single-tenant environments.

### gVisor (User-Space Kernel)

gVisor interposes a user-space kernel ("Sentry") between container and host kernel:

- System calls intercepted via ptrace or KVM before reaching host kernel
- Only tiny, audited subset of syscalls reach host kernel
- Used in production at Google (Cloud Run, App Engine, Cloud Functions)
- **10-30% overhead on I/O-heavy workloads**, minimal on compute-heavy tasks
- Does not require hardware virtualization -- runs on any Linux host
- Uses optimized seccomp-BPF filters internally with in-sandbox caching

**Limitations**: No systemd, no Docker-in-Docker, some syscall compatibility issues with obscure/new syscalls.

### Firecracker MicroVMs

Written in ~50,000 lines of Rust, Firecracker emulates exactly 5 virtual devices:

- **Density**: Up to 150 VMs/second/host. AWS Lambda runs millions in production.
- **Security**: Companion "jailer" process sets up cgroups and namespaces around VMM itself
- **No GPU support, no pause/resume, no live migration**
- Used by: E2B, Docker Sandboxes, Fly.io Sprites, AWS Lambda, AWS Fargate

### Kata Containers

OCI-compliant -- each container runs in its own VM while keeping Docker/K8s workflows intact:

- Supports multiple VMMs: Firecracker, Cloud Hypervisor, QEMU
- Kubernetes RuntimeClass integration
- Agent Sandbox project (CNCF) added Kata Containers support at KubeCon NA 2025

### Docker Sandboxes (Docker Desktop)

- MicroVM isolation (not regular containers): each sandbox in dedicated microVM
- Private Docker daemon per sandbox
- Network allow/deny lists
- Workspace syncing at matching absolute paths
- **Platform**: macOS and Windows (microVM); Linux legacy container mode only
- Sandboxes don't appear in `docker ps` -- use `docker sandbox ls`

### Kubernetes Agent Sandbox (CNCF)

Open-source Kubernetes controller by Google (presented KubeCon NA 2025):

- **Isolation**: gVisor or Kata Containers (backend-agnostic)
- **CRDs**: Sandbox, SandboxTemplate, SandboxClaim
- **Warm pools**: Pre-warmed pods for sub-second startup (90% improvement over cold starts)
- **Python SDK**: High-level API for lifecycle management
- **Integration**: ADK, LangChain, and other agent frameworks

---

## 9. Multi-Agent Orchestration Sandboxing

### Claude Code Agent Teams (Official, Experimental)

Claude Code supports coordinating multiple instances as a team:

- One session acts as team lead, coordinating work and assigning tasks
- Teammates work independently in their own context windows
- Direct inter-agent messaging
- Enabled via `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1`
- Best for: research/review, new modules, debugging with competing hypotheses, cross-layer coordination

### Third-Party Multi-Agent Tools

| Tool | Isolation Method | Key Feature |
|---|---|---|
| **claude-squad** | tmux sessions + git worktrees | Multiple agents on separate branches |
| **ccswarm** (Rust) | Git worktree isolation | 93% token savings via MessageBus, crash recovery, session persistence |
| **claude-flow** | MCP protocol | Spec-first with ADR enforcement, distributed swarm intelligence |
| **agentrooms** | @mention routing | Specialized agents, collaborative workflows |
| **wshobson/agents** | Plugin isolation | 112 agents, 16 orchestrators, 146 skills, 79 tools in 73 plugins |
| **Gas Town** | Agent hierarchy | "Mayor" agent breaks down tasks, spawns designated agents |
| **Multiclaude** | Brownian ratchet | Auto-merge if CI passes; singleplayer or multiplayer modes |

### Common Isolation Patterns

1. **Git worktrees**: Each agent works on a separate branch/worktree. Prevents merge conflicts but does NOT enforce filesystem boundaries. As AgentFS notes: "nothing prevents an agent from modifying files outside its worktree."
2. **tmux sessions**: Terminal-level isolation per agent. No true sandbox.
3. **Process-level**: Each Claude Code instance runs as separate process with own context window. Fresh memory at step boundaries prevents context contamination.
4. **Plugin isolation**: Completely isolated plugins with own agents, commands, and skills.

### Design Pattern Consensus

Multi-agent systems are "an expensive and experimental way to complete larger projects" -- they "aren't for everyone and don't make sense for 95% of agent-assisted development tasks" (Shipyard). But when applicable:

- **Context isolation**: Prevents cross-contamination from debugging output
- **Workspace isolation**: Each agent gets its own filesystem area
- **Artifact passing**: Structured mechanism for inter-step communication
- **Shared read-only areas**: Common reference data accessible to all agents

---

## 10. Applicability to Wave

### Wave's Current Workspace Model

Wave's `internal/workspace/workspace.go` implements workspace isolation through:

- **Per-step directories**: Each pipeline step gets `<baseDir>/<pipelineID>/<stepID>/`
- **Mount-based copying**: Source directories recursively copied into workspace (NOT bind-mounted, NOT symlinked)
- **Read-only mode**: Mounts can be set to `readonly` via `chmod 0555`
- **Artifact injection**: Previous step outputs copied into `artifacts/` subdirectory
- **Skip lists**: Large/tool directories excluded (`node_modules`, `.git`, `vendor`, `.wave`, `.claude`, `__pycache__`, `.venv`, `dist`, `build`, `.next`, `.cache`)
- **File size limits**: Files >10MB skipped during copy
- **Symlink resolution**: Applied before copy to avoid directory symlink crashes

Wave's `internal/security/` provides:
- **Path traversal prevention**: Validates paths against approved directories, blocks `..` sequences and encoded variants
- **Symlink detection**: Blocks symbolic links (configurable via `AllowSymlinks`)
- **Input sanitization**: Prompt injection pattern detection (7 regex patterns)
- **Audit logging**: Security events with severity levels, credential scrubbing

### Gap Analysis

| Capability | Wave Current | What Sandbox Could Add |
|---|---|---|
| Filesystem isolation | Copy-based (chmod 0555) | OS-level enforcement (bubblewrap/Landlock) |
| Network isolation | None | Proxy-based domain filtering |
| Process isolation | None (shared host) | Namespaces, cgroups, or microVMs |
| Syscall filtering | None | seccomp-BPF or Landlock |
| Resource limits | None | Cgroups (CPU, memory, disk I/O) |
| Artifact integrity | File copy | OverlayFS or FUSE for COW isolation |
| Cross-step contamination | Separate directories | True namespace isolation |
| Prompt injection defense | Regex patterns | Sandbox prevents execution even if injection succeeds |

### Recommended Integration Approaches

**Tier 1 -- Immediate (Low Effort, High Impact)**:

1. **Enable Claude Code native sandbox for all subprocess invocations**: When Wave launches `claude` CLI processes, ensure sandboxing is active. Set `allowUnsandboxedCommands: false` to eliminate escape hatch. This is the single highest-impact change.

2. **Generate per-persona `settings.json` files**: Map Wave persona definitions to Claude Code `allowedTools` and `deny` rules. The `navigator` persona should have `Bash(*)` denied. The `implementer` should have `Write` allowed only within workspace. Write these to the step workspace before launching Claude Code.

3. **Register hooks for contract enforcement**: Use `PostToolUse` hooks to validate outputs against Wave contracts before step completion. Use `PreToolUse` hooks to enforce persona-specific tool restrictions.

4. **Use `@anthropic-ai/sandbox-runtime` for MCP server sandboxing**: If Wave uses MCP servers, wrap them with the sandbox-runtime to prevent malicious tool execution.

**Tier 2 -- Medium Term (Moderate Effort)**:

5. **Landlock integration in Go**: Since Wave is a Go binary on Linux, use Landlock syscalls directly to restrict each step's process tree. The [go-landlock](https://github.com/landlock-lsm/go-landlock) library provides a Go API. This gives unprivileged, kernel-enforced filesystem and network restrictions without containers.

6. **OverlayFS workspaces**: Replace the current `copyRecursive` workspace creation with OverlayFS mounts. Lower layer = read-only project snapshot. Upper layer = per-step writable area. Benefits: dramatically faster workspace creation, lower disk usage (75% memory reduction per Blaxel), atomic rollback by discarding upper layer. Note: requires root or user namespace with unprivileged mounts.

7. **Network proxy per pipeline**: Run a per-pipeline HTTP/SOCKS5 proxy that enforces domain allowlists based on persona permissions. Route all step subprocess traffic through this proxy via environment variables (`HTTP_PROXY`, `HTTPS_PROXY`).

8. **Resource limits via cgroups**: Prevent any single step from consuming excessive CPU, memory, or disk I/O. Can be applied per-step or per-pipeline.

**Tier 3 -- Long Term (Higher Effort, Strongest Isolation)**:

9. **MicroVM per step**: Use Firecracker or microsandbox (libkrun) to run each pipeline step in a dedicated microVM. Provides hardware-level isolation between steps. Most suitable for multi-tenant or untrusted pipeline execution.

10. **Kubernetes Agent Sandbox integration**: For cloud/enterprise deployments, use the CNCF Agent Sandbox controller with warm pools for sub-second step startup with Kata Container isolation.

---

## 11. Comparison Matrix

### All Approaches Compared

| Approach | Security Level | Attack Vectors Mitigated | Perf Overhead | Ease of Setup | Cross-Platform | Cost | Maturity |
|---|---|---|---|---|---|---|---|
| **Claude Code Permissions** | Low-Medium | Accidental misuse, broad access | None | Trivial | All (built-in) | Free | Production |
| **Claude Code Hooks** | Medium | Deterministic policy enforcement | Negligible | Easy | All (built-in) | Free | Production |
| **Claude Code Native Sandbox** | High | FS escape, net exfiltration, prompt injection | Minimal | Easy (macOS OOB) | macOS, Linux, WSL2 | Free | Production |
| **`sandbox-runtime` (npm)** | High | Same + arbitrary process sandboxing | Minimal | Easy | macOS, Linux | Free (Apache-2.0) | Beta |
| **Landlock LSM** | High | FS escape, net connections, priv escalation | Near-zero | Moderate | Linux 5.13+ only | Free | Production (kernel) |
| **seccomp-BPF** | Medium-High | Syscall attacks (not io_uring) | Near-zero | Moderate | Linux only | Free | Production (kernel) |
| **OverlayFS Workspaces** | Medium | FS corruption, cross-step contamination | Low | Moderate (root needed) | Linux only | Free | Production (kernel) |
| **FUSE (AgentFS/SecretFS)** | Medium-High | Fine-grained file access, audit | Low-Medium | Moderate | Linux, macOS (NFS) | Free (varies) | Emerging |
| **Docker Containers** | Medium | Process escape (not kernel) | Low | Easy | All major OS | Free | Production |
| **Docker Sandboxes** | High | Kernel exploits, process escape, net | Low-Medium | Easy (Docker Desktop) | macOS, Windows | Free (w/ Desktop) | Production |
| **gVisor** | High | Kernel attack surface (10-30% I/O) | 10-30% I/O | Moderate | Linux only | Free | Production (Google) |
| **Firecracker** | Very High | Hardware isolation, all kernel attacks | Low (5MB, 125ms) | Complex (KVM req) | Linux only (KVM) | Free (Apache-2.0) | Production (AWS) |
| **Kata Containers** | Very High | Same + K8s native orchestration | Low-Med (200ms) | Moderate (K8s req) | Linux only (KVM) | Free | Production |
| **E2B** | Very High | Full microVM isolation | Low (150ms) | Easy (SDK) | Cloud only | Paid | Production |
| **Daytona** | High | Docker + fast provisioning | Very Low (90ms) | Easy (SDK) | Cloud + self-host | Paid + OSS | Production |
| **Fly.io Sprites** | Very High | KVM isolation + persistence | Low (<1s) | Easy (API) | Cloud only | Paid (per-sec) | New (Jan 2026) |
| **Northflank** | Very High | Multi-tech (Kata + gVisor) | Low | Easy (any OCI) | Cloud + BYOC | Paid | Production (2019) |
| **Microsandbox** | Very High | Hardware isolation (libkrun) | Low (<200ms) | Moderate (self-host) | Linux (self-host) | Free (Apache-2.0) | Emerging (2025) |
| **Modal** | High | gVisor + serverless | Variable (2-5s cold) | Easy (Python SDK) | Cloud only | Paid | Production |
| **K8s Agent Sandbox** | Very High | gVisor/Kata + warm pools | Low (sub-sec w/ pools) | Complex (K8s) | Linux (K8s) | Free (OSS) | Emerging (2025) |
| **GitHub Codespaces** | Medium-High | FS isolation, not network | Low | Easy (GitHub) | Cloud only | Paid (per-hour) | Production |
| **WireGuard Tunnels** | Medium (net only) | Net exfiltration, unauthorized access | Near-zero | Moderate | All major OS | Free | Production |
| **eBPF Transparent Proxy** | Medium-High (net) | Net exfiltration + inspection | ~1ms/conn | Complex | Linux only | Free | Production |
| **DevContainers** | Medium-High | FS isolation + tooling | Low | Easy (VS Code) | All major OS | Free | Production |
| **nsjail** | High | Combined NS + seccomp + cgroups | Low | Moderate | Linux only | Free | Production (Google) |

### Wave-Specific Recommendation Priority

| Priority | Approach | Why | Effort | Impact |
|---|---|---|---|---|
| 1 | Claude Code Native Sandbox | Immediate security uplift, zero code changes | Low | High |
| 2 | Per-persona permissions (settings.json generation) | Maps directly to Wave persona model | Low | High |
| 3 | Hooks for contract validation | Deterministic enforcement at step boundaries | Low-Medium | Medium |
| 4 | `sandbox-runtime` for MCP sandboxing | Protects against malicious MCP tools | Low | Medium |
| 5 | Landlock for Wave process isolation | Kernel-enforced, no containers, Go library | Medium | High |
| 6 | OverlayFS workspaces | Faster creation, lower disk, rollback | Medium | Medium |
| 7 | Network proxy per pipeline | Prevents data exfiltration between steps | Medium | High |
| 8 | Docker container per step | Full namespace isolation | Medium-High | High |
| 9 | MicroVM per step (Firecracker/microsandbox) | Maximum isolation for multi-tenant | High | Very High |

---

## 12. Recommendations

### For Wave Specifically

**Immediate actions (prototype phase)**:

1. **Enable Claude Code's native sandbox for all subprocess invocations.** When Wave launches `claude` CLI processes, ensure sandboxing is active with `allowedDomains` restricted to necessary APIs. Set `allowUnsandboxedCommands: false`. This requires no changes to Wave's Go code -- only passing the right environment/config to Claude Code.

2. **Generate per-step Claude Code configuration** that maps Wave persona permissions to Claude Code `allowedTools` and `deny` rules. The `navigator` persona should have `Bash(*)` denied. The `implementer` persona should have `Write` allowed only within workspace. Write a `.claude/settings.json` into each workspace before launching.

3. **Evaluate replacing `copyRecursive`** workspace creation with OverlayFS mounts on Linux. The current approach copies entire project trees per step; OverlayFS would provide faster setup, lower disk usage, and instant rollback.

**Medium-term (production hardening)**:

4. **Integrate Landlock** for Wave's own process tree. When Wave forks child processes for step execution, apply Landlock rules restricting filesystem access to the step workspace and network access to allowed endpoints. Use [go-landlock](https://github.com/landlock-lsm/go-landlock).

5. **Implement a per-pipeline network proxy** that enforces domain allowlists. This prevents any step from exfiltrating data to unauthorized endpoints, even if Claude Code's own sandbox is bypassed.

6. **Add resource limits via cgroups** to prevent any single step from consuming excessive CPU, memory, or disk I/O.

7. **Disable io_uring** in agent execution environments on kernel 6.6+ to close the syscall bypass blind spot.

**Enterprise/multi-tenant**:

8. For deployments where multiple users run pipelines on shared infrastructure, consider Firecracker microVMs, microsandbox (libkrun), or Kata Containers for per-step isolation. The Kubernetes Agent Sandbox controller provides a managed path.

### Industry-Wide Observations

1. **The sandbox landscape is consolidating around microVMs**: Firecracker-based solutions (E2B, Sprites, Docker Sandboxes) are becoming the standard for production AI agent isolation. Gartner predicts 40% of enterprise apps will have embedded agents by 2026 (up from <5% in early 2025). Startup Hub research shows sandboxed agents reduce security incidents by **90%**.

2. **OS-level sandboxing is underappreciated**: Anthropic's bubblewrap/seatbelt approach provides strong isolation without VM/container overhead. The open-sourcing of `sandbox-runtime` makes this accessible to all agent frameworks.

3. **Network isolation is as important as filesystem isolation**: Without both, sandboxes can be bypassed. The proxy-based approach (domain allowlists) has become the standard pattern.

4. **Stateful sandboxes are replacing ephemeral ones**: Fly.io Sprites and Daytona reflect a shift toward persistent agent environments that survive across interactions, with checkpoint/restore. This challenges Wave's ephemeral workspace model but may be worth evaluating for long-running pipelines.

5. **Kubernetes is becoming the orchestration layer**: Google's Agent Sandbox controller, with warm pools and multi-backend isolation, positions K8s as the enterprise standard for agent sandbox management.

6. **io_uring is a blind spot**: Most sandbox approaches do not adequately address io_uring bypass. Organizations should disable io_uring in agent execution environments.

7. **FUSE filesystems offer a novel interface pattern**: Rather than restricting tools, FUSE presents a controlled filesystem as the agent's workspace interface, reducing tool surface area while providing audit trails.

---

## 13. Sources

### Anthropic Official
- [Making Claude Code More Secure and Autonomous](https://www.anthropic.com/engineering/claude-code-sandboxing)
- [Sandboxing - Claude Code Docs](https://code.claude.com/docs/en/sandboxing)
- [Security - Claude Code Docs](https://code.claude.com/docs/en/security)
- [Hooks Reference - Anthropic Docs](https://docs.anthropic.com/en/docs/claude-code/hooks)
- [Hooks Guide - Anthropic Docs](https://docs.anthropic.com/en/docs/claude-code/hooks-guide)
- [Configure Permissions - Claude Code Docs](https://code.claude.com/docs/en/permissions)
- [Settings - Claude Code Docs](https://code.claude.com/docs/en/settings)
- [Agent Teams - Claude Code Docs](https://code.claude.com/docs/en/agent-teams)
- [SDK Permissions - Claude API Docs](https://docs.anthropic.com/en/docs/claude-code/sdk/sdk-permissions)
- [sandbox-runtime - GitHub](https://github.com/anthropic-experimental/sandbox-runtime)
- [@anthropic-ai/sandbox-runtime - npm](https://www.npmjs.com/package/@anthropic-ai/sandbox-runtime)

### Docker
- [Docker Sandboxes Overview](https://docs.docker.com/ai/sandboxes)
- [Docker Sandboxes: Run Claude Code Safely](https://www.docker.com/blog/docker-sandboxes-run-claude-code-and-other-coding-agents-unsupervised-but-safely/)
- [A New Approach for Coding Agent Safety](https://www.docker.com/blog/docker-sandboxes-a-new-approach-for-coding-agent-safety/)
- [Docker Sandboxes Network Policies](https://docs.docker.com/ai/sandboxes/network-policies/)
- [Configure Claude Code in Docker Sandbox](https://docs.docker.com/ai/sandboxes/claude-code/)
- [Docker Sandboxes + Claude Code: What Works, What Breaks](https://blog.arcade.dev/using-docker-sandboxes-with-claude-code)

### Cloud Sandbox Platforms
- [E2B - The Enterprise AI Agent Cloud](https://e2b.dev/)
- [E2B Breakdown - Memo](https://memo.d.foundation/breakdown/e2b)
- [Daytona - Secure Infrastructure for AI-Generated Code](https://www.daytona.io/)
- [Daytona - Sandboxing AI Development](https://www.daytona.io/dotfiles/sandboxing-ai-development-with-agent-agnostic-infrastructure)
- [Fly.io Sprites.dev](https://sprites.dev/) / [Simon Willison on Sprites](https://simonwillison.net/2026/Jan/9/sprites-dev/)
- [Fly.io - Code And Let Live](https://fly.io/blog/code-and-let-live/)
- [Modal - Top AI Code Sandbox Products](https://modal.com/blog/top-code-agent-sandbox-products)
- [Northflank - How to Sandbox AI Agents in 2026](https://northflank.com/blog/how-to-sandbox-ai-agents)
- [Northflank - Best Code Execution Sandbox](https://northflank.com/blog/best-code-execution-sandbox-for-ai-agents)
- [Northflank - Top AI Sandbox Platforms 2026](https://northflank.com/blog/top-ai-sandbox-platforms-for-code-execution)
- [Northflank - Kata vs Firecracker vs gVisor](https://northflank.com/blog/kata-containers-vs-firecracker-vs-gvisor)
- [Microsandbox - GitHub](https://github.com/zerocore-ai/microsandbox)
- [AI Sandboxes: Daytona vs Microsandbox](https://pixeljets.com/blog/ai-sandboxes-daytona-vs-microsandbox/)
- [Koyeb - Top Sandbox Platforms 2026](https://www.koyeb.com/blog/top-sandbox-code-execution-platforms-for-ai-code-execution-2026)
- [Better Stack - 10 Best Sandbox Runners 2026](https://betterstack.com/community/comparisons/best-sandbox-runners/)
- [GitHub Codespaces for AI Coding](https://elite-ai-assisted-coding.dev/p/sandboxing-for-ai-coding-with-github-codespaces)

### Kubernetes & Multi-Agent
- [Agent Sandbox - Kubernetes SIGs](https://github.com/kubernetes-sigs/agent-sandbox)
- [Agent Sandbox Documentation](https://agent-sandbox.sigs.k8s.io/)
- [Google - AI Agents on Kubernetes](https://opensource.googleblog.com/2025/11/unleashing-autonomous-ai-agents-why-kubernetes-needs-a-new-standard-for-agent-execution.html)
- [GKE Agent Sandbox Docs](https://docs.cloud.google.com/kubernetes-engine/docs/how-to/agent-sandbox)
- [Kata Containers + Agent Sandbox Integration](https://katacontainers.io/blog/kata-containers-agent-sandbox-integration/)
- [claude-squad - GitHub](https://github.com/smtg-ai/claude-squad)
- [ccswarm - GitHub](https://github.com/nwiizo/ccswarm)
- [claude-flow - GitHub](https://github.com/ruvnet/claude-flow)
- [Multi-Agent Orchestration: 10+ Claude Instances - DEV Community](https://dev.to/bredmond1019/multi-agent-orchestration-running-10-claude-instances-in-parallel-part-3-29da)
- [Claude Code Multiple Agent Systems Guide - eesel AI](https://www.eesel.ai/blog/claude-code-multiple-agent-systems-complete-2026-guide)

### Linux Primitives
- [Landlock Documentation](https://landlock.io/)
- [Landlock Kernel Docs](https://docs.kernel.org/userspace-api/landlock.html)
- [Landlock LSM Kernel Docs](https://docs.kernel.org/security/landlock.html)
- [landrun - GitHub](https://github.com/Zouuup/landrun)
- [island - GitHub](https://github.com/landlock-lsm/island)
- [Bubblewrap - GitHub](https://github.com/containers/bubblewrap)
- [OverlayFS Kernel Docs](https://docs.kernel.org/filesystems/overlayfs.html)
- [Seccomp BPF Kernel Docs](https://docs.kernel.org/userspace-api/seccomp_filter.html)
- [gVisor](https://gvisor.dev/)
- [gVisor Seccomp Optimization Blog](https://gvisor.dev/blog/2024/02/01/seccomp/)
- [Firecracker MicroVM](https://firecracker-microvm.github.io/)
- [E2B - Firecracker vs QEMU](https://e2b.dev/blog/firecracker-vs-qemu)

### Filesystem Virtualization
- [AgentFS - Turso Blog](https://turso.tech/blog/agentfs-fuse)
- [AgentFS - GitHub](https://github.com/tursodatabase/agentfs)
- [AgentFS Website](https://www.agentfs.ai/)
- [FUSE is All You Need - Jakob Emmerling](https://jakobemmerling.de/posts/fuse-is-all-you-need/)
- [SecretFS - GitHub](https://github.com/obormot/secretfs)
- [Poof - Ephemeral OverlayFS Isolation](https://github.com/Jarred-Sumner/poof)
- [OverlayFS for Project Work](https://blog.cubieserver.de/2021/using-overlayfs-for-project-work-with-copy-on-write/)
- [Blaxel - OverlayFS 75% Memory Reduction](https://blaxel.ai/blog/how-to-slash-sandbox-memory-usage-by-75-using-overlayfs)
- [OverlayFS Demystified - InfluentCoder](https://influentcoder.com/posts/overlayfs/)

### Network Security
- [eBPF Transparent Proxy Implementation](https://ebpfchirp.substack.com/p/transparent-proxy-implementation)
- [dae - eBPF Transparent Proxy - GitHub](https://github.com/daeuniverse/dae)
- [NetBird - WireGuard Overlay Network](https://github.com/netbirdio/netbird)
- [Netmaker - WireGuard Management](https://www.netmaker.io/solutions/wireguard)

### Security Research
- [io_uring Rootkit Bypasses Linux Security Tools - ARMO](https://www.armosec.io/blog/io_uring-rootkit-bypasses-linux-security/)
- [io_uring PoC Rootkit - The Hacker News](https://thehackernews.com/2025/04/linux-iouring-poc-rootkit-bypasses.html)
- [Google Restricting io_uring - Phoronix](https://www.phoronix.com/news/Google-Restricting-IO_uring)
- [io_uring Security - Upwind](https://www.upwind.io/feed/io_uring-linux-performance-boost-or-security-headache)
- [Sandboxing AI Agents in Linux - Senko Rasic](https://blog.senko.net/sandboxing-ai-agents-in-linux)
- [State of AI Agent Security 2026 - Clawhatch](https://clawhatch.com/blog/state-of-ai-agent-security-2026)
- [Claude Code Security Best Practices - Backslash](https://www.backslash.security/blog/claude-code-security-best-practices)
- [Claude Code Enterprise Security - MintMCP](https://www.mintmcp.com/blog/claude-code-security)
- [Claude Security 2026 Guide - Concentric AI](https://concentric.ai/claude-security-guide/)
- [Deep Dive Security Claude Code - eesel AI](https://www.eesel.ai/blog/security-claude-code)
- [Claude Code Permissions Guide - eesel AI](https://www.eesel.ai/blog/claude-code-permissions)

### Community & Third-Party Tools
- [claude-code-sandbox (TextCortex) - GitHub](https://github.com/textcortex/claude-code-sandbox)
- [cco (Claude Condom) - GitHub](https://github.com/nikvdp/cco) -- auto-selects best sandbox method (sandbox-exec on macOS, bubblewrap on Linux, Docker fallback)
- [bubblewrap (soodoku) - GitHub](https://github.com/soodoku/bubblewrap) -- cross-platform (Linux + macOS) generic sandboxing wrapper for any AI coding assistant
- [awesome-sandbox - GitHub](https://github.com/restyler/awesome-sandbox)
- [Claude Code Hooks Mastery - GitHub](https://github.com/disler/claude-code-hooks-mastery)
- [DevContainer AI Agent Guide - Code With Andrea](https://codewithandrea.com/articles/run-ai-agents-inside-devcontainer/)
- [Running Claude Code in Docker - Medium](https://medium.com/@dan.avila7/running-claude-code-agents-in-docker-containers-for-complete-isolation-63036a2ef6f4)

### Ecosystem Reports
- [Awesome Code Sandboxing for AI - GitHub](https://github.com/restyler/awesome-sandbox)
- [5 Code Sandboxes for AI Agents - KDnuggets](https://www.kdnuggets.com/5-code-sandbox-for-your-ai-agents)
- [Serverless AI Infrastructure 2026 - Koyeb](https://www.koyeb.com/blog/serverless-ai-infrastructure-going-into-2026)
- [Container Use: Isolated Parallel Coding Agents - InfoQ](https://www.infoq.com/news/2025/08/container-use/)
- [Why AI Agents Need Sandboxing - Collabnix](https://collabnix.com/why-ai-agents-need-sandboxing/)
- [Sandboxing Guide for Claude Code - claudefa.st](https://claudefa.st/blog/guide/sandboxing-guide)
- [Anthropic Sandbox Runtime Guide - aiengineerguide](https://aiengineerguide.com/blog/anthropic-sandbox-runtime/)
- [Practical Guide to srt Sandbox](https://www.xugj520.cn/en/archives/securing-ai-agents-srt-sandbox.html)
