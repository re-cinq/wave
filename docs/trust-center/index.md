# Trust Center

Wave's security posture relies on layered isolation and explicit configuration. None of these protections are magic — they require the operator to set them up.

## Core Principles

| Principle | What It Means | Where It Lives |
|-----------|---------------|----------------|
| **Deny-First Permissions** | Persona allow/deny patterns projected into `settings.json` and `CLAUDE.md` | [Personas](/concepts/personas) |
| **Fresh Memory** | No chat history inheritance between steps; inter-step data flows through explicit artifacts | [Workspaces](/concepts/workspaces) |
| **Contract Validation** | Step outputs validated against JSON schemas before downstream injection | [Contracts](/concepts/contracts) |
| **Curated Environment** | Only `env_passthrough` vars reach adapter subprocesses; credentials never touch disk | [Environment](/reference/environment) |
| **Process Sandbox** | Optional Nix + bubblewrap sandbox isolates the entire session (Linux only) | [Sandbox Setup](/guides/sandbox-setup) |

## What Requires Operator Action

- **Sandbox**: You must run `nix develop` to get bubblewrap isolation. Without it, Wave runs unsandboxed (Claude Code's built-in Seatbelt applies on macOS).
- **Permissions**: Persona deny/allow rules only work if you define them in your manifest. The defaults ship with reasonable restrictions but you should review them.
- **Contracts**: Contract validation only runs for steps that declare a `handover.contract`. Unchecked steps pass output without validation.
- **Credential scrubbing**: The trace logger redacts patterns like `*_KEY`, `*_TOKEN`, `*_SECRET` in log output. It does not prevent the LLM from seeing credentials passed via `env_passthrough`.

## Vulnerability Disclosure

If you discover a security issue in Wave, please report it via [GitHub Issues](https://github.com/re-cinq/wave/issues) with the `security` label, or open a private security advisory on the repository.

## Further Reading

- [Sandbox Setup](/guides/sandbox-setup) - Nix + bubblewrap configuration
- [Personas](/concepts/personas) - Permission model and deny-first evaluation
- [Environment & Credentials](/reference/environment) - Environment variable reference
