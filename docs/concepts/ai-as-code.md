# AI-as-Code

Wave brings Infrastructure-as-Code principles to AI workflows. Define your AI pipelines declaratively, version them in git, and run them with the same rigor you apply to infrastructure.

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

## Why AI Needs the Same Treatment

AI outputs are non-deterministic by nature. Without guardrails:

- **Chat history is not version control** — Prompts drift, context gets lost, and successful patterns disappear
- **Copy-paste prompts don't scale** — Teams can't share, review, or iterate on workflows
- **No reproducibility** — The same task produces different results each time
- **No audit trail** — When something goes wrong, there's no trace to investigate
- **No permission boundaries** — AI agents have unbounded access to your codebase

Enterprise adoption requires the same predictability we expect from infrastructure.

## Wave's AI-as-Code Principles

Wave implements six core principles borrowed from Infrastructure-as-Code:

### 1. Declarative Pipelines

Define **what** you want, not **how** to get there. Your pipeline is a YAML file that describes steps, dependencies, and contracts.

```yaml
kind: WavePipeline
metadata:
  name: code-review

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
    path: output/analysis.json
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
- Pipelines are just YAML files in `.wave/`
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

| | Wave | Gastown | Claude Flow |
|--|:----:|:-------:|:-----------:|
| **Declarative pipelines** | YAML | JSON/TOML | Programmatic |
| **Version controlled** | ✅ | ✅ (git worktree) | ❌ |
| **Contract validation** | ✅ | ❌ | ❌ |
| **Step isolation** | Fresh memory | Shared context | Shared memory |
| **Permission scoping** | Per-persona | ❌ | ❌ |

### Gastown

Multi-agent workspace manager with Mayor/Polecat architecture. Strong git integration with worktree-based persistence. Different philosophy: persistent shared state vs Wave's fresh-memory isolation.

### Claude Flow

Agent swarm orchestration with 60+ agents and MCP tools. Optimized for parallel execution and collective learning. Different philosophy: shared knowledge base vs Wave's contract-validated handoffs.

### Raw Claude Code

Direct LLM interaction. Great for ad-hoc tasks. Wave adds structure for repeatable, team-scalable workflows.

## Getting Started

Ready to bring Infrastructure-as-Code rigor to your AI workflows?

1. [Quickstart Guide](/quickstart) — Get Wave running in 5 minutes
2. [Pipelines Concept](/concepts/pipelines) — Deep dive into pipeline structure
3. [Use Cases](/use-cases/) — Real-world examples
