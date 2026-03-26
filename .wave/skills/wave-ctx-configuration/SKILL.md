---
name: wave-ctx-configuration
description: Domain context for Single-source-of-truth manifest system — wave.yaml loading with synchronous validation, persona definitions with system prompt files and permission models, pipeline YAML definitions as separate DAG files with template variable resolution, embedded defaults (personas, pipelines, contracts, prompts) compiled into the binary, five-level configuration layering with CLI override precedence, deprecated name resolution for backward compatibility, and forge-aware template expansion.
---

# Configuration Context

Single-source-of-truth manifest system — wave.yaml loading with synchronous validation, persona definitions with system prompt files and permission models, pipeline YAML definitions as separate DAG files with template variable resolution, embedded defaults (personas, pipelines, contracts, prompts) compiled into the binary, five-level configuration layering with CLI override precedence, deprecated name resolution for backward compatibility, and forge-aware template expansion.

## Invariants

- metadata.name is required — empty/whitespace returns ValidationError
- runtime.workspace_root is required — empty/whitespace returns ValidationError
- Each adapter must have non-empty binary and mode fields
- Each persona must have a non-empty adapter field referencing an existing adapter in the manifest
- Each persona must have a non-empty system_prompt_file that exists on disk — resolved relative to manifest directory
- Token scopes must be syntactically valid — format: resource:permission where resource in {issues,pulls,repos,actions,packages} and permission in {read,write,admin}
- Ontology context names must be non-empty and unique — duplicate names produce ValidationError
- Validation errors returned as first error only from Load path — not aggregated
- Pipeline DAG must be acyclic — DFS cycle detection rejects circular dependencies
- Pipeline dependencies must reference existing step IDs within the same pipeline
- Memory strategy defaults to 'fresh' — constitutional requirement, set in YAMLPipelineLoader.Unmarshal
- Pipeline Kind defaults to 'WavePipeline' if empty after YAML parse
- kind must be WaveManifest or Wave — validated by wave validate command
- apiVersion is required — validated by wave validate command
- Every non-composition step must have a persona — composition steps (sub-pipeline, branch, gate, loop, aggregate) exempt
- Step persona must exist in manifest after forge template resolution
- Prompt source files, contract schema files, and sub-pipeline files must exist on disk
- Step IDs must be unique within a pipeline
- No programming language references in persona files — test scans for 12 language patterns
- Persona files must be 100-400 tokens — excluding base-protocol.md
- Persona files must have three mandatory sections: H1 identity heading, Responsibilities section, Output contract section
- Unresolved {{ project.* }} and {{ ontology.* }} placeholders stripped to empty string after resolution — prevents mustache syntax leaking into prompts

## Key Decisions

- Single-file, single-pass manifest loading — no multi-file merging, wave.yaml is the authoritative source
- Validation is synchronous with loading — first error returned immediately, manifest never partially returned
- Personas defined in manifest, system prompts in separate .md files — decouples identity/permissions from behavioral instructions
- Pipelines stored as separate YAML files in .wave/pipelines/ — not in wave.yaml, enables independent versioning
- Pipeline search order: .wave/pipelines/<name>.yaml > .wave/pipelines/<name> > absolute/relative path
- Five embedded filesystems compiled into binary — personas .md, persona configs .yaml, pipelines .yaml, contracts .json, prompts .md
- Release-filtered installation — only pipelines with metadata.release:true installed by default, --all for everything
- No backward-compatibility or deprecated name resolution — removed pre-1.0.0
- Five-level configuration layering: embedded defaults < installed .wave/ files < wave.yaml < CLI flags < template variables at execution time
- Merge semantics for wave init --merge — preserves existing files (status:'preserved'), creates new files (status:'new')
- Forge-aware template expansion — {{ forge.type }}, {{ forge.cli_tool }}, {{ forge.pr_term }}, {{ forge.pr_command }} resolved from git remote detection
- System personas (summarizer, navigator, philosopher) always included regardless of pipeline references — used by relay, meta-pipelines, ad-hoc operations
- Skill validation requires explicit store — LoadWithSkillStore extends Load, nil store skips skill validation entirely

## Domain Vocabulary

| Term | Meaning |
|------|--------|
| Manifest | Top-level config: apiVersion, kind, metadata, project, ontology, adapters, personas, skills, runtime — single source of truth |
| Metadata | Project identity: name, description, repo |
| Project | Language/tooling config: language, flavour, test_command, lint_command, build_command, format_command, source_glob, skill |
| Ontology | Domain model: telos (project purpose), contexts (bounded contexts with invariants), conventions |
| OntologyContext | Named bounded context with description and list of domain invariants |
| Adapter | CLI binding: binary, mode, output_format, project_files, default_permissions, hooks_template |
| Persona | AI agent definition: adapter, system_prompt_file, temperature, model, permissions, hooks, sandbox, skills, token_scopes |
| Permissions | Tool access control: allowed_tools list and deny list — deny takes precedence |
| Runtime | Execution config: workspace_root, max_concurrent_workers, default_timeout, relay, audit, meta_pipeline, routing, sandbox, artifacts |
| RuntimeSandbox | Sandbox config: enabled, backend (bubblewrap/docker/none), docker_image, default_allowed_domains, env_passthrough |
| RelayConfig | Context compaction config: token_threshold_percent, strategy, context_window, summarizer_persona |
| AuditConfig | Trace logging config: log_dir, log_all_tool_calls, log_all_file_operations |
| RoutingConfig | Pipeline routing: default pipeline and rules with pattern/label matching for automatic pipeline selection |
| ValidationError | Structured manifest error with file, line, column, field, reason, suggestion |
| ManifestLoader | Interface: Load(path string) (*Manifest, error) — pluggable manifest loading |
| PipelineLoader | Interface: Load(path string) (*Pipeline, error) — pluggable pipeline loading |
| PipelineMetadata | Pipeline identity: name, description, release flag, category, disabled flag |
| PipelineContext | Runtime template resolution context: branch_name, feature_num, pipeline_id, custom_variables, artifact_paths |
| WorkspaceConfig | Step workspace setup: root, mount, type (worktree/basic), branch, base, ref |
| ExecConfig | Step execution method: type (prompt/command/slash_command), source, source_path, command, args |
| HandoverConfig | Step completion config: contract, compaction, on_review_fail, target_step |
| ChatContextConfig | Post-pipeline chat injection: max_context_tokens with default 8000 |
| taxonomyMappings | Deprecated-to-current pipeline name mapping table for backward compatibility |
| forge | Git hosting platform type: github, gitlab, gitea, bitbucket — detected from git remotes |
| ProjectVars | Template variables from Project config: project.test_command, project.language, etc. |
| OntologyVars | Template variables from Ontology: ontology.telos, ontology.contexts, etc. |

## Neighboring Contexts

- **execution**
- **security**
- **validation**

## Key Files

- `internal/manifest/types.go`
- `internal/manifest/parser.go`
- `internal/pipeline/types.go`
- `internal/pipeline/dag.go`
- `internal/defaults/embed.go`
- `internal/pipeline/context.go`
- `internal/pipeline/deprecated.go`
- `cmd/wave/commands/run.go`
- `cmd/wave/commands/init.go`
- `cmd/wave/commands/validate.go`
- `wave.yaml`

