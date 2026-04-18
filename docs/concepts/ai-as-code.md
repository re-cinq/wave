# AI-as-Code

Wave is the open-source orchestration layer for AI agent factories. It brings Infrastructure-as-Code discipline to a harder problem: keeping agents useful without letting them run wild.

## The Evolution of X-as-Code

The industry has progressively codified operational concerns:

| Era | Paradigm | Tools |
|-----|----------|-------|
| 2000s | **Infrastructure-as-Code** | Terraform, Pulumi, CloudFormation |
| 2010s | **Configuration-as-Code** | Ansible, Chef, Puppet |
| 2016+ | **Policy-as-Code** | Open Policy Agent, Sentinel |
| 2018+ | **GitOps** | ArgoCD, Flux |
| Now | **AI-as-Code** | Wave |

Each evolution brought the same benefits: version control, reproducibility, collaboration, and audit trails. AI workflows deserve the same treatment.

## The Guardrail Spectrum

Agentic coding tools sit on a spectrum. Most end up at one of two extremes:

| | Too tight | Too loose |
|-|-----------|-----------|
| **Approach** | Approval loops, permission dialogs, human sign-off at every step | Full codebase access — read, write, push, deploy |
| **Result** | Safe on paper, useless in practice | Fast until the first bad prompt |
| **Failure mode** | No real leverage — you're still doing the work | Secrets leaked, files deleted, broken code in production |

Wave finds the middle path: **just the right amount of guardrails**.

Each agent (persona) gets a precisely scoped permission set — fully empowered inside its role, hard-constrained outside it. You don't disable agents. You shape them.

This is the model described in [Building Agent Factories](https://re-cinq.com/blog/building-agent-factories): *"The factory sets boundaries on what's safe to do, not what's allowed."* Wave is the runtime that enforces those boundaries.

## Why AI Workflows Also Need Structure

Beyond permissions, agent workflows have broader failure modes without structure:

- **Chat history is not version control** — Prompts drift, context gets lost, successful patterns disappear
- **Copy-paste prompts don't scale** — Teams can't share, review, or iterate on workflows
- **No reproducibility** — The same task produces different results each time
- **No audit trail** — When something goes wrong, there's no trace to investigate

Enterprise adoption requires the same predictability we expect from infrastructure.

## Wave's AI-as-Code Principles

Wave implements six core principles borrowed from Infrastructure-as-Code:

### 1. Declarative Pipelines

Define **what** you want, not **how** to get there. Your pipeline is a YAML file that describes steps, dependencies, and contracts.

```yaml
kind: WavePipeline
metadata:
  name: ops-pr-review

steps:
  - id: analyze
    persona: navigator
  - id: review
    persona: auditor
    dependencies: [analyze]
```

### 2. Version Controlled

Pipelines live in git, not chat history. You can:
- Review pipeline changes in PRs
- Roll back to previous versions
- Share workflows across teams
- Track who changed what and when

### 3. Contract Validation

Every step validates its output against a schema before the next step begins. Malformed outputs trigger retries or halt the pipeline — no garbage in, no garbage out.

```yaml
output_artifacts:
  - name: analysis
    path: .agents/output/analysis.json
    type: json
    contract: contracts/analysis-schema.json
```

### 4. Fresh Memory Isolation

Each step runs with completely fresh context in an ephemeral workspace. No context bleed between steps means:
- Predictable behavior regardless of history
- No accidental information leakage
- Each persona sees only what it needs

### 5. Git-Native Workflows

Wave integrates with your existing git workflow:
- Initialize with `wave init` in any repo
- Pipelines are just YAML files in `.agents/`
- Artifacts are git-friendly

### 6. Observable Execution

Complete audit trails with credential scrubbing:
- Every tool call logged
- Execution traces for debugging
- Permission decisions recorded
- No sensitive data in logs

## IaC Principle Mapping

| IaC Principle | How Wave Implements It |
|---------------|------------------------|
| **Declarative** | YAML pipeline definitions, not imperative scripts |
| **Version controlled** | Pipelines live in git, not chat history |
| **Reproducible** | Contract validation ensures consistent outputs |
| **Idempotent** | Fresh memory at every step boundary |
| **Auditable** | Complete execution traces with credential scrubbing |
| **Reviewable** | PR your AI workflows like any other code |

## Comparison with Alternatives

Wave's approach differs from other multi-agent tools:

| | Wave | Gastown | Claude Flow | Raw Claude Code |
|--|:----:|:-------:|:-----------:|:---------------:|
| **Guardrail model** | Per-persona scoping | Prompt-based only | Constitution + enforcement gates | Project-level only |
| **Declarative pipelines** | YAML | TOML | Hybrid (code + YAML) | ❌ |
| **Version controlled** | ✅ | ✅ (git worktree) | ✅ (agent configs) | ❌ |
| **Contract validation** | Schema-based | ❌ | Behavioral (hooks, trust scoring) | ❌ |
| **Step isolation** | Fresh memory | Fresh sessions, git-persisted | Shared memory | Single session |
| **Permission scoping** | Per-persona deny/allow | ❌ | Claims + trust throttling | Project-level |

### Gastown

Multi-agent workspace manager with Mayor/Polecat architecture. Strong git integration with worktree-based persistence. Fresh ephemeral sessions with git-persisted state. Different philosophy: prompt-based role enforcement vs Wave's declarative permission scoping.

<small>Sources: <a href="https://github.com/steveyegge/gastown" target="_blank">GitHub</a> · <a href="https://maggieappleton.com/gastown" target="_blank">Appleton analysis</a> · <a href="https://paddo.dev/blog/gastown-two-kinds-of-multi-agent/" target="_blank">paddo.dev review</a></small>

### Claude Flow

Agent swarm orchestration with 60+ agents and MCP tools. V3 adds a Constitution/Shards guidance system, enforcement gates, and trust-based throttling. Different philosophy: shared memory with behavioral validation vs Wave's fresh-memory isolation with schema-based contracts.

<small>Sources: <a href="https://github.com/ruvnet/ruflo" target="_blank">GitHub</a> · <a href="https://deepwiki.com/ruvnet/ruflo" target="_blank">DeepWiki analysis</a> · <a href="https://github.com/ruvnet/ruflo/wiki/Memory-System" target="_blank">Memory System</a></small>

### Raw Claude Code

Direct LLM interaction. Great for ad-hoc tasks. Wave adds structure for repeatable, team-scalable workflows.

<small>Source: <a href="https://docs.anthropic.com/en/docs/claude-code" target="_blank">Claude Code docs</a></small>

## Getting Started

Ready to find the sweet spot between agent autonomy and structured control?

1. [Quickstart Guide](/quickstart) — Get Wave running in 5 minutes
2. [Pipelines Concept](/concepts/pipelines) — Deep dive into pipeline structure
3. [Use Cases](/use-cases/) — Real-world examples
