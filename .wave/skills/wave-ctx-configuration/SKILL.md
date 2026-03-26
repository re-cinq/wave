---
name: wave-ctx-configuration
description: Domain context for Wave's manifest loading, persona management, and pipeline definition bounded context
---

# Configuration Context

Manifest parsing, persona definitions, pipeline loading, embedded defaults, template variable resolution, and parity enforcement.

## Invariants

- `wave.yaml` is the single source of truth for project configuration: adapters, personas, runtime settings, project metadata, ontology, and skill mounts
- The Wave binary is a single static binary with no runtime dependencies (except adapter binaries like `claude`); all default personas, pipelines, contracts, and prompts are embedded via `go:embed`
- Parity enforcement: files in `.wave/personas/`, `.wave/pipelines/`, and `.wave/contracts/` must be byte-identical to their counterparts in `internal/defaults/`; `parity_test.go` enforces this with bidirectional checks
- Exception to parity: files prefixed with `wave-` (e.g. `wave-bugfix.yaml`) are development-only and not shipped in embedded defaults
- Pipeline YAML defaults `memory.strategy` to `"fresh"` during unmarshalling -- this is a constitutional requirement, not optional
- Deprecated pipeline names resolve via `ResolveDeprecatedName()` (e.g. `gh-implement` maps to `impl-issue`); the original names continue to work
- Forge template variables (`{{ forge.cli_tool }}`, `{{ forge.pr_term }}`, etc.) are populated from `forge.DetectFromGitRemotes()` -- pipelines must not hardcode forge-specific commands
- Project template variables (`{{ project.test_command }}`, `{{ project.language }}`, etc.) are injected from `wave.yaml`'s `project:` section; unresolved placeholders are stripped to empty strings

## Key Decisions

- Manifest validation produces structured `ValidationError` values with file path, line number, field name, reason, and suggestion -- not bare error strings
- Persona permission model uses glob-pattern allow/deny lists; `AllowedTools` controls auto-approval and `Deny` controls tool blocking; they are independent axes (allow = convenience, deny = security)
- Adapters define `default_permissions` that serve as a baseline; persona-level permissions override them
- The `Ontology` type provides project-level domain modeling (telos, bounded contexts with invariants, conventions) that gets rendered as markdown and injected into per-step CLAUDE.md files; context filtering allows steps to receive only relevant bounded contexts
- Pipeline metadata includes `release: bool` (included in `wave init` without `--all`), `disabled: bool`, and `category: string` for organization
- `PersonaSandbox.AllowedDomains` supplements `RuntimeSandbox.DefaultAllowedDomains` -- the effective domain allowlist is the union of both
- `RoutingConfig` supports pattern-based and label-based pipeline selection with priority ordering

## Domain Vocabulary

| Term | Meaning |
|------|---------|
| Manifest | The `wave.yaml` file parsed into a `Manifest` struct; contains all project-level configuration |
| Persona | A named AI agent configuration: adapter reference, system prompt file, model, temperature, permissions, sandbox overrides, token scopes |
| Adapter | A subprocess backend definition: binary name, execution mode, output format, project files, default permissions |
| Pipeline | A YAML file in `.wave/pipelines/` defining a DAG of steps with metadata, input config, and output aliases |
| Embedded defaults | Personas, pipelines, contracts, and prompts compiled into the binary via `go:embed` in `internal/defaults/` |
| Parity test | `internal/defaults/parity_test.go` -- asserts byte-identical content between embedded defaults and `.wave/` working tree files |
| Project vars | Key-value pairs from `wave.yaml`'s `project:` section injected as `{{ project.<key> }}` template variables |
| Ontology | Domain model definition in `wave.yaml`: telos (purpose), bounded contexts with invariants, and naming conventions |
| Forge | Git hosting platform (GitHub, GitLab, Gitea, etc.) detected from remote URLs; provides template variables for forge-agnostic pipelines |
| Deprecated name | Old pipeline name that maps to a unified name via `ResolveDeprecatedName()` |
| Release pipeline | A pipeline with `metadata.release: true` that is included in default `wave init` output |
| Token scope | A persona-level declaration (`token_scopes`) validated at preflight to ensure the persona has appropriate API access |

## Neighboring Contexts

- **Execution** (`internal/pipeline/`) -- the executor consumes manifests to resolve personas, adapters, and runtime config; `PipelineContext` injects project and ontology vars from the manifest
- **Validation** (`internal/contract/`, `internal/preflight/`) -- contract schema paths and test commands originate from manifest configuration
- **Security** (`internal/security/`) -- persona permissions and sandbox config in the manifest feed into security enforcement

## Key Files

- `internal/manifest/types.go` -- `Manifest`, `Persona`, `Adapter`, `Project`, `Ontology`, `OntologyContext`, `Runtime`, `RuntimeSandbox`, `RoutingConfig`, `Permissions` structs; `ProjectVars()`, `OntologyVars()`, `RenderMarkdown()` methods
- `internal/manifest/parser.go` -- YAML parsing, `ValidationError` type with file/line/field/suggestion, manifest validation logic
- `internal/defaults/embed.go` -- `go:embed` directives for personas, pipelines, contracts, prompts; `GetPersonas()`, `GetPipelines()`, `GetContracts()`, `GetPrompts()`, `GetReleasePipelines()`
- `internal/defaults/parity_test.go` -- bidirectional parity assertions between embedded defaults and `.wave/` working tree
- `internal/pipeline/dag.go` -- `YAMLPipelineLoader`, pipeline YAML unmarshalling with fresh-memory defaulting
- `internal/pipeline/deprecated.go` -- `ResolveDeprecatedName()` mapping old pipeline names to unified names
- `internal/pipeline/context.go` -- `InjectForgeVariables()`, `newContextWithProject()`, project/ontology var injection
- `internal/forge/` -- `DetectFromGitRemotes()`, `ForgeInfo` struct with type/host/owner/repo/CLI tool/PR terminology
- `internal/scope/` -- token scope parsing and validation for persona-level API access control
- `internal/onboarding/` -- `wave init` flow: flavour auto-detection, metadata extraction, manifest generation
