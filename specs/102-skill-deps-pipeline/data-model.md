# Data Model: Skill Dependency Installation in Pipeline Steps

**Branch**: `102-skill-deps-pipeline` | **Date**: 2026-02-14

## Entity Relationship Overview

```
┌─────────────────────┐         ┌──────────────────────┐
│     Manifest        │ 1    *  │    SkillConfig        │
│  (wave.yaml)        │─────────│  (per external skill) │
│                     │         │                       │
│  Skills map[string] │         │  Check   string       │
│                     │         │  Install string       │
└─────────────────────┘         │  Init    string       │
                                │  CommandsGlob string  │
                                └──────────────────────┘
                                         │
                                         │ referenced by name
                                         ▼
┌─────────────────────┐         ┌──────────────────────┐
│     Pipeline        │ 0..1  * │    Requires           │
│  (pipeline YAML)    │─────────│  (dependency block)   │
│                     │         │                       │
│  Requires *Requires │         │  Skills []string      │
│  Steps    []Step    │         │  Tools  []string      │
└─────────────────────┘         └──────────────────────┘
         │                               │
         │                               │ validated by
         ▼                               ▼
┌─────────────────────┐         ┌──────────────────────┐
│   Executor          │ uses    │    Checker            │
│  (pipeline run)     │─────────│  (preflight)          │
│                     │         │                       │
│                     │         │  skills map[string]   │
│                     │         │    SkillConfig        │
│                     │         │  emitter EventEmitter │
│                     │         │  runCmd func          │
└─────────────────────┘         └──────────────────────┘
         │                               │
         │ provisions                    │ produces
         ▼                               ▼
┌─────────────────────┐         ┌──────────────────────┐
│   Provisioner       │         │    Result             │
│  (skill commands)   │         │  (preflight result)   │
│                     │         │                       │
│  skills map[string] │         │  Name    string       │
│    SkillConfig      │         │  Kind    string       │
│  repoRoot string    │         │  OK      bool         │
└─────────────────────┘         │  Message string       │
         │                      └──────────────────────┘
         │ copies to
         ▼
┌─────────────────────┐
│  Step Workspace     │
│  (per pipeline step)│
│                     │
│  .wave-skill-cmds/  │──► adapter copies to .claude/commands/
│    .claude/commands/ │
└─────────────────────┘
```

## Existing Entities (No Changes Required)

### SkillConfig (`internal/manifest/types.go:142-148`)

```go
type SkillConfig struct {
    Install      string `yaml:"install,omitempty"`
    Init         string `yaml:"init,omitempty"`
    Check        string `yaml:"check,omitempty"`
    CommandsGlob string `yaml:"commands_glob,omitempty"`
}
```

**Status**: Complete. All fields needed by the spec are present.

### Requires (`internal/pipeline/types.go:20-23`)

```go
type Requires struct {
    Skills []string `yaml:"skills,omitempty"`
    Tools  []string `yaml:"tools,omitempty"`
}
```

**Status**: Complete. Referenced from `Pipeline.Requires` at types.go:14.

### Result (`internal/preflight/preflight.go:12-17`)

```go
type Result struct {
    Name    string
    Kind    string // "tool" or "skill"
    OK      bool
    Message string
}
```

**Status**: Complete. Captures per-dependency preflight outcomes.

### Provisioner (`internal/skill/skill.go:13-16`)

```go
type Provisioner struct {
    skills   map[string]manifest.SkillConfig
    repoRoot string
}
```

**Status**: Complete. Handles command file discovery and copying.

## Modified Entities

### Checker (`internal/preflight/preflight.go:20-24`)

**Current**:
```go
type Checker struct {
    skills  map[string]manifest.SkillConfig
    runCmd  func(name string, args ...string) error
}
```

**Proposed** (add optional event emission):
```go
type Checker struct {
    skills  map[string]manifest.SkillConfig
    runCmd  func(name string, args ...string) error
    emitter func(name, kind, message string) // optional progress callback
}
```

**Rationale**: FR-010 requires per-dependency progress events. Adding a callback keeps the preflight package decoupled from the event package — the executor wires the callback to its event emitter.

**Alternative considered**: Accept an `event.EventEmitter` directly. Rejected because it would create a dependency cycle risk and make the preflight package less testable.

## New Constants

### StatePreflight (`internal/event/emitter.go`)

```go
const StatePreflight = "preflight"
```

**Rationale**: FR-010 + C4 clarification. Replace the string literal `"preflight"` at executor.go:173 with a named constant for consistency with `StateStarted`, `StateCompleted`, etc.

## Configuration Data (wave.yaml additions)

### Skills Section

```yaml
skills:
  speckit:
    check: "test -d .specify"
    install: "npx -y @anthropic/speckit init"
    commands_glob: ".claude/commands/speckit.*.md"
  bmad:
    check: "test -f .claude/commands/bmad.*.md"
    commands_glob: ".claude/commands/bmad.*.md"
  openspec:
    check: "test -d .openspec"
    commands_glob: ".claude/commands/openspec.*.md"
```

**Rationale**: FR-011 requires support for these three skills. The definitions use presence-based checks (`test -d`, `test -f`) since these skills are filesystem-based.

## Data Flow

### Preflight Phase (FR-004, FR-005, FR-006)

```
Pipeline.Requires
    │
    ├── Tools: ["git", "go"]
    │       │
    │       └──► Checker.CheckTools()
    │               │
    │               └──► exec.LookPath(tool)
    │                       │
    │                       └──► Result{Name, Kind:"tool", OK, Message}
    │
    └── Skills: ["speckit"]
            │
            └──► Checker.CheckSkills()
                    │
                    ├──► SkillConfig lookup in manifest
                    │
                    ├──► isSkillInstalled() → sh -c <check>
                    │       │
                    │       ├── OK → Result{OK:true}
                    │       │
                    │       └── FAIL → runShellCommand(<install>)
                    │               │
                    │               ├── runShellCommand(<init>) [if configured]
                    │               │
                    │               └── isSkillInstalled() [re-check]
                    │                       │
                    │                       └──► Result{OK:true/false}
                    │
                    └──► Event emission per step (via callback)
```

### Provisioning Phase (FR-007, FR-012)

```
Executor.runStepExecution()
    │
    ├── Provisioner.DiscoverCommands(skills)
    │       │
    │       └── filepath.Glob(repoRoot + commands_glob)
    │               │
    │               └── map[skill][]commandPaths
    │
    ├── Provisioner.Provision(tmpDir, skills)
    │       │
    │       └── copyFile(src, .wave-skill-commands/.claude/commands/<file>)
    │
    └── Adapter.prepareWorkspace()
            │
            └── copySkillCommands(settingsDir, skillCommandsDir)
                    │
                    └── final: workspace/.claude/commands/<file>
```

## Invariants

1. **Skills referenced by pipelines MUST exist in the manifest** — Validated by Checker at runtime (FR-003)
2. **Every skill MUST have a `check` command** — Enforced by manifest validation (FR-008)
3. **Install and init are optional** — If check fails and no install exists, preflight fails with descriptive error
4. **Each step gets independent commands** — Provisioned per-step, not shared (FR-012)
5. **Preflight runs before any step** — Fail-fast guarantee (FR-004)
6. **Tool checks are presence-only** — No version validation (Edge Case 6)
